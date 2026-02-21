package quiz

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware"
)

// Handler holds the service dependency
type Handler struct {
	service *Service
}

type VisibilityPayload struct {
	IsVisible *bool `json:"is_visible" binding:"required"`
}

type OptionPayload struct {
	Text string `json:"text" binding:"required"`
}

type QuestionPayload struct {
	Type          string          `json:"type" binding:"required"`
	Points        int             `json:"points"`
	TextMarkdown  string          `json:"text_markdown" binding:"required"`
	ImageURL      string          `json:"image_url"`
	Options       []OptionPayload `json:"options" binding:"required,min=2"`
	CorrectAnswer string          `json:"correct_answer" binding:"required,len=1"` // "a", "b", "c", etc.
}

type CreateQuizRequest struct {
	Title       string            `json:"title" binding:"required"`
	Description string            `json:"description"`
	DurationMin int               `json:"duration_min"`
	Medium      string            `json:"medium" binding:"required"`
	IsVisible   *bool             `json:"is_visible"`
	Questions   []QuestionPayload `json:"questions" binding:"required,min=1"`
}

// NewHandler initializes the handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes automatically sets up the API endpoints for this module.
// This keeps main.go clean.
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	routes := router.Group("/quizzes")
	routes.Use(middleware.RequireAuth())
	{
		routes.GET("", h.ListQuizzes)
		routes.GET("/:id/start", h.StartQuiz)
		routes.GET("/:id/questions", h.GetQuizQuestions)
		routes.POST(
			"",
			middleware.FeatureToggle("ENABLE_QUIZ_CREATION"),
			h.CreateQuiz,
		)
		routes.PUT(
			"/:id",
			middleware.FeatureToggle("ENABLE_QUIZ_CREATION"),
			h.UpdateQuiz,
		)
		routes.PATCH("/:id/visibility", h.ToggleVisibility)
		routes.DELETE("/:id", h.DeleteQuiz)
	}
}

// CreateQuiz handles creating a new quiz with markdown questions
func (h *Handler) CreateQuiz(c *gin.Context) {
	var req CreateQuizRequest

	// 1. Bind and validate the JSON payload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload format",
		})
		return
	}

	// 2. Map the DTO string values to the GORM struct booleans
	quizModel := mapPayloadToModel(req)

	// 3. Save to the database
	if err := h.service.CreateQuiz(&quizModel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create quiz in the database",
		})
		return
	}

	// 4. Return the created model (which now includes the generated DB IDs)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Quiz created successfully",
		"data":    quizModel,
	})
}

// ListQuizzes returns a paginated list (Lightweight metadata only)
func (h *Handler) ListQuizzes(c *gin.Context) {
	// 1. Parse Query Params with defaults
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// 1. Extract the user's identity from the context (set by the Auth Middleware)
	userRole := c.GetString("user_role")
	userMedium := c.GetString("user_medium")

	var filterMedium string
	var onlyVisible bool

	// 2. Apply Role-Based Filtering Logic
	if userRole == "student" {
		// Force the filter to match the student's database medium
		filterMedium = userMedium
		onlyVisible = true // Students NEVER see hidden quizzes
	} else {
		// If they are an admin/staff, let them see everything OR use the query param manually
		filterMedium = c.Query("medium")
		// Admins see everything by default, but can optionally filter by visibility
		onlyVisible = c.Query("visible_only") == "true"
	}

	// 2. Call Service with the new filter
	quizzes, total, err := h.service.ListQuizzes(page, limit, filterMedium, onlyVisible)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quizzes"})
		return
	}

	// 2. CHECK FOR EMPTY RESULTS
	if len(quizzes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Take a breather! 🧘‍♂️ There are no mock exams live for your medium right now. Relax, review your notes, and check back soon!",
			// You can optionally return the meta so the frontend knows what was searched
			"meta": gin.H{
				"medium": filterMedium,
			},
		})
		return
	}

	// 3. Return Standard Pagination Response
	c.JSON(http.StatusOK, gin.H{
		"data": quizzes,
		"meta": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"medium": filterMedium,
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found. DB Error: " + err.Error()})
		return
	}

	// 🔒 SECURITY SCRUB: Remove the correct answers before sending to the student
	for i := range quiz.Questions {
		quiz.Questions[i].CorrectAnswer = "" // Hide the regex/text answer

		// If you use the IsCorrect boolean on Options, hide that too!
		for j := range quiz.Questions[i].Options {
			quiz.Questions[i].Options[j].IsCorrect = false
		}
	}

	c.JSON(http.StatusOK, quiz)
}

// GetQuizQuestions returns ONLY the questions for a specific quiz (NEW HANDLER)
func (h *Handler) GetQuizQuestions(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid quiz ID",
		})
		return
	}

	// Fetch questions using the service layer
	questions, err := h.service.GetQuestionsByQuizID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch questions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"quiz_id": id,
		"data":    questions,
	})
}

// UpdateQuiz handles the full replacement of a quiz's questions
func (h *Handler) UpdateQuiz(c *gin.Context) {
	// 1. Extract the Quiz ID from the URL
	idParam := c.Param("id")
	quizID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid quiz ID",
		})
		return
	}

	var req CreateQuizRequest

	// 2. Bind and validate the incoming JSON replacement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request payload format",
		})
		return
	}

	// 3. Map the DTO to the GORM model
	quizModel := mapPayloadToModel(req)

	// IMPORTANT: Attach the ID from the URL so GORM knows exactly which parent record to update
	quizModel.ID = uint(quizID)

	// 4. Run the update service (which should wipe old questions/options and insert these new ones)
	if err := h.service.UpdateQuiz(&quizModel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update quiz",
		})
		return
	}

	// 5. Return success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quiz updated successfully",
		"data":    quizModel,
	})
}

// ToggleVisibility allows admins to quickly hide or publish a quiz
func (h *Handler) ToggleVisibility(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	var payload VisibilityPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_visible boolean is required"})
		return
	}

	if err := h.service.UpdateVisibility(uint(id), *payload.IsVisible); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update visibility"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quiz visibility updated successfully",
	})
}

// DeleteQuiz handles the DELETE /quizzes/:id endpoint
func (h *Handler) DeleteQuiz(c *gin.Context) {
	// 1. Extract and validate the ID
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid quiz ID",
		})
		return
	}

	// 2. Optional: Check user role here if you want strictly admins to delete
	userRole := c.GetString("user_role")
	if userRole == "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Students are not allowed to delete quizzes",
		})
		return
	}

	// 3. Call the service
	if err := h.service.DeleteQuiz(uint(id)); err != nil {
		if err.Error() == "quiz not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete quiz",
		})
		return
	}

	// 4. Return success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quiz deleted successfully",
	})
}

func mapPayloadToModel(req CreateQuizRequest) Quiz {
	quiz := Quiz{
		Title:       req.Title,
		Description: req.Description,
		Medium:      req.Medium,
		IsVisible:   req.IsVisible,
		DurationMin: req.DurationMin,
	}

	for _, qReq := range req.Questions {
		question := Question{
			Type:          QuestionType(qReq.Type),
			Points:        qReq.Points,
			TextMarkdown:  qReq.TextMarkdown,
			ImageURL:      qReq.ImageURL,
			CorrectAnswer: strings.ToLower(strings.TrimSpace(qReq.CorrectAnswer)), // Save "a", "b", "c"...
		}

		// Convert the letter ("a", "b", "c"...) to lowercase, grab the first character,
		// and subtract 'a' to get the numeric index (0, 1, 2...)
		correctLetter := strings.ToLower(qReq.CorrectAnswer)[0]
		correctIndex := int(correctLetter - 'a')

		for i, optReq := range qReq.Options {
			option := Option{
				Text: optReq.Text,
				// If the loop index matches our calculated correctIndex, set to true!
				IsCorrect: (i == correctIndex),
			}
			question.Options = append(question.Options, option)
		}

		quiz.Questions = append(quiz.Questions, question)
	}

	return quiz
}
