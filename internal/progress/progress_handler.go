package progress

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the Progress & Achievements Dashboard
type Handler struct {
	service *Service
}

// NewHandler initializes a new progress handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the progress endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/progress", h.GetProgress)
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

// GetProgress returns the complete Progress & Achievements dashboard payload
func (h *Handler) GetProgress(c *gin.Context) {
	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required to view progress",
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

	data, err := h.service.GetProgress(userID, userStream, userMedium)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load progress: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
