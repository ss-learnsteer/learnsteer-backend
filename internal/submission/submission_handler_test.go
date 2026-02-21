package submission

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz" // Adjust to your actual module path
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSubmissionTestEnv initializes the DB, seeds a test quiz, and mocks the auth middleware
func setupSubmissionTestEnv() (*gin.Engine, *gorm.DB, uint, uint) {
	// 1. Setup In-Memory DB
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})

	// Migrate all necessary tables, including quiz.Option!
	db.AutoMigrate(&quiz.Quiz{}, &quiz.Question{}, &quiz.Option{}, &Submission{}, &Answer{})

	// 2. Seed Test Data (The new "Truth" in the database)
	testQuiz := quiz.Quiz{Title: "Physics Unit 1"}
	db.Create(&testQuiz)

	q1 := quiz.Question{
		QuizID:        testQuiz.ID,
		Type:          quiz.TypeMCQ,
		Points:        10,
		CorrectAnswer: "b", // The grading service compares selected_option against this
		Options: []quiz.Option{
			{Text: "10 m/s", IsCorrect: false}, // "a"
			{Text: "14 m/s", IsCorrect: true},  // "b" - correct
			{Text: "12 m/s", IsCorrect: false}, // "c"
		},
	}
	db.Create(&q1)

	// 3. Setup Router and Handlers
	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// 4. Mock the Auth Middleware
	// We inject a fake userID into the context, perfectly matching your AuthMiddleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(99)) // Mock Student ID
		c.Next()
	})

	router.POST("/submissions", handler.SubmitQuiz)

	return router, db, testQuiz.ID, q1.ID
}

func TestAutoGradingLogic(t *testing.T) {
	router, _, quizID, q1ID := setupSubmissionTestEnv()

	// ---------------------------------------------------------
	// TEST CASE 1: Correct Answer
	// ---------------------------------------------------------
	t.Run("Calculates Points for Correct Answer", func(t *testing.T) {
		// Use the DTO struct that the handler expects
		payload := SubmitQuizPayload{
			QuizID: quizID,
			Answers: []SubmitAnswerPayload{
				{QuestionID: q1ID, SelectedOption: "b"}, // "b" maps to index 1 (IsCorrect: true)
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/submissions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Parse the JSON response
		var response struct {
			Data struct {
				Score int `json:"score"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response.Data.Score != 10 {
			t.Errorf("Expected score 10, got %d", response.Data.Score)
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 2: Incorrect Answer
	// ---------------------------------------------------------
	t.Run("Assigns Zero Points for Wrong Answer", func(t *testing.T) {
		payload := SubmitQuizPayload{
			QuizID: quizID,
			Answers: []SubmitAnswerPayload{
				{QuestionID: q1ID, SelectedOption: "a"}, // "a" maps to index 0 (IsCorrect: false)
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/submissions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response struct {
			Data struct {
				Score int `json:"score"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response.Data.Score != 0 {
			t.Errorf("Expected score 0, got %d", response.Data.Score)
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 3: Out of Bounds & Security Resilience
	// ---------------------------------------------------------
	t.Run("Resilient to Out of Bounds and Fake IDs", func(t *testing.T) {
		payload := SubmitQuizPayload{
			QuizID: quizID,
			Answers: []SubmitAnswerPayload{
				{QuestionID: q1ID, SelectedOption: "z"}, // Malicious out-of-bounds letter
				{QuestionID: 9999, SelectedOption: "b"}, // Fake Question ID
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/submissions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response struct {
			Data struct {
				Score int `json:"score"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response.Data.Score != 0 {
			t.Errorf("Expected score 0 from malicious payload, got %d", response.Data.Score)
		}
	})
}
