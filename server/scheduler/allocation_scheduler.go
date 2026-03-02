package scheduler

import (
	"log"
	"net/http"
	"net/http/httptest"
	"server/db"
	"server/handlers/allocation"
	"time"
)

// StartAllocationScheduler starts the background scheduler
func StartAllocationScheduler() {
	log.Println("📅 Allocation scheduler started - checking every 1 MINUTE for testing")

	// Run immediately on startup
	go checkAndRunAllocations()

	// Then run every minute for testing
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			checkAndRunAllocations()
		}
	}()
}

// checkAndRunAllocations checks if any teacher selection windows have closed
func checkAndRunAllocations() {
	log.Printf("⏰ [%s] Checking for closed teacher selection windows...", time.Now().Format("2006-01-02 15:04:05"))
	
	// Query teacher_course_tracking for windows that closed
	// Window closes at END OF DAY (23:59:59) on window_end date
	// So we compare with DATE_ADD(window_end, INTERVAL 1 DAY) at midnight
	query := `
		SELECT academic_year, window_start, window_end
		FROM teacher_course_tracking
		WHERE DATE_ADD(window_end, INTERVAL 1 DAY) <= NOW()
		AND DATE_ADD(window_end, INTERVAL 1 DAY) >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		LIMIT 1
	`

	var academicYear string
	var windowStart, windowEnd time.Time

	err := db.DB.QueryRow(query).Scan(&academicYear, &windowStart, &windowEnd)
	if err != nil {
		log.Printf("   ℹ️  No closed windows found (all windows are either still open or closed more than 7 days ago)")
		return
	}

	log.Printf("🚀 Teacher selection window CLOSED for Academic Year: %s", academicYear)
	log.Printf("   Window period: %v to %v (closes at 23:59:59 on end date)", windowStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"))
	log.Printf("   Triggering automatic allocation for ALL semesters...")

	// Trigger allocation by calling the handler
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/allocations/run", nil)

	allocation.RunAutoAllocation(w, r)

	if w.Code == http.StatusOK {
		log.Println("✅ Automatic allocation completed successfully for all semesters")
	} else {
		log.Printf("❌ Automatic allocation failed with status: %d", w.Code)
		log.Printf("   Response: %s", w.Body.String())
	}
}
