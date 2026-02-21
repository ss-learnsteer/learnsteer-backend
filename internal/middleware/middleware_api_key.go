package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// RequireAPIKey blocks requests that don't have a valid X-API-Key header
func RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get the key from the incoming request header
		clientKey := c.GetHeader("X-API-Key")
		
		// 2. Get your master B2B key from Heroku/environment
		serverKey := os.Getenv("B2B_API_KEY")

		// 3. Validate
		if clientKey == "" || clientKey != serverKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or missing B2B API Key",
			})
			return
		}

		c.Next()
	}
}