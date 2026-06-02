package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ── Main Skill (admin creates) ────────────────────────────────────────────────

type MainSkill struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name"          json:"name"`
	CreatedBy primitive.ObjectID `bson:"created_by"    json:"created_by"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
}

type MainSkillRequest struct {
	Name string `json:"name" binding:"required,min=2,max=50"`
}

// ── Sub Skill (any user creates or picks) ─────────────────────────────────────

type SubSkill struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"  json:"id"`
	MainSkillID primitive.ObjectID `bson:"main_skill_id"  json:"main_skill_id"`
	Name        string             `bson:"name"           json:"name"`         // lowercase normalized
	DisplayName string             `bson:"display_name"   json:"display_name"` // original casing
	CreatedBy   primitive.ObjectID `bson:"created_by"     json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at"     json:"created_at"`
}

type SubSkillRequest struct {
	DisplayName string `json:"display_name" binding:"required,min=2,max=50"`
}

// ── User Skill Rank (auto-updated on artwork approval) ────────────────────────

type UserSkillRank struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"   json:"id"`
	UserID        primitive.ObjectID `bson:"user_id"         json:"user_id"`
	SubSkillID    primitive.ObjectID `bson:"sub_skill_id"    json:"sub_skill_id"`
	ApprovalCount int                `bson:"approval_count"  json:"approval_count"`
	Rank          string             `bson:"rank"            json:"rank"` // "" | "Beginner" | "Intermediate" | "Advanced"
	UpdatedAt     time.Time          `bson:"updated_at"      json:"updated_at"`
}

// CalcRank returns the rank string for a given approval count.
func CalcRank(count int) string {
	switch {
	case count >= 25:
		return "Advanced"
	case count >= 10:
		return "Intermediate"
	case count >= 1:
		return "Beginner"
	default:
		return ""
	}
}

// RankProgress returns how many approvals until the next rank.
// Returns current count, target count, and next rank name.
func RankProgress(count int) (current, target int, nextRank string) {
	switch {
	case count >= 25:
		return count, 25, "Advanced" // already max
	case count >= 10:
		return count, 25, "Advanced"
	case count >= 1:
		return count, 10, "Intermediate"
	default:
		return count, 1, "Beginner"
	}
}
