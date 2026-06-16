package auth

import (
	"errors"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	Email          string
	Password       string
	FirstName      string
	LastName       string
	ExamYear       int
	Stream         string
	District       string
	School         string
	NIC            string
	WhatsappNumber string
	ALBatch        string
	ALAttempt      string
	Medium         string
}

// Register now accepts the DTO object
func (s *Service) Register(req RegisterDTO) error {
	// 1. Check if user exists by email
	var existing User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return errors.New("email already registered")
	}

	// 1b. Check if user exists by NIC
	if err := s.db.Where("nic = ?", req.NIC).First(&existing).Error; err == nil {
		return errors.New("a user with this NIC is already registered")
	}

	// 2. Hash Password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 3. Create User with ALL new fields
	user := User{
		Email:          req.Email,
		PasswordHash:   string(hashed),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Role:           "student",
		ExamYear:       req.ExamYear,
		Stream:         req.Stream,
		District:       req.District,
		School:         req.School,
		NIC:            req.NIC,
		WhatsappNumber: req.WhatsappNumber,
		ALBatch:        req.ALBatch,
		ALAttempt:      req.ALAttempt,
		Medium:         req.Medium,
	}

	err = s.db.Create(&user).Error
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "idx_users_nic") || strings.Contains(errStr, "users_nic_key") {
			return errors.New("a user with this NIC is already registered")
		}
		if strings.Contains(errStr, "idx_users_email") || strings.Contains(errStr, "users_email_key") {
			return errors.New("email already registered")
		}
		return err
	}
	return nil
}

func (s *Service) Login(email, password string) (string, error) {
	var user User

	// 1. Check if the user exists
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Security Best Practice: Return a generic error so attackers
			// don't know if the email exists or the password was just wrong.
			return "", errors.New("invalid email or password")
		}
		return "", err
	}

	// 2. Verify the Password
	// For students synced via the webhook, this compares the plain text NIC they
	// typed into the login form against the bcrypt hash in the database.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	// 3. Generate the JWT Token
	token, err := s.generateJWT(user)
	if err != nil {
		return "", errors.New("failed to generate authentication token")
	}

	return token, nil
}

// generateJWT is a private helper to create the token payload (claims)
func (s *Service) generateJWT(user User) (string, error) {
	// Define the claims (the data embedded inside the token)
	claims := jwt.MapClaims{
		"sub":    user.ID,                               // Subject (User ID)
		"role":   user.Role,                             // Important for Role-Based Access Control (student vs ss_member)
		"medium": user.Medium,
		"stream": user.Stream,
		"exp":    time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		"iat":    time.Now().Unix(),                     // Issued at
	}

	// Create a new token object, specifying the signing method and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with your server's secret key
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable is missing")
	}

	// Generate the encoded, secure token string
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

	switch result.Error {
	case nil:
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

	case gorm.ErrRecordNotFound:
		// --- CREATE NEW USER ---
		// Use OnConflict so that if the NIC already exists (e.g. a duplicate
		// submission from the sheet), we update instead of crashing.
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
		return s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "nic"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"email", "first_name", "last_name", "whatsapp_number",
				"school", "district", "stream", "medium",
				"al_batch", "al_attempt", "role", "updated_at",
			}),
		}).Create(&newUser).Error
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

// VerifyPasswordByNIC checks if the provided password matches the user's hash
func (s *Service) VerifyPasswordByNIC(nic, password string) (bool, error) {
	var user User

	// 1. Find the user by NIC
	err := s.db.Where("nic = ?", nic).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// User doesn't exist, so the password is automatically invalid
			return false, nil
		}
		// An actual database connection error occurred
		return false, err
	}

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Password does not match
		return false, nil
	}

	// Password matches perfectly
	return true, nil
}

// GenerateSSOTicket creates a 60-second secure ticket for a user
func (s *Service) GenerateSSOTicket(nic string) (string, error) {
	// 1. Generate a 32-byte secure random string
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	ticketStr := hex.EncodeToString(bytes)

	// 2. Save it to the database with a 60-second lifespan
	ticket := SSOTicket{
		Ticket:    ticketStr,
		NIC:       nic,
		ExpiresAt: time.Now().Add(60 * time.Second), // Very short window!
	}

	if err := s.db.Create(&ticket).Error; err != nil {
		return "", err
	}

	return ticketStr, nil
}

// ConsumeSSOTicket verifies the ticket, deletes it, and returns the User
func (s *Service) ConsumeSSOTicket(ticketStr string) (*User, error) {
	var ticket SSOTicket
	var user User

	// 1. We use a Database Transaction to ensure this ticket is only used EXACTLY once
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Find the ticket
		if err := tx.Where("ticket = ?", ticketStr).First(&ticket).Error; err != nil {
			return errors.New("invalid or expired ticket")
		}

		// Check expiration
		if time.Now().After(ticket.ExpiresAt) {
			tx.Delete(&ticket) // Clean up the expired ticket
			return errors.New("ticket has expired")
		}

		// Fetch the actual user associated with this NIC
		if err := tx.Where("nic = ?", ticket.NIC).First(&user).Error; err != nil {
			return errors.New("user not found")
		}

		// IMPORTANT: Delete the ticket immediately so it can never be reused
		if err := tx.Delete(&ticket).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}
