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

// DefaultGuide returns a built-in development guide when the user leaves it empty.
func DefaultGuide(level string) string {
	switch level {
	case "Beginner":
		return "Start with the fundamentals: study basic shapes, proportions, and composition. " +
			"Practice daily sketches (15-30 min). Follow beginner tutorials and copy master works to build muscle memory."
	case "Intermediate":
		return "Deepen your understanding: study anatomy, color theory, and lighting. " +
			"Take on personal projects with deadlines. Seek feedback from peers and iterate on your weaknesses."
	case "Advanced":
		return "Refine your style and push boundaries: develop a signature aesthetic. " +
			"Build a professional portfolio. Mentor others, contribute to community critiques, and explore cross-disciplinary techniques."
	default:
		return ""
	}
}
