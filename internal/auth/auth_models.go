package auth

import (
	"time"

	"gorm.io/gorm"
)

// User represents a registered user (Student or Admin)
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Platform Identity (Auto-generated at creation, never changes)
	StudentID string `gorm:"uniqueIndex;not null" json:"student_id"` // LS-xxxxxxxxxx
	Nickname  string `gorm:"uniqueIndex;not null" json:"nickname"`   // BravePanda42

	// Core Identity (All required at creation)
	Email          string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash   string `json:"-"` // Never send password in JSON
	FirstName      string `gorm:"not null" json:"first_name"`
	LastName       string `gorm:"not null" json:"last_name"`
	NIC            string `gorm:"uniqueIndex;not null" json:"nic"`
	WhatsappNumber string `gorm:"not null" json:"whatsapp_number"`
	Role           string `gorm:"default:'student'" json:"role"` // 'student', 'admin'

	// Academic Profile (Stream + Medium required at creation)
	Stream    string `gorm:"not null" json:"stream"`
	Medium    string `gorm:"not null" json:"medium"`
	ALYear    string `json:"al_year"`    // Replaces exam_year + al_batch
	ALAttempt string `json:"al_attempt"` // 1st Attempt, 2nd Attempt, 3rd Attempt

	// Geographic (Collected later via Update Student)
	District string `json:"district"`
	School   string `json:"school"`
	City     string `json:"city"`

	// Demographics (Collected later via Update Student)
	Phone         string `json:"phone"`
	DateOfBirth   string `json:"date_of_birth"`
	Gender        string `json:"gender"`
	GuardianPhone string `json:"guardian_phone"`
}

// SSOTicket represents a short-lived token for cross-service authentication
type SSOTicket struct {
	ID        uint      `gorm:"primaryKey"`
	Ticket    string    `gorm:"uniqueIndex;not null"` // The random string (e.g., abc-123)
	StudentID string    `gorm:"not null"`             // Which user this ticket belongs to (LS-xxx)
	ExpiresAt time.Time `gorm:"not null"`             // Strict 60-second expiration
	CreatedAt time.Time
}