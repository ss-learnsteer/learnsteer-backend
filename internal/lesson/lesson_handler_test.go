package lesson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupLessonTestEnv() (*gin.Engine, *gorm.DB, uint, uint, uint) {
	dbName := fmt.Sprintf("file:lessontestdb%d?mode=memory&cache=private", atomic.AddUint64(&testDBCounter, 1))
	db, _ := gorm.Open(sqlite.Open(dbName), &gorm.Config{})

	db.AutoMigrate(
		&Subject{},
		&Unit{},
		&Lesson{},
		&LessonNote{},
		&LessonResource{},
		&UserLessonProgress{},
		&RevisionModule{},
		&PastPaper{},
	)

	// Seed Subject
	sub := Subject{
		Name:   "Biology",
		Code:   "BIO",
		Stream: "Bio Science",
		Medium: "Sinhala",
		Color:  "#059669",
	}
	db.Create(&sub)

	// Seed Unit
	unit := Unit{
		SubjectID:   sub.ID,
		UnitNumber:  1,
		Name:        "Cell Biology",
		Description: "Fundamentals of cell biology",
		Color:       "#059669",
	}
	db.Create(&unit)

	// Seed Lesson
	les := Lesson{
		UnitID:       unit.ID,
		LessonNumber: 1,
		Title:        "Cell Division: Mitosis",
		Instructor:   "Dr. S. Perera",
		VideoURL:     "https://www.youtube.com/embed/fJfTDc3WzQ8",
		DurationMin:  32,
		LessonType:   "Animation Pack",
		IsVisible:    true,
	}
	db.Create(&les)

	// Seed Note
	note := LessonNote{
		LessonID:        les.ID,
		Title:           "Introduction to Mitosis",
		ContentMarkdown: "Mitosis is part of the cell cycle...",
		SectionType:     "text",
	}
	db.Create(&note)

	// Seed Resource
	res := LessonResource{
		LessonID: les.ID,
		Title:    "Mitosis Diagram Pack",
		FileURL:  "https://example.com/mitosis.pdf",
		FileType: "PDF",
		FileSize: "4.2 MB",
	}
	db.Create(&res)

	// Seed Revision Module
	rev := RevisionModule{
		SubjectID:   sub.ID,
		Title:       "Cell Biology Short Notes",
		TopicsCount: 16,
		IsPopular:   true,
	}
	db.Create(&rev)

	// Seed Past Paper
	pp := PastPaper{
		SubjectID: sub.ID,
		Year:      2023,
		Title:     "2023 G.C.E. A/L Biology",
		Badge:     "A/L PAST PAPER",
		IsModel:   false,
	}
	db.Create(&pp)

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock Auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(101))
		c.Set("user_role", c.GetHeader("X-Test-Role"))
		c.Set("user_medium", c.GetHeader("X-Test-Medium"))
		c.Set("user_stream", c.GetHeader("X-Test-Stream"))
		c.Next()
	})

	handler.RegisterRoutes(router.Group("/api/v1"))

	return router, db, sub.ID, unit.ID, les.ID
}

func TestListSubjects(t *testing.T) {
	router, _, _, _, _ := setupLessonTestEnv()

	t.Run("Student fetches subjects matching their stream and medium", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/subjects", nil)
		req.Header.Set("X-Test-Role", "student")
		req.Header.Set("X-Test-Stream", "Bio Science")
		req.Header.Set("X-Test-Medium", "Sinhala")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Stream     string              `json:"stream"`
				IslandRank int                 `json:"island_rank"`
				Subjects   []SubjectSummaryDTO `json:"subjects"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}
		if resp.Data.Stream != "Bio Science" {
			t.Errorf("Expected stream 'Bio Science', got '%s'", resp.Data.Stream)
		}
		if resp.Data.IslandRank <= 0 {
			t.Errorf("Expected positive island rank, got %d", resp.Data.IslandRank)
		}
		if len(resp.Data.Subjects) != 1 {
			t.Fatalf("Expected 1 subject, got %d", len(resp.Data.Subjects))
		}
		if resp.Data.Subjects[0].Name != "Biology" {
			t.Errorf("Expected subject Biology, got %s", resp.Data.Subjects[0].Name)
		}
		if resp.Data.Subjects[0].TotalLessons != 1 {
			t.Errorf("Expected 1 total lesson, got %d", resp.Data.Subjects[0].TotalLessons)
		}
	})
}

func TestGetSubjectUnits(t *testing.T) {
	router, _, subID, _, _ := setupLessonTestEnv()

	t.Run("Fetches units and nested lessons for subject", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/subjects/%d/units", subID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool             `json:"success"`
			Data    []UnitSummaryDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data) != 1 {
			t.Fatalf("Expected 1 unit, got %d", len(resp.Data))
		}
		if resp.Data[0].Name != "Cell Biology" {
			t.Errorf("Expected unit Cell Biology, got %s", resp.Data[0].Name)
		}
		if len(resp.Data[0].Lessons) != 1 {
			t.Fatalf("Expected 1 lesson inside unit, got %d", len(resp.Data[0].Lessons))
		}
		if resp.Data[0].Lessons[0].Title != "Cell Division: Mitosis" {
			t.Errorf("Expected lesson title Cell Division: Mitosis, got %s", resp.Data[0].Lessons[0].Title)
		}
	})
}

func TestGetLessonDetail(t *testing.T) {
	router, _, _, _, lesID := setupLessonTestEnv()

	t.Run("Fetches full lesson player context with notes and resources", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/lessons/%d", lesID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool            `json:"success"`
			Data    LessonDetailDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.Data.Title != "Cell Division: Mitosis" {
			t.Errorf("Expected lesson title Cell Division: Mitosis, got %s", resp.Data.Title)
		}
		if len(resp.Data.Tabs) != 4 {
			t.Errorf("Expected 4 tabs, got %d", len(resp.Data.Tabs))
		}
		if len(resp.Data.Notes) == 0 {
			t.Errorf("Expected lesson notes to be present")
		}
		if len(resp.Data.Resources) == 0 {
			t.Errorf("Expected lesson resources to be present")
		}
		if resp.Data.NextUp.Title == "" {
			t.Errorf("Expected Next Up lesson title to be present")
		}
		if resp.Data.UnitProgress.TotalLessons <= 0 {
			t.Errorf("Expected unit progress total lessons > 0, got %d", resp.Data.UnitProgress.TotalLessons)
		}
	})
}

func TestToggleLessonComplete(t *testing.T) {
	router, _, _, _, lesID := setupLessonTestEnv()

	t.Run("Toggles lesson completion status", func(t *testing.T) {
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/lessons/%d/toggle-complete", lesID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success     bool `json:"success"`
			IsCompleted bool `json:"is_completed"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.IsCompleted {
			t.Errorf("Expected lesson to be completed after first toggle")
		}

		// Toggle again (should uncomplete)
		req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/lessons/%d/toggle-complete", lesID), nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		var resp2 struct {
			IsCompleted bool `json:"is_completed"`
		}
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		if resp2.IsCompleted {
			t.Errorf("Expected lesson to be uncompleted after second toggle")
		}
	})
}

func TestSaveLessonProgress(t *testing.T) {
	router, _, _, _, lesID := setupLessonTestEnv()

	t.Run("Saves playback progress percentage", func(t *testing.T) {
		payload := ProgressPayload{ProgressPercent: 65}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/lessons/%d/progress", lesID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool               `json:"success"`
			Data    UserLessonProgress `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.Data.ProgressPercent != 65 {
			t.Errorf("Expected progress 65, got %d", resp.Data.ProgressPercent)
		}
	})
}

func TestGetRevisionAndPastPapers(t *testing.T) {
	router, _, subID, _, _ := setupLessonTestEnv()
	subIDStr := strconv.Itoa(int(subID))

	t.Run("Fetches revision modules for subject", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/subjects/"+subIDStr+"/revision", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool             `json:"success"`
			Data    []RevisionModule `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data) != 1 {
			t.Fatalf("Expected 1 revision module, got %d", len(resp.Data))
		}
		if resp.Data[0].Title != "Cell Biology Short Notes" {
			t.Errorf("Expected title Cell Biology Short Notes, got %s", resp.Data[0].Title)
		}
	})

	t.Run("Fetches past papers for subject", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/subjects/"+subIDStr+"/past-papers?type=past", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool        `json:"success"`
			Data    []PastPaper `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if len(resp.Data) != 1 {
			t.Fatalf("Expected 1 past paper, got %d", len(resp.Data))
		}
		if resp.Data[0].Year != 2023 {
			t.Errorf("Expected year 2023, got %d", resp.Data[0].Year)
		}
	})
}

func TestGetLessonListOverview(t *testing.T) {
	router, _, subID, _, _ := setupLessonTestEnv()
	subIDStr := strconv.Itoa(int(subID))

	t.Run("Fetches full lesson list overview for subject", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/subjects/"+subIDStr+"/lessonlist", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool              `json:"success"`
			Data    LessonListPageDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}
		if resp.Data.SubjectTitle == "" {
			t.Errorf("Expected subject title to be present")
		}
		if len(resp.Data.Units) == 0 {
			t.Errorf("Expected units list to be present")
		}
		if resp.Data.SubjectProgress.TotalLessons <= 0 {
			t.Errorf("Expected total lessons > 0, got %d", resp.Data.SubjectProgress.TotalLessons)
		}
		if resp.Data.UpcomingMock.Name == "" {
			t.Errorf("Expected upcoming mock name to be present")
		}
	})

	t.Run("Fetches default lesson list overview via /lessonlist", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/lessonlist", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Success bool              `json:"success"`
			Data    LessonListPageDTO `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if !resp.Success {
			t.Errorf("Expected success to be true")
		}
		if len(resp.Data.Units) == 0 {
			t.Errorf("Expected units list to be present")
		}
	})
}

