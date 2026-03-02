package curriculum

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"server/db"

	"github.com/chromedp/chromedp"
	"github.com/xuri/excelize/v2"
)

type markEntryAssignmentRow struct {
	CourseID     int    `json:"course_id"`
	CourseCode   string `json:"course_code"`
	CourseName   string `json:"course_name"`
	TeacherID    string `json:"teacher_id"`
	TeacherName  string `json:"teacher_name"`
	DepartmentID int    `json:"department_id"`
	Department   string `json:"department_name"`
	Semester     int    `json:"semester"`
}

type markEntryOverviewRow struct {
	CourseID            int     `json:"course_id"`
	CourseCode          string  `json:"course_code"`
	CourseName          string  `json:"course_name"`
	TeacherID           string  `json:"teacher_id"`
	TeacherName         string  `json:"teacher_name"`
	DepartmentID        int     `json:"department_id"`
	DepartmentName      string  `json:"department_name"`
	Semester            int     `json:"semester"`
	WindowID            *int    `json:"window_id,omitempty"`
	WindowStartAt       *string `json:"window_start_at,omitempty"`
	WindowEndAt         *string `json:"window_end_at,omitempty"`
	WindowStatus        string  `json:"window_status"`
	AssignedStudents    int     `json:"assigned_students"`
	CompletedStudents   int     `json:"completed_students"`
	CompletionPercent   float64 `json:"completion_percent"`
	Late                bool    `json:"late"`
	ExtensionStatus     string  `json:"extension_status"`
	AllowedComponentIDs []int   `json:"allowed_component_ids"`
	StudentIDs          []int   `json:"student_ids,omitempty"`
}

type resultAnalysisRow struct {
	CourseID         int                `json:"course_id"`
	CourseCode       string             `json:"course_code"`
	CourseName       string             `json:"course_name"`
	Registered       int                `json:"registered"`
	Appeared         int                `json:"appeared"`
	Absent           int                `json:"absent"`
	Passed           int                `json:"passed"`
	Failed           int                `json:"failed"`
	PassPercent      float64            `json:"pass_percent"`
	MaximumMark      float64            `json:"maximum_mark"`
	MinimumMark      float64            `json:"minimum_mark"`
	AverageMark      float64            `json:"average_mark"`
	Ranges           map[string]int     `json:"ranges"`
	StudentMarkItems []studentMarkTotal `json:"-"`
}

type studentMarkTotal struct {
	StudentID   int
	StudentName string
	TotalMarks  float64
	Appeared    bool
}

type extensionRequestBody struct {
	WindowID       int    `json:"window_id"`
	CourseID       int    `json:"course_id"`
	TeacherID      string `json:"teacher_id"`
	Reason         string `json:"reason"`
	RequestedEndAt string `json:"requested_end_at"`
	RequesterRole  string `json:"requester_role"`
	RequesterUser  string `json:"requester_username"`
	DepartmentID   *int   `json:"department_id,omitempty"`
	Semester       *int   `json:"semester,omitempty"`
	ExamType       string `json:"exam_type,omitempty"`
}

func getDepartmentContextFromUsername(username string) (int, string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, "", fmt.Errorf("username is required")
	}

	var email sql.NullString
	err := db.DB.QueryRow(`SELECT email FROM users WHERE username = ? AND is_active = 1`, username).Scan(&email)
	if err != nil {
		return 0, "", err
	}

	if !email.Valid || strings.TrimSpace(email.String) == "" {
		return 0, "", fmt.Errorf("user has no email")
	}

	var departmentID int
	var departmentName string
	err = db.DB.QueryRow(`
		SELECT d.id, d.department_name
		FROM teachers t
		JOIN departments d ON CAST(t.dept AS UNSIGNED) = d.id
		WHERE t.email = ?
		LIMIT 1
	`, strings.TrimSpace(email.String)).Scan(&departmentID, &departmentName)
	if err == nil {
		return departmentID, departmentName, nil
	}

	return 0, "", fmt.Errorf("department mapping not found for user")
}

