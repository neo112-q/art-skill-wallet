package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Artwork struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty"  json:"id"`
	UserID        primitive.ObjectID   `bson:"user_id"        json:"user_id"`
	SkillID       primitive.ObjectID   `bson:"skill_id"       json:"skill_id"`        // primary skill (backward compat)
	SkillIDs      []primitive.ObjectID `bson:"skill_ids"      json:"skill_ids"`        // all skills
	Title         string               `bson:"title"          json:"title"`
	Description   string               `bson:"description"    json:"description"`
	PrivacyStatus string               `bson:"privacy_status" json:"privacy_status"`
	Status        string               `bson:"status"         json:"status"`
	UploadDate    time.Time            `bson:"upload_date"    json:"upload_date"`
	CreatedAt     time.Time            `bson:"created_at"     json:"created_at"`
	UpdatedAt     time.Time            `bson:"updated_at"     json:"updated_at"`
}

type ArtworkRequest struct {
	SkillID       string   `json:"skill_id"`                                          // backward compat (single)
	SkillIDs      []string `json:"skill_ids"`                                         // multi-skill
	Title         string   `json:"title"          binding:"required"`
	Description   string   `json:"description"`
	PrivacyStatus string   `json:"privacy_status" binding:"required,oneof=Public Private"`
}

type ArtworkUpdateRequest struct {
	Title         string `json:"title"          binding:"required"`
	Description   string `json:"description"`
	PrivacyStatus string `json:"privacy_status" binding:"required,oneof=Public Private"`
}
