package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
	"art-skill-wallet/pkg/response"
)

// ── Main Skills ───────────────────────────────────────────────────────────────

// GetMainSkills godoc
// @Summary      List all main skill categories
// @Tags         skills
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /main-skills [get]
func GetMainSkills(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("main_skills").Find(ctx, bson.M{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch main skills")
		return
	}
	defer cursor.Close(ctx)

	var skills []models.MainSkill
	if err := cursor.All(ctx, &skills); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode main skills")
		return
	}
	if skills == nil {
		skills = []models.MainSkill{}
	}
	response.Success(c, http.StatusOK, skills)
}

// CreateMainSkill godoc
// @Summary      Create a new main skill category (admin only)
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Success      201 {object} response.APIResponse
// @Router       /admin/main-skills [post]
func CreateMainSkill(c *gin.Context) {
	adminID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(adminID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req models.MainSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check for duplicate name (case-insensitive)
	count, _ := db.Col("main_skills").CountDocuments(ctx, bson.M{
		"name": bson.M{"$regex": "^" + req.Name + "$", "$options": "i"},
	})
	if count > 0 {
		response.Error(c, http.StatusConflict, "Main skill already exists")
		return
	}

	skill := models.MainSkill{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		CreatedBy: objID,
		CreatedAt: time.Now(),
	}

	if _, err := db.Col("main_skills").InsertOne(ctx, skill); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create main skill")
		return
	}
	response.Success(c, http.StatusCreated, skill)
}

// UpdateMainSkill godoc
// @Summary      Rename a main skill category (admin only)
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Main Skill ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/main-skills/{id} [put]
func UpdateMainSkill(c *gin.Context) {
	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	name := strings.TrimSpace(req.Name)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check duplicate (excluding self)
	count, _ := db.Col("main_skills").CountDocuments(ctx, bson.M{
		"_id":  bson.M{"$ne": skillID},
		"name": bson.M{"$regex": "^" + name + "$", "$options": "i"},
	})
	if count > 0 {
		response.Error(c, http.StatusConflict, "A main skill with that name already exists")
		return
	}

	result, err := db.Col("main_skills").UpdateOne(ctx,
		bson.M{"_id": skillID},
		bson.M{"$set": bson.M{"name": name}},
	)
	if err != nil || result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Main skill not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Main skill updated", "name": name})
}

// DeleteMainSkill godoc
// @Summary      Delete a main skill and all its sub-skills (admin only)
// @Tags         skills
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Main Skill ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/main-skills/{id} [delete]
func DeleteMainSkill(c *gin.Context) {
	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db.Col("main_skills").DeleteOne(ctx, bson.M{"_id": skillID})
	db.Col("sub_skills").DeleteMany(ctx, bson.M{"main_skill_id": skillID})

	response.Success(c, http.StatusOK, gin.H{"message": "Main skill deleted"})
}

// ── Sub Skills ────────────────────────────────────────────────────────────────

// GetSubSkills godoc
// @Summary      List sub-skills under a main skill
// @Tags         skills
// @Produce      json
// @Param        id path string true "Main Skill ID"
// @Success      200 {object} response.APIResponse
// @Router       /main-skills/{id}/sub-skills [get]
func GetSubSkills(c *gin.Context) {
	mainID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid main skill ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("sub_skills").Find(ctx, bson.M{"main_skill_id": mainID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch sub skills")
		return
	}
	defer cursor.Close(ctx)

	var skills []models.SubSkill
	if err := cursor.All(ctx, &skills); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to decode sub skills")
		return
	}
	if skills == nil {
		skills = []models.SubSkill{}
	}
	response.Success(c, http.StatusOK, skills)
}

// CreateSubSkill godoc
// @Summary      Create a sub-skill under a main skill
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Main Skill ID"
// @Success      201 {object} response.APIResponse
// @Router       /main-skills/{id}/sub-skills [post]
func CreateSubSkill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	mainID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid main skill ID")
		return
	}

	var req models.SubSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// Normalize: lowercase for storage, trim spaces
	normalized := strings.ToLower(strings.TrimSpace(req.DisplayName))
	displayName := strings.TrimSpace(req.DisplayName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify main skill exists
	count, _ := db.Col("main_skills").CountDocuments(ctx, bson.M{"_id": mainID})
	if count == 0 {
		response.Error(c, http.StatusNotFound, "Main skill not found")
		return
	}

	// Return existing sub skill if name already taken (case-insensitive)
	var existing models.SubSkill
	if err := db.Col("sub_skills").FindOne(ctx, bson.M{
		"main_skill_id": mainID,
		"name":          normalized,
	}).Decode(&existing); err == nil {
		response.Success(c, http.StatusOK, existing) // already exists — return it
		return
	}

	skill := models.SubSkill{
		ID:          primitive.NewObjectID(),
		MainSkillID: mainID,
		Name:        normalized,
		DisplayName: displayName,
		CreatedBy:   objID,
		CreatedAt:   time.Now(),
	}

	if _, err := db.Col("sub_skills").InsertOne(ctx, skill); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create sub skill")
		return
	}
	response.Success(c, http.StatusCreated, skill)
}

