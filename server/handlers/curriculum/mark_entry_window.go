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

	// No longer deleting existing windows for the same scope
	// Multiple windows can coexist and their components get merged by resolveMarkEntryWindow

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

	result, err := database.Exec(`
		INSERT INTO mark_entry_windows
		(teacher_id, department_id, semester, course_id, start_at, end_at, enabled, window_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, teacherValue, departmentValue, semesterValue, courseValue, startAt, endAt, enabledValue, req.WindowName)
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
func resolveMarkEntryWindow(courseID int, teacherID string) (bool, []int, []int, error) {
	database := db.DB
	if database == nil {
		return false, nil, nil, sql.ErrConnDone
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
		return false, nil, nil, err
	}

	// Get ALL departments the teacher belongs to (a teacher can be in multiple depts)
	var teacherDeptIDs []int64
	teacherDeptRows, err := database.Query(`
		SELECT department_id
		FROM department_teachers
		WHERE teacher_id = ? AND status = 1
	`, facultyID)
	if err != nil && err != sql.ErrNoRows {
		return false, nil, nil, err
	}
	if err == nil {
		for teacherDeptRows.Next() {
			var did int64
			if teacherDeptRows.Scan(&did) == nil {
				teacherDeptIDs = append(teacherDeptIDs, did)
			}
		}
		teacherDeptRows.Close()
	}

	log.Printf("Teacher %s belongs to departments: %v", facultyID, teacherDeptIDs)

	// Also look up which department(s) the COURSE belongs to via the curriculum chain.
	// A Math teacher teaching 22AI401 in the AIDS curriculum should match an AIDS-dept window.
	// Path: course → curriculum_courses → normal_cards → curriculum ← departments.current_curriculum_id
	var courseDeptIDs []int64
	deptRows, err := database.Query(`
		SELECT DISTINCT d.id
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN departments d ON d.current_curriculum_id = nc.curriculum_id
		WHERE cc.course_id = ?
	`, courseID)
	if err == nil {
		for deptRows.Next() {
			var did int64
			if deptRows.Scan(&did) == nil {
				courseDeptIDs = append(courseDeptIDs, did)
			}
		}
		deptRows.Close()
	}

	// Also collect department(s) of students actually allocated for this teacher+course.
	// This is required for cross-department teaching where curriculum mapping can be incomplete.
	var studentDeptIDs []int64
	studentDeptRows, studentDeptErr := database.Query(`
		SELECT DISTINCT s.department_id
		FROM course_student_teacher_allocation csta
		JOIN students s ON csta.student_id = s.id
		WHERE csta.course_id = ?
		  AND csta.teacher_id = ?
		  AND s.department_id IS NOT NULL
	`, courseID, facultyID)
	if studentDeptErr == nil {
		for studentDeptRows.Next() {
			var did int64
			if studentDeptRows.Scan(&did) == nil {
				studentDeptIDs = append(studentDeptIDs, did)
			}
		}
		studentDeptRows.Close()
	}

	// Merge teacher departments + course departments (deduplicated)
	allDeptIDs := make([]int64, 0, len(courseDeptIDs)+len(teacherDeptIDs)+len(studentDeptIDs))
	deptSeen := make(map[int64]bool)
	for _, did := range courseDeptIDs {
		if !deptSeen[did] {
			allDeptIDs = append(allDeptIDs, did)
			deptSeen[did] = true
		}
	}
	for _, did := range studentDeptIDs {
		if !deptSeen[did] {
			allDeptIDs = append(allDeptIDs, did)
			deptSeen[did] = true
		}
	}
	for _, did := range teacherDeptIDs {
		if !deptSeen[did] {
			allDeptIDs = append(allDeptIDs, did)
			deptSeen[did] = true
		}
	}

	log.Printf("Course %d belongs to departments: %v (student depts: %v, teacher depts: %v, merged: %v)", courseID, courseDeptIDs, studentDeptIDs, teacherDeptIDs, allDeptIDs)

	// Collect ALL semester numbers the course appears in across all curricula.
	// A shared course (e.g. 22MA401) may be in sem 4 of AIDS and sem 3 of Math.
	// We need to match a window scoped to ANY of these semesters.
	var courseSemesters []int64
	var hasNonSemesterCard bool
	semRows, err := database.Query(`
		SELECT nc.semester_number, COALESCE(nc.card_type, 'semester')
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		WHERE cc.course_id = ?
	`, courseID)
	if err == nil {
		for semRows.Next() {
			var semNum sql.NullInt64
			var cType string
			if semRows.Scan(&semNum, &cType) == nil {
				if cType != "semester" {
					hasNonSemesterCard = true
				} else if semNum.Valid {
					// Deduplicate
					found := false
					for _, s := range courseSemesters {
						if s == semNum.Int64 {
							found = true
							break
						}
					}
					if !found {
						courseSemesters = append(courseSemesters, semNum.Int64)
					}
				}
			}
		}
		semRows.Close()
	}

	if hasNonSemesterCard {
		log.Printf("Course %d is on a non-semester card — will skip semester check for window matching", courseID)
	}

	log.Printf("Course %d semesters across curricula: %v (nonSemCard=%v)", courseID, courseSemesters, hasNonSemesterCard)

	// DEBUG: Log what we're matching against
	log.Printf("Resolving window for courseID=%d, original teacherID=%s, facultyID=%s, numericUserID=%v, teacherDepts=%v, courseDepts=%v, studentDepts=%v, allDepts=%v, courseSems=%v",
		courseID, teacherID, facultyID, numericUserID, teacherDeptIDs, courseDeptIDs, studentDeptIDs, allDeptIDs, courseSemesters)

	// Prepare typed values for query parameters
	userIDValue := interface{}(nil)
	if numericUserID.Valid {
		userIDValue = numericUserID.Int64
	}

	// Build the query dynamically for both department and semester matching.
	// This ensures a course shared across curricula (different depts/semesters) can match
	// windows scoped to ANY of those departments or semesters.
	var deptClause string
	var semClause string
	var queryArgs []interface{}

	queryArgs = append(queryArgs, facultyID, userIDValue, courseID)

	// Department clause — use merged teacher + course departments
	if len(allDeptIDs) == 0 {
		// No department info at all — match all windows (backward-compatible fallback)
		deptClause = "1=1"
	} else {
		placeholders := make([]string, len(allDeptIDs))
		for i, did := range allDeptIDs {
			placeholders[i] = "?"
			queryArgs = append(queryArgs, did)
		}
		deptClause = fmt.Sprintf("(department_id IS NULL OR department_id = 0 OR department_id IN (%s))",
			strings.Join(placeholders, ","))
	}

	// Semester clause
	if hasNonSemesterCard || len(courseSemesters) == 0 {
		// Non-semester card or no semester info — match all windows (backward-compatible fallback)
		semClause = "(semester IS NULL OR semester = 0 OR 1=1)"
	} else {
		placeholders := make([]string, len(courseSemesters))
		for i, s := range courseSemesters {
			placeholders[i] = "?"
			queryArgs = append(queryArgs, s)
		}
		semClause = fmt.Sprintf("(semester IS NULL OR semester = 0 OR semester IN (%s))",
			strings.Join(placeholders, ","))
	}

	query := fmt.Sprintf(`
		SELECT id, start_at, end_at, enabled
		FROM mark_entry_windows
		WHERE ((teacher_id IS NULL AND user_id IS NULL) OR teacher_id = ? OR user_id = ?)
		  AND (course_id IS NULL OR course_id = ?)
		  AND %s
		  AND %s
		ORDER BY
		  (teacher_id IS NOT NULL) DESC,
		  (user_id IS NOT NULL) DESC,
		  (course_id IS NOT NULL) DESC,
		  (department_id IS NOT NULL) DESC,
		  (semester IS NOT NULL) DESC,
		  updated_at DESC
		LIMIT 25
	`, deptClause, semClause)

	rows, rowErr := database.Query(query, queryArgs...)
	if rowErr != nil {
		if rowErr == sql.ErrNoRows {
			log.Printf("No matching window rule found")
			return false, nil, nil, nil
		}
		return false, nil, nil, rowErr
	}
	defer rows.Close()

	nowLocal := time.Now()
	nowUTC := nowLocal.UTC()

	var activeWindowIDs []int

	for rows.Next() {
		var windowID int
		var startAt time.Time
		var endAt time.Time
		var enabledInt int
		if err := rows.Scan(&windowID, &startAt, &endAt, &enabledInt); err != nil {
			return false, nil, nil, err
		}

		if enabledInt != 1 {
			continue
		}

		inLocal := !nowLocal.Before(startAt) && !nowLocal.After(endAt)
		inUTC := !nowUTC.Before(startAt.UTC()) && !nowUTC.After(endAt.UTC())
		if inLocal || inUTC {
			activeWindowIDs = append(activeWindowIDs, windowID)
		}
	}

	if len(activeWindowIDs) == 0 {
		log.Printf("No matching active window rule found")
		return false, nil, nil, nil
	}

	log.Printf("Found %d active window(s): %v", len(activeWindowIDs), activeWindowIDs)

	// Merge allowed component IDs from ALL active windows
	componentSet := make(map[int]bool)
	anyWindowHasNoComponents := false

	for _, wid := range activeWindowIDs {
		componentRows, err := database.Query(`
			SELECT assessment_component_id
			FROM mark_entry_window_components
			WHERE window_id = ?
		`, wid)
		if err != nil {
			log.Printf("Error loading components for window %d: %v", wid, err)
			continue
		}

		hasComponents := false
		for componentRows.Next() {
			var componentID int
			if err := componentRows.Scan(&componentID); err == nil {
				componentSet[componentID] = true
				hasComponents = true
			}
		}
		componentRows.Close()

		if !hasComponents {
			// A window with no components means "all components allowed"
			anyWindowHasNoComponents = true
		}
	}
	var mergedComponents []int
	if anyWindowHasNoComponents {
		// If any window allows all components, return empty = all allowed
		log.Printf("At least one window allows all components — returning empty (all allowed)")
	} else {
		for cid := range componentSet {
			mergedComponents = append(mergedComponents, cid)
		}
	}

	log.Printf("Merged components from %d windows: %v (empty = all allowed)", len(activeWindowIDs), mergedComponents)
	return true, activeWindowIDs, mergedComponents, nil
}

func resolveMarkEntryWindowWithSelection(courseID int, teacherID string, selectedWindowID int) (bool, int, []int, error) {
	windowOpen, windowIDs, mergedComponents, err := resolveMarkEntryWindow(courseID, teacherID)
	if err != nil {
		return false, 0, nil, err
	}
	if !windowOpen || len(windowIDs) == 0 {
		return false, 0, nil, nil
	}

	resolvedWindowID := windowIDs[0]
	if selectedWindowID > 0 {
		found := false
		for _, windowID := range windowIDs {
			if windowID == selectedWindowID {
				resolvedWindowID = selectedWindowID
				found = true
				break
			}
		}
		if !found {
			log.Printf("Selected window %d is not applicable for courseID=%d teacherID=%s", selectedWindowID, courseID, teacherID)
			return false, 0, nil, nil
		}

		database := db.DB
		if database == nil {
			return false, 0, nil, sql.ErrConnDone
		}

		selectedComponents := make([]int, 0)
		componentRows, queryErr := database.Query(`
			SELECT assessment_component_id
			FROM mark_entry_window_components
			WHERE window_id = ?
		`, resolvedWindowID)
		if queryErr == nil {
			for componentRows.Next() {
				var componentID int
				if scanErr := componentRows.Scan(&componentID); scanErr == nil {
					selectedComponents = append(selectedComponents, componentID)
				}
			}
			componentRows.Close()
		}

		return true, resolvedWindowID, selectedComponents, nil
	}

	return true, resolvedWindowID, mergedComponents, nil
}

func GetApplicableMarkEntryWindows(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	teacherID := strings.TrimSpace(r.URL.Query().Get("teacher_id"))
	courseIDStr := strings.TrimSpace(r.URL.Query().Get("course_id"))
	if teacherID == "" || courseIDStr == "" {
		http.Error(w, "teacher_id and course_id are required", http.StatusBadRequest)
		return
	}

	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil || courseID <= 0 {
		http.Error(w, "invalid course_id", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	windowOpen, windowIDs, _, resolveErr := resolveMarkEntryWindow(courseID, teacherID)
	if resolveErr != nil {
		http.Error(w, "failed to load applicable windows", http.StatusInternalServerError)
		return
	}
	if !windowOpen || len(windowIDs) == 0 {
		json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}

	items := make([]map[string]interface{}, 0, len(windowIDs))
	for _, windowID := range windowIDs {
		var startAt time.Time
		var endAt time.Time
		var windowName sql.NullString
		var departmentID sql.NullInt64
		detailErr := database.QueryRow(`
			SELECT start_at, end_at, window_name, department_id
			FROM mark_entry_windows
			WHERE id = ? AND enabled = 1
		`, windowID).Scan(&startAt, &endAt, &windowName, &departmentID)
		if detailErr != nil {
			continue
		}

		componentIDs := make([]int, 0)
		compRows, compErr := database.Query(`
			SELECT assessment_component_id
			FROM mark_entry_window_components
			WHERE window_id = ?
		`, windowID)
		if compErr == nil {
			for compRows.Next() {
				var componentID int
				if compScanErr := compRows.Scan(&componentID); compScanErr == nil {
					componentIDs = append(componentIDs, componentID)
				}
			}
			compRows.Close()
		}

		item := map[string]interface{}{
			"id":            windowID,
			"window_name":   strings.TrimSpace(windowName.String),
			"start_at":      startAt.Local().Format(markEntryTimeLayout),
			"end_at":        endAt.Local().Format(markEntryTimeLayout),
			"component_ids": componentIDs,
		}
		if departmentID.Valid {
			item["department_id"] = int(departmentID.Int64)
		} else {
			item["department_id"] = nil
		}
		items = append(items, item)
	}

	json.NewEncoder(w).Encode(items)
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

	// Check if filtering for user-only windows or specific teacher
	userOnly := r.URL.Query().Get("user_only") == "true"
	teacherID := r.URL.Query().Get("teacher_id")

	log.Printf("[GetAllMarkEntryWindows] Request: userOnly=%v, teacherID=%s", userOnly, teacherID)

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
				COALESCE(w.window_name, '') as window_name,
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
	} else if teacherID != "" {
		// Get windows that apply to this teacher
		// Match windows by: direct teacher assignment, course assignment, or general windows
		query = `
			SELECT DISTINCT
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
				w.enabled,
				COALESCE(w.window_name, '') as window_name
			FROM mark_entry_windows w
			LEFT JOIN teachers t ON w.teacher_id = t.faculty_id
			LEFT JOIN departments d ON w.department_id = d.id
			LEFT JOIN courses c ON w.course_id = c.id
			WHERE w.enabled = 1 
			AND (
				w.teacher_id = ?
				OR w.course_id IN (
					SELECT course_id FROM teacher_course_allocation WHERE teacher_id = ?
				)
				OR w.teacher_id IS NULL
			)
			ORDER BY 
				CASE WHEN w.end_at > NOW() THEN 0 ELSE 1 END,
				w.start_at DESC
		`
	} else {
		// Get all windows with teacher_id set (teacher windows)
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
				w.enabled,
				COALESCE(w.window_name, '') as window_name
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

	var rows *sql.Rows
	var err error

	if teacherID != "" && !userOnly {
		rows, err = database.Query(query, teacherID, teacherID)
	} else {
		rows, err = database.Query(query)
	}
	if err != nil {
		log.Printf("[GetAllMarkEntryWindows] Error fetching windows: %v", err)
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
		WindowName     string    `json:"window_name"`
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
				&window.WindowName,
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
				&window.WindowName,
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
		log.Printf("[GetAllMarkEntryWindows] Error iterating windows: %v", err)
		http.Error(w, "Error processing windows", http.StatusInternalServerError)
		return
	}

	if windows == nil {
		windows = []WindowWithDetails{}
	}

	log.Printf("[GetAllMarkEntryWindows] Returning %d windows for teacherID=%s", len(windows), teacherID)
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

	log.Printf("[UpdateMarkEntryWindow] incoming id=%d teacher=%v user=%v dept=%v sem=%v course=%v window_name=%q enabled=%v",
		windowID,
		request.TeacherID,
		request.UserID,
		request.DepartmentID,
		request.Semester,
		request.CourseID,
		request.WindowName,
		request.Enabled,
	)

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

	teacherIDValue := interface{}(nil)
	if request.TeacherID != nil && strings.TrimSpace(*request.TeacherID) != "" {
		teacherIDValue = strings.TrimSpace(*request.TeacherID)
	}

	userIDValue := interface{}(nil)
	if request.UserID != nil && strings.TrimSpace(*request.UserID) != "" {
		trimmedUserID := strings.TrimSpace(*request.UserID)
		var numericUserID int
		lookupErr := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, trimmedUserID).Scan(&numericUserID)
		if lookupErr == nil {
			userIDValue = numericUserID
		} else {
			userIDValue = trimmedUserID
		}
	}

	departmentIDValue := interface{}(nil)
	if request.DepartmentID != nil && *request.DepartmentID > 0 {
		departmentIDValue = *request.DepartmentID
	}

	semesterValue := interface{}(nil)
	if request.Semester != nil && *request.Semester > 0 {
		semesterValue = *request.Semester
	}

	courseIDValue := interface{}(nil)
	if request.CourseID != nil && *request.CourseID > 0 {
		courseIDValue = *request.CourseID
	}

	// Safety net: if update request carries no scope fields at all,
	// preserve the existing scope instead of writing an invalid "unknown scope" row.
	if teacherIDValue == nil && userIDValue == nil && departmentIDValue == nil && semesterValue == nil && courseIDValue == nil {
		var existingTeacherID sql.NullString
		var existingUserID sql.NullInt64
		var existingDepartmentID sql.NullInt64
		var existingSemester sql.NullInt64
		var existingCourseID sql.NullInt64

		err := database.QueryRow(`
			SELECT teacher_id, user_id, department_id, semester, course_id
			FROM mark_entry_windows
			WHERE id = ?
		`, windowID).Scan(&existingTeacherID, &existingUserID, &existingDepartmentID, &existingSemester, &existingCourseID)
		if err == nil {
			if existingTeacherID.Valid && strings.TrimSpace(existingTeacherID.String) != "" {
				teacherIDValue = strings.TrimSpace(existingTeacherID.String)
			}
			if existingUserID.Valid {
				userIDValue = int(existingUserID.Int64)
			}
			if existingDepartmentID.Valid {
				departmentIDValue = int(existingDepartmentID.Int64)
			}
			if existingSemester.Valid {
				semesterValue = int(existingSemester.Int64)
			}
			if existingCourseID.Valid {
				courseIDValue = int(existingCourseID.Int64)
			}
		}
	}

	// Update window
	updateQuery := `
		UPDATE mark_entry_windows
		SET teacher_id = ?, user_id = ?, department_id = ?, semester = ?, course_id = ?,
		    start_at = ?, end_at = ?, enabled = ?, window_name = ?
		WHERE id = ?
	`

	_, err = database.Exec(updateQuery,
		teacherIDValue,
		userIDValue,
		departmentIDValue,
		semesterValue,
		courseIDValue,
		startAt,
		endAt,
		request.Enabled,
		request.WindowName,
		windowID,
	)
	if err != nil {
		log.Printf("Error updating window: %v", err)
		http.Error(w, "Failed to update window", http.StatusInternalServerError)
		return
	}

	log.Printf("[UpdateMarkEntryWindow] saved id=%d teacher=%v user=%v dept=%v sem=%v course=%v",
		windowID,
		teacherIDValue,
		userIDValue,
		departmentIDValue,
		semesterValue,
		courseIDValue,
	)

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

// GetWindowsPendingSubmissions returns windows with teachers who haven't submitted marks.
// Query params:
//   - status=active  (default) — currently live windows (start_at <= now <= end_at)
//   - status=closed  — expired windows (end_at < now) so admins can see who missed the deadline
func GetWindowsPendingSubmissions(w http.ResponseWriter, r *http.Request) {
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

	// Determine whether to fetch active or closed windows
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

	// Fetch all enabled windows — we'll filter by time in Go to avoid timezone mismatches
	windowQuery := `
		SELECT 
			w.id, w.teacher_id, w.department_id, w.semester, w.course_id,
			w.start_at, w.end_at, w.enabled,
			COALESCE(d.department_name, '') as department_name,
			COALESCE(c.course_code, '') as course_code,
			COALESCE(c.course_name, '') as course_name,
			COALESCE(w.window_name, '') as window_name
		FROM mark_entry_windows w
		LEFT JOIN departments d ON w.department_id = d.id
		LEFT JOIN courses c ON w.course_id = c.id
		WHERE w.enabled = 1
		ORDER BY w.end_at DESC
	`

	rows, err := database.Query(windowQuery)
	if err != nil {
		log.Printf("Error fetching windows: %v", err)
		http.Error(w, "Failed to fetch windows", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	now := time.Now()
	log.Printf("[PendingSubmissions] status=%s, now(local)=%v, now(utc)=%v", status, now, now.UTC())

	type PendingTeacher struct {
		TeacherID   string `json:"teacher_id"`
		TeacherName string `json:"teacher_name"`
		CourseID    int    `json:"course_id"`
		CourseCode  string `json:"course_code"`
		CourseName  string `json:"course_name"`
		Submitted   bool   `json:"submitted"`
		SubmittedAt string `json:"submitted_at,omitempty"`
	}

	type WindowWithPending struct {
		WindowID       int              `json:"window_id"`
		WindowName     string           `json:"window_name"`
		DepartmentID   *int             `json:"department_id"`
		DepartmentName string           `json:"department_name"`
		Semester       *int             `json:"semester"`
		CourseID       *int             `json:"course_id"`
		CourseCode     string           `json:"course_code"`
		CourseName     string           `json:"course_name"`
		StartAt        string           `json:"start_at"`
		EndAt          string           `json:"end_at"`
		Pending        []PendingTeacher `json:"pending_teachers"`
		Completed      []PendingTeacher `json:"completed_teachers"`
	}

	var result []WindowWithPending

	for rows.Next() {
		var windowID int
		var teacherID sql.NullString
		var deptID, semester, courseID sql.NullInt64
		var startAt, endAt time.Time
		var enabled int
		var deptName, courseCode, courseName, windowName sql.NullString

		err := rows.Scan(
			&windowID,
			&teacherID,
			&deptID,
			&semester,
			&courseID,
			&startAt,
			&endAt,
			&enabled,
			&deptName,
			&courseCode,
			&courseName,
			&windowName,
		)
		if err != nil {
			log.Printf("Error scanning window row: %v", err)
			continue
		}

		// Filter by time in Go — avoids MySQL timezone mismatches
		log.Printf("[PendingSubmissions] Window %d: startAt=%v endAt=%v now=%v", windowID, startAt, endAt, now)
		switch status {
		case "closed":
			if !endAt.Before(now) {
				// Window hasn't expired yet, skip for closed status
				continue
			}
		default: // "active"
			if !(!now.Before(startAt) && !now.After(endAt)) {
				// Window is not currently active, skip
				continue
			}
		}

		windowData := WindowWithPending{
			WindowID:       windowID,
			WindowName:     windowName.String,
			DepartmentName: deptName.String,
			CourseCode:     courseCode.String,
			CourseName:     courseName.String,
			StartAt:        startAt.Format(time.RFC3339),
			EndAt:          endAt.Format(time.RFC3339),
			Pending:        []PendingTeacher{},
			Completed:      []PendingTeacher{},
		}

		// Convert nullable fields to pointers for the response
		if deptID.Valid {
			v := int(deptID.Int64)
			windowData.DepartmentID = &v
		}
		if semester.Valid {
			v := int(semester.Int64)
			windowData.Semester = &v
		}
		if courseID.Valid {
			v := int(courseID.Int64)
			windowData.CourseID = &v
		}

		log.Printf("[PendingSubmissions] Processing window %d: teacher=%v dept=%v sem=%v course=%v",
			windowID, teacherID, deptID, semester, courseID)

		// Build query dynamically based on which scope fields have meaningful values
		var teacherArgs []interface{}

		// Determine which scope fields are meaningful (non-nil and non-zero where applicable)
		hasTeacher := teacherID.Valid && teacherID.String != ""
		hasCourse := courseID.Valid && courseID.Int64 != 0
		hasDepartment := deptID.Valid && deptID.Int64 != 0
		hasSemester := semester.Valid && semester.Int64 != 0

		// Must have at least one meaningful scope field
		if !hasTeacher && !hasCourse && !hasDepartment && !hasSemester {
			log.Printf("Skipping window %d: no meaningful scope fields", windowID)
			continue
		}

		// Build query parts dynamically — get ALL teachers in scope (no submission filter)
		extraJoins := ""
		extraWhere := ""

		if hasTeacher {
			extraWhere += " AND tca.teacher_id = ?"
			teacherArgs = append(teacherArgs, teacherID.String)
		}

		if hasCourse {
			extraWhere += " AND tca.course_id = ?"
			teacherArgs = append(teacherArgs, courseID.Int64)
		}

		if hasDepartment {
			extraJoins += " INNER JOIN department_teachers dt ON dt.teacher_id = tca.teacher_id AND dt.status = 1"
			extraWhere += " AND dt.department_id = ?"
			teacherArgs = append(teacherArgs, deptID.Int64)
		}

		if hasSemester {
			extraJoins += " INNER JOIN curriculum_courses cc ON cc.course_id = tca.course_id"
			extraJoins += " INNER JOIN normal_cards nc ON cc.semester_id = nc.id"
			extraWhere += " AND nc.semester_number = ?"
			teacherArgs = append(teacherArgs, semester.Int64)
		}

		teacherQuery := fmt.Sprintf(`
			SELECT DISTINCT
				tca.teacher_id,
				COALESCE(t.name, tca.teacher_id) as teacher_name,
				tca.course_id,
				c.course_code,
				c.course_name
			FROM teacher_course_allocation tca
			LEFT JOIN teachers t ON t.faculty_id = tca.teacher_id
			LEFT JOIN courses c ON c.id = tca.course_id
			%s
			WHERE tca.is_active = 1
			%s
		`, extraJoins, extraWhere)

		teacherRows, err := database.Query(teacherQuery, teacherArgs...)
		if err != nil {
			log.Printf("Error fetching teachers for window %d: %v", windowID, err)
			continue
		}

		// Collect all teachers in scope
		type teacherCourse struct {
			TeacherID   string
			TeacherName string
			CourseID    int
			CourseCode  string
			CourseName  string
		}
		var allTeachers []teacherCourse

		for teacherRows.Next() {
			var tc teacherCourse
			err := teacherRows.Scan(&tc.TeacherID, &tc.TeacherName, &tc.CourseID, &tc.CourseCode, &tc.CourseName)
			if err != nil {
				log.Printf("Error scanning teacher row: %v", err)
				continue
			}
			allTeachers = append(allTeachers, tc)
		}
		teacherRows.Close()

		if len(allTeachers) == 0 {
			log.Printf("Window %d: no teachers found in scope", windowID)
			continue
		}

		// Fetch completed teachers directly from mark_submissions for this window
		submissionRows, err := database.Query(`
			SELECT ms.teacher_id, COALESCE(t.name, ms.teacher_id), ms.course_id,
			       COALESCE(c.course_code, ''), COALESCE(c.course_name, ''), ms.submitted_at
			FROM mark_submissions ms
			LEFT JOIN teachers t ON t.faculty_id = ms.teacher_id
			LEFT JOIN courses c ON c.id = ms.course_id
			WHERE ms.window_id = ?
		`, windowID)
		if err != nil {
			log.Printf("Error fetching submissions for window %d: %v", windowID, err)
			continue
		}

		// Build completed list and a set for quick lookup
		submittedSet := make(map[string]bool)
		for submissionRows.Next() {
			var tid, tname, ccode, cname string
			var cid int
			var submittedAt time.Time
			if err := submissionRows.Scan(&tid, &tname, &cid, &ccode, &cname, &submittedAt); err != nil {
				log.Printf("Error scanning submission row: %v", err)
				continue
			}
			key := fmt.Sprintf("%s|%d", tid, cid)
			submittedSet[key] = true
			windowData.Completed = append(windowData.Completed, PendingTeacher{
				TeacherID:   tid,
				TeacherName: tname,
				CourseID:    cid,
				CourseCode:  ccode,
				CourseName:  cname,
				Submitted:   true,
				SubmittedAt: submittedAt.Format(time.RFC3339),
			})
		}
		submissionRows.Close()

		log.Printf("Window %d: %d teachers in scope, %d submissions found", windowID, len(allTeachers), len(submittedSet))

		// Pending = teachers in scope who are NOT in mark_submissions
		for _, tc := range allTeachers {
			key := fmt.Sprintf("%s|%d", tc.TeacherID, tc.CourseID)
			if !submittedSet[key] {
				windowData.Pending = append(windowData.Pending, PendingTeacher{
					TeacherID:   tc.TeacherID,
					TeacherName: tc.TeacherName,
					CourseID:    tc.CourseID,
					CourseCode:  tc.CourseCode,
					CourseName:  tc.CourseName,
					Submitted:   false,
				})
			}
		}

		// Include windows that have any teachers at all (pending or completed)
		if len(windowData.Pending) > 0 || len(windowData.Completed) > 0 {
			result = append(result, windowData)
		}
	}

	if result == nil {
		result = []WindowWithPending{}
	}
	json.NewEncoder(w).Encode(result)
}

// ExtendWindowForTeachers creates individual teacher-scoped windows for selected
// teacher+course pairs from a closed window, effectively re-opening the window
// only for those specific teachers.
func ExtendWindowForTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	type TeacherCourse struct {
		TeacherID string `json:"teacher_id"`
		CourseID  int    `json:"course_id"`
	}

	type ExtendRequest struct {
		SourceWindowID int             `json:"source_window_id"`
		Teachers       []TeacherCourse `json:"teachers"`
		NewEndAt       string          `json:"new_end_at"`
	}

	var req ExtendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Teachers) == 0 {
		http.Error(w, "No teachers selected", http.StatusBadRequest)
		return
	}

	if req.NewEndAt == "" {
		http.Error(w, "New end date is required", http.StatusBadRequest)
		return
	}

	newEndAt, err := parseDateTime(req.NewEndAt)
	if err != nil {
		log.Printf("Error parsing new end date '%s': %v", req.NewEndAt, err)
		http.Error(w, "Invalid end date format", http.StatusBadRequest)
		return
	}

	if !newEndAt.After(time.Now()) {
		http.Error(w, "New end date must be in the future", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Fetch the source window to copy component associations
	var sourceComponents []int
	compRows, err := database.Query(
		`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`,
		req.SourceWindowID,
	)
	if err == nil {
		for compRows.Next() {
			var cid int
			if err := compRows.Scan(&cid); err == nil {
				sourceComponents = append(sourceComponents, cid)
			}
		}
		compRows.Close()
	}

	// Start now
	startAt := time.Now()

	created := 0
	var errors []string

	for _, tc := range req.Teachers {
		// Create an individual teacher+course scoped window
		result, err := database.Exec(`
			INSERT INTO mark_entry_windows
			(teacher_id, department_id, semester, course_id, start_at, end_at, enabled)
			VALUES (?, NULL, NULL, ?, ?, ?, 1)
		`, tc.TeacherID, tc.CourseID, startAt, newEndAt)
		if err != nil {
			log.Printf("Error creating extension window for teacher %s course %d: %v", tc.TeacherID, tc.CourseID, err)
			errors = append(errors, fmt.Sprintf("teacher %s course %d: %v", tc.TeacherID, tc.CourseID, err))
			continue
		}

		// Copy component associations from the source window
		if len(sourceComponents) > 0 {
			windowID, err := result.LastInsertId()
			if err == nil {
				for _, compID := range sourceComponents {
					database.Exec(
						`INSERT INTO mark_entry_window_components (window_id, assessment_component_id) VALUES (?, ?)`,
						windowID, compID,
					)
				}
			}
		}
		created++
	}

	if created == 0 {
		http.Error(w, fmt.Sprintf("Failed to create any extension windows: %v", errors), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Extended window for %d teacher(s)", created),
		"created": created,
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	json.NewEncoder(w).Encode(resp)
}
