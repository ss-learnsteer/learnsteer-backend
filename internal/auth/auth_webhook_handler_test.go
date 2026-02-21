package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// dbCounter gives each test its own isolated in-memory SQLite database.
var dbCounter uint64

// setupTestEnvironment initializes an in-memory DB and returns the Gin engine.
// Each call gets a brand-new database so tests never share state.
func setupTestEnvironment() (*gin.Engine, *gorm.DB) {
	// Use a unique name per call so there is no shared state between test runs.
	dbName := fmt.Sprintf("file:testdb%d?mode=memory&cache=private", atomic.AddUint64(&dbCounter, 1))
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to in-memory database")
	}
	db.AutoMigrate(&User{})

	service := NewService(db)
	handler := NewHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/webhook", handler.HandleGoogleSheetWebhook)

	return router, db
}

func TestHandleGoogleSheetWebhook(t *testing.T) {
	// Set the environment variable just for this test
	os.Setenv("WEBHOOK_SECRET", "test-secret")
	defer os.Unsetenv("WEBHOOK_SECRET")

	router, db := setupTestEnvironment()

	// ---------------------------------------------------------
	// TEST CASE 1: Successful New User Registration
	// ---------------------------------------------------------
	t.Run("Valid Webhook Payload - Creates New User", func(t *testing.T) {
		// Clean slate before this subtest
		db.Exec("DELETE FROM users")
		payload := WebhookPayload{
			Timestamp:      "2023-10-27T10:00:00Z",
			FirstName:      "Kasun",
			LastName:       "Perera",
			NIC:            "200112345678",
			Email:          "kasun@example.com",
			WhatsappNumber: "0771234567",
			School:         "Royal College",
			District:       "Colombo",
			Stream:         "Physical Science",
			Medium:         "Sinhala",
			ALBatch:        "2025",
			ALAttempt:      "1",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
		req.Header.Set("X-Webhook-Secret", "test-secret")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert HTTP Status
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}

		// Assert Database State
		var user User
		if err := db.Where("email = ?", "kasun@example.com").First(&user).Error; err != nil {
			t.Errorf("Expected user to be created in DB, but got error: %v", err)
		}
		if user.FirstName != "Kasun" {
			t.Errorf("Expected first name 'Kasun', got '%s'", user.FirstName)
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 2: Successful Update (Upsert Logic)
	// ---------------------------------------------------------
	t.Run("Duplicate Email Payload - Updates Existing User", func(t *testing.T) {
		// Ensure Kasun from Test Case 1 is present (but no duplicate NIC rows)
		db.Exec("DELETE FROM users")
		// Kasun changes his district to Gampaha
		payload := WebhookPayload{
			FirstName: "Kasun",
			LastName:  "Perera",
			NIC:       "200112345678",
			Email:     "kasun@example.com",
			District:  "Gampaha", // <--- The changed field
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
		req.Header.Set("X-Webhook-Secret", "test-secret")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}

		// Verify the database was updated, not duplicated
		var count int64
		db.Model(&User{}).Where("email = ?", "kasun@example.com").Count(&count)
		if count != 1 {
			t.Errorf("Expected exactly 1 user with this email, found %d", count)
		}

		var updatedUser User
		db.Where("email = ?", "kasun@example.com").First(&updatedUser)
		if updatedUser.District != "Gampaha" {
			t.Errorf("Expected district to be updated to 'Gampaha', got '%s'", updatedUser.District)
		}
	})

	// ---------------------------------------------------------
	// TEST CASE 3: Unauthorized (Wrong Secret)
	// ---------------------------------------------------------
	t.Run("Unauthorized - Wrong Secret Key", func(t *testing.T) {
		payload := WebhookPayload{Email: "hacker@example.com"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
		req.Header.Set("X-Webhook-Secret", "wrong-secret-123") // Incorrect secret
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 Unauthorized, got %d", w.Code)
		}
	})
}