// UpdateSubSkill godoc
// @Summary      Rename a sub-skill (admin only)
// @Tags         skills
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Sub-Skill ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/sub-skills/{id} [put]
func UpdateSubSkill(c *gin.Context) {
	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	var req struct {
		DisplayName string `json:"display_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	normalized := strings.ToLower(strings.TrimSpace(req.DisplayName))
	displayName := strings.TrimSpace(req.DisplayName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("sub_skills").UpdateOne(ctx,
		bson.M{"_id": skillID},
		bson.M{"$set": bson.M{"name": normalized, "display_name": displayName}},
	)
	if err != nil || result.MatchedCount == 0 {
		response.Error(c, http.StatusNotFound, "Sub skill not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Sub skill updated", "display_name": displayName})
}

// DeleteSubSkill godoc
// @Summary      Delete a sub-skill (admin only)
// @Tags         skills
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Sub-Skill ID"
// @Success      200 {object} response.APIResponse
// @Router       /admin/sub-skills/{id} [delete]
func DeleteSubSkill(c *gin.Context) {
	skillID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid skill ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Col("sub_skills").DeleteOne(ctx, bson.M{"_id": skillID})
	if err != nil || result.DeletedCount == 0 {
		response.Error(c, http.StatusNotFound, "Sub skill not found")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Sub skill deleted"})
}

// SearchSubSkills godoc
// @Summary      Search sub-skills by name across all categories
// @Tags         skills
// @Produce      json
// @Param        q query string true "Search query"
// @Success      200 {object} response.APIResponse
// @Router       /sub-skills/search [get]
func SearchSubSkills(c *gin.Context) {
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if q == "" {
		response.Success(c, http.StatusOK, []models.SubSkill{})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("sub_skills").Find(ctx, bson.M{
		"name": bson.M{"$regex": q, "$options": "i"},
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Search failed")
		return
	}
	defer cursor.Close(ctx)

	var skills []models.SubSkill
	_ = cursor.All(ctx, &skills)
	if skills == nil {
		skills = []models.SubSkill{}
	}
	response.Success(c, http.StatusOK, skills)
}

// ── User Skill Ranks ──────────────────────────────────────────────────────────

// GetMyRanks godoc
// @Summary      Get the authenticated user's skill ranks
// @Tags         skills
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.APIResponse
// @Router       /my-ranks [get]
func GetMyRanks(c *gin.Context) {
	userID, _ := c.Get("user_id")
	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch ranks
	cursor, err := db.Col("user_skill_ranks").Find(ctx, bson.M{"user_id": objID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch ranks")
		return
	}
	defer cursor.Close(ctx)

	var ranks []models.UserSkillRank
	_ = cursor.All(ctx, &ranks)
	if ranks == nil {
		ranks = []models.UserSkillRank{}
	}

	// Enrich with sub skill + main skill names
	type RankRow struct {
		models.UserSkillRank
		SubSkillName  string `json:"sub_skill_name"`
		DisplayName   string `json:"display_name"`
		MainSkillID   string `json:"main_skill_id"`
		MainSkillName string `json:"main_skill_name"`
		NextRank      string `json:"next_rank"`
		Target        int    `json:"target"`
	}

	rows := make([]RankRow, 0, len(ranks))
	for _, r := range ranks {
		var sub models.SubSkill
		if err := db.Col("sub_skills").FindOne(ctx, bson.M{"_id": r.SubSkillID}).Decode(&sub); err != nil {
			continue
		}
		var main models.MainSkill
		_ = db.Col("main_skills").FindOne(ctx, bson.M{"_id": sub.MainSkillID}).Decode(&main)

		_, target, nextRank := models.RankProgress(r.ApprovalCount)
		rows = append(rows, RankRow{
			UserSkillRank: r,
			SubSkillName:  sub.Name,
			DisplayName:   sub.DisplayName,
			MainSkillID:   sub.MainSkillID.Hex(),
			MainSkillName: main.Name,
			NextRank:      nextRank,
			Target:        target,
		})
	}

	response.Success(c, http.StatusOK, rows)
}

// GetUserRanks godoc
// @Summary      Get a public user's skill ranks
// @Tags         skills
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse
// @Router       /users/{id}/ranks [get]
func GetUserRanks(c *gin.Context) {
	userObjID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("user_skill_ranks").Find(ctx, bson.M{"user_id": userObjID})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch ranks")
		return
	}
	defer cursor.Close(ctx)

	var ranks []models.UserSkillRank
	_ = cursor.All(ctx, &ranks)
	if ranks == nil {
		ranks = []models.UserSkillRank{}
	}

	type RankRow struct {
		SubSkillID    string `json:"sub_skill_id"`
		DisplayName   string `json:"display_name"`
		MainSkillName string `json:"main_skill_name"`
		Rank          string `json:"rank"`
		ApprovalCount int    `json:"approval_count"`
	}

	rows := make([]RankRow, 0, len(ranks))
	for _, r := range ranks {
		var sub models.SubSkill
		if err := db.Col("sub_skills").FindOne(ctx, bson.M{"_id": r.SubSkillID}).Decode(&sub); err != nil {
			continue
		}
		var main models.MainSkill
		_ = db.Col("main_skills").FindOne(ctx, bson.M{"_id": sub.MainSkillID}).Decode(&main)

		rows = append(rows, RankRow{
			SubSkillID:    r.SubSkillID.Hex(),
			DisplayName:   sub.DisplayName,
			MainSkillName: main.Name,
			Rank:          r.Rank,
			ApprovalCount: r.ApprovalCount,
		})
	}

	response.Success(c, http.StatusOK, rows)
}

// ── Internal: update ranks after artwork approval ─────────────────────────────

// UpdateRanksAfterApproval increments approval_count for each sub skill used
// and recalculates rank. Called when artwork is auto-approved.
func UpdateRanksAfterApproval(userID primitive.ObjectID, subSkillIDs []primitive.ObjectID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, skillID := range subSkillIDs {
		// Upsert the rank document
		filter := bson.M{"user_id": userID, "sub_skill_id": skillID}

		var existing models.UserSkillRank
		err := db.Col("user_skill_ranks").FindOne(ctx, filter).Decode(&existing)

		newCount := existing.ApprovalCount + 1
		newRank := models.CalcRank(newCount)

		if err != nil {
			// Create new rank entry
			db.Col("user_skill_ranks").InsertOne(ctx, models.UserSkillRank{
				ID:            primitive.NewObjectID(),
				UserID:        userID,
				SubSkillID:    skillID,
				ApprovalCount: newCount,
				Rank:          newRank,
				UpdatedAt:     time.Now(),
			})
		} else {
			// Increment existing
			db.Col("user_skill_ranks").UpdateOne(ctx, filter, bson.M{
				"$inc": bson.M{"approval_count": 1},
				"$set": bson.M{"rank": newRank, "updated_at": time.Now()},
			})
		}
	}
}
