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

// SubmissionRow is the enriched row returned to the admin table.
type SubmissionRow struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Initials  string    `json:"initials"`
	Title     string    `json:"title"`
	Skill     string    `json:"skill"`    // primary skill name (backward compat)
	Skills    []string  `json:"skills"`   // all skill names
	Level     string    `json:"level"`
	Status    string    `json:"status"`
	Submitted time.Time `json:"submitted"`
}

// GetSubmissionsEnriched godoc
// @Summary      Admin — list all submissions with user and skill names
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /admin/submissions [get]
func GetSubmissionsEnriched(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Artworks
	artCursor, err := db.Col("artworks").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch artworks")
		return
	}
	defer artCursor.Close(ctx)
	var artworks []models.Artwork
	_ = artCursor.All(ctx, &artworks)

	// Users
	userCursor, err := db.Col("users").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer userCursor.Close(ctx)
	var users []models.User
	_ = userCursor.All(ctx, &users)
	userMap := make(map[string]string)
	for _, u := range users {
		userMap[u.ID.Hex()] = u.Username
	}

	// Skills
	skillCursor, err := db.Col("skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch skills")
		return
	}
	defer skillCursor.Close(ctx)
	var skills []models.Skill
	_ = skillCursor.All(ctx, &skills)
	skillMap := make(map[string]models.Skill)
	for _, s := range skills {
		skillMap[s.ID.Hex()] = s
	}

	rows := make([]SubmissionRow, 0, len(artworks))
	for _, a := range artworks {
		uname := userMap[a.UserID.Hex()]
		if uname == "" {
			uname = "Unknown"
		}
		initials := ""
		if len(uname) >= 2 {
			initials = string([]rune(uname)[:2])
		}
		// Collect all skill IDs — prefer skill_ids array, fall back to single skill_id
		allIDs := a.SkillIDs
		if len(allIDs) == 0 && !a.SkillID.IsZero() {
			allIDs = []primitive.ObjectID{a.SkillID}
		}

		var skillNames []string
		sLevel := ""
		for _, sid := range allIDs {
			if sk, ok := skillMap[sid.Hex()]; ok {
				skillNames = append(skillNames, sk.SkillName)
				if sLevel == "" {
					sLevel = sk.Level // use level from first skill
				}
			}
		}

		// Primary skill name for backward compat
		primarySkill := ""
		if len(skillNames) > 0 {
			primarySkill = skillNames[0]
		}

		status := a.Status
		if status == "" {
			status = "Pending"
		}
		rows = append(rows, SubmissionRow{
			ID:        a.ID.Hex(),
			User:      uname,
			Initials:  initials,
			Title:     a.Title,
			Skill:     primarySkill,
			Skills:    skillNames,
			Level:     sLevel,
			Status:    status,
			Submitted: a.UploadDate,
		})
	}

	response.Success(c, http.StatusOK, rows)
}

// GetPublicArtworkProofs returns proofs for a public+approved artwork — no auth required.
func GetPublicArtworkProofs(c *gin.Context) {
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Only allow proofs for public + approved artworks
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{
		"_id":            artworkID,
		"privacy_status": "Public",
		"status":         "Approved",
	}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found or not public")
		return
	}

	cursor, err := db.Col("proofs").Find(ctx, bson.M{"artwork_id": artworkID})
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

// GetSubmissionProofs godoc
// @Summary      Admin — view proofs for a submission
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Artwork ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/submissions/{id}/proofs [get]
func GetSubmissionProofs(c *gin.Context) {
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("proofs").Find(ctx, bson.M{"artwork_id": artworkID})
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

// UpdateSubmissionStatus godoc
// @Summary      Admin — approve or reject a submission
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path   string                     true "Artwork ID"
// @Param        body body   models.StatusUpdateRequest  true "New status"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /admin/submissions/{id} [put]
func UpdateSubmissionStatus(c *gin.Context) {
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	var req models.StatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("artworks").UpdateOne(ctx,
		bson.M{"_id": artworkID},
		bson.M{"$set": bson.M{"status": req.Status, "updated_at": time.Now()}},
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update status")
		return
	}
	if result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Artwork not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Status updated to " + req.Status})
}
