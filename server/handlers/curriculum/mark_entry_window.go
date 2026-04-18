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

const markEntryTimeLayout = "2006-01-02T15:04"

func sanitizeDepartmentIDs(ids []int) []int {
	seen := make(map[int]struct{})
	cleaned := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleaned = append(cleaned, id)
	}
	return cleaned
}

func ensureWindowDepartmentsTable(database *sql.DB) error {
	_, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS mark_entry_window_departments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			window_id INT NOT NULL,
			department_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_window_department (window_id, department_id),
			KEY idx_window_id (window_id),
			KEY idx_department_id (department_id),
			CONSTRAINT fk_mewd_window FOREIGN KEY (window_id) REFERENCES mark_entry_windows(id) ON DELETE CASCADE,
			CONSTRAINT fk_mewd_department FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE
		)
	`)
	return err
}

func replaceWindowDepartmentsTx(tx *sql.Tx, windowID int, departmentIDs []int) error {
	if _, err := tx.Exec(`DELETE FROM mark_entry_window_departments WHERE window_id = ?`, windowID); err != nil {
		return err
	}

	for _, departmentID := range sanitizeDepartmentIDs(departmentIDs) {
		if _, err := tx.Exec(`
			INSERT INTO mark_entry_window_departments (window_id, department_id)
			SELECT ?, d.id
			FROM departments d
			WHERE d.id = ?
		`, windowID, departmentID); err != nil {
			return err
		}
	}

	return nil
}

func replaceWindowDepartments(database *sql.DB, windowID int, departmentIDs []int) error {
	if _, err := database.Exec(`DELETE FROM mark_entry_window_departments WHERE window_id = ?`, windowID); err != nil {
		return err
	}

	for _, departmentID := range sanitizeDepartmentIDs(departmentIDs) {
		if _, err := database.Exec(`
			INSERT INTO mark_entry_window_departments (window_id, department_id)
			SELECT ?, d.id
			FROM departments d
			WHERE d.id = ?
		`, windowID, departmentID); err != nil {
			return err
		}
	}

	return nil
}

func getWindowDepartments(database *sql.DB, windowID int, fallbackDepartmentID *int) ([]int, []string) {
	deptIDs := make([]int, 0)
	deptNames := make([]string, 0)

	rows, err := database.Query(`
		SELECT d.id, d.department_name
		FROM mark_entry_window_departments mwd
		INNER JOIN departments d ON d.id = mwd.department_id
		WHERE mwd.window_id = ?
		ORDER BY d.department_name
	`, windowID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name sql.NullString
			if scanErr := rows.Scan(&id, &name); scanErr == nil {
				deptIDs = append(deptIDs, id)
				deptNames = append(deptNames, strings.TrimSpace(name.String))
			}
		}
	}

	if len(deptIDs) == 0 && fallbackDepartmentID != nil && *fallbackDepartmentID > 0 {
		deptIDs = append(deptIDs, *fallbackDepartmentID)
		var fallbackName sql.NullString
		if err := database.QueryRow(`SELECT department_name FROM departments WHERE id = ?`, *fallbackDepartmentID).Scan(&fallbackName); err == nil {
			deptNames = append(deptNames, strings.TrimSpace(fallbackName.String))
		}
	}

	return deptIDs, deptNames
}

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

	if err := ensureWindowDepartmentsTable(database); err != nil {
		log.Printf("[GetAllMarkEntryWindows] Error ensuring mark_entry_window_departments table: %v", err)
		http.Error(w, "Failed to initialize selected departments support", http.StatusInternalServerError)
		return
	}

	if err := ensureWindowDepartmentsTable(database); err != nil {
		log.Printf("Error ensuring mark_entry_window_departments table: %v", err)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	query := `
		SELECT id, teacher_id, department_id, semester, course_id, start_at, end_at, enabled, window_name
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

	var windowName sql.NullString

	err := database.QueryRow(query, args...).Scan(
		&window.ID,
		&teacherIDNull,
		&departmentIDNull,
		&semesterNull,
		&courseIDNull,
		&startAt,
		&endAt,
		&enabledInt,
		&windowName,
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
	if windowName.Valid {
		window.WindowName = windowName.String
	}

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

	departmentIDs := sanitizeDepartmentIDs(req.DepartmentIDs)
	if len(departmentIDs) == 0 && req.DepartmentID != nil && *req.DepartmentID > 0 {
		departmentIDs = []int{*req.DepartmentID}
	}

	departmentValue := sql.NullInt64{}
	if len(departmentIDs) == 1 {
		departmentValue = sql.NullInt64{Int64: int64(departmentIDs[0]), Valid: true}
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

	tx, err := database.Begin()
	if err != nil {
		log.Printf("Error starting transaction for mark entry windows: %v", err)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	result, execErr := tx.Exec(`
		INSERT INTO mark_entry_windows
		(teacher_id, department_id, semester, course_id, start_at, end_at, enabled, window_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, teacherValue, departmentValue, semesterValue, courseValue, startAt, endAt, enabledValue, req.WindowName)
	if execErr != nil {
		_ = tx.Rollback()
		log.Printf("Error saving mark entry window: %v", execErr)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	windowID, idErr := result.LastInsertId()
	if idErr != nil {
		_ = tx.Rollback()
		log.Printf("Error getting inserted window ID: %v", idErr)
		http.Error(w, "Failed to save mark entry window", http.StatusInternalServerError)
		return
	}

	if err := replaceWindowDepartmentsTx(tx, int(windowID), departmentIDs); err != nil {
		// Department mapping is supplementary metadata; do not fail window creation.
		log.Printf("Warning: unable to persist selected departments for window %d (ids=%v): %v", windowID, departmentIDs, err)
	}

	if len(req.ComponentIDs) > 0 {
		for _, componentID := range req.ComponentIDs {
			_, componentErr := tx.Exec(`
				INSERT INTO mark_entry_window_components (window_id, assessment_component_id)
				VALUES (?, ?)
			`, windowID, componentID)
			if componentErr != nil {
				_ = tx.Rollback()
				log.Printf("Error saving window component for window %d: %v", windowID, componentErr)
				http.Error(w, "Failed to save window components", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing mark entry windows transaction: %v", err)
		http.Error(w, "Failed to save mark entry windows", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":         "Mark entry window saved",
		"window_ids":      []int{int(windowID)},
		"windows_created": 1,
	}
	json.NewEncoder(w).Encode(response)
}
func resolveMarkEntryWindow(courseID int, teacherID string) (bool, []int, []int, error) {
	database := db.DB
	if database == nil {
		return false, nil, nil, sql.ErrConnDone
	}

	// Convert numeric teacher ID to faculty_id if needed
	// All allocation tables (teacher_course_allocation, department_teachers, mark_entry_windows) use faculty_id
	var facultyID string
	var teacherEmail sql.NullString
	err := database.QueryRow(`SELECT faculty_id, email FROM teachers WHERE id = ? OR faculty_id = ?`, teacherID, teacherID).Scan(&facultyID, &teacherEmail)
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

	// Try to lookup numeric user ID from different identifiers (username/email)
	var numericUserID sql.NullInt64
	userLookupErr := database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, facultyID).Scan(&numericUserID)
	if userLookupErr == sql.ErrNoRows && teacherEmail.Valid {
		userLookupErr = database.QueryRow(`SELECT id FROM users WHERE email = ? AND is_active = 1`, teacherEmail.String).Scan(&numericUserID)
	}
	if userLookupErr == sql.ErrNoRows && teacherID != facultyID {
		// Try teacherID as username directly if it differs
		userLookupErr = database.QueryRow(`SELECT id FROM users WHERE username = ? AND is_active = 1`, teacherID).Scan(&numericUserID)
	}
	if userLookupErr != nil && userLookupErr != sql.ErrNoRows {
		log.Printf("Error looking up user ID for %s: %v", facultyID, userLookupErr)
		return false, nil, nil, userLookupErr
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

	// Also look up which department(s) the COURSE belongs to via curriculum.department_id.
	// This must not depend on departments.current_curriculum_id, because a course can belong
	// to another valid curriculum of the same department.
	var courseDeptIDs []int64
	deptRows, err := database.Query(`
		SELECT DISTINCT d.id
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN departments d ON d.current_curriculum_id = nc.curriculum_id
		WHERE cc.course_id = ?
		  AND d.id IS NOT NULL
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

	// Collect semester/card info the course appears in across all curricula.
	// Semester matching is applicable only for semester cards.
	var courseSemesters []int64
	studentSemesterSet := make(map[int64]bool)
	var hasNonSemesterCard bool
	var hasSemesterCard bool
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
				if strings.EqualFold(strings.TrimSpace(cType), "semester") {
					hasSemesterCard = true
					if semNum.Valid {
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
				} else {
					hasNonSemesterCard = true
				}
			}
		}
		semRows.Close()
	}

	if hasNonSemesterCard {
		log.Printf("Course %d is on a non-semester card — semester checks apply only to semester cards", courseID)
	}

	// Also include semester footprint from allocated students for this teacher+course.
	// This is important for dept+semester windows where curriculum mapping can be incomplete.
	studentSemRows, studentSemErr := database.Query(`
		SELECT DISTINCT COALESCE(ac.current_semester, 0)
		FROM course_student_teacher_allocation csta
		JOIN students s ON csta.student_id = s.id
		LEFT JOIN academic_calendar ac ON ac.id = s.year
		WHERE csta.course_id = ?
		  AND csta.teacher_id = ?
		  AND s.status = 1
	`, courseID, facultyID)
	if studentSemErr == nil {
		for studentSemRows.Next() {
			var currentSem int64
			if studentSemRows.Scan(&currentSem) == nil {
				if currentSem > 0 {
					studentSemesterSet[currentSem] = true
				}
			}
		}
		studentSemRows.Close()
	}

	for sem := range studentSemesterSet {
		found := false
		for _, existing := range courseSemesters {
			if existing == sem {
				found = true
				break
			}
		}
		if !found {
			courseSemesters = append(courseSemesters, sem)
		}
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

	// Department clause
	// For semester courses, keep existing dept-scope behavior.
	// For non-semester courses, additionally allow explicit curriculum->department matching
	// against the opened window department.
	deptIDsForScope := allDeptIDs
	if !hasSemesterCard && hasNonSemesterCard {
		deptIDsForScope = courseDeptIDs
	}

	baseDeptClause := "1=1"
	if len(deptIDsForScope) == 0 {
		// No department info at all — match all windows (backward-compatible fallback)
		baseDeptClause = "1=1"
	} else {
		placeholders := make([]string, len(deptIDsForScope))
		for i, did := range deptIDsForScope {
			placeholders[i] = "?"
			queryArgs = append(queryArgs, did)
		}
		baseDeptClause = fmt.Sprintf("(w.department_id IS NULL OR w.department_id = 0 OR w.department_id IN (%s))",
			strings.Join(placeholders, ","))
	}

	deptClause = baseDeptClause
	if hasNonSemesterCard {
		nonSemesterDeptClause := `EXISTS (
			SELECT 1
			FROM curriculum_courses ccv
			JOIN normal_cards ncv ON ccv.semester_id = ncv.id
			JOIN departments dv ON dv.current_curriculum_id = ncv.curriculum_id
			WHERE ccv.course_id = ?
			  AND LOWER(COALESCE(ncv.card_type, 'semester')) <> 'semester'
			  AND (COALESCE(w.department_id, 0) = 0 OR dv.id = w.department_id)
		)`
		queryArgs = append(queryArgs, courseID)
		deptClause = fmt.Sprintf("(%s OR %s)", baseDeptClause, nonSemesterDeptClause)
	}

	// Semester clause
	// Rule:
	// 1) Semester cards must match window semester (or all-semester window).
	// 2) Non-semester cards (vertical, etc.) must match window department, and semester is ignored.
	var semesterOnlyClause string
	if !hasSemesterCard && hasNonSemesterCard {
		// Course exists only on non-semester cards for matching scope.
		semesterOnlyClause = "(w.semester IS NULL OR w.semester = 0)"
	} else if len(courseSemesters) == 0 {
		// No semester footprint for semester cards: allow only all-semester windows.
		semesterOnlyClause = "(w.semester IS NULL OR w.semester = 0)"
	} else {
		placeholders := make([]string, len(courseSemesters))
		for i, s := range courseSemesters {
			placeholders[i] = "?"
			queryArgs = append(queryArgs, s)
		}
		semesterOnlyClause = fmt.Sprintf("(w.semester IS NULL OR w.semester = 0 OR w.semester IN (%s))",
			strings.Join(placeholders, ","))
	}

	nonSemesterByDeptClause := "0=1"
	if hasNonSemesterCard {
		if len(allDeptIDs) == 0 {
			// If no department context is derivable for this non-semester course,
			// avoid blocking an otherwise valid active window by semester.
			nonSemesterByDeptClause = "1=1"
		} else {
			placeholders := make([]string, len(allDeptIDs))
			for i, did := range allDeptIDs {
				placeholders[i] = "?"
				queryArgs = append(queryArgs, did)
			}
			nonSemesterByDeptClause = fmt.Sprintf("(COALESCE(w.department_id, 0) = 0 OR w.department_id IN (%s))", strings.Join(placeholders, ","))
		}
	}
	semClause = fmt.Sprintf("(%s OR %s)", semesterOnlyClause, nonSemesterByDeptClause)

	query := fmt.Sprintf(`
		SELECT id, start_at, end_at, enabled
		FROM mark_entry_windows w
		WHERE ((w.teacher_id IS NULL AND w.user_id IS NULL) OR w.teacher_id = ? OR w.user_id = ?)
		  AND (w.course_id IS NULL OR w.course_id = ?)
		  AND %s
		  AND %s
		ORDER BY
		  (w.teacher_id IS NOT NULL) DESC,
		  (w.user_id IS NOT NULL) DESC,
		  (w.course_id IS NOT NULL) DESC,
		  (w.department_id IS NOT NULL) DESC,
		  (w.semester IS NOT NULL) DESC,
		  w.updated_at DESC
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
					SELECT course_id FROM teacher_course_history WHERE teacher_id = ? AND record_type = 'course' AND archived_at IS NULL
				)
				OR w.teacher_id IS NULL
			)
			ORDER BY 
				CASE WHEN w.end_at > NOW() THEN 0 ELSE 1 END,
				w.start_at DESC
		`
	} else {
		// Get all windows used in mark entry cards, including teacher, global, and user-assigned windows.
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
			WHERE 1=1
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
		DepartmentIDs  []int     `json:"department_ids,omitempty"`
		DepartmentList []string  `json:"department_names,omitempty"`
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

		window.DepartmentIDs, window.DepartmentList = getWindowDepartments(database, window.ID, window.DepartmentID)
		if len(window.DepartmentList) > 0 {
			window.DepartmentName = strings.Join(window.DepartmentList, ", ")
		}

		// Load components for this window
		compRows, err := database.Query(`
			SELECT assessment_component_id
			FROM mark_entry_window_components
			WHERE window_id = ?
		`, window.ID)
		if err == nil {
			componentIDs := make([]int, 0)
			for compRows.Next() {
				var compID int
				if err := compRows.Scan(&compID); err == nil {
					componentIDs = append(componentIDs, compID)
				}
			}
			compRows.Close()
			sort.Ints(componentIDs)
			window.Components = componentIDs
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

	if err := ensureWindowDepartmentsTable(database); err != nil {
		log.Printf("Error ensuring mark_entry_window_departments table: %v", err)
		http.Error(w, "Failed to update window", http.StatusInternalServerError)
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

	departmentIDs := sanitizeDepartmentIDs(request.DepartmentIDs)
	if len(departmentIDs) == 0 && request.DepartmentID != nil && *request.DepartmentID > 0 {
		departmentIDs = []int{*request.DepartmentID}
	}

	departmentIDValue := interface{}(nil)
	if len(departmentIDs) == 1 {
		departmentIDValue = departmentIDs[0]
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

	if len(departmentIDs) == 0 {
		existingMappedDepartmentIDs, _ := getWindowDepartments(database, windowID, nil)
		if len(existingMappedDepartmentIDs) > 0 {
			departmentIDs = existingMappedDepartmentIDs
			if len(departmentIDs) == 1 {
				departmentIDValue = departmentIDs[0]
			}
		} else if v, ok := departmentIDValue.(int); ok && v > 0 {
			departmentIDs = []int{v}
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

	if err := replaceWindowDepartments(database, windowID, departmentIDs); err != nil {
		// Department mapping is supplementary metadata; do not fail core window updates.
		log.Printf("Warning: unable to update selected departments for window %d (ids=%v): %v", windowID, departmentIDs, err)
	}

	log.Printf("[UpdateMarkEntryWindow] saved id=%d teacher=%v user=%v dept=%v sem=%v course=%v",
		windowID,
		teacherIDValue,
		userIDValue,
		departmentIDValue,
		semesterValue,
		courseIDValue,
	)

	// Capture existing components to detect whether new components were added.
	oldComponentSet := make(map[int]struct{})
	oldCompRows, oldCompErr := database.Query(`
		SELECT assessment_component_id
		FROM mark_entry_window_components
		WHERE window_id = ?
	`, windowID)
	if oldCompErr != nil {
		log.Printf("Error reading existing components for window %d: %v", windowID, oldCompErr)
		http.Error(w, "Failed to update components", http.StatusInternalServerError)
		return
	}
	for oldCompRows.Next() {
		var componentID int
		if scanErr := oldCompRows.Scan(&componentID); scanErr != nil {
			oldCompRows.Close()
			log.Printf("Error scanning existing component for window %d: %v", windowID, scanErr)
			http.Error(w, "Failed to update components", http.StatusInternalServerError)
			return
		}
		oldComponentSet[componentID] = struct{}{}
	}
	if rowsErr := oldCompRows.Err(); rowsErr != nil {
		oldCompRows.Close()
		log.Printf("Error iterating existing components for window %d: %v", windowID, rowsErr)
		http.Error(w, "Failed to update components", http.StatusInternalServerError)
		return
	}
	oldCompRows.Close()

	newComponentSet := make(map[int]struct{})
	for _, componentID := range request.ComponentIDs {
		if componentID > 0 {
			newComponentSet[componentID] = struct{}{}
		}
	}

	addedComponentIDs := make([]int, 0)
	for componentID := range newComponentSet {
		if _, exists := oldComponentSet[componentID]; !exists {
			addedComponentIDs = append(addedComponentIDs, componentID)
		}
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

	if len(addedComponentIDs) > 0 {
		deleteResult, deleteErr := database.Exec(`DELETE FROM mark_submissions WHERE window_id = ?`, windowID)
		if deleteErr != nil {
			log.Printf("Warning: failed to clear stale submissions for window %d after component additions %v: %v", windowID, addedComponentIDs, deleteErr)
		} else {
			removed, _ := deleteResult.RowsAffected()
			log.Printf("[UpdateMarkEntryWindow] window %d added components %v; cleared %d stale mark_submissions rows", windowID, addedComponentIDs, removed)
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

// getTeacherLearningModes determines which learning mode(s) a teacher teaches for a given course
// by checking the allocated students' learning modes
func getTeacherLearningModes(database *sql.DB, teacherID string, courseID int) []string {
	query := `
		SELECT DISTINCT s.learning_mode_id
		FROM course_student_teacher_allocation csta
		INNER JOIN students s ON s.id = csta.student_id
		WHERE csta.teacher_id = ? AND csta.course_id = ?
	`

	rows, err := database.Query(query, teacherID, courseID)
	if err != nil {
		log.Printf("Error fetching learning modes for teacher %s course %d: %v", teacherID, courseID, err)
		return []string{}
	}
	defer rows.Close()

	modes := []string{}
	hasPBL := false
	hasUAL := false

	for rows.Next() {
		var lmID sql.NullInt64
		if err := rows.Scan(&lmID); err == nil && lmID.Valid {
			if lmID.Int64 == 1 {
				hasUAL = true
			} else if lmID.Int64 == 2 {
				hasPBL = true
			}
		}
	}

	if hasUAL {
		modes = append(modes, "UAL")
	}
	if hasPBL {
		modes = append(modes, "PBL")
	}

	return modes
}

// getLearningModesForComponents determines if a window's components correspond to UAL, PBL, or both.
func getLearningModesForComponents(database *sql.DB, componentIDs []int) (isUAL bool, isPBL bool) {
	if len(componentIDs) == 0 {
		return true, true // No specific components means both are implicitly included
	}

	placeholders := strings.Repeat("?,", len(componentIDs)-1) + "?"
	query := fmt.Sprintf(`
		SELECT DISTINCT COALESCE(mct.learning_mode_id, 0)
		FROM mark_category_types mct
		WHERE mct.id IN (%s)
	`, placeholders)

	args := make([]interface{}, len(componentIDs))
	for i, id := range componentIDs {
		args[i] = id
	}

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("Error checking component learning modes: %v", err)
		return false, false
	}
	defer rows.Close()

	for rows.Next() {
		var learningModeID int
		if err := rows.Scan(&learningModeID); err == nil {
			switch learningModeID {
			case 1:
				isUAL = true
			case 2:
				isPBL = true
			}
		}
	}

	// If no specific UAL/PBL type is found in any component name, assume it could be for both
	if !isUAL && !isPBL {
		return true, true
	}
	return isUAL, isPBL
}

// buildTeacherQuery constructs the query to fetch teachers based on the window's learning modes.
func buildTeacherQuery(isUAL bool, isPBL bool) string {
	baseQuery := `
		SELECT
			t.faculty_id,
			t.name,
			t.department_id,
			d.department_name,
			COUNT(DISTINCT csta.student_id) as student_count,
			SUM(CASE WHEN sm.id IS NOT NULL THEN 1 ELSE 0 END) as submitted_count,
			SUM(CASE WHEN s.learning_mode_id = 1 THEN 1 ELSE 0 END) as ual_student_count,
			SUM(CASE WHEN s.learning_mode_id = 2 THEN 1 ELSE 0 END) as pbl_student_count
		FROM mark_entry_windows w
		JOIN teacher_course_history tca ON (w.course_id IS NULL OR w.course_id = tca.course_id)
			AND tca.record_type = 'course' AND tca.archived_at IS NULL
		JOIN teachers t ON tca.teacher_id = t.faculty_id
		JOIN departments d ON t.department_id = d.id
		JOIN course_student_teacher_allocation csta ON t.faculty_id = csta.teacher_id AND tca.course_id = csta.course_id
		JOIN students s ON csta.student_id = s.id
		LEFT JOIN student_marks sm ON sm.student_id = csta.student_id AND sm.course_id = csta.course_id AND sm.teacher_id = t.faculty_id AND sm.assessment_component_id IN (
			SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = w.id
		)
		WHERE w.id = ?
		GROUP BY t.faculty_id, t.name, t.department_id, d.department_name
	`

	var havingClauses []string
	if isUAL && !isPBL {
		havingClauses = append(havingClauses, "ual_student_count > 0")
	}
	if isPBL && !isUAL {
		havingClauses = append(havingClauses, "pbl_student_count > 0")
	}

	if len(havingClauses) > 0 {
		return baseQuery + " HAVING " + strings.Join(havingClauses, " AND ") + " ORDER BY t.name"
	}

	return baseQuery + " ORDER BY t.name"
}

// GetWindowsPendingSubmissions returns active or closed windows with pending submissions.
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

	windowIDParam := strings.TrimSpace(r.URL.Query().Get("window_id"))
	hasWindowFilter := false
	filteredWindowID := 0
	if windowIDParam != "" {
		parsedWindowID, parseErr := strconv.Atoi(windowIDParam)
		if parseErr != nil || parsedWindowID <= 0 {
			http.Error(w, "Invalid window_id", http.StatusBadRequest)
			return
		}
		hasWindowFilter = true
		filteredWindowID = parsedWindowID
	}

	departmentIDParam := strings.TrimSpace(r.URL.Query().Get("department_id"))
	filterDepartmentID := 0
	if departmentIDParam != "" {
		parsedDepartmentID, parseErr := strconv.Atoi(departmentIDParam)
		if parseErr != nil || parsedDepartmentID <= 0 {
			http.Error(w, "Invalid department_id", http.StatusBadRequest)
			return
		}
		filterDepartmentID = parsedDepartmentID
	}

	// Fetch all enabled windows — we'll filter by time in Go to avoid timezone mismatches.
	// Build WHERE first, then append ORDER BY so optional filters don't break SQL syntax.
	windowQuery := `
		SELECT
			w.id, w.teacher_id, w.user_id, w.department_id, w.semester, w.course_id,
			w.start_at, w.end_at, w.enabled,
			COALESCE(d.department_name, '') as department_name,
			COALESCE(c.course_code, '') as course_code,
			COALESCE(c.course_name, '') as course_name,
			COALESCE(u.username, '') as user_name,
			COALESCE(w.window_name, '') as window_name
		FROM mark_entry_windows w
		LEFT JOIN departments d ON w.department_id = d.id
		LEFT JOIN courses c ON w.course_id = c.id
		LEFT JOIN users u ON u.id = w.user_id
		WHERE w.enabled = 1
	`

	var rows *sql.Rows
	var err error
	if hasWindowFilter {
		windowQuery += " AND w.id = ?"
	}
	windowQuery += " ORDER BY w.end_at DESC"

	if hasWindowFilter {
		rows, err = database.Query(windowQuery, filteredWindowID)
	} else {
		rows, err = database.Query(windowQuery)
	}
	if err != nil {
		log.Printf("Error fetching windows: %v", err)
		http.Error(w, "Failed to fetch windows", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	now := time.Now()
	log.Printf("[PendingSubmissions] status=%s, now(local)=%v, now(utc)=%v", status, now, now.UTC())

	type PendingTeacher struct {
		TeacherID     string   `json:"teacher_id"`
		TeacherName   string   `json:"teacher_name"`
		CourseID      int      `json:"course_id"`
		CourseCode    string   `json:"course_code"`
		CourseName    string   `json:"course_name"`
		Submitted     bool     `json:"submitted"`
		SubmittedAt   string   `json:"submitted_at,omitempty"`
		LearningModes []string `json:"learning_modes"` // e.g., ["PBL"], ["UAL"], or ["PBL", "UAL"]
	}

	type WindowWithPending struct {
		WindowID       int              `json:"window_id"`
		WindowName     string           `json:"window_name"`
		UserID         *int             `json:"user_id,omitempty"`
		UserName       string           `json:"user_name,omitempty"`
		DepartmentID   *int             `json:"department_id"`
		DepartmentName string           `json:"department_name"`
		DepartmentIDs  []int            `json:"department_ids,omitempty"`
		DepartmentList []string         `json:"department_names,omitempty"`
		Semester       *int             `json:"semester"`
		CourseID       *int             `json:"course_id"`
		CourseCode     string           `json:"course_code"`
		CourseName     string           `json:"course_name"`
		StartAt        string           `json:"start_at"`
		EndAt          string           `json:"end_at"`
		HasPBL         bool             `json:"has_pbl"` // Window has at least one PBL component
		HasUAL         bool             `json:"has_ual"` // Window has at least one UAL component
		Pending        []PendingTeacher `json:"pending_teachers"`
		Completed      []PendingTeacher `json:"completed_teachers"`
	}

	var result []WindowWithPending

	for rows.Next() {
		var windowID int
		var teacherID sql.NullString
		var userID sql.NullInt64
		var deptID, semester, courseID sql.NullInt64
		var startAt, endAt time.Time
		var enabled int
		var deptName, courseCode, courseName, userName, windowName sql.NullString

		err := rows.Scan(
			&windowID,
			&teacherID,
			&userID,
			&deptID,
			&semester,
			&courseID,
			&startAt,
			&endAt,
			&enabled,
			&deptName,
			&courseCode,
			&courseName,
			&userName,
			&windowName,
		)
		if err != nil {
			log.Printf("Error scanning window row: %v", err)
			continue
		}

		// Filter by time in Go — avoids MySQL timezone mismatches.
		// If a specific window is requested, skip time filtering and return exact window details.
		if !hasWindowFilter {
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
		}

		windowData := WindowWithPending{
			WindowID:       windowID,
			WindowName:     windowName.String,
			UserName:       strings.TrimSpace(userName.String),
			DepartmentName: deptName.String,
			CourseCode:     courseCode.String,
			CourseName:     courseName.String,
			StartAt:        startAt.Format(time.RFC3339),
			EndAt:          endAt.Format(time.RFC3339),
			HasPBL:         false,
			HasUAL:         false,
			Pending:        []PendingTeacher{},
			Completed:      []PendingTeacher{},
		}

		// Convert nullable fields to pointers for the response
		if deptID.Valid {
			v := int(deptID.Int64)
			windowData.DepartmentID = &v
		}
		if userID.Valid && userID.Int64 > 0 {
			v := int(userID.Int64)
			windowData.UserID = &v
		}
		windowData.DepartmentIDs, windowData.DepartmentList = getWindowDepartments(database, windowID, windowData.DepartmentID)
		if filterDepartmentID > 0 {
			match := false
			for _, dept := range windowData.DepartmentIDs {
				if dept == filterDepartmentID {
					match = true
					break
				}
			}
			if len(windowData.DepartmentIDs) > 0 && !match {
				continue
			}
			if len(windowData.DepartmentIDs) > 0 {
				windowData.DepartmentIDs = []int{filterDepartmentID}
				windowData.DepartmentList = []string{}
				if err := database.QueryRow(`SELECT COALESCE(department_name, '') FROM departments WHERE id = ?`, filterDepartmentID).Scan(&windowData.DepartmentName); err == nil {
					windowData.DepartmentList = []string{windowData.DepartmentName}
				}
			}
		}
		if len(windowData.DepartmentList) > 0 {
			windowData.DepartmentName = strings.Join(windowData.DepartmentList, ", ")
		}
		if semester.Valid {
			v := int(semester.Int64)
			windowData.Semester = &v
		}
		if courseID.Valid {
			v := int(courseID.Int64)
			windowData.CourseID = &v
		}

		// Check if window has PBL/UAL components
		componentLearningModes, err := database.Query(`
			SELECT DISTINCT mct.learning_mode_id
			FROM mark_entry_window_components wc
			INNER JOIN mark_category_types mct ON mct.id = wc.assessment_component_id
			WHERE wc.window_id = ? AND mct.learning_mode_id IS NOT NULL
		`, windowID)
		if err == nil {
			for componentLearningModes.Next() {
				var lmID int
				if err := componentLearningModes.Scan(&lmID); err == nil {
					log.Printf("[PendingSubmissions] Window %d found learning_mode_id=%d", windowID, lmID)
					if lmID == 1 {
						windowData.HasUAL = true
					} else if lmID == 2 {
						windowData.HasPBL = true
					}
				}
			}
			componentLearningModes.Close()
		} else {
			log.Printf("[PendingSubmissions] Error querying learning modes for window %d: %v", windowID, err)
		}

		log.Printf("[PendingSubmissions] Window %d: HasPBL=%v, HasUAL=%v", windowID, windowData.HasPBL, windowData.HasUAL)

		log.Printf("[PendingSubmissions] Processing window %d: teacher=%v dept=%v sem=%v course=%v",
			windowID, teacherID, deptID, semester, courseID)

		// Build query dynamically based on which scope fields have meaningful values
		var teacherArgs []interface{}

		// Determine which scope fields are meaningful (non-nil and non-zero where applicable)
		hasTeacher := teacherID.Valid && teacherID.String != ""
		hasCourse := courseID.Valid && courseID.Int64 != 0
		hasDepartment := len(windowData.DepartmentIDs) > 0
		hasSemester := semester.Valid && semester.Int64 != 0
		isUserScoped := windowData.UserID != nil && *windowData.UserID > 0

		if isUserScoped {
			// User-scoped windows are handled by dedicated user details API.
			if hasWindowFilter {
				result = append(result, windowData)
			}
			continue
		}

		if !hasTeacher && !hasCourse && !hasDepartment && !hasSemester {
			log.Printf("Skipping window %d: no meaningful scope fields", windowID)
			continue
		}

		whereClauses := []string{
			"csta.status = 1",
			"s.status = 1",
		}

		if hasTeacher {
			whereClauses = append(whereClauses, "csta.teacher_id = ?")
			teacherArgs = append(teacherArgs, teacherID.String)
		}

		if hasCourse {
			whereClauses = append(whereClauses, "csta.course_id = ?")
			teacherArgs = append(teacherArgs, courseID.Int64)
		}

		if hasDepartment {
			placeholders := make([]string, 0, len(windowData.DepartmentIDs))
			for _, selectedDepartmentID := range windowData.DepartmentIDs {
				placeholders = append(placeholders, "?")
				teacherArgs = append(teacherArgs, selectedDepartmentID)
			}
			whereClauses = append(whereClauses, fmt.Sprintf("s.department_id IN (%s)", strings.Join(placeholders, ",")))
		}

		if hasSemester {
			whereClauses = append(whereClauses, "nc.semester_number = ?")
			teacherArgs = append(teacherArgs, semester.Int64)
			whereClauses = append(whereClauses, "COALESCE(nc.card_type, 'semester') = 'semester'")
		}

		// Filter teachers strictly by selected window component learning modes.
		// PBL-only window => only teachers having at least one PBL student in scope.
		// UAL-only window => only teachers having at least one UAL student in scope.
		if windowData.HasPBL && !windowData.HasUAL {
			whereClauses = append(whereClauses, "s.learning_mode_id = 2")
		} else if windowData.HasUAL && !windowData.HasPBL {
			whereClauses = append(whereClauses, "s.learning_mode_id = 1")
		}

		teacherQuery := fmt.Sprintf(`
			SELECT DISTINCT
				csta.teacher_id,
				COALESCE(t.name, csta.teacher_id) as teacher_name,
				csta.course_id,
				COALESCE(c.course_code, '') as course_code,
				COALESCE(c.course_name, '') as course_name
			FROM course_student_teacher_allocation csta
			INNER JOIN students s ON s.id = csta.student_id
			LEFT JOIN teachers t ON t.faculty_id = csta.teacher_id
			LEFT JOIN courses c ON c.id = csta.course_id
			LEFT JOIN curriculum_courses cc ON cc.course_id = csta.course_id
			LEFT JOIN normal_cards nc ON nc.id = cc.semester_id
			WHERE %s
		`, strings.Join(whereClauses, " AND "))

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

		if len(allTeachers) == 0 && !hasWindowFilter {
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

			// Determine learning modes for this teacher+course
			learningModes := getTeacherLearningModes(database, tid, cid)

			windowData.Completed = append(windowData.Completed, PendingTeacher{
				TeacherID:     tid,
				TeacherName:   tname,
				CourseID:      cid,
				CourseCode:    ccode,
				CourseName:    cname,
				Submitted:     true,
				SubmittedAt:   submittedAt.Format(time.RFC3339),
				LearningModes: learningModes,
			})
		}
		submissionRows.Close()

		log.Printf("Window %d: %d teachers in scope, %d submissions found", windowID, len(allTeachers), len(submittedSet))

		// Pending = teachers in scope who are NOT in mark_submissions
		for _, tc := range allTeachers {
			key := fmt.Sprintf("%s|%d", tc.TeacherID, tc.CourseID)
			if !submittedSet[key] {
				// Determine learning modes for this teacher+course
				learningModes := getTeacherLearningModes(database, tc.TeacherID, tc.CourseID)

				windowData.Pending = append(windowData.Pending, PendingTeacher{
					TeacherID:     tc.TeacherID,
					TeacherName:   tc.TeacherName,
					CourseID:      tc.CourseID,
					CourseCode:    tc.CourseCode,
					CourseName:    tc.CourseName,
					Submitted:     false,
					LearningModes: learningModes,
				})
			}
		}

		// Include windows that have any teachers at all (pending or completed)
		if hasWindowFilter || len(windowData.Pending) > 0 || len(windowData.Completed) > 0 {
			result = append(result, windowData)
		}
	}

	if result == nil {
		result = []WindowWithPending{}
	}
	json.NewEncoder(w).Encode(result)
}

func getUserIdentifierAliases(database *sql.DB, numericUserID int, identifier string) []string {
	aliases := map[string]bool{}
	identifier = strings.TrimSpace(identifier)
	if identifier != "" {
		aliases[identifier] = true
	}
	if numericUserID > 0 {
		aliases[strconv.Itoa(numericUserID)] = true
	}

	var username, email sql.NullString
	if numericUserID > 0 {
		_ = database.QueryRow(`SELECT username, email FROM users WHERE id = ? LIMIT 1`, numericUserID).Scan(&username, &email)
	} else if identifier != "" {
		_ = database.QueryRow(`SELECT username, email FROM users WHERE username = ? LIMIT 1`, identifier).Scan(&username, &email)
	}

	if username.Valid && strings.TrimSpace(username.String) != "" {
		aliases[strings.TrimSpace(username.String)] = true
	}
	if email.Valid && strings.TrimSpace(email.String) != "" {
		emailValue := strings.TrimSpace(email.String)
		aliases[emailValue] = true

		var facultyID sql.NullString
		_ = database.QueryRow(`SELECT faculty_id FROM teachers WHERE email = ? LIMIT 1`, emailValue).Scan(&facultyID)
		if facultyID.Valid && strings.TrimSpace(facultyID.String) != "" {
			aliases[strings.TrimSpace(facultyID.String)] = true
		}
	}

	result := make([]string, 0, len(aliases))
	for alias := range aliases {
		result = append(result, alias)
	}
	sort.Strings(result)
	return result
}

func getWindowIDFromPath(path string, segment string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if part == segment && index+1 < len(parts) {
			value, err := strconv.Atoi(parts[index+1])
			if err != nil || value <= 0 {
				return 0, fmt.Errorf("invalid id")
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("id not found")
}

// GetWindowUserSubmissionsSummary returns submitted/not-submitted user-course entries for a user-scoped window.
func GetWindowUserSubmissionsSummary(w http.ResponseWriter, r *http.Request) {
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

	windowID, err := getWindowIDFromPath(r.URL.Path, "mark-entry-windows")
	if err != nil {
		http.Error(w, "Invalid window id", http.StatusBadRequest)
		return
	}

	departmentIDParam := strings.TrimSpace(r.URL.Query().Get("department_id"))
	filterDepartmentID := 0
	if departmentIDParam != "" {
		parsedDepartmentID, parseErr := strconv.Atoi(departmentIDParam)
		if parseErr != nil || parsedDepartmentID <= 0 {
			http.Error(w, "Invalid department_id", http.StatusBadRequest)
			return
		}
		filterDepartmentID = parsedDepartmentID
	}

	type userWindowSummaryRow struct {
		WindowID     int
		WindowName   sql.NullString
		UserID       sql.NullInt64
		DepartmentID sql.NullInt64
		Semester     sql.NullInt64
		CourseID     sql.NullInt64
		StartAt      time.Time
		EndAt        time.Time
		Enabled      bool
	}

	var summary userWindowSummaryRow
	err = database.QueryRow(`
		SELECT id, COALESCE(window_name, ''), user_id, department_id, semester, course_id, start_at, end_at, enabled
		FROM mark_entry_windows
		WHERE id = ?
	`, windowID).Scan(
		&summary.WindowID,
		&summary.WindowName,
		&summary.UserID,
		&summary.DepartmentID,
		&summary.Semester,
		&summary.CourseID,
		&summary.StartAt,
		&summary.EndAt,
		&summary.Enabled,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Window not found", http.StatusNotFound)
			return
		}
		log.Printf("Error fetching user window summary: %v", err)
		http.Error(w, "Failed to fetch window", http.StatusInternalServerError)
		return
	}

	if !summary.UserID.Valid || summary.UserID.Int64 <= 0 {
		http.Error(w, "Window is not user-scoped", http.StatusBadRequest)
		return
	}

	numericUserID := int(summary.UserID.Int64)
	var username sql.NullString
	_ = database.QueryRow(`SELECT username FROM users WHERE id = ? LIMIT 1`, numericUserID).Scan(&username)

	departmentIDs, departmentNames := getWindowDepartments(database, windowID, nil)
	if len(departmentIDs) == 0 {
		rows, depErr := database.Query(`SELECT id, department_name FROM departments WHERE status = 1 ORDER BY department_name`)
		if depErr == nil {
			defer rows.Close()
			for rows.Next() {
				var depID int
				var depName sql.NullString
				if scanErr := rows.Scan(&depID, &depName); scanErr == nil {
					departmentIDs = append(departmentIDs, depID)
					departmentNames = append(departmentNames, strings.TrimSpace(depName.String))
				}
			}
		}
	}

	selectedDepartmentIDs := departmentIDs
	selectedDepartmentNames := departmentNames
	if filterDepartmentID > 0 {
		match := false
		for _, depID := range departmentIDs {
			if depID == filterDepartmentID {
				match = true
				break
			}
		}
		if len(departmentIDs) > 0 && !match {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"window_id":          windowID,
				"window_name":        summary.WindowName.String,
				"user_id":            numericUserID,
				"user_name":          strings.TrimSpace(username.String),
				"department_options": []map[string]interface{}{},
				"pending_users":      []map[string]interface{}{},
				"completed_users":    []map[string]interface{}{},
			})
			return
		}
		selectedDepartmentIDs = []int{filterDepartmentID}
		selectedDepartmentNames = []string{}
		var depName sql.NullString
		if err := database.QueryRow(`SELECT COALESCE(department_name, '') FROM departments WHERE id = ?`, filterDepartmentID).Scan(&depName); err == nil {
			selectedDepartmentNames = []string{strings.TrimSpace(depName.String)}
		}
	}

	aliases := getUserIdentifierAliases(database, numericUserID, strings.TrimSpace(username.String))
	if len(aliases) == 0 {
		aliases = []string{strings.TrimSpace(username.String), strconv.Itoa(numericUserID)}
	}

	allowedComponentIDs := make([]int, 0)
	allowedRows, allowedErr := database.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if allowedErr == nil {
		for allowedRows.Next() {
			var componentID int
			if scanErr := allowedRows.Scan(&componentID); scanErr == nil {
				allowedComponentIDs = append(allowedComponentIDs, componentID)
			}
		}
		allowedRows.Close()
	}

	courseArgs := []interface{}{windowID, numericUserID}
	whereClauses := []string{"mesp.window_id = ?", "mesp.user_id = ?", "s.status = 1"}
	if summary.CourseID.Valid && summary.CourseID.Int64 > 0 {
		whereClauses = append(whereClauses, "sc.course_id = ?")
		courseArgs = append(courseArgs, summary.CourseID.Int64)
	}
	if summary.Semester.Valid && summary.Semester.Int64 > 0 {
		whereClauses = append(whereClauses, "ac.current_semester = ?")
		courseArgs = append(courseArgs, summary.Semester.Int64)
	}
	if len(selectedDepartmentIDs) > 0 {
		deptPlaceholders := make([]string, 0, len(selectedDepartmentIDs))
		for _, depID := range selectedDepartmentIDs {
			deptPlaceholders = append(deptPlaceholders, "?")
			courseArgs = append(courseArgs, depID)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("s.department_id IN (%s)", strings.Join(deptPlaceholders, ",")))
	}

	courseQuery := fmt.Sprintf(`
		SELECT
			sc.course_id,
			COALESCE(c.course_code, ''),
			COALESCE(c.course_name, ''),
			COUNT(DISTINCT s.id) AS assigned_students,
			GROUP_CONCAT(DISTINCT s.learning_mode_id ORDER BY s.learning_mode_id) AS learning_modes
		FROM mark_entry_student_permissions mesp
		INNER JOIN students s ON s.id = mesp.student_id
		LEFT JOIN academic_calendar ac ON ac.id = s.year
		INNER JOIN student_courses sc ON sc.student_id = s.id
		LEFT JOIN courses c ON c.id = sc.course_id
		WHERE %s
		GROUP BY sc.course_id, c.course_code, c.course_name
		ORDER BY c.course_code
	`, strings.Join(whereClauses, " AND "))

	courseRows, err := database.Query(courseQuery, courseArgs...)
	if err != nil {
		log.Printf("Error fetching user submission courses: %v", err)
		http.Error(w, "Failed to fetch user submission summary", http.StatusInternalServerError)
		return
	}
	defer courseRows.Close()

	pendingUsers := make([]map[string]interface{}, 0)
	completedUsers := make([]map[string]interface{}, 0)

	for courseRows.Next() {
		var courseID int
		var courseCode, courseName string
		var assignedStudents int
		var learningModesRaw sql.NullString
		if scanErr := courseRows.Scan(&courseID, &courseCode, &courseName, &assignedStudents, &learningModesRaw); scanErr != nil {
			continue
		}

		updatedArgs := []interface{}{windowID, numericUserID, courseID}
		updatedWhere := []string{"mesp.window_id = ?", "mesp.user_id = ?", "sc.course_id = ?", "s.status = 1"}
		if summary.Semester.Valid && summary.Semester.Int64 > 0 {
			updatedWhere = append(updatedWhere, "ac.current_semester = ?")
			updatedArgs = append(updatedArgs, summary.Semester.Int64)
		}
		if len(selectedDepartmentIDs) > 0 {
			deptPlaceholders := make([]string, 0, len(selectedDepartmentIDs))
			for _, depID := range selectedDepartmentIDs {
				deptPlaceholders = append(deptPlaceholders, "?")
				updatedArgs = append(updatedArgs, depID)
			}
			updatedWhere = append(updatedWhere, fmt.Sprintf("s.department_id IN (%s)", strings.Join(deptPlaceholders, ",")))
		}

		aliasPlaceholders := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			aliasPlaceholders = append(aliasPlaceholders, "?")
			updatedArgs = append(updatedArgs, strings.ToLower(strings.TrimSpace(alias)))
		}
		updatedWhere = append(updatedWhere, fmt.Sprintf("LOWER(TRIM(COALESCE(sm.faculty_id, ''))) IN (%s)", strings.Join(aliasPlaceholders, ",")))

		if len(allowedComponentIDs) > 0 {
			componentPlaceholders := make([]string, 0, len(allowedComponentIDs))
			for _, componentID := range allowedComponentIDs {
				componentPlaceholders = append(componentPlaceholders, "?")
				updatedArgs = append(updatedArgs, componentID)
			}
			updatedWhere = append(updatedWhere, fmt.Sprintf("sm.assessment_component_id IN (%s)", strings.Join(componentPlaceholders, ",")))
		}

		updatedQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT s.id)
			FROM mark_entry_student_permissions mesp
			INNER JOIN students s ON s.id = mesp.student_id
			LEFT JOIN academic_calendar ac ON ac.id = s.year
			INNER JOIN student_courses sc ON sc.student_id = s.id
			INNER JOIN student_marks sm ON sm.student_id = s.id AND sm.course_id = sc.course_id AND sm.status = 1
			WHERE %s
		`, strings.Join(updatedWhere, " AND "))

		updatedStudents := 0
		if err := database.QueryRow(updatedQuery, updatedArgs...).Scan(&updatedStudents); err != nil {
			updatedStudents = 0
		}

		submittedArgs := make([]interface{}, 0, len(aliases)+2)
		submittedArgs = append(submittedArgs, windowID, courseID)
		submittedPlaceholders := make([]string, 0, len(aliases))
		for _, alias := range aliases {
			submittedPlaceholders = append(submittedPlaceholders, "?")
			submittedArgs = append(submittedArgs, strings.TrimSpace(alias))
		}

		submittedQuery := fmt.Sprintf(`
			SELECT COUNT(*) > 0
			FROM mark_submissions
			WHERE window_id = ? AND course_id = ? AND teacher_id IN (%s)
		`, strings.Join(submittedPlaceholders, ","))

		isSubmitted := false
		_ = database.QueryRow(submittedQuery, submittedArgs...).Scan(&isSubmitted)

		learningModes := make([]string, 0)
		if learningModesRaw.Valid {
			for _, value := range strings.Split(learningModesRaw.String, ",") {
				trimmed := strings.TrimSpace(value)
				if trimmed == "1" {
					learningModes = append(learningModes, "UAL")
				} else if trimmed == "2" {
					learningModes = append(learningModes, "PBL")
				}
			}
		}

		item := map[string]interface{}{
			"user_id":           numericUserID,
			"user_name":         strings.TrimSpace(username.String),
			"course_id":         courseID,
			"course_code":       courseCode,
			"course_name":       courseName,
			"submitted":         isSubmitted,
			"learning_modes":    learningModes,
			"assigned_students": assignedStudents,
			"updated_students":  updatedStudents,
		}

		if isSubmitted {
			completedUsers = append(completedUsers, item)
		} else {
			pendingUsers = append(pendingUsers, item)
		}
	}

	departmentOptions := make([]map[string]interface{}, 0, len(departmentIDs))
	for index, depID := range departmentIDs {
		depName := ""
		if index < len(departmentNames) {
			depName = departmentNames[index]
		}
		departmentOptions = append(departmentOptions, map[string]interface{}{
			"department_id":   depID,
			"department_name": depName,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"window_id":          windowID,
		"window_name":        summary.WindowName.String,
		"user_id":            numericUserID,
		"user_name":          strings.TrimSpace(username.String),
		"department_ids":     selectedDepartmentIDs,
		"department_names":   selectedDepartmentNames,
		"department_options": departmentOptions,
		"pending_users":      pendingUsers,
		"completed_users":    completedUsers,
	})
}

// GetWindowUserEnteredStudents returns student-wise entered marks for one user+course within a user-scoped window.
func GetWindowUserEnteredStudents(w http.ResponseWriter, r *http.Request) {
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

	windowID, err := getWindowIDFromPath(r.URL.Path, "mark-entry-windows")
	if err != nil {
		http.Error(w, "Invalid window id", http.StatusBadRequest)
		return
	}

	userIdentifier := strings.TrimSpace(r.URL.Query().Get("user_id"))
	courseIDStr := strings.TrimSpace(r.URL.Query().Get("course_id"))
	if userIdentifier == "" || courseIDStr == "" {
		http.Error(w, "user_id and course_id are required", http.StatusBadRequest)
		return
	}

	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil || courseID <= 0 {
		http.Error(w, "invalid course_id", http.StatusBadRequest)
		return
	}

	departmentIDParam := strings.TrimSpace(r.URL.Query().Get("department_id"))
	filterDepartmentID := 0
	if departmentIDParam != "" {
		parsedDepartmentID, parseErr := strconv.Atoi(departmentIDParam)
		if parseErr != nil || parsedDepartmentID <= 0 {
			http.Error(w, "Invalid department_id", http.StatusBadRequest)
			return
		}
		filterDepartmentID = parsedDepartmentID
	}

	numericUserID, err := resolveNumericUserID(database, userIdentifier)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	var username sql.NullString
	_ = database.QueryRow(`SELECT username FROM users WHERE id = ? LIMIT 1`, numericUserID).Scan(&username)
	aliases := getUserIdentifierAliases(database, numericUserID, strings.TrimSpace(username.String))
	if len(aliases) == 0 {
		aliases = []string{strings.TrimSpace(username.String), strconv.Itoa(numericUserID)}
	}

	allowedByWindow := map[int]bool{}
	rows, compErr := database.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if compErr == nil {
		for rows.Next() {
			var componentID int
			if scanErr := rows.Scan(&componentID); scanErr == nil {
				allowedByWindow[componentID] = true
			}
		}
		rows.Close()
	}

	studentWhere := []string{
		"mesp.window_id = ?",
		"mesp.user_id = ?",
		"sc.course_id = ?",
		"s.status = 1",
	}
	studentArgs := []interface{}{windowID, numericUserID, courseID}

	var semester sql.NullInt64
	_ = database.QueryRow(`SELECT semester FROM mark_entry_windows WHERE id = ?`, windowID).Scan(&semester)
	if semester.Valid && semester.Int64 > 0 {
		studentWhere = append(studentWhere, "ac.current_semester = ?")
		studentArgs = append(studentArgs, semester.Int64)
	}

	if filterDepartmentID > 0 {
		studentWhere = append(studentWhere, "s.department_id = ?")
		studentArgs = append(studentArgs, filterDepartmentID)
	}

	studentQuery := fmt.Sprintf(`
		SELECT DISTINCT s.id, COALESCE(s.register_no, ''), s.student_name
		FROM mark_entry_student_permissions mesp
		INNER JOIN students s ON s.id = mesp.student_id
		LEFT JOIN academic_calendar ac ON ac.id = s.year
		INNER JOIN student_courses sc ON sc.student_id = s.id
		WHERE %s
		ORDER BY s.student_name
	`, strings.Join(studentWhere, " AND "))

	studentRows, err := database.Query(studentQuery, studentArgs...)
	if err != nil {
		log.Printf("Error fetching user entered students: %v", err)
		http.Error(w, "failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer studentRows.Close()

	placeholderParts := make([]string, 0, len(aliases))
	markArgs := make([]interface{}, 0, len(aliases)+1)
	markArgs = append(markArgs, courseID)
	for _, alias := range aliases {
		placeholderParts = append(placeholderParts, "?")
		markArgs = append(markArgs, strings.ToLower(strings.TrimSpace(alias)))
	}

	markQuery := fmt.Sprintf(`
		SELECT sm.student_id, sm.assessment_component_id,
			COALESCE(mct.name, CONCAT('Component ', sm.assessment_component_id)),
			COALESCE(sm.obtained_marks, 0)
		FROM student_marks sm
		LEFT JOIN mark_category_types mct ON mct.id = sm.assessment_component_id
		WHERE sm.course_id = ?
		  AND LOWER(TRIM(COALESCE(sm.faculty_id, ''))) IN (%s)
		  AND sm.status = 1
	`, strings.Join(placeholderParts, ","))

	markRows, markErr := database.Query(markQuery, markArgs...)
	if markErr != nil {
		log.Printf("Error fetching marks for user drill-down: %v", markErr)
		http.Error(w, "failed to fetch mark components", http.StatusInternalServerError)
		return
	}
	defer markRows.Close()

	componentMapByStudent := map[int][]map[string]interface{}{}
	totalByStudent := map[int]float64{}
	for markRows.Next() {
		var studentID int
		var componentID int
		var componentName string
		var obtained float64
		if scanErr := markRows.Scan(&studentID, &componentID, &componentName, &obtained); scanErr != nil {
			continue
		}
		if len(allowedByWindow) > 0 && !allowedByWindow[componentID] {
			continue
		}

		componentMapByStudent[studentID] = append(componentMapByStudent[studentID], map[string]interface{}{
			"assessment_component_id":   componentID,
			"assessment_component_name": componentName,
			"obtained_marks":            obtained,
			"allowed_in_window":         true,
		})
		totalByStudent[studentID] += obtained
	}

	items := make([]map[string]interface{}, 0)
	seenStudents := make(map[int]bool)
	for studentRows.Next() {
		var studentID int
		var registerID string
		var studentName string
		if scanErr := studentRows.Scan(&studentID, &registerID, &studentName); scanErr != nil {
			continue
		}
		if seenStudents[studentID] {
			continue
		}
		seenStudents[studentID] = true

		components := componentMapByStudent[studentID]
		items = append(items, map[string]interface{}{
			"student_id":     studentID,
			"register_id":    registerID,
			"student_name":   studentName,
			"total_marks":    totalByStudent[studentID],
			"has_mark_entry": len(components) > 0,
			"components":     components,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":   userIdentifier,
		"course_id": courseID,
		"students":  items,
	})
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
