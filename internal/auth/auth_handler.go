package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware"
	"gorm.io/gorm"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

// CheckNICRequest matches the incoming JSON payload exactly
type CheckNICRequest struct {
	NIC string `json:"NIC" binding:"required"`
}

type VerifyPasswordRequest struct {
	NIC      string `json:"NIC" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserProfileResponse is a safe DTO (Data Transfer Object) that hides the password hash
type UserProfileResponse struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	NIC            string `json:"nic"`
	WhatsappNumber string `json:"whatsapp_number"`
	School         string `json:"school"`
	District       string `json:"district"`
	Stream         string `json:"stream"`
	Medium         string `json:"medium"`
	ALBatch        string `json:"al_batch"`
	ALAttempt      string `json:"al_attempt"`
	Role           string `json:"role"`
}

type CreateTicketRequest struct {
	NIC string `json:"nic" binding:"required"`
}

type ExchangeTicketRequest struct {
	Ticket string `json:"ticket" binding:"required"`
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
		// authGroup.POST("/webhook/google-sheets", h.HandleGoogleSheetWebhook)
		authGroup.POST("/check-nic", h.CheckNIC)
		authGroup.POST("/verify-password", h.VerifyPassword)
		authGroup.GET("/profile/:nic", h.GetProfile)
		// The React app calls this to trade the ticket for a JWT
		authGroup.POST("/exchange", h.ExchangeSSOTicket)
	}

	// The B2B Server-to-Server Group
	b2bGroup := r.Group("/b2b")
	// Protect this entire group with the API Key middleware
	b2bGroup.Use(middleware.RequireAPIKey())
	{
		// The Node.js server calls this
		b2bGroup.POST("/tickets", h.CreateB2BTicket)
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
	Stream         string `json:"stream" binding:"required"`          // e.g., "Physical Science"
	District       string `json:"district" binding:"required"`        // e.g., "Gampaha"
	School         string `json:"school" binding:"required"`          // e.g., "Royal College"
	NIC            string `json:"nic" binding:"required"`             // National Identity Card
	WhatsappNumber string `json:"whatsapp_number" binding:"required"` // Mobile Whatsapp No.
	ALBatch        string `json:"al_batch" binding:"required"`        // e.g., "2025"
	ALAttempt      string `json:"al_attempt" binding:"required"`      // "1", "2", or "3"
	Medium         string `json:"medium" binding:"required"`          // "Sinhala", "Tamil", "English"
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
		Email:          req.Email,
		Password:       req.Password,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Stream:         req.Stream,
		District:       req.District,
		School:         req.School,
		NIC:            req.NIC,
		WhatsappNumber: req.WhatsappNumber,
		ALBatch:        req.ALBatch,
		ALAttempt:      req.ALAttempt,
		Medium:         req.Medium,
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

// CheckNIC responds to the frontend with success and exists boolean
func (h *Handler) CheckNIC(c *gin.Context) {
	var req CheckNICRequest

	// 1. Bind the incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		// Return 400 Bad Request if the JSON is malformed or NIC is missing
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Valid NIC is required in the payload",
		})
		return
	}

	// Fetch the user
	user, err := h.service.GetUserByNIC(req.NIC)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// NIC does not exist
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"exists":  false,
			})
			return
		}
		// Some other database error
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Internal server error",
		})
		return
	}

	// NIC exists. Construct the response.
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"exists":        true,
		"has_password":  user.PasswordHash != "",
	})
}

// VerifyPassword responds to the microservice indicating if the credentials are valid
func (h *Handler) VerifyPassword(c *gin.Context) {
	var req VerifyPasswordRequest

	// 1. Bind the JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Both NIC and password are required in the payload",
		})
		return
	}

	// 2. Check the credentials via the service
	isValid, err := h.service.VerifyPasswordByNIC(req.NIC, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Internal server error while verifying credentials",
		})
		return
	}

	// 3. Return the exact verification status
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"is_valid": isValid,
	})
}

// GetProfile fetches a user's safe profile data by their NIC
func (h *Handler) GetProfile(c *gin.Context) {
	// 1. Extract the NIC from the URL path parameter (e.g., /profile/200112345678)
	nic := c.Param("nic")
	if nic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "NIC parameter is required",
		})
		return
	}

	// 2. Fetch the user using your existing service method
	user, err := h.service.GetUserByNIC(nic)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "User profile not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Internal server error",
		})
		return
	}

	// 3. Map the database model to our safe DTO
	safeProfile := UserProfileResponse{
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Email:          user.Email,
		NIC:            user.NIC,
		WhatsappNumber: user.WhatsappNumber,
		School:         user.School,
		District:       user.District,
		Stream:         user.Stream,
		Medium:         user.Medium,
		ALBatch:        user.ALBatch,
		ALAttempt:      user.ALAttempt,
		Role:           user.Role,
	}

	// 4. Return the secure data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    safeProfile,
	})
}

// CreateB2BTicket is called by the external Node.js server
func (h *Handler) CreateB2BTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NIC is required"})
		return
	}

	// In a real B2B system, you might also verify the NIC exists here first
	ticket, err := h.service.GenerateSSOTicket(req.NIC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate ticket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"ticket":  ticket,
		"expires": 60, // Tell the client they have 60 seconds to use it
	})
}

// ExchangeSSOTicket is called by your React frontend
func (h *Handler) ExchangeSSOTicket(c *gin.Context) {
	var req ExchangeTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket is required"})
		return
	}

	// 1. Consume the ticket
	user, err := h.service.ConsumeSSOTicket(req.Ticket)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 2. Generate the real Go JWT using your existing method!
	token, err := h.service.generateJWT(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}

	// 3. Return the token and safe user data to React
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
		"user": gin.H{
			"nic":  user.NIC,
			"role": user.Role,
		},
	})
}