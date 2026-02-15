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
func setupSubmissionTestEnv() (*gin.Engine, *gorm.DB, uint, uint, uint) {
	// 1. Setup In-Memory DB
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	
	// Migrate both Quiz and Submission tables
	db.AutoMigrate(&quiz.Quiz{}, &quiz.Question{}, &Submission{}, &Answer{})

	// 2. Seed Test Data (The "Truth" in the database)
	testQuiz := quiz.Quiz{Title: "Physics Unit 1"}
	db.Create(&testQuiz)

	q1 := quiz.Question{
		QuizID:        testQuiz.ID,
		Type:          quiz.TypeMCQ, // Assuming "mcq"
		Points:        10,
		CorrectAnswer: "b",
	}
	db.Create(&q1)

	q2 := quiz.Question{
		QuizID:        testQuiz.ID,
		Type:          quiz.TypeText, // Assuming "text"
		Points:        5,
		CorrectAnswer: "(?i)vector", // Regex: case-insensitive match for "vector"
	}
	db.Create(&q2)

	// 3. Setup Router and Handlers
	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// 4. Mock the Auth Middleware
	// We inject a fake userID into the context, exactly like your AuthMiddleware does
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(99)) // Mock Student ID
		c.Next()
	})

	router.POST("/submissions", handler.SubmitQuiz)

	return router, db, testQuiz.ID, q1.ID, q2.ID
}

func TestAutoGradingLogic(t *testing.T) {
	router, _, quizID, q1ID, q2ID := setupSubmissionTestEnv()

	// ---------------------------------------------------------
	// TEST CASE 1: Perfect Score (15/15 points)
	// ---------------------------------------------------------
	t.Run("Perfect Score - All answers correct", func(t *testing.T) {
		payload := Submission{
			QuizID: quizID,
			Answers: []Answer{
				{QuestionID: q1ID, SelectedOption: "b"},                            // Correct MCQ
				{QuestionID: q2ID, TextResponse: "Velocity is a Vector quantity"}, // Correct Regex Match
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

		// Parse the JSON response to check the calculated score
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		expectedScore := float64(15) // JSON numbers parse as float64 in Go map[string]interface{}
		if response["score"] != expectedScore {
			t.Errorf("Expected score 15, got %v", response["score"])
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 2: Partial Score (10/15 points)
	// ---------------------------------------------------------
	t.Run("Partial Score - One wrong, one right", func(t *testing.T) {
		payload := Submission{
			QuizID: quizID,
			Answers: []Answer{
				{QuestionID: q1ID, SelectedOption: "b"},      // Correct (10 pts)
				{QuestionID: q2ID, TextResponse: "scalar"},   // Wrong text (0 pts)
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/submissions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["score"] != float64(10) {
			t.Errorf("Expected score 10, got %v", response["score"])
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 3: Zero Score & Invalid Question ID
	// ---------------------------------------------------------
	t.Run("Zero Score and Resilient to Fake Question IDs", func(t *testing.T) {
		payload := Submission{
			QuizID: quizID,
			Answers: []Answer{
				{QuestionID: q1ID, SelectedOption: "c"}, // Wrong MCQ
				{QuestionID: 9999, SelectedOption: "a"}, // Fake Question ID (Hacker attempt)
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/submissions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["score"] != float64(0) {
			t.Errorf("Expected score 0, got %v", response["score"])
		}
	})
}