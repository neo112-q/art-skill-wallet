package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Proof struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"        json:"id"`
	ArtworkID          primitive.ObjectID `bson:"artwork_id"           json:"artwork_id"`
	FileURL            string             `bson:"file_url"             json:"file_url"`
	FileType           string             `bson:"file_type"            json:"file_type"`
	CloudinaryPublicID string             `bson:"cloudinary_public_id" json:"cloudinary_public_id,omitempty"`
}

type ProofRequest struct {
	ArtworkID string `json:"artwork_id" binding:"required"`
	FileURL   string `json:"file_url"   binding:"required"`
	FileType  string `json:"file_type"  binding:"required,oneof=image video"`
}
