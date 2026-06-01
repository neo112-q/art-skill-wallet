package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// HistoryEvent is a single timeline entry returned to the frontend.
type HistoryEvent struct {
	Type      string    `json:"type"`       // "upload" | "skill_created"
	Date      time.Time `json:"date"`
	Title     string    `json:"title"`
	SkillName string    `json:"skill_name"`
	Level     string    `json:"level"`
	ArtworkID string    `json:"artwork_id,omitempty"`
	SkillID   string    `json:"skill_id,omitempty"`
}

// GetHistory godoc
// @Summary      Activity timeline for the logged-in user
// @Description  Merges artwork uploads and skill creations into a single date-sorted timeline.
// @Tags         history
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /history [get]
func GetHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ── Fetch artworks ───────────────────────────────────────────────
	artCursor, err := db.Col("artworks").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch artworks")
		return
	}
	defer artCursor.Close(ctx)

	var artworks []models.Artwork
	if err := artCursor.All(ctx, &artworks); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode artworks")
		return
	}

	// ── Fetch skills ─────────────────────────────────────────────────
	skillCursor, err := db.Col("skills").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch skills")
		return
	}
	defer skillCursor.Close(ctx)

	var skills []models.Skill
	if err := skillCursor.All(ctx, &skills); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode skills")
		return
	}

	// Build skill lookup
	skillMap := make(map[string]models.Skill)
	for _, s := range skills {
		skillMap[s.ID.Hex()] = s
	}

	// ── Build timeline ───────────────────────────────────────────────
	events := make([]HistoryEvent, 0, len(artworks)+len(skills))

	for _, a := range artworks {
		sName := ""
		sLevel := ""
		if s, ok := skillMap[a.SkillID.Hex()]; ok {
			sName = s.SkillName
			sLevel = s.Level
		}
		events = append(events, HistoryEvent{
			Type:      "upload",
			Date:      a.UploadDate,
			Title:     a.Title,
			SkillName: sName,
			Level:     sLevel,
			ArtworkID: a.ID.Hex(),
		})
	}

	for _, s := range skills {
		events = append(events, HistoryEvent{
			Type:      "skill_created",
			Date:      s.CreatedAt,
			Title:     s.SkillName + " added",
			SkillName: s.SkillName,
			Level:     s.Level,
			SkillID:   s.ID.Hex(),
		})
	}

	// Sort newest first
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.After(events[j].Date)
	})

	response.Success(c, http.StatusOK, events)
}
