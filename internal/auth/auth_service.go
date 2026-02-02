package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"os"
)

// Service struct remains the same
type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// RegisterDTO holds all the data needed for registration
type RegisterDTO struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	ExamYear  int
	Stream    string
	District  string
	School    string
}

// Register now accepts the DTO object
func (s *Service) Register(req RegisterDTO) error {
	// 1. Check if user exists
	var existing User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return errors.New("email already registered")
	}

	// 2. Hash Password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 3. Create User with ALL new fields
	user := User{
		Email:        req.Email,
		PasswordHash: string(hashed),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         "student",
		// New Demographic Data
		ExamYear: req.ExamYear,
		Stream:   req.Stream,
		District: req.District,
		School:   req.School,
	}

	return s.db.Create(&user).Error
}

// Login function remains the same...
func (s *Service) Login(email, password string) (string, error) {
    // ... (Keep existing Login logic)
    // 1. Find User
    var user User
    if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
        return "", errors.New("invalid credentials")
    }

    // 2. Compare Passwords
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return "", errors.New("invalid credentials")
    }

    // 3. Generate JWT
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub":  user.ID,
        "role": user.Role,
        "exp":  time.Now().Add(time.Hour * 24 * 7).Unix(),
    })

    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        return "", errors.New("JWT_SECRET not configured")
    }

    return token.SignedString([]byte(secret))
}