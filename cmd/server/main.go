package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/database"
)

func main() {
	// 1. Initialize Database
	// We crash the app if DB fails, as we can't run without it
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("❌ Database initialization failed: %v", err)
	}

	// Just for testing/GSoC demo: Auto-migrate schema (creates tables automatically)
	// In production, you might use a migration tool, but this is great for GSoC students.
	// db.AutoMigrate(&User{}, &Quiz{}) // We will add models here later

	// 1. Setup Router
	r := gin.Default()

	// 2. Middleware: GZIP Compression (Saves Azure Bandwidth)
	// Compresses API responses. Critical for sending large Quiz JSONs.
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// 3. Middleware: CORS (Allows React Frontend access)
	// In production, change AllowAllOrigins to your specific domain.
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 4. Health Check (For Azure Load Balancer)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "ss-quiz-platform"})
	})

	// 5. API Versioning Group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			// Check if DB is actually alive
			sqlDB, _ := db.DB()
			if err := sqlDB.Ping(); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_down"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "up", "database": "connected"})
		})

		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Login implementation pending"})
			})
			authGroup.POST("/register", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Register implementation pending"})
			})
		}

		quizGroup := v1.Group("/quizzes")
		{
			// The "Mega-Fetch" endpoint we discussed
			quizGroup.GET("/:id/start", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Quiz fetch optimized implementation pending"})
			})
		}
	}

	// 6. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting ss-quiz-platform on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
