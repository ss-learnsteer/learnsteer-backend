package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware"
	"gorm.io/gorm"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

// CreateStudentRequest — Collect essential data to get a student learning
type CreateStudentRequest struct {
	FirstName      string `json:"first_name" binding:"required"`
	LastName       string `json:"last_name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	NIC            string `json:"nic" binding:"required"`
	WhatsappNumber string `json:"whatsapp_number" binding:"required"`
	Stream         string `json:"stream" binding:"required"`  // Bio Science, Physical Science, Commerce, Technology, Arts
	Medium         string `json:"medium" binding:"required"`  // Sinhala, Tamil, English
}

// UpdateStudentRequest — Profile completion, collected post-login (all optional)
type UpdateStudentRequest struct {
	Phone         string `json:"phone"`
	District      string `json:"district"`
	School        string `json:"school"`
	City          string `json:"city"`
	ALYear        string `json:"al_year"`
	ALAttempt     string `json:"al_attempt"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender"`
	GuardianPhone string `json:"guardian_phone"`
}

// LoginRequest accepts Nickname, Email, or NIC as the identifier
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // Nickname, Email, or NIC
	Password   string `json:"password" binding:"required"`
}

// CheckNICRequest matches the incoming JSON payload exactly
type CheckNICRequest struct {
	NIC string `json:"NIC" binding:"required"`
}

type VerifyPasswordRequest struct {
	NIC      string `json:"NIC" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserProfileResponse is a safe DTO that hides internal fields
type UserProfileResponse struct {
	StudentID      string `json:"student_id"`
	Nickname       string `json:"nickname"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	NIC            string `json:"nic"`
	WhatsappNumber string `json:"whatsapp_number"`
	Phone          string `json:"phone"`
	School         string `json:"school"`
	District       string `json:"district"`
	City           string `json:"city"`
	Stream         string `json:"stream"`
	Medium         string `json:"medium"`
	ALYear         string `json:"al_year"`
	ALAttempt      string `json:"al_attempt"`
	DateOfBirth    string `json:"date_of_birth"`
	Gender         string `json:"gender"`
	GuardianPhone  string `json:"guardian_phone"`
	Role           string `json:"role"`
}

type CreateTicketRequest struct {
	NIC       string `json:"nic"`
	StudentID string `json:"student_id"`
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
		// Public routes
		authGroup.POST("/create-student", h.CreateStudent)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/webhook/google-sheets", h.HandleGoogleSheetWebhook)
		authGroup.POST("/check-nic", h.CheckNIC)
		authGroup.POST("/verify-password", h.VerifyPassword)
		authGroup.GET("/profile/:student_id", h.GetProfile)
		// The React app calls this to trade the ticket for a JWT
		authGroup.POST("/exchange", h.ExchangeSSOTicket)

		// Protected routes (require JWT)
		protectedAuth := authGroup.Group("/")
		protectedAuth.Use(middleware.RequireAuth())
		{
			protectedAuth.PUT("/update-student", h.UpdateStudent)
		}
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

// ---------------------------------------------------------------------------
// Create Student (replaces Register)
// ---------------------------------------------------------------------------

// CreateStudent handles creating a new student account
func (h *Handler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest

	// 1. Validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Map Request JSON to Service DTO
	dto := CreateStudentDTO{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		Password:       req.Password,
		NIC:            req.NIC,
		WhatsappNumber: req.WhatsappNumber,
		Stream:         req.Stream,
		Medium:         req.Medium,
	}

	// 3. Call Service Logic
	user, err := h.service.CreateStudent(dto)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Auto-login: Generate JWT so the student can start immediately
	token, err := h.service.generateJWT(*user)
	if err != nil {
		// User was created but token generation failed — still a success
		c.JSON(http.StatusCreated, gin.H{
			"success":    true,
			"message":    fmt.Sprintf("Welcome to LearnSteer, %s!", user.Nickname),
			"student_id": user.StudentID,
			"nickname":   user.Nickname,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"message":    fmt.Sprintf("Welcome to LearnSteer, %s!", user.Nickname),
		"student_id": user.StudentID,
		"nickname":   user.Nickname,
		"token":      token,
		"type":       "Bearer",
	})
}

// ---------------------------------------------------------------------------
// Update Student (profile completion)
// ---------------------------------------------------------------------------

// UpdateStudent handles updating a student's profile data post-login
func (h *Handler) UpdateStudent(c *gin.Context) {
	var req UpdateStudentRequest

	// 1. Validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Extract user ID from JWT context
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Convert user_id from JWT claims (float64 from JSON) to uint
	var userID uint
	switch v := uid.(type) {
	case float64:
		userID = uint(v)
	case uint:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user identity"})
		return
	}

	// 3. Map to DTO
	dto := UpdateStudentDTO{
		Phone:         req.Phone,
		District:      req.District,
		School:        req.School,
		City:          req.City,
		ALYear:        req.ALYear,
		ALAttempt:     req.ALAttempt,
		DateOfBirth:   req.DateOfBirth,
		Gender:        req.Gender,
		GuardianPhone: req.GuardianPhone,
	}

	// 4. Call Service Logic
	if err := h.service.UpdateStudent(userID, dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile updated successfully",
	})
}

// ---------------------------------------------------------------------------
// Login (accepts Nickname, Email, or NIC)
// ---------------------------------------------------------------------------

// Login handles user authentication and JWT generation
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	// 1. Validate JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 2. Call Service to get Token (accepts nickname, email, or NIC)
	token, user, err := h.service.Login(req.Identifier, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 3. Return Token and safe user info
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"type":  "Bearer",
		"user": gin.H{
			"student_id": user.StudentID,
			"nickname":   user.Nickname,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email,
			"role":       user.Role,
			"medium":     user.Medium,
			"stream":     user.Stream,
		},
	})
}

// ---------------------------------------------------------------------------
// Check NIC
// ---------------------------------------------------------------------------

// CheckNIC responds to the frontend with success and exists boolean
func (h *Handler) CheckNIC(c *gin.Context) {
	var req CheckNICRequest

	// 1. Bind the incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
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
		"success":      true,
		"exists":       true,
		"student_id":   user.StudentID,
		"nickname":     user.Nickname,
		"has_password": user.PasswordHash != "",
	})
}

// ---------------------------------------------------------------------------
// Verify Password
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Get Profile (by Student ID)
// ---------------------------------------------------------------------------

// GetProfile fetches a user's safe profile data by their Student ID
func (h *Handler) GetProfile(c *gin.Context) {
	// 1. Extract the student_id from the URL path parameter
	studentID := c.Param("student_id")
	if studentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Student ID parameter is required",
		})
		return
	}

	// 2. Fetch the user using the student ID
	user, err := h.service.GetUserByStudentID(studentID)
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
		StudentID:      user.StudentID,
		Nickname:       user.Nickname,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Email:          user.Email,
		NIC:            user.NIC,
		WhatsappNumber: user.WhatsappNumber,
		Phone:          user.Phone,
		School:         user.School,
		District:       user.District,
		City:           user.City,
		Stream:         user.Stream,
		Medium:         user.Medium,
		ALYear:         user.ALYear,
		ALAttempt:      user.ALAttempt,
		DateOfBirth:    user.DateOfBirth,
		Gender:         user.Gender,
		GuardianPhone:  user.GuardianPhone,
		Role:           user.Role,
	}

	// 4. Return the secure data
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    safeProfile,
	})
}

// ---------------------------------------------------------------------------
// B2B SSO Tickets
// ---------------------------------------------------------------------------

// CreateB2BTicket is called by external servers (e.g. Node.js backend)
func (h *Handler) CreateB2BTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.NIC == "" && req.StudentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either nic or student_id is required in the payload"})
		return
	}

	var user *User
	var err error
	if req.StudentID != "" {
		user, err = h.service.GetUserByStudentID(req.StudentID)
	} else {
		user, err = h.service.GetUserByNIC(req.NIC)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	ticket, err := h.service.GenerateSSOTicket(user.StudentID)
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

	// 2. Generate the real Go JWT
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
			"student_id": user.StudentID,
			"nickname":   user.Nickname,
			"role":       user.Role,
		},
	})
}