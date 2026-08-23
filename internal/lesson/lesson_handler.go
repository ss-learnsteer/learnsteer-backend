package lesson

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler holds the service dependency for lessons
type Handler struct {
	service *Service
}

// NewHandler initializes a new lesson handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all lessons, subjects, units, revision, and past papers routes
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Subjects & Units
	r.GET("/subjects", h.ListSubjects)
	r.GET("/subjects/:id/units", h.GetSubjectUnits)
	r.GET("/subjects/:id/lessonlist", h.GetLessonListOverview)
	r.GET("/lessonlist", h.GetDefaultLessonListOverview)
	r.POST("/subjects", h.CreateSubject)
	r.POST("/units", h.CreateUnit)

	// Lessons
	r.GET("/lessons/:id", h.GetLessonDetail)
	r.POST("/lessons", h.CreateLesson)
	r.POST("/lessons/:id/toggle-complete", h.ToggleLessonComplete)
	r.POST("/lessons/:id/progress", h.SaveLessonProgress)
	r.POST("/lessons/:id/questions", h.PostLessonQuestion)
	r.POST("/lessons/:id/questions/:questionId/helpful", h.LikeLessonQuestion)

	// Revision Modules
	r.GET("/subjects/:id/revision", h.GetRevisionModules)
	r.POST("/revision-modules", h.CreateRevisionModule)

	// Past Papers
	r.GET("/subjects/:id/past-papers", h.GetPastPapers)
	r.POST("/past-papers", h.CreatePastPaper)
}

func extractUserID(c *gin.Context) uint {
	if uid, exists := c.Get("userID"); exists {
		switch v := uid.(type) {
		case float64:
			return uint(v)
		case uint:
			return v
		}
	}
	if uid, exists := c.Get("user_id"); exists {
		switch v := uid.(type) {
		case float64:
			return uint(v)
		case uint:
			return v
		}
	}
	return 0
}

// ListSubjects returns all subjects available for student's stream and medium,
// wrapped with stream context and island rank for the Subjects page
func (h *Handler) ListSubjects(c *gin.Context) {
	userRole := c.GetString("user_role")
	userMedium := c.GetString("user_medium")
	userStream := c.GetString("user_stream")
	userID := extractUserID(c)

	var stream string
	var medium string

	switch userRole {
	case "student":
		stream = userStream
		medium = userMedium
	case "ss_member", "admin":
		stream = c.Query("stream")
		medium = c.Query("medium")
	default:
		stream = userStream
		medium = userMedium
	}

	subjects, err := h.service.ListSubjects(stream, medium, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch subjects: " + err.Error(),
		})
		return
	}

	// Calculate Island Rank for the student's stream
	islandRank := h.service.GetStreamIslandRank(userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"stream":      stream,
			"island_rank": islandRank,
			"subjects":    subjects,
		},
	})
}

// GetSubjectUnits returns the syllabus units and lessons for a subject
func (h *Handler) GetSubjectUnits(c *gin.Context) {
	idParam := c.Param("id")
	subjectID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subject ID",
		})
		return
	}

	userID := extractUserID(c)
	units, err := h.service.GetSubjectUnits(uint(subjectID), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch units: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"subject_id": subjectID,
		"data":       units,
	})
}

// GetLessonListOverview returns full syllabus units, lessons, and sidebar widgets for /lessonlist
func (h *Handler) GetLessonListOverview(c *gin.Context) {
	idParam := c.Param("id")
	subjectID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subject ID",
		})
		return
	}

	userID := extractUserID(c)
	userStream := c.GetString("user_stream")
	userMedium := c.GetString("user_medium")
	if userStream == "" {
		userStream = "Bio Science"
	}
	if userMedium == "" {
		userMedium = "Sinhala"
	}

	data, err := h.service.GetLessonListOverview(uint(subjectID), userID, userStream, userMedium)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch lesson list: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetDefaultLessonListOverview returns the lesson list overview for the student's primary stream subject
