package studentteacher

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	
	"github.com/gorilla/mux"
)

// GetTeacherCourseWindow returns the window period from teacher_course_tracking
func GetTeacherCourseWindow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	academicYear := vars["academic_year"]

	if academicYear == "" {
		http.Error(w, "Academic year is required", http.StatusBadRequest)
		return
	}

	var windowStart, windowEnd sql.NullTime
	err := db.DB.QueryRow(`
		SELECT window_start, window_end
		FROM teacher_course_tracking
		WHERE academic_year = ?
		LIMIT 1
	`, academicYear).Scan(&windowStart, &windowEnd)

	if err == sql.ErrNoRows {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"academic_year":       academicYear,
			"window_start":        nil,
			"window_end":          nil,
			"configured":          false,
			"current_semester_type": nil,
			"next_semester_type":  nil,
		})
		return
	}

	if err != nil {
		log.Printf("Error fetching window period: %v", err)
		http.Error(w, "Failed to fetch window period", http.StatusInternalServerError)
		return
	}

	// Get current semester type - default to "even" or fetch from teacher_course_preferences if it exists
	var currentSemesterType string
	err2 := db.DB.QueryRow(`
		SELECT COALESCE(current_semester_type, 'even')
		FROM teacher_course_preferences
		WHERE academic_year = ?
		LIMIT 1
	`, academicYear).Scan(&currentSemesterType)

	if err2 != nil {
		log.Printf("Warning: Could not fetch current semester type, defaulting to 'even': %v", err2)
		currentSemesterType = "even"
	}

	// Calculate next semester type
	var nextSemesterType string
	if currentSemesterType == "even" {
		nextSemesterType = "odd"
	} else {
		nextSemesterType = "even"
	}

	response := map[string]interface{}{
		"academic_year":         academicYear,
		"configured":            windowStart.Valid && windowEnd.Valid,
		"current_semester_type": currentSemesterType,
		"next_semester_type":    nextSemesterType,
	}

	if windowStart.Valid {
		response["window_start"] = windowStart.Time.Format("2006-01-02")
	} else {
		response["window_start"] = nil
	}

	if windowEnd.Valid {
		response["window_end"] = windowEnd.Time.Format("2006-01-02")
	} else {
		response["window_end"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
