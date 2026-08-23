package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"

	// Internal Modules
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/auth"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/dashboard"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/examshub"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/lesson"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/database"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/middleware" // Added Middleware
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/progress"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/submission"
)

func main() {
	// 0. Set Global Application Timezone to Sri Lanka (+05:30)
	// This forces GORM's time.Now() to always use IST for CreatedAt/UpdatedAt
	loc, err := time.LoadLocation("Asia/Colombo")
	if err != nil {
		log.Printf("⚠️  Could not load Asia/Colombo timezone, falling back to UTC: %v", err)
	} else {
		time.Local = loc
		log.Println("🕒 System timezone set to Asia/Colombo (+05:30)")
	}

	// 0. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, relying on system environment variables")
	}

	// 1. DB Connection
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("❌ Database initialization failed: %v", err)
	}

	// 2. Migrations (Run in background to guarantee instant Heroku port binding)
	go func() {
		log.Println("🔄 Running database AutoMigrate in background...")
		if err := db.AutoMigrate(
			&auth.User{},
			&quiz.Quiz{},
			&quiz.Question{},
			&submission.Submission{},
			&submission.Answer{},
			&quiz.Option{},
			&auth.SSOTicket{},
			&lesson.Subject{},
			&lesson.Unit{},
			&lesson.Lesson{},
			&lesson.LessonNote{},
			&lesson.LessonResource{},
			&lesson.UserLessonProgress{},
			&lesson.RevisionModule{},
			&lesson.PastPaper{},
		); err != nil {
			log.Printf("⚠️ Background migration warning: %v", err)
		} else {
			log.Println("✅ Database migration completed successfully")
		}
	}()

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

	// Lesson Module
	lessonService := lesson.NewService(db)
	lessonHandler := lesson.NewHandler(lessonService)

	// Dashboard Module
	dashboardService := dashboard.NewService(db)
	dashboardHandler := dashboard.NewHandler(dashboardService)

	// Exams Hub Module
	examsHubService := examshub.NewService(db)
	examsHubHandler := examshub.NewHandler(examsHubService)

	// Progress & Achievements Module
	progressService := progress.NewService(db)
	progressHandler := progress.NewHandler(progressService)

	// 4. Setup Router
	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// Configure CORS (Important for React Frontend)
	// In production, you might want to restrict AllowOrigins to your specific domain
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(config))

	v1 := r.Group("/api/v1")
	{
		// ----------------------------
		// A. Public Routes (No Login)
		// ----------------------------

		// 1. Health Check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "up", "database": "connected", "time": time.Now().Format(time.RFC3339)})		
		})

		// 2. Pre-Warm / Wakeup Endpoint (Public)
        // Triggers the DB to wake up from scale-to-zero
        v1.GET("/wakeup", quizHandler.WakeUp)

		// 3. Auth Routes (Register, Login)
		authHandler.RegisterRoutes(v1)

		// 4. Public Platform Configurations (Server-Driven Landing, About, Contact)
		v1.GET("/public/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"platform_name": "LearnSteer by Sasnaka Sansada",
					"tagline":       "Free A/L Students Sri Lanka - Learn Smarter. Score Higher.",
					"description":   "The all-in-one free self-learning portal for Sri Lankan A/L students - videos, past papers, revision notes & mock exams.",
					"badge":         "Official Platform For Sri Lankan Students",
					"stats": gin.H{
						"students_enrolled": "+50,000",
						"video_lessons":     "+1,200",
						"past_papers":       "+500",
						"island_rank_coverage": "99.8%",
					},
				},
			})
		})

		v1.GET("/public/about", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"title": "About Sasnaka Sansada LearnSteer",
					"mission": "Empowering every Sri Lankan Advanced Level student with world-class, free educational resources.",
					"vision": "Bridging the educational inequality gap across all 25 districts through technology and peer mentorship.",
					"story": "Sasnaka Sansada is a non-profit youth organization committed to uplifting educational standards across Sri Lanka.",
					"pillars": []string{"High Quality Video Lessons", "Structured Revision Notes", "Island-wide Mock Exams", "Real-Time Ranking & Z-Score Diagnostics"},
				},
			})
		})

		v1.GET("/public/contact", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"email": "support@learnsteer.lk",
					"phone": "+94 11 234 5678",
					"hotline": "+94 77 123 4567",
					"office_address": "Sasnaka Sansada Headquarters, Colombo, Sri Lanka",
					"working_hours": "Monday - Saturday: 8:30 AM - 5:30 PM",
				},
			})
		})

		// ----------------------------
		// B. Protected Routes (Login Required)
		// ----------------------------
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// Quiz, Submission, Lesson, and Dashboard Routes
			// Require a valid 'Authorization: Bearer <token>' header
			quizHandler.RegisterRoutes(protected)
			submissionHandler.RegisterRoutes(protected)
			lessonHandler.RegisterRoutes(protected)
			dashboardHandler.RegisterRoutes(protected)
			examsHubHandler.RegisterRoutes(protected)
			progressHandler.RegisterRoutes(protected)
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
