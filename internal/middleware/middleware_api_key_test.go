package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireAPIKey(t *testing.T) {
	// Set the server's expected API Key
	os.Setenv("B2B_API_KEY", "super_secret_test_key")
	defer os.Unsetenv("B2B_API_KEY")

	// Setup a dummy Gin router using the middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireAPIKey())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name           string
		headerKey      string
		headerValue    string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			headerKey:      "X-API-Key",
			headerValue:    "super_secret_test_key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid API Key",
			headerKey:      "X-API-Key",
			headerValue:    "wrong_key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing Header Completely",
			headerKey:      "",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/protected", nil)
			if tc.headerKey != "" {
				req.Header.Set(tc.headerKey, tc.headerValue)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}