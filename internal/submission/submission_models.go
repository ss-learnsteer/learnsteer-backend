package submission

import (
	"time"
)

type Submission struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	QuizID    uint      `gorm:"index" json:"quiz_id"`
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is null if they are still taking it
	CompletedAt *time.Time `json:"completed_at"`

	Score   int      `json:"score"`
	Answers []Answer `json:"answers"`
}

type Answer struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	SubmissionID uint `gorm:"index" json:"submission_id"`
	QuestionID   uint `json:"question_id"`

	// User's response
	SelectedOption string `json:"selected_option"` // For MCQ (e.g., "a")
	TextResponse   string `json:"text_response"`   // For Text questions

	IsCorrect bool `json:"is_correct"`
}
