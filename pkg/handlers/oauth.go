package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func randomStateStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func fetchGoogleUser(cfg *oauth2.Config, code string) (*googleUserInfo, error) {
	ctx := context.Background()
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}
	resp, err := cfg.Client(ctx, tok).Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var info googleUserInfo
	json.Unmarshal(body, &info)
	return &info, nil
}

func frontendURL() string {
	if u := os.Getenv("FRONTEND_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}
// Google OAuth สุ่มชื่อ
func genUsername(name, email string) string {
	base := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	if base == "" {
		base = strings.Split(email, "@")[0]
	}
	b := make([]byte, 3)
	rand.Read(b)
	return base + "_" + fmt.Sprintf("%x", b)
}

func GoogleOAuthRedirect(c *gin.Context) {
	mode := c.DefaultQuery("mode", "login")
	state := mode + ":" + randomStateStr()
	c.Redirect(http.StatusTemporaryRedirect, googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline))
}

func GoogleOAuthCallback(c *gin.Context) {
	state := c.Query("state")
	fe := frontendURL()
	parts := strings.SplitN(state, ":", 2)
	mode := "login"
	if len(parts) > 0 {
		mode = parts[0]
	}
	gUser, err := fetchGoogleUser(googleOAuthConfig(), c.Query("code"))
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=google_failed")
		return
	}
	if mode == "reset" {
		handleGoogleReset(c, gUser, fe)
	} else {
		handleGoogleAuth(c, gUser, fe)
	}
}

func handleGoogleAuth(c *gin.Context, gUser *googleUserInfo, fe string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := db.Col("users").FindOne(ctx, bson.M{"email": gUser.Email}).Decode(&user)
	if err != nil {
		username := genUsername(gUser.Name, gUser.Email)
		user = models.User{
			ID:             primitive.NewObjectID(),
			Username:       username,
			Email:          gUser.Email,
			GoogleID:       gUser.Sub,
			ProfilePicture: gUser.Picture,
			Role:           "user",
			CreatedAt:      time.Now(),
		}
		if _, e := db.Col("users").InsertOne(ctx, user); e != nil {
			c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=db_error")
			return
		}
	} else {
		if user.Banned {
			c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=banned")
			return
		}
		if user.GoogleID == "" {
			db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"google_id": gUser.Sub}})
		}
	}

	access, err := signAccessToken(user.ID.Hex(), user.Role)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=token_error")
		return
	}
	refresh, err := signRefreshToken(user.ID.Hex())
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=token_error")
		return
	}
	db.Col("refresh_tokens").InsertOne(ctx, models.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		TokenHash: hashToken(refresh),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		CreatedAt: time.Now(),
	})
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf(
		"%s/home.html?access_token=%s&refresh_token=%s&username=%s&role=%s",
		fe, access, refresh, user.Username, user.Role,
	))
}

func handleGoogleReset(c *gin.Context, gUser *googleUserInfo, fe string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"email": gUser.Email}).Decode(&user); err != nil {
		c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=email_not_found")
		return
	}
	if user.Banned {
		c.Redirect(http.StatusTemporaryRedirect, fe+"/home.html?oauth_error=banned")
		return
	}

	raw := make([]byte, 32)
	rand.Read(raw)
	resetToken := base64.URLEncoding.EncodeToString(raw)
	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{
		"reset_token":      resetToken,
		"reset_expires_at": time.Now().Add(15 * time.Minute),
	}})
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/home.html?reset_token=%s", fe, resetToken))
}

func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return fmt.Sprintf("%x", sum)
}

func sendOTPEmail(to, otp string) error {
	from := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	host := "smtp.gmail.com"
	port := "465"

	subject := "ArtSkillWallet — รหัส OTP รีเซ็ตรหัสผ่าน"
	body := fmt.Sprintf(
		"รหัส OTP ของคุณ: %s\n\nรหัสนี้ใช้ได้ภายใน 10 นาที\nหากคุณไม่ได้ร้องขอ กรุณาเพิกเฉยต่ออีเมลนี้",
		otp,
	)
	msg := fmt.Sprintf("From: ArtSkillWallet <%s>\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	tlsCfg := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", host+":"+port, tlsCfg)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", from, pass, host)
	if err = client.Auth(auth); err != nil {
		return err
	}
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(msg))
	w.Close()
	return err
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		response.Success(c, http.StatusOK, gin.H{"message": "หากอีเมลนี้มีในระบบ OTP จะถูกส่งไป"})
		return
	}
	if user.Banned {
		response.Error(c, http.StatusForbidden, "บัญชีนี้ถูกระงับ")
		return
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	otp := fmt.Sprintf("%06d", n.Int64())

	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{
		"otp_hash":       hashOTP(otp),
		"otp_expires_at": time.Now().Add(10 * time.Minute),
	}})

	go sendOTPEmail(req.Email, otp)

	response.Success(c, http.StatusOK, gin.H{"message": "ส่ง OTP ไปยังอีเมลแล้ว"})
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp"   binding:"required"`
}

func VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{
		"email":          req.Email,
		"otp_hash":       hashOTP(req.OTP),
		"otp_expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&user); err != nil {
		response.Error(c, http.StatusUnauthorized, "OTP ไม่ถูกต้องหรือหมดอายุแล้ว")
		return
	}

	raw := make([]byte, 32)
	rand.Read(raw)
	resetToken := base64.URLEncoding.EncodeToString(raw)

	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set":   bson.M{"reset_token": resetToken, "reset_expires_at": time.Now().Add(15 * time.Minute)},
		"$unset": bson.M{"otp_hash": "", "otp_expires_at": ""},
	})

	response.Success(c, http.StatusOK, gin.H{"reset_token": resetToken})
}

type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token"  binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{
		"reset_token":      req.ResetToken,
		"reset_expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&user); err != nil {
		response.Error(c, http.StatusUnauthorized, "Reset token ไม่ถูกต้องหรือหมดอายุ")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "เกิดข้อผิดพลาด")
		return
	}

	db.Col("users").UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set":   bson.M{"password_hash": string(hash)},
		"$unset": bson.M{"reset_token": "", "reset_expires_at": ""},
	})

	response.Success(c, http.StatusOK, gin.H{"message": "ตั้งรหัสผ่านใหม่สำเร็จ"})
}
