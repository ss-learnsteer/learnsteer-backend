package quiz

import (
	"errors"

	"gorm.io/gorm"
)

// Service defines the methods our handler will use
type Service struct {
	db *gorm.DB
}

// NewService initializes the service with the database connection
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateQuiz handles the creation of a new quiz and its questions
// It uses a Transaction to ensure if questions fail, the quiz isn't created.
func (s *Service) Create(quiz *Quiz) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quiz).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetStartQuiz fetches the FULL quiz with all questions for the "Start" endpoint.
// Optimization: We use Preload("Questions") to fetch everything in one go.
// Note: Since 'CorrectAnswer' has `json:"-"` in models.go, it won't be sent to frontend.
func (s *Service) GetStartQuiz(id uint) (*Quiz, error) {
	var quiz Quiz

	// Query: Select Quiz WHERE id = ? AND Preload Questions
	err := s.db.Preload("Questions").First(&quiz, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("quiz not found")
		}
		return nil, err
	}

	return &quiz, nil
}

// ListQuizzes fetches a paginated list of quizzes WITHOUT questions.
// Optimization: This saves massive bandwidth by not loading the heavy questions data
// on the dashboard list.
func (s *Service) ListQuizzes(page, limit int) ([]Quiz, int64, error) {
	var quizzes []Quiz
	var total int64

	offset := (page - 1) * limit

	// 1. Get Total Count (for pagination UI)
	if err := s.db.Model(&Quiz{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 2. Get the light-weight list
	// We deliberately do NOT call Preload("Questions") here.
	err := s.db.Limit(limit).Offset(offset).Order("created_at desc").Find(&quizzes).Error
	if err != nil {
		return nil, 0, err
	}

	return quizzes, total, nil
}

// DeleteQuiz removes a quiz and (optionally) its questions
func (s *Service) Delete(id uint) error {
	// GORM handles soft delete automatically if DeletedAt is in the model
	result := s.db.Delete(&Quiz{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("quiz not found")
	}
	return nil
}

// GetQuestionsByQuizID fetches all questions for a given quiz.
// It uses Preload to automatically fetch the associated multiple-choice options 
// so the frontend has everything it needs to render the test.
func (s *Service) GetQuestionsByQuizID(quizID uint) ([]Question, error) {
	var questions []Question

	// We query the questions table where the foreign key matches the quizID.
	// Preload("Options") tells GORM to execute a second highly-optimized query 
	// to fetch all related options and attach them to the correct questions in memory.
	err := s.db.Where("quiz_id = ?", quizID).
		Preload("Options"). 
		Find(&questions).Error

	if err != nil {
		return nil, err
	}

	return questions, nil
}
