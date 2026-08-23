package examshub

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the Exams Hub page
type Handler struct {
	service *Service
}

// NewHandler creates a new exams hub handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the exams hub endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/exams/hub", h.GetExamsHub)
	r.GET("/exams/mocks", h.GetMockExams)
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

// GetExamsHub returns the full Exams Hub analytics page payload
func (h *Handler) GetExamsHub(c *gin.Context) {
	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required to view exams hub",
		})
		return
	}

	data, err := h.service.GetExamsHub(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load exams hub: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetMockExams returns the full Mock Exams page payload
func (h *Handler) GetMockExams(c *gin.Context) {
	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required to view mock exams",
		})
		return
	}

	userStream := c.GetString("user_stream")
	userMedium := c.GetString("user_medium")
	if userStream == "" {
		userStream = "Bio Science"
	}
	if userMedium == "" {
		userMedium = "Sinhala"
	}

	data, err := h.service.GetMockExams(userID, userStream, userMedium)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load mock exams: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

