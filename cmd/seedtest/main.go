// cmd/seedtest — สร้างบัญชีทดสอบ + ผลงานครบทุกระดับ (Beginner / Intermediate / Advanced)
//
// ภาพผลงาน + proof เป็นภาพงานศิลปะจริง (public domain จาก Art Institute of Chicago)
// ที่ถูก "อัปโหลดเข้า Cloudinary" จริง — เก็บทั้ง secure_url และ public_id เหมือนผู้ใช้อัปโหลดเอง
//
// วิธีรัน (ที่ project root):
//   go run ./cmd/seedtest
//
// บัญชีทดสอบที่ได้:
//   email:    test@artskill.com
//   password: test123
//
// ต้องตั้งค่า CLOUDINARY_* ใน .env และรัน `go run ./cmd/seed` มาก่อน (ให้มี main/sub skills)
// รันซ้ำได้ — ล้างข้อมูล+ไฟล์ Cloudinary เดิมของ test user แล้วสร้างใหม่ (ไม่ซ้ำซ้อน)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"art-skill-wallet/pkg/cloud"
)

const seedFolder = "artskill_seed" // โฟลเดอร์ใน Cloudinary สำหรับไฟล์ที่ seed สร้าง

// แต่ละระดับ: ชื่อ sub-skill + จำนวนผลงานที่ approve (= approval_count) + คำค้นภาพศิลปะ
type tier struct {
	subSkillName string
	count        int
	wantRank     string
	query        string // คำค้นภาพงานศิลปะจริงใน Art Institute of Chicago API
}

var tiers = []tier{
	{"Watercolor", 5, "Beginner", "watercolor painting"},
	{"Digital Illustration", 14, "Intermediate", "illustration"},
	{"Pencil Sketch", 28, "Advanced", "pencil drawing"},
}

const (
	testUsername = "testartist"
	testEmail    = "test@artskill.com"
	testPassword = "test123"
)

type cloudImg struct {
	url string
	pid string
}

func calcRank(count int) string {
	switch {
	case count >= 25:
		return "Advanced"
	case count >= 10:
		return "Intermediate"
	case count >= 1:
		return "Beginner"
	default:
		return ""
	}
}

