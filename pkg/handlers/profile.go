package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// UpdateMe godoc
// @Summary      Update the authenticated user's profile
// @Tags         user
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body models.UpdateProfileRequest true "Profile fields"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Router       /me [put]
func UpdateMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch current user
	var current models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&current); err != nil {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	// Check uniqueness if username changed
	if req.Username != current.Username {
		cnt, _ := db.Col("users").CountDocuments(ctx, bson.M{"username": req.Username, "_id": bson.M{"$ne": objID}})
		if cnt > 0 {
			response.Error(c, http.StatusConflict, "Username already taken")
			return
		}
	}

	// Check uniqueness if email changed
	if req.Email != current.Email {
		cnt, _ := db.Col("users").CountDocuments(ctx, bson.M{"email": req.Email, "_id": bson.M{"$ne": objID}})
		if cnt > 0 {
			response.Error(c, http.StatusConflict, "Email already registered")
			return
		}
	}

	update := bson.M{
		"username": req.Username,
		"email":    req.Email,
		"bio":      req.Bio,
	}

	// Optional password change
	if req.NewPassword != "" {
		if req.OldPassword == "" {
			response.Error(c, http.StatusBadRequest, "Current password is required to set a new one")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(req.OldPassword)); err != nil {
			response.Error(c, http.StatusUnauthorized, "Current password is incorrect")
			return
		}
		if len(req.NewPassword) < 6 {
			response.Error(c, http.StatusBadRequest, "New password must be at least 6 characters")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		update["password_hash"] = string(hash)
	}

	if _, err := db.Col("users").UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update}); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Return updated user
	var updated models.User
	_ = db.Col("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&updated)

	response.Success(c, http.StatusOK, updated)
}

// UploadAvatar godoc
// @Summary      Upload or replace the user's profile picture
// @Tags         user
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar formData file true "Avatar image"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Router       /me/avatar [put]
func UploadAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Avatar file is required")
		return
	}
	defer file.Close()

	// Save to ./uploads/avatar-<userId>.<ext>
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("avatar-%s%s", objID.Hex(), ext)
	dir := filepath.Join(".", "uploads")
	_ = os.MkdirAll(dir, 0o755)
	destPath := filepath.Join(dir, storedName)

	if err := c.SaveUploadedFile(header, destPath); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save avatar")
		return
	}

	publicURL := "/uploads/" + storedName

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db.Col("users").UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{
		"profile_picture": publicURL,
	}})

	response.Success(c, http.StatusOK, gin.H{"profile_picture": publicURL})
}
