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
	Medium      string     `json:"medium"` // e.g., "Sinhala", "English", "Tamil"
	DurationMin int    `json:"duration_min"` // 0 = unlimited

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
	CorrectAnswer string `json:"correct_answer"` // e.g., "a" for MCQ or regex for Text
	Points        int    `json:"points"`
}

type Option struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	QuestionID uint   `json:"question_id"`
	Text       string `json:"text"`
	IsCorrect  bool   `json:"-"` // Hidden from JSON so students can't cheat via the API!
}
