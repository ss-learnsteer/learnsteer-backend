package quiz

import (
	"time"

	"gorm.io/gorm"
	"github.com/lib/pq"
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
    Stream pq.StringArray `json:"stream" gorm:"type:text[];default:'{\"General\"}';not null"`	
	IsVisible   *bool  `json:"is_visible" gorm:"default:false"`
	// The custom explicit soft-delete flag
	IsDeleted   bool `json:"is_deleted" gorm:"default:false"`
	DurationMin int  `json:"duration_min"` // 0 = unlimited

	ReleaseDate      *time.Time `json:"release_date"`       // When it opens
	EndDate          *time.Time `json:"end_date"`           // When it closes
	MarkingSchemeURL string     `json:"marking_scheme_url"` // Cloudinary PDF link (Wiwarana)
	SubjectID        *uint      `gorm:"index" json:"subject_id,omitempty"`
	UnitID           *uint      `gorm:"index" json:"unit_id,omitempty"`
	IsMock           bool       `gorm:"default:false" json:"is_mock"`
	SessionNumber    int        `gorm:"default:1" json:"session_number"`

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
	Explanation  string `gorm:"type:text" json:"explanation,omitempty"`
	UnitName     string `json:"unit_name,omitempty"`
	OrderIndex   int    `gorm:"default:0" json:"order_index"`

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

// LeaderboardEntry represents a ranked student score in a quiz leaderboard
type LeaderboardEntry struct {
	Rank        int        `json:"rank"`
	UserID      uint       `json:"user_id"`
	StudentID   string     `json:"student_id"`
	Nickname    string     `json:"nickname"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	School      string     `json:"school"`
	District    string     `json:"district"`
	Score       int        `json:"score"`
	CompletedAt *time.Time `json:"completed_at"`
}
