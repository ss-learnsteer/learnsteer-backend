package quiz

import (
	"time"

	"gorm.io/gorm"
)

// Quiz represents a collection of questions
type Quiz struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string `json:"title"`
	Description string `json:"description"`
	Medium      string `json:"medium" gorm:"default:'Sinhala';not null"` // e.g., "Sinhala", "English", "Tamil"
	Stream string `json:"stream" gorm:"not null"`
	IsVisible   *bool `json:"is_visible" gorm:"default:false"`
	// The custom explicit soft-delete flag
	IsDeleted bool `json:"is_deleted" gorm:"default:false"`
	DurationMin int   `json:"duration_min"` // 0 = unlimited

	ReleaseDate      *time.Time `json:"release_date"`       // When it opens
	EndDate          *time.Time `json:"end_date"`           // When it closes
	MarkingSchemeURL string     `json:"marking_scheme_url"` // Cloudinary PDF link (Wiwarana)

	// HasMany relationship: A quiz has many questions
	Questions []Question `json:"questions" gorm:"foreignKey:QuizID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// QuestionType enum helper
type QuestionType string

const (
	TypeMCQ  QuestionType = "mcq"
	TypeText QuestionType = "text"
)

// Question holds the content in Markdown format
type Question struct {
	ID     uint         `gorm:"primaryKey" json:"id"`
	QuizID uint         `gorm:"index" json:"quiz_id"` // Foreign Key
	Type   QuestionType `json:"type"`                 // 'mcq' or 'text'

	// Content
	TextMarkdown string `gorm:"type:text" json:"text_markdown"` // "What is **velocity**?"
	ImageURL     string `json:"image_url,omitempty"`            // Optional diagram

	Options []Option `json:"options" gorm:"foreignKey:QuestionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Correct Answer (Hidden from frontend JSON usually, handled in service layer)
	CorrectAnswer string `json:"-"` // e.g., "a" for MCQ or regex for Text
	Points        int    `json:"points"`
}

type Option struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	QuestionID uint   `json:"question_id"`
	Text       string `json:"text"`
	ImageURL   string `json:"image_url,omitempty"`
	IsCorrect  bool   `json:"-"` // Hidden from JSON so students can't cheat via the API!
}
