package submission

import (
	"time"

	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
)

type Submission struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	UserID      uint       `gorm:"index" json:"user_id"`
	QuizID      uint       `gorm:"index" json:"quiz_id"`
	Quiz        *quiz.Quiz `gorm:"foreignKey:QuizID;constraint:-;" json:"quiz,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Score       int        `json:"score"`
	Answers     []Answer   `json:"answers,omitempty"`
}

type Answer struct {
	ID           uint `gorm:"primaryKey" json:"id"`
	SubmissionID uint `gorm:"index" json:"submission_id"`
	QuestionID   uint `json:"question_id"`

	// User's response
	SelectedOption string `json:"selected_option"` // For MCQ: the chosen option letter (e.g. "a", "b")
	TextResponse   string `json:"text_response"`   // For text questions: the written answer

	IsCorrect bool `json:"is_correct"`
}
