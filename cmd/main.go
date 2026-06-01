package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/handlers"
	"art-skill-wallet/pkg/middleware"
)


func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot determine working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			log.Fatalf("could not locate project root (go.mod not found)")
		}
		dir = parent
	}
}

func main() {
	root := projectRoot()

	// Load .env from the project root (not the working directory)
	if err := godotenv.Load(filepath.Join(root, ".env")); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// ── Database ────────────────────────────────────────────────────────
	if err := db.Connect(); err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	db.CreateIndexes()

	// ── Router ──────────────────────────────────────────────────────────
	r := gin.Default()

	// Static assets
	r.Static("/css", filepath.Join(root, "css"))
	r.Static("/js", filepath.Join(root, "js"))
	r.Static("/img", filepath.Join(root, "img"))
	r.Static("/uploads", filepath.Join(root, "uploads"))

	// ── Public HTML pages (no token required) ───────────────────────────
	r.StaticFile("/", filepath.Join(root, "home.html"))
	r.StaticFile("/home.html", filepath.Join(root, "home.html"))
	r.StaticFile("/explore.html", filepath.Join(root, "explore.html"))

	// ── Protected HTML pages (JS guard handles redirect) ────────────────
	// HTML is served freely; the real lock is on the API.
	// Each page includes auth.js which redirects to home if no valid token.
	r.StaticFile("/profile.html", filepath.Join(root, "profile.html"))
	r.StaticFile("/history.html", filepath.Join(root, "history.html"))
	r.StaticFile("/upload.html", filepath.Join(root, "upload.html"))
	r.StaticFile("/skillManage.html", filepath.Join(root, "skillManage.html"))
	r.StaticFile("/editProfile.html", filepath.Join(root, "editProfile.html"))
	r.StaticFile("/admin.html", filepath.Join(root, "admin.html"))

	// ── API v1 ──────────────────────────────────────────────────────────
	api := r.Group("/api/v1")

	// Public auth endpoints
	auth := api.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/refresh", handlers.Refresh)
		auth.POST("/logout", middleware.JWTAuth(), handlers.Logout)
	}

	// Public data endpoints (no auth required)
	api.GET("/explore", handlers.GetExplore)

	// Protected endpoints — valid JWT required
	protected := api.Group("/")
	protected.Use(middleware.JWTAuth())
	{
		protected.GET("/me", handlers.GetMe)
		protected.GET("/skills", handlers.GetSkills)
		protected.POST("/skills", handlers.CreateSkill)
		protected.PUT("/skills/:id", handlers.UpdateSkill)
		protected.DELETE("/skills/:id", handlers.DeleteSkill)
		protected.GET("/artworks", handlers.GetArtworks)
		protected.POST("/artworks", handlers.CreateArtwork)
		protected.GET("/uploads", handlers.GetUploads)
		protected.POST("/uploads", handlers.CreateUpload)
		protected.PUT("/uploads/:id", handlers.UpdateUpload)
		protected.DELETE("/uploads/:id", handlers.DeleteUpload)
	}

	// Admin-only endpoints — valid JWT + role:"admin" required
	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.JWTAuth(), middleware.CheckAdmin())
	{
		adminGroup.GET("/submissions", handlers.GetSubmissions)
	}

	// ── Start ───────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
