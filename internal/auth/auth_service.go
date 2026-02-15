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

// ProcessWebhookRegistration handles the "Upsert" logic
func (s *Service) ProcessWebhookRegistration(data WebhookPayload) error {
	var user User

	// 1. Check if user already exists by Email
	result := s.db.Where("email = ?", data.Email).First(&user)

	// 2. Prepare the Default Password (NIC)
	// We hash the NIC so they can use it to login
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(data.NIC), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Safety fallback: if no role is provided, default to student
	assignedRole := data.Role
	if assignedRole == "" {
		assignedRole = "student"
	}

	if result.Error == nil {
		// --- UPDATE EXISTING USER ---
		// We update everything EXCEPT the password (in case they changed it manually)
		user.FirstName = data.FirstName
		user.LastName = data.LastName
		user.NIC = data.NIC
		user.WhatsappNumber = data.WhatsappNumber
		user.School = data.School
		user.District = data.District
		user.Stream = data.Stream
		user.Medium = data.Medium
		user.ALBatch = data.ALBatch
		user.ALAttempt = data.ALAttempt
		user.Role = assignedRole
		
		return s.db.Save(&user).Error

	} else if result.Error == gorm.ErrRecordNotFound {
		// --- CREATE NEW USER ---
		newUser := User{
			Email:          data.Email,
			PasswordHash:   string(hashedPassword), // Default Password = NIC
			Role:           assignedRole,
			FirstName:      data.FirstName,
			LastName:       data.LastName,
			NIC:            data.NIC,
			WhatsappNumber: data.WhatsappNumber,
			School:         data.School,
			District:       data.District,
			Stream:         data.Stream,
			Medium:         data.Medium,
			ALBatch:        data.ALBatch,
			ALAttempt:      data.ALAttempt,
		}
		return s.db.Create(&newUser).Error
	}

	return result.Error
}

// CheckNICExists queries the database to see if the NIC is already registered
func (s *Service) CheckNICExists(nic string) (bool, error) {
	var count int64
	
	// Query the User table where the nic matches
	err := s.db.Model(&User{}).Where("nic = ?", nic).Count(&count).Error
	if err != nil {
		return false, err
	}
	
	// If count is greater than 0, the NIC exists
	return count > 0, nil
}

// GetUserByNIC fetches the user by NIC so we can access their password hash
func (s *Service) GetUserByNIC(nic string) (*User, error) {
	var user User
	err := s.db.Where("nic = ?", nic).First(&user).Error
	if err != nil {
		return nil, err // Returns gorm.ErrRecordNotFound if they don't exist
	}
	return &user, nil
}