package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the student dashboard
type Handler struct {
	service *Service
}

// NewHandler initializes a new dashboard handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the dashboard endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/dashboard", h.GetDashboard)
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

// GetDashboard returns the complete student dashboard payload
func (h *Handler) GetDashboard(c *gin.Context) {
	userID := extractUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required to view dashboard",
		})
		return
	}

	data, err := h.service.GetDashboard(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to load dashboard: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}
