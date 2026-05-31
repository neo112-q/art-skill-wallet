package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var Database *mongo.Database

func Connect() error {
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
		return err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}
	Client = client
	Database = client.Database(dbName)
	log.Printf("MongoDB connected: %s / %s", uri, dbName)
	return nil
}

func Col(name string) *mongo.Collection {
	return Database.Collection(name)
}

func CreateIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Unique index on username
	Database.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	Database.Collection("skills").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})

	Database.Collection("artworks").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})

	// Unique index on token_hash for fast revocation lookups
	Database.Collection("refresh_tokens").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token_hash", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// TTL index — MongoDB auto-deletes expired refresh tokens
	Database.Collection("refresh_tokens").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	})

	log.Println("MongoDB indexes ensured")
}