func (h *Handler) GetDefaultLessonListOverview(c *gin.Context) {
	userID := extractUserID(c)
	userStream := c.GetString("user_stream")
	userMedium := c.GetString("user_medium")
	if userStream == "" {
		userStream = "Bio Science"
	}
	if userMedium == "" {
		userMedium = "Sinhala"
	}

	// Optional subject query parameter
	subjectID := 0
	if subParam := c.Query("subject_id"); subParam != "" {
		subjectID, _ = strconv.Atoi(subParam)
	}

	data, err := h.service.GetLessonListOverview(uint(subjectID), userID, userStream, userMedium)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch lesson list: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetLessonDetail returns complete data to render the /lessonview player
func (h *Handler) GetLessonDetail(c *gin.Context) {
	idParam := c.Param("id")
	lessonID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid lesson ID",
		})
		return
	}

	userID := extractUserID(c)
	detail, err := h.service.GetLessonDetail(uint(lessonID), userID)
	if err != nil {
		if err.Error() == "lesson not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Lesson not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch lesson detail: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// ToggleLessonComplete marks a lesson completed/uncompleted
func (h *Handler) ToggleLessonComplete(c *gin.Context) {
	idParam := c.Param("id")
	lessonID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid lesson ID",
		})
		return
	}

	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	progress, err := h.service.ToggleLessonComplete(userID, uint(lessonID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to toggle lesson completion: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"is_completed": progress.IsCompleted,
		"data":         progress,
	})
}

type ProgressPayload struct {
	ProgressPercent int `json:"progress_percent" binding:"required,min=0,max=100"`
}

// SaveLessonProgress records video playback progress percentage
func (h *Handler) SaveLessonProgress(c *gin.Context) {
	idParam := c.Param("id")
	lessonID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid lesson ID",
		})
		return
	}

	var req ProgressPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid payload. progress_percent must be between 0 and 100",
		})
		return
	}

	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required",
		})
		return
	}

	progress, err := h.service.SaveLessonProgress(userID, uint(lessonID), req.ProgressPercent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save progress: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    progress,
	})
}

// GetRevisionModules returns the revision notes & guides for a subject
func (h *Handler) GetRevisionModules(c *gin.Context) {
	idParam := c.Param("id")
	subjectID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subject ID",
		})
		return
	}

	modules, err := h.service.GetRevisionModules(uint(subjectID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch revision modules: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"subject_id": subjectID,
		"data":       modules,
	})
}

// GetPastPapers returns past papers and model papers for a subject
func (h *Handler) GetPastPapers(c *gin.Context) {
	idParam := c.Param("id")
	subjectID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid subject ID",
		})
		return
	}

	paperType := c.Query("type") // "past", "model", or ""
	year, _ := strconv.Atoi(c.Query("year"))

	papers, err := h.service.GetPastPapers(uint(subjectID), paperType, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch past papers: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"subject_id": subjectID,
		"data":       papers,
	})
}

// Admin Endpoints

func (h *Handler) CreateSubject(c *gin.Context) {
	var req Subject
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subject payload: " + err.Error()})
		return
	}
	if err := h.service.CreateSubject(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subject: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func (h *Handler) CreateUnit(c *gin.Context) {
	var req Unit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit payload: " + err.Error()})
		return
	}
	if err := h.service.CreateUnit(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create unit: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func (h *Handler) CreateLesson(c *gin.Context) {
	var req Lesson
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lesson payload: " + err.Error()})
		return
	}
	if err := h.service.CreateLesson(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lesson: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func (h *Handler) CreateRevisionModule(c *gin.Context) {
	var req RevisionModule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid revision module payload: " + err.Error()})
		return
	}
	if err := h.service.CreateRevisionModule(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create revision module: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

func (h *Handler) CreatePastPaper(c *gin.Context) {
	var req PastPaper
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid past paper payload: " + err.Error()})
		return
	}
	if err := h.service.CreatePastPaper(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create past paper: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// PostLessonQuestion handles posting a new student question in the lesson Q&A thread
func (h *Handler) PostLessonQuestion(c *gin.Context) {
	idParam := c.Param("id")
	lessonID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid lesson ID"})
		return
	}

	userID := extractUserID(c)
	var req PostQuestionRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Question text is required"})
		return
	}

	qa, err := h.service.PostLessonQuestion(uint(lessonID), userID, req.Question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to post question: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": qa})
}

// LikeLessonQuestion handles upvoting / marking a Q&A question as helpful
func (h *Handler) LikeLessonQuestion(c *gin.Context) {
	idParam := c.Param("id")
	lessonID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid lesson ID"})
		return
	}

	qParam := c.Param("questionId")
	questionID, err := strconv.Atoi(qParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid question ID"})
		return
	}

	helpfulCount, err := h.service.LikeLessonQuestion(uint(lessonID), uint(questionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update helpful count: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"helpful_count": helpfulCount,
		"helpful_text":  fmt.Sprintf("%d Helpful", helpfulCount),
	})
}
