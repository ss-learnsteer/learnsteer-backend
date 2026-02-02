package submission

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// POST /api/v1/submissions
	r.POST("/submissions", h.SubmitQuiz)
}

func (h *Handler) SubmitQuiz(c *gin.Context) {
	var submission Submission

	// 1. Bind JSON
	if err := c.ShouldBindJSON(&submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data: " + err.Error()})
		return
	}

	// 2. Get User ID from Context (Set by your AuthMiddleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	// Convert interface{} to uint
	// Note: In JWT claims, numbers often come as float64, check your JWT parser
	// Using a safe cast here assuming float64 from standard JWT parsers
	if idFloat, ok := userID.(float64); ok {
		submission.UserID = uint(idFloat)
	} else if idUint, ok := userID.(uint); ok { // In case your parser uses uint
		submission.UserID = idUint
	}

	// 3. Call Service
	if err := h.service.Submit(&submission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit quiz"})
		return
	}

	// 4. Return the Grade
	c.JSON(http.StatusCreated, gin.H{
		"message": "Quiz submitted successfully",
		"score":   submission.Score,
		"id":      submission.ID,
	})
}