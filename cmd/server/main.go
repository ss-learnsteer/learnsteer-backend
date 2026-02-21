package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"

	// Internal Modules
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/auth"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/database"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/middleware" // Added Middleware
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/submission"
)

func main() {
	// 0. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, relying on system environment variables")
	}

	// 1. DB Connection
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("❌ Database initialization failed: %v", err)
	}

	// 2. Migrations
	db.AutoMigrate(
		&auth.User{},
		&quiz.Quiz{},
		&quiz.Question{},
		&submission.Submission{},
		&submission.Answer{},
		&quiz.Option{},
		&auth.SSOTicket{},
	)

	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. Initialize Services & Handlers (Dependency Injection)
	// Auth Module
	authService := auth.NewService(db)
	authHandler := auth.NewHandler(authService)

	// Quiz Module
	quizService := quiz.NewService(db)
	quizHandler := quiz.NewHandler(quizService)

	// Submission Module
	submissionService := submission.NewService(db)
	submissionHandler := submission.NewHandler(submissionService)

	// 4. Setup Router
	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// Configure CORS (Important for React Frontend)
	// In production, you might want to restrict AllowOrigins to your specific domain
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	v1 := r.Group("/api/v1")
	{
		// ----------------------------
		// A. Public Routes (No Login)
		// ----------------------------

		// 1. Health Check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "up", "database": "connected"})
		})

		// 2. Auth Routes (Register, Login)
		authHandler.RegisterRoutes(v1)

		// ----------------------------
		// B. Protected Routes (Login Required)
		// ----------------------------
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// 3. Quiz Routes
			// Now all quiz endpoints require a valid 'Authorization: Bearer <token>' header
			quizHandler.RegisterRoutes(protected)
			submissionHandler.RegisterRoutes(protected)
		}
	}

	// 6. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Starting ss-quiz-platform on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
