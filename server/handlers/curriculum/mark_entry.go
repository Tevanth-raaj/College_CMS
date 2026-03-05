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

func mapCourseCategoryToTypeID(category string) int {
	categoryLower := strings.ToLower(strings.TrimSpace(category))
	if categoryLower == "" {
		return 0
	}
	if strings.Contains(categoryLower, "theory") && strings.Contains(categoryLower, "lab") {
		return 3
	}
	if strings.Contains(categoryLower, "lab") {
		return 2
	}
	return 1
}

func normalizeFacultyIdentifier(raw string) string {
	identifier := strings.TrimSpace(raw)
	if identifier == "" {
		return identifier
	}

	var facultyID sql.NullString

	// 1. Already a faculty_id
	err := db.DB.QueryRow(`SELECT faculty_id FROM teachers WHERE faculty_id = ? LIMIT 1`, identifier).Scan(&facultyID)
	if err == nil && facultyID.Valid && strings.TrimSpace(facultyID.String) != "" {
		return strings.TrimSpace(facultyID.String)
	}

	// 2. Numeric primary key (id column) from teachers table
	err = db.DB.QueryRow(`SELECT faculty_id FROM teachers WHERE id = ? LIMIT 1`, identifier).Scan(&facultyID)
	if err == nil && facultyID.Valid && strings.TrimSpace(facultyID.String) != "" {
		return strings.TrimSpace(facultyID.String)
	}

	// 3. Username → email → faculty_id
	var email sql.NullString
	err = db.DB.QueryRow(`SELECT email FROM users WHERE username = ? LIMIT 1`, identifier).Scan(&email)
	if err == nil && email.Valid && strings.TrimSpace(email.String) != "" {
		err = db.DB.QueryRow(`SELECT faculty_id FROM teachers WHERE email = ? LIMIT 1`, strings.TrimSpace(email.String)).Scan(&facultyID)
		if err == nil && facultyID.Valid && strings.TrimSpace(facultyID.String) != "" {
			return strings.TrimSpace(facultyID.String)
		}
	}

	return identifier
}

