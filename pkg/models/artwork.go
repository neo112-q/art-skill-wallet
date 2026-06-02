package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ArtworkVote is one admin's vote on an artwork.
type ArtworkVote struct {
	AdminID primitive.ObjectID `bson:"admin_id" json:"admin_id"`
	Vote    string             `bson:"vote"     json:"vote"`    // "approve" | "reject"
	VotedAt time.Time          `bson:"voted_at" json:"voted_at"`
}

type Artwork struct {
	ID               primitive.ObjectID   `bson:"_id,omitempty"         json:"id"`
	UserID           primitive.ObjectID   `bson:"user_id"               json:"user_id"`
	SubSkillIDs      []primitive.ObjectID `bson:"sub_skill_ids"         json:"sub_skill_ids"`
	Title            string               `bson:"title"                 json:"title"`
	Description      string               `bson:"description"           json:"description"`
	PrivacyStatus    string               `bson:"privacy_status"        json:"privacy_status"`
	Status           string               `bson:"status"                json:"status"` // Pending | Approved | Rejected
	Votes            []ArtworkVote        `bson:"votes"                 json:"votes"`
	VoteApproveCount int                  `bson:"vote_approve_count"    json:"vote_approve_count"`
	VoteRejectCount  int                  `bson:"vote_reject_count"     json:"vote_reject_count"`
	UploadDate       time.Time            `bson:"upload_date"           json:"upload_date"`
	CreatedAt        time.Time            `bson:"created_at"            json:"created_at"`
	UpdatedAt        time.Time            `bson:"updated_at"            json:"updated_at"`
}

type ArtworkRequest struct {
	SubSkillIDs   []string `json:"sub_skill_ids"  binding:"required"`
	Title         string   `json:"title"          binding:"required"`
	Description   string   `json:"description"`
	PrivacyStatus string   `json:"privacy_status" binding:"required,oneof=Public Private"`
}

type ArtworkUpdateRequest struct {
	Title         string `json:"title"          binding:"required"`
	Description   string `json:"description"`
	PrivacyStatus string `json:"privacy_status" binding:"required,oneof=Public Private"`
}

type VoteRequest struct {
	Vote string `json:"vote" binding:"required,oneof=approve reject"`
}
