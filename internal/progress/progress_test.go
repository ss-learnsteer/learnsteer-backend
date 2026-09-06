package progress

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

func setupProgressTestEnv() (*gin.Engine, *gorm.DB, uint) {
	dbName := fmt.Sprintf("file:progresstestdb%d?mode=memory&cache=private", atomic.AddUint64(&testDBCounter, 1))
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})

	db.AutoMigrate(
		&auth.User{},
		&lesson.Subject{},
		&lesson.Unit{},
		&lesson.Lesson{},
		&lesson.UserLessonProgress{},
		&quiz.Quiz{},
		&submission.Submission{},
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

	// Seed subject, unit, lesson
	bio := lesson.Subject{Name: "Biology", Code: "BIO", Stream: "Bio Science", Medium: "Sinhala"}
	db.Create(&bio)
	unit := lesson.Unit{SubjectID: bio.ID, UnitNumber: 1, Name: "Cell Biology"}
	db.Create(&unit)
	les := lesson.Lesson{UnitID: unit.ID, LessonNumber: 1, Title: "Cell Structure", VideoURL: "https://example.com", IsVisible: true}
	db.Create(&les)

	db.Create(&lesson.UserLessonProgress{UserID: user.ID, LessonID: les.ID, IsCompleted: true})

	// Seed submissions
	db.Create(&submission.Submission{UserID: user.ID, QuizID: 1, Score: 85})
	db.Create(&submission.Submission{UserID: user.ID, QuizID: 2, Score: 92})

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("userID", user.ID)
		c.Set("user_stream", "Bio Science")
		c.Set("user_medium", "Sinhala")
		c.Next()
	})

	handler.RegisterRoutes(router.Group("/api/v1"))

	return router, db, user.ID
}

func TestGetProgress(t *testing.T) {
	router, _, _ := setupProgressTestEnv()

	t.Run("Returns complete progress and achievements payload for authenticated student", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool            `json:"success"`
			Data    ProgressPageDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}

		// 1. Z-Score Card
		if resp.Data.CurrentZScore.Value <= 0 {
			t.Errorf("Expected positive Z-score, got %f", resp.Data.CurrentZScore.Value)
		}
		if resp.Data.CurrentZScore.FormattedValue == "" {
			t.Errorf("Expected formatted Z-score string")
		}

		// 2. Island Rank Card
		if resp.Data.IslandRank.Rank == "" {
			t.Errorf("Expected rank to be present")
		}
		if resp.Data.IslandRank.Percentile != "Top 0.1%" {
			t.Errorf("Expected 'Top 0.1%%', got '%s'", resp.Data.IslandRank.Percentile)
		}
		if resp.Data.IslandRank.NextMilestone != "Top 10 in District" {
			t.Errorf("Expected 'Top 10 in District', got '%s'", resp.Data.IslandRank.NextMilestone)
		}

		// 3. Overall Completion Card
		if resp.Data.OverallCompletion.Percentage <= 0 {
			t.Errorf("Expected positive overall completion percentage")
		}

		// 4. Z-Score Performance Trend
		if len(resp.Data.ZScoreTrends.SixMonths) != 6 {
			t.Errorf("Expected 6 months data points, got %d", len(resp.Data.ZScoreTrends.SixMonths))
		}
		if len(resp.Data.ZScoreTrends.Yearly) != 6 {
			t.Errorf("Expected 6 yearly data points, got %d", len(resp.Data.ZScoreTrends.Yearly))
		}

		// 5. Weekly Activity Heatmap
		if resp.Data.WeeklyActivity.TotalHours <= 0 {
			t.Errorf("Expected positive total activity hours")
		}
		if len(resp.Data.WeeklyActivity.Weeks) != 4 {
			t.Errorf("Expected 4 weeks activity grid, got %d", len(resp.Data.WeeklyActivity.Weeks))
		}

		// 6. Curriculum Mastery
		if len(resp.Data.CurriculumMastery) == 0 {
			t.Errorf("Expected curriculum mastery list")
		}

		// 7. Recent Reports
		if len(resp.Data.RecentReports) < 3 {
			t.Errorf("Expected at least 3 recent reports, got %d", len(resp.Data.RecentReports))
		}

		// 8. Achievements & Badges
		if resp.Data.Achievements.UnlockedCount < 3 {
			t.Errorf("Expected at least 3 unlocked badges, got %d", resp.Data.Achievements.UnlockedCount)
		}
		if len(resp.Data.Achievements.Badges) < 5 {
			t.Errorf("Expected at least 5 badges in total, got %d", len(resp.Data.Achievements.Badges))
		}

		// 9. Daily Motivation
		if resp.Data.DailyMotivation.Quote == "" || resp.Data.DailyMotivation.Author == "" {
			t.Errorf("Expected quote and author to be present")
		}
	})
}
