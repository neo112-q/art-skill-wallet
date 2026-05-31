package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProgressCheckpoint struct {
	Date    time.Time `bson:"date"     json:"date"`
	LogText string    `bson:"log_text" json:"log_text"`
	Tags    []string  `bson:"tags"     json:"tags"`
}

type Artwork struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty"             json:"id"`
	UserID              primitive.ObjectID   `bson:"user_id"                   json:"user_id"`
	SkillID             primitive.ObjectID   `bson:"skill_id"                  json:"skill_id"`
	Path                string               `bson:"path"                      json:"path"`
	Date                time.Time            `bson:"date"                      json:"date"`
	Privacy             string               `bson:"privacy"                   json:"privacy"`
	ProgressCheckpoints []ProgressCheckpoint `bson:"progress_checkpoints"      json:"progress_checkpoints"`
	CreatedAt           time.Time            `bson:"created_at"                json:"created_at"`
	UpdatedAt           time.Time            `bson:"updated_at"                json:"updated_at"`
}

// ArtworkRequest godoc
type ArtworkRequest struct {
	SkillID             string               `json:"skill_id"             binding:"required"`
	Path                string               `json:"path"`
	Date                time.Time            `json:"date"`
	Privacy             string               `json:"privacy"              binding:"required,oneof=Public Private"`
	ProgressCheckpoints []ProgressCheckpoint `json:"progress_checkpoints"`
}
