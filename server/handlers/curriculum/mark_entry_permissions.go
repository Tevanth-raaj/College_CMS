package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server/db"
	"server/models"
)

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

	query := `
		SELECT 
			s.id,
			s.enrollment_no,
			s.student_name,
			ad.department,
			ad.year,
			ad.semester,
			ad.batch
		FROM students s
		LEFT JOIN academic_details ad ON s.id = ad.student_id
		WHERE s.status = 1`

	args := []interface{}{}

	// Filter by department_id (from students table - FK to departments)
	if departmentIDStr != "" {
		departmentID, err := strconv.Atoi(departmentIDStr)
		if err == nil {
			query += " AND s.department_id = ?"
			args = append(args, departmentID)
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

	if searchQuery != "" {
		query += " AND (s.enrollment_no LIKE ? OR s.student_name LIKE ?)"
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	query += " ORDER BY s.student_name ASC LIMIT 500"

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
		var enrollmentNo, department, batch sql.NullString
		var year, semester sql.NullInt64

		err := rows.Scan(&studentID, &enrollmentNo, &studentName, &department, &year, &semester, &batch)
		if err != nil {
			log.Printf("Error scanning student: %v", err)
			continue
		}

		student := map[string]interface{}{
			"student_id":   studentID,
			"student_name": studentName,
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

	// Create the mark entry window with user_id
	result, err := tx.Exec(`
		INSERT INTO mark_entry_windows 
		(user_id, department_id, semester, course_id, start_at, end_at, enabled, window_name)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?)
	`, numericUserID, req.DepartmentID, req.Semester, req.CourseID, req.StartAt, req.EndAt, req.WindowName)

	if err != nil {
		_ = tx.Rollback()
		log.Printf("Error creating window: %v", err)
		http.Error(w, "Failed to create window", http.StatusInternalServerError)
		return
	}

	windowID, err := result.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		log.Printf("Error getting window ID: %v", err)
		http.Error(w, "Failed to get window ID", http.StatusInternalServerError)
		return
	}

	log.Printf("Created mark entry window: id=%d, user_id=%d, course_id=%v, students=%d",
		windowID, numericUserID, req.CourseID, len(req.StudentIDs))

	// Add component permissions if specified
	if len(req.ComponentIDs) > 0 {
		for _, componentID := range req.ComponentIDs {
			_, err := tx.Exec(`
				INSERT INTO mark_entry_window_components (window_id, assessment_component_id)
				VALUES (?, ?)
			`, windowID, componentID)

			if err != nil {
				_ = tx.Rollback()
				log.Printf("Error adding component %d to window: %v", componentID, err)
				http.Error(w, "Failed to add components to window", http.StatusInternalServerError)
				return
			}
		}
		log.Printf("Added %d components to window %d", len(req.ComponentIDs), windowID)
	}

	// Create student assignments for this window
	assignmentsCreated := 0
	for _, studentID := range req.StudentIDs {
		log.Printf("Inserting student assignment: window_id=%d, user_id=%d, student_id=%d, created_by=%s",
			windowID, numericUserID, studentID, req.CreatedBy)

		result, err := tx.Exec(`
			INSERT INTO mark_entry_student_permissions
			(window_id, user_id, student_id, created_by)
			VALUES (?, ?, ?, ?)
		`, windowID, numericUserID, studentID, req.CreatedBy)

		if err != nil {
			_ = tx.Rollback()
			log.Printf("ERROR assigning student %d to window %d: %v", studentID, windowID, err)
			http.Error(w, fmt.Sprintf("Failed to assign student %d: %v", studentID, err), http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("Student assignment successful: student_id=%d, rows_affected=%d", studentID, rowsAffected)
		assignmentsCreated++
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to create window and assignments", http.StatusInternalServerError)
		return
	}

	log.Printf("Transaction committed successfully! Window %d created with %d student assignments", windowID, assignmentsCreated)

	// Verify the data was actually inserted
	var verifyCount int
	database.QueryRow(`SELECT COUNT(*) FROM mark_entry_student_permissions WHERE window_id = ?`, windowID).Scan(&verifyCount)
	log.Printf("Verification: Found %d student assignments in DB for window %d", verifyCount, windowID)

	response := models.CreateUserStudentWindowResponse{
		Success:            true,
		Message:            fmt.Sprintf("Successfully created window and assigned %d students", assignmentsCreated),
		WindowID:           int(windowID),
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

	// Resolve numeric user ID (accept numeric ID directly or lookup by username)
	var numericUserID int
	if parsedID, err := strconv.Atoi(userIdentifier); err == nil && parsedID > 0 {
		numericUserID = parsedID
		log.Printf("[DEBUG] GetUserAssignedStudents: Using numeric user_id=%d", numericUserID)
	} else {
		err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userIdentifier).Scan(&numericUserID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("[ERROR] GetUserAssignedStudents: User '%s' not found or inactive", userIdentifier)
				http.Error(w, "User not found or inactive", http.StatusBadRequest)
			} else {
				log.Printf("[ERROR] GetUserAssignedStudents: Error looking up user ID: %v", err)
				http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
			}
			return
		}
		log.Printf("[DEBUG] GetUserAssignedStudents: Resolved username '%s' to user_id=%d", userIdentifier, numericUserID)
	}

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
			COALESCE(c.course_name, '') as course_name
		FROM mark_entry_student_permissions mesp
		INNER JOIN mark_entry_windows mew ON mesp.window_id = mew.id
		INNER JOIN students s ON mesp.student_id = s.id
		LEFT JOIN academic_details ad ON s.id = ad.student_id
		LEFT JOIN courses c ON mew.course_id = c.id
		WHERE mesp.user_id = ?
			AND mew.enabled = 1
			AND mew.start_at <= ?
			AND mew.end_at > ?
			AND s.status = 1`

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
	err := database.QueryRow(`
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
		var courseName sql.NullString
		var enrollmentNo sql.NullString

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
			&courseName,
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
		if courseName.Valid {
			student.CourseName = courseName.String
		}
		if enrollmentNo.Valid {
			student.EnrollmentNo = enrollmentNo.String
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

	// Resolve numeric user ID (accept numeric ID directly or lookup by username)
	var numericUserID int
	if parsedID, err := strconv.Atoi(userIdentifier); err == nil && parsedID > 0 {
		numericUserID = parsedID
	} else {
		err := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, userIdentifier).Scan(&numericUserID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("GetUserCourses: User '%s' not found or inactive", userIdentifier)
				http.Error(w, "User not found or inactive", http.StatusBadRequest)
			} else {
				log.Printf("Error looking up user ID: %v", err)
				http.Error(w, "Failed to lookup user", http.StatusInternalServerError)
			}
			return
		}
	}

	log.Printf("GetUserCourses: Looking up courses for username='%s', user_id=%d", userIdentifier, numericUserID)

	// Get courses from active windows assigned to this user
	// This includes both:
	// 1. Windows with specific course_id set
	// 2. Courses inferred from student enrollments when window has no course_id
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
		INNER JOIN student_courses sce ON mesp.student_id = sce.student_id
		INNER JOIN courses c ON sce.course_id = c.id
		WHERE w.user_id = ?
		  AND mesp.user_id = ?
		  AND w.enabled = 1
		AND w.start_at <= ?
		AND w.end_at >= ?
		  AND (w.course_id IS NULL OR w.course_id = c.id)
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
		courses = append(courses, course)
	}

	if courses == nil {
		courses = []UserCourse{}
	}

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
