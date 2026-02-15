package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

// NewHandler initializes the auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes sets up the API endpoints for the auth module
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)

		// New Webhook Endpoint
		authGroup.POST("/webhook/google-sheets", h.HandleGoogleSheetWebhook)
	}
}

// RegisterRequest defines the JSON payload for user registration
type RegisterRequest struct {
	// Identity & Security
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=6"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`

	// New Demographic Data (Required for Impact Analytics)
	ExamYear int    `json:"exam_year" binding:"required"` // e.g., 2025
	Stream   string `json:"stream" binding:"required"`    // e.g., "Physical Science"
	District string `json:"district" binding:"required"`  // e.g., "Gampaha"
	School   string `json:"school" binding:"required"`    // e.g., "Royal College"
}

// LoginRequest defines the JSON payload for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register handles creating a new student account
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	// 1. Validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Map Request JSON to Service DTO
	// This separates the API layer from the Domain layer
	dto := RegisterDTO{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		ExamYear:  req.ExamYear,
		Stream:    req.Stream,
		District:  req.District,
		School:    req.School,
	}

	// 3. Call Service Logic
	if err := h.service.Register(dto); err != nil {
		// In a real app, check if error is "email exists" vs "db error" for better status codes
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// Login handles user authentication and JWT generation
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	// 1. Validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Call Service to get Token
	token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		// We return 401 Unauthorized for login failures
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// 3. Return Token
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"type":  "Bearer",
	})
}