package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MainSkill struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `bson:"name"`
	CreatedBy primitive.ObjectID `bson:"created_by"`
	CreatedAt time.Time          `bson:"created_at"`
}

type SubSkill struct {
	ID          primitive.ObjectID `bson:"_id"`
	MainSkillID primitive.ObjectID `bson:"main_skill_id"`
	Name        string             `bson:"name"`
	DisplayName string             `bson:"display_name"`
	CreatedBy   primitive.ObjectID `bson:"created_by"`
	CreatedAt   time.Time          `bson:"created_at"`
}

var seed = []struct {
	main string
	subs []string
}{
	{"Painting", []string{"Watercolor", "Oil Painting", "Acrylic Painting"}},
	{"Drawing", []string{"Pencil Sketch", "Charcoal Drawing", "Ink Drawing"}},
	{"Digital Art", []string{"Digital Illustration", "Pixel Art", "Digital Painting"}},
	{"Photography", []string{"Portrait Photography", "Landscape Photography", "Street Photography"}},
	{"Sculpture", []string{"Clay Sculpting", "Stone Carving", "3D Sculpting"}},
	{"Printmaking", []string{"Screen Printing", "Linocut", "Etching"}},
	{"Animation", []string{"2D Animation", "3D Animation", "Stop Motion"}},
	{"Graphic Design", []string{"Logo Design", "Typography", "UI Design"}},
	{"Textile Art", []string{"Embroidery", "Weaving", "Batik"}},
	{"Mixed Media", []string{"Collage", "Assemblage", "Street Art"}},
}

func main() {
	// Load .env
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)
	mainCol := db.Collection("main_skills")
	subCol := db.Collection("sub_skills")
	systemID := primitive.NewObjectID() // placeholder admin ID

	inserted := 0
	skipped := 0

	for _, entry := range seed {
		// Skip if main skill already exists
		count, _ := mainCol.CountDocuments(ctx, bson.M{
			"name": bson.M{"$regex": "^" + entry.main + "$", "$options": "i"},
		})
		if count > 0 {
			log.Printf("  SKIP main skill (exists): %s", entry.main)
			skipped++

			// Still try to insert missing sub-skills
			var existing MainSkill
			mainCol.FindOne(ctx, bson.M{
				"name": bson.M{"$regex": "^" + entry.main + "$", "$options": "i"},
			}).Decode(&existing)
			insertSubs(ctx, subCol, existing.ID, entry.subs, systemID, &inserted, &skipped)
			continue
		}

		mainID := primitive.NewObjectID()
		_, err := mainCol.InsertOne(ctx, MainSkill{
			ID:        mainID,
			Name:      entry.main,
			CreatedBy: systemID,
			CreatedAt: time.Now(),
		})
		if err != nil {
			log.Printf("  ERROR inserting main skill %s: %v", entry.main, err)
			continue
		}
		log.Printf("  + Main skill: %s", entry.main)
		inserted++

		insertSubs(ctx, subCol, mainID, entry.subs, systemID, &inserted, &skipped)
	}

	log.Printf("Done. Inserted: %d, Skipped (already exist): %d", inserted, skipped)
}

func insertSubs(ctx context.Context, col *mongo.Collection, mainID primitive.ObjectID, subs []string, createdBy primitive.ObjectID, inserted, skipped *int) {
	for _, sub := range subs {
		norm := strings.ToLower(strings.TrimSpace(sub))
		count, _ := col.CountDocuments(ctx, bson.M{"main_skill_id": mainID, "name": norm})
		if count > 0 {
			log.Printf("    SKIP sub-skill (exists): %s", sub)
			*skipped++
			continue
		}
		_, err := col.InsertOne(ctx, SubSkill{
			ID:          primitive.NewObjectID(),
			MainSkillID: mainID,
			Name:        norm,
			DisplayName: sub,
			CreatedBy:   createdBy,
			CreatedAt:   time.Now(),
		})
		if err != nil {
			log.Printf("    ERROR inserting sub-skill %s: %v", sub, err)
			continue
		}
		log.Printf("    + Sub-skill: %s", sub)
		*inserted++
	}
}