// GetMarkCategoriesByType fetches all mark categories for a specific course type
func GetMarkCategoriesByType(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract courseTypeId from URL path: /api/mark-categories-by-type/{courseTypeId}
	pathParts := strings.Split(r.URL.Path, "/")
	courseTypeIdStr := pathParts[len(pathParts)-1]

	courseTypeID, err := strconv.Atoi(courseTypeIdStr)
	if err != nil {
		http.Error(w, "Invalid course type ID", http.StatusBadRequest)
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

	// Default to PBL (mode 2) if no valid modes specified
	if len(learningModes) == 0 {
		learningModes = []int{2}
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Build WHERE clause for learning modes
	learningModePlaceholders := make([]string, len(learningModes))
	learningModeArgs := make([]interface{}, len(learningModes)+1)
	learningModeArgs[0] = courseTypeID
	for i, mode := range learningModes {
		learningModePlaceholders[i] = "?"
		learningModeArgs[i+1] = mode
	}
	learningModeClause := strings.Join(learningModePlaceholders, ",")

	// Query mark categories filtered by course_type_id and learning_mode_id, ordered by position
	query := fmt.Sprintf(`
		SELECT 
			mct.id,
			mct.name,
			mct.max_marks,
			mct.conversion_marks,
			mct.position,
			mct.course_type_id,
			COALESCE(ct.course_type, '') as course_type_name,
			mct.category_name_id,
			COALESCE(mcn.category_name, '') as category_name,
			mct.learning_mode_id,
			mct.status
		FROM mark_category_types mct
		LEFT JOIN course_type ct ON mct.course_type_id = ct.id
		LEFT JOIN mark_category_name mcn ON mct.category_name_id = mcn.id
		WHERE mct.course_type_id = ? AND mct.learning_mode_id IN (%s) AND mct.status = 1
		ORDER BY mct.position ASC
	`, learningModeClause)

	log.Printf("[DEBUG] Executing query with courseTypeID=%d, learningModes=%v", courseTypeID, learningModes)
	rows, err := database.Query(query, learningModeArgs...)
	if err != nil {
		log.Printf("Error fetching mark categories: %v", err)
		http.Error(w, "Error fetching mark categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.MarkCategoryType
	for rows.Next() {
		var category models.MarkCategoryType
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.MaxMarks,
			&category.ConversionMarks,
			&category.Position,
			&category.CourseTypeID,
			&category.CourseTypeName,
			&category.CategoryNameID,
			&category.CategoryName,
			&category.LearningModeID,
			&category.Status,
		)
		if err != nil {
			log.Printf("Error scanning mark category: %v", err)
			continue
		}
		log.Printf("[DEBUG] Category ID=%d, Name=%s, CourseTypeName=%s, CategoryName=%s",
			category.ID, category.Name, category.CourseTypeName, category.CategoryName)
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating mark categories: %v", err)
		http.Error(w, "Error processing mark categories", http.StatusInternalServerError)
		return
	}

	// Return empty array if no categories found
	if categories == nil {
		categories = []models.MarkCategoryType{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// GetMarkCategoriesForCourse returns mark categories enabled for a course and teacher.
func GetMarkCategoriesForCourse(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract courseId from URL path: /api/course/{courseId}/mark-categories
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	courseIdStr := pathParts[len(pathParts)-2]
	courseID, err := strconv.Atoi(courseIdStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	teacherID := r.URL.Query().Get("teacher_id")
	if teacherID == "" {
		http.Error(w, "Teacher ID is required", http.StatusBadRequest)
		return
	}

	selectedWindowID := 0
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		value, convErr := strconv.Atoi(windowIDStr)
		if convErr != nil || value <= 0 {
			http.Error(w, "Invalid window ID", http.StatusBadRequest)
			return
		}
		selectedWindowID = value
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	windowOpen, _, allowedComponents, err := resolveMarkEntryWindowWithSelection(courseID, teacherID, selectedWindowID)
	if err != nil {
		log.Printf("Error resolving mark entry window: %v", err)
		http.Error(w, "Failed to validate mark entry window", http.StatusInternalServerError)
		return
	}
	if !windowOpen {
		http.Error(w, "Mark entry window is closed", http.StatusForbidden)
		return
	}

	_ = allowedComponents // used for component filtering downstream

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
	learningModeArgs := make([]interface{}, len(learningModes)+1)
	learningModeArgs[0] = courseTypeID
	for i, mode := range learningModes {
		learningModePlaceholders[i] = "?"
		learningModeArgs[i+1] = mode
	}
	learningModeClause := strings.Join(learningModePlaceholders, ",")

	// Window component filtering now handles all access control
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
			mct.status
		FROM mark_category_types mct
		WHERE mct.course_type_id = ? AND mct.learning_mode_id IN (%s) AND mct.status = 1
		ORDER BY mct.position ASC
	`, learningModeClause)

	rows, err := database.Query(query, learningModeArgs...)
	if err != nil {
		log.Printf("Error fetching mark categories: %v", err)
		http.Error(w, "Error fetching mark categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.MarkCategoryType
	for rows.Next() {
		var category models.MarkCategoryType
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
		)
		if err != nil {
			log.Printf("Error scanning mark category: %v", err)
			continue
		}

		// Filter by window component permissions (if specified)
		if len(allowedComponents) > 0 {
			allowed := false
			for _, allowedID := range allowedComponents {
				if category.ID == allowedID {
					allowed = true
					break
				}
			}
			if !allowed {
				continue // Skip this component - not in window's allowed list
			}
		}

		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating mark categories: %v", err)
		http.Error(w, "Error processing mark categories", http.StatusInternalServerError)
		return
	}

	if categories == nil {
		categories = []models.MarkCategoryType{}
	}

	json.NewEncoder(w).Encode(categories)
}

// SaveStudentMarks saves or updates student mark entries
func SaveStudentMarks(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var saveRequest models.MarkEntrySaveRequest
	err := json.NewDecoder(r.Body).Decode(&saveRequest)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if saveRequest.CourseID == 0 || len(saveRequest.MarkEntries) == 0 {
		http.Error(w, "Course ID and mark entries are required", http.StatusBadRequest)
		return
	}

	normalizedFacultyID := normalizeFacultyIdentifier(saveRequest.FacultyID)
	if normalizedFacultyID == "" {
		http.Error(w, "Faculty ID is required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	windowOpen, windowID, allowedComponents, err := resolveMarkEntryWindowWithSelection(saveRequest.CourseID, normalizedFacultyID, saveRequest.WindowID)
	if err != nil {
		log.Printf("Error resolving mark entry window: %v", err)
		http.Error(w, "Failed to validate mark entry window", http.StatusInternalServerError)
		return
	}
	if !windowOpen || windowID == 0 {
		http.Error(w, "Mark entry window is closed", http.StatusForbidden)
		return
	}

	_, windowIDs, _, allWindowErr := resolveMarkEntryWindow(saveRequest.CourseID, normalizedFacultyID)
	if allWindowErr != nil || len(windowIDs) == 0 {
		windowIDs = []int{windowID}
	}

	// Get assigned student IDs for this user (if student-specific permissions exist)
	assignedStudents, err := getAssignedStudentIDs(normalizedFacultyID, saveRequest.CourseID)
	if err != nil {
		log.Printf("Error fetching assigned students: %v", err)
		http.Error(w, "Failed to validate student permissions", http.StatusInternalServerError)
		return
	}

	// Create a map for quick lookup of assigned students
	assignedStudentMap := make(map[int]bool)
	for _, studentID := range assignedStudents {
		assignedStudentMap[studentID] = true
	}

	// If there are assigned students, restrict to only those students
	hasStudentRestrictions := len(assignedStudents) > 0

	// Build map of window-allowed components (empty = all allowed)
	allowedByWindow := map[int]bool{}
	if len(allowedComponents) > 0 {
		for _, componentID := range allowedComponents {
			allowedByWindow[componentID] = true
		}
	}

	savedCount := 0
	var errors []string

	// Process each mark entry
	for _, entry := range saveRequest.MarkEntries {
		// Check student-specific permissions if restrictions exist
		if hasStudentRestrictions && !assignedStudentMap[entry.StudentID] {
			errors = append(errors, fmt.Sprintf("Student %d: not assigned to this user for mark entry", entry.StudentID))
			continue
		}

		// Check window component permissions
		if len(allowedByWindow) > 0 && !allowedByWindow[entry.AssessmentComponentID] {
			errors = append(errors, fmt.Sprintf("Student %d: component %d not allowed by window", entry.StudentID, entry.AssessmentComponentID))
			continue
		}

		// Check absentee status — block mark entry if student is absent for this component in ANY active window
		absentBlocked := false
		for _, wid := range windowIDs {
			log.Printf("[ABSENTEE CHECK] Checking: windowID=%d, courseID=%d, studentID=%d, componentID=%d", wid, entry.CourseID, entry.StudentID, entry.AssessmentComponentID)
			isAbsent, absentErr := IsStudentAbsentForComponent(wid, entry.CourseID, entry.StudentID, entry.AssessmentComponentID)
			if absentErr != nil {
				log.Printf("[ABSENTEE CHECK] Error checking absentee status: %v", absentErr)
			} else if isAbsent {
				log.Printf("[ABSENTEE CHECK] ✗ BLOCKED - Student %d is absent for component %d in window %d", entry.StudentID, entry.AssessmentComponentID, wid)
				errors = append(errors, fmt.Sprintf("Student %d: marked absent for component %d — mark entry is blocked", entry.StudentID, entry.AssessmentComponentID))
				absentBlocked = true
				break
			} else {
				log.Printf("[ABSENTEE CHECK] ✓ PASSED - Student %d is NOT absent for component %d in window %d", entry.StudentID, entry.AssessmentComponentID, wid)
			}
		}
		if absentBlocked {
			continue
		}

		// Validate student enrollment in course
		var studentEnrolled bool
		err := database.QueryRow(`
			SELECT COUNT(*) > 0 FROM student_courses 
			WHERE student_id = ? AND course_id = ?
		`, entry.StudentID, entry.CourseID).Scan(&studentEnrolled)
		if err != nil {
			log.Printf("Error validating student enrollment: %v", err)
			errors = append(errors, fmt.Sprintf("Student %d: enrollment validation failed", entry.StudentID))
			continue
		}

		if !studentEnrolled {
			log.Printf("Student %d not enrolled in course %d", entry.StudentID, entry.CourseID)
			errors = append(errors, fmt.Sprintf("Student %d is not enrolled in this course", entry.StudentID))
			continue
		}

		// Get mark category details for conversion calculation
		var maxMarks float64
		var conversionMarks float64
		err = database.QueryRow(`
			SELECT max_marks, conversion_marks FROM mark_category_types 
			WHERE id = ?
		`, entry.AssessmentComponentID).Scan(&maxMarks, &conversionMarks)
		if err != nil {
			log.Printf("Error fetching mark category: %v", err)
			errors = append(errors, fmt.Sprintf("Mark category %d not found", entry.AssessmentComponentID))
			continue
		}

		// Validate obtained marks against max marks
		if entry.ObtainedMarks < 0 || entry.ObtainedMarks > maxMarks {
			errors = append(errors, fmt.Sprintf("Student %d: marks %.2f exceed maximum %.0f",
				entry.StudentID, entry.ObtainedMarks, maxMarks))
			continue
		}

		// Calculate converted marks: (obtained_marks / max_marks) * conversion_marks
		var convertedMarks float64
		if maxMarks > 0 {
			convertedMarks = (entry.ObtainedMarks / maxMarks) * conversionMarks
		}

		// Upsert mark entry
		query := `
			INSERT INTO student_marks 
			(student_id, course_id, faculty_id, assessment_component_id, obtained_marks, converted_marks, status)
			VALUES (?, ?, ?, ?, ?, ?, 1)
			ON DUPLICATE KEY UPDATE 
			obtained_marks = VALUES(obtained_marks),
			converted_marks = VALUES(converted_marks),
			status = 1
		`

		_, err = database.Exec(query,
			entry.StudentID,
			entry.CourseID,
			normalizedFacultyID,
			entry.AssessmentComponentID,
			entry.ObtainedMarks,
			convertedMarks,
		)
		if err != nil {
			log.Printf("Error saving student mark: %v", err)
			errors = append(errors, fmt.Sprintf("Student %d: database error", entry.StudentID))
			continue
		}

		savedCount++
	}

	response := models.MarkEntrySaveResponse{
		Success:    len(errors) == 0,
		SavedCount: savedCount,
	}

	if len(errors) > 0 {
		response.Message = fmt.Sprintf("Saved %d/%d marks. Errors: %s",
			savedCount, len(saveRequest.MarkEntries), strings.Join(errors, "; "))
	} else {
		response.Message = fmt.Sprintf("Successfully saved %d mark entries", savedCount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStudentMarks retrieves existing marks for a course
func GetStudentMarks(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract courseId from URL path: /api/course/{courseId}/student-marks
	pathParts := strings.Split(r.URL.Path, "/")
	courseIdStr := pathParts[len(pathParts)-2]

	courseID, err := strconv.Atoi(courseIdStr)
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	teacherID := r.URL.Query().Get("teacher_id")
	if teacherID == "" {
		http.Error(w, "Teacher ID is required", http.StatusBadRequest)
		return
	}

	selectedWindowID := 0
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		value, convErr := strconv.Atoi(windowIDStr)
		if convErr != nil || value <= 0 {
			http.Error(w, "Invalid window ID", http.StatusBadRequest)
			return
		}
		selectedWindowID = value
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	windowOpen, _, _, err := resolveMarkEntryWindowWithSelection(courseID, teacherID, selectedWindowID)
	if err != nil {
		log.Printf("Error resolving mark entry window: %v", err)
		http.Error(w, "Failed to validate mark entry window", http.StatusInternalServerError)
		return
	}
	if !windowOpen {
		http.Error(w, "Mark entry window is closed", http.StatusForbidden)
		return
	}

	// Get assigned student IDs for this user (if student-specific permissions exist)
	assignedStudents, err := getAssignedStudentIDs(teacherID, courseID)
	if err != nil {
		log.Printf("Error fetching assigned students: %v", err)
		http.Error(w, "Failed to validate student permissions", http.StatusInternalServerError)
		return
	}

	// Query marks - filter by assigned students if restrictions exist
	var query string
	var args []interface{}

	if len(assignedStudents) > 0 {
		// Build IN clause for assigned students
		placeholders := make([]string, len(assignedStudents))
		args = make([]interface{}, len(assignedStudents)+1)
		args[0] = courseID
		for i, studentID := range assignedStudents {
			placeholders[i] = "?"
			args[i+1] = studentID
		}

		query = fmt.Sprintf(`
			SELECT 
				id, student_id, course_id, faculty_id, assessment_component_id,
				obtained_marks, converted_marks, status
			FROM student_marks
			WHERE course_id = ? AND student_id IN (%s) AND status = 1
			ORDER BY student_id, assessment_component_id
		`, strings.Join(placeholders, ","))
	} else {
		// No student restrictions - show all marks for the course
		query = `
			SELECT 
				id, student_id, course_id, faculty_id, assessment_component_id,
				obtained_marks, converted_marks, status
			FROM student_marks
			WHERE course_id = ? AND status = 1
			ORDER BY student_id, assessment_component_id
		`
		args = []interface{}{courseID}
	}

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching student marks: %v", err)
		http.Error(w, "Error fetching student marks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var marks []models.StudentMark
	for rows.Next() {
		var mark models.StudentMark
		err := rows.Scan(
			&mark.ID,
			&mark.StudentID,
			&mark.CourseID,
			&mark.FacultyID,
			&mark.AssessmentComponentID,
			&mark.ObtainedMarks,
			&mark.ConvertedMarks,
			&mark.Status,
		)
		if err != nil {
			log.Printf("Error scanning student mark: %v", err)
			continue
		}
		marks = append(marks, mark)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating student marks: %v", err)
		http.Error(w, "Error processing student marks", http.StatusInternalServerError)
		return
	}

	// Return empty array if no marks found
	if marks == nil {
		marks = []models.StudentMark{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marks)
}

// SubmitMarks records a mark submission for a teacher+course into the mark_submissions table.
// It inserts one row per active window that applies to the teacher+course.
func SubmitMarks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		TeacherID string `json:"teacher_id"`
		CourseID  int    `json:"course_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.TeacherID) == "" || req.CourseID == 0 {
		http.Error(w, "teacher_id and course_id are required", http.StatusBadRequest)
		return
	}

	// Resolve to faculty_id so mark_submissions is consistent with teacher_course_allocation
	req.TeacherID = normalizeFacultyIdentifier(req.TeacherID)
	if strings.TrimSpace(req.TeacherID) == "" {
		http.Error(w, "Could not resolve faculty_id for given teacher_id", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Resolve active windows for this teacher+course
	windowOpen, windowIDs, _, err := resolveMarkEntryWindow(req.CourseID, req.TeacherID)
	if err != nil {
		log.Printf("[SubmitMarks] Error resolving window: %v", err)
		http.Error(w, "Failed to resolve mark entry window", http.StatusInternalServerError)
		return
	}
	if !windowOpen || len(windowIDs) == 0 {
		http.Error(w, "No active mark entry window for this course", http.StatusForbidden)
		return
	}

	// Insert a submission record for each active window (ignore duplicates)
	insertedCount := 0
	for _, wid := range windowIDs {
		_, err := database.Exec(`
			INSERT INTO mark_submissions (window_id, teacher_id, course_id)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE submitted_at = CURRENT_TIMESTAMP
		`, wid, req.TeacherID, req.CourseID)
		if err != nil {
			log.Printf("[SubmitMarks] Error inserting submission for window %d: %v", wid, err)
		} else {
			insertedCount++
		}
	}

	log.Printf("[SubmitMarks] Recorded %d submission(s) for teacher=%s course=%d", insertedCount, req.TeacherID, req.CourseID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Marks submitted for %d window(s)", insertedCount),
	})
}

// CheckMarkSubmission checks whether a teacher has already submitted marks for a course
// in any active window. Returns { submitted: true/false }.
func CheckMarkSubmission(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid course_id", http.StatusBadRequest)
		return
	}

	// Resolve to faculty_id for consistent lookup in mark_submissions
	teacherID = normalizeFacultyIdentifier(teacherID)

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// If a specific window_id is supplied, bypass active-window resolution
	// and check directly against that window (supports expired windows)
	specificWindowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id"))
	if specificWindowIDStr != "" {
		specificWindowID, err := strconv.Atoi(specificWindowIDStr)
		if err == nil && specificWindowID > 0 {
			var exists bool
			var submittedAt sql.NullTime
			err = database.QueryRow(`
				SELECT COUNT(*) > 0, MAX(submitted_at)
				FROM mark_submissions
				WHERE window_id = ? AND teacher_id = ? AND course_id = ?
			`, specificWindowID, teacherID, courseID).Scan(&exists, &submittedAt)
			if err != nil {
				log.Printf("[CheckMarkSubmission] Error: %v", err)
				http.Error(w, "Failed to check submission status", http.StatusInternalServerError)
				return
			}
			resp := map[string]interface{}{"submitted": exists}
			if exists && submittedAt.Valid {
				resp["submitted_at"] = submittedAt.Time.Format(time.RFC3339)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	// Resolve active windows for this teacher+course
	windowOpen, windowIDs, _, err := resolveMarkEntryWindow(courseID, teacherID)
	if err != nil {
		log.Printf("[CheckMarkSubmission] Error resolving window: %v", err)
		http.Error(w, "Failed to resolve mark entry window", http.StatusInternalServerError)
		return
	}
	if !windowOpen || len(windowIDs) == 0 {
		// No active window — not submitted
		json.NewEncoder(w).Encode(map[string]bool{"submitted": false})
		return
	}

	// Check if there's a submission record for ANY of the active windows
	placeholders := make([]string, len(windowIDs))
	args := []interface{}{}
	for i, wid := range windowIDs {
		placeholders[i] = "?"
		args = append(args, wid)
	}
	args = append(args, teacherID, courseID)

	var exists bool
	query := fmt.Sprintf(`
		SELECT COUNT(*) > 0
		FROM mark_submissions
		WHERE window_id IN (%s) AND teacher_id = ? AND course_id = ?
	`, strings.Join(placeholders, ","))

	err = database.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		log.Printf("[CheckMarkSubmission] Error checking submission: %v", err)
		http.Error(w, "Failed to check submission status", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"submitted": exists})
}
