package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

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

// ---------------------------------------------------------------------------
// Student ID & Nickname Generators
// ---------------------------------------------------------------------------

// generateStudentID creates a unique LS-xxxxxxxxxx platform identifier
func generateStudentID() (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyz23456789" // Ambiguity-safe (no 0/o/1/l/i)
	const length = 10
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return "LS-" + string(b), nil
}

// Adjective + Animal word lists for cartoon-themed nicknames
var adjectives = []string{
	"Brave", "Swift", "Clever", "Mighty", "Lucky", "Cosmic", "Blazing", "Turbo", "Epic", "Super",
	"Hyper", "Mega", "Ultra", "Astro", "Nitro", "Pixel", "Neon", "Thunder", "Storm", "Flash",
	"Sonic", "Rocket", "Stellar", "Zen", "Quantum", "Mystic", "Crystal", "Shadow", "Golden", "Silver",
	"Frost", "Ember", "Nova", "Volt", "Blaze", "Apex", "Prime", "Chill", "Dusk", "Dawn",
	"Iron", "Steel", "Aqua", "Solar", "Lunar", "Arctic", "Rapid", "Bold", "Noble", "Vivid",
}

var animals = []string{
	"Panda", "Fox", "Owl", "Tiger", "Eagle", "Falcon", "Wolf", "Bear", "Shark", "Phoenix",
	"Dragon", "Lion", "Hawk", "Dolphin", "Koala", "Penguin", "Rabbit", "Turtle", "Cheetah", "Panther",
	"Otter", "Lynx", "Raven", "Cobra", "Gecko", "Puma", "Bison", "Crane", "Parrot", "Jaguar",
	"Whale", "Husky", "Badger", "Viper", "Mantis", "Toucan", "Condor", "Ibis", "Flamingo", "Chameleon",
}

// generateNickname creates a cartoon-themed nickname like BravePanda42
func generateNickname() (string, error) {
	adjIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	if err != nil {
		return "", err
	}
	aniIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(animals))))
	if err != nil {
		return "", err
	}
	numVal, err := rand.Int(rand.Reader, big.NewInt(90)) // 10–99
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%d", adjectives[adjIdx.Int64()], animals[aniIdx.Int64()], numVal.Int64()+10), nil
}

// ---------------------------------------------------------------------------
// DTOs (Data Transfer Objects)
// ---------------------------------------------------------------------------

// CreateStudentDTO holds data needed for student creation
type CreateStudentDTO struct {
	FirstName      string
	LastName       string
	Email          string
	Password       string
	NIC            string
	WhatsappNumber string
	Stream         string
	Medium         string
}

// UpdateStudentDTO holds data for profile completion (all optional)
type UpdateStudentDTO struct {
	Phone         string
	District      string
	School        string
	City          string
	ALYear        string
	ALAttempt     string
	DateOfBirth   string
	Gender        string
	GuardianPhone string
}

// ---------------------------------------------------------------------------
// Create Student (replaces Register)
// ---------------------------------------------------------------------------

// CreateStudent creates a new student account with auto-generated StudentID and Nickname
func (s *Service) CreateStudent(req CreateStudentDTO) (*User, error) {
	// 1. Check if email already exists
	var existing User
	if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, errors.New("email already registered")
	}

	// 2. Check if NIC already exists
	if err := s.db.Where("nic = ?", req.NIC).First(&existing).Error; err == nil {
		return nil, errors.New("NIC already registered")
	}

	// 3. Hash Password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 4. Generate unique StudentID (retry up to 5 times on collision)
	var studentID string
	for i := 0; i < 5; i++ {
		studentID, err = generateStudentID()
		if err != nil {
			return nil, err
		}
		var count int64
		s.db.Model(&User{}).Where("student_id = ?", studentID).Count(&count)
		if count == 0 {
			break
		}
		if i == 4 {
			return nil, errors.New("failed to generate unique student ID")
		}
	}

	// 5. Generate unique Nickname (retry up to 5 times on collision)
	var nickname string
	for i := 0; i < 5; i++ {
		nickname, err = generateNickname()
		if err != nil {
			return nil, err
		}
		var count int64
		s.db.Model(&User{}).Where("nickname = ?", nickname).Count(&count)
		if count == 0 {
			break
		}
		if i == 4 {
			return nil, errors.New("failed to generate unique nickname")
		}
	}

	// 6. Create User
	user := User{
		StudentID:      studentID,
		Nickname:       nickname,
		Email:          req.Email,
		PasswordHash:   string(hashed),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		NIC:            req.NIC,
		WhatsappNumber: req.WhatsappNumber,
		Role:           "student",
		Stream:         req.Stream,
		Medium:         req.Medium,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// ---------------------------------------------------------------------------
// Update Student (profile completion)
// ---------------------------------------------------------------------------

// UpdateStudent updates a student's profile with additional data (only non-empty fields)
func (s *Service) UpdateStudent(userID uint, dto UpdateStudentDTO) error {
	updates := map[string]interface{}{}

	if dto.Phone != "" {
		updates["phone"] = dto.Phone
	}
	if dto.District != "" {
		updates["district"] = dto.District
	}
	if dto.School != "" {
		updates["school"] = dto.School
	}
	if dto.City != "" {
		updates["city"] = dto.City
	}
	if dto.ALYear != "" {
		updates["al_year"] = dto.ALYear
	}
	if dto.ALAttempt != "" {
		updates["al_attempt"] = dto.ALAttempt
	}
	if dto.DateOfBirth != "" {
		updates["date_of_birth"] = dto.DateOfBirth
	}
	if dto.Gender != "" {
		updates["gender"] = dto.Gender
	}
	if dto.GuardianPhone != "" {
		updates["guardian_phone"] = dto.GuardianPhone
	}

	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	return s.db.Model(&User{}).Where("id = ?", userID).Updates(updates).Error
}

// ---------------------------------------------------------------------------
// Login (accepts Nickname, Email, or NIC)
// ---------------------------------------------------------------------------

func (s *Service) Login(identifier, password string) (string, *User, error) {
	var user User

	// 1. Find user by nickname, email, or NIC
	err := s.db.Where("nickname = ? OR email = ? OR nic = ?", identifier, identifier, identifier).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}

	// 2. Verify the Password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// 3. Generate the JWT Token
	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, errors.New("failed to generate authentication token")
	}

	return token, &user, nil
}

// ---------------------------------------------------------------------------
// JWT Generation
// ---------------------------------------------------------------------------

// generateJWT creates a token with both internal ID and public student_id/nickname
func (s *Service) generateJWT(user User) (string, error) {
	claims := jwt.MapClaims{
		"sub":        user.ID,        // Internal numeric PK (for DB queries)
		"student_id": user.StudentID, // Public platform ID (for API responses)
		"nickname":   user.Nickname,  // Cartoon display name
		"role":       user.Role,
		"medium":     user.Medium,
		"stream":     user.Stream,
		"exp":        time.Now().Add(time.Hour * 24).Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET environment variable is missing")
	}

	return token.SignedString([]byte(secret))
}

// ---------------------------------------------------------------------------
// Webhook Registration (Google Sheets sync)
// ---------------------------------------------------------------------------

// ProcessWebhookRegistration handles the "Upsert" logic for webhook-synced users
func (s *Service) ProcessWebhookRegistration(data WebhookPayload) error {
	var user User

	// 1. Check if user already exists by Email
	result := s.db.Where("email = ?", data.Email).First(&user)

	// 2. Prepare the Default Password (NIC)
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
		user.FirstName = data.FirstName
		user.LastName = data.LastName
		user.NIC = data.NIC
		user.WhatsappNumber = data.WhatsappNumber
		user.School = data.School
		user.District = data.District
		user.Stream = data.Stream
		user.Medium = data.Medium
		user.ALYear = data.ALYear
		user.ALAttempt = data.ALAttempt
		user.Role = assignedRole

		return s.db.Save(&user).Error

	case gorm.ErrRecordNotFound:
		// --- CREATE NEW USER ---
		// Auto-generate StudentID and Nickname for webhook-created users
		studentID, err := generateStudentID()
		if err != nil {
			return err
		}
		nickname, err := generateNickname()
		if err != nil {
			return err
		}

		newUser := User{
			StudentID:      studentID,
			Nickname:       nickname,
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
			ALYear:         data.ALYear,
			ALAttempt:      data.ALAttempt,
		}
		return s.db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "nic"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"email", "first_name", "last_name", "whatsapp_number",
				"school", "district", "stream", "medium",
				"al_year", "al_attempt", "role", "updated_at",
			}),
		}).Create(&newUser).Error
	}

	return result.Error
}

// ---------------------------------------------------------------------------
// NIC & Profile Lookups
// ---------------------------------------------------------------------------

// CheckNICExists queries the database to see if the NIC is already registered
func (s *Service) CheckNICExists(nic string) (bool, error) {
	var count int64
	err := s.db.Model(&User{}).Where("nic = ?", nic).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserByNIC fetches the user by NIC
func (s *Service) GetUserByNIC(nic string) (*User, error) {
	var user User
	err := s.db.Where("nic = ?", nic).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByStudentID fetches the user by their platform student ID
func (s *Service) GetUserByStudentID(studentID string) (*User, error) {
	var user User
	err := s.db.Where("student_id = ?", studentID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// VerifyPasswordByNIC checks if the provided password matches the user's hash
func (s *Service) VerifyPasswordByNIC(nic, password string) (bool, error) {
	var user User

	err := s.db.Where("nic = ?", nic).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}

// ---------------------------------------------------------------------------
// SSO Tickets (B2B Single Sign-On)
// ---------------------------------------------------------------------------

// GenerateSSOTicket creates a 60-second secure ticket for a user
func (s *Service) GenerateSSOTicket(studentID string) (string, error) {
	// 1. Generate a 32-byte secure random string
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	ticketStr := fmt.Sprintf("%x", bytes)

	// 2. Save it to the database with a 60-second lifespan
	ticket := SSOTicket{
		Ticket:    ticketStr,
		StudentID: studentID,
		ExpiresAt: time.Now().Add(60 * time.Second),
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

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Find the ticket
		if err := tx.Where("ticket = ?", ticketStr).First(&ticket).Error; err != nil {
			return errors.New("invalid or expired ticket")
		}

		// Check expiration
		if time.Now().After(ticket.ExpiresAt) {
			tx.Delete(&ticket)
			return errors.New("ticket has expired")
		}

		// Fetch the actual user associated with this StudentID
		if err := tx.Where("student_id = ?", ticket.StudentID).First(&user).Error; err != nil {
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
