package examshub

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/auth"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/submission"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupExamsHubTestEnv() (*gin.Engine, *gorm.DB, uint) {
	dbName := fmt.Sprintf("file:examshubtestdb%d?mode=memory&cache=private", atomic.AddUint64(&testDBCounter, 1))
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})

	db.AutoMigrate(
		&auth.User{},
		&quiz.Quiz{},
		&quiz.Question{},
		&quiz.Option{},
		&submission.Submission{},
		&submission.Answer{},
	)

	// Seed user
	user := auth.User{
		Email:     "kasun@example.com",
		FirstName: "Kasun",
		LastName:  "Perera",
		Role:      "student",
		Stream:    "Bio Science",
		Medium:    "Sinhala",
	}
	db.Create(&user)

	// Seed quiz
	visible := true
	q := quiz.Quiz{Title: "Biology Mock Exam #01", Medium: "Sinhala", IsVisible: &visible, DurationMin: 60}
	db.Create(&q)

	// Seed questions
	q1 := quiz.Question{QuizID: q.ID, Type: "mcq", TextMarkdown: "What is mitosis?", CorrectAnswer: "a", Points: 1}
	q2 := quiz.Question{QuizID: q.ID, Type: "mcq", TextMarkdown: "What is DNA?", CorrectAnswer: "b", Points: 1}
	db.Create(&q1)
	db.Create(&q2)

	// Seed submission
	sub := submission.Submission{UserID: user.ID, QuizID: q.ID, Score: 85}
	db.Create(&sub)

	// Seed answers
	db.Create(&submission.Answer{SubmissionID: sub.ID, QuestionID: q1.ID, SelectedOption: "a", IsCorrect: true})
	db.Create(&submission.Answer{SubmissionID: sub.ID, QuestionID: q2.ID, SelectedOption: "c", IsCorrect: false})

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	})

	handler.RegisterRoutes(router.Group("/api/v1"))

	return router, db, user.ID
}

func TestGetExamsHub(t *testing.T) {
	router, _, _ := setupExamsHubTestEnv()

	t.Run("Returns complete exams hub analytics for authenticated student", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/exams/hub", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool                `json:"success"`
			Data    ExamsHubResponseDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}

		// 1. Latest Exam
		if resp.Data.LatestExam.ExamTitle == "" {
			t.Errorf("Expected exam title to be present")
		}
		if resp.Data.LatestExam.MaxScore != 100 {
			t.Errorf("Expected max score 100, got %d", resp.Data.LatestExam.MaxScore)
		}
		if resp.Data.LatestExam.MCQMaxMarks != 40 {
			t.Errorf("Expected MCQ max 40, got %d", resp.Data.LatestExam.MCQMaxMarks)
		}
		if resp.Data.LatestExam.EssayMaxMarks != 60 {
			t.Errorf("Expected Essay max 60, got %d", resp.Data.LatestExam.EssayMaxMarks)
		}

		// 2. Unit Performance
		if len(resp.Data.UnitPerformance) == 0 {
			t.Errorf("Expected unit performance data to be present")
		}

		// 3. Predicted Z Score
		if resp.Data.PredictedZScore.Trend == "" {
			t.Errorf("Expected z-score trend to be present")
		}

		// 4. Island Rank
		if resp.Data.IslandRank.CurrentRank <= 0 {
			t.Errorf("Expected positive island rank, got %d", resp.Data.IslandRank.CurrentRank)
		}
		if resp.Data.IslandRank.Message == "" {
			t.Errorf("Expected island rank message to be present")
		}

		// 5. Badge
		if resp.Data.Badge.Name == "" {
			t.Errorf("Expected badge name to be present")
		}
		if resp.Data.Badge.Description == "" {
			t.Errorf("Expected badge description to be present")
		}

		// 6. Assessment History
		if resp.Data.AssessmentHistory.CurrentWeekPct <= 0 {
			t.Errorf("Expected current week percentage > 0, got %d", resp.Data.AssessmentHistory.CurrentWeekPct)
		}
		if resp.Data.AssessmentHistory.LastWeekLabel != "Last Week" {
			t.Errorf("Expected 'Last Week', got %s", resp.Data.AssessmentHistory.LastWeekLabel)
		}

		// 7. Report URL
		if resp.Data.ReportURL == "" {
			t.Errorf("Expected report URL to be present")
		}
	})
}

func TestGetMockExams(t *testing.T) {
	router, _, _ := setupExamsHubTestEnv()

	t.Run("Returns complete mock exams page payload for authenticated student", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/exams/mocks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool             `json:"success"`
			Data    MockExamsPageDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}

		// 1. Last Result
		if resp.Data.LastResult.Label == "" {
			t.Errorf("Expected last result label to be present")
		}
		if resp.Data.LastResult.Percentage <= 0 {
			t.Errorf("Expected last result percentage > 0, got %d", resp.Data.LastResult.Percentage)
		}

		// 2. Upcoming Schedule
		if len(resp.Data.UpcomingSchedule) == 0 {
			t.Errorf("Expected upcoming mock schedule items to be present")
		}
		if resp.Data.UpcomingSchedule[0].Month == "" || resp.Data.UpcomingSchedule[0].Subject == "" {
			t.Errorf("Expected upcoming mock month and subject to be populated")
		}

		// 3. Eligibility
		if len(resp.Data.Eligibility) == 0 {
			t.Errorf("Expected eligibility items to be present")
		}

		// 4. Strategy
		if resp.Data.Strategy.Message == "" {
			t.Errorf("Expected strategy message to be present")
		}
		if resp.Data.Strategy.StudentsRegistered <= 0 {
			t.Errorf("Expected registered students count > 0, got %d", resp.Data.Strategy.StudentsRegistered)
		}
	})
}

