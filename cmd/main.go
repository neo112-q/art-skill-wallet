package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"art-skill-wallet/pkg/db"
	"art-skill-wallet/pkg/handlers"
	"art-skill-wallet/pkg/middleware"
)

func main() {
	// ── Database ────────────────────────────────────────────────────────
	if err := db.Connect(); err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	db.CreateIndexes()

	// ── Router ──────────────────────────────────────────────────────────
	r := gin.Default()

	// Static assets
	r.Static("/css", "./css")
	r.Static("/js", "./js")
	r.Static("/img", "./img")

	// ── Public HTML pages (no token required) ───────────────────────────
	r.StaticFile("/", "./home.html")
	r.StaticFile("/home.html", "./home.html")
	r.StaticFile("/explore.html", "./explore.html")

	// ── Protected HTML pages (JS guard handles redirect) ────────────────
	// HTML is served freely; the real lock is on the API.
	// Each page includes auth.js which redirects to home if no valid token.
	r.StaticFile("/profile.html", "./profile.html")
	r.StaticFile("/history.html", "./history.html")
	r.StaticFile("/upload.html", "./upload.html")
	r.StaticFile("/skillManage.html", "./skillManage.html")
	r.StaticFile("/editProfile.html", "./editProfile.html")
	r.StaticFile("/admin.html", "./admin.html")

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

	// Protected endpoints — valid JWT required
	protected := api.Group("/")
	protected.Use(middleware.JWTAuth())
	{
		protected.GET("/me", handlers.GetMe)
		protected.GET("/skills", handlers.GetSkills)
		protected.POST("/skills", handlers.CreateSkill)
		protected.GET("/artworks", handlers.GetArtworks)
		protected.POST("/artworks", handlers.CreateArtwork)
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
