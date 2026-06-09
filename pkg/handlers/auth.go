package handlers

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	bcryptCost      = 12            // high cost factor
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// ── JWT Claims ───────────────────────────────────────────────────────────────

// AccessClaims holds only non-sensitive identity data.
// No email, username, or PII in the payload.
type AccessClaims struct {
	Role string `json:"role"` // "user" | "admin"
	jwt.RegisteredClaims
}

// RefreshClaims holds only the subject (user ID).
type RefreshClaims struct {
	jwt.RegisteredClaims
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "artskilwallet-change-in-production-secret-key"
	}
	return []byte(s)
}

func refreshSecret() []byte {
	s := os.Getenv("JWT_REFRESH_SECRET")
	if s == "" {
		s = "artskilwallet-refresh-change-in-production-key"
	}
	return []byte(s)
}

// hashToken returns the SHA-256 hex digest of a raw token string.
// We store this hash in MongoDB — never the raw token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

func signAccessToken(userID, role string) (string, error) {
	claims := AccessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func signRefreshToken(userID string) (string, error) {
	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshSecret())
}

// ── Register ─────────────────────────────────────────────────────────────────

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body models.RegisterRequest true "Register payload"
// @Success      201  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      409  {object} response.APIResponse
// @Router       /auth/register [post]
func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check username uniqueness
	count, err := db.Col("users").CountDocuments(ctx, bson.M{"username": req.Username})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Database error")
		return
	}
	if count > 0 {
		response.Error(c, http.StatusConflict, "Username already taken")
		return
	}

	// Check email uniqueness
	emailCount, err := db.Col("users").CountDocuments(ctx, bson.M{"email": req.Email})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Database error")
		return
	}
	if emailCount > 0 {
		response.Error(c, http.StatusConflict, "Email already registered")
		return
	}

	// Hash password — NEVER store plain text
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := models.User{
		ID:           primitive.NewObjectID(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "user",
		CreatedAt:    time.Now(),
	}

	if _, err := db.Col("users").InsertOne(ctx, user); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"username": user.Username})
}

// ── Login ────────────────────────────────────────────────────────────────────

// Login godoc
// @Summary      Login and receive JWT access + refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body models.LoginRequest true "Login payload"
// @Success      200  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      401  {object} response.APIResponse
// @Router       /auth/login [post]
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := db.Col("users").FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)

	// Always run bcrypt.CompareHashAndPassword to prevent timing attacks
	// that reveal whether an email exists in the database.
	dummyHash, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcryptCost)
	compareHash := user.PasswordHash
	if err != nil {
		compareHash = string(dummyHash)
	}

	if bcrypt.CompareHashAndPassword([]byte(compareHash), []byte(req.Password)) != nil || err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Block banned users
	if user.Banned {
		response.Error(c, http.StatusForbidden, "Your account has been banned. Please contact support.")
		return
	}

	// Sign tokens — payload contains only sub + role, no sensitive data
	accessToken, err := signAccessToken(user.ID.Hex(), user.Role)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	refreshToken, err := signRefreshToken(user.ID.Hex())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}
// refresh
	// Persist hashed refresh token in MongoDB
	rtDoc := models.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		CreatedAt: time.Now(),
	}
	if _, err := db.Col("refresh_tokens").InsertOne(ctx, rtDoc); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save session")
		return
	}

	response.Success(c, http.StatusOK, models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Username:     user.Username,
	})
}

// ── Refresh ──────────────────────────────────────────────────────────────────

// Refresh godoc
// @Summary      Exchange a valid refresh token for a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body models.RefreshRequest true "Refresh payload"
// @Success      200  {object} response.APIResponse
// @Failure      401  {object} response.APIResponse
// @Router       /auth/refresh [post]
func Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Verify refresh token signature & expiry
	// Enforce algorithm strictly — prevent downgrade attack
	token, err := jwt.ParseWithClaims(
		req.RefreshToken,
		&RefreshClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return refreshSecret(), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil || !token.Valid {
		response.Error(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Invalid token claims")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify the hashed token exists in the database (revocation check)
	tokenHash := hashToken(req.RefreshToken)
	var storedToken models.RefreshToken
	if err := db.Col("refresh_tokens").FindOne(ctx, bson.M{
		"token_hash": tokenHash,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&storedToken); err != nil {
		response.Error(c, http.StatusUnauthorized, "Refresh token has been revoked or expired")
		return
	}

	// Fetch current user role for the new access token
	userObjID, err := primitive.ObjectIDFromHex(claims.Subject)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid subject in token")
		return
	}

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err != nil {
		response.Error(c, http.StatusUnauthorized, "User not found")
		return
	}

	// Issue new access token
	newAccessToken, err := signAccessToken(user.ID.Hex(), user.Role)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"access_token": newAccessToken})
}

// ── Logout ───────────────────────────────────────────

// Logout godoc
// @Summary      Revoke the refresh token
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body models.RefreshRequest true "Refresh token to revoke"
// @Success      200  {object} response.APIResponse
// @Router       /auth/logout [post]
func Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete the refresh token from the database — token is now revoked
	db.Col("refresh_tokens").DeleteOne(ctx, bson.M{"token_hash": hashToken(req.RefreshToken)})

	response.Success(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe ──────────────────────────────────────────────────────

// GetMe godoc
// @Summary      Get the currently authenticated user's profile
// @Tags         user
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /me [get]
func GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	response.Success(c, http.StatusOK, user)
}
