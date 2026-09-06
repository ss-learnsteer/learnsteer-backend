package submission

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

// SubmitAnswerPayload represents a single answer in the submission
type SubmitAnswerPayload struct {
	QuestionID     uint   `json:"question_id" binding:"required"`
	SelectedOption string `json:"selected_option" binding:"required"`
}

// SubmitQuizPayload represents the full quiz submission from the frontend
type SubmitQuizPayload struct {
	QuizID      uint                  `json:"quiz_id" binding:"required"`
	StartedAt   *time.Time            `json:"started_at"`   // When the student opened the quiz
	CompletedAt *time.Time            `json:"completed_at"` // When the student clicked submit
	Answers     []SubmitAnswerPayload `json:"answers" binding:"required"`
}

// NewHandler initializes the handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes sets up the API endpoints for the submission module
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/submissions", h.SubmitQuiz)
	r.GET("/submissions", h.GetMySubmissions)
}

// extractUserID safely extracts the user ID from the JWT context
func extractUserID(c *gin.Context) (uint, bool) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity not found"})
		return 0, false
	}

	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in token"})
		return 0, false
	}
	return userID, true
}

// SubmitQuiz handles grading and saving a quiz submission
func (h *Handler) SubmitQuiz(c *gin.Context) {
	var req SubmitQuizPayload

	// 1. Validate incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid submission format",
		})
		return
	}

	// 2. Get the logged-in student's ID from the JWT Middleware
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	// 3. Grade and Save
	submission, err := h.service.GradeAndSubmit(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 4. Return the calculated score to the frontend!
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Quiz submitted successfully!",
		"data": gin.H{
			"submission_id": submission.ID,
			"score":         submission.Score,
		},
	})
}

// GetMySubmissions returns the authenticated user's submission history
func (h *Handler) GetMySubmissions(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	submissions, err := h.service.GetSubmissionsByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch submission history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    submissions,
	})
}
