package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"server/db"
	"server/models"
)

// resolveNumericUserID resolves user identifier from username, numeric id, or teacher mapping.
func resolveNumericUserID(database *sql.DB, identifier string) (int, error) {
	if identifier == "" {
		return 0, sql.ErrNoRows
	}

	// Direct numeric user id path
	if parsedID, err := strconv.Atoi(identifier); err == nil && parsedID > 0 {
		var numericUserID int
		err = database.QueryRow(`SELECT id FROM users WHERE id = ? AND is_active = 1`, parsedID).Scan(&numericUserID)
		if err == nil {
			return numericUserID, nil
		}
		if err != sql.ErrNoRows {
			return 0, err
		}

		// Try to map teacher data to a user account via faculty_id or teacher id
		var teacherEmail sql.NullString
		err = database.QueryRow(`SELECT email FROM teachers WHERE faculty_id = ? OR id = ?`, identifier, identifier).Scan(&teacherEmail)
		if err == nil && teacherEmail.Valid {
			err = database.QueryRow(`SELECT id FROM users WHERE email = ? AND is_active = 1`, teacherEmail.String).Scan(&numericUserID)
			if err == nil {
				return numericUserID, nil
			}
			if err != sql.ErrNoRows {
				return 0, err
			}
		}

		return 0, sql.ErrNoRows
	}

	// Non-numeric path: username or email
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, identifier).Scan(&numericUserID)
	if err == nil {
		return numericUserID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	err = database.QueryRow(`SELECT id FROM users WHERE email = ? AND is_active = 1`, identifier).Scan(&numericUserID)
	if err == nil {
		return numericUserID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// try teacher mapping by faculty_id or teacher id-> user email
	var teacherEmail sql.NullString
	err = database.QueryRow(`SELECT email FROM teachers WHERE faculty_id = ? OR id = ?`, identifier, identifier).Scan(&teacherEmail)
	if err == nil && teacherEmail.Valid {
		err = database.QueryRow(`SELECT id FROM users WHERE email = ? AND is_active = 1`, teacherEmail.String).Scan(&numericUserID)
		if err == nil {
			return numericUserID, nil
		}
		if err != sql.ErrNoRows {
			return 0, err
		}
	}

	return 0, sql.ErrNoRows
}

// GetMarkEntryPermissions returns all mark categories with enabled status for a course and teacher.
func GetMarkEntryPermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	courseIDStr := r.URL.Query().Get("course_id")
	teacherIDStr := r.URL.Query().Get("teacher_id")

	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil || courseID == 0 {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	if teacherIDStr == "" {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	var courseCategory string
	err = database.QueryRow(`SELECT COALESCE(category, '') FROM courses WHERE id = ?`, courseID).Scan(&courseCategory)
	if err != nil {
		log.Printf("Error fetching course category: %v", err)
		http.Error(w, "Failed to resolve course category", http.StatusInternalServerError)
		return
	}

	courseTypeID := mapCourseCategoryToTypeID(courseCategory)
	if courseTypeID == 0 {
		http.Error(w, "Could not determine course type", http.StatusBadRequest)
		return
	}

	// Get learning modes from query parameter (comma-separated)
	learningModesStr := r.URL.Query().Get("learning_modes")
	var learningModes []int
	if learningModesStr != "" {
		modeStrs := strings.Split(learningModesStr, ",")
		for _, modeStr := range modeStrs {
			mode, err := strconv.Atoi(strings.TrimSpace(modeStr))
			if err == nil && (mode == 1 || mode == 2) {
				learningModes = append(learningModes, mode)
			}
		}
	}

	// Default to both UAL and PBL if no modes specified (backward compatibility)
	if len(learningModes) == 0 {
		learningModes = []int{1, 2}
	}

	// Build WHERE clause for learning modes
	learningModePlaceholders := make([]string, len(learningModes))
	learningModeArgs := make([]interface{}, 0)
	learningModeArgs = append(learningModeArgs, courseID, teacherIDStr, courseTypeID)
	for i, mode := range learningModes {
		learningModePlaceholders[i] = "?"
		learningModeArgs = append(learningModeArgs, mode)
	}
	learningModeClause := strings.Join(learningModePlaceholders, ",")

	query := fmt.Sprintf(`
		SELECT 
			mct.id,
			mct.name,
			mct.max_marks,
			mct.conversion_marks,
			mct.position,
			mct.course_type_id,
			mct.category_name_id,
			mct.learning_mode_id,
			mct.status,
			COALESCE(p.enabled, 1) AS enabled
		FROM mark_category_types mct
		LEFT JOIN mark_entry_field_permissions p
			ON p.course_id = ? AND p.teacher_id = ? AND p.assessment_component_id = mct.id
		WHERE mct.course_type_id = ? AND mct.learning_mode_id IN (%s) AND mct.status = 1
		ORDER BY mct.position ASC
	`, learningModeClause)

	rows, err := database.Query(query, learningModeArgs...)
	if err != nil {
		log.Printf("Error fetching mark entry permissions: %v", err)
		http.Error(w, "Error fetching mark entry permissions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.MarkEntryPermissionCategory
	for rows.Next() {
		var category models.MarkEntryPermissionCategory
		var enabledInt int
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.MaxMarks,
			&category.ConversionMarks,
			&category.Position,
			&category.CourseTypeID,
			&category.CategoryNameID,
			&category.LearningModeID,
			&category.Status,
			&enabledInt,
		)
		if err != nil {
			log.Printf("Error scanning mark entry permission row: %v", err)
			continue
		}
		category.Enabled = enabledInt == 1
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating mark entry permissions: %v", err)
		http.Error(w, "Error processing mark entry permissions", http.StatusInternalServerError)
		return
	}

	if categories == nil {
		categories = []models.MarkEntryPermissionCategory{}
	}

	json.NewEncoder(w).Encode(categories)
}

// SaveMarkEntryPermissions updates the enabled state of mark categories for a course and teacher.
func SaveMarkEntryPermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req models.MarkEntryPermissionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CourseID == 0 || req.TeacherID == "" || len(req.Permissions) == 0 {
		http.Error(w, "Course ID, teacher ID, and permissions are required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	if err := ensureWindowDepartmentsTable(database); err != nil {
		log.Printf("Error ensuring mark_entry_window_departments table: %v", err)
		http.Error(w, "Failed to create window", http.StatusInternalServerError)
		return
	}

	tx, err := database.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Failed to update permissions", http.StatusInternalServerError)
		return
	}

	for _, permission := range req.Permissions {
		enabledValue := 0
		if permission.Enabled {
			enabledValue = 1
		}

		_, err := tx.Exec(`
			INSERT INTO mark_entry_field_permissions
			(course_id, teacher_id, assessment_component_id, enabled)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE enabled = VALUES(enabled)
		`, req.CourseID, req.TeacherID, permission.AssessmentComponentID, enabledValue)
		if err != nil {
			_ = tx.Rollback()
			log.Printf("Error saving mark entry permission: %v", err)
			http.Error(w, "Failed to update permissions", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing permissions update: %v", err)
		http.Error(w, "Failed to update permissions", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Permissions updated successfully"})
}

// GetAvailableUsersForAssignment returns all users that can be assigned for mark entry
func GetAvailableUsersForAssignment(w http.ResponseWriter, r *http.Request) {
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

	// Get optional search query
	searchQuery := r.URL.Query().Get("search")
	roleFilter := r.URL.Query().Get("role")

	query := `
		SELECT id, username, email, role, is_active
		FROM users
		WHERE is_active = 1`

	args := []interface{}{}

	if roleFilter != "" {
		query += " AND role = ?"
		args = append(args, roleFilter)
	}

	if searchQuery != "" {
		query += " AND (username LIKE ? OR email LIKE ?)"
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	query += " ORDER BY username ASC"

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var username, email, role string
		var isActive bool

		err := rows.Scan(&id, &username, &email, &role, &isActive)
		if err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}

		users = append(users, map[string]interface{}{
			"id":        id,
			"username":  username,
			"email":     email,
			"role":      role,
			"is_active": isActive,
		})
	}

	if users == nil {
		users = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(users)
}

// GetStudentsForAssignment returns students that can be assigned for mark entry with filtering
func GetStudentsForAssignment(w http.ResponseWriter, r *http.Request) {
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

	// Get filter parameters
	searchQuery := r.URL.Query().Get("search")
	department := r.URL.Query().Get("department")
	departmentIDStr := r.URL.Query().Get("department_id")
	yearStr := r.URL.Query().Get("year")
	semester := r.URL.Query().Get("semester")
	learningMode := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("learning_mode"))) // UAL / PBL / UAL&PBL
	limitStr := r.URL.Query().Get("limit")

	query := `
		SELECT 
			s.id,
			s.enrollment_no,
			s.student_name,
			COALESCE(s.learning_mode_id, 2) AS learning_mode_id,
			ad.department,
			ad.year,
			ad.semester,
			ad.batch
		FROM students s
		LEFT JOIN academic_details ad ON s.id = ad.student_id
		WHERE s.status = 1`

	args := []interface{}{}

	// Filter by department_id (from students table - FK to departments).
	// Supports a single ID ("3") or comma-separated IDs ("3,5,7").
	if departmentIDStr != "" {
		departmentParts := strings.Split(departmentIDStr, ",")
		departmentIDs := make([]int, 0, len(departmentParts))
		seenDepartmentIDs := make(map[int]bool)
		for _, part := range departmentParts {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			departmentID, err := strconv.Atoi(value)
			if err != nil || departmentID <= 0 || seenDepartmentIDs[departmentID] {
				continue
			}
			seenDepartmentIDs[departmentID] = true
			departmentIDs = append(departmentIDs, departmentID)
		}

		if len(departmentIDs) == 1 {
			query += " AND s.department_id = ?"
			args = append(args, departmentIDs[0])
		} else if len(departmentIDs) > 1 {
			placeholders := make([]string, len(departmentIDs))
			for i, departmentID := range departmentIDs {
				placeholders[i] = "?"
				args = append(args, departmentID)
			}
			query += " AND s.department_id IN (" + strings.Join(placeholders, ",") + ")"
		}
	}

	// Filter by department name (from academic_details - for backward compatibility)
	if department != "" {
		query += " AND ad.department = ?"
		args = append(args, department)
	}

	if yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err == nil {
			query += " AND ad.year = ?"
			args = append(args, year)
		}
	}

	if semester != "" {
		query += " AND ad.semester = ?"
		args = append(args, semester)
	}

	if learningMode != "" && learningMode != "ALL" && learningMode != "UAL&PBL" {
		if learningMode == "UAL" {
			query += " AND COALESCE(s.learning_mode_id, 2) = ?"
			args = append(args, 1)
		} else if learningMode == "PBL" {
			query += " AND COALESCE(s.learning_mode_id, 2) = ?"
			args = append(args, 2)
		}
	}

	if searchQuery != "" {
		query += " AND (s.enrollment_no LIKE ? OR s.student_name LIKE ?)"
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	query += " ORDER BY s.student_name ASC"
	// Default limit to 5000 to avoid unbounded result sets, but allow override
	maxLimit := 5000
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 20000 {
			maxLimit = parsedLimit
		}
	}
	query += " LIMIT ?"
	args = append(args, maxLimit)

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching students: %v", err)
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var students []map[string]interface{}
	for rows.Next() {
		var studentID int
		var studentName string
		var learningModeID sql.NullInt64
		var enrollmentNo, department, batch sql.NullString
		var year, semester sql.NullInt64

		err := rows.Scan(&studentID, &enrollmentNo, &studentName, &learningModeID, &department, &year, &semester, &batch)
		if err != nil {
			log.Printf("Error scanning student: %v", err)
			continue
		}

		student := map[string]interface{}{
			"student_id":       studentID,
			"student_name":     studentName,
			"learning_mode_id": 2,
		}
		if learningModeID.Valid {
			student["learning_mode_id"] = int(learningModeID.Int64)
		}

		if enrollmentNo.Valid {
			student["enrollment_no"] = enrollmentNo.String
		} else {
			student["enrollment_no"] = ""
		}

		if department.Valid {
			student["department"] = department.String
		}
		if year.Valid {
			student["year"] = int(year.Int64)
		}
		if semester.Valid {
			student["semester"] = int(semester.Int64)
		}
		if batch.Valid {
			student["batch"] = batch.String
		}

		students = append(students, student)
	}

	if students == nil {
		students = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(students)
}

// AssignStudentsToUser assigns specific students to a user within a mark entry window
// If matching window criteria are provided, it finds or creates a window with that configuration
func AssignStudentsToUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req models.AssignStudentsToUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || len(req.StudentIDs) == 0 {
		http.Error(w, "User ID and student IDs are required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Lookup numeric user ID from username
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, req.UserID).Scan(&numericUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found or inactive", http.StatusBadRequest)
		} else {
			log.Printf("Error looking up user ID: %v", err)
			http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		}
		return
	}

	var windowID int

	// If window_id is provided, use it directly
	if req.WindowID != 0 {
		var windowExists bool
		var windowEnabled int
		err := database.QueryRow(`
			SELECT COUNT(*) > 0, COALESCE(MAX(enabled), 0)
			FROM mark_entry_windows
			WHERE id = ? AND end_at > UTC_TIMESTAMP()
		`, req.WindowID).Scan(&windowExists, &windowEnabled)

		if err != nil || !windowExists || windowEnabled == 0 {
			http.Error(w, "Invalid or expired mark entry window", http.StatusBadRequest)
			return
		}
		windowID = req.WindowID
	} else {
		// Find or create a matching window based on criteria
		// This allows overlapping with existing teacher windows
		http.Error(w, "Window ID is required. Please provide window_id or use the find/create endpoint.", http.StatusBadRequest)
		return
	}

	tx, err := database.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Failed to assign students", http.StatusInternalServerError)
		return
	}

	assignmentsCreated := 0
	for _, studentID := range req.StudentIDs {
		_, err := tx.Exec(`
			INSERT INTO mark_entry_student_permissions
			(window_id, user_id, student_id, created_by)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP
		`, windowID, numericUserID, studentID, req.CreatedBy)

		if err != nil {
			_ = tx.Rollback()
			log.Printf("Error assigning student %d to user %s: %v", studentID, req.UserID, err)
			http.Error(w, "Failed to assign students", http.StatusInternalServerError)
			return
		}
		assignmentsCreated++
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing student assignments: %v", err)
		http.Error(w, "Failed to assign students", http.StatusInternalServerError)
		return
	}

	response := models.AssignStudentsToUserResponse{
		Success:            true,
		Message:            fmt.Sprintf("Successfully assigned %d students to user in window #%d", assignmentsCreated, windowID),
		AssignmentsCreated: assignmentsCreated,
	}

	json.NewEncoder(w).Encode(response)
}

// CreateUserStudentWindow creates a new mark entry window for user-student assignment
// This window includes full configuration like regular windows but with user_id instead of teacher_id
func CreateUserStudentWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req models.CreateUserStudentWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || len(req.StudentIDs) == 0 || req.StartAt == "" || req.EndAt == "" {
		http.Error(w, "UserID, student IDs, start_at, and end_at are required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Lookup numeric user ID from username
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, req.UserID).Scan(&numericUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found or inactive", http.StatusBadRequest)
		} else {
			log.Printf("Error looking up user ID: %v", err)
			http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		}
		return
	}

	tx, err := database.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Failed to create window", http.StatusInternalServerError)
		return
	}

	selectedDepartmentIDs := sanitizeDepartmentIDs(req.DepartmentIDs)
	if len(selectedDepartmentIDs) == 0 && req.DepartmentID != nil && *req.DepartmentID > 0 {
		selectedDepartmentIDs = []int{*req.DepartmentID}
	}

	departmentValue := sql.NullInt64{}
	if len(selectedDepartmentIDs) == 1 {
		departmentValue = sql.NullInt64{Int64: int64(selectedDepartmentIDs[0]), Valid: true}
	}

	studentDepartmentMap := make(map[int]int)
	if len(req.StudentIDs) > 0 {
		placeholders := make([]string, len(req.StudentIDs))
		studentArgs := make([]interface{}, len(req.StudentIDs))
		for i, studentID := range req.StudentIDs {
			placeholders[i] = "?"
			studentArgs[i] = studentID
		}

		studentDeptQuery := fmt.Sprintf(`
			SELECT id, COALESCE(department_id, 0)
			FROM students
			WHERE id IN (%s)
		`, strings.Join(placeholders, ","))

		studentRows, studentErr := tx.Query(studentDeptQuery, studentArgs...)
		if studentErr != nil {
			_ = tx.Rollback()
			log.Printf("Error fetching student department mapping: %v", studentErr)
			http.Error(w, "Failed to validate student departments", http.StatusInternalServerError)
			return
		}
		for studentRows.Next() {
			var studentID int
			var departmentID int
			if scanErr := studentRows.Scan(&studentID, &departmentID); scanErr == nil {
				studentDepartmentMap[studentID] = departmentID
			}
		}
		studentRows.Close()
	}

	allowedDepartmentSet := make(map[int]struct{}, len(selectedDepartmentIDs))
	for _, departmentID := range selectedDepartmentIDs {
		allowedDepartmentSet[departmentID] = struct{}{}
	}

	windowIDs := make([]int, 0, 1)
	assignmentsCreated := 0

	result, execErr := tx.Exec(`
		INSERT INTO mark_entry_windows 
		(user_id, department_id, semester, course_id, start_at, end_at, enabled, window_name)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?)
	`, numericUserID, departmentValue, req.Semester, req.CourseID, req.StartAt, req.EndAt, req.WindowName)
	if execErr != nil {
		_ = tx.Rollback()
		log.Printf("Error creating window: %v", execErr)
		http.Error(w, "Failed to create window", http.StatusInternalServerError)
		return
	}

	windowID, idErr := result.LastInsertId()
	if idErr != nil {
		_ = tx.Rollback()
		log.Printf("Error getting window ID: %v", idErr)
		http.Error(w, "Failed to get window ID", http.StatusInternalServerError)
		return
	}
	windowIDs = append(windowIDs, int(windowID))

	if err := replaceWindowDepartmentsTx(tx, int(windowID), selectedDepartmentIDs); err != nil {
		_ = tx.Rollback()
		log.Printf("Error saving selected departments for user window: %v", err)
		http.Error(w, "Failed to save selected departments", http.StatusInternalServerError)
		return
	}

	if len(req.ComponentIDs) > 0 {
		for _, componentID := range req.ComponentIDs {
			_, componentErr := tx.Exec(`
				INSERT INTO mark_entry_window_components (window_id, assessment_component_id)
				VALUES (?, ?)
			`, windowID, componentID)
			if componentErr != nil {
				_ = tx.Rollback()
				log.Printf("Error adding component %d to window: %v", componentID, componentErr)
				http.Error(w, "Failed to add components to window", http.StatusInternalServerError)
				return
			}
		}
	}

	for _, studentID := range req.StudentIDs {
		if len(allowedDepartmentSet) > 0 {
			if _, ok := allowedDepartmentSet[studentDepartmentMap[studentID]]; !ok {
				continue
			}
		}

		assignResult, assignErr := tx.Exec(`
			INSERT INTO mark_entry_student_permissions
			(window_id, user_id, student_id, created_by)
			VALUES (?, ?, ?, ?)
		`, windowID, numericUserID, studentID, req.CreatedBy)
		if assignErr != nil {
			_ = tx.Rollback()
			log.Printf("ERROR assigning student %d to window %d: %v", studentID, windowID, assignErr)
			http.Error(w, fmt.Sprintf("Failed to assign student %d: %v", studentID, assignErr), http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := assignResult.RowsAffected()
		if rowsAffected > 0 {
			assignmentsCreated++
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to create window and assignments", http.StatusInternalServerError)
		return
	}

	log.Printf("Transaction committed successfully! Windows %v created with %d student assignments", windowIDs, assignmentsCreated)

	// Verify assignments for the first created window for debugging parity.
	if len(windowIDs) > 0 {
		var verifyCount int
		database.QueryRow(`SELECT COUNT(*) FROM mark_entry_student_permissions WHERE window_id = ?`, windowIDs[0]).Scan(&verifyCount)
		log.Printf("Verification: Found %d student assignments in DB for window %d", verifyCount, windowIDs[0])
	}

	response := models.CreateUserStudentWindowResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully created %d window(s) and assigned %d students", len(windowIDs), assignmentsCreated),
		WindowID: func() int {
			if len(windowIDs) > 0 {
				return windowIDs[0]
			}
			return 0
		}(),
		WindowIDs:          windowIDs,
		WindowsCreated:     len(windowIDs),
		AssignmentsCreated: assignmentsCreated,
	}

	json.NewEncoder(w).Encode(response)
}

// GetUserAssignedStudents returns students assigned to a specific user within active mark entry windows
func GetUserAssignedStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	userIdentifier := r.URL.Query().Get("user_id")
	log.Printf("[DEBUG] GetUserAssignedStudents: Received request - user_id=%s", userIdentifier)
	if userIdentifier == "" {
		log.Printf("[ERROR] GetUserAssignedStudents: user_id parameter is missing")
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	courseIDStr := r.URL.Query().Get("course_id")
	var courseID *int
	if courseIDStr != "" {
		id, err := strconv.Atoi(courseIDStr)
		if err == nil && id > 0 {
			courseID = &id
			log.Printf("[DEBUG] GetUserAssignedStudents: course_id filter=%d", id)
		}
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Resolve numeric user ID from username/user_id/teacher mapping
	numericUserID, err := resolveNumericUserID(database, userIdentifier)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[ERROR] GetUserAssignedStudents: User '%s' not found or inactive", userIdentifier)
			http.Error(w, "User not found or inactive", http.StatusBadRequest)
			return
		}
		log.Printf("[ERROR] GetUserAssignedStudents: Error resolving user ID for '%s': %v", userIdentifier, err)
		http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		return
	}
	log.Printf("[DEBUG] GetUserAssignedStudents: Resolved user identifier '%s' to user_id=%d", userIdentifier, numericUserID)

	log.Printf("[INFO] GetUserAssignedStudents: user_id=%s resolved_user_id=%d course_id=%v now=%s", userIdentifier, numericUserID, courseID, time.Now().Format("2006-01-02 15:04:05"))

	now := time.Now()
	query := `
		SELECT DISTINCT
			s.id as student_id,
			s.enrollment_no,
			s.student_name,
			COALESCE(ad.department, '') as department,
			COALESCE(ad.year, 0) as year,
			mew.id as window_id,
			mew.start_at,
			mew.end_at,
			mew.course_id,
			COALESCE(c.course_code, '') as course_code,
			COALESCE(c.course_name, '') as course_name,
			s.learning_mode_id,
			COALESCE(s.register_no, '') as register_no
		FROM mark_entry_student_permissions mesp
		INNER JOIN mark_entry_windows mew ON mesp.window_id = mew.id
		INNER JOIN students s ON mesp.student_id = s.id
		LEFT JOIN academic_details ad ON s.id = ad.student_id
		LEFT JOIN academic_calendar ac ON ac.id = s.year
		LEFT JOIN courses c ON mew.course_id = c.id
		WHERE mesp.user_id = ?
			AND mew.enabled = 1
			AND mew.start_at <= ?
			AND mew.end_at > ?
			AND s.status = 1
			AND (COALESCE(mew.semester, 0) = 0 OR ac.current_semester = mew.semester)`

	args := []interface{}{numericUserID, now, now}

	if courseID != nil {
		query += " AND (mew.course_id = ? OR (mew.course_id IS NULL AND EXISTS (SELECT 1 FROM student_courses sc WHERE sc.student_id = s.id AND sc.course_id = ?)))"
		args = append(args, *courseID, *courseID)
	}

	query += " ORDER BY s.student_name ASC"

	log.Printf("[DEBUG] GetUserAssignedStudents: Full SQL query: %s", query)
	log.Printf("[DEBUG] GetUserAssignedStudents: Query args=%v", args)

	// Check windows before executing main query
	var windowCount int
	err = database.QueryRow(`
		SELECT COUNT(*) FROM mark_entry_windows 
		WHERE user_id = ? AND enabled = 1 AND start_at <= ? AND end_at > ?
	`, numericUserID, now, now).Scan(&windowCount)
	if err == nil {
		log.Printf("[DEBUG] GetUserAssignedStudents: Active windows for user_id=%d: %d", numericUserID, windowCount)
	}

	// Check all windows for this user (even inactive/expired)
	var totalWindowCount int
	err = database.QueryRow(`SELECT COUNT(*) FROM mark_entry_windows WHERE user_id = ?`, numericUserID).Scan(&totalWindowCount)
	if err == nil {
		log.Printf("[DEBUG] GetUserAssignedStudents: Total windows (all statuses) for user_id=%d: %d", numericUserID, totalWindowCount)
	}

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("[ERROR] GetUserAssignedStudents: Error fetching assigned students: %v", err)
		http.Error(w, "Failed to fetch assigned students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Debug counts to locate filtering mismatch
	var permCount int
	_ = database.QueryRow(`SELECT COUNT(*) FROM mark_entry_student_permissions WHERE user_id = ?`, numericUserID).Scan(&permCount)
	log.Printf("[DEBUG] GetUserAssignedStudents: Total student permissions for user_id=%d: %d", numericUserID, permCount)
	if courseID != nil {
		var enrollmentCount int
		_ = database.QueryRow(`SELECT COUNT(*) FROM student_courses WHERE course_id = ?`, *courseID).Scan(&enrollmentCount)
		log.Printf("GetUserAssignedStudents: student_courses count for course_id=%d is %d", *courseID, enrollmentCount)
	}

	var assignedStudents []models.AssignedStudentInfo
	rowCount := 0
	log.Printf("[DEBUG] GetUserAssignedStudents: Starting to scan rows...")
	for rows.Next() {
		var student models.AssignedStudentInfo
		var startAt, endAt time.Time
		var courseIDNull sql.NullInt64
		var courseCode sql.NullString
		var courseName sql.NullString
		var enrollmentNo sql.NullString
		var registerNo sql.NullString

		err := rows.Scan(
			&student.StudentID,
			&enrollmentNo,
			&student.StudentName,
			&student.Department,
			&student.Year,
			&student.WindowID,
			&startAt,
			&endAt,
			&courseIDNull,
			&courseCode,
			&courseName,
			&student.LearningModeID,
			&registerNo,
		)
		if err != nil {
			log.Printf("[ERROR] GetUserAssignedStudents: Error scanning assigned student: %v", err)
			continue
		}

		student.WindowStart = startAt.Format("2006-01-02T15:04")
		student.WindowEnd = endAt.Format("2006-01-02T15:04")

		if courseIDNull.Valid {
			id := int(courseIDNull.Int64)
			student.CourseID = &id
		}
		if courseCode.Valid {
			student.CourseCode = courseCode.String
		}
		if courseName.Valid {
			student.CourseName = courseName.String
		}
		if enrollmentNo.Valid {
			student.EnrollmentNo = enrollmentNo.String
		}
		if registerNo.Valid {
			student.RegisterNo = registerNo.String
		}

		log.Printf("[DEBUG] GetUserAssignedStudents: Row %d - student_id=%d, name=%s, window_id=%d, window: %s to %s",
			rowCount+1, student.StudentID, student.StudentName, student.WindowID, student.WindowStart, student.WindowEnd)

		assignedStudents = append(assignedStudents, student)
		rowCount++
	}

	log.Printf("[INFO] GetUserAssignedStudents: Total rows returned=%d", rowCount)

	if assignedStudents == nil {
		log.Printf("[DEBUG] GetUserAssignedStudents: No students found, returning empty array")
		assignedStudents = []models.AssignedStudentInfo{}
	} else {
		log.Printf("[INFO] GetUserAssignedStudents: Returning %d students", len(assignedStudents))
	}

	json.NewEncoder(w).Encode(assignedStudents)
}

// RemoveStudentAssignment removes a student assignment from a user
func RemoveStudentAssignment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	windowIDStr := r.URL.Query().Get("window_id")
	userIdentifier := r.URL.Query().Get("user_id")
	studentIDStr := r.URL.Query().Get("student_id")

	windowID, err1 := strconv.Atoi(windowIDStr)
	studentID, err2 := strconv.Atoi(studentIDStr)

	if err1 != nil || err2 != nil || windowID == 0 || studentID == 0 || userIdentifier == "" {
		http.Error(w, "Window ID, user ID, and student ID are required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Lookup numeric user ID from username
	var numericUserID int
	err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userIdentifier).Scan(&numericUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found or inactive", http.StatusBadRequest)
		} else {
			log.Printf("Error looking up user ID: %v", err)
			http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		}
		return
	}

	result, err := database.Exec(`
		DELETE FROM mark_entry_student_permissions
		WHERE window_id = ? AND user_id = ? AND student_id = ?
	`, windowID, numericUserID, studentID)

	if err != nil {
		log.Printf("Error removing student assignment: %v", err)
		http.Error(w, "Failed to remove student assignment", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Student assignment removed successfully",
	})
}

// GetUserCourses returns all courses assigned to a user via mark entry windows
func GetUserCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract user ID from path parameter
	pathParts := strings.Split(r.URL.Path, "/")
	var userIdentifier string
	for i, part := range pathParts {
		if part == "users" && i+1 < len(pathParts) {
			userIdentifier = pathParts[i+1]
			break
		}
	}

	if userIdentifier == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Resolve numeric user ID from username/user_id/teacher mapping
	numericUserID, err := resolveNumericUserID(database, userIdentifier)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("GetUserCourses: User '%s' not found or inactive", userIdentifier)
			http.Error(w, "User not found or inactive", http.StatusBadRequest)
			return
		}
		log.Printf("GetUserCourses: Error resolving user ID for '%s': %v", userIdentifier, err)
		http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
		return
	}

	log.Printf("GetUserCourses: Looking up courses for username='%s', user_id=%d", userIdentifier, numericUserID)

	// Get courses from active windows assigned to this user
	// This includes student-course mappings only when the window is not department/semester filtered
	now := time.Now()
	query := `
		SELECT DISTINCT 
			c.id as course_id,
			c.course_code,
			c.course_name,
			c.category,
			w.semester,
			w.id as window_id
		FROM mark_entry_windows w
		INNER JOIN mark_entry_student_permissions mesp ON w.id = mesp.window_id
		INNER JOIN students s ON mesp.student_id = s.id
		LEFT JOIN academic_calendar ac ON ac.id = s.year
		INNER JOIN student_courses sce ON mesp.student_id = sce.student_id
		INNER JOIN courses c ON sce.course_id = c.id
		WHERE w.user_id = ?
		  AND mesp.user_id = ?
		  AND w.enabled = 1
		AND w.start_at <= ?
		AND w.end_at >= ?
		  AND s.status = 1
		  AND (
			COALESCE(w.semester, 0) = 0
			OR ac.current_semester = w.semester
			OR EXISTS (
				SELECT 1
				FROM curriculum_courses ccx
				JOIN normal_cards ncx ON ccx.semester_id = ncx.id
				JOIN departments dx ON dx.current_curriculum_id = ncx.curriculum_id
				WHERE ccx.course_id = c.id
				  AND (COALESCE(w.department_id, 0) = 0 OR dx.id = w.department_id)
				  AND LOWER(TRIM(COALESCE(ncx.card_type, 'semester'))) <> 'semester'
			)
		  )
		  AND (
			(w.course_id IS NOT NULL AND w.course_id = c.id)
			OR (w.course_id IS NULL OR w.course_id = 0)
		)
		ORDER BY c.course_code
	`

	log.Printf("GetUserCourses: Executing query with user_id=%d, now=%s", numericUserID, now.Format("2006-01-02 15:04:05"))
	rows, err := database.Query(query, numericUserID, numericUserID, now, now)
	if err != nil {
		log.Printf("Error fetching user courses: %v", err)
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type UserCourse struct {
		CourseID   int    `json:"course_id"`
		CourseCode string `json:"course_code"`
		CourseName string `json:"course_name"`
		Category   string `json:"category"`
		Semester   *int   `json:"semester"`
		WindowID   int    `json:"window_id"`
	}

	// Use map to deduplicate courses (by course_id)
	courseMap := make(map[int]UserCourse)
	addCourse := func(course UserCourse) {
		if course.CourseID == 0 {
			return
		}
		if existing, found := courseMap[course.CourseID]; found {
			// Preserve window_id if not set in existing and set now
			if existing.WindowID == 0 && course.WindowID != 0 {
				existing.WindowID = course.WindowID
				courseMap[course.CourseID] = existing
			}
			return
		}
		courseMap[course.CourseID] = course
	}

	var courses []UserCourse
	for rows.Next() {
		var course UserCourse
		var semester sql.NullInt64
		err := rows.Scan(
			&course.CourseID,
			&course.CourseCode,
			&course.CourseName,
			&course.Category,
			&semester,
			&course.WindowID,
		)
		if err != nil {
			log.Printf("Error scanning course: %v", err)
			continue
		}
		if semester.Valid {
			semInt := int(semester.Int64)
			course.Semester = &semInt
		}
		addCourse(course)
	}

	// Include courses from active user windows (course_id, dept+semester windows)
	windowRows, err := database.Query(`
		SELECT id, course_id, department_id, semester
		FROM mark_entry_windows
		WHERE user_id = ? AND enabled = 1 AND start_at <= ? AND end_at >= ?
	`, numericUserID, now, now)
	if err == nil {
		defer windowRows.Close()
		for windowRows.Next() {
			var windowID, courseID, deptID, semID sql.NullInt64
			if err := windowRows.Scan(&windowID, &courseID, &deptID, &semID); err != nil {
				continue
			}

			// If explicit course_id is set, include that course directly
			if courseID.Valid && courseID.Int64 > 0 {
				var c UserCourse
				c.WindowID = int(windowID.Int64)
				c.Semester = nil
				if semID.Valid {
					v := int(semID.Int64)
					c.Semester = &v
				}
				courseDetailRow := database.QueryRow(`SELECT id, course_code, course_name, category FROM courses WHERE id = ?`, courseID.Int64)
				if err := courseDetailRow.Scan(&c.CourseID, &c.CourseCode, &c.CourseName, &c.Category); err == nil {
					addCourse(c)
				}
				continue
			}

			// If window is dept/semester based, query curriculum-based courses
			if deptID.Valid || semID.Valid {
				query := `
				SELECT DISTINCT c.id, c.course_code, c.course_name, c.category
				FROM departments d
				INNER JOIN curriculum_courses cc ON d.current_curriculum_id = cc.curriculum_id
				INNER JOIN normal_cards nc ON cc.semester_id = nc.id
				INNER JOIN courses c ON cc.course_id = c.id
				WHERE d.status = 1`
				args := []interface{}{}
				// When department_id=0 it means all departments, so omit department filter
				if deptID.Valid && deptID.Int64 > 0 {
					query += ` AND d.id = ?`
					args = append(args, deptID.Int64)
				}
				// When semester=0 it means all semesters, so omit semester filter
				if semID.Valid && semID.Int64 > 0 {
					query += ` AND (
						nc.semester_number = ?
						OR LOWER(COALESCE(nc.card_type, 'semester')) <> 'semester'
					)`
					args = append(args, semID.Int64)
				}
				query += `
				AND EXISTS (
					SELECT 1
					FROM mark_entry_student_permissions mesp
					INNER JOIN students s ON s.id = mesp.student_id AND s.status = 1
					LEFT JOIN academic_calendar ac ON ac.id = s.year
					INNER JOIN student_courses sc ON sc.student_id = s.id AND sc.course_id = c.id
					WHERE mesp.window_id = ? AND mesp.user_id = ?
					  AND (
						COALESCE(?, 0) = 0
						OR (LOWER(COALESCE(nc.card_type, 'semester')) = 'semester' AND ac.current_semester = ?)
						OR LOWER(COALESCE(nc.card_type, 'semester')) <> 'semester'
					  )
				)`
				args = append(args, windowID.Int64, numericUserID, semID.Int64, semID.Int64)
				query += ` ORDER BY c.course_code`

				deptRows, deptErr := database.Query(query, args...)
				if deptErr != nil {
					continue
				}
				for deptRows.Next() {
					var c UserCourse
					c.WindowID = int(windowID.Int64)
					if semID.Valid {
						tmp := int(semID.Int64)
						c.Semester = &tmp
					}
					if err := deptRows.Scan(&c.CourseID, &c.CourseCode, &c.CourseName, &c.Category); err == nil {
						addCourse(c)
					}
				}
				deptRows.Close()
			}
		}
	}

	// If no courses found, convert to empty slice
	courses = make([]UserCourse, 0, len(courseMap))
	for _, c := range courseMap {
		courses = append(courses, c)
	}

	// Sort courses by code for stable order
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].CourseCode < courses[j].CourseCode
	})

	log.Printf("GetUserCourses: Returning %d courses for user '%s' (id=%d)", len(courses), userIdentifier, numericUserID)

	json.NewEncoder(w).Encode(courses)
}