func findBestWindowForAssignment(teacherID string, departmentID int, semester int, courseID int) (*int, *time.Time, *time.Time, []int, string, error) {
	query := `
		SELECT id, start_at, end_at
		FROM mark_entry_windows
		WHERE enabled = 1
			AND (teacher_id = ? OR teacher_id IS NULL)
			AND (department_id = ? OR department_id IS NULL)
			AND (semester = ? OR semester IS NULL)
			AND (course_id = ? OR course_id IS NULL)
		ORDER BY
			(teacher_id IS NOT NULL) DESC,
			(department_id IS NOT NULL) DESC,
			(semester IS NOT NULL) DESC,
			(course_id IS NOT NULL) DESC,
			updated_at DESC
		LIMIT 1
	`

	var windowID int
	var startAt, endAt time.Time
	err := db.DB.QueryRow(query, teacherID, departmentID, semester, courseID).Scan(&windowID, &startAt, &endAt)
	if err == sql.ErrNoRows {
		status := "not_configured"
		return nil, nil, nil, []int{}, status, nil
	}
	if err != nil {
		return nil, nil, nil, nil, "", err
	}

	componentRows, err := db.DB.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	defer componentRows.Close()

	componentIDs := make([]int, 0)
	for componentRows.Next() {
		var componentID int
		if scanErr := componentRows.Scan(&componentID); scanErr == nil {
			componentIDs = append(componentIDs, componentID)
		}
	}

	now := time.Now().UTC()
	status := "closed"
	if now.Before(startAt) {
		status = "upcoming"
	} else if now.After(startAt) && now.Before(endAt) {
		status = "open"
	}

	return &windowID, &startAt, &endAt, componentIDs, status, nil
}

func getCourseTypeForCourse(courseID int) (int, error) {
	var category sql.NullString
	err := db.DB.QueryRow(`SELECT category FROM courses WHERE id = ?`, courseID).Scan(&category)
	if err != nil {
		return 0, err
	}
	return mapCourseCategoryToTypeID(category.String), nil
}

func getExpectedComponents(courseTypeID int, learningModeID int, allowedComponents []int) (map[int]bool, error) {
	rows, err := db.DB.Query(`
		SELECT id
		FROM mark_category_types
		WHERE course_type_id = ? AND learning_mode_id = ? AND status = 1
	`, courseTypeID, learningModeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expected := make(map[int]bool)
	allowedFilter := make(map[int]bool)
	if len(allowedComponents) > 0 {
		for _, componentID := range allowedComponents {
			allowedFilter[componentID] = true
		}
	}

	for rows.Next() {
		var componentID int
		if scanErr := rows.Scan(&componentID); scanErr != nil {
			continue
		}
		if len(allowedFilter) > 0 && !allowedFilter[componentID] {
			continue
		}
		expected[componentID] = true
	}

	return expected, nil
}

func isStudentCompleteForAssignment(studentID int, courseID int, teacherID string, expected map[int]bool) (bool, float64, bool, error) {
	rows, err := db.DB.Query(`
		SELECT assessment_component_id, obtained_marks
		FROM student_marks
		WHERE student_id = ? AND course_id = ? AND faculty_id = ?
	`, studentID, courseID, teacherID)
	if err != nil {
		return false, 0, false, err
	}
	defer rows.Close()

	entered := make(map[int]bool)
	total := 0.0
	hasAny := false
	for rows.Next() {
		var componentID int
		var marks sql.NullFloat64
		if scanErr := rows.Scan(&componentID, &marks); scanErr != nil {
			continue
		}
		entered[componentID] = true
		if marks.Valid {
			total += marks.Float64
			hasAny = true
		}
	}

	if len(expected) == 0 {
		return hasAny, total, hasAny, nil
	}

	for componentID := range expected {
		if !entered[componentID] {
			return false, total, hasAny, nil
		}
	}

	return true, total, hasAny, nil
}

func getLatestExtensionStatus(windowID int, courseID int, teacherID string) string {
	var status sql.NullString
	err := db.DB.QueryRow(`
		SELECT status
		FROM mark_entry_extension_requests
		WHERE window_id = ? AND course_id = ? AND teacher_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, windowID, courseID, teacherID).Scan(&status)
	if err != nil || !status.Valid {
		return "none"
	}
	return status.String
}

func getDepartmentAssignments(departmentID int, semester int) ([]markEntryAssignmentRow, error) {
	rows, err := db.DB.Query(`
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			tca.teacher_id,
			COALESCE(t.name, tca.teacher_id) AS teacher_name,
			d.id,
			d.department_name,
			nc.semester_number
		FROM teacher_course_allocation tca
		JOIN courses c ON c.id = tca.course_id
		JOIN curriculum_courses cc ON cc.course_id = c.id
		JOIN normal_cards nc ON nc.id = cc.semester_id
		JOIN teachers t ON t.faculty_id = tca.teacher_id
		JOIN departments d ON CAST(t.dept AS UNSIGNED) = d.id
		WHERE d.id = ?
			AND nc.semester_number = ?
		ORDER BY c.course_code, teacher_name
	`, departmentID, semester)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]markEntryAssignmentRow, 0)
	for rows.Next() {
		var item markEntryAssignmentRow
		if scanErr := rows.Scan(
			&item.CourseID,
			&item.CourseCode,
			&item.CourseName,
			&item.TeacherID,
			&item.TeacherName,
			&item.DepartmentID,
			&item.Department,
			&item.Semester,
		); scanErr != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func buildMarkEntryOverview(departmentID int, semester int) ([]markEntryOverviewRow, error) {
	assignments, err := getDepartmentAssignments(departmentID, semester)
	if err != nil {
		return nil, err
	}

	rows := make([]markEntryOverviewRow, 0, len(assignments))
	now := time.Now().UTC()

	for _, assignment := range assignments {
		courseTypeID, typeErr := getCourseTypeForCourse(assignment.CourseID)
		if typeErr != nil {
			continue
		}

		windowID, windowStart, windowEnd, allowedComponents, windowStatus, windowErr := findBestWindowForAssignment(
			assignment.TeacherID,
			assignment.DepartmentID,
			assignment.Semester,
			assignment.CourseID,
		)
		if windowErr != nil {
			return nil, windowErr
		}

		studentRows, studentErr := db.DB.Query(`
			SELECT s.id, COALESCE(s.learning_mode_id, 2)
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ? AND csta.teacher_id = ? AND csta.status = 1
		`, assignment.CourseID, assignment.TeacherID)
		if studentErr != nil {
			return nil, studentErr
		}

		assigned := 0
		completed := 0
		studentIDs := make([]int, 0)
		expectedByMode := make(map[int]map[int]bool)
		for studentRows.Next() {
			var studentID int
			var learningMode int
			if scanErr := studentRows.Scan(&studentID, &learningMode); scanErr != nil {
				continue
			}
			assigned++
			studentIDs = append(studentIDs, studentID)

			expected, ok := expectedByMode[learningMode]
			if !ok {
				expected, err = getExpectedComponents(courseTypeID, learningMode, allowedComponents)
				if err != nil {
					studentRows.Close()
					return nil, err
				}
				expectedByMode[learningMode] = expected
			}

			isComplete, _, _, completeErr := isStudentCompleteForAssignment(studentID, assignment.CourseID, assignment.TeacherID, expected)
			if completeErr != nil {
				studentRows.Close()
				return nil, completeErr
			}
			if isComplete {
				completed++
			}
		}
		studentRows.Close()

		completion := 0.0
		if assigned > 0 {
			completion = (float64(completed) / float64(assigned)) * 100
		}

		late := false
		if windowEnd != nil && windowEnd.Before(now) && completion < 100 {
			late = true
		}

		extensionStatus := "none"
		if windowID != nil {
			extensionStatus = getLatestExtensionStatus(*windowID, assignment.CourseID, assignment.TeacherID)
		}

		overview := markEntryOverviewRow{
			CourseID:            assignment.CourseID,
			CourseCode:          assignment.CourseCode,
			CourseName:          assignment.CourseName,
			TeacherID:           assignment.TeacherID,
			TeacherName:         assignment.TeacherName,
			DepartmentID:        assignment.DepartmentID,
			DepartmentName:      assignment.Department,
			Semester:            assignment.Semester,
			WindowID:            windowID,
			WindowStatus:        windowStatus,
			AssignedStudents:    assigned,
			CompletedStudents:   completed,
			CompletionPercent:   completion,
			Late:                late,
			ExtensionStatus:     extensionStatus,
			AllowedComponentIDs: allowedComponents,
			StudentIDs:          studentIDs,
		}

		if windowStart != nil {
			s := windowStart.Local().Format(markEntryTimeLayout)
			overview.WindowStartAt = &s
		}
		if windowEnd != nil {
			e := windowEnd.Local().Format(markEntryTimeLayout)
			overview.WindowEndAt = &e
		}

		rows = append(rows, overview)
	}

	return rows, nil
}

func GetHODMarkEntryOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	if semesterStr == "" {
		http.Error(w, "semester is required", http.StatusBadRequest)
		return
	}
	semester, err := strconv.Atoi(semesterStr)
	if err != nil || semester <= 0 {
		http.Error(w, "invalid semester", http.StatusBadRequest)
		return
	}

	departmentID, departmentName, err := getDepartmentContextFromUsername(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	overviewRows, err := buildMarkEntryOverview(departmentID, semester)
	if err != nil {
		log.Printf("Error building HOD mark entry overview: %v", err)
		http.Error(w, "failed to build overview", http.StatusInternalServerError)
		return
	}

	total := len(overviewRows)
	completed := 0
	late := 0
	requested := 0
	notRequested := 0
	for _, row := range overviewRows {
		if row.CompletionPercent >= 100 {
			completed++
		}
		if row.Late {
			late++
			if row.ExtensionStatus == "pending" || row.ExtensionStatus == "approved" {
				requested++
			} else {
				notRequested++
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"department_id":   departmentID,
		"department_name": departmentName,
		"semester":        semester,
		"summary": map[string]int{
			"total_assignments": total,
			"completed":         completed,
			"late":              late,
			"requested":         requested,
			"not_requested":     notRequested,
		},
		"rows": overviewRows,
	})
}

func GetTeacherEnteredStudents(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.DB.Query(`
		SELECT
			s.id,
			COALESCE(s.enrollment_no, ''),
			s.student_name,
			COALESCE(SUM(sm.obtained_marks), 0) AS total_marks,
			COUNT(sm.id) > 0 AS has_entries
		FROM course_student_teacher_allocation csta
		JOIN students s ON s.id = csta.student_id
		LEFT JOIN student_marks sm ON sm.student_id = s.id AND sm.course_id = csta.course_id AND sm.faculty_id = csta.teacher_id
		WHERE csta.teacher_id = ? AND csta.course_id = ?
		GROUP BY s.id, s.enrollment_no, s.student_name
		ORDER BY s.student_name
	`, teacherID, courseID)
	if err != nil {
		log.Printf("Error fetching teacher entered students: %v", err)
		http.Error(w, "failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var studentID int
		var enrollmentNo string
		var studentName string
		var totalMarks float64
		var hasEntries bool
		if scanErr := rows.Scan(&studentID, &enrollmentNo, &studentName, &totalMarks, &hasEntries); scanErr != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"student_id":     studentID,
			"enrollment_no":  enrollmentNo,
			"student_name":   studentName,
			"total_marks":    totalMarks,
			"has_mark_entry": hasEntries,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"teacher_id": teacherID,
		"course_id":  courseID,
		"students":   items,
	})
}

func CreateMarkEntryExtensionRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req extensionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.RequesterRole = strings.ToLower(strings.TrimSpace(req.RequesterRole))
	req.RequesterUser = strings.TrimSpace(req.RequesterUser)
	req.TeacherID = strings.TrimSpace(req.TeacherID)
	req.Reason = strings.TrimSpace(req.Reason)

	if req.WindowID <= 0 || req.CourseID <= 0 || req.TeacherID == "" || req.RequesterUser == "" || req.RequesterRole == "" || req.Reason == "" {
		http.Error(w, "window_id, course_id, teacher_id, requester_username, requester_role and reason are required", http.StatusBadRequest)
		return
	}

	if req.RequesterRole != "teacher" && req.RequesterRole != "hod" {
		http.Error(w, "only teacher or hod can request extension", http.StatusForbidden)
		return
	}

	requestedEndAt, err := parseDateTime(req.RequestedEndAt)
	if err != nil {
		http.Error(w, "invalid requested_end_at", http.StatusBadRequest)
		return
	}

	var windowEndAt time.Time
	err = db.DB.QueryRow(`SELECT end_at FROM mark_entry_windows WHERE id = ?`, req.WindowID).Scan(&windowEndAt)
	if err != nil {
		http.Error(w, "window not found", http.StatusBadRequest)
		return
	}

	if !requestedEndAt.After(windowEndAt) {
		http.Error(w, "requested_end_at must be later than window end", http.StatusBadRequest)
		return
	}

	if req.RequesterRole == "teacher" {
		var exists int
		err = db.DB.QueryRow(`
			SELECT COUNT(*)
			FROM teacher_course_allocation
			WHERE teacher_id = ? AND course_id = ?
		`, req.TeacherID, req.CourseID).Scan(&exists)
		if err != nil || exists == 0 {
			http.Error(w, "teacher not allocated to this course", http.StatusForbidden)
			return
		}
	}

	if req.RequesterRole == "hod" {
		hodDepartmentID, _, depErr := getDepartmentContextFromUsername(req.RequesterUser)
		if depErr != nil {
			http.Error(w, "unable to resolve hod department", http.StatusForbidden)
			return
		}

		var teacherDepartmentID int
		err = db.DB.QueryRow(`
			SELECT CAST(t.dept AS UNSIGNED)
			FROM teachers t
			WHERE t.faculty_id = ?
		`, req.TeacherID).Scan(&teacherDepartmentID)
		if err != nil || teacherDepartmentID != hodDepartmentID {
			http.Error(w, "hod can request only for own department", http.StatusForbidden)
			return
		}
	}

	_, err = db.DB.Exec(`
		INSERT INTO mark_entry_extension_requests
		(window_id, course_id, teacher_id, department_id, semester, exam_type, requester_username, requester_role, reason, requested_end_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')
	`, req.WindowID, req.CourseID, req.TeacherID, req.DepartmentID, req.Semester, strings.TrimSpace(req.ExamType), req.RequesterUser, req.RequesterRole, req.Reason, requestedEndAt)
	if err != nil {
		log.Printf("Error creating extension request: %v", err)
		http.Error(w, "failed to create extension request", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "extension request submitted",
	})
}

func GetMarkEntryExtensionRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	requester := strings.TrimSpace(r.URL.Query().Get("requester_username"))
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))

	query := `
		SELECT
			er.id,
			er.window_id,
			er.course_id,
			COALESCE(c.course_code, ''),
			COALESCE(c.course_name, ''),
			er.teacher_id,
			COALESCE(t.name, er.teacher_id),
			COALESCE(er.department_id, 0),
			COALESCE(d.department_name, ''),
			er.semester,
			COALESCE(er.exam_type, ''),
			er.requester_username,
			er.requester_role,
			er.reason,
			er.requested_end_at,
			er.status,
			COALESCE(er.approver_username, ''),
			er.approved_end_at,
			COALESCE(er.rejection_reason, ''),
			er.created_at,
			er.updated_at
		FROM mark_entry_extension_requests er
		LEFT JOIN courses c ON c.id = er.course_id
		LEFT JOIN teachers t ON t.faculty_id = er.teacher_id
		LEFT JOIN departments d ON d.id = er.department_id
		WHERE 1 = 1
	`
	args := make([]interface{}, 0)

	if status != "" {
		query += " AND er.status = ?"
		args = append(args, status)
	}

	if requester != "" && (role == "teacher" || role == "hod") {
		query += " AND er.requester_username = ?"
		args = append(args, requester)
	}

	query += " ORDER BY er.created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching extension requests: %v", err)
		http.Error(w, "failed to fetch extension requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id              int
			windowID        int
			courseID        int
			courseCode      string
			courseName      string
			teacherID       string
			teacherName     string
			departmentID    int
			departmentName  string
			semester        sql.NullInt64
			examType        string
			requesterUser   string
			requesterRole   string
			reason          string
			requestedEndAt  time.Time
			requestStatus   string
			approver        string
			approvedEndAt   sql.NullTime
			rejectionReason string
			createdAt       time.Time
			updatedAt       time.Time
		)

		if scanErr := rows.Scan(
			&id,
			&windowID,
			&courseID,
			&courseCode,
			&courseName,
			&teacherID,
			&teacherName,
			&departmentID,
			&departmentName,
			&semester,
			&examType,
			&requesterUser,
			&requesterRole,
			&reason,
			&requestedEndAt,
			&requestStatus,
			&approver,
			&approvedEndAt,
			&rejectionReason,
			&createdAt,
			&updatedAt,
		); scanErr != nil {
			continue
		}

		item := map[string]interface{}{
			"id":                 id,
			"window_id":          windowID,
			"course_id":          courseID,
			"course_code":        courseCode,
			"course_name":        courseName,
			"teacher_id":         teacherID,
			"teacher_name":       teacherName,
			"department_id":      departmentID,
			"department_name":    departmentName,
			"exam_type":          examType,
			"requester_username": requesterUser,
			"requester_role":     requesterRole,
			"reason":             reason,
			"requested_end_at":   requestedEndAt.Format(time.RFC3339),
			"status":             requestStatus,
			"approver_username":  approver,
			"rejection_reason":   rejectionReason,
			"created_at":         createdAt.Format(time.RFC3339),
			"updated_at":         updatedAt.Format(time.RFC3339),
		}
		if semester.Valid {
			item["semester"] = int(semester.Int64)
		}
		if approvedEndAt.Valid {
			item["approved_end_at"] = approvedEndAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	json.NewEncoder(w).Encode(items)
}

func UpdateMarkEntryExtensionRequestStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 5 {
		http.Error(w, "invalid request path", http.StatusBadRequest)
		return
	}
	requestID, err := strconv.Atoi(pathParts[len(pathParts)-2])
	if err != nil || requestID <= 0 {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}
	action := strings.ToLower(strings.TrimSpace(pathParts[len(pathParts)-1]))

	var body struct {
		ApproverUsername string `json:"approver_username"`
		ApproverRole     string `json:"approver_role"`
		ApprovedEndAt    string `json:"approved_end_at,omitempty"`
		RejectionReason  string `json:"rejection_reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	body.ApproverRole = strings.ToLower(strings.TrimSpace(body.ApproverRole))
	if body.ApproverRole != "admin" && body.ApproverRole != "coe" {
		http.Error(w, "only admin or coe can approve/reject", http.StatusForbidden)
		return
	}

	if strings.TrimSpace(body.ApproverUsername) == "" {
		http.Error(w, "approver_username is required", http.StatusBadRequest)
		return
	}

	var currentStatus string
	var windowID int
	err = db.DB.QueryRow(`SELECT status, window_id FROM mark_entry_extension_requests WHERE id = ?`, requestID).Scan(&currentStatus, &windowID)
	if err != nil {
		http.Error(w, "request not found", http.StatusNotFound)
		return
	}
	if currentStatus != "pending" {
		http.Error(w, "request already processed", http.StatusConflict)
		return
	}

	if action == "approve" {
		approvedEndAt, parseErr := parseDateTime(body.ApprovedEndAt)
		if parseErr != nil {
			http.Error(w, "approved_end_at is required for approval", http.StatusBadRequest)
			return
		}

		_, err = db.DB.Exec(`
			UPDATE mark_entry_extension_requests
			SET status = 'approved', approver_username = ?, approved_end_at = ?, updated_at = UTC_TIMESTAMP()
			WHERE id = ?
		`, body.ApproverUsername, approvedEndAt, requestID)
		if err != nil {
			http.Error(w, "failed to approve request", http.StatusInternalServerError)
			return
		}

		_, err = db.DB.Exec(`
			UPDATE mark_entry_windows
			SET end_at = ?, updated_at = UTC_TIMESTAMP()
			WHERE id = ?
		`, approvedEndAt, windowID)
		if err != nil {
			http.Error(w, "failed to apply extension to window", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "extension approved"})
		return
	}

	if action == "reject" {
		reason := strings.TrimSpace(body.RejectionReason)
		if reason == "" {
			reason = "Rejected by approver"
		}

		_, err = db.DB.Exec(`
			UPDATE mark_entry_extension_requests
			SET status = 'rejected', approver_username = ?, rejection_reason = ?, updated_at = UTC_TIMESTAMP()
			WHERE id = ?
		`, body.ApproverUsername, reason, requestID)
		if err != nil {
			http.Error(w, "failed to reject request", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "extension rejected"})
		return
	}

	http.Error(w, "invalid action", http.StatusBadRequest)
}

func GetMarkEntryExtensionAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id"))
	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	if departmentIDStr == "" || semesterStr == "" {
		http.Error(w, "department_id and semester are required", http.StatusBadRequest)
		return
	}

	departmentID, depErr := strconv.Atoi(departmentIDStr)
	semester, semErr := strconv.Atoi(semesterStr)
	if depErr != nil || semErr != nil || departmentID <= 0 || semester <= 0 {
		http.Error(w, "invalid department_id or semester", http.StatusBadRequest)
		return
	}

	overviewRows, err := buildMarkEntryOverview(departmentID, semester)
	if err != nil {
		http.Error(w, "failed to build analytics", http.StatusInternalServerError)
		return
	}

	late := 0
	requested := 0
	notRequested := 0
	approved := 0
	pending := 0
	rejected := 0

	for _, row := range overviewRows {
		if row.Late {
			late++
			switch row.ExtensionStatus {
			case "approved":
				requested++
				approved++
			case "pending":
				requested++
				pending++
			case "rejected":
				requested++
				rejected++
			default:
				notRequested++
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"department_id": departmentID,
		"semester":      semester,
		"late":          late,
		"requested":     requested,
		"not_requested": notRequested,
		"approved":      approved,
		"pending":       pending,
		"rejected":      rejected,
	})
}

func buildResultAnalysis(departmentID int, semester int) ([]resultAnalysisRow, error) {
	assignments, err := getDepartmentAssignments(departmentID, semester)
	if err != nil {
		return nil, err
	}

	courseMap := make(map[int]resultAnalysisRow)
	courseTeacherMap := make(map[int]map[string]bool)
	for _, assignment := range assignments {
		if _, exists := courseMap[assignment.CourseID]; !exists {
			courseMap[assignment.CourseID] = resultAnalysisRow{
				CourseID:   assignment.CourseID,
				CourseCode: assignment.CourseCode,
				CourseName: assignment.CourseName,
				Ranges:     map[string]int{"0-10": 0, "11-20": 0, "21-24": 0, "25-30": 0, "31-40": 0, "41-50": 0},
			}
			courseTeacherMap[assignment.CourseID] = make(map[string]bool)
		}
		courseTeacherMap[assignment.CourseID][assignment.TeacherID] = true
	}

	rows := make([]resultAnalysisRow, 0, len(courseMap))
	for courseID, base := range courseMap {
		teachers := courseTeacherMap[courseID]

		studentRows, err := db.DB.Query(`
			SELECT DISTINCT csta.student_id, s.student_name
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ?
		`, courseID)
		if err != nil {
			return nil, err
		}

		studentTotals := make([]studentMarkTotal, 0)
		for studentRows.Next() {
			var studentID int
			var studentName string
			if scanErr := studentRows.Scan(&studentID, &studentName); scanErr != nil {
				continue
			}

			total := 0.0
			hasMarks := false
			for teacherID := range teachers {
				markRows, markErr := db.DB.Query(`
					SELECT obtained_marks
					FROM student_marks
					WHERE course_id = ? AND student_id = ? AND faculty_id = ?
				`, courseID, studentID, teacherID)
				if markErr != nil {
					continue
				}
				for markRows.Next() {
					var marks sql.NullFloat64
					if scanErr := markRows.Scan(&marks); scanErr == nil && marks.Valid {
						total += marks.Float64
						hasMarks = true
					}
				}
				markRows.Close()
			}

			studentTotals = append(studentTotals, studentMarkTotal{
				StudentID:   studentID,
				StudentName: studentName,
				TotalMarks:  total,
				Appeared:    hasMarks,
			})
		}
		studentRows.Close()

		analysis := base
		analysis.Registered = len(studentTotals)
		analysis.StudentMarkItems = studentTotals

		maxVal := 0.0
		minVal := 0.0
		totalVal := 0.0
		appeared := 0
		passed := 0
		for idx, item := range studentTotals {
			if !item.Appeared {
				continue
			}
			appeared++
			totalVal += item.TotalMarks
			if idx == 0 || item.TotalMarks > maxVal {
				maxVal = item.TotalMarks
			}
			if idx == 0 || item.TotalMarks < minVal {
				minVal = item.TotalMarks
			}

			if item.TotalMarks >= 25 {
				passed++
			}

			switch {
			case item.TotalMarks <= 10:
				analysis.Ranges["0-10"]++
			case item.TotalMarks <= 20:
				analysis.Ranges["11-20"]++
			case item.TotalMarks <= 24:
				analysis.Ranges["21-24"]++
			case item.TotalMarks <= 30:
				analysis.Ranges["25-30"]++
			case item.TotalMarks <= 40:
				analysis.Ranges["31-40"]++
			default:
				analysis.Ranges["41-50"]++
			}
		}

		analysis.Appeared = appeared
		analysis.Absent = analysis.Registered - appeared
		analysis.Passed = passed
		analysis.Failed = analysis.Appeared - analysis.Passed
		analysis.MaximumMark = maxVal
		analysis.MinimumMark = minVal
		if appeared > 0 {
			analysis.AverageMark = totalVal / float64(appeared)
			analysis.PassPercent = (float64(analysis.Passed) / float64(appeared)) * 100
		}

		rows = append(rows, analysis)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CourseCode < rows[j].CourseCode
	})

	return rows, nil
}

func GetResultAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	if semesterStr == "" {
		http.Error(w, "semester is required", http.StatusBadRequest)
		return
	}
	semester, err := strconv.Atoi(semesterStr)
	if err != nil || semester <= 0 {
		http.Error(w, "invalid semester", http.StatusBadRequest)
		return
	}

	departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id"))
	username := strings.TrimSpace(r.URL.Query().Get("username"))

	departmentID := 0
	departmentName := ""
	if departmentIDStr != "" {
		departmentID, err = strconv.Atoi(departmentIDStr)
		if err != nil || departmentID <= 0 {
			http.Error(w, "invalid department_id", http.StatusBadRequest)
			return
		}
		_ = db.DB.QueryRow(`SELECT department_name FROM departments WHERE id = ?`, departmentID).Scan(&departmentName)
	} else {
		if username == "" {
			http.Error(w, "username is required when department_id is absent", http.StatusBadRequest)
			return
		}
		departmentID, departmentName, err = getDepartmentContextFromUsername(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	rows, err := buildResultAnalysis(departmentID, semester)
	if err != nil {
		log.Printf("Error generating result analysis: %v", err)
		http.Error(w, "failed to generate result analysis", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"department_id":   departmentID,
		"department_name": departmentName,
		"semester":        semester,
		"exam_type":       strings.TrimSpace(r.URL.Query().Get("exam_type")),
		"rows":            rows,
	})
}

func exportStudentWiseRows(departmentID int, semester int) ([][]string, error) {
	overviewRows, err := buildMarkEntryOverview(departmentID, semester)
	if err != nil {
		return nil, err
	}

	records := [][]string{{"Course Code", "Course Name", "Teacher ID", "Teacher Name", "Student ID", "Enrollment No", "Student Name", "Has Mark Entry", "Completion"}}
	for _, row := range overviewRows {
		students, queryErr := db.DB.Query(`
			SELECT
				s.id,
				COALESCE(s.enrollment_no, ''),
				s.student_name,
				COALESCE(s.learning_mode_id, 2)
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ? AND csta.teacher_id = ? AND csta.status = 1
		`, row.CourseID, row.TeacherID)
		if queryErr != nil {
			continue
		}

		courseTypeID, typeErr := getCourseTypeForCourse(row.CourseID)
		if typeErr != nil {
			students.Close()
			continue
		}

		for students.Next() {
			var studentID int
			var enrollmentNo string
			var studentName string
			var learningMode int
			if scanErr := students.Scan(&studentID, &enrollmentNo, &studentName, &learningMode); scanErr != nil {
				continue
			}

			expected, expectedErr := getExpectedComponents(courseTypeID, learningMode, row.AllowedComponentIDs)
			if expectedErr != nil {
				continue
			}
			complete, _, hasEntry, completeErr := isStudentCompleteForAssignment(studentID, row.CourseID, row.TeacherID, expected)
			if completeErr != nil {
				continue
			}

			records = append(records, []string{
				row.CourseCode,
				row.CourseName,
				row.TeacherID,
				row.TeacherName,
				strconv.Itoa(studentID),
				enrollmentNo,
				studentName,
				strconv.FormatBool(hasEntry),
				strconv.FormatBool(complete),
			})
		}
		students.Close()
	}

	return records, nil
}

func exportCourseWiseRows(departmentID int, semester int) ([][]string, error) {
	overviewRows, err := buildMarkEntryOverview(departmentID, semester)
	if err != nil {
		return nil, err
	}

	records := [][]string{{"Course Code", "Course Name", "Teacher ID", "Teacher Name", "Assigned Students", "Completed Students", "Completion %", "Window Status", "Late", "Extension Status"}}
	for _, row := range overviewRows {
		records = append(records, []string{
			row.CourseCode,
			row.CourseName,
			row.TeacherID,
			row.TeacherName,
			strconv.Itoa(row.AssignedStudents),
			strconv.Itoa(row.CompletedStudents),
			fmt.Sprintf("%.2f", row.CompletionPercent),
			row.WindowStatus,
			strconv.FormatBool(row.Late),
			row.ExtensionStatus,
		})
	}

	return records, nil
}

func writeCSVAttachment(w http.ResponseWriter, filename string, records [][]string) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	writer.WriteAll(records)
	writer.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buffer.Bytes())
}

func writeXLSXAttachment(w http.ResponseWriter, filename string, records [][]string) error {
	file := excelize.NewFile()
	sheetName := "Report"
	index, err := file.NewSheet(sheetName)
	if err != nil {
		return err
	}

	for rowIdx, row := range records {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			_ = file.SetCellValue(sheetName, cell, value)
		}
	}

	file.SetActiveSheet(index)
	if err := file.DeleteSheet("Sheet1"); err != nil {
		// ignore delete failures
	}

	buf, err := file.WriteToBuffer()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
	return nil
}

