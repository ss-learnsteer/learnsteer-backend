package submission

import (
	"errors"
	"strings"
	"time"

	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// GradeAndSubmit calculates the score and saves the student's submission
func (s *Service) GradeAndSubmit(userID uint, req SubmitQuizPayload) (*Submission, error) {
	var targetQuiz quiz.Quiz

	// 1. Fetch the quiz with all questions and options
	if err := s.db.Preload("Questions.Options").First(&targetQuiz, req.QuizID).Error; err != nil {
		return nil, errors.New("quiz not found")
	}

	// 2. Build a memory map for instant O(1) Question lookups
	questionsMap := make(map[uint]quiz.Question)
	for _, q := range targetQuiz.Questions {
		questionsMap[q.ID] = q
	}

	// 3. Grade the student's submission
	var totalScore int
	var submissionAnswers []Answer

	for _, studentAnswer := range req.Answers {
		isCorrect := false

		// Lookup the actual question from the database
		if q, exists := questionsMap[studentAnswer.QuestionID]; exists {

			// Safety check: ensure the string isn't empty
			if len(studentAnswer.SelectedOption) > 0 {
				// Convert "a" -> 0, "b" -> 1, "c" -> 2
				char := strings.ToLower(studentAnswer.SelectedOption)[0]
				optIndex := int(char - 'a')

				// Bounds check: prevent a panic if a student maliciously sends "z"
				if optIndex >= 0 && optIndex < len(q.Options) {
					// Check if the option at that index is the correct one
					if q.Options[optIndex].IsCorrect {
						isCorrect = true
						totalScore += q.Points
					}
				}
			}
		}

		// 4. Create the Answer record using YOUR exact struct fields
		submissionAnswers = append(submissionAnswers, Answer{
			QuestionID:     studentAnswer.QuestionID,
			SelectedOption: studentAnswer.SelectedOption, // Safely save the "a" or "b"
			IsCorrect:      isCorrect,
		})
	}

	// 5. Create and save the final Submission record
	now := time.Now()
	submission := Submission{
		UserID:      userID,
		QuizID:      req.QuizID,
		Score:       totalScore,
		CompletedAt: &now, // Mark it as finished right now
		Answers:     submissionAnswers,
	}

	if err := s.db.Create(&submission).Error; err != nil {
		return nil, errors.New("failed to save submission")
	}

	return &submission, nil
}