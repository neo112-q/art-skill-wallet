package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Skill struct {
	ID           primitive.ObjectID     `bson:"_id,omitempty"        json:"id"`
	UserID       primitive.ObjectID     `bson:"user_id"              json:"user_id"`
	Name         string                 `bson:"name"                 json:"name"`
	Level        string                 `bson:"level"                json:"level"`
	ProgressData map[string]interface{} `bson:"progress_data,omitempty" json:"progress_data,omitempty"`
	CreatedAt    time.Time              `bson:"created_at"           json:"created_at"`
	UpdatedAt    time.Time              `bson:"updated_at"           json:"updated_at"`
}

// SkillRequest godoc
type SkillRequest struct {
	Name         string                 `json:"name"          binding:"required"`
	Level        string                 `json:"level"         binding:"required,oneof=Beginner Intermediate Advanced"`
	ProgressData map[string]interface{} `json:"progress_data"`
}
