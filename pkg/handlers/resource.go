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
		ID:           primitive.NewObjectID(),
		UserID:       objID,
		Name:         req.Name,
		Level:        req.Level,
		ProgressData: req.ProgressData,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if _, err := db.Col("skills").InsertOne(ctx, skill); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create skill")
		return
	}

	response.Success(c, http.StatusCreated, skill)
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

	artwork := models.Artwork{
		ID:                  primitive.NewObjectID(),
		UserID:              objID,
		SkillID:             skillObjID,
		Path:                req.Path,
		Date:                req.Date,
		Privacy:             req.Privacy,
		ProgressCheckpoints: req.ProgressCheckpoints,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
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
