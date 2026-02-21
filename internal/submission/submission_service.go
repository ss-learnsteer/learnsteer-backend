package submission

import (
	"errors"
	"strconv"
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

		if q, exists := questionsMap[studentAnswer.QuestionID]; exists {
			selected := strings.TrimSpace(studentAnswer.SelectedOption)

			// Build a map of optionID -> IsCorrect for this question
			optionCorrectMap := make(map[uint]bool)
			for _, opt := range q.Options {
				optionCorrectMap[opt.ID] = opt.IsCorrect
			}

			// Try to parse selected_option as a numeric option ID (frontend sends DB IDs)
			if optID, err := strconv.ParseUint(selected, 10, 64); err == nil {
				// Frontend mode: "563" → look up IsCorrect by option ID
				if correct, found := optionCorrectMap[uint(optID)]; found && correct {
					isCorrect = true
					totalScore += q.Points
				}
			} else {
				// Test/API mode: "b" → compare against CorrectAnswer letter
				selectedLower := strings.ToLower(selected)
				correctLower := strings.ToLower(strings.TrimSpace(q.CorrectAnswer))
				if selectedLower != "" && selectedLower == correctLower {
					isCorrect = true
					totalScore += q.Points
				}
			}
		}

		submissionAnswers = append(submissionAnswers, Answer{
			QuestionID:     studentAnswer.QuestionID,
			SelectedOption: studentAnswer.SelectedOption,
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
