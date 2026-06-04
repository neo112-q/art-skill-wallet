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

// SubmissionRow is the enriched row returned to the admin table.
type SubmissionRow struct {
	ID               string    `json:"id"`
	User             string    `json:"user"`
	Initials         string    `json:"initials"`
	Title            string    `json:"title"`
	Skill            string    `json:"skill"`    // primary skill name (backward compat)
	Skills           []string  `json:"skills"`   // all skill names
	Level            string    `json:"level"`
	Status           string    `json:"status"`
	Submitted        time.Time `json:"submitted"`
	VoteApproveCount int       `json:"vote_approve_count"`
	VoteRejectCount  int       `json:"vote_reject_count"`
	FileURL          string    `json:"file_url"`
	FileType         string    `json:"file_type"`
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

	// Sub skills
	subSkillCursor, err := db.Col("sub_skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch sub skills")
		return
	}
	defer subSkillCursor.Close(ctx)
	var subSkills []models.SubSkill
	_ = subSkillCursor.All(ctx, &subSkills)
	subSkillMap := make(map[string]models.SubSkill)
	for _, s := range subSkills {
		subSkillMap[s.ID.Hex()] = s
	}

	// Main skills
	mainSkillCursor, err := db.Col("main_skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch main skills")
		return
	}
	defer mainSkillCursor.Close(ctx)
	var mainSkills []models.MainSkill
	_ = mainSkillCursor.All(ctx, &mainSkills)
	mainSkillMap := make(map[string]models.MainSkill)
	for _, m := range mainSkills {
		mainSkillMap[m.ID.Hex()] = m
	}

	// Uploads — keyed by "userID:title" for artwork image lookup
	uploadCursor, _ := db.Col("uploads").Find(ctx, bson.M{})
	var uploads []models.Upload
	if uploadCursor != nil {
		_ = uploadCursor.All(ctx, &uploads)
		uploadCursor.Close(ctx)
	}
	uploadMap := make(map[string]models.Upload)
	for _, u := range uploads {
		key := u.UserID.Hex() + ":" + u.Title
		uploadMap[key] = u
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

		var skillNames []string
		for _, sid := range a.SubSkillIDs {
			if sub, ok := subSkillMap[sid.Hex()]; ok {
				name := sub.DisplayName
				if main, ok := mainSkillMap[sub.MainSkillID.Hex()]; ok {
					name = main.Name + " › " + sub.DisplayName
				}
				skillNames = append(skillNames, name)
			}
		}

		primarySkill := ""
		if len(skillNames) > 0 {
			primarySkill = skillNames[0]
		}

		status := a.Status
		if status == "" {
			status = "Pending"
		}
		fileURL := ""
		fileType := ""
		if upload, ok := uploadMap[a.UserID.Hex()+":"+a.Title]; ok {
			fileURL = upload.FileURL
			fileType = upload.MimeType
		}

		rows = append(rows, SubmissionRow{
			ID:               a.ID.Hex(),
			User:             uname,
			Initials:         initials,
			Title:            a.Title,
			Skill:            primarySkill,
			Skills:           skillNames,
			Level:            "",
			Status:           status,
			Submitted:        a.UploadDate,
			VoteApproveCount: a.VoteApproveCount,
			VoteRejectCount:  a.VoteRejectCount,
			FileURL:          fileURL,
			FileType:         fileType,
		})
	}

	response.Success(c, http.StatusOK, rows)
}

// GetPublicArtworkProofs godoc
// @Summary      Get proofs for a public approved artwork (no auth required)
// @Tags         admin
// @Produce      json
// @Param        id path string true "Artwork ID"
// @Success      200 {object} response.APIResponse
// @Router       /public/artworks/{id}/proofs [get]
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

	// Load artwork first to check previous status and get skill IDs
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found")
		return
	}

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

	// Update skill ranks only when transitioning to Approved for the first time
	// (prevents double-counting if admin approves the same artwork more than once)
	if req.Status == "Approved" && artwork.Status != "Approved" {
		UpdateRanksAfterApproval(artwork.UserID, artwork.SubSkillIDs)
	}
	// Roll back ranks when moving an approved artwork out of the Approved state
	if req.Status != "Approved" && artwork.Status == "Approved" {
		DecrementRanksAfterRemoval(artwork.UserID, artwork.SubSkillIDs)
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Status updated to " + req.Status})
}

