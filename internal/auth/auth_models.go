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

	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `json:"-"` // Never send password in JSON
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Role         string `gorm:"default:'student'" json:"role"` // 'student', 'admin'
	// New Demographic Data
	ExamYear int    `json:"exam_year"`
	Stream   string `json:"stream"`
	District string `json:"district"`
	School   string `json:"school"`

	NIC      string `gorm:"uniqueIndex" json:"nic"`
	WhatsappNumber string `json:"whatsapp_number"`
	ALBatch  string    `json:"al_batch"`   // Exam Year
	ALAttempt string    `json:"al_attempt"` // 1, 2, or 3
	Medium   string `json:"medium"`     // Sinhala, Tamil, English
}
