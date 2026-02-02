package submission

import (
	"regexp"

	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/quiz"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Submit handles the grading and saving of a quiz submission
func (s *Service) Submit(submission *Submission) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Fetch all questions for this quiz to get the Correct Answers
		// We need the "truth" from the database, not what the user sent.
		var questions []quiz.Question
		if err := tx.Where("quiz_id = ?", submission.QuizID).Find(&questions).Error; err != nil {
			return err
		}

		// Create a map for fast lookup: QuestionID -> Question
		qMap := make(map[uint]quiz.Question)
		for _, q := range questions {
			qMap[q.ID] = q
		}

		// 2. Auto-Grade the Answers
		totalScore := 0
		for i := range submission.Answers {
			ans := &submission.Answers[i]
			question, exists := qMap[ans.QuestionID]

			if !exists {
				continue // Skip invalid question IDs
			}

			// Grade based on type
			isCorrect := false
			if question.Type == quiz.TypeMCQ {
				if ans.SelectedOption == question.CorrectAnswer {
					isCorrect = true
				}
			} else if question.Type == quiz.TypeText {
				// Simple Regex matching for text answers
				// e.g., CorrectAnswer regex: "(?i)vector" matches "It is a Vector quantity"
				matched, _ := regexp.MatchString(question.CorrectAnswer, ans.TextResponse)
				if matched {
					isCorrect = true
				}
			}

			ans.IsCorrect = isCorrect
			if isCorrect {
				totalScore += question.Points
			}
		}

		// 3. Update Submission Metadata
		submission.Score = totalScore

		// 4. Save to DB
		// GORM will insert the Submission AND all the Answers in one go
		if err := tx.Create(submission).Error; err != nil {
			return err
		}

		return nil
	})
}
