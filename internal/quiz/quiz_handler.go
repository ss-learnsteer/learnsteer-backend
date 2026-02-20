package quiz

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

// NewHandler initializes the handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes automatically sets up the API endpoints for this module.
// This keeps main.go clean.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	routes := router.Group("/quizzes")
	{
		// GET /api/v1/quizzes?page=1&limit=10
		routes.GET("", h.ListQuizzes)

		// GET /api/v1/quizzes/:id/start
		routes.GET("/:id/start", h.StartQuiz)

		// GET /api/v1/quizzes/:id/questions
		routes.GET("/:id/questions", h.GetQuizQuestions)

		// POST /api/v1/quizzes (RESTful creation)
		routes.POST(
			"", 
			middleware.FeatureToggle("ENABLE_QUIZ_CREATION"), 
			h.CreateQuiz,
		)

		routes.PUT(
			"/:id", 
			middleware.FeatureToggle("ENABLE_QUIZ_CREATION"), 
			h.UpdateQuiz,
		)
	}
}

// CreateQuiz handles creating a new quiz with markdown questions
func (h *Handler) CreateQuiz(c *gin.Context) {
	var quiz Quiz

	// 1. Bind JSON to Struct
	if err := c.ShouldBindJSON(&quiz); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload: " + err.Error(),
		})
		return
	}

	// 2. Call Service to save to database
	if err := h.service.Create(&quiz); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create quiz",
		})
		return
	}

	// 3. Return the specific ID alongside a success message
	// GORM automatically populates quiz.ID after a successful insert
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Quiz created successfully",
		"quiz_id": quiz.ID,
	})
}

// ListQuizzes returns a paginated list (Lightweight metadata only)
func (h *Handler) ListQuizzes(c *gin.Context) {
	// 1. Parse Query Params with defaults
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 2. Call Service
	quizzes, total, err := h.service.ListQuizzes(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quizzes"})
		return
	}

	// 3. Return Standard Pagination Response
	c.JSON(http.StatusOK, gin.H{
		"data": quizzes,
		"meta": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// StartQuiz returns the full context for a student to take the quiz
func (h *Handler) StartQuiz(c *gin.Context) {
	// 1. Parse ID
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	// 2. Fetch the "Mega Object" (Quiz + Questions + Options)
	quiz, err := h.service.GetStartQuiz(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// GetQuizQuestions returns ONLY the questions for a specific quiz (NEW HANDLER)
func (h *Handler) GetQuizQuestions(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid quiz ID",
		})
		return
	}

	// Fetch questions using the service layer
	questions, err := h.service.GetQuestionsByQuizID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch questions for this quiz",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"quiz_id": id,
		"data":    questions,
	})
}

// UpdateQuiz handles the full replacement of a quiz's questions
func (h *Handler) UpdateQuiz(c *gin.Context) {
	// 1. Get the Quiz ID from the URL parameter
	idParam := c.Param("id")
	quizID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid quiz ID",
		})
		return
	}

	// 2. Bind the incoming JSON payload (The fully fresh object from React)
	var updatedQuiz Quiz
	if err := c.ShouldBindJSON(&updatedQuiz); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload: " + err.Error(),
		})
		return
	}

	// 3. Execute the Transaction in the Service
	if err := h.service.ReplaceQuizContent(uint(quizID), &updatedQuiz); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update quiz content: " + err.Error(),
		})
		return
	}

	// 4. Return Success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quiz updated successfully",
		"quiz_id": quizID,
	})
}