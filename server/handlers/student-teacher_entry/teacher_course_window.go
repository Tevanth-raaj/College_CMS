package studentteacher

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"server/db"
	"strings"
	
	"github.com/gorilla/mux"
)

type TeacherCourseWindowPayload struct {
	AcademicYear        string `json:"academic_year"`
	CurrentSemesterType string `json:"current_semester_type"`
	WindowStart         string `json:"window_start"`
	WindowEnd           string `json:"window_end"`
}

// GetTeacherCourseWindows returns all configured teacher selection windows (admin view)
func GetTeacherCourseWindows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	academicYear := strings.TrimSpace(r.URL.Query().Get("academic_year"))

	query := `
		SELECT
			id,
			academic_year,
			COALESCE(current_semester_type, 'even') AS current_semester_type,
			COALESCE(DATE_FORMAT(window_start, '%Y-%m-%d'), '') AS window_start,
			COALESCE(DATE_FORMAT(window_end, '%Y-%m-%d'), '') AS window_end,
			COALESCE(is_active, 1) AS is_active
		FROM teacher_course_tracking
	`
	args := []interface{}{}
	if academicYear != "" {
		query += ` WHERE academic_year = ?`
		args = append(args, academicYear)
	}
	query += ` ORDER BY id DESC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching teacher course windows: %v", err)
		http.Error(w, "Failed to fetch teacher course windows", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var year, semesterType, start, end string
		var isActive int
		if err := rows.Scan(&id, &year, &semesterType, &start, &end, &isActive); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":                    id,
			"academic_year":         year,
			"current_semester_type": strings.ToLower(strings.TrimSpace(semesterType)),
			"window_start":          start,
			"window_end":            end,
			"is_active":             isActive,
		})
	}

	json.NewEncoder(w).Encode(results)
}

// CreateTeacherCourseWindow creates a new teacher elective selection window.
func CreateTeacherCourseWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload TeacherCourseWindowPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	payload.AcademicYear = strings.TrimSpace(payload.AcademicYear)
	payload.CurrentSemesterType = strings.ToLower(strings.TrimSpace(payload.CurrentSemesterType))
	payload.WindowStart = strings.TrimSpace(payload.WindowStart)
	payload.WindowEnd = strings.TrimSpace(payload.WindowEnd)

	if payload.AcademicYear == "" || payload.WindowStart == "" || payload.WindowEnd == "" {
		http.Error(w, "academic_year, window_start, and window_end are required", http.StatusBadRequest)
		return
	}
	if payload.CurrentSemesterType != "odd" && payload.CurrentSemesterType != "even" {
		payload.CurrentSemesterType = "even"
	}

	result, err := db.DB.Exec(`
		INSERT INTO teacher_course_tracking
		(academic_year, window_start, window_end, current_semester_type, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, 1, NOW(), NOW())
	`, payload.AcademicYear, payload.WindowStart, payload.WindowEnd, payload.CurrentSemesterType)
	if err != nil {
		log.Printf("Error creating teacher course window: %v", err)
		http.Error(w, "Failed to create teacher course window", http.StatusInternalServerError)
		return
	}

	insertID, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      insertID,
		"message": "Teacher elective selection window created",
	})
}

// UpdateTeacherCourseWindow updates an existing teacher elective selection window.
// Important: update always reactivates the window by setting is_active = 1.
func UpdateTeacherCourseWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	windowIDText := mux.Vars(r)["id"]
	windowID, err := strconv.Atoi(windowIDText)
	if err != nil || windowID <= 0 {
		http.Error(w, "Invalid window id", http.StatusBadRequest)
		return
	}

	var payload TeacherCourseWindowPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	payload.AcademicYear = strings.TrimSpace(payload.AcademicYear)
	payload.CurrentSemesterType = strings.ToLower(strings.TrimSpace(payload.CurrentSemesterType))
	payload.WindowStart = strings.TrimSpace(payload.WindowStart)
	payload.WindowEnd = strings.TrimSpace(payload.WindowEnd)

	if payload.AcademicYear == "" || payload.WindowStart == "" || payload.WindowEnd == "" {
		http.Error(w, "academic_year, window_start, and window_end are required", http.StatusBadRequest)
		return
	}
	if payload.CurrentSemesterType != "odd" && payload.CurrentSemesterType != "even" {
		payload.CurrentSemesterType = "even"
	}

	result, err := db.DB.Exec(`
		UPDATE teacher_course_tracking
		SET academic_year = ?,
			window_start = ?,
			window_end = ?,
			current_semester_type = ?,
			is_active = 1,
			updated_at = NOW()
		WHERE id = ?
	`, payload.AcademicYear, payload.WindowStart, payload.WindowEnd, payload.CurrentSemesterType, windowID)
	if err != nil {
		log.Printf("Error updating teacher course window %d: %v", windowID, err)
		http.Error(w, "Failed to update teacher course window", http.StatusInternalServerError)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Teacher course window not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Teacher elective selection window updated and reactivated",
	})
}

// DeleteTeacherCourseWindow deletes a configured teacher elective selection window.
func DeleteTeacherCourseWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	windowIDText := mux.Vars(r)["id"]
	windowID, err := strconv.Atoi(windowIDText)
	if err != nil || windowID <= 0 {
		http.Error(w, "Invalid window id", http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec(`
		DELETE FROM teacher_course_tracking
		WHERE id = ?
	`, windowID)
	if err != nil {
		log.Printf("Error deleting teacher course window %d: %v", windowID, err)
		http.Error(w, "Failed to delete teacher course window", http.StatusInternalServerError)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		http.Error(w, "Teacher course window not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Teacher elective selection window deleted",
	})
}

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
