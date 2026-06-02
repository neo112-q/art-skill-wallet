package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

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

// deriveFileType returns "image" or "video" based on the MIME type.
func deriveFileType(mime string) string {
	lower := strings.ToLower(mime)
	if strings.HasPrefix(lower, "video/") {
		return "video"
	}
	return "image" // images and PDFs treated as image proof
}

// ── Create Proof ─────────────────────────────────────────────────────────────

// CreateProof godoc
// @Summary      Upload a proof file linked to an artwork
// @Description  Accepts a multipart/form-data request with a file and an artwork_id.
//
//	Validates ownership of the artwork, saves the file, and creates a proof record.
//
// @Tags         proofs
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file       formData file   true "Proof file (image/video/PDF, max 10 MB)"
// @Param        artwork_id formData string true "Artwork ID this proof belongs to"
// @Success      201 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      500 {object} response.APIResponse
// @Router       /proofs [post]
func CreateProof(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// ── Parse form ──────────────────────────────────────────────────────
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

	// ── Validate MIME type (#5 — backend file type validation) ───────
	mime := header.Header.Get("Content-Type")
	if !isAllowedProofMIME(mime) {
		response.Error(c, http.StatusBadRequest,
			"Invalid file type. Allowed: image/*, video/*, application/pdf")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ── Validate artwork ownership ──────────────────────────────────
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{
		"_id": artworkObjID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusBadRequest, "Artwork not found or does not belong to you")
		return
	}

	// ── Save file ───────────────────────────────────────────────────
	proofID := primitive.NewObjectID()
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("proof-%s%s", proofID.Hex(), ext)
	dir := filepath.Join(".", "uploads")
	_ = os.MkdirAll(dir, 0o755)
	destPath := filepath.Join(dir, storedName)

	if err := c.SaveUploadedFile(header, destPath); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save proof file")
		return
	}

	fileURL := "/uploads/" + storedName

	proof := models.Proof{
		ID:        proofID,
		ArtworkID: artworkObjID,
		FileURL:   fileURL,
		FileType:  deriveFileType(mime),
	}

	if _, err := db.Col("proofs").InsertOne(ctx, proof); err != nil {
		_ = os.Remove(destPath)
		response.Error(c, http.StatusInternalServerError, "Failed to save proof record")
		return
	}

	response.Success(c, http.StatusCreated, proof)
}

// ── List Proofs ──────────────────────────────────────────────────────────────

// GetProofs godoc
// @Summary      List proofs for an artwork
// @Description  Returns all proof files linked to the given artwork_id.
// @Tags         proofs
// @Security     BearerAuth
// @Produce      json
// @Param        artwork_id query string true "Artwork ID to fetch proofs for"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Router       /proofs [get]
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

// UpdateProof godoc
// @Summary      Replace the file of an existing proof
// @Tags         proofs
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        id   path     string true "Proof ID"
// @Param        file formData file   true "New proof file (image/video/PDF, max 10 MB)"
// @Success      200 {object} response.APIResponse
// @Router       /proofs/{id} [put]
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
	if !isAllowedProofMIME(mime) {
		response.Error(c, http.StatusBadRequest, "Invalid file type. Allowed: image/*, video/*, application/pdf")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch existing proof
	var proof models.Proof
	if err := db.Col("proofs").FindOne(ctx, bson.M{"_id": proofID}).Decode(&proof); err != nil {
		response.Error(c, http.StatusNotFound, "Proof not found")
		return
	}

	// Verify artwork ownership
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{
		"_id": proof.ArtworkID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusForbidden, "You do not own this proof")
		return
	}

	// Save new file
	ext := filepath.Ext(header.Filename)
	storedName := fmt.Sprintf("proof-%s%s", proofID.Hex(), ext)
	dir := filepath.Join(".", "uploads")
	_ = os.MkdirAll(dir, 0o755)
	destPath := filepath.Join(dir, storedName)

	if err := c.SaveUploadedFile(header, destPath); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save proof file")
		return
	}

	// Delete old file if different name
	if proof.FileURL != "" {
		oldName := filepath.Base(proof.FileURL)
		if oldName != storedName {
			_ = os.Remove(filepath.Join(".", "uploads", oldName))
		}
	}

	newURL := "/uploads/" + storedName
	newType := deriveFileType(mime)

	db.Col("proofs").UpdateOne(ctx, bson.M{"_id": proofID}, bson.M{
		"$set": bson.M{"file_url": newURL, "file_type": newType},
	})

	response.Success(c, http.StatusOK, gin.H{"id": proofID.Hex(), "file_url": newURL, "file_type": newType})
}

// ── Delete Proof ─────────────────────────────────────────────────────────────

// DeleteProof godoc
// @Summary      Delete a proof by ID
// @Description  Removes the proof file from disk and deletes the record. Validates artwork ownership.
// @Tags         proofs
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Proof ID"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /proofs/{id} [delete]
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

	// Fetch the proof
	var proof models.Proof
	if err := db.Col("proofs").FindOne(ctx, bson.M{"_id": proofID}).Decode(&proof); err != nil {
		response.Error(c, http.StatusNotFound, "Proof not found")
		return
	}

	// Verify the artwork belongs to this user
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{
		"_id": proof.ArtworkID, "user_id": objID,
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusForbidden, "You do not own this artwork's proof")
		return
	}

	// Delete record
	db.Col("proofs").DeleteOne(ctx, bson.M{"_id": proofID})

	// Remove file — extract filename from URL "/uploads/proof-xxx.ext"
	if proof.FileURL != "" {
		fname := filepath.Base(proof.FileURL)
		_ = os.Remove(filepath.Join(".", "uploads", fname))
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Proof deleted successfully"})
}
