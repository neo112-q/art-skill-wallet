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

// ExploreArtwork is a public artwork shown on the explore page.
type ExploreArtwork struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	SkillName  string `json:"skill_name"`
	UploadDate string `json:"upload_date"`
}

// ExploreArtist is the public-facing shape returned by the explore feed.
type ExploreArtist struct {
	ID             string           `json:"id"`
	Username       string           `json:"username"`
	Bio            string           `json:"bio"`
	ProfilePicture string           `json:"profile_picture"`
	Skills         []models.Skill   `json:"skills"`
	Artworks       []ExploreArtwork `json:"artworks"`
	ArtworkCount   int              `json:"artwork_count"`
}

// GetExplore godoc
// @Summary      Public explore feed
// @Description  Returns all users with their skills and public+approved artworks. No auth required.
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

	// 2. Fetch all skills
	skillCursor, err := db.Col("skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch skills")
		return
	}
	defer skillCursor.Close(ctx)

	var allSkills []models.Skill
	_ = skillCursor.All(ctx, &allSkills)

	skillMap := make(map[string][]models.Skill)
	skillNameMap := make(map[string]string) // skill_id → skill_name
	for _, s := range allSkills {
		uid := s.UserID.Hex()
		skillMap[uid] = append(skillMap[uid], s)
		skillNameMap[s.ID.Hex()] = s.SkillName
	}

	// 3. Fetch only Public + Approved artworks (#3 — privacy filter)
	artCursor, err := db.Col("artworks").Find(ctx, bson.M{
		"privacy_status": "Public",
		"status":         "Approved",
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch artworks")
		return
	}
	defer artCursor.Close(ctx)

	var allArtworks []models.Artwork
	_ = artCursor.All(ctx, &allArtworks)

	artworkMap := make(map[string][]ExploreArtwork)
	for _, a := range allArtworks {
		uid := a.UserID.Hex()
		artworkMap[uid] = append(artworkMap[uid], ExploreArtwork{
			ID:         a.ID.Hex(),
			Title:      a.Title,
			SkillName:  skillNameMap[a.SkillID.Hex()],
			UploadDate: a.UploadDate.Format("2 January 2006"),
		})
	}

	// 4. Build the response
	artists := make([]ExploreArtist, 0, len(users))
	for _, u := range users {
		skills := skillMap[u.ID.Hex()]
		if skills == nil {
			skills = []models.Skill{}
		}
		arts := artworkMap[u.ID.Hex()]
		if arts == nil {
			arts = []ExploreArtwork{}
		}
		artists = append(artists, ExploreArtist{
			ID:             u.ID.Hex(),
			Username:       u.Username,
			Bio:            u.Bio,
			ProfilePicture: u.ProfilePicture,
			Skills:         skills,
			Artworks:       arts,
			ArtworkCount:   len(arts),
		})
	}

	response.Success(c, http.StatusOK, artists)
}
