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
	Type       string    `json:"type"` // "upload"
	Date       time.Time `json:"date"`
	Title      string    `json:"title"`
	SkillNames []string  `json:"skill_names"`
	Status     string    `json:"status"`
	ArtworkID  string    `json:"artwork_id,omitempty"`
}

// GetHistory godoc
// @Summary      Get the authenticated user's artwork submission history
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

	subSkillMap, mainSkillMap := loadSkillMaps(ctx)

	events := make([]HistoryEvent, 0, len(artworks))
	for _, a := range artworks {
		var skillNames []string
		for _, sid := range a.SubSkillIDs {
			sub := subSkillMap[sid.Hex()]
			name := sub.DisplayName
			if main, ok := mainSkillMap[sub.MainSkillID.Hex()]; ok {
				name = main.Name + " › " + sub.DisplayName
			}
			if name != "" {
				skillNames = append(skillNames, name)
			}
		}
		if skillNames == nil {
			skillNames = []string{}
		}
		events = append(events, HistoryEvent{
			Type:       "upload",
			Date:       a.UploadDate,
			Title:      a.Title,
			SkillNames: skillNames,
			Status:     a.Status,
			ArtworkID:  a.ID.Hex(),
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date.After(events[j].Date)
	})

	response.Success(c, http.StatusOK, events)
}
