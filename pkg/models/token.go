package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RefreshToken stores a hashed refresh token in MongoDB.
// The raw token is NEVER persisted — only its SHA-256 hash.
type RefreshToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	UserID    primitive.ObjectID `bson:"user_id"       json:"-"`
	TokenHash string             `bson:"token_hash"    json:"-"`
	ExpiresAt time.Time          `bson:"expires_at"    json:"-"`
	CreatedAt time.Time          `bson:"created_at"    json:"-"`
}
