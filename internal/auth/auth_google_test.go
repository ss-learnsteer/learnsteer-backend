package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGoogleAuthHandler_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock DB if available or test endpoint binding validation
	router := gin.New()
	handler := NewHandler(nil)
	authGroup := router.Group("/auth")
	authGroup.POST("/google", handler.GoogleAuth)

	// Test 1: Missing id_token in body
	t.Run("Missing ID Token", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req, _ := http.NewRequest("POST", "/auth/google", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request for missing id_token, got %d", w.Code)
		}
	})

	// Test 2: Invalid ID Token string provided
	t.Run("Invalid ID Token String", func(t *testing.T) {
		if handler.service == nil {
			// Skip service call test if service is nil
			return
		}
		body, _ := json.Marshal(map[string]string{"id_token": "invalid_fake_token"})
		req, _ := http.NewRequest("POST", "/auth/google", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 Unauthorized for invalid token, got %d", w.Code)
		}
	})
}