func main() {
	// โหลด .env จาก project root
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

	// ── Cloudinary client ───────────────────────────────────────────────────
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		log.Fatalf("cloudinary init: %v (ตรวจ CLOUDINARY_* ใน .env)", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	// avatar = ภาพ portrait จริง อัปเข้า Cloudinary
	avatar := cloudImg{url: "https://picsum.photos/seed/avatar/300/300"}
	if ports := fetchArtImages("portrait painting"); len(ports) > 0 {
		if u, pid, e := uploadURL(cld, ports[0]); e == nil {
			avatar = cloudImg{url: u, pid: pid}
		}
	}

	// ── 1) สร้าง/หา test user ──────────────────────────────────────────────
	var user struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err = db.Collection("users").FindOne(ctx, bson.M{"email": testEmail}).Decode(&user)
	if err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)
		user.ID = primitive.NewObjectID()
		_, err = db.Collection("users").InsertOne(ctx, bson.M{
			"_id":             user.ID,
			"username":        testUsername,
			"email":           testEmail,
			"password_hash":   string(hash),
			"bio":             "บัญชีทดสอบ — มีผลงานครบทุกระดับ Beginner / Intermediate / Advanced",
			"profile_picture": avatar.url,
			"role":            "user",
			"banned":          false,
			"created_at":      time.Now(),
		})
		if err != nil {
			log.Fatalf("insert user: %v", err)
		}
		log.Printf("✓ สร้าง test user: %s (%s)", testUsername, testEmail)
	} else {
		db.Collection("users").UpdateOne(ctx, bson.M{"_id": user.ID},
			bson.M{"$set": bson.M{"profile_picture": avatar.url}})
		log.Printf("• พบ test user เดิม: %s — ล้างข้อมูล+ไฟล์เก่าแล้วสร้างใหม่", testEmail)
	}

	// ── 2) ล้างข้อมูล + ไฟล์ Cloudinary เก่าของ user นี้ ───────────────────────
	cleanup(ctx, db, user.ID)

	// ── 3) วนแต่ละระดับ: อัปโหลดภาพจริงเข้า Cloudinary แล้วสร้าง artworks/proofs/uploads/ranks
	now := time.Now()
	grand := 0
	for _, t := range tiers {
		subID, mainName, ok := findSubSkill(ctx, db, t.subSkillName)
		if !ok {
			log.Fatalf("ไม่พบ sub-skill %q — กรุณารัน `go run ./cmd/seed` ก่อน", t.subSkillName)
		}

		// อัปโหลดภาพศิลปะจริงเข้า Cloudinary (สูงสุด count รูปต่อสกิล)
		log.Printf("  … กำลังอัปโหลดภาพ %q เข้า Cloudinary", t.subSkillName)
		pool := uploadPool(cld, fetchArtImages(t.query), t.count)
		if len(pool) == 0 {
			log.Fatalf("อัปโหลดภาพเข้า Cloudinary ไม่สำเร็จเลยสำหรับ %s", t.subSkillName)
		}
		at := func(k int) cloudImg { return pool[((k%len(pool))+len(pool))%len(pool)] }

		for i := 1; i <= t.count; i++ {
			artID := primitive.NewObjectID()
			title := fmt.Sprintf("%s #%d", t.subSkillName, i)
			uploadDate := now.AddDate(0, 0, -grand)
			mainImg := at(i - 1)
			proofA := at(2 * (i - 1))
			proofB := at(2*(i-1) + 1)

			// artwork (Public + Approved)
			if _, err := db.Collection("artworks").InsertOne(ctx, bson.M{
				"_id":                artID,
				"user_id":            user.ID,
				"sub_skill_ids":      []primitive.ObjectID{subID},
				"title":              title,
				"description":        fmt.Sprintf("ผลงาน %s ฝีมือระดับ %s", mainName, t.wantRank),
				"privacy_status":     "Public",
				"status":             "Approved",
				"votes":              bson.A{},
				"vote_approve_count": 2,
				"vote_reject_count":  0,
				"upload_date":        uploadDate,
				"created_at":         uploadDate,
				"updated_at":         uploadDate,
			}); err != nil {
				log.Fatalf("insert artwork: %v", err)
			}

			// proofs (before/after) — เก็บ URL + public_id ของ Cloudinary
			db.Collection("proofs").InsertMany(ctx, []interface{}{
				bson.M{"_id": primitive.NewObjectID(), "artwork_id": artID,
					"file_url": proofA.url, "file_type": "image", "cloudinary_public_id": proofA.pid},
				bson.M{"_id": primitive.NewObjectID(), "artwork_id": artID,
					"file_url": proofB.url, "file_type": "image", "cloudinary_public_id": proofB.pid},
			})

			// upload record
			db.Collection("uploads").InsertOne(ctx, bson.M{
				"_id":                  primitive.NewObjectID(),
				"user_id":              user.ID,
				"title":                title,
				"description":          fmt.Sprintf("ผลงาน %s", mainName),
				"file_name":            fmt.Sprintf("%s-%d.jpg", strings.ToLower(strings.ReplaceAll(t.subSkillName, " ", "-")), i),
				"file_path":            "",
				"file_url":             mainImg.url,
				"cloudinary_public_id": mainImg.pid,
				"file_size":            int64(123456),
				"mime_type":            "image/jpeg",
				"created_at":           uploadDate,
				"updated_at":           uploadDate,
			})

			grand++
		}

		// user_skill_ranks — approval_count = จำนวน artwork ที่ approve
		db.Collection("user_skill_ranks").InsertOne(ctx, bson.M{
			"_id":            primitive.NewObjectID(),
			"user_id":        user.ID,
			"sub_skill_id":   subID,
			"approval_count": t.count,
			"rank":           calcRank(t.count),
			"updated_at":     now,
		})

		log.Printf("  ✓ %-22s | %2d ผลงาน | %2d ภาพใน Cloudinary | rank = %s",
			t.subSkillName, t.count, len(pool), calcRank(t.count))
	}

	log.Printf("เสร็จสิ้น! สร้างผลงานรวม %d ชิ้น ครบ 3 ระดับ (ภาพเก็บใน Cloudinary)", grand)
	log.Printf("──────────────────────────────────────────")
	log.Printf(" เข้าสู่ระบบด้วย:  email = %s   password = %s", testEmail, testPassword)
	log.Printf("──────────────────────────────────────────")
}

