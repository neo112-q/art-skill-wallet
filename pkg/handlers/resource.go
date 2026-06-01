package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ── Skills ────────────────────────────────────────────────────────────────────

// GetSkills godoc
// @Summary      List all skills for the logged-in user
// @Description  Returns every skill document belonging to the authenticated user.
// @Tags         skills
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Router       /skills [get]
func GetSkills(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("skills").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch skills")
		return
	}
	defer cursor.Close(ctx)

	var skills []models.Skill
	if err := cursor.All(ctx, &skills); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode skills")
		return
	}
	if skills == nil {
		skills = []models.Skill{}
	}

	response.Success(c, http.StatusOK, skills)
}

// CreateSkill godoc
// @Summary      Create a new skill
// @Description  Adds a new skill card for the authenticated user with name, level, and optional development guide.
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body     models.SkillRequest true "Skill payload"
// @Success      201  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Router       /skills [post]
func CreateSkill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skill := models.Skill{
		ID:               primitive.NewObjectID(),
		UserID:           objID,
		SkillName:        req.SkillName,
		Level:            req.Level,
		DevelopmentGuide: req.DevelopmentGuide,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if _, err := db.Col("skills").InsertOne(ctx, skill); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create skill")
		return
	}

	response.Success(c, http.StatusCreated, skill)
}

// UpdateSkill godoc
// @Summary      Update an existing skill by ID
// @Description  Updates skill_name, level, and development_guide for a skill owned by the authenticated user.
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path     string              true "Skill ID (ObjectID hex)"
// @Param        body body     models.SkillRequest  true "Updated skill payload"
// @Success      200  {object} response.APIResponse
// @Failure      400  {object} response.APIResponse
// @Failure      404  {object} response.APIResponse
// @Router       /skills/{id} [put]
func UpdateSkill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	var req models.SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": skillID, "user_id": objID}
	update := bson.M{"$set": bson.M{
		"skill_name":        req.SkillName,
		"level":             req.Level,
		"development_guide": req.DevelopmentGuide,
		"updated_at":        time.Now(),
	}}

	result, err := db.Col("skills").UpdateOne(ctx, filter, update)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update skill")
		return
	}
	if result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Skill not found or does not belong to you")
		return
	}

	// Return the updated document
	var updated models.Skill
	if err := db.Col("skills").FindOne(ctx, bson.M{"_id": skillID}).Decode(&updated); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch updated skill")
		return
	}

	response.Success(c, http.StatusOK, updated)
}

// DeleteSkill godoc
// @Summary      Delete a skill by ID
// @Description  Permanently removes a skill owned by the authenticated user.
// @Tags         skills
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Skill ID (ObjectID hex)"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /skills/{id} [delete]
func DeleteSkill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("skills").DeleteOne(ctx, bson.M{"_id": skillID, "user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete skill")
		return
	}
	if result.DeletedCount == 0 {
		response.Error(c, http.StatusNotFound, "Skill not found or does not belong to you")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Skill deleted successfully"})
}

// ── Artworks ──────────────────────────────────────────────────────────────────

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

	skillObjID, err := primitive.ObjectIDFromHex(req.SkillID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate that the skill exists and belongs to this user
	var skill models.Skill
	if err := db.Col("skills").FindOne(ctx, bson.M{"_id": skillObjID, "user_id": objID}).Decode(&skill); err != nil {
		response.Error(c, http.StatusBadRequest, "Skill not found or does not belong to you")
		return
	}

	artwork := models.Artwork{
		ID:            primitive.NewObjectID(),
		UserID:        objID,
		SkillID:       skillObjID,
		Title:         req.Title,
		Description:   req.Description,
		PrivacyStatus: req.PrivacyStatus,
		UploadDate:    time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if _, err := db.Col("artworks").InsertOne(ctx, artwork); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create artwork")
		return
	}

	response.Success(c, http.StatusCreated, artwork)
}

// ── Admin ─────────────────────────────────────────────────────────────────────

func GetSubmissions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("artworks").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch submissions")
		return
	}
	defer cursor.Close(ctx)

	var artworks []models.Artwork
	if err := cursor.All(ctx, &artworks); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode submissions")
		return
	}
	if artworks == nil {
		artworks = []models.Artwork{}
	}

	response.Success(c, http.StatusOK, artworks)
}
