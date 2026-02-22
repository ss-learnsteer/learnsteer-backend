package quiz

import (
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// Service defines the methods our handler will use
type Service struct {
	db *gorm.DB
	cache *cache.Cache
}

// NewService initializes the service with the database connection
func NewService(db *gorm.DB) *Service {
	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 10 minutes
	c := cache.New(5*time.Minute, 10*time.Minute)
	
	return &Service{
		db: db,
		cache: c,
	}
}

// CreateQuiz handles the creation of a new quiz and its questions
// It uses a Transaction to ensure if questions fail, the quiz isn't created.
func (s *Service) CreateQuiz(quiz *Quiz) error {
	// GORM automatically handles creating the Questions and Options 
	// because they are nested inside the quiz struct we passed in.
	return s.db.Create(quiz).Error
}

// GetStartQuiz fetches the FULL quiz with all questions for the "Start" endpoint.
// Optimization: We use Preload("Questions") to fetch everything in one go.
// Note: Since 'CorrectAnswer' has `json:"-"` in models.go, it won't be sent to frontend.
func (s *Service) GetStartQuiz(id uint) (*Quiz, error) {
	cacheKey := fmt.Sprintf("quiz_start_%d", id)

	// Check the In-Memory Cache first
	if cachedData, found := s.cache.Get(cacheKey); found {
		// Cache HIT! Cast the interface back to a Quiz pointer
		// This skips the database completely and returns in microseconds
		return cachedData.(*Quiz), nil
	}
	
	var quiz Quiz

	// Query: Select Quiz WHERE id = ? AND Preload Questions
	err := s.db.Where("id = ?", id).
		Preload("Questions").
		Preload("Questions.Options").
		First(&quiz).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("quiz not found")
		}
		return nil, err
	}

	s.cache.Set(cacheKey, &quiz, cache.DefaultExpiration)

	return &quiz, nil
}

// ListQuizzes fetches a paginated list of quizzes WITHOUT questions.
// Optimization: This saves massive bandwidth by not loading the heavy questions data
// on the dashboard list.
func (s *Service) ListQuizzes(page, limit int, medium string, onlyVisible bool) ([]Quiz, int64, error) {
	// 1. Generate a unique cache key based on the exact request
	cacheKey := fmt.Sprintf("quizzes_list_p%d_l%d_m%s_v%t", page, limit, medium, onlyVisible)
	
	// 2. Check if the data is already in memory
	if cachedData, found := s.cache.Get(cacheKey); found {
		// Cache HIT! Return it instantly without touching Neon PostgreSQL
		// We expect cachedData to be a map holding our quizzes and total count
		data := cachedData.(map[string]interface{})
		return data["quizzes"].([]Quiz), data["total"].(int64), nil
	}

	// 3. Cache MISS! Go to the database (existing logic)
	var quizzes []Quiz
	var total int64

	// 1. Start building the base query
	query := s.db.Model(&Quiz{})

	// 2. Apply the medium filter ONLY if one was provided
	if medium != "" {
		query = query.Where("medium = ?", medium)
	}

	// Filter out hidden quizzes if the flag is true
	if onlyVisible {
		query = query.Where("is_visible = ?", true)
	}

	// 3. Get the total count of rows AFTER the filter is applied
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 4. Apply pagination and fetch the data
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&quizzes).Error; err != nil {
		return nil, 0, err
	}

	// 5. Save the result into the cache for the next 5 minutes
	s.cache.Set(cacheKey, map[string]interface{}{
		"quizzes": quizzes,
		"total":   total,
	}, cache.DefaultExpiration)

	return quizzes, total, nil
}

// DeleteQuiz performs a "Soft Delete" by hiding the quiz from public view.
// This preserves student submission history and grades!
func (s *Service) DeleteQuiz(quizID uint) error {
	// Instead of deleting, we force is_visible to false
	result := s.db.Model(&Quiz{}).
		Where("id = ?", quizID).
		Update("is_visible", false)

	if result.Error != nil {
		return result.Error
	}

	// If no rows were affected, the ID didn't exist in the database
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

// ReplaceQuizContent updates the quiz metadata and completely replaces all questions/options
func (s *Service) ReplaceQuizContent(quizID uint, updatedQuiz *Quiz) error {
	// Start a Database Transaction
	return s.db.Transaction(func(tx *gorm.DB) error {

		// 1. Update the parent Quiz metadata (Title, Description, etc.)
		// We only update specific columns to avoid overwriting the ID or CreatedAt
		if err := tx.Model(&Quiz{}).Where("id = ?", quizID).Updates(Quiz{
			Title:       updatedQuiz.Title,
			Description: updatedQuiz.Description,
			// Add any other top-level quiz fields here
		}).Error; err != nil {
			return err
		}

		// 2. Wipe the old Options and Questions cleanly
		// First, delete options tied to this quiz's questions to prevent foreign key errors
		if err := tx.Exec(`
			DELETE FROM options 
			WHERE question_id IN (SELECT id FROM questions WHERE quiz_id = ?)
		`, quizID).Error; err != nil {
			return err
		}

		// Next, delete the questions themselves
		if err := tx.Where("quiz_id = ?", quizID).Delete(&Question{}).Error; err != nil {
			return err
		}

		// 3. Insert the fully fresh Questions and Options
		// We must explicitly link the new questions to the existing quizID
		for i := range updatedQuiz.Questions {
			updatedQuiz.Questions[i].QuizID = quizID
		}

		if len(updatedQuiz.Questions) > 0 {
			// GORM will automatically insert the questions and their nested options here
			if err := tx.Create(&updatedQuiz.Questions).Error; err != nil {
				return err
			}
		}

		// If we reach here, returning nil tells GORM to COMMIT the transaction safely.
		return nil
	})
}

// UpdateVisibility explicitly toggles the is_visible flag for a specific quiz
func (s *Service) UpdateVisibility(quizID uint, isVisible bool) error {
	// We use .Update("column", value) rather than .Updates(struct).
	// This forces GORM to update the specific column, perfectly bypassing 
	// the zero-value issue where it ignores 'false' booleans.
	result := s.db.Model(&Quiz{}).
		Where("id = ?", quizID).
		Update("is_visible", isVisible)

	// Catch actual database errors (like connection drops)
	if result.Error != nil {
		return result.Error
	}

	// If the query succeeded but no rows were changed, the quiz doesn't exist
	if result.RowsAffected == 0 {
		return errors.New("quiz not found")
	}

	return nil
}

// UpdateQuiz completely replaces the quiz metadata, questions, and options
func (s *Service) UpdateQuiz(quiz *Quiz) error {
	// We use a Transaction so that if anything fails, the database rolls back
	// and we don't end up with a quiz that has no questions!
	return s.db.Transaction(func(tx *gorm.DB) error {
		
		// 1. Update the parent Quiz metadata (Title, Description, Medium, etc.)
		// We use .Updates() to apply the new fields to the existing ID
		if err := tx.Model(&Quiz{}).Where("id = ?", quiz.ID).Updates(quiz).Error; err != nil {
			return err
		}

		// 2. Wipe the old Questions (and by cascade, the old Options)
		if err := tx.Where("quiz_id = ?", quiz.ID).Delete(&Question{}).Error; err != nil {
			return err
		}

		// 3. Ensure all the incoming new questions have the correct QuizID attached
		for i := range quiz.Questions {
			quiz.Questions[i].QuizID = quiz.ID
		}

		// 4. Bulk insert the brand new Questions and Options
		if len(quiz.Questions) > 0 {
			if err := tx.Create(&quiz.Questions).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
