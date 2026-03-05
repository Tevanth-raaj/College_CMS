package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"server/db"
	"server/handlers/allocation"
	"time"
)

// StartAllocationScheduler starts the background scheduler
func StartAllocationScheduler() {
	log.Println("📅 Allocation scheduler started - checking every 1 MINUTE for closed teacher selection windows")

	// Run immediately on startup
	go checkAndRunAllocations()

	// Then run every minute to detect closed windows
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			checkAndRunAllocations()
		}
	}()
}

// checkAndRunAllocations checks if any teacher selection windows have closed and triggers auto-allocation
func checkAndRunAllocations() {
	log.Printf("⏰ [%s] Checking for closed teacher selection windows...", time.Now().Format("2006-01-02 15:04:05"))

	// Skip if any window is currently active (end date is today or future)
	var activeCount int
	err := db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM teacher_course_tracking
		WHERE is_active = 1
		AND window_start IS NOT NULL
		AND window_end IS NOT NULL
		AND DATE(window_start) <= DATE(NOW())
		AND DATE(window_end) >= DATE(NOW())
	`).Scan(&activeCount)
	if err != nil {
		log.Printf("⚠️  Failed to check active windows: %v", err)
		return
	}
	if activeCount > 0 {
		log.Printf("ℹ️  Active window found - skipping allocation run")
		return
	}

	// Find the most recent closed window (end date strictly before today)
	query := `
		SELECT academic_year, current_semester_type, window_start, window_end
		FROM teacher_course_tracking
		WHERE is_active = 1
		AND window_end IS NOT NULL
		AND DATE(window_end) < DATE(NOW())
		ORDER BY window_end DESC
		LIMIT 1
	`

	var academicYear, semesterType string
	var windowStart, windowEnd time.Time

	err = db.DB.QueryRow(query).Scan(&academicYear, &semesterType, &windowStart, &windowEnd)
	if err != nil {
		log.Printf("ℹ️  No closed windows to process")
		return // No closed windows found to process
	}

	log.Printf("🚀 Found closed window: Academic Year %s (%s)", academicYear, semesterType)
	log.Printf("   Window closed at: %v\n", windowEnd)

	// 1. Create a proper JSON payload so the handler knows WHAT to save
	payload := map[string]interface{}{
		"academic_year": academicYear,
		"semester_type": semesterType,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/allocations/run", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")

	// 2. Inject a dummy User ID into the Context 
	// (Prevents the DB save from crashing if it tries to fetch the logged-in user)
	// Change "user_id" to whatever context key your auth middleware uses if needed.
	ctx := context.WithValue(r.Context(), "user_id", 1)
	r = r.WithContext(ctx)

	// 3. Catch silent database crashes and print them!
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("🔥 CRITICAL DB CRASH during allocation save: %v", rec)
			}
		}()
		
		// Run the allocation
		allocation.RunAutoAllocation(w, r)
	}()

	// 4. Check result and mark window as inactive so it doesn't run again
	if w.Code == http.StatusOK {
		log.Printf("✅ Automatic allocation completed AND saved for Academic Year %s\n", academicYear)
		
		// IMPORTANT: Set is_active to 0 so the scheduler doesn't process this same window again!
		_, dbErr := db.DB.Exec(`
			UPDATE teacher_course_tracking 
			SET is_active = 0
			WHERE academic_year = ? AND current_semester_type = ?
		`, academicYear, semesterType)
		
		if dbErr != nil {
			log.Printf("⚠️ Failed to mark window as inactive: %v", dbErr)
		} else {
			log.Printf("🧹 Window marked as inactive for Academic Year %s", academicYear)
		}
	} else {
		log.Printf("❌ Automatic allocation failed with status: %d for Academic Year %s", w.Code, academicYear)
		log.Printf("   Response: %s\n", w.Body.String())
	}
}