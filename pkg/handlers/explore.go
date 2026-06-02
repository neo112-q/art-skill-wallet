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
	SkillName  string   `json:"skill_name"`  // primary skill (backward compat)
	SkillNames []string `json:"skill_names"` // all skills
	ThumbUrl   string   `json:"thumb_url"`
	UploadDate string   `json:"upload_date"`
}

// ExploreSkill is the skill rank shape returned in the explore feed.
type ExploreSkill struct {
	SubSkillID    string `json:"sub_skill_id"`
	DisplayName   string `json:"display_name"`
	MainSkillName string `json:"main_skill_name"`
	Rank          string `json:"rank"`
}

// ExploreArtist is the public-facing shape returned by the explore feed.
type ExploreArtist struct {
	ID             string           `json:"id"`
	Username       string           `json:"username"`
	Bio            string           `json:"bio"`
	ProfilePicture string           `json:"profile_picture"`
	Skills         []ExploreSkill   `json:"skills"`
	Artworks       []ExploreArtwork `json:"artworks"`
	ArtworkCount   int              `json:"artwork_count"`
}

func GetExplore(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Users
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

	// Sub skills + main skills for name lookup
	subSkillMap, mainSkillMap := loadSkillMaps(ctx)

	// Skill ranks grouped by user
	rankCursor, _ := db.Col("user_skill_ranks").Find(ctx, bson.M{})
	var allRanks []models.UserSkillRank
	if rankCursor != nil {
		_ = rankCursor.All(ctx, &allRanks)
		rankCursor.Close(ctx)
	}
	rankMap := make(map[string][]ExploreSkill)
	for _, r := range allRanks {
		if r.Rank == "" {
			continue
		}
		uid := r.UserID.Hex()
		sub := subSkillMap[r.SubSkillID.Hex()]
		main := mainSkillMap[sub.MainSkillID.Hex()]
		rankMap[uid] = append(rankMap[uid], ExploreSkill{
			SubSkillID:    r.SubSkillID.Hex(),
			DisplayName:   sub.DisplayName,
			MainSkillName: main.Name,
			Rank:          r.Rank,
		})
	}

	// Public + Approved artworks
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
		ea := buildExploreArtwork(a, subSkillMap, mainSkillMap, "")
		artworkMap[uid] = append(artworkMap[uid], ea)
	}

	artists := make([]ExploreArtist, 0, len(users))
	for _, u := range users {
		skills := rankMap[u.ID.Hex()]
		if skills == nil {
			skills = []ExploreSkill{}
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

func GetPublicProfile(c *gin.Context) {
	userObjID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := db.Col("users").FindOne(ctx, bson.M{"_id": userObjID}).Decode(&user); err != nil {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	subSkillMap, mainSkillMap := loadSkillMaps(ctx)

	// Skill ranks for this user
	rankCursor, _ := db.Col("user_skill_ranks").Find(ctx, bson.M{"user_id": userObjID})
	var ranks []models.UserSkillRank
	if rankCursor != nil {
		_ = rankCursor.All(ctx, &ranks)
		rankCursor.Close(ctx)
	}
	skills := make([]ExploreSkill, 0)
	for _, r := range ranks {
		if r.Rank == "" {
			continue
		}
		sub := subSkillMap[r.SubSkillID.Hex()]
		main := mainSkillMap[sub.MainSkillID.Hex()]
		skills = append(skills, ExploreSkill{
			SubSkillID:    r.SubSkillID.Hex(),
			DisplayName:   sub.DisplayName,
			MainSkillName: main.Name,
			Rank:          r.Rank,
		})
	}

	// Uploads for thumbnail lookup
	uploadCursor, _ := db.Col("uploads").Find(ctx, bson.M{"user_id": userObjID})
	var uploads []models.Upload
	if uploadCursor != nil {
		_ = uploadCursor.All(ctx, &uploads)
		uploadCursor.Close(ctx)
	}
	thumbMap := make(map[string]string)
	for _, u := range uploads {
		var imageURL string
		if u.FileURL != "" {
			imageURL = u.FileURL
		} else if u.FilePath != "" {
			storedName := filepath.Base(strings.ReplaceAll(u.FilePath, "\\", "/"))
			if storedName != "" && storedName != "." {
				imageURL = "/uploads/" + storedName
			}
		}
		if imageURL != "" {
			thumbMap[strings.ToLower(u.Title)] = imageURL
		}
	}

	// Public + Approved artworks
	artCursor, _ := db.Col("artworks").Find(ctx, bson.M{
		"user_id":        userObjID,
		"privacy_status": "Public",
		"status":         "Approved",
	})
	var allArtworks []models.Artwork
	if artCursor != nil {
		_ = artCursor.All(ctx, &allArtworks)
		artCursor.Close(ctx)
	}

	artworks := make([]ExploreArtwork, 0, len(allArtworks))
	for _, a := range allArtworks {
		thumb := thumbMap[strings.ToLower(a.Title)]
		artworks = append(artworks, buildExploreArtwork(a, subSkillMap, mainSkillMap, thumb))
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

// loadSkillMaps fetches sub_skills and main_skills into lookup maps.
func loadSkillMaps(ctx context.Context) (map[string]models.SubSkill, map[string]models.MainSkill) {
	subMap := make(map[string]models.SubSkill)
	mainMap := make(map[string]models.MainSkill)

	subCursor, _ := db.Col("sub_skills").Find(ctx, bson.M{})
	if subCursor != nil {
		var subs []models.SubSkill
		_ = subCursor.All(ctx, &subs)
		subCursor.Close(ctx)
		for _, s := range subs {
			subMap[s.ID.Hex()] = s
		}
	}

	mainCursor, _ := db.Col("main_skills").Find(ctx, bson.M{})
	if mainCursor != nil {
		var mains []models.MainSkill
		_ = mainCursor.All(ctx, &mains)
		mainCursor.Close(ctx)
		for _, m := range mains {
			mainMap[m.ID.Hex()] = m
		}
	}

	return subMap, mainMap
}

// buildExploreArtwork converts an Artwork to ExploreArtwork using the skill maps.
func buildExploreArtwork(a models.Artwork, subMap map[string]models.SubSkill, mainMap map[string]models.MainSkill, thumb string) ExploreArtwork {
	var skillNames []string
	for _, sid := range a.SubSkillIDs {
		sub := subMap[sid.Hex()]
		name := sub.DisplayName
		if main, ok := mainMap[sub.MainSkillID.Hex()]; ok {
			name = main.Name + " › " + sub.DisplayName
		}
		if name != "" {
			skillNames = append(skillNames, name)
		}
	}
	if skillNames == nil {
		skillNames = []string{}
	}
	primary := ""
	if len(skillNames) > 0 {
		primary = skillNames[0]
	}
	return ExploreArtwork{
		ID:         a.ID.Hex(),
		Title:      a.Title,
		SkillName:  primary,
		SkillNames: skillNames,
		ThumbUrl:   thumb,
		UploadDate: a.UploadDate.Format("2 January 2006"),
	}
}
