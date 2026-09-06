package quiz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware" // Adjust to your module path
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

// MockAuthMiddleware injects fake JWT claims into the context for testing
func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read fake headers sent by our tests and set them in the Gin context
		c.Set("user_role", c.GetHeader("X-Test-Role"))
		c.Set("user_medium", c.GetHeader("X-Test-Medium"))
		c.Next()
	}
}

type TestUser struct {
	ID        uint   `gorm:"primaryKey"`
	StudentID string `json:"student_id"`
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	School    string `json:"school"`
	District  string `json:"district"`
}

type TestSubmission struct {
	ID          uint       `gorm:"primaryKey"`
	UserID      uint       `json:"user_id"`
	QuizID      uint       `json:"quiz_id"`
	Score       int        `json:"score"`
	CompletedAt *time.Time `json:"completed_at"`
}

// setupQuizTestEnv initializes an in-memory DB, seeds a quiz, and wires up the router
func setupQuizTestEnv() (*gin.Engine, *gorm.DB, uint) {
	// 1. Setup In-Memory SQLite Database
	dbName := fmt.Sprintf("file:testdb%d?mode=memory&cache=private", atomic.AddUint64(&testDBCounter, 1))
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	db.Table("users").AutoMigrate(&TestUser{})
	db.Table("submissions").AutoMigrate(&TestSubmission{})
	db.AutoMigrate(&Quiz{}, &Question{}, &Option{})

	// Seed multiple quizzes with different mediums.
	// IsVisible must be true so the student role filter (onlyVisible=true) can find them.
	visible := true
	quizzes := []Quiz{
		{Title: "Sinhala Mock Exam", Medium: "Sinhala", IsVisible: &visible},
		{Title: "Sinhala Term Test", Medium: "Sinhala", IsVisible: &visible},
		{Title: "English Mock Exam", Medium: "English", IsVisible: &visible},
		{Title: "Tamil Mock Exam", Medium: "Tamil", IsVisible: &visible},
	}
	for _, q := range quizzes {
		db.Create(&q)
	}

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Register routes for testing
	router.PUT(
		"/api/v1/quizzes/:id",
		middleware.FeatureToggle("ENABLE_QUIZ_CREATION"),
		handler.UpdateQuiz,
	)

	// NEW: Register the GET route with our Mock JWT Middleware
	router.GET("/api/v1/quizzes", MockAuthMiddleware(), handler.ListQuizzes)
	router.GET("/api/v1/quizzes/:id/leaderboard", MockAuthMiddleware(), handler.GetLeaderboard)

	// Return the ID of the first quiz for the PUT tests
	var firstQuiz Quiz
	db.First(&firstQuiz)
	return router, db, firstQuiz.ID
}

func TestUpdateQuiz(t *testing.T) {
	router, db, quizID := setupQuizTestEnv()
	quizIDStr := strconv.Itoa(int(quizID))

	// ---------------------------------------------------------
	// TEST CASE 1: Successful Complete Replacement
	// ---------------------------------------------------------
	t.Run("Successfully Replaces Quiz Content", func(t *testing.T) {
		// Enable the feature flag for this test
		os.Setenv("ENABLE_QUIZ_CREATION", "true")
		defer os.Unsetenv("ENABLE_QUIZ_CREATION")

		// This payload simulates what the React frontend will send.
		// Field names must match the CreateQuizRequest binding tags exactly.
		payload := map[string]interface{}{
			"title":       "New Physics Quiz (Updated)",
			"description": "Version 2.0",
			"medium":      "Sinhala",                     // required
			"stream":      []string{"Physical Science"}, // required, min=1
			"questions": []map[string]interface{}{
				{
					"text_markdown":  "Brand New Question 1", // must be text_markdown
					"type":           "mcq",
					"correct_answer": "a", // must be correct_answer, len=1
					"options": []map[string]interface{}{
						{"text": "New Option X"},
						{"text": "New Option Y"},
						{"text": "New Option Z"},
					},
				},
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PUT", "/api/v1/quizzes/"+quizIDStr, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 1. Assert HTTP Status
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// 2. Assert Database State: Parent Quiz metadata updated
		var updatedQuiz Quiz
		db.First(&updatedQuiz, quizID)
		if updatedQuiz.Title != "New Physics Quiz (Updated)" {
			t.Errorf("Expected title to be updated, got '%s'", updatedQuiz.Title)
		}

		// 3. Assert Database State: Old data was wiped and new data was inserted
		var questionCount int64
		var optionCount int64

		db.Model(&Question{}).Count(&questionCount)
		db.Model(&Option{}).Count(&optionCount)

		// We should have exactly 1 question and 3 options in the ENTIRE database now
		if questionCount != 1 {
			t.Errorf("Expected exactly 1 question in DB, found %d", questionCount)
		}
		if optionCount != 3 {
			t.Errorf("Expected exactly 3 options in DB, found %d", optionCount)
		}

		// Ensure the old texts are completely gone
		var oldQuestion Question
		if err := db.Where("text_markdown = ?", "Old Question 1").First(&oldQuestion).Error; err == nil {
			t.Errorf("Old question was not deleted!")
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 2: Feature Flag Blocked
	// ---------------------------------------------------------
	t.Run("Feature Flag Blocks Update", func(t *testing.T) {
		// Turn the API off
		os.Setenv("ENABLE_QUIZ_CREATION", "false")
		defer os.Unsetenv("ENABLE_QUIZ_CREATION")

		req, _ := http.NewRequest("PUT", "/api/v1/quizzes/"+quizIDStr, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert HTTP 503 Service Unavailable
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}
	})
}

func TestListQuizzesWithMediumFilter(t *testing.T) {
	router, _, _ := setupQuizTestEnv()

	// ---------------------------------------------------------
	// TEST CASE 1: Student is strictly locked to their medium
	// ---------------------------------------------------------
	t.Run("Student gets only Sinhala quizzes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/quizzes", nil)
		// Simulate a JWT belonging to a Sinhala student
		req.Header.Set("X-Test-Role", "student")
		req.Header.Set("X-Test-Medium", "Sinhala")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		// Parse the response
		var response struct {
			Data []Quiz                 `json:"data"`
			Meta map[string]interface{} `json:"meta"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		// We seeded exactly 2 Sinhala quizzes
		if len(response.Data) != 2 {
			t.Errorf("Expected 2 Sinhala quizzes, got %d", len(response.Data))
		}
		// Verify no other mediums sneaked in
		for _, q := range response.Data {
			if q.Medium != "Sinhala" {
				t.Errorf("Security breach: Student saw a %s quiz!", q.Medium)
			}
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 2: Admin bypassing the lock (sees all)
	// ---------------------------------------------------------
	t.Run("Admin sees all quizzes when no query param is passed", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/quizzes", nil)
		// Simulate a JWT belonging to an admin
		req.Header.Set("X-Test-Role", "admin")
		req.Header.Set("X-Test-Medium", "English") // Their personal medium shouldn't matter

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response struct {
			Data []Quiz `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		// We seeded 4 total quizzes across all mediums
		if len(response.Data) != 4 {
			t.Errorf("Expected 4 total quizzes for Admin, got %d", len(response.Data))
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 3: Admin manually filtering via URL
	// ---------------------------------------------------------
	t.Run("Admin can manually filter by URL query", func(t *testing.T) {
		// Admin requests only Tamil quizzes via the URL
		req, _ := http.NewRequest("GET", "/api/v1/quizzes?medium=Tamil", nil)
		req.Header.Set("X-Test-Role", "admin")
		req.Header.Set("X-Test-Medium", "English")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response struct {
			Data []Quiz `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		// We seeded exactly 1 Tamil quiz
		if len(response.Data) != 1 {
			t.Errorf("Expected 1 Tamil quiz for Admin filter, got %d", len(response.Data))
		}
		if len(response.Data) > 0 && response.Data[0].Medium != "Tamil" {
			t.Errorf("Expected Tamil quiz, got %s", response.Data[0].Medium)
		}
	})
}

func TestGetLeaderboard(t *testing.T) {
	router, _, quizID := setupQuizTestEnv()
	quizIDStr := strconv.Itoa(int(quizID))

	t.Run("Returns empty leaderboard when no submissions exist", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/quizzes/"+quizIDStr+"/leaderboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response struct {
			Success bool               `json:"success"`
			QuizID  int                `json:"quiz_id"`
			Data    []LeaderboardEntry `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		if !response.Success {
			t.Errorf("Expected success to be true")
		}
		if response.QuizID != int(quizID) {
			t.Errorf("Expected quiz_id %d, got %d", quizID, response.QuizID)
		}
		if len(response.Data) != 0 {
			t.Errorf("Expected 0 leaderboard entries, got %d", len(response.Data))
		}
	})
}
