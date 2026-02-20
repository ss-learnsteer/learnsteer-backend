package quiz

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware" // Adjust to your module path
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupQuizTestEnv initializes an in-memory DB, seeds a quiz, and wires up the router
func setupQuizTestEnv() (*gin.Engine, *gorm.DB, uint) {
	// 1. Setup In-Memory SQLite Database
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&Quiz{}, &Question{}, &Option{})

	// 2. Seed an initial "Old" Quiz
	initialQuiz := Quiz{
		Title:       "Old Physics Quiz",
		Description: "Version 1.0",
		Questions: []Question{
			{
				TextMarkdown: "Old Question 1",
				Options: []Option{
					{Text: "Old Option A"},
					{Text: "Old Option B"},
				},
			},
		},
	}
	db.Create(&initialQuiz)

	// 3. Setup Service, Handler, and Router
	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// 4. Register the specific PUT route with the feature flag middleware
	router.PUT(
		"/api/v1/quizzes/:id",
		middleware.FeatureToggle("ENABLE_QUIZ_CREATION"),
		handler.UpdateQuiz,
	)

	return router, db, initialQuiz.ID
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

		// This payload simulates what the React frontend will send
		// Notice it has NO question or option IDs, just pure fresh data
		payload := map[string]interface{}{
			"title":       "New Physics Quiz (Updated)",
			"description": "Version 2.0",
			"questions": []map[string]interface{}{
				{
					"text": "Brand New Question 1",
					"type": "mcq",
					"options": []map[string]interface{}{
						{"text": "New Option X", "is_correct": true},
						{"text": "New Option Y", "is_correct": false},
						{"text": "New Option Z", "is_correct": false},
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
