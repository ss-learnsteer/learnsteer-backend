package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/auth"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/lesson"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/submission"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupDashboardTestEnv() (*gin.Engine, *gorm.DB, uint) {
	dbName := fmt.Sprintf("file:dashboardtestdb%d?mode=memory&cache=private", atomic.AddUint64(&testDBCounter, 1))
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})

	db.AutoMigrate(
		&auth.User{},
		&lesson.Subject{},
		&lesson.Unit{},
		&lesson.Lesson{},
		&lesson.UserLessonProgress{},
		&quiz.Quiz{},
		&quiz.Question{},
		&submission.Submission{},
		&submission.Answer{},
	)

	// Seed User
	user := auth.User{
		Email:     "kasun@example.com",
		FirstName: "Kasun",
		LastName:  "Perera",
		Role:      "student",
		Stream:    "Bio Science",
		Medium:    "Sinhala",
		StudentID:      "LS-test12345",
		Nickname:       "BraveLion42",
		NIC:            "200112345678",
		WhatsappNumber: "0771234567",
		ALYear:         "2024 Batch",
	}
	db.Create(&user)

	// Seed Subjects
	bio := lesson.Subject{Name: "Biology", Code: "BIO", Stream: "Bio Science", Medium: "Sinhala", Color: "#059669"}
	chem := lesson.Subject{Name: "Chemistry", Code: "CHEM", Stream: "Bio Science", Medium: "Sinhala", Color: "#7c3aed"}
	db.Create(&bio)
	db.Create(&chem)

	// Seed Unit & Lesson
	unit := lesson.Unit{SubjectID: bio.ID, UnitNumber: 1, Name: "Cell Biology"}
	db.Create(&unit)

	les := lesson.Lesson{UnitID: unit.ID, LessonNumber: 1, Title: "Cell Structure", VideoURL: "https://example.com/video", IsVisible: true}
	db.Create(&les)

	// Seed Progress
	db.Create(&lesson.UserLessonProgress{UserID: user.ID, LessonID: les.ID, IsCompleted: true, ProgressPercent: 100})

	// Seed Quiz
	visible := true
	q := quiz.Quiz{Title: "Organic Chemistry", Medium: "Sinhala", IsVisible: &visible}
	db.Create(&q)

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock Auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Next()
	})

	handler.RegisterRoutes(router.Group("/api/v1"))

	return router, db, user.ID
}

func TestGetDashboard(t *testing.T) {
	router, _, _ := setupDashboardTestEnv()

	t.Run("Returns complete dashboard view for authenticated student", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool                 `json:"success"`
			Data    DashboardResponseDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}

		// 1. Verify User Profile & Day Streak
		if resp.Data.User.FullName != "Kasun Perera" {
			t.Errorf("Expected user name Kasun Perera, got %s", resp.Data.User.FullName)
		}
		if resp.Data.User.DayStreak <= 0 {
			t.Errorf("Expected positive day streak, got %d", resp.Data.User.DayStreak)
		}
		if resp.Data.User.Batch != "2024 Batch" {
			t.Errorf("Expected 2024 Batch, got %s", resp.Data.User.Batch)
		}

		// 2. Verify Syllabus Coverage
		if resp.Data.SyllabusCoverage.Percentage <= 0 {
			t.Errorf("Expected syllabus coverage percentage > 0, got %d", resp.Data.SyllabusCoverage.Percentage)
		}

		// 3. Verify Next Mock Exam
		if resp.Data.NextMockExam.Title == "" {
			t.Errorf("Expected next mock exam title to be present")
		}

		// 4. Verify Subject Progresses
		if len(resp.Data.SubjectProgress) < 2 {
			t.Errorf("Expected at least 2 subject progresses, got %d", len(resp.Data.SubjectProgress))
		}

		// 5. Verify Smart Revision Box
		if resp.Data.SmartRevision.Topic == "" {
			t.Errorf("Expected smart revision topic to be present")
		}
		if resp.Data.SmartRevision.AccuracyPct <= 0 {
			t.Errorf("Expected smart revision accuracy percentage > 0, got %d", resp.Data.SmartRevision.AccuracyPct)
		}
		if resp.Data.SmartRevision.Message == "" {
			t.Errorf("Expected smart revision message to be present")
		}

		// 6. Verify Upcoming Deadlines
		if len(resp.Data.UpcomingDeadlines) == 0 {
			t.Errorf("Expected upcoming deadlines to be present")
		}
	})
}
