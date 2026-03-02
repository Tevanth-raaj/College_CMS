package main

import (
	"fmt"
	"log"
	"net/http"

	"server/db"
	"server/middleware"
	"server/routes"
	"server/scheduler"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env in server directory
	// Try loading from current directory first, then from server/ subdirectory (if running from root)
	if err := godotenv.Load(".env"); err != nil {
		if err := godotenv.Load("server/.env"); err != nil {
			log.Println("No .env file found or failed to load; using environment defaults")
		}
	}
	// Initialize database
	err := db.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.CloseDB()

	// Create department overview tables
	if err := db.CreateDepartmentOverviewTables(); err != nil {
		log.Fatal("Failed to create department overview tables:", err)
	}

	// Create cluster tables
	if err := db.CreateClusterTables(); err != nil {
		log.Fatal("Failed to create cluster tables:", err)
	}

	// Add visibility columns to department data tables
	if err := db.AddVisibilityColumns(); err != nil {
		log.Fatal("Failed to add visibility columns:", err)
	}

	// Rename semesters table to normal_cards (must happen before AddVisibilitySemestersCourses)
	if err := db.RenameSemestersToNormalCards(); err != nil {
		log.Fatal("Failed to rename semesters to normal_cards:", err)
	}

	// Add visibility columns to normal_cards and courses tables
	if err := db.AddVisibilitySemestersCourses(); err != nil {
		log.Fatal("Failed to add visibility columns to semesters/courses:", err)
	}

	// Add source tracking columns for shared items
	if err := db.AddSourceDepartmentColumns(); err != nil {
		log.Fatal("Failed to add source tracking columns:", err)
	}

	// Create sharing tracking table
	if err := db.CreateSharingTrackingTable(); err != nil {
		log.Fatal("Failed to create sharing tracking table:", err)
	}

	// Remove UNIQUE constraints on position to allow soft deletes
	if err := db.RemoveUniquePositionConstraints(); err != nil {
		log.Fatal("Failed to remove unique position constraints:", err)
	}

	// Create regulation management tables (PHASE 1 - isolated, zero breakage)
	if err := db.CreateRegulationTables(); err != nil {
		log.Fatal("Failed to create regulation tables:", err)
	}

	// Add regulation reference columns (PHASE 2 - shadow links, nullable)
	if err := db.AddRegulationRefColumns(); err != nil {
		log.Fatal("Failed to add regulation reference columns:", err)
	}

	// Create honour card tables
	if err := db.CreateHonourCardTables(); err != nil {
		log.Fatal("Failed to create honour card tables:", err)
	}

	// Add name column to normal_cards table
	if err := db.AddSemesterNameColumn(); err != nil {
		log.Fatal("Failed to add semester name column:", err)
	}

	// Remove name column from normal_cards table (card_type is now used as the display name)
	if err := db.RemoveNameColumnFromNormalCards(); err != nil {
		log.Fatal("Failed to remove name column from normal_cards:", err)
	}

	// Create mark entry student permissions table
	if err := db.CreateMarkEntryStudentPermissionsTables(); err != nil {
		log.Fatal("Failed to create mark entry student permissions tables:", err)
	}

	// Add user_id to mark_entry_windows for user-based windows
	if err := db.AddUserIdToMarkEntryWindows(); err != nil {
		log.Fatal("Failed to add user_id to mark_entry_windows:", err)
	}

	// Start the allocation scheduler (auto-runs when teacher selection window closes)
	scheduler.StartAllocationScheduler()

	// Setup routes
	router := routes.SetupRoutes()

	// Serve static files from uploads directory
	uploadDir := http.Dir("./uploads")
	fileServer := http.FileServer(uploadDir)
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fileServer))

	// Wrap with CORS middleware
	handler := middleware.CORSMiddleware(router)

	fmt.Println("Server started at http://localhost:5000")
	log.Fatal(http.ListenAndServe(":5000", handler))
}
