package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/cloud"
	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ── Artworks ──────────────────────────────────────────────────────────────────

// GetArtworks godoc
// @Summary      List all artworks for the authenticated user
// @Tags         artworks
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /artworks [get]
func GetArtworks(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("artworks").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch artworks")
		return
	}
	defer cursor.Close(ctx)

	var artworks []models.Artwork
	if err := cursor.All(ctx, &artworks); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode artworks")
		return
	}
	if artworks == nil {
		artworks = []models.Artwork{}
	}

	response.Success(c, http.StatusOK, artworks)
}

// CreateArtwork godoc
// @Summary      Create a new artwork record
// @Tags         artworks
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      201 {object} response.APIResponse
// @Router       /artworks [post]
func CreateArtwork(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req models.ArtworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(req.SubSkillIDs) == 0 {
		response.Error(c, http.StatusBadRequest, "At least one sub skill is required")
		return
	}

	// Validate all sub skill IDs exist
	var subSkillObjIDs []primitive.ObjectID
	for _, sid := range req.SubSkillIDs {
		id, err := primitive.ObjectIDFromHex(sid)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid sub skill ID: "+sid)
			return
		}
		count, _ := db.Col("sub_skills").CountDocuments(ctx, bson.M{"_id": id})
		if count == 0 {
			response.Error(c, http.StatusBadRequest, "Sub skill not found: "+sid)
			return
		}
		subSkillObjIDs = append(subSkillObjIDs, id)
	}

	artwork := models.Artwork{
		ID:          primitive.NewObjectID(),
		UserID:      objID,
		SubSkillIDs: subSkillObjIDs,
		Title:       req.Title,
		Description: req.Description,
		PrivacyStatus: req.PrivacyStatus,
		Status:      "Pending",
		Votes:       []models.ArtworkVote{},
		UploadDate:  time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if _, err := db.Col("artworks").InsertOne(ctx, artwork); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create artwork")
		return
	}

	response.Success(c, http.StatusCreated, artwork)
}

// UpdateArtwork godoc
// @Summary      Update artwork metadata
// @Description  Updates title, description, and privacy of an artwork owned by the authenticated user.
// @Tags         artworks
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string                       true "Artwork ID"
// @Param        body body     models.ArtworkUpdateRequest   true "Updated fields"
// @Success      200  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      404  {object} response.APIResponse
// @Router       /artworks/{id} [put]
func UpdateArtwork(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	var req models.ArtworkUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch existing artwork to check status
	var existing models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID, "user_id": objID}).Decode(&existing); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found or does not belong to you")
		return
	}

	// Approved artworks: only privacy_status can change
	setFields := bson.M{
		"privacy_status": req.PrivacyStatus,
		"updated_at":     time.Now(),
	}
	if existing.Status != "Approved" {
		setFields["title"]       = req.Title
		setFields["description"] = req.Description
	}

	filter := bson.M{"_id": artworkID, "user_id": objID}
	update := bson.M{"$set": setFields}

	result, err := db.Col("artworks").UpdateOne(ctx, filter, update)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update artwork")
		return
	}
	if result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Artwork not found or does not belong to you")
		return
	}

	var updated models.Artwork
	_ = db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID}).Decode(&updated)
	response.Success(c, http.StatusOK, updated)
}

// DeleteArtwork godoc
// @Summary      Delete an artwork and its proofs
// @Description  Permanently removes an artwork and all linked proofs (files + records).
// @Tags         artworks
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Artwork ID"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /artworks/{id} [delete]
func DeleteArtwork(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch artwork first to get the Cloudinary public ID
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID, "user_id": objID}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found or does not belong to you")
		return
	}

	// Delete the artwork record
	db.Col("artworks").DeleteOne(ctx, bson.M{"_id": artworkID})

	// If it was approved, roll back the skill ranks so the level stays accurate
	if artwork.Status == "Approved" {
		DecrementRanksAfterRemoval(objID, artwork.SubSkillIDs)
	}

	// Delete artwork file from Cloudinary via the linked upload record
	var upload models.Upload
	if err := db.Col("uploads").FindOne(ctx, bson.M{"user_id": objID, "title": artwork.Title}).Decode(&upload); err == nil {
		if upload.CloudinaryPublicID != "" {
			_ = cloud.DeleteFile(upload.CloudinaryPublicID)
		}
		db.Col("uploads").DeleteOne(ctx, bson.M{"_id": upload.ID})
	}

	// Cascade-delete all proofs linked to this artwork (Cloudinary + DB)
	proofCursor, _ := db.Col("proofs").Find(ctx, bson.M{"artwork_id": artworkID})
	if proofCursor != nil {
		var proofs []models.Proof
		_ = proofCursor.All(ctx, &proofs)
		proofCursor.Close(ctx)
		for _, p := range proofs {
			if p.CloudinaryPublicID != "" {
				_ = cloud.DeleteFile(p.CloudinaryPublicID)
			}
		}
		db.Col("proofs").DeleteMany(ctx, bson.M{"artwork_id": artworkID})
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Artwork and its proofs deleted successfully"})
}
