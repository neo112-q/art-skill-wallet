package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Upload struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"          json:"id"`
	UserID            primitive.ObjectID `bson:"user_id"                json:"user_id"`
	Title             string             `bson:"title"                  json:"title"`
	Description       string             `bson:"description"            json:"description"`
	FileName          string             `bson:"file_name"              json:"file_name"`
	FilePath          string             `bson:"file_path"              json:"file_path"`           // legacy local path
	FileURL           string             `bson:"file_url"               json:"file_url"`            // Cloudinary URL
	CloudinaryPublicID string            `bson:"cloudinary_public_id"   json:"cloudinary_public_id,omitempty"`
	FileSize          int64              `bson:"file_size"              json:"file_size"`
	MimeType          string             `bson:"mime_type"              json:"mime_type"`
	CreatedAt         time.Time          `bson:"created_at"             json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at"             json:"updated_at"`
}

type UploadUpdateRequest struct {
	Title       string `json:"title"       binding:"required"`
	Description string `json:"description"`
}