// fetchArtImages ดึง URL ภาพงานศิลปะจริงจาก Art Institute of Chicago (public domain)
func fetchArtImages(query string) []string {
	endpoint := fmt.Sprintf(
		"https://api.artic.edu/api/v1/artworks/search?q=%s&fields=id,title,image_id&limit=100",
		url.QueryEscape(query),
	)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("AIC-User-Agent", "ArtSkillWallet seed (test@artskill.com)")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var out struct {
		Data []struct {
			ImageID string `json:"image_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}

	urls := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		if d.ImageID != "" {
			urls = append(urls, fmt.Sprintf(
				"https://www.artic.edu/iiif/2/%s/full/843,/0/default.jpg", d.ImageID))
		}
	}
	return urls
}

// uploadURL อัปโหลดภาพจาก remote URL เข้า Cloudinary แล้วคืน secure_url + public_id
func uploadURL(cld *cloudinary.Cloudinary, src string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	res, err := cld.Upload.Upload(ctx, src, uploader.UploadParams{
		Folder:       seedFolder,
		ResourceType: "auto",
	})
	if err != nil {
		return "", "", err
	}
	return res.SecureURL, res.PublicID, nil
}

// uploadPool อัปโหลดภาพจาก list URL เข้า Cloudinary สูงสุด max รูป
func uploadPool(cld *cloudinary.Cloudinary, srcs []string, max int) []cloudImg {
	pool := make([]cloudImg, 0, max)
	for _, s := range srcs {
		if len(pool) >= max {
			break
		}
		u, pid, err := uploadURL(cld, s)
		if err != nil {
			continue
		}
		pool = append(pool, cloudImg{url: u, pid: pid})
	}
	return pool
}

// findSubSkill หา sub-skill จาก display_name (case-insensitive)
func findSubSkill(ctx context.Context, db *mongo.Database, displayName string) (primitive.ObjectID, string, bool) {
	var sub struct {
		ID          primitive.ObjectID `bson:"_id"`
		MainSkillID primitive.ObjectID `bson:"main_skill_id"`
	}
	if err := db.Collection("sub_skills").FindOne(ctx, bson.M{
		"display_name": bson.M{"$regex": "^" + displayName + "$", "$options": "i"},
	}).Decode(&sub); err != nil {
		return primitive.NilObjectID, "", false
	}
	var main struct {
		Name string `bson:"name"`
	}
	db.Collection("main_skills").FindOne(ctx, bson.M{"_id": sub.MainSkillID}).Decode(&main)
	return sub.ID, main.Name, true
}

// cleanup ลบ artworks/proofs/uploads/ranks + ไฟล์ Cloudinary เก่าของ user (กันซ้ำตอนรันซ้ำ)
func cleanup(ctx context.Context, db *mongo.Database, userID primitive.ObjectID) {
	// artwork ids ของ user
	cur, _ := db.Collection("artworks").Find(ctx, bson.M{"user_id": userID})
	var arts []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if cur != nil {
		cur.All(ctx, &arts)
	}
	ids := make([]primitive.ObjectID, 0, len(arts))
	for _, a := range arts {
		ids = append(ids, a.ID)
	}

	// รวบรวม public_id ทั้งหมดที่ต้องลบจาก Cloudinary (proofs + uploads)
	pids := map[string]struct{}{}
	if len(ids) > 0 {
		if pc, _ := db.Collection("proofs").Find(ctx, bson.M{"artwork_id": bson.M{"$in": ids}}); pc != nil {
			var ps []struct {
				PID string `bson:"cloudinary_public_id"`
			}
			pc.All(ctx, &ps)
			for _, p := range ps {
				if p.PID != "" {
					pids[p.PID] = struct{}{}
				}
			}
		}
	}
	if uc, _ := db.Collection("uploads").Find(ctx, bson.M{"user_id": userID}); uc != nil {
		var us []struct {
			PID string `bson:"cloudinary_public_id"`
		}
		uc.All(ctx, &us)
		for _, u := range us {
			if u.PID != "" {
				pids[u.PID] = struct{}{}
			}
		}
	}
	for pid := range pids {
		_ = cloud.DeleteFile(pid)
	}
	if len(pids) > 0 {
		log.Printf("  ลบไฟล์เก่าใน Cloudinary %d ไฟล์", len(pids))
	}

	// ลบ documents
	if len(ids) > 0 {
		db.Collection("proofs").DeleteMany(ctx, bson.M{"artwork_id": bson.M{"$in": ids}})
	}
	db.Collection("artworks").DeleteMany(ctx, bson.M{"user_id": userID})
	db.Collection("uploads").DeleteMany(ctx, bson.M{"user_id": userID})
	db.Collection("user_skill_ranks").DeleteMany(ctx, bson.M{"user_id": userID})
}
