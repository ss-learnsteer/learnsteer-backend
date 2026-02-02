package quiz

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

		// GET /api/v1/quizzes/:id/start (The "Mega-Fetch")
		routes.GET("/:id/start", h.StartQuiz)

		// POST /api/v1/quizzes (Admin Create)
		routes.POST("", h.CreateQuiz)
	}
}

// CreateQuiz handles creating a new quiz with markdown questions
func (h *Handler) CreateQuiz(c *gin.Context) {
	var quiz Quiz

	// 1. Bind JSON to Struct
	// GORM + Gin will automatically map the nested Questions & Options JSON
	if err := c.ShouldBindJSON(&quiz); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// 2. Call Service
	if err := h.service.Create(&quiz); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create quiz"})
		return
	}

	c.JSON(http.StatusCreated, quiz)
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
