package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/middleware" // Update to your module path
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupSSOTestEnv initializes the DB, seeds a user, and wires the router
func setupSSOTestEnv() (*gin.Engine, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open("test_sso.db"), &gorm.Config{})
	if err == nil && db != nil {
		db.AutoMigrate(&User{}, &SSOTicket{})
		db.Create(&User{
			NIC:          "200112345678",
			Role:         "student",
			PasswordHash: "dummy_hash",
		})
	}

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Apply JWT Secret for the final token generation
	os.Setenv("JWT_SECRET", "test_jwt_secret")

	// Map the routes exactly as they are in production
	b2bGroup := router.Group("/api/v1/b2b")
	b2bGroup.Use(middleware.RequireAPIKey())
	{
		b2bGroup.POST("/tickets", handler.CreateB2BTicket)
	}

	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/exchange", handler.ExchangeSSOTicket)
	}

	return router, db
}

func TestSSOTicketFlow(t *testing.T) {
	router, db := setupSSOTestEnv()
	
	// Set the B2B Key
	os.Setenv("B2B_API_KEY", "b2b_test_key")
	defer os.Unsetenv("B2B_API_KEY")
	defer os.Unsetenv("JWT_SECRET")

	var generatedTicket string

	// ---------------------------------------------------------
	// PART 1: Node.js Server generates a ticket
	// ---------------------------------------------------------
	t.Run("Generate Ticket via /b2b/tickets", func(t *testing.T) {
		payload := map[string]string{"nic": "200112345678"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", "/api/v1/b2b/tickets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "b2b_test_key") // Valid key

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Extract the ticket from the response
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		ticketRaw, ok := response["ticket"].(string)
		if !ok || ticketRaw == "" {
			t.Fatalf("Expected a ticket string in the response")
		}
		generatedTicket = ticketRaw

		// Assert Database State: The ticket must exist
		var ticketCount int64
		db.Model(&SSOTicket{}).Where("ticket = ?", generatedTicket).Count(&ticketCount)
		if ticketCount != 1 {
			t.Errorf("Expected 1 ticket in database, found %d", ticketCount)
		}
	})

	// ---------------------------------------------------------
	// PART 2: React Frontend consumes the ticket
	// ---------------------------------------------------------
	t.Run("Exchange Ticket via /auth/exchange", func(t *testing.T) {
		payload := map[string]string{"ticket": generatedTicket}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", "/api/v1/auth/exchange", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Assert Response: Should contain a JWT
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if _, ok := response["token"].(string); !ok {
			t.Errorf("Expected a JWT token in the response")
		}

		// SECURITY ASSERTION: The ticket MUST be deleted from the database
		var ticketCount int64
		db.Model(&SSOTicket{}).Where("ticket = ?", generatedTicket).Count(&ticketCount)
		if ticketCount != 0 {
			t.Errorf("CRITICAL SECURITY FAILURE: Ticket was not deleted after use!")
		}
	})

	// ---------------------------------------------------------
	// PART 3: React Frontend tries to replay the used ticket
	// ---------------------------------------------------------
	t.Run("Prevent Replay Attack", func(t *testing.T) {
		payload := map[string]string{"ticket": generatedTicket}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", "/api/v1/auth/exchange", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for reused ticket, got %d", w.Code)
		}
	})
}