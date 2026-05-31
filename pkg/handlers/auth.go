package handlers

import (
	"context"
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

	count, err := db.Col("users").CountDocuments(ctx, bson.M{"username": req.Username})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Database error")
		return
	}
	if count > 0 {
		response.Error(c, http.StatusConflict, "Username already taken")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.HashPass), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := models.User{
		ID:        primitive.NewObjectID(),
		Username:  req.Username,
		Bio:       req.Bio,
		HashPass:  string(hash),
		CreatedAt: time.Now(),
	}

	if _, err := db.Col("users").InsertOne(ctx, user); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"username": user.Username})
}

// Login godoc
// @Summary      Login and receive JWT token
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
	if err := db.Col("users").FindOne(ctx, bson.M{"username": req.Username}).Decode(&user); err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashPass), []byte(req.Password)); err != nil {
		response.Error(c, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "artskilwallet-default-secret"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response.Success(c, http.StatusOK, models.LoginResponse{
		Token:    tokenStr,
		Username: user.Username,
	})
}
