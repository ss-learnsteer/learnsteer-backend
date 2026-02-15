package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/auth"
	"github.com/sasnaka-learnsteer/ss-quiz-platform-backend/internal/platform/database"
)

type DuplicateNIC struct {
	NIC   string
	Count int
}

func main() {
	// 1. Load configuration from the root .env file
	// First, try looking in the current execution directory (e.g., if run from project root)
	err := godotenv.Load(".env")
	if err != nil {
		// Fallback: If you navigated into the tool's folder to run it, look 3 levels up to the root
		err = godotenv.Load("../../../.env")
	}
	
	if err != nil {
		log.Println("⚠️ No .env file found at project root. Using system environment variables.")
	} else {
		log.Println("✅ Loaded environment variables from root .env file.")
	}

	// 2. Connect to Database
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("❌ Database connection failed: %v", err)
	}

	log.Println("🔍 Scanning for duplicate NICs in the 'users' table...")

	// 3. Find which NICs are duplicated
	var duplicateNICs []DuplicateNIC
	err = db.Table("users").
		Select("nic, count(*) as count").
		Where("nic IS NOT NULL AND nic != ''").
		Group("nic").
		Having("count(*) > 1").
		Find(&duplicateNICs).Error

	if err != nil {
		log.Fatalf("❌ Error querying duplicates: %v", err)
	}

	if len(duplicateNICs) == 0 {
		log.Println("✅ No duplicate NICs found! Your database is clean.")
		return
	}

	log.Printf("⚠️ Found %d NIC(s) that are duplicated.\n\n", len(duplicateNICs))

	// 4. Fetch and print the actual user records for those NICs
	for _, dup := range duplicateNICs {
		log.Printf("--- NIC: %s (Appears %d times) ---", dup.NIC, dup.Count)
		
		var users []auth.User
		db.Where("nic = ?", dup.NIC).Order("created_at ASC").Find(&users)

		for _, u := range users {
			log.Printf("   ID: %d | Email: %-25s | Name: %s %s | Created: %s", 
				u.ID, u.Email, u.FirstName, u.LastName, u.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		log.Println()
	}
}