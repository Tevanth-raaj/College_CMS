package studentteacher

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"server/db"
)

// TeacherCoursePreference represents a faculty course selection
type TeacherCoursePreference struct {
	ID                  int            `json:"id"`
	TeacherID           int            `json:"teacher_id"`
	CourseID            string         `json:"course_id"`
	CourseName          string         `json:"course_name"` // Added for display
	Semester            int            `json:"semester"`
	Batch               string         `json:"batch"`
	CourseType          string         `json:"course_type"` // theory or theory_with_lab
	AcademicYear        string         `json:"academic_year"`
	CurrentSemesterType string         `json:"current_semester_type"` // odd or even
	Status              string         `json:"status"`
	Priority            int            `json:"priority"`
	IsActive            bool           `json:"is_active"`
	CreatedAt           sql.NullTime   `json:"created_at"` // Changed to sql.NullTime
	UpdatedAt           sql.NullTime   `json:"updated_at"` // Changed to sql.NullTime
	CreatedBy           sql.NullInt64  `json:"created_by"`
	ApprovedBy          sql.NullInt64  `json:"approved_by"`
	ApprovedAt          sql.NullTime   `json:"approved_at"`
	Notes               sql.NullString `json:"notes"`
}

// TeacherAllocationSummary shows teacher's allocation vs actual selections
type TeacherAllocationSummary struct {
	TeacherID           int                 `json:"teacher_id"`
	TeacherName         string              `json:"teacher_name"`
	FacultyID           string              `json:"faculty_id"`
	Batch               string              `json:"batch"`
	AcademicYear        string              `json:"academic_year"`
	CurrentSemesterType string              `json:"current_semester_type"`
	TypeSummaries       []CourseTypeSummary `json:"type_summaries"`
	AllocationStatus    string              `json:"allocation_status"` // Complete, Incomplete, Over-allocated
}

type CourseTypeSummary struct {
	CourseTypeID int    `json:"course_type_id"`
	TypeName     string `json:"type_name"`
	Allocated    int    `json:"allocated"`
	Selected     int    `json:"selected"`
	Remaining    int    `json:"remaining"`
}

// TeacherCoursePreferencesResponse includes lock status and preference list
type TeacherCoursePreferencesResponse struct {
	Locked      bool                      `json:"locked"` // true if preferences already submitted for next semester
	Preferences []TeacherCoursePreference `json:"preferences"`
}

// GetTeacherAllocationSummary returns the allocation summary for a teacher
func GetTeacherAllocationSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teacherID := vars["teacher_id"]
	
	// Optional query params
	batch := r.URL.Query().Get("batch")
	academicYear := r.URL.Query().Get("academic_year")

	// Get basic teacher info
	var summary TeacherAllocationSummary
	query := `SELECT id, name, faculty_id FROM teachers WHERE id = ?`
	err := db.DB.QueryRow(query, teacherID).Scan(&summary.TeacherID, &summary.TeacherName, &summary.FacultyID)
	if err != nil {
		log.Printf("Error fetching teacher info: %v", err)
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}

	summary.Batch = batch
	summary.AcademicYear = academicYear
	summary.CurrentSemesterType = "odd" // Default or fetch from somewhere

	// Fetch all course types and their limits/selections for this teacher
	// Use faculty_id for teacher_course_limits (varchar) and id for teacher_course_preferences (int)
	typeQuery := `
		SELECT 
			ct.id, 
			ct.course_type, 
			COALESCE(tcl.max_count, 0) as allocated,
			(SELECT COUNT(*) FROM teacher_course_preferences tcp 
			 WHERE tcp.teacher_id = ? AND tcp.course_type = ct.id
			 AND tcp.status != 'rejected') as selected
		FROM course_type ct
		LEFT JOIN teacher_course_limits tcl ON ct.id = tcl.course_type_id AND tcl.teacher_id = ?
		ORDER BY ct.id
	`
	rows, err := db.DB.Query(typeQuery, teacherID, summary.FacultyID)
	if err != nil {
		log.Printf("Error fetching type summaries: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	allComplete := true
	anyOver := false

	for rows.Next() {
		var ts CourseTypeSummary
		if err := rows.Scan(&ts.CourseTypeID, &ts.TypeName, &ts.Allocated, &ts.Selected); err != nil {
			continue
		}
		ts.Remaining = ts.Allocated - ts.Selected
		summary.TypeSummaries = append(summary.TypeSummaries, ts)

		if ts.Selected < ts.Allocated {
			allComplete = false
		}
		if ts.Selected > ts.Allocated {
			anyOver = true
		}
	}

	if anyOver {
		summary.AllocationStatus = "Over-allocated"
	} else if allComplete {
		summary.AllocationStatus = "Complete"
	} else {
		summary.AllocationStatus = "Incomplete"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"summary": summary,
	})
}

// GetTeacherCoursePreferences returns all course preferences for a teacher
func GetTeacherCoursePreferences(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teacherID := vars["teacher_id"]

	academicYear := r.URL.Query().Get("academic_year")
	if strings.TrimSpace(academicYear) == "" {
		http.Error(w, "academic_year is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("GetTeacherCoursePreferences - teacherID: %s, academicYear: %s", teacherID, academicYear)

	// Simple lock check: does this teacher have ANY active (is_active=1) preferences?
	var lockedCount int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM teacher_course_preferences
		WHERE teacher_id = ? AND is_active = 1
	`, teacherID).Scan(&lockedCount)
	if err != nil {
		log.Printf("Error checking lock status: %v", err)
	}
	isLocked := lockedCount > 0
	log.Printf("[LOCK_CHECK] Teacher %s - active preference count: %d, is_locked: %v", teacherID, lockedCount, isLocked)

	query := `
		SELECT tcp.id, tcp.teacher_id, tcp.course_id, tcp.semester, COALESCE(tcp.batch, ''), COALESCE(ct.course_type, 'theory') as course_type, 
			   tcp.academic_year, COALESCE(tcp.current_semester_type, ''), COALESCE(tcp.status, 'approved'), COALESCE(tcp.priority, 0),
			   COALESCE(tcp.is_active, 1), COALESCE(c.course_name, '') as course_name
		FROM teacher_course_preferences tcp
		LEFT JOIN courses c ON tcp.course_id = c.course_code
		LEFT JOIN course_type ct ON tcp.course_type = ct.id
		WHERE tcp.teacher_id = ? AND tcp.is_active = 1
	`

	args := []interface{}{teacherID}

	query += " ORDER BY tcp.semester, tcp.priority"

	log.Printf("Executing query to fetch preferences for teacher %s, academic_year %s, is_locked: %v", teacherID, academicYear, isLocked)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching teacher course preferences: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	preferences := []TeacherCoursePreference{}
	for rows.Next() {
		var pref TeacherCoursePreference
		err := rows.Scan(
			&pref.ID,
			&pref.TeacherID,
			&pref.CourseID,
			&pref.Semester,
			&pref.Batch,
			&pref.CourseType,
			&pref.AcademicYear,
			&pref.CurrentSemesterType,
			&pref.Status,
			&pref.Priority,
			&pref.IsActive,
			&pref.CourseName,
		)
		if err != nil {
			log.Printf("Error scanning preference: %v", err)
			continue
		}
		preferences = append(preferences, pref)
	}

	log.Printf("Returning %d preferences for teacher %s, locked: %v", len(preferences), teacherID, isLocked)

	response := TeacherCoursePreferencesResponse{
		Locked:      isLocked,
		Preferences: preferences,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"locked":      response.Locked,
		"preferences": response.Preferences,
	})
}

// SaveTeacherCoursePreferences saves/updates teacher course selections
func SaveTeacherCoursePreferences(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		TeacherID           int      `json:"teacher_id"`
		AcademicYear        string   `json:"academic_year"`
		CurrentSemesterType string   `json:"current_semester_type"`
		Preferences         []struct {
			CourseID   string `json:"course_id"`
			Semester   int    `json:"semester"`
			Batch      string `json:"batch"`
			CourseType string `json:"course_type"` // Changed to string (theory, theory_with_lab, lab)
			Priority   int    `json:"priority"`
		} `json:"preferences"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("=== SaveTeacherCoursePreferences Request ===")
	log.Printf("TeacherID: %d", requestData.TeacherID)
	log.Printf("AcademicYear: %s", requestData.AcademicYear)
	log.Printf("CurrentSemesterType: %s", requestData.CurrentSemesterType)
	log.Printf("Number of preferences: %d", len(requestData.Preferences))
	for i, pref := range requestData.Preferences {
		log.Printf("  Pref %d: CourseID=%s, Semester=%d, Batch=%s, CourseType=%s, Priority=%d", 
			i+1, pref.CourseID, pref.Semester, pref.Batch, pref.CourseType, pref.Priority)
	}
	log.Printf("==========================================")

	requestData.AcademicYear = strings.TrimSpace(requestData.AcademicYear)
	if requestData.AcademicYear == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "academic_year is required",
		})
		return
	}

	var err error

	// Check for pending appeals - teacher cannot modify preferences until appeal is resolved
	var hasPendingAppeal bool
	err = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM teacher_course_appeal 
			WHERE faculty_id = ? AND appeal_status = 0
		)
	`, requestData.TeacherID).Scan(&hasPendingAppeal)

	if err != nil {
		log.Printf("Error checking pending appeals: %v", err)
		http.Error(w, "Error checking appeal status", http.StatusInternalServerError)
		return
	}

	if hasPendingAppeal {
		log.Printf("Teacher %d has pending appeal - cannot modify preferences", requestData.TeacherID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You cannot modify your course selections while an appeal is pending. Please wait for HR decision on your appeal.",
		})
		return
	}

	// Get current_semester_type from the active tracking window
	var currentSemesterType string
	err = db.DB.QueryRow(`
		SELECT COALESCE(LOWER(TRIM(current_semester_type)), 'even')
		FROM teacher_course_tracking
		WHERE is_active = 1
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&currentSemesterType)
	if err != nil {
		log.Printf("Error fetching semester type from tracking: %v", err)
		currentSemesterType = "even"
	}

	// Determine academic year to store preferences under:
	// If current semester is 'odd', choices are for next year's ODD semester -> shift year forward
	saveAcademicYear := requestData.AcademicYear
	if currentSemesterType == "odd" {
		parts := strings.Split(requestData.AcademicYear, "-")
		if len(parts) == 2 {
			var year1, year2 int
			fmt.Sscanf(parts[0], "%d", &year1)
			fmt.Sscanf(parts[1], "%d", &year2)
			saveAcademicYear = fmt.Sprintf("%d-%d", year1+1, year2+1)
		}
	}

	log.Printf("Current tracking semester: %s, Saving preferences under academic year: %s", currentSemesterType, saveAcademicYear)

	// Check if teacher has ALREADY submitted active preferences
	var existingCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM teacher_course_preferences 
		WHERE teacher_id = ? AND is_active = 1
	`, requestData.TeacherID).Scan(&existingCount)

	if err != nil {
		log.Printf("Error checking existing active preferences: %v", err)
		http.Error(w, "Error validating existing submission", http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		log.Printf("Teacher %d already has active preferences - blocking duplicate save", requestData.TeacherID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "You have already submitted your course preferences for this semester window.",
			"locked":  true,
		})
		return
	}

	// Check if selection window is open
	var windowStart, windowEnd sql.NullTime
	var isActive int
	err = db.DB.QueryRow(`
		SELECT window_start, window_end, COALESCE(is_active, 0)
		FROM teacher_course_tracking
		WHERE is_active = 1
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&windowStart, &windowEnd, &isActive)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching window dates: %v", err)
		http.Error(w, "Error validating selection window", http.StatusInternalServerError)
		return
	}

	// Check if window is active (is_active = 1)
	if isActive != 1 {
		log.Printf("[SAVE_VALIDATION] Window is_active flag is off (value: %d) - cannot save", isActive)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Selection window is not active",
			"locked":  true,
		})
		return
	}

	// Check if window dates are configured
	if !windowStart.Valid || !windowEnd.Valid {
		log.Printf("[SAVE_VALIDATION] Window dates not set - cannot save")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Selection window is not configured",
			"locked":  true,
		})
		return
	}

	// Check if current time is within window range
	// Extend window_end to end-of-day (23:59:59) in the stored time's own timezone,
	// so a date-only value like "2026-03-04 00:00:00 IST" becomes "2026-03-04 23:59:59 IST"
	now := time.Now()
	weLoc := windowEnd.Time.Location()
	weYear, weMonth, weDay := windowEnd.Time.In(weLoc).Date()
	windowEndEOD := time.Date(weYear, weMonth, weDay, 23, 59, 59, 999999999, weLoc)
	if now.Before(windowStart.Time) || now.After(windowEndEOD) {
		log.Printf("[SAVE_VALIDATION] Outside window dates - now: %v, start: %v, end (EOD): %v", now, windowStart.Time, windowEndEOD)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "The selection window is currently closed",
			"locked":  true,
		})
		return
	}
	log.Printf("[SAVE_VALIDATION] Window is active and dates valid - now: %v (between %v and %v)", now, windowStart.Time, windowEndEOD)

	// Convert numeric teacher ID to alphanumeric faculty_id for teacher_course_limits query
	var facultyID string
	err = db.DB.QueryRow("SELECT faculty_id FROM teachers WHERE id = ?", requestData.TeacherID).Scan(&facultyID)
	if err != nil {
		log.Printf("Error fetching faculty_id for teacher %d: %v", requestData.TeacherID, err)
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}

	// Build course type mappings (name string <-> numeric ID)
	courseTypeMap := make(map[string]int) // course_type string -> ID
	typeNameMap := make(map[int]string)   // ID -> course_type string
	typeRows, err := db.DB.Query("SELECT id, course_type FROM course_type")
	if err != nil {
		log.Printf("Error fetching course types: %v", err)
		http.Error(w, "Failed to fetch course type mappings", http.StatusInternalServerError)
		return
	}
	defer typeRows.Close()
	
	for typeRows.Next() {
		var id int
		var typeName string
		if err := typeRows.Scan(&id, &typeName); err == nil {
			courseTypeMap[typeName] = id
			typeNameMap[id] = typeName
		}
	}

	// Validate teacher allocation dynamically - using alphanumeric faculty_id (VARCHAR)
	rows, err := db.DB.Query(`
		SELECT ct.id, ct.course_type, COALESCE(tcl.max_count, 0)
		FROM course_type ct
		LEFT JOIN teacher_course_limits tcl ON ct.id = tcl.course_type_id AND tcl.teacher_id = ?
	`, facultyID)

	if err != nil {
		log.Printf("Error fetching teacher allocation: %v", err)
		http.Error(w, "Error validating allocation", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	limitMap := make(map[int]int) // Changed to map[int]int for course_type_id
	for rows.Next() {
		var typeID int
		var name string
		var max int
		if err := rows.Scan(&typeID, &name, &max); err == nil {
			limitMap[typeID] = max
		}
	}

	// Count selections by type ID
	selectionCounts := make(map[int]int)
	for _, pref := range requestData.Preferences {
		if courseTypeID, exists := courseTypeMap[pref.CourseType]; exists {
			selectionCounts[courseTypeID]++
		}
	}

	// Validate counts
	for typeID, count := range selectionCounts {
		maxAllowed, exists := limitMap[typeID]
		typeName := typeNameMap[typeID]
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Invalid course type ID: %d", typeID),
			})
			return
		}
		if count > maxAllowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("%s subject count (%d) exceeds allocation (%d)", typeName, count, maxAllowed),
			})
			return
		}
	}

	// Begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert new preferences (no delete since we now prevent resubmission)
	log.Printf("Inserting %d new preferences for teacher %d", len(requestData.Preferences), requestData.TeacherID)

	// Insert new preferences
	stmt, err := tx.Prepare(`
		INSERT INTO teacher_course_preferences 
		(teacher_id, course_id, semester, batch, course_type, academic_year, current_semester_type, status, priority, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'approved', ?, 1, NOW(), NOW())
	`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	log.Printf("Inserting %d preferences for teacher %d", len(requestData.Preferences), requestData.TeacherID)
	
	// Validate and prepare preference data with converted course_type IDs
	var prefsToInsert []map[string]interface{}
	for i, pref := range requestData.Preferences {
		courseTypeID, exists := courseTypeMap[pref.CourseType]
		if !exists {
			log.Printf("Invalid course_type '%s' for course %s (preference %d)", pref.CourseType, pref.CourseID, i+1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Invalid course type '%s'. Valid types are: theory, theory_with_lab, lab", pref.CourseType),
			})
			return
		}
		
		prefsToInsert = append(prefsToInsert, map[string]interface{}{
			"course_id":   pref.CourseID,
			"semester":    pref.Semester,
			"batch":       pref.Batch,
			"course_type": courseTypeID, // Converted to numeric ID
			"priority":    pref.Priority,
		})
	}
	
	for i, prefData := range prefsToInsert {
		log.Printf("Inserting preference %d: CourseID=%s, Semester=%d, Batch=%s, CourseTypeID=%d, Priority=%d", 
			i+1, prefData["course_id"], prefData["semester"], prefData["batch"], prefData["course_type"], prefData["priority"])
			
		result, err := stmt.Exec(
			requestData.TeacherID,
			prefData["course_id"],
			prefData["semester"],
			prefData["batch"],
			prefData["course_type"],
			saveAcademicYear,
			currentSemesterType,
			prefData["priority"],
		)
		if err != nil {
			log.Printf("Error inserting preference: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		insertID, _ := result.LastInsertId()
		log.Printf("Successfully inserted preference with ID: %d", insertID)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✓ Transaction committed successfully - %d preferences saved (is_active=1 for new records)", len(requestData.Preferences))

	// After successful save, preferences are now locked for the next semester type
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Course preferences saved and locked successfully",
		"locked":  true,
	})
}
