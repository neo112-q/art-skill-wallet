package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username"      json:"username"`
	Bio       string             `bson:"bio"           json:"bio"`
	HashPass  string             `bson:"hash_pass"     json:"-"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
}

// RegisterRequest godoc
type RegisterRequest struct {
	Username string `json:"username"  binding:"required,min=3,max=30"`
	Bio      string `json:"bio"`
	HashPass string `json:"hash_pass" binding:"required,min=6"`
}

// LoginRequest godoc
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse godoc
type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}
