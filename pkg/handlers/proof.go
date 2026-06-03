package handlers

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/cloud"
	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// maxProofSize is 10 MB.
const maxProofSize = 10 << 20

// allowedProofMIME is the set of MIME prefixes accepted for proof files.
var allowedProofMIME = []string{
	"image/",
	"video/",
	"application/pdf",
}

func isAllowedProofMIME(mime string) bool {
	lower := strings.ToLower(mime)
	for _, prefix := range allowedProofMIME {
		if strings.HasPrefix(lower, prefix) || lower == prefix {
			return true
		}
	}
	return false
}

// detectMIMEFromFilename guesses the MIME type from the file extension.
// Used as fallback when the browser omits Content-Type in the multipart part.
func detectMIMEFromFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		return "video/mp4"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	}
	return ""
}

// deriveFileType returns "image" or "video" based on the MIME type.
func deriveFileType(mime string) string {
	lower := strings.ToLower(mime)
	if strings.HasPrefix(lower, "video/") {
		return "video"
	}
	return "image" // images and PDFs treated as image proof
}

// ── Create Proof ─────────────────────────────────────────────────────────────

func CreateProof(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := c.Request.ParseMultipartForm(maxProofSize); err != nil {
		response.Error(c, http.StatusBadRequest, "File too large (max 10 MB) or invalid form data")
		return
	}

	artworkIDStr := c.PostForm("artwork_id")
	if artworkIDStr == "" {
		response.Error(c, http.StatusBadRequest, "artwork_id is required")
		return
	}
	artworkObjID, err := primitive.ObjectIDFromHex(artworkIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork_id")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "File field 'file' is required")
		return
	}
	defer file.Close()

	// Detect MIME from the part header; fall back to sniffing the filename extension
	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = detectMIMEFromFilename(header.Filename)
	}
	if !isAllowedProofMIME(mime) {
		response.Error(c, http.StatusBadRequest,
			"Invalid file type. Allowed: image/*, video/*, application/pdf")
		return
	}

	// Short-lived context just for the DB ownership check
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	// Validate artwork ownership
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(dbCtx, bson.M{
		"_id": artworkObjID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusBadRequest, "Artwork not found or does not belong to you")
		return
	}

	// Upload to Cloudinary (uses its own 30-second context internally)
	uploaded, err := cloud.UploadFile(file, "proofs")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upload proof to Cloudinary: "+err.Error())
		return
	}

	proof := models.Proof{
		ID:                 primitive.NewObjectID(),
		ArtworkID:          artworkObjID,
		FileURL:            uploaded.URL,
		FileType:           deriveFileType(mime),
		CloudinaryPublicID: uploaded.PublicID,
	}

	// Fresh context for the DB write — the old one may have expired during upload
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer saveCancel()

	if _, err := db.Col("proofs").InsertOne(saveCtx, proof); err != nil {
		_ = cloud.DeleteFile(uploaded.PublicID)
		response.Error(c, http.StatusInternalServerError, "Failed to save proof record")
		return
	}

	response.Success(c, http.StatusCreated, proof)
}

// ── List Proofs ──────────────────────────────────────────────────────────────

func GetProofs(c *gin.Context) {
	artworkIDStr := c.Query("artwork_id")
	if artworkIDStr == "" {
		response.Error(c, http.StatusBadRequest, "artwork_id query parameter is required")
		return
	}

	artworkObjID, err := primitive.ObjectIDFromHex(artworkIDStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork_id")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("proofs").Find(ctx, bson.M{"artwork_id": artworkObjID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch proofs")
		return
	}
	defer cursor.Close(ctx)

	var proofs []models.Proof
	if err := cursor.All(ctx, &proofs); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode proofs")
		return
	}
	if proofs == nil {
		proofs = []models.Proof{}
	}

	response.Success(c, http.StatusOK, proofs)
}

// ── Update Proof ─────────────────────────────────────────────────────────────

func UpdateProof(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	proofID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid proof ID")
		return
	}

	if err := c.Request.ParseMultipartForm(maxProofSize); err != nil {
		response.Error(c, http.StatusBadRequest, "File too large (max 10 MB) or invalid form data")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "File field 'file' is required")
		return
	}
	defer file.Close()

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = detectMIMEFromFilename(header.Filename)
	}
	if !isAllowedProofMIME(mime) {
		response.Error(c, http.StatusBadRequest, "Invalid file type. Allowed: image/*, video/*, application/pdf")
		return
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()

	var proof models.Proof
	if err := db.Col("proofs").FindOne(dbCtx, bson.M{"_id": proofID}).Decode(&proof); err != nil {
		response.Error(c, http.StatusNotFound, "Proof not found")
		return
	}

	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(dbCtx, bson.M{
		"_id": proof.ArtworkID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusForbidden, "You do not own this proof")
		return
	}

	// Upload new file to Cloudinary (uses its own 30-second context internally)
	uploaded, err := cloud.UploadFile(file, "proofs")
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upload proof to Cloudinary: "+err.Error())
		return
	}

	// Delete old file from Cloudinary
	if proof.CloudinaryPublicID != "" {
		_ = cloud.DeleteFile(proof.CloudinaryPublicID)
	}

	newType := deriveFileType(mime)
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer saveCancel()
	db.Col("proofs").UpdateOne(saveCtx, bson.M{"_id": proofID}, bson.M{
		"$set": bson.M{
			"file_url":             uploaded.URL,
			"file_type":            newType,
			"cloudinary_public_id": uploaded.PublicID,
		},
	})

	response.Success(c, http.StatusOK, gin.H{
		"id":        proofID.Hex(),
		"file_url":  uploaded.URL,
		"file_type": newType,
	})
}

// ── Delete Proof ─────────────────────────────────────────────────────────────

func DeleteProof(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	proofID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid proof ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var proof models.Proof
	if err := db.Col("proofs").FindOne(ctx, bson.M{"_id": proofID}).Decode(&proof); err != nil {
		response.Error(c, http.StatusNotFound, "Proof not found")
		return
	}

	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{
		"_id": proof.ArtworkID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusForbidden, "You do not own this artwork's proof")
		return
	}

	db.Col("proofs").DeleteOne(ctx, bson.M{"_id": proofID})

	// Delete from Cloudinary (best-effort)
	_ = cloud.DeleteFile(proof.CloudinaryPublicID)

	response.Success(c, http.StatusOK, gin.H{"message": "Proof deleted successfully"})
}
