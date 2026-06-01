package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Skill struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"        json:"id"`
	UserID           primitive.ObjectID `bson:"user_id"              json:"user_id"`
	SkillName        string             `bson:"skill_name"           json:"skill_name"`
	Level            string             `bson:"level"                json:"level"`
	DevelopmentGuide string             `bson:"development_guide"    json:"development_guide"`
	CreatedAt        time.Time          `bson:"created_at"           json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at"           json:"updated_at"`
}

type SkillRequest struct {
	SkillName        string `json:"skill_name"        binding:"required"`
	Level            string `json:"level"             binding:"required,oneof=Beginner Intermediate Advanced"`
	DevelopmentGuide string `json:"development_guide"`
}