func writePDFAttachment(w http.ResponseWriter, filename string, title string, records [][]string) error {
	var html strings.Builder
	html.WriteString("<!DOCTYPE html><html><head><meta charset='utf-8'><style>")
	html.WriteString("body{font-family:Arial,sans-serif;font-size:12px}table{width:100%;border-collapse:collapse}th,td{border:1px solid #999;padding:6px;text-align:left}th{background:#f2f2f2}")
	html.WriteString("</style></head><body>")
	html.WriteString("<h2>" + title + "</h2><table>")
	for idx, row := range records {
		if idx == 0 {
			html.WriteString("<thead><tr>")
			for _, col := range row {
				html.WriteString("<th>" + col + "</th>")
			}
			html.WriteString("</tr></thead><tbody>")
			continue
		}
		html.WriteString("<tr>")
		for _, col := range row {
			html.WriteString("<td>" + col + "</td>")
		}
		html.WriteString("</tr>")
	}
	html.WriteString("</tbody></table></body></html>")

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var pdfBuf []byte
	err := chromedp.Run(ctx, printToPDF(html.String(), &pdfBuf))
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBuf)
	return nil
}

func DownloadMarkEntryReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	reportType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("report_type")))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))

	if username == "" || semesterStr == "" || reportType == "" || format == "" {
		http.Error(w, "username, semester, report_type and format are required", http.StatusBadRequest)
		return
	}

	semester, err := strconv.Atoi(semesterStr)
	if err != nil || semester <= 0 {
		http.Error(w, "invalid semester", http.StatusBadRequest)
		return
	}

	departmentID, departmentName, err := getDepartmentContextFromUsername(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var records [][]string
	if reportType == "student" {
		records, err = exportStudentWiseRows(departmentID, semester)
	} else if reportType == "course" {
		records, err = exportCourseWiseRows(departmentID, semester)
	} else {
		http.Error(w, "report_type must be student or course", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error generating report records: %v", err)
		http.Error(w, "failed to generate report", http.StatusInternalServerError)
		return
	}

	baseName := fmt.Sprintf("mark_entry_%s_sem%d_dept%s", reportType, semester, strings.ReplaceAll(strings.ToLower(departmentName), " ", "_"))
	switch format {
	case "csv":
		writeCSVAttachment(w, baseName+".csv", records)
	case "xlsx":
		if writeErr := writeXLSXAttachment(w, baseName+".xlsx", records); writeErr != nil {
			http.Error(w, "failed to generate xlsx", http.StatusInternalServerError)
		}
	case "pdf":
		title := fmt.Sprintf("Mark Entry %s Report - Semester %d - %s", strings.Title(reportType), semester, departmentName)
		if writeErr := writePDFAttachment(w, baseName+".pdf", title, records); writeErr != nil {
			http.Error(w, "failed to generate pdf", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "format must be csv, xlsx or pdf", http.StatusBadRequest)
	}
}