// VoteArtwork godoc
// @Summary      Admin — cast a vote on a submission
// @Tags         admin
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path   string             true "Artwork ID"
// @Param        body body   models.VoteRequest  true "Vote payload"
// @Success      200 {object} response.APIResponse
// @Router       /admin/submissions/{id}/vote [post]
func VoteArtwork(c *gin.Context) {
	adminID, _ := c.Get("user_id")
	adminObjID, err := primitive.ObjectIDFromHex(adminID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid admin ID")
		return
	}

	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load artwork
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found")
		return
	}

	// Only allow voting on pending artworks
	if artwork.Status != "Pending" {
		response.Error(c, http.StatusBadRequest, "Artwork is already "+artwork.Status)
		return
	}

	// Check if this admin already voted
	for _, v := range artwork.Votes {
		if v.AdminID == adminObjID {
			response.Error(c, http.StatusConflict, "You have already voted on this artwork")
			return
		}
	}

	// Record the vote
	newVote := models.ArtworkVote{
		AdminID: adminObjID,
		Vote:    req.Vote,
		VotedAt: time.Now(),
	}

	approveCount := artwork.VoteApproveCount
	rejectCount  := artwork.VoteRejectCount
	if req.Vote == "approve" {
		approveCount++
	} else {
		rejectCount++
	}

	// Count total admins to determine majority
	totalAdmins, _ := db.Col("users").CountDocuments(ctx, bson.M{"role": "admin"})
	majority := int((totalAdmins + 1) / 2) // ceil(total/2)
	if majority < 1 {
		majority = 1
	}

	// Determine if majority reached → auto-update status
	newStatus := artwork.Status
	if approveCount >= majority {
		newStatus = "Approved"
	} else if rejectCount >= majority {
		newStatus = "Rejected"
	}

	statusChanged := newStatus != artwork.Status

	if statusChanged {
		// Decision reached — clear all vote data
		db.Col("artworks").UpdateOne(ctx, bson.M{"_id": artworkID}, bson.M{
			"$set": bson.M{
				"status":             newStatus,
				"vote_approve_count": 0,
				"vote_reject_count":  0,
				"votes":              []models.ArtworkVote{},
				"updated_at":         time.Now(),
			},
		})
		approveCount = 0
		rejectCount = 0
	} else {
		// Still pending — just record the vote
		setFields := bson.M{
			"vote_approve_count": approveCount,
			"vote_reject_count":  rejectCount,
			"updated_at":         time.Now(),
		}
		db.Col("artworks").UpdateOne(ctx, bson.M{"_id": artworkID}, bson.M{
			"$push": bson.M{"votes": newVote},
			"$set":  setFields,
		})
	}

	response.Success(c, http.StatusOK, gin.H{
		"vote":          req.Vote,
		"approve_count": approveCount,
		"reject_count":  rejectCount,
		"majority":      majority,
		"status":        newStatus,
	})
}

// AdminDeleteArtwork godoc
// @Summary      Admin — permanently delete an artwork and all its proofs
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Artwork ID"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /admin/submissions/{id} [delete]
func AdminDeleteArtwork(c *gin.Context) {
	artworkID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid artwork ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify artwork exists
	var artwork models.Artwork
	if err := db.Col("artworks").FindOne(ctx, bson.M{"_id": artworkID}).Decode(&artwork); err != nil {
		response.Error(c, http.StatusNotFound, "Artwork not found")
		return
	}

	// Delete upload record + Cloudinary file
	var upload models.Upload
	if err := db.Col("uploads").FindOne(ctx, bson.M{
		"user_id": artwork.UserID,
		"title":   artwork.Title,
	}).Decode(&upload); err == nil {
		// Delete from Cloudinary if cloud upload
		if upload.CloudinaryPublicID != "" {
			_ = cloud.DeleteFile(upload.CloudinaryPublicID)
		}
		db.Col("uploads").DeleteOne(ctx, bson.M{"_id": upload.ID})
	}

	// Delete all proofs linked to this artwork
	proofCursor, _ := db.Col("proofs").Find(ctx, bson.M{"artwork_id": artworkID})
	if proofCursor != nil {
		var proofs []models.Proof
		_ = proofCursor.All(ctx, &proofs)
		proofCursor.Close(ctx)
		for _, p := range proofs {
			// Delete proof file from Cloudinary
			if p.CloudinaryPublicID != "" {
				_ = cloud.DeleteFile(p.CloudinaryPublicID)
			}
		}
		db.Col("proofs").DeleteMany(ctx, bson.M{"artwork_id": artworkID})
	}

	// Delete the artwork itself
	db.Col("artworks").DeleteOne(ctx, bson.M{"_id": artworkID})

	// If it was approved, roll back the skill ranks so the level stays accurate
	if artwork.Status == "Approved" {
		DecrementRanksAfterRemoval(artwork.UserID, artwork.SubSkillIDs)
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Artwork deleted successfully"})
}

// ── User Management ───────────────────────────────────────────────────────────

// GetAllUsers godoc
// @Summary      List all users (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /admin/users [get]
func GetAllUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("users").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch users")
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode users")
		return
	}
	if users == nil {
		users = []models.User{}
	}
	response.Success(c, http.StatusOK, users)
}

// BanUser godoc
// @Summary      Ban a user (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/users/{id}/ban [put]
func BanUser(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("users").UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"banned": true}},
	)
	if err != nil || result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	// Revoke all refresh tokens so banned user is logged out immediately
	db.Col("refresh_tokens").DeleteMany(ctx, bson.M{"user_id": userID})

	response.Success(c, http.StatusOK, gin.H{"message": "User banned successfully"})
}

// UnbanUser godoc
// @Summary      Unban a user (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/users/{id}/unban [put]
func UnbanUser(c *gin.Context) {
	userID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("users").UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"banned": false}},
	)
	if err != nil || result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "User not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "User unbanned successfully"})
}