// CheckUserHasAssignedWindows checks if a user has any assigned mark entry windows
func CheckUserHasAssignedWindows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	userIdentifier := r.URL.Query().Get("user_id")
	if userIdentifier == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Resolve numeric user ID (accept numeric ID directly or lookup by username)
	var numericUserID int
	if parsedID, err := strconv.Atoi(userIdentifier); err == nil && parsedID > 0 {
		numericUserID = parsedID
	} else {
		err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userIdentifier).Scan(&numericUserID)
		if err != nil {
			if err == sql.ErrNoRows {
				json.NewEncoder(w).Encode(map[string]bool{"has_windows": false})
				return
			}
			log.Printf("Error looking up user ID: %v", err)
			http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
			return
		}
	}

	// Check if user has any assigned windows with students
	var count int
	now := time.Now()
	query := `
		SELECT COUNT(DISTINCT mesp.window_id)
		FROM mark_entry_student_permissions mesp
		INNER JOIN mark_entry_windows mew ON mesp.window_id = mew.id
		WHERE mesp.user_id = ?
			AND mew.enabled = 1
			AND mew.start_at <= ?
			AND mew.end_at > ?`

	err := database.QueryRow(query, numericUserID, now, now).Scan(&count)
	if err != nil {
		log.Printf("Error checking user windows: %v", err)
		http.Error(w, "Failed to check user windows", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"has_windows": count > 0})
}
