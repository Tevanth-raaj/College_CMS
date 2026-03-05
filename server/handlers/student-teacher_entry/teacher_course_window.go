package studentteacher

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"strings"
	
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
	var currentSemesterType sql.NullString

	var hasIsActive int
	err := db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = 'teacher_course_tracking'
		AND COLUMN_NAME = 'is_active'
	`).Scan(&hasIsActive)
	if err != nil {
		log.Printf("Error checking is_active column on teacher_course_tracking: %v", err)
		hasIsActive = 0
	}

	if hasIsActive > 0 {
		err = db.DB.QueryRow(`
			SELECT window_start, window_end, COALESCE(current_semester_type, 'even')
			FROM teacher_course_tracking
			WHERE academic_year = ?
			AND is_active = 1
			AND window_start IS NOT NULL
			AND window_end IS NOT NULL
			AND DATE(window_start) <= CURDATE()
			AND DATE(window_end) >= CURDATE()
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		`, academicYear).Scan(&windowStart, &windowEnd, &currentSemesterType)

		if err == sql.ErrNoRows {
			err = db.DB.QueryRow(`
				SELECT window_start, window_end, COALESCE(current_semester_type, 'even')
				FROM teacher_course_tracking
				WHERE academic_year = ?
				AND is_active = 1
				ORDER BY updated_at DESC, id DESC
				LIMIT 1
			`, academicYear).Scan(&windowStart, &windowEnd, &currentSemesterType)
		}
	} else {
		err = db.DB.QueryRow(`
			SELECT window_start, window_end, COALESCE(current_semester_type, 'even')
			FROM teacher_course_tracking
			WHERE academic_year = ?
			AND window_start IS NOT NULL
			AND window_end IS NOT NULL
			AND DATE(window_start) <= CURDATE()
			AND DATE(window_end) >= CURDATE()
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		`, academicYear).Scan(&windowStart, &windowEnd, &currentSemesterType)

		if err == sql.ErrNoRows {
			err = db.DB.QueryRow(`
				SELECT window_start, window_end, COALESCE(current_semester_type, 'even')
				FROM teacher_course_tracking
				WHERE academic_year = ?
				ORDER BY updated_at DESC, id DESC
				LIMIT 1
			`, academicYear).Scan(&windowStart, &windowEnd, &currentSemesterType)
		}
	}

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

	semesterType := "even"
	if currentSemesterType.Valid && strings.TrimSpace(currentSemesterType.String) != "" {
		semesterType = strings.ToLower(strings.TrimSpace(currentSemesterType.String))
	}

	// Calculate next semester type
	var nextSemesterType string
	if semesterType == "odd" {
		nextSemesterType = "even"
	} else {
		nextSemesterType = "odd"
	}

	response := map[string]interface{}{
		"academic_year":         academicYear,
		"configured":            windowStart.Valid && windowEnd.Valid,
		"current_semester_type": semesterType,
		"next_semester_type":    nextSemesterType,
		"is_active_column":      hasIsActive > 0,
	}

	if windowStart.Valid {
		response["window_start"] = windowStart.Time.Local().Format("2006-01-02")
	} else {
		response["window_start"] = nil
	}

	if windowEnd.Valid {
		response["window_end"] = windowEnd.Time.Local().Format("2006-01-02")
	} else {
		response["window_end"] = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
