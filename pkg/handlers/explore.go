package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ExploreArtist is the public-facing shape returned by the explore feed.
type ExploreArtist struct {
	ID             string         `json:"id"`
	Username       string         `json:"username"`
	Bio            string         `json:"bio"`
	ProfilePicture string         `json:"profile_picture"`
	Skills         []models.Skill `json:"skills"`
}

// GetExplore godoc
// @Summary      Public explore feed
// @Description  Returns all users with their skills for the public explore page. No auth required.
// @Tags         explore
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /explore [get]
func GetExplore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Fetch all users
	userCursor, err := db.Col("users").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer userCursor.Close(ctx)

	var users []models.User
	if err := userCursor.All(ctx, &users); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode users")
		return
	}

	// 2. Fetch all skills in one query
	skillCursor, err := db.Col("skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch skills")
		return
	}
	defer skillCursor.Close(ctx)

	var allSkills []models.Skill
	if err := skillCursor.All(ctx, &allSkills); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode skills")
		return
	}

	// 3. Group skills by user_id
	skillMap := make(map[string][]models.Skill)
	for _, s := range allSkills {
		uid := s.UserID.Hex()
		skillMap[uid] = append(skillMap[uid], s)
	}

	// 4. Build the response
	artists := make([]ExploreArtist, 0, len(users))
	for _, u := range users {
		skills := skillMap[u.ID.Hex()]
		if skills == nil {
			skills = []models.Skill{}
		}
		artists = append(artists, ExploreArtist{
			ID:             u.ID.Hex(),
			Username:       u.Username,
			Bio:            u.Bio,
			ProfilePicture: u.ProfilePicture,
			Skills:         skills,
		})
	}

	response.Success(c, http.StatusOK, artists)
}
