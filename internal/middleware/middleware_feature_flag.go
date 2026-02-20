package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// FeatureToggle checks an environment variable to see if a route is enabled.
// If the variable is explicitly set to "false", it blocks the request.
func FeatureToggle(envKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the flag from Heroku/local environment
		flagStatus := os.Getenv(envKey)

		// If explicitly disabled, abort the request before it hits the handler
		if flagStatus == "false" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "Quiz creation is temporarily disabled for maintenance.",
			})
			return
		}

		// Otherwise, let the request proceed to the handler
		c.Next()
	}
}