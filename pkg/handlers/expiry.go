package handlers

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/models"
)

const artworkTTL = 48 * time.Hour

// StartExpiryWorker runs a background goroutine that checks every hour for
// pending artworks older than 48 hours and auto-rejects them.
func StartExpiryWorker() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			expireOldArtworks()
		}
	}()
}

func expireOldArtworks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-artworkTTL)

	cursor, err := db.Col("artworks").Find(ctx, bson.M{
		"status":     "Pending",
		"created_at": bson.M{"$lt": cutoff},
	})
	if err != nil {
		log.Printf("[expiry] query error: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var expired []models.Artwork
	if err := cursor.All(ctx, &expired); err != nil || len(expired) == 0 {
		return
	}

	ids := make([]primitive.ObjectID, len(expired))
	for i, a := range expired {
		ids[i] = a.ID
	}

	_, err = db.Col("artworks").UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"$set": bson.M{"status": "Rejected", "updated_at": time.Now()}},
	)
	if err != nil {
		log.Printf("[expiry] update error: %v", err)
		return
	}

	log.Printf("[expiry] auto-rejected %d artwork(s) after 48h timeout", len(expired))
}
