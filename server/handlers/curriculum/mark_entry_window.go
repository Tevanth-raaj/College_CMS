package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server/db"
	"server/models"
)

const markEntryTimeLayout = "2006-01-02T15:04"

// parseDateTime attempts to parse datetime in multiple formats (ISO 8601 with timezone, or simple local format)
func parseDateTime(dateStr string) (time.Time, error) {
	// Try RFC3339 first (ISO 8601 with timezone: "2006-01-02T15:04:05Z07:00")
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.UTC(), nil
	}

	// Try RFC3339 without seconds
	if t, err := time.Parse("2006-01-02T15:04Z07:00", dateStr); err == nil {
		return t.UTC(), nil
	}

	// Fallback to local time format for backward compatibility
	t, err := time.ParseInLocation(markEntryTimeLayout, dateStr, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// GetMarkEntryWindow returns a window rule matching the exact scope.
func GetMarkEntryWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	teacherID := strings.TrimSpace(r.URL.Query().Get("teacher_id"))
	departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id"))
	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	courseIDStr := strings.TrimSpace(r.URL.Query().Get("course_id"))

	var departmentID *int
	if departmentIDStr != "" {
		value, err := strconv.Atoi(departmentIDStr)
		if err != nil || value <= 0 {
			http.Error(w, "Invalid department ID", http.StatusBadRequest)
			return
		}
		departmentID = &value
	}

	var semester *int
	if semesterStr != "" {
		value, err := strconv.Atoi(semesterStr)
		if err != nil || value <= 0 {
			http.Error(w, "Invalid semester", http.StatusBadRequest)
			return
		}
		semester = &value
	}

	var courseID *int
	if courseIDStr != "" {
		value, err := strconv.Atoi(courseIDStr)
		if err != nil || value <= 0 {
			http.Error(w, "Invalid course ID", http.StatusBadRequest)
			return
		}
		courseID = &value
	}

	if teacherID == "" && departmentID == nil && semester == nil && courseID == nil {
		http.Error(w, "At least one scope field is required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	query := `
		SELECT id, teacher_id, department_id, semester, course_id, start_at, end_at, enabled
		FROM mark_entry_windows
		WHERE 1 = 1`
	args := []interface{}{}

	if teacherID != "" {
		query += " AND teacher_id = ?"
		args = append(args, teacherID)
	} else {
		query += " AND teacher_id IS NULL"
	}

	if departmentID != nil {
		query += " AND department_id = ?"
		args = append(args, *departmentID)
	} else {
		query += " AND department_id IS NULL"
	}

	if semester != nil {
		query += " AND semester = ?"
		args = append(args, *semester)
	} else {
		query += " AND semester IS NULL"
	}

	if courseID != nil {
		query += " AND course_id = ?"
		args = append(args, *courseID)
	} else {
		query += " AND course_id IS NULL"
	}

	query += " ORDER BY updated_at DESC LIMIT 1"

	var window models.MarkEntryWindow
	var startAt time.Time
	var endAt time.Time
	var enabledInt int
	var teacherIDNull sql.NullString
	var departmentIDNull sql.NullInt64
	var semesterNull sql.NullInt64
	var courseIDNull sql.NullInt64

	err := database.QueryRow(query, args...).Scan(
		&window.ID,
		&teacherIDNull,
		&departmentIDNull,
		&semesterNull,
		&courseIDNull,
		&startAt,
		&endAt,
		&enabledInt,
	)
	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(nil)
		return
	}
	if err != nil {
		log.Printf("Error fetching mark entry window: %v", err)
		http.Error(w, "Failed to fetch mark entry window", http.StatusInternalServerError)
		return
	}

	if teacherIDNull.Valid {
		value := teacherIDNull.String
		window.TeacherID = &value
	}
	if departmentIDNull.Valid {
		value := int(departmentIDNull.Int64)
		window.DepartmentID = &value
	}
	if semesterNull.Valid {
		value := int(semesterNull.Int64)
		window.Semester = &value
	}
	if courseIDNull.Valid {
		value := int(courseIDNull.Int64)
		window.CourseID = &value
	}
	window.StartAt = startAt.Format(markEntryTimeLayout)
	window.EndAt = endAt.Format(markEntryTimeLayout)
	window.Enabled = enabledInt == 1

	// Load component IDs if any
	componentRows, err := database.Query(`
		SELECT assessment_component_id
		FROM mark_entry_window_components
		WHERE window_id = ?
	`, window.ID)
	if err == nil {
		defer componentRows.Close()
		for componentRows.Next() {
			var componentID int
			if err := componentRows.Scan(&componentID); err == nil {
				window.ComponentIDs = append(window.ComponentIDs, componentID)
			}
		}
	}

	json.NewEncoder(w).Encode(window)
}

// SaveMarkEntryWindow creates or updates a window rule for a scope.
func SaveMarkEntryWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req models.MarkEntryWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if (req.TeacherID == nil || strings.TrimSpace(*req.TeacherID) == "") && req.DepartmentID == nil && req.Semester == nil && req.CourseID == nil {
		http.Error(w, "At least one scope field is required", http.StatusBadRequest)
		return
	}

	startAt, err := parseDateTime(req.StartAt)
	if err != nil {
		log.Printf("Error parsing start date '%s': %v", req.StartAt, err)
		http.Error(w, "Invalid start date", http.StatusBadRequest)
		return
	}

	endAt, err := parseDateTime(req.EndAt)
	if err != nil {
		log.Printf("Error parsing end date '%s': %v", req.EndAt, err)
		http.Error(w, "Invalid end date", http.StatusBadRequest)
		return
	}

	if !endAt.After(startAt) {
		http.Error(w, "End date must be after start date", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Only delete non-expired windows for the same scope
	deleteQuery := "DELETE FROM mark_entry_windows WHERE end_at > UTC_TIMESTAMP() AND 1 = 1"
	deleteArgs := []interface{}{}

	if req.TeacherID != nil && strings.TrimSpace(*req.TeacherID) != "" {
		deleteQuery += " AND teacher_id = ?"
		deleteArgs = append(deleteArgs, strings.TrimSpace(*req.TeacherID))
	} else {
		deleteQuery += " AND teacher_id IS NULL"
	}

	if req.DepartmentID != nil {
		deleteQuery += " AND department_id = ?"
		deleteArgs = append(deleteArgs, *req.DepartmentID)
	} else {
		deleteQuery += " AND department_id IS NULL"
	}

	if req.Semester != nil {
		deleteQuery += " AND semester = ?"
		deleteArgs = append(deleteArgs, *req.Semester)
	} else {
		deleteQuery += " AND semester IS NULL"
	}

	if req.CourseID != nil {
		deleteQuery += " AND course_id = ?"
		deleteArgs = append(deleteArgs, *req.CourseID)
	} else {
		deleteQuery += " AND course_id IS NULL"
	}

	result, err := database.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		log.Printf("Error clearing mark entry window: %v", err)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Deleted %d non-expired window(s) for scope", rowsAffected)

	teacherValue := sql.NullString{}
	if req.TeacherID != nil && strings.TrimSpace(*req.TeacherID) != "" {
		teacherValue = sql.NullString{String: strings.TrimSpace(*req.TeacherID), Valid: true}
	}

	departmentValue := sql.NullInt64{}
	if req.DepartmentID != nil {
		departmentValue = sql.NullInt64{Int64: int64(*req.DepartmentID), Valid: true}
	}

	semesterValue := sql.NullInt64{}
	if req.Semester != nil {
		semesterValue = sql.NullInt64{Int64: int64(*req.Semester), Valid: true}
	}

	courseValue := sql.NullInt64{}
	if req.CourseID != nil {
		courseValue = sql.NullInt64{Int64: int64(*req.CourseID), Valid: true}
	}

	enabledValue := 0
	if req.Enabled {
		enabledValue = 1
	}

	result, err = database.Exec(`
		INSERT INTO mark_entry_windows
		(teacher_id, department_id, semester, course_id, start_at, end_at, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, teacherValue, departmentValue, semesterValue, courseValue, startAt, endAt, enabledValue)
	if err != nil {
		log.Printf("Error saving mark entry window: %v", err)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	// Save component associations if specified
	if len(req.ComponentIDs) > 0 {
		windowID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting window ID: %v", err)
			http.Error(w, "Failed to save window components", http.StatusInternalServerError)
			return
		}

		for _, componentID := range req.ComponentIDs {
			_, err := database.Exec(`
				INSERT INTO mark_entry_window_components (window_id, assessment_component_id)
				VALUES (?, ?)
			`, windowID, componentID)
			if err != nil {
				log.Printf("Error saving window component: %v", err)
				// Continue saving other components even if one fails
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Mark entry window saved"})
}

func resolveMarkEntryWindow(courseID int, teacherID string) (bool, int, []int, error) {
	database := db.DB
	if database == nil {
		return false, 0, nil, sql.ErrConnDone
	}

	// Convert numeric teacher ID to faculty_id if needed
	// All allocation tables (teacher_course_allocation, department_teachers, mark_entry_windows) use faculty_id
	var facultyID string
	err := database.QueryRow(`SELECT faculty_id FROM teachers WHERE id = ? OR faculty_id = ?`, teacherID, teacherID).Scan(&facultyID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error looking up faculty_id for teacher %s: %v", teacherID, err)
		// Not a teacher, might be a user - continue
		facultyID = teacherID
	} else if err == nil {
		log.Printf("Resolved teacherID '%s' to faculty_id '%s' for mark entry window resolution", teacherID, facultyID)
	} else {
		// No teacher found, use original ID (might be username)
		facultyID = teacherID
	}

	// Try to lookup numeric user ID from username (for users)
	var numericUserID sql.NullInt64
	err = database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, facultyID).Scan(&numericUserID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error looking up user ID for %s: %v", facultyID, err)
		return false, 0, nil, err
	}

	var departmentID sql.NullInt64
	err = database.QueryRow(`
		SELECT department_id
		FROM department_teachers
		WHERE teacher_id = ? AND status = 1
		ORDER BY id DESC
		LIMIT 1
	`, facultyID).Scan(&departmentID)
	if err != nil && err != sql.ErrNoRows {
		return false, 0, nil, err
	}

	var semester sql.NullInt64
	var cardType string
	err = database.QueryRow(`
		SELECT nc.semester_number, COALESCE(nc.card_type, 'semester')
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		WHERE cc.course_id = ?
		LIMIT 1
	`, courseID).Scan(&semester, &cardType)
	if err != nil && err != sql.ErrNoRows {
		return false, 0, nil, err
	}

	// For non-semester cards (vertical, elective, open_elective, etc.),
	// skip semester matching since these courses aren't tied to a fixed semester
	if cardType != "semester" {
		log.Printf("Course %d is on a '%s' card — skipping semester check for window matching", courseID, cardType)
		semester = sql.NullInt64{Valid: false}
	}

	// DEBUG: Log what we're matching against
	log.Printf("Resolving window for courseID=%d, original teacherID=%s, facultyID=%s, numericUserID=%v, departmentID=%v, semester=%v",
		courseID, teacherID, facultyID, numericUserID, departmentID, semester)

	query := `
		SELECT id, start_at, end_at, enabled
		FROM mark_entry_windows
		WHERE ((teacher_id IS NULL AND user_id IS NULL) OR teacher_id = ? OR user_id = ?)
		  AND (course_id IS NULL OR course_id = ?)
		  AND (? IS NULL OR department_id IS NULL OR department_id = 0 OR department_id = ?)
		  AND (? IS NULL OR semester IS NULL OR semester = 0 OR semester = ?)
		ORDER BY
		  (teacher_id IS NOT NULL) DESC,
		  (user_id IS NOT NULL) DESC,
		  (course_id IS NOT NULL) DESC,
		  (department_id IS NOT NULL) DESC,
		  (semester IS NOT NULL) DESC,
		  updated_at DESC
		LIMIT 25
	`

	deptValue := interface{}(nil)
	if departmentID.Valid {
		deptValue = departmentID.Int64
	}

	semValue := interface{}(nil)
	if semester.Valid {
		semValue = semester.Int64
	}

	userIDValue := interface{}(nil)
	if numericUserID.Valid {
		userIDValue = numericUserID.Int64
	}

	rows, rowErr := database.Query(query, facultyID, userIDValue, courseID, deptValue, deptValue, semValue, semValue)
	if rowErr != nil {
		if rowErr == sql.ErrNoRows {
			log.Printf("No matching window rule found")
			return false, 0, nil, nil
		}
		return false, 0, nil, rowErr
	}
	defer rows.Close()

	nowLocal := time.Now()
	nowUTC := nowLocal.UTC()

	var windowID int
	var startAt time.Time
	var endAt time.Time
	var enabledInt int
	found := false
	for rows.Next() {
		if err := rows.Scan(&windowID, &startAt, &endAt, &enabledInt); err != nil {
			return false, 0, nil, err
		}

		if enabledInt != 1 {
			continue
		}

		inLocal := !nowLocal.Before(startAt) && !nowLocal.After(endAt)
		inUTC := !nowUTC.Before(startAt.UTC()) && !nowUTC.After(endAt.UTC())
		if inLocal || inUTC {
			found = true
			break
		}
	}

	if !found {
		log.Printf("No matching window rule found")
		return false, 0, nil, nil
	}

	log.Printf("Found active window: id=%d, start=%s, end=%s, now=%s",
		windowID, startAt.Format("2006-01-02 15:04:05"), endAt.Format("2006-01-02 15:04:05"), nowLocal.Format("2006-01-02 15:04:05"))

	// Load allowed component IDs for this window (empty = all allowed)
	var allowedComponents []int
	componentRows, err := database.Query(`
		SELECT assessment_component_id
		FROM mark_entry_window_components
		WHERE window_id = ?
	`, windowID)
	if err == nil {
		defer componentRows.Close()
		for componentRows.Next() {
			var componentID int
			if err := componentRows.Scan(&componentID); err == nil {
				allowedComponents = append(allowedComponents, componentID)
			}
		}
	}

	log.Printf("Window allows components: %v (empty = all allowed)", allowedComponents)
	return true, windowID, allowedComponents, nil
}

// validateStudentPermission checks if a user has permission to enter marks for a specific student
func validateStudentPermission(userID string, studentID int, courseID int) (bool, int, error) {
	database := db.DB
	if database == nil {
		return false, 0, sql.ErrConnDone
	}

	// Try to lookup numeric user ID from username
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userID).Scan(&numericUserID)
	if err == sql.ErrNoRows {
		// User not found, no permission
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	// Check if there's an active window with student-specific permission
	query := `
		SELECT mesp.window_id
		FROM mark_entry_student_permissions mesp
		INNER JOIN mark_entry_windows mew ON mesp.window_id = mew.id
		WHERE mesp.user_id = ?
		  AND mesp.student_id = ?
		  AND (mew.course_id IS NULL OR mew.course_id = ?)
		AND mew.enabled = 1
		AND mew.start_at <= NOW()
		AND mew.end_at > NOW()
		LIMIT 1
	`

	var windowID int
	err = database.QueryRow(query, numericUserID, studentID, courseID).Scan(&windowID)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	return true, windowID, nil
}

// getAssignedStudentIDs returns the list of student IDs assigned to a user for a specific course
// userID can be either a username (for users) or faculty_id (for teachers)
func getAssignedStudentIDs(userID string, courseID int) ([]int, error) {
	database := db.DB
	if database == nil {
		return nil, sql.ErrConnDone
	}

	// Try to lookup numeric user ID from username
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userID).Scan(&numericUserID)
	if err != nil && err != sql.ErrNoRows {
		// If error is not "no rows", return the error
		log.Printf("Error looking up user ID for %s: %v", userID, err)
		return nil, err
	}

	// If user not found in users table, they might be a teacher with no student assignments
	if err == sql.ErrNoRows {
		// Return empty list - no student-specific permissions
		return []int{}, nil
	}

	query := `
		SELECT DISTINCT mesp.student_id
		FROM mark_entry_student_permissions mesp
		INNER JOIN mark_entry_windows mew ON mesp.window_id = mew.id
		WHERE mesp.user_id = ?
		  AND (mew.course_id IS NULL OR mew.course_id = ?)
		AND mew.enabled = 1
		AND mew.start_at <= NOW()
		AND mew.end_at > NOW()
	`

	rows, err := database.Query(query, numericUserID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var studentID int
		if err := rows.Scan(&studentID); err == nil {
			studentIDs = append(studentIDs, studentID)
		}
	}

	return studentIDs, nil
}

// GetAllMarkEntryWindows returns all mark entry windows for admin management
func GetAllMarkEntryWindows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Check if filtering for user-only windows
	userOnly := r.URL.Query().Get("user_only") == "true"

	var query string
	if userOnly {
		// Only get windows with user_id set (user-assigned windows)
		query = `
			SELECT 
				w.id,
				w.user_id,
				COALESCE(u.username, '') as user_username,
				w.department_id,
				COALESCE(d.department_name, '') as department_name,
				w.semester,
				w.course_id,
				COALESCE(c.course_code, '') as course_code,
				COALESCE(c.course_name, '') as course_name,
				w.start_at,
				w.end_at,
				w.enabled,
				COUNT(DISTINCT mesp.student_id) as student_count
			FROM mark_entry_windows w
			LEFT JOIN users u ON w.user_id = u.id
			LEFT JOIN departments d ON w.department_id = d.id
			LEFT JOIN courses c ON w.course_id = c.id
			LEFT JOIN mark_entry_student_permissions mesp ON w.id = mesp.window_id
			WHERE w.user_id IS NOT NULL
			GROUP BY w.id
			ORDER BY 
				CASE WHEN w.end_at > NOW() THEN 0 ELSE 1 END,
				w.start_at DESC
		`
	} else {
		// Get windows with teacher_id set (teacher windows)
		query = `
			SELECT 
				w.id,
				w.teacher_id,
				COALESCE(t.name, '') as teacher_name,
				w.department_id,
				COALESCE(d.department_name, '') as department_name,
				w.semester,
				w.course_id,
				COALESCE(c.course_code, '') as course_code,
				COALESCE(c.course_name, '') as course_name,
				w.start_at,
				w.end_at,
				w.enabled
			FROM mark_entry_windows w
			LEFT JOIN teachers t ON w.teacher_id = t.faculty_id
			LEFT JOIN departments d ON w.department_id = d.id
			LEFT JOIN courses c ON w.course_id = c.id
			WHERE w.teacher_id IS NOT NULL OR w.user_id IS NULL
			ORDER BY 
				CASE WHEN w.end_at > NOW() THEN 0 ELSE 1 END,
				w.start_at DESC
		`
	}

	rows, err := database.Query(query)
	if err != nil {
		log.Printf("Error fetching windows: %v", err)
		http.Error(w, "Failed to fetch windows", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type WindowWithDetails struct {
		ID             int       `json:"id"`
		TeacherID      *string   `json:"teacher_id,omitempty"`
		TeacherName    string    `json:"teacher_name,omitempty"`
		UserID         *int      `json:"user_id,omitempty"`
		UserUsername   string    `json:"user_username,omitempty"`
		DepartmentID   *int      `json:"department_id"`
		DepartmentName string    `json:"department_name"`
		Semester       *int      `json:"semester"`
		CourseID       *int      `json:"course_id"`
		CourseCode     string    `json:"course_code"`
		CourseName     string    `json:"course_name"`
		StartAt        time.Time `json:"start_at"`
		EndAt          time.Time `json:"end_at"`
		Enabled        bool      `json:"enabled"`
		Components     []int     `json:"component_ids"`
		StudentCount   *int      `json:"student_count,omitempty"`
	}

	var windows []WindowWithDetails
	for rows.Next() {
		var window WindowWithDetails

		if userOnly {
			var userID, studentCount sql.NullInt64
			var userUsername, deptName, courseCode, courseName sql.NullString
			var deptID, semester, courseID sql.NullInt64

			err := rows.Scan(
				&window.ID,
				&userID,
				&userUsername,
				&deptID,
				&deptName,
				&semester,
				&courseID,
				&courseCode,
				&courseName,
				&window.StartAt,
				&window.EndAt,
				&window.Enabled,
				&studentCount,
			)
			if err != nil {
				log.Printf("Error scanning user window: %v", err)
				continue
			}

			if userID.Valid {
				id := int(userID.Int64)
				window.UserID = &id
				window.UserUsername = userUsername.String
			}
			if studentCount.Valid {
				count := int(studentCount.Int64)
				window.StudentCount = &count
			}
			if deptID.Valid {
				id := int(deptID.Int64)
				window.DepartmentID = &id
				window.DepartmentName = deptName.String
			}
			if semester.Valid {
				sem := int(semester.Int64)
				window.Semester = &sem
			}
			if courseID.Valid {
				id := int(courseID.Int64)
				window.CourseID = &id
				window.CourseCode = courseCode.String
				window.CourseName = courseName.String
			}
		} else {
			var teacherID, teacherName, deptName, courseCode, courseName sql.NullString
			var deptID, semester, courseID sql.NullInt64

			err := rows.Scan(
				&window.ID,
				&teacherID,
				&teacherName,
				&deptID,
				&deptName,
				&semester,
				&courseID,
				&courseCode,
				&courseName,
				&window.StartAt,
				&window.EndAt,
				&window.Enabled,
			)
			if err != nil {
				log.Printf("Error scanning window: %v", err)
				continue
			}

			if teacherID.Valid {
				window.TeacherID = &teacherID.String
				window.TeacherName = teacherName.String
			}
			if deptID.Valid {
				id := int(deptID.Int64)
				window.DepartmentID = &id
				window.DepartmentName = deptName.String
			}
			if semester.Valid {
				sem := int(semester.Int64)
				window.Semester = &sem
			}
			if courseID.Valid {
				id := int(courseID.Int64)
				window.CourseID = &id
				window.CourseCode = courseCode.String
				window.CourseName = courseName.String
			}
		}

		// Load components for this window
		compRows, err := database.Query(`
			SELECT assessment_component_id
			FROM mark_entry_window_components
			WHERE window_id = ?
		`, window.ID)
		if err == nil {
			defer compRows.Close()
			for compRows.Next() {
				var compID int
				if err := compRows.Scan(&compID); err == nil {
					window.Components = append(window.Components, compID)
				}
			}
		}

		windows = append(windows, window)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating windows: %v", err)
		http.Error(w, "Error processing windows", http.StatusInternalServerError)
		return
	}

	if windows == nil {
		windows = []WindowWithDetails{}
	}

	json.NewEncoder(w).Encode(windows)
}

// UpdateMarkEntryWindow updates an existing mark entry window
func UpdateMarkEntryWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract window ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid window ID", http.StatusBadRequest)
		return
	}
	windowIDStr := pathParts[len(pathParts)-1]
	windowID, err := strconv.Atoi(windowIDStr)
	if err != nil {
		http.Error(w, "Invalid window ID", http.StatusBadRequest)
		return
	}

	var request models.MarkEntryWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Parse times
	startAt, err := parseDateTime(request.StartAt)
	if err != nil {
		log.Printf("Error parsing start time '%s': %v", request.StartAt, err)
		http.Error(w, "Invalid start time format", http.StatusBadRequest)
		return
	}

	endAt, err := parseDateTime(request.EndAt)
	if err != nil {
		log.Printf("Error parsing end time '%s': %v", request.EndAt, err)
		http.Error(w, "Invalid end time format", http.StatusBadRequest)
		return
	}

	// Update window
	updateQuery := `
		UPDATE mark_entry_windows
		SET start_at = ?, end_at = ?, enabled = ?
		WHERE id = ?
	`

	_, err = database.Exec(updateQuery, startAt, endAt, request.Enabled, windowID)
	if err != nil {
		log.Printf("Error updating window: %v", err)
		http.Error(w, "Failed to update window", http.StatusInternalServerError)
		return
	}

	// Update components: delete existing and insert new
	_, err = database.Exec(`DELETE FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if err != nil {
		log.Printf("Error deleting old components: %v", err)
		http.Error(w, "Failed to update components", http.StatusInternalServerError)
		return
	}

	if len(request.ComponentIDs) > 0 {
		insertCompQuery := `INSERT INTO mark_entry_window_components (window_id, assessment_component_id) VALUES (?, ?)`
		for _, componentID := range request.ComponentIDs {
			_, err := database.Exec(insertCompQuery, windowID, componentID)
			if err != nil {
				log.Printf("Error inserting component %d: %v", componentID, err)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Window updated successfully",
	})
}

// DeleteMarkEntryWindow deletes a mark entry window
func DeleteMarkEntryWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract window ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid window ID", http.StatusBadRequest)
		return
	}
	windowIDStr := pathParts[len(pathParts)-1]
	windowID, err := strconv.Atoi(windowIDStr)
	if err != nil {
		http.Error(w, "Invalid window ID", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Components will be deleted automatically due to ON DELETE CASCADE
	_, err = database.Exec(`DELETE FROM mark_entry_windows WHERE id = ?`, windowID)
	if err != nil {
		log.Printf("Error deleting window: %v", err)
		http.Error(w, "Failed to delete window", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Window deleted successfully",
	})
}

// GetMarkEntryStats returns statistics about mark entry windows and permissions
func GetMarkEntryStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	stats := make(map[string]interface{})

	// Total number of mark entry windows
	var totalWindows int
	err := database.QueryRow("SELECT COUNT(*) FROM mark_entry_windows").Scan(&totalWindows)
	if err != nil {
		log.Printf("Error counting total windows: %v", err)
		totalWindows = 0
	}
	stats["totalWindows"] = totalWindows

	// Active windows (currently enabled and within time range)
	var activeWindows int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM mark_entry_windows 
		WHERE enabled = 1 
		AND start_at <= UTC_TIMESTAMP() 
		AND end_at >= UTC_TIMESTAMP()
	`).Scan(&activeWindows)
	if err != nil {
		log.Printf("Error counting active windows: %v", err)
		activeWindows = 0
	}
	stats["activeWindows"] = activeWindows

	// Upcoming windows (enabled but not yet started)
	var upcomingWindows int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM mark_entry_windows 
		WHERE enabled = 1 
		AND start_at > UTC_TIMESTAMP()
	`).Scan(&upcomingWindows)
	if err != nil {
		log.Printf("Error counting upcoming windows: %v", err)
		upcomingWindows = 0
	}
	stats["upcomingWindows"] = upcomingWindows

	// Total teachers with mark entry permissions
	var totalTeachersWithPermissions int
	err = database.QueryRow(`
		SELECT COUNT(DISTINCT teacher_id) 
		FROM mark_entry_windows 
		WHERE teacher_id IS NOT NULL
	`).Scan(&totalTeachersWithPermissions)
	if err != nil {
		log.Printf("Error counting teachers with permissions: %v", err)
		totalTeachersWithPermissions = 0
	}
	stats["teachersWithPermissions"] = totalTeachersWithPermissions

	// Department-wide windows count
	var departmentWindows int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM mark_entry_windows 
		WHERE department_id IS NOT NULL 
		AND teacher_id IS NULL
	`).Scan(&departmentWindows)
	if err != nil {
		log.Printf("Error counting department windows: %v", err)
		departmentWindows = 0
	}
	stats["departmentWindows"] = departmentWindows

	// Teacher-specific windows count
	var teacherWindows int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM mark_entry_windows 
		WHERE teacher_id IS NOT NULL
	`).Scan(&teacherWindows)
	if err != nil {
		log.Printf("Error counting teacher windows: %v", err)
		teacherWindows = 0
	}
	stats["teacherWindows"] = teacherWindows

	json.NewEncoder(w).Encode(stats)
}
