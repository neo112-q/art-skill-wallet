package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username"      json:"username"`
	Bio       string             `bson:"bio"           json:"bio"`
	Role      string             `bson:"role"          json:"role"`      // "user" | "admin"
	HashPass  string             `bson:"hash_pass"     json:"-"`         // never exposed
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
}

// RegisterRequest is the request body for POST /auth/register.
type RegisterRequest struct {
	Username string `json:"username"  binding:"required,min=3,max=30"`
	Bio      string `json:"bio"`
	HashPass string `json:"hash_pass" binding:"required,min=6"`
}

// LoginRequest is the request body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful login.
// Contains both tokens — no sensitive user data.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest is the request body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
