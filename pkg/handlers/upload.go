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

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// maxUploadSize is 50 MB.
const maxUploadSize = 50 << 20

// uploadsDir returns the absolute path to the uploads folder,
// creating it if it doesn't exist.
func uploadsDir() string {
	dir := filepath.Join(".", "uploads")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// ── Create Upload ────────────────────────────────────────────────────────────

// CreateUpload godoc
// @Summary      Upload a file with metadata
// @Description  Accepts a multipart/form-data request containing a file plus title and description fields.
//
//	The file is saved to the local ./uploads folder and a metadata record is stored in MongoDB.
//
// @Tags         uploads
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file        formData file   true  "File to upload (max 50 MB)"
// @Param        title       formData string true  "Document title"
// @Param        description formData string false "Optional description"
// @Success      201 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /uploads [post]
func CreateUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// ── Parse multipart form ────────────────────────────────────────────
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		response.Error(c, http.StatusBadRequest, "File too large or invalid form data")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "File field 'file' is required")
		return
	}
	defer file.Close()

	title := c.PostForm("title")
	if title == "" {
		response.Error(c, http.StatusBadRequest, "Title is required")
		return
	}
	description := c.PostForm("description")

	// ── Save file to disk ───────────────────────────────────────────────
	docID := primitive.NewObjectID()
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("%s%s", docID.Hex(), ext)
	destPath := filepath.Join(uploadsDir(), storedName)

	if err := c.SaveUploadedFile(header, destPath); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save file")
		return
	}

	// ── Insert record into MongoDB ──────────────────────────────────────
	now := time.Now()
	doc := models.Upload{
		ID:          docID,
		UserID:      objID,
		Title:       title,
		Description: description,
		FileName:    header.Filename,
		FilePath:    destPath,
		FileSize:    header.Size,
		MimeType:    header.Header.Get("Content-Type"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := db.Col("uploads").InsertOne(ctx, doc); err != nil {
		// Roll back the saved file on DB failure
		_ = os.Remove(destPath)
		response.Error(c, http.StatusInternalServerError, "Failed to save upload record")
		return
	}

	response.Success(c, http.StatusCreated, doc)
}

// ── List Uploads ─────────────────────────────────────────────────────────────

// GetUploads godoc
// @Summary      List all uploads for the logged-in user
// @Description  Returns every upload document belonging to the authenticated user, newest first.
// @Tags         uploads
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Router       /uploads [get]
func GetUploads(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("uploads").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch uploads")
		return
	}
	defer cursor.Close(ctx)

	var uploads []models.Upload
	if err := cursor.All(ctx, &uploads); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode uploads")
		return
	}
	if uploads == nil {
		uploads = []models.Upload{}
	}

	response.Success(c, http.StatusOK, uploads)
}

// ── Update Upload ────────────────────────────────────────────────────────────

// UpdateUpload godoc
// @Summary      Update upload metadata
// @Description  Updates the title and description of an upload owned by the authenticated user.
//
//	The file itself is not replaced — only the metadata record changes.
//
// @Tags         uploads
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string                     true "Upload ID (ObjectID hex)"
// @Param        body body     models.UploadUpdateRequest  true "Updated metadata"
// @Success      200  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      404  {object} response.APIResponse
// @Router       /uploads/{id} [put]
func UpdateUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	uploadID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid upload ID")
		return
	}

	var req models.UploadUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": uploadID, "user_id": objID}
	update := bson.M{"$set": bson.M{
		"title":       req.Title,
		"description": req.Description,
		"updated_at":  time.Now(),
	}}

	result, err := db.Col("uploads").UpdateOne(ctx, filter, update)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update upload")
		return
	}
	if result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Upload not found or does not belong to you")
		return
	}

	var updated models.Upload
	if err := db.Col("uploads").FindOne(ctx, bson.M{"_id": uploadID}).Decode(&updated); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch updated upload")
		return
	}

	response.Success(c, http.StatusOK, updated)
}

// ── Delete Upload ────────────────────────────────────────────────────────────

// DeleteUpload godoc
// @Summary      Delete an upload
// @Description  Removes the file from local storage and deletes the metadata record from MongoDB.
//
//	Only the owner of the upload can delete it.
//
// @Tags         uploads
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Upload ID (ObjectID hex)"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /uploads/{id} [delete]
func DeleteUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	uploadID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid upload ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch the record first so we know the file path
	var doc models.Upload
	if err := db.Col("uploads").FindOne(ctx, bson.M{"_id": uploadID, "user_id": objID}).Decode(&doc); err != nil {
		response.Error(c, http.StatusNotFound, "Upload not found or does not belong to you")
		return
	}

	// Delete from MongoDB
	if _, err := db.Col("uploads").DeleteOne(ctx, bson.M{"_id": uploadID}); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete upload record")
		return
	}

	// Remove the physical file (best-effort — don't fail the response if this errors)
	_ = os.Remove(doc.FilePath)

	response.Success(c, http.StatusOK, gin.H{"message": "Upload deleted successfully"})
}
