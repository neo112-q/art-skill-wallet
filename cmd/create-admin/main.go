package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Find and load .env from project root
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	godotenv.Load(filepath.Join(dir, ".env"))

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "artskiliwallet"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx)

	col := client.Database(dbName).Collection("users")

	username := "admin2"
	email := "admin2@artskill.com"
	password := "admin123"

	// Check for existing username or email
	count, err := col.CountDocuments(ctx, bson.M{"$or": bson.A{
		bson.M{"username": username},
		bson.M{"email": email},
	}})
	if err != nil {
		log.Fatalf("db check: %v", err)
	}
	if count > 0 {
		log.Fatalf("User with username '%s' or email '%s' already exists.", username, email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	_, err = col.InsertOne(ctx, bson.M{
		"_id":             primitive.NewObjectID(),
		"username":        username,
		"email":           email,
		"password_hash":   string(hash),
		"bio":             "",
		"profile_picture": "",
		"role":            "admin",
		"banned":          false,
		"created_at":      time.Now(),
	})
	if err != nil {
		log.Fatalf("insert: %v", err)
	}

	log.Printf("✓ Admin account created: %s (%s)", username, email)
}
