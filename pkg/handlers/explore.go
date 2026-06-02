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

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ExploreArtwork is a public artwork shown on the explore page.
type ExploreArtwork struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	SkillName  string   `json:"skill_name"`   // primary skill (backward compat)
	SkillNames []string `json:"skill_names"`  // all skills
	ThumbUrl   string   `json:"thumb_url"`
	UploadDate string   `json:"upload_date"`
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

// GetPublicProfile godoc
// @Summary      Get a single user's public profile
// @Tags         explore
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /users/{id}/public [get]
func GetPublicProfile(c *gin.Context) {
	userObjID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch user
	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err != nil {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	// Fetch user's skills
	skillCursor, _ := db.Col("skills").Find(ctx, bson.M{"user_id": userObjID})
	defer skillCursor.Close(ctx)
	var skills []models.Skill
	_ = skillCursor.All(ctx, &skills)
	if skills == nil {
		skills = []models.Skill{}
	}

	// Build skill name map
	skillNameMap := make(map[string]string)
	for _, s := range skills {
		skillNameMap[s.ID.Hex()] = s.SkillName
	}

	// Fetch user's uploads to build title → image URL map
	uploadCursor, _ := db.Col("uploads").Find(ctx, bson.M{"user_id": userObjID})
	defer uploadCursor.Close(ctx)
	var uploads []models.Upload
	_ = uploadCursor.All(ctx, &uploads)

	thumbMap := make(map[string]string) // lowercase title → /uploads/<storedFilename>
	for _, u := range uploads {
		// FilePath is the full disk path e.g. "./uploads/686abc123.jpg"
		// Extract just the stored filename using filepath.Base
		storedName := filepath.Base(strings.ReplaceAll(u.FilePath, "\\", "/"))
		if storedName != "" && storedName != "." {
			thumbMap[strings.ToLower(u.Title)] = "/uploads/" + storedName
		}
	}

	// Fetch only Public + Approved artworks
	artCursor, _ := db.Col("artworks").Find(ctx, bson.M{
		"user_id":        userObjID,
		"privacy_status": "Public",
		"status":         "Approved",
	})
	defer artCursor.Close(ctx)
	var allArtworks []models.Artwork
	_ = artCursor.All(ctx, &allArtworks)

	artworks := make([]ExploreArtwork, 0, len(allArtworks))
	for _, a := range allArtworks {
		// Collect all skill names — prefer skill_ids, fall back to skill_id
		allIDs := a.SkillIDs
		if len(allIDs) == 0 && !a.SkillID.IsZero() {
			allIDs = append(allIDs, a.SkillID)
		}
		var skillNames []string
		for _, sid := range allIDs {
			if name := skillNameMap[sid.Hex()]; name != "" {
				skillNames = append(skillNames, name)
			}
		}
		if skillNames == nil {
			skillNames = []string{}
		}
		primarySkill := ""
		if len(skillNames) > 0 {
			primarySkill = skillNames[0]
		}

		artworks = append(artworks, ExploreArtwork{
			ID:         a.ID.Hex(),
			Title:      a.Title,
			SkillName:  primarySkill,
			SkillNames: skillNames,
			ThumbUrl:   thumbMap[strings.ToLower(a.Title)],
			UploadDate: a.UploadDate.Format("2 January 2006"),
		})
	}

	response.Success(c, http.StatusOK, ExploreArtist{
		ID:             user.ID.Hex(),
		Username:       user.Username,
		Bio:            user.Bio,
		ProfilePicture: user.ProfilePicture,
		Skills:         skills,
		Artworks:       artworks,
		ArtworkCount:   len(artworks),
	})
}
