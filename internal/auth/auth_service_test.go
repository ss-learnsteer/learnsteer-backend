package auth

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	os.Setenv("JWT_SECRET", "test-secret-key-1234567890123456")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to in-memory test DB: %v", err)
	}

	err = db.AutoMigrate(&User{}, &SSOTicket{})
	if err != nil {
		t.Fatalf("Failed to auto-migrate auth models: %v", err)
	}

	return db
}

func TestIDAndNicknameGeneration(t *testing.T) {
	t.Run("generateStudentID format", func(t *testing.T) {
		id, err := generateStudentID()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		matched, _ := regexp.MatchString(`^LS-[a-z0-9]{10}$`, id)
		if !matched {
			t.Errorf("Expected student ID matching pattern 'LS-xxxxxxxxxx', got '%s'", id)
		}
	})

	t.Run("generateNickname format", func(t *testing.T) {
		nickname, err := generateNickname()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Should match WordWord + 2 digits
		matched, _ := regexp.MatchString(`^[A-Z][a-zA-Z]+[0-9]{2}$`, nickname)
		if !matched {
			t.Errorf("Expected nickname matching pattern 'WordWord99', got '%s'", nickname)
		}
	})
}

func TestCreateStudentFlow(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	dto := CreateStudentDTO{
		FirstName:      "Nimal",
		LastName:       "Silva",
		Email:          "nimal@example.com",
		Password:       "Password@123",
		NIC:            "200212345678",
		WhatsappNumber: "0779998877",
		Stream:         "Bio Science",
		Medium:         "Sinhala",
	}

	created, err := svc.CreateStudent(dto)
	if err != nil {
		t.Fatalf("CreateStudent failed: %v", err)
	}

	if !strings.HasPrefix(created.StudentID, "LS-") {
		t.Errorf("Expected StudentID to start with 'LS-', got '%s'", created.StudentID)
	}
	if created.Nickname == "" {
		t.Errorf("Expected non-empty Nickname")
	}
	if created.Email != "nimal@example.com" {
		t.Errorf("Expected email nimal@example.com, got '%s'", created.Email)
	}
	if created.PasswordHash == "" || created.PasswordHash == "Password@123" {
		t.Errorf("Password was not properly hashed")
	}

	// Test Duplicate Email
	t.Run("Duplicate Email Rejection", func(t *testing.T) {
		dupDTO := dto
		dupDTO.NIC = "200299999999"
		dupDTO.WhatsappNumber = "0770001122"
		_, err := svc.CreateStudent(dupDTO)
		if err == nil {
			t.Errorf("Expected error for duplicate email, got nil")
		}
	})

	// Test Duplicate NIC
	t.Run("Duplicate NIC Rejection", func(t *testing.T) {
		dupDTO := dto
		dupDTO.Email = "another@example.com"
		dupDTO.WhatsappNumber = "0770001122"
		_, err := svc.CreateStudent(dupDTO)
		if err == nil {
			t.Errorf("Expected error for duplicate NIC, got nil")
		}
	})
}

func TestLoginMultiIdentifier(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	dto := CreateStudentDTO{
		FirstName:      "Sunil",
		LastName:       "Perera",
		Email:          "sunil@example.com",
		Password:       "CorrectPass123",
		NIC:            "200155556666",
		WhatsappNumber: "0712223344",
		Stream:         "Physical Science",
		Medium:         "Sinhala",
	}

	student, err := svc.CreateStudent(dto)
	if err != nil {
		t.Fatalf("Failed to create student: %v", err)
	}

	// 1. Login with Nickname
	t.Run("Login via Nickname", func(t *testing.T) {
		token, user, err := svc.Login(student.Nickname, "CorrectPass123")
		if err != nil {
			t.Fatalf("Login with nickname failed: %v", err)
		}
		if token == "" {
			t.Errorf("Expected non-empty JWT token")
		}
		if user.StudentID != student.StudentID {
			t.Errorf("Expected student_id %s, got %s", student.StudentID, user.StudentID)
		}
	})

	// 2. Login with Email
	t.Run("Login via Email", func(t *testing.T) {
		_, user, err := svc.Login("sunil@example.com", "CorrectPass123")
		if err != nil {
			t.Fatalf("Login with email failed: %v", err)
		}
		if user.Email != "sunil@example.com" {
			t.Errorf("Expected email sunil@example.com, got %s", user.Email)
		}
	})

	// 3. Login with NIC
	t.Run("Login via NIC", func(t *testing.T) {
		_, user, err := svc.Login("200155556666", "CorrectPass123")
		if err != nil {
			t.Fatalf("Login with NIC failed: %v", err)
		}
		if user.NIC != "200155556666" {
			t.Errorf("Expected NIC 200155556666, got %s", user.NIC)
		}
	})

	// 4. Invalid Password
	t.Run("Login with Wrong Password", func(t *testing.T) {
		_, _, err := svc.Login(student.Nickname, "WrongPassword")
		if err == nil {
			t.Errorf("Expected invalid credentials error, got nil")
		}
	})

	// 5. Unknown Identifier
	t.Run("Login with Non-existent Identifier", func(t *testing.T) {
		_, _, err := svc.Login("NoSuchUserEver", "CorrectPass123")
		if err == nil {
			t.Errorf("Expected error for non-existent identifier, got nil")
		}
	})
}

func TestUpdateStudentProfile(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	student, err := svc.CreateStudent(CreateStudentDTO{
		FirstName:      "Kamal",
		LastName:       "Fernando",
		Email:          "kamal@example.com",
		Password:       "KamalPass123",
		NIC:            "200377778888",
		WhatsappNumber: "0751112233",
		Stream:         "Commerce",
		Medium:         "English",
	})
	if err != nil {
		t.Fatalf("CreateStudent failed: %v", err)
	}

	updateDTO := UpdateStudentDTO{
		Phone:         "0112345678",
		District:      "Kandy",
		School:        "Trinity College",
		City:          "Kandy",
		ALYear:        "2025",
		ALAttempt:     "1",
		DateOfBirth:   "2006-08-12",
		Gender:        "Male",
		GuardianPhone: "0770003344",
	}

	err = svc.UpdateStudent(student.ID, updateDTO)
	if err != nil {
		t.Fatalf("UpdateStudent failed: %v", err)
	}

	var updated User
	if err := db.First(&updated, student.ID).Error; err != nil {
		t.Fatalf("Failed to fetch updated user: %v", err)
	}

	if updated.District != "Kandy" {
		t.Errorf("Expected district Kandy, got '%s'", updated.District)
	}
	if updated.School != "Trinity College" {
		t.Errorf("Expected school Trinity College, got '%s'", updated.School)
	}
	if updated.City != "Kandy" {
		t.Errorf("Expected city Kandy, got '%s'", updated.City)
	}
	if updated.ALYear != "2025" {
		t.Errorf("Expected ALYear 2025, got '%s'", updated.ALYear)
	}
	if updated.StudentID != student.StudentID {
		t.Errorf("Expected student_id %s to remain unchanged, got %s", student.StudentID, updated.StudentID)
	}
}
