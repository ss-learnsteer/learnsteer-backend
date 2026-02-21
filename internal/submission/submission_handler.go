package submission

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

type SubmitAnswerPayload struct {
	QuestionID uint `json:"question_id" binding:"required"`
	SelectedOption string `json:"selected_option" binding:"required"`
}

type SubmitQuizPayload struct {
	QuizID  uint                  `json:"quiz_id" binding:"required"`
	Answers []SubmitAnswerPayload `json:"answers" binding:"required"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// POST /api/v1/submissions
	r.POST("/submissions", h.SubmitQuiz)
}

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
	// In Gin, numbers saved in context often come out as float64 when parsed from JWTs
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity not found"})
		return
	}

	// Safely cast the ID to uint
	var userID uint
	switch v := userIDRaw.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type in token"})
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
