package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"      json:"id"`
	Username       string             `bson:"username"           json:"username"`
	Email          string             `bson:"email"              json:"email"`
	PasswordHash   string             `bson:"password_hash"      json:"-"`
	Bio            string             `bson:"bio"                json:"bio"`
	ProfilePicture string             `bson:"profile_picture"    json:"profile_picture"`
	Role           string             `bson:"role"               json:"role"`
	CreatedAt      time.Time          `bson:"created_at"         json:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
