package auth

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// WebhookPayload matches the JSON sent by Google Apps Script
type WebhookPayload struct {
	Timestamp      string `json:"timestamp"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	NIC            string `json:"nic"`
	Email          string `json:"email"`
	WhatsappNumber string `json:"whatsapp_number"`
	School         string `json:"school"`
	District       string `json:"district"`
	Stream         string `json:"stream"`
	Medium         string `json:"medium"`

	// Updated to String
	ALBatch   string `json:"al_batch"`
	ALAttempt string `json:"al_attempt"`
	Role      string `json:"role"`
}

func (h *Handler) HandleGoogleSheetWebhook(c *gin.Context) {
	// 1. Security: Verify Secret Key
	secret := c.GetHeader("X-Webhook-Secret")
	if secret != os.Getenv("WEBHOOK_SECRET") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized webhook access"})
		return
	}

	// 2. Bind JSON
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	// 3. Process Logic (Upsert)
	if err := h.service.ProcessWebhookRegistration(payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "User synced successfully"})
}