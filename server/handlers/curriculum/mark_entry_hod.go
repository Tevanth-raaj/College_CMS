package curriculum

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
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
	WindowName          *string `json:"window_name,omitempty"`
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
	Components       []resultComponent  `json:"components"`
	StudentMarkItems []studentMarkTotal `json:"-"`
}

type resultComponent struct {
	ComponentID   int     `json:"component_id"`
	ComponentName string  `json:"component_name"`
	Registered    int     `json:"registered"`
	Appeared      int     `json:"appeared"`
	Absent        int     `json:"absent"`
	MaximumMark   float64 `json:"maximum_mark"`
	MinimumMark   float64 `json:"minimum_mark"`
	AverageMark   float64 `json:"average_mark"`
	TotalMarks    float64 `json:"total_marks"`
}

func doesComponentMatchExamType(componentName string, examType string) bool {
	rawFilter := strings.TrimSpace(examType)
	examType = strings.ToUpper(rawFilter)
	if examType == "" {
		return true
	}

	name := strings.ToUpper(strings.TrimSpace(componentName))
	switch examType {
	case "PT1", "PT-1", "PERIODICAL TEST 1":
		return strings.Contains(name, "PT1") || strings.Contains(name, "PT-1") || strings.Contains(name, "PERIODICAL TEST 1")
	case "PT2", "PT-2", "PERIODICAL TEST 2":
		return strings.Contains(name, "PT2") || strings.Contains(name, "PT-2") || strings.Contains(name, "PERIODICAL TEST 2")
	case "MODEL", "MODEL EXAM":
		return strings.Contains(name, "MODEL")
	case "ENDSEM", "END SEM", "END-SEM", "SEE":
		return strings.Contains(name, "END") || strings.Contains(name, "SEM") || strings.Contains(name, "SEE")
	default:
		return strings.EqualFold(strings.TrimSpace(componentName), rawFilter)
	}
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

type selectedWindowContext struct {
	ID           int
	WindowName   sql.NullString
	TeacherID    sql.NullString
	DepartmentID sql.NullInt64
	Semester     sql.NullInt64
	CourseID     sql.NullInt64
	StartAt      time.Time
	EndAt        time.Time
	Enabled      bool
	ComponentIDs []int
}

func getTeacherIdentifierAliases(teacherID string) []string {
	teacherID = strings.TrimSpace(teacherID)
	aliases := map[string]bool{}
	if teacherID != "" {
		aliases[teacherID] = true
	}

	var numericTeacherID sql.NullInt64
	_ = db.DB.QueryRow(`SELECT id FROM teachers WHERE faculty_id = ? LIMIT 1`, teacherID).Scan(&numericTeacherID)
	if numericTeacherID.Valid {
		aliases[strconv.FormatInt(numericTeacherID.Int64, 10)] = true
	}

	var teacherEmail sql.NullString
	_ = db.DB.QueryRow(`SELECT email FROM teachers WHERE faculty_id = ? LIMIT 1`, teacherID).Scan(&teacherEmail)
	if teacherEmail.Valid && strings.TrimSpace(teacherEmail.String) != "" {
		email := strings.TrimSpace(teacherEmail.String)
		aliases[email] = true

		var username sql.NullString
		_ = db.DB.QueryRow(`SELECT username FROM users WHERE email = ? LIMIT 1`, email).Scan(&username)
		if username.Valid && strings.TrimSpace(username.String) != "" {
			aliases[strings.TrimSpace(username.String)] = true
		}
	}

	var userEmail sql.NullString
	_ = db.DB.QueryRow(`SELECT email FROM users WHERE username = ? LIMIT 1`, teacherID).Scan(&userEmail)
	if userEmail.Valid && strings.TrimSpace(userEmail.String) != "" {
		email := strings.TrimSpace(userEmail.String)
		aliases[email] = true

		var facultyID sql.NullString
		_ = db.DB.QueryRow(`SELECT faculty_id FROM teachers WHERE email = ? LIMIT 1`, email).Scan(&facultyID)
		if facultyID.Valid && strings.TrimSpace(facultyID.String) != "" {
			aliases[strings.TrimSpace(facultyID.String)] = true
		}

		_ = db.DB.QueryRow(`SELECT id FROM teachers WHERE email = ? LIMIT 1`, email).Scan(&numericTeacherID)
		if numericTeacherID.Valid {
			aliases[strconv.FormatInt(numericTeacherID.Int64, 10)] = true
		}
	}

	out := make([]string, 0, len(aliases))
	for alias := range aliases {
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func getSelectedWindowContext(windowID int) (*selectedWindowContext, error) {
	ctx := &selectedWindowContext{}
	err := db.DB.QueryRow(`
		SELECT id, COALESCE(window_name, ''), teacher_id, department_id, semester, course_id, start_at, end_at, enabled
		FROM mark_entry_windows
		WHERE id = ?
	`, windowID).Scan(
		&ctx.ID,
		&ctx.WindowName,
		&ctx.TeacherID,
		&ctx.DepartmentID,
		&ctx.Semester,
		&ctx.CourseID,
		&ctx.StartAt,
		&ctx.EndAt,
		&ctx.Enabled,
	)
	if err != nil {
		return nil, err
	}

	rows, err := db.DB.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ctx.ComponentIDs = []int{}
	for rows.Next() {
		var componentID int
		if scanErr := rows.Scan(&componentID); scanErr == nil {
			ctx.ComponentIDs = append(ctx.ComponentIDs, componentID)
		}
	}

	return ctx, nil
}

func doesWindowMatchAssignment(window *selectedWindowContext, assignment markEntryAssignmentRow) bool {
	if window == nil {
		return true
	}
	if window.TeacherID.Valid && strings.TrimSpace(window.TeacherID.String) != "" && strings.TrimSpace(window.TeacherID.String) != assignment.TeacherID {
		return false
	}
	if window.DepartmentID.Valid && int(window.DepartmentID.Int64) > 0 && int(window.DepartmentID.Int64) != assignment.DepartmentID {
		return false
	}
	if window.Semester.Valid && int(window.Semester.Int64) > 0 && int(window.Semester.Int64) != assignment.Semester {
		return false
	}
	if window.CourseID.Valid && int(window.CourseID.Int64) > 0 && int(window.CourseID.Int64) != assignment.CourseID {
		return false
	}
	return true
}

func getAbsentComponentsByStudent(windowID int, courseID int) map[int]map[int]bool {
	result := map[int]map[int]bool{}
	rows, err := db.DB.Query(`
		SELECT student_id, mark_category_id
		FROM exam_absentees
		WHERE window_id = ? AND course_id = ?
	`, windowID, courseID)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var studentID int
		var componentID int
		if scanErr := rows.Scan(&studentID, &componentID); scanErr != nil {
			continue
		}
		if _, ok := result[studentID]; !ok {
			result[studentID] = map[int]bool{}
		}
		result[studentID][componentID] = true
	}

	return result
}

func getEnteredMarksByStudent(courseID int, aliases []string) map[int]map[int]float64 {
	result := map[int]map[int]float64{}
	if len(aliases) == 0 {
		return result
	}

	placeholders := make([]string, len(aliases))
	args := make([]interface{}, 0, len(aliases)+1)
	args = append(args, courseID)
	for index, alias := range aliases {
		placeholders[index] = "?"
		args = append(args, strings.ToLower(strings.TrimSpace(alias)))
	}

	query := fmt.Sprintf(`
		SELECT student_id, assessment_component_id, COALESCE(obtained_marks, 0)
		FROM student_marks
		WHERE course_id = ?
		  AND (
			LOWER(TRIM(COALESCE(faculty_id, ''))) IN (%s)
			OR TRIM(COALESCE(faculty_id, '')) = ''
		  )
	`, strings.Join(placeholders, ","))

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var studentID int
		var componentID int
		var marks float64
		if scanErr := rows.Scan(&studentID, &componentID, &marks); scanErr != nil {
			continue
		}
		if _, ok := result[studentID]; !ok {
			result[studentID] = map[int]float64{}
		}
		result[studentID][componentID] = marks
	}

	return result
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
		LEFT JOIN department_teachers dt ON dt.teacher_id = t.faculty_id
		LEFT JOIN departments d ON d.id = COALESCE(dt.department_id, CAST(t.dept AS UNSIGNED))
		WHERE t.email = ?
		  AND d.id IS NOT NULL
		LIMIT 1
	`, strings.TrimSpace(email.String)).Scan(&departmentID, &departmentName)
	if err == nil {
		return departmentID, departmentName, nil
	}

	return 0, "", fmt.Errorf("department mapping not found for user")
}

func findBestWindowForAssignment(teacherID string, departmentID int, semester int, courseID int) (*int, *string, *time.Time, *time.Time, []int, string, error) {
	query := `
		SELECT id, COALESCE(window_name, ''), start_at, end_at
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
	var windowName string
	var startAt, endAt time.Time
	err := db.DB.QueryRow(query, teacherID, departmentID, semester, courseID).Scan(&windowID, &windowName, &startAt, &endAt)
	if err == sql.ErrNoRows {
		status := "not_configured"
		return nil, nil, nil, nil, []int{}, status, nil
	}
	if err != nil {
		return nil, nil, nil, nil, nil, "", err
	}

	componentRows, err := db.DB.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
	if err != nil {
		return nil, nil, nil, nil, nil, "", err
	}
	defer componentRows.Close()

	componentIDs := make([]int, 0)
	for componentRows.Next() {
		var componentID int
		if scanErr := componentRows.Scan(&componentID); scanErr == nil {
			componentIDs = append(componentIDs, componentID)
		}
	}

	var windowNamePtr *string
	trimmedWindowName := strings.TrimSpace(windowName)
	if trimmedWindowName != "" {
		windowNamePtr = &trimmedWindowName
	}

	now := time.Now().UTC()
	status := "closed"
	if now.Before(startAt) {
		status = "upcoming"
	} else if now.After(startAt) && now.Before(endAt) {
		status = "open"
	}

	return &windowID, windowNamePtr, &startAt, &endAt, componentIDs, status, nil
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
	aliases := getTeacherIdentifierAliases(teacherID)
	enteredMarksByStudent := getEnteredMarksByStudent(courseID, aliases)
	studentMarks := enteredMarksByStudent[studentID]

	if studentMarks == nil {
		if len(expected) == 0 {
			return false, 0, false, nil
		}
		return false, 0, false, nil
	}

	total := 0.0
	hasAny := false
	for _, marks := range studentMarks {
		total += marks
		hasAny = true
	}

	if len(expected) == 0 {
		return hasAny, total, hasAny, nil
	}

	for componentID := range expected {
		if _, ok := studentMarks[componentID]; !ok {
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
			COALESCE(d_curr.id, d_teacher.id, 0) AS department_id,
			COALESCE(d_curr.department_name, d_teacher.department_name, 'Unmapped') AS department_name,
			nc.semester_number
		FROM teacher_course_allocation tca
		JOIN courses c ON c.id = tca.course_id
		JOIN curriculum_courses cc ON cc.course_id = c.id
		JOIN normal_cards nc ON nc.id = cc.semester_id
		LEFT JOIN teachers t ON t.faculty_id = tca.teacher_id
		LEFT JOIN department_teachers dt ON dt.teacher_id = tca.teacher_id AND dt.status = 1
		LEFT JOIN departments d_curr ON d_curr.current_curriculum_id = cc.curriculum_id
		LEFT JOIN departments d_teacher ON d_teacher.id = COALESCE(dt.department_id, CAST(t.dept AS UNSIGNED))
		WHERE (d_curr.id = ? OR d_teacher.id = ?)
			AND nc.semester_number = ?
		ORDER BY c.course_code, teacher_name
	`, departmentID, departmentID, semester)
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
		teacherAliases := getTeacherIdentifierAliases(assignment.TeacherID)
		enteredMarksByStudent := getEnteredMarksByStudent(assignment.CourseID, teacherAliases)

		windowID, windowName, windowStart, windowEnd, allowedComponents, windowStatus, windowErr := findBestWindowForAssignment(
			assignment.TeacherID,
			assignment.DepartmentID,
			assignment.Semester,
			assignment.CourseID,
		)
		if windowErr != nil {
			return nil, windowErr
		}

		completionAllowedComponents := allowedComponents
		if len(allowedComponents) > 0 {
			allowedSet := make(map[int]bool, len(allowedComponents))
			for _, componentID := range allowedComponents {
				allowedSet[componentID] = true
			}

			hasMarksInAllowed := false
			for _, studentMarks := range enteredMarksByStudent {
				for componentID := range studentMarks {
					if allowedSet[componentID] {
						hasMarksInAllowed = true
						break
					}
				}
				if hasMarksInAllowed {
					break
				}
			}

			if !hasMarksInAllowed {
				completionAllowedComponents = []int{}
			}
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
		absentByStudent := map[int]map[int]bool{}
		if windowID != nil {
			absentByStudent = getAbsentComponentsByStudent(*windowID, assignment.CourseID)
		}
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
				expected, err = getExpectedComponents(courseTypeID, learningMode, completionAllowedComponents)
				if err != nil {
					studentRows.Close()
					return nil, err
				}
				expectedByMode[learningMode] = expected
			}

			expectedFiltered := map[int]bool{}
			for componentID := range expected {
				if absentByStudent[studentID][componentID] {
					continue
				}
				expectedFiltered[componentID] = true
			}

			studentMarks := enteredMarksByStudent[studentID]
			isComplete := true
			for componentID := range expectedFiltered {
				if studentMarks == nil {
					isComplete = false
					break
				}
				if _, ok := studentMarks[componentID]; !ok {
					isComplete = false
					break
				}
			}

			if len(expectedFiltered) == 0 {
				isComplete = true
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
			WindowName:          windowName,
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

func getAssignmentsForWindowMonitor(semester int, departmentID *int) ([]markEntryAssignmentRow, error) {
	query := `
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			tca.teacher_id,
			COALESCE(t.name, tca.teacher_id) AS teacher_name,
			COALESCE(d.id, 0) AS department_id,
			COALESCE(d.department_name, 'Unmapped') AS department_name,
			nc.semester_number
		FROM teacher_course_allocation tca
		JOIN courses c ON c.id = tca.course_id
		JOIN curriculum_courses cc ON cc.course_id = c.id
		JOIN normal_cards nc ON nc.id = cc.semester_id
		LEFT JOIN teachers t ON t.faculty_id = tca.teacher_id
		LEFT JOIN department_teachers dt ON dt.teacher_id = tca.teacher_id
		LEFT JOIN departments d ON d.id = COALESCE(dt.department_id, CAST(t.dept AS UNSIGNED))
		WHERE nc.semester_number = ?
	`
	args := []interface{}{semester}
	if departmentID != nil {
		query += " AND d.id = ?"
		args = append(args, *departmentID)
	}
	query += " ORDER BY department_name, c.course_code, teacher_name"

	rows, err := db.DB.Query(query, args...)
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

func buildWindowMonitorRows(assignments []markEntryAssignmentRow, selectedWindow *selectedWindowContext) ([]markEntryOverviewRow, error) {
	now := time.Now().UTC()
	rows := make([]markEntryOverviewRow, 0)

	for _, assignment := range assignments {
		if !doesWindowMatchAssignment(selectedWindow, assignment) {
			continue
		}

		courseTypeID, typeErr := getCourseTypeForCourse(assignment.CourseID)
		if typeErr != nil {
			continue
		}

		var windowID *int
		var windowName *string
		var windowStart, windowEnd *time.Time
		allowedComponents := []int{}
		windowStatus := "not_configured"

		if selectedWindow != nil {
			windowID = &selectedWindow.ID
			if selectedWindow.WindowName.Valid {
				trimmedWindowName := strings.TrimSpace(selectedWindow.WindowName.String)
				if trimmedWindowName != "" {
					windowName = &trimmedWindowName
				}
			}
			windowStart = &selectedWindow.StartAt
			windowEnd = &selectedWindow.EndAt
			allowedComponents = selectedWindow.ComponentIDs
			if !selectedWindow.Enabled {
				windowStatus = "disabled"
			} else if now.Before(selectedWindow.StartAt) {
				windowStatus = "upcoming"
			} else if now.After(selectedWindow.EndAt) {
				windowStatus = "closed"
			} else {
				windowStatus = "open"
			}
		} else {
			bestWindowID, bestWindowName, startAt, endAt, components, status, err := findBestWindowForAssignment(
				assignment.TeacherID,
				assignment.DepartmentID,
				assignment.Semester,
				assignment.CourseID,
			)
			if err != nil {
				return nil, err
			}
			windowID = bestWindowID
			windowName = bestWindowName
			windowStart = startAt
			windowEnd = endAt
			allowedComponents = components
			windowStatus = status
		}

		teacherAliases := getTeacherIdentifierAliases(assignment.TeacherID)
		enteredMarksByStudent := getEnteredMarksByStudent(assignment.CourseID, teacherAliases)

		completionAllowedComponents := allowedComponents
		if len(allowedComponents) > 0 {
			allowedSet := make(map[int]bool, len(allowedComponents))
			for _, componentID := range allowedComponents {
				allowedSet[componentID] = true
			}

			hasMarksInAllowed := false
			for _, studentMarks := range enteredMarksByStudent {
				for componentID := range studentMarks {
					if allowedSet[componentID] {
						hasMarksInAllowed = true
						break
					}
				}
				if hasMarksInAllowed {
					break
				}
			}

			if !hasMarksInAllowed {
				completionAllowedComponents = []int{}
			}
		}
		absentByStudent := map[int]map[int]bool{}
		if windowID != nil {
			absentByStudent = getAbsentComponentsByStudent(*windowID, assignment.CourseID)
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
				expected, _ = getExpectedComponents(courseTypeID, learningMode, completionAllowedComponents)
				expectedByMode[learningMode] = expected
			}

			expectedFiltered := map[int]bool{}
			for componentID := range expected {
				if absentByStudent[studentID][componentID] {
					continue
				}
				expectedFiltered[componentID] = true
			}

			studentMarks := enteredMarksByStudent[studentID]
			isComplete := true
			for componentID := range expectedFiltered {
				if studentMarks == nil {
					isComplete = false
					break
				}
				if _, ok := studentMarks[componentID]; !ok {
					isComplete = false
					break
				}
			}
			if len(expectedFiltered) == 0 {
				isComplete = true
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
			WindowName:          windowName,
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

func GetHODWindowMonitor(w http.ResponseWriter, r *http.Request) {
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

	assignments, err := getAssignmentsForWindowMonitor(semester, &departmentID)
	if err != nil {
		log.Printf("[GetHODWindowMonitor] getAssignmentsForWindowMonitor failed: %v", err)
		http.Error(w, "failed to load assignments", http.StatusInternalServerError)
		return
	}

	var selectedWindow *selectedWindowContext
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		windowID, convErr := strconv.Atoi(windowIDStr)
		if convErr != nil || windowID <= 0 {
			http.Error(w, "invalid window_id", http.StatusBadRequest)
			return
		}
		selectedWindow, err = getSelectedWindowContext(windowID)
		if err != nil {
			http.Error(w, "window not found", http.StatusNotFound)
			return
		}
	}

	rows, err := buildWindowMonitorRows(assignments, selectedWindow)
	if err != nil {
		log.Printf("[GetHODWindowMonitor] buildWindowMonitorRows failed: %v", err)
		http.Error(w, "failed to build monitor rows", http.StatusInternalServerError)
		return
	}

	totalAssignments := len(rows)
	completedAssignments := 0
	totalStudents := 0
	completedStudents := 0
	for _, row := range rows {
		totalStudents += row.AssignedStudents
		completedStudents += row.CompletedStudents
		if row.CompletionPercent >= 100 {
			completedAssignments++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"department_id":   departmentID,
		"department_name": departmentName,
		"semester":        semester,
		"summary": map[string]int{
			"total_assignments":     totalAssignments,
			"completed_assignments": completedAssignments,
			"pending_assignments":   totalAssignments - completedAssignments,
			"total_students":        totalStudents,
			"completed_students":    completedStudents,
		},
		"rows": rows,
	})
}

func GetAdminWindowMonitor(w http.ResponseWriter, r *http.Request) {
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

	var departmentID *int
	if departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id")); departmentIDStr != "" {
		parsed, convErr := strconv.Atoi(departmentIDStr)
		if convErr != nil || parsed <= 0 {
			http.Error(w, "invalid department_id", http.StatusBadRequest)
			return
		}
		departmentID = &parsed
	}

	assignments, err := getAssignmentsForWindowMonitor(semester, departmentID)
	if err != nil {
		log.Printf("[GetAdminWindowMonitor] getAssignmentsForWindowMonitor failed: %v", err)
		http.Error(w, "failed to load assignments", http.StatusInternalServerError)
		return
	}

	var selectedWindow *selectedWindowContext
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		windowID, convErr := strconv.Atoi(windowIDStr)
		if convErr != nil || windowID <= 0 {
			http.Error(w, "invalid window_id", http.StatusBadRequest)
			return
		}
		selectedWindow, err = getSelectedWindowContext(windowID)
		if err != nil {
			http.Error(w, "window not found", http.StatusNotFound)
			return
		}
	}

	rows, err := buildWindowMonitorRows(assignments, selectedWindow)
	if err != nil {
		log.Printf("[GetAdminWindowMonitor] buildWindowMonitorRows failed: %v", err)
		http.Error(w, "failed to build monitor rows", http.StatusInternalServerError)
		return
	}

	totalAssignments := len(rows)
	completedAssignments := 0
	totalStudents := 0
	completedStudents := 0
	for _, row := range rows {
		totalStudents += row.AssignedStudents
		completedStudents += row.CompletedStudents
		if row.CompletionPercent >= 100 {
			completedAssignments++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"semester": semester,
		"summary": map[string]int{
			"total_assignments":     totalAssignments,
			"completed_assignments": completedAssignments,
			"pending_assignments":   totalAssignments - completedAssignments,
			"total_students":        totalStudents,
			"completed_students":    completedStudents,
		},
		"rows": rows,
	})
}

func AdminFillRandomMarksForWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		WindowID     int  `json:"window_id"`
		Semester     int  `json:"semester"`
		DepartmentID *int `json:"department_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.WindowID <= 0 {
		http.Error(w, "window_id is required", http.StatusBadRequest)
		return
	}
	if req.Semester <= 0 {
		http.Error(w, "semester is required", http.StatusBadRequest)
		return
	}

	selectedWindow, err := getSelectedWindowContext(req.WindowID)
	if err != nil {
		http.Error(w, "window not found", http.StatusNotFound)
		return
	}

	assignments, err := getAssignmentsForWindowMonitor(req.Semester, req.DepartmentID)
	if err != nil {
		log.Printf("[AdminFillRandomMarksForWindow] failed to load assignments: %v", err)
		http.Error(w, "failed to load assignments", http.StatusInternalServerError)
		return
	}

	type componentMeta struct {
		MaxMarks        float64
		ConversionMarks float64
	}

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))
	componentCache := make(map[int]componentMeta)
	entriesUpserted := 0
	studentsTouched := 0
	assignmentsMatched := 0
	absentSkipped := 0

	for _, assignment := range assignments {
		if !doesWindowMatchAssignment(selectedWindow, assignment) {
			continue
		}
		assignmentsMatched++

		normalizedFacultyID := normalizeFacultyIdentifier(assignment.TeacherID)
		if strings.TrimSpace(normalizedFacultyID) == "" {
			normalizedFacultyID = assignment.TeacherID
		}

		courseTypeID, typeErr := getCourseTypeForCourse(assignment.CourseID)
		if typeErr != nil {
			continue
		}

		absentByStudent := getAbsentComponentsByStudent(req.WindowID, assignment.CourseID)
		expectedByMode := make(map[int]map[int]bool)
		touchedStudentsForAssignment := map[int]bool{}

		studentRows, studentErr := db.DB.Query(`
			SELECT s.id, COALESCE(s.learning_mode_id, 2)
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ? AND csta.teacher_id = ? AND csta.status = 1
		`, assignment.CourseID, assignment.TeacherID)
		if studentErr != nil {
			continue
		}

		for studentRows.Next() {
			var studentID int
			var learningModeID int
			if scanErr := studentRows.Scan(&studentID, &learningModeID); scanErr != nil {
				continue
			}

			expectedComponents, exists := expectedByMode[learningModeID]
			if !exists {
				expectedComponents, _ = getExpectedComponents(courseTypeID, learningModeID, selectedWindow.ComponentIDs)
				expectedByMode[learningModeID] = expectedComponents
			}

			for componentID := range expectedComponents {
				if absentByStudent[studentID][componentID] {
					absentSkipped++
					continue
				}

				meta, cached := componentCache[componentID]
				if !cached {
					var maxMarks float64
					var conversionMarks float64
					metaErr := db.DB.QueryRow(`
						SELECT COALESCE(max_marks, 0), COALESCE(conversion_marks, 0)
						FROM mark_category_types
						WHERE id = ?
					`, componentID).Scan(&maxMarks, &conversionMarks)
					if metaErr != nil {
						continue
					}
					meta = componentMeta{MaxMarks: maxMarks, ConversionMarks: conversionMarks}
					componentCache[componentID] = meta
				}

				if meta.MaxMarks <= 0 {
					continue
				}

				upperBound := int(math.Floor(meta.MaxMarks))
				if upperBound < 0 {
					upperBound = 0
				}
				obtainedMarks := float64(randSource.Intn(upperBound + 1))
				convertedMarks := 0.0
				if meta.MaxMarks > 0 {
					convertedMarks = (obtainedMarks / meta.MaxMarks) * meta.ConversionMarks
				}

				_, execErr := db.DB.Exec(`
					INSERT INTO student_marks
					(student_id, course_id, faculty_id, assessment_component_id, obtained_marks, converted_marks, status)
					VALUES (?, ?, ?, ?, ?, ?, 1)
					ON DUPLICATE KEY UPDATE
					obtained_marks = VALUES(obtained_marks),
					converted_marks = VALUES(converted_marks),
					status = 1
				`, studentID, assignment.CourseID, normalizedFacultyID, componentID, obtainedMarks, convertedMarks)
				if execErr != nil {
					continue
				}

				entriesUpserted++
				touchedStudentsForAssignment[studentID] = true
			}
		}

		studentRows.Close()
		studentsTouched += len(touchedStudentsForAssignment)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"window_id":           req.WindowID,
		"semester":            req.Semester,
		"department_id":       req.DepartmentID,
		"assignments_matched": assignmentsMatched,
		"students_touched":    studentsTouched,
		"entries_upserted":    entriesUpserted,
		"absent_skipped":      absentSkipped,
		"message":             "Random marks filled for matching faculty assignments in selected window",
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

	aliases := getTeacherIdentifierAliases(teacherID)
	if len(aliases) == 0 {
		aliases = []string{teacherID}
	}
	log.Printf("[GetTeacherEnteredStudents] teacher_id=%s aliases=%v course_id=%d", teacherID, aliases, courseID)

	teacherPlaceholders := make([]string, len(aliases))
	teacherArgs := make([]interface{}, 0, len(aliases)+1)
	for index, alias := range aliases {
		teacherPlaceholders[index] = "?"
		teacherArgs = append(teacherArgs, alias)
	}
	teacherArgs = append(teacherArgs, courseID)

	allowedByWindow := map[int]bool{}
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		if windowID, convErr := strconv.Atoi(windowIDStr); convErr == nil && windowID > 0 {
			rows, compErr := db.DB.Query(`SELECT assessment_component_id FROM mark_entry_window_components WHERE window_id = ?`, windowID)
			if compErr == nil {
				for rows.Next() {
					var componentID int
					if scanErr := rows.Scan(&componentID); scanErr == nil {
						allowedByWindow[componentID] = true
					}
				}
				rows.Close()
			}
		}
	}

	studentQuery := fmt.Sprintf(`
		SELECT
			DISTINCT
			s.id,
			COALESCE(s.enrollment_no, ''),
			s.student_name
		FROM course_student_teacher_allocation csta
		JOIN students s ON s.id = csta.student_id
		WHERE csta.teacher_id IN (%s) AND csta.course_id = ? AND csta.status = 1
		ORDER BY s.student_name
	`, strings.Join(teacherPlaceholders, ","))

	rows, err := db.DB.Query(studentQuery, teacherArgs...)
	if err != nil {
		log.Printf("Error fetching teacher entered students: %v", err)
		http.Error(w, "failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	placeholders := make([]string, len(aliases))
	args := make([]interface{}, 0, len(aliases)+1)
	args = append(args, courseID)
	for index, alias := range aliases {
		placeholders[index] = "?"
		args = append(args, strings.ToLower(strings.TrimSpace(alias)))
	}

	marksQuery := fmt.Sprintf(`
		SELECT sm.student_id, sm.assessment_component_id, COALESCE(mct.name, CONCAT('Component ', sm.assessment_component_id)), COALESCE(sm.obtained_marks, 0)
		FROM student_marks sm
		LEFT JOIN mark_category_types mct ON mct.id = sm.assessment_component_id
		WHERE sm.course_id = ?
		  AND (
			LOWER(TRIM(COALESCE(sm.faculty_id, ''))) IN (%s)
			OR TRIM(COALESCE(sm.faculty_id, '')) = ''
		  )
	`, strings.Join(placeholders, ","))
	markRows, markErr := db.DB.Query(marksQuery, args...)
	if markErr != nil {
		log.Printf("Error fetching marks for drill-down: %v", markErr)
		http.Error(w, "failed to fetch mark components", http.StatusInternalServerError)
		return
	}
	defer markRows.Close()

	componentMapByStudent := map[int][]map[string]interface{}{}
	totalByStudent := map[int]float64{}
	markRowCount := 0
	for markRows.Next() {
		var studentID int
		var componentID int
		var componentName string
		var obtained float64
		if scanErr := markRows.Scan(&studentID, &componentID, &componentName, &obtained); scanErr != nil {
			continue
		}
		allowedInWindow := true
		if len(allowedByWindow) > 0 {
			allowedInWindow = allowedByWindow[componentID]
		}

		componentMapByStudent[studentID] = append(componentMapByStudent[studentID], map[string]interface{}{
			"assessment_component_id":   componentID,
			"assessment_component_name": componentName,
			"obtained_marks":            obtained,
			"allowed_in_window":         allowedInWindow,
		})
		totalByStudent[studentID] += obtained
		markRowCount++
	}
	log.Printf("[GetTeacherEnteredStudents] matched mark rows=%d for teacher_id=%s course_id=%d", markRowCount, teacherID, courseID)

	items := make([]map[string]interface{}, 0)
	seenStudents := make(map[int]bool)
	for rows.Next() {
		var studentID int
		var enrollmentNo string
		var studentName string
		if scanErr := rows.Scan(&studentID, &enrollmentNo, &studentName); scanErr != nil {
			continue
		}
		if seenStudents[studentID] {
			continue
		}
		seenStudents[studentID] = true
		components := componentMapByStudent[studentID]
		hasEntries := len(components) > 0
		items = append(items, map[string]interface{}{
			"student_id":     studentID,
			"enrollment_no":  enrollmentNo,
			"student_name":   studentName,
			"total_marks":    totalByStudent[studentID],
			"has_mark_entry": hasEntries,
			"components":     components,
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

func buildResultAnalysis(departmentID int, semester int, window *selectedWindowContext, learningModeID *int, examType string) ([]resultAnalysisRow, error) {
	assignments, err := getAssignmentsForWindowMonitor(semester, &departmentID)
	if err != nil {
		return nil, err
	}

	courseMap := make(map[int]resultAnalysisRow)
	courseAliasMap := make(map[int]map[string]bool)
	for _, assignment := range assignments {
		if !doesWindowMatchAssignment(window, assignment) {
			continue
		}
		if _, exists := courseMap[assignment.CourseID]; !exists {
			courseMap[assignment.CourseID] = resultAnalysisRow{
				CourseID:   assignment.CourseID,
				CourseCode: assignment.CourseCode,
				CourseName: assignment.CourseName,
				Ranges:     map[string]int{"0-10": 0, "11-20": 0, "21-24": 0, "25-30": 0, "31-40": 0, "41-50": 0},
			}
			courseAliasMap[assignment.CourseID] = make(map[string]bool)
		}
		for _, alias := range getTeacherIdentifierAliases(assignment.TeacherID) {
			courseAliasMap[assignment.CourseID][alias] = true
		}
	}

	rows := make([]resultAnalysisRow, 0, len(courseMap))
	for courseID, base := range courseMap {
		courseTypeID, typeErr := getCourseTypeForCourse(courseID)
		if typeErr != nil || courseTypeID == 0 {
			continue
		}

		aliases := make([]string, 0, len(courseAliasMap[courseID]))
		for alias := range courseAliasMap[courseID] {
			aliases = append(aliases, alias)
		}

		studentQuery := `
			SELECT DISTINCT csta.student_id, s.student_name
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ? AND csta.status = 1
		`
		studentArgs := []interface{}{courseID}
		if learningModeID != nil {
			studentQuery += ` AND COALESCE(s.learning_mode_id, 2) = ?`
			studentArgs = append(studentArgs, *learningModeID)
		}

		studentRows, err := db.DB.Query(studentQuery, studentArgs...)
		if err != nil {
			return nil, err
		}

		studentIDs := make([]int, 0)
		studentNameByID := make(map[int]string)
		for studentRows.Next() {
			var studentID int
			var studentName string
			if scanErr := studentRows.Scan(&studentID, &studentName); scanErr != nil {
				continue
			}
			studentIDs = append(studentIDs, studentID)
			studentNameByID[studentID] = studentName
		}
		studentRows.Close()

		studentIDFilter := make(map[int]bool, len(studentIDs))
		for _, studentID := range studentIDs {
			studentIDFilter[studentID] = true
		}

		componentQuery := `
			SELECT id, COALESCE(name, CONCAT('Component ', id))
			FROM mark_category_types
			WHERE course_type_id = ? AND status = 1
		`
		componentArgs := []interface{}{courseTypeID}
		if learningModeID != nil {
			componentQuery += ` AND learning_mode_id = ?`
			componentArgs = append(componentArgs, *learningModeID)
		}
		componentQuery += ` ORDER BY position ASC, id ASC`

		componentRows, componentErr := db.DB.Query(componentQuery, componentArgs...)
		if componentErr != nil {
			return nil, componentErr
		}

		expectedComponents := map[int]string{}
		for componentRows.Next() {
			var componentID int
			var componentName string
			if scanErr := componentRows.Scan(&componentID, &componentName); scanErr != nil {
				continue
			}
			if !doesComponentMatchExamType(componentName, examType) {
				continue
			}
			expectedComponents[componentID] = componentName
		}
		componentRows.Close()

		enteredMarksByStudent := map[int]float64{}
		appearedByStudent := map[int]bool{}
		componentTotalsByStudent := map[int]map[int]float64{}
		componentNames := map[int]string{}
		for componentID, componentName := range expectedComponents {
			componentTotalsByStudent[componentID] = map[int]float64{}
			componentNames[componentID] = componentName
		}
		if len(aliases) > 0 {
			placeholders := make([]string, len(aliases))
			args := make([]interface{}, 0, len(aliases)+1)
			args = append(args, courseID)
			for index, alias := range aliases {
				placeholders[index] = "?"
				args = append(args, alias)
			}

			marksQuery := fmt.Sprintf(`
				SELECT sm.student_id, sm.assessment_component_id, COALESCE(mct.name, CONCAT('Component ', sm.assessment_component_id)), COALESCE(sm.obtained_marks, 0)
				FROM student_marks sm
				LEFT JOIN mark_category_types mct ON mct.id = sm.assessment_component_id
				WHERE course_id = ? AND faculty_id IN (%s)
			`, strings.Join(placeholders, ","))
			markRows, markErr := db.DB.Query(marksQuery, args...)
			if markErr == nil {
				for markRows.Next() {
					var studentID int
					var componentID int
					var componentName string
					var marks float64
					if scanErr := markRows.Scan(&studentID, &componentID, &componentName, &marks); scanErr != nil {
						continue
					}
					if !doesComponentMatchExamType(componentName, examType) {
						continue
					}
					if _, ok := expectedComponents[componentID]; !ok {
						continue
					}
					if !studentIDFilter[studentID] {
						continue
					}
					enteredMarksByStudent[studentID] += marks
					appearedByStudent[studentID] = true
					if _, ok := componentTotalsByStudent[componentID]; !ok {
						componentTotalsByStudent[componentID] = map[int]float64{}
					}
					componentTotalsByStudent[componentID][studentID] += marks
					if strings.TrimSpace(componentName) != "" {
						componentNames[componentID] = componentName
					}
				}
				markRows.Close()
			}
		}

		studentTotals := make([]studentMarkTotal, 0, len(studentIDs))
		for _, studentID := range studentIDs {
			studentTotals = append(studentTotals, studentMarkTotal{
				StudentID:   studentID,
				StudentName: studentNameByID[studentID],
				TotalMarks:  enteredMarksByStudent[studentID],
				Appeared:    appearedByStudent[studentID],
			})
		}

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

		components := make([]resultComponent, 0, len(componentTotalsByStudent))
		for componentID, marksByStudent := range componentTotalsByStudent {
			component := resultComponent{
				ComponentID:   componentID,
				ComponentName: componentNames[componentID],
				Registered:    analysis.Registered,
				Appeared:      len(marksByStudent),
				Absent:        analysis.Registered - len(marksByStudent),
			}

			maxComponent := 0.0
			minComponent := 0.0
			totalComponent := 0.0
			count := 0
			for _, mark := range marksByStudent {
				totalComponent += mark
				if count == 0 || mark > maxComponent {
					maxComponent = mark
				}
				if count == 0 || mark < minComponent {
					minComponent = mark
				}
				count++
			}

			component.TotalMarks = totalComponent
			component.MaximumMark = maxComponent
			component.MinimumMark = minComponent
			if count > 0 {
				component.AverageMark = totalComponent / float64(count)
			}

			components = append(components, component)
		}
		sort.Slice(components, func(i, j int) bool {
			return components[i].ComponentID < components[j].ComponentID
		})
		analysis.Components = components

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

	examType := strings.TrimSpace(r.URL.Query().Get("exam_type"))

	learningModeStr := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("learning_mode")))
	var learningModeID *int
	if learningModeStr != "" {
		mode := 0
		switch learningModeStr {
		case "PBL", "2":
			mode = 2
		case "UAL", "1":
			mode = 1
		default:
			http.Error(w, "invalid learning_mode", http.StatusBadRequest)
			return
		}
		learningModeID = &mode
	}

	departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id"))
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
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
		if role == "admin" || role == "coe" {
			http.Error(w, "department_id is required for admin/coe", http.StatusBadRequest)
			return
		}
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

	var selectedWindow *selectedWindowContext
	if windowIDStr := strings.TrimSpace(r.URL.Query().Get("window_id")); windowIDStr != "" {
		windowID, convErr := strconv.Atoi(windowIDStr)
		if convErr != nil || windowID <= 0 {
			http.Error(w, "invalid window_id", http.StatusBadRequest)
			return
		}
		selectedWindow, err = getSelectedWindowContext(windowID)
		if err != nil {
			http.Error(w, "window not found", http.StatusNotFound)
			return
		}
	}

	rows, err := buildResultAnalysis(departmentID, semester, selectedWindow, learningModeID, examType)
	if err != nil {
		log.Printf("Error generating result analysis: %v", err)
		http.Error(w, "failed to generate result analysis", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"department_id":   departmentID,
		"department_name": departmentName,
		"semester":        semester,
		"exam_type":       examType,
		"learning_mode":   learningModeStr,
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

	records := [][]string{{"Course Code", "Course Name", "Teacher ID", "Teacher Name", "Assigned Students", "Completed Students", "Completion %", "Window Name", "Window Status", "Late", "Extension Status"}}
	for _, row := range overviewRows {
		windowName := ""
		if row.WindowName != nil {
			windowName = strings.TrimSpace(*row.WindowName)
		}
		records = append(records, []string{
			row.CourseCode,
			row.CourseName,
			row.TeacherID,
			row.TeacherName,
			strconv.Itoa(row.AssignedStudents),
			strconv.Itoa(row.CompletedStudents),
			fmt.Sprintf("%.2f", row.CompletionPercent),
			windowName,
			row.WindowStatus,
			strconv.FormatBool(row.Late),
			row.ExtensionStatus,
		})
	}

	return records, nil
}

type detailedMarkEntryRow struct {
	DepartmentName string
	Semester       int
	WindowName     string
	TeacherID      string
	TeacherName    string
	CourseCode     string
	CourseName     string
	StudentID      int
	EnrollmentNo   string
	StudentName    string
	ComponentID    int
	ComponentName  string
	Marks          float64
}

func buildDetailedMarkEntryRows(departmentID int, semester int) ([]detailedMarkEntryRow, error) {
	overviewRows, err := buildMarkEntryOverview(departmentID, semester)
	if err != nil {
		return nil, err
	}

	rows := make([]detailedMarkEntryRow, 0)
	for _, overview := range overviewRows {
		aliases := getTeacherIdentifierAliases(overview.TeacherID)
		if len(aliases) == 0 {
			continue
		}

		aliasPlaceholders := make([]string, len(aliases))
		aliasArgs := make([]interface{}, 0, len(aliases)+1)
		aliasArgs = append(aliasArgs, overview.CourseID)
		for index, alias := range aliases {
			aliasPlaceholders[index] = "?"
			aliasArgs = append(aliasArgs, alias)
		}

		studentQuery := fmt.Sprintf(`
			SELECT DISTINCT csta.student_id, COALESCE(s.enrollment_no, ''), s.student_name
			FROM course_student_teacher_allocation csta
			JOIN students s ON s.id = csta.student_id
			WHERE csta.course_id = ? AND csta.status = 1 AND csta.teacher_id IN (%s)
		`, strings.Join(aliasPlaceholders, ","))

		studentRows, studentErr := db.DB.Query(studentQuery, aliasArgs...)
		if studentErr != nil {
			continue
		}

		type studentMeta struct {
			EnrollmentNo string
			StudentName  string
		}
		studentInfo := make(map[int]studentMeta)
		studentIDs := make([]int, 0)
		for studentRows.Next() {
			var studentID int
			var enrollmentNo string
			var studentName string
			if scanErr := studentRows.Scan(&studentID, &enrollmentNo, &studentName); scanErr != nil {
				continue
			}
			studentInfo[studentID] = studentMeta{EnrollmentNo: enrollmentNo, StudentName: studentName}
			studentIDs = append(studentIDs, studentID)
		}
		studentRows.Close()

		if len(studentIDs) == 0 {
			continue
		}

		studentPlaceholders := make([]string, len(studentIDs))
		markArgs := make([]interface{}, 0, 1+len(aliases)+len(studentIDs)+len(overview.AllowedComponentIDs))
		markArgs = append(markArgs, overview.CourseID)
		for _, alias := range aliases {
			markArgs = append(markArgs, alias)
		}
		for index, studentID := range studentIDs {
			studentPlaceholders[index] = "?"
			markArgs = append(markArgs, studentID)
		}

		marksQuery := fmt.Sprintf(`
			SELECT sm.student_id, sm.assessment_component_id, COALESCE(mct.name, CONCAT('Component ', sm.assessment_component_id)), COALESCE(sm.obtained_marks, 0)
			FROM student_marks sm
			LEFT JOIN mark_category_types mct ON mct.id = sm.assessment_component_id
			WHERE sm.course_id = ? AND sm.faculty_id IN (%s) AND sm.student_id IN (%s)
		`, strings.Join(aliasPlaceholders, ","), strings.Join(studentPlaceholders, ","))

		if len(overview.AllowedComponentIDs) > 0 {
			componentPlaceholders := make([]string, len(overview.AllowedComponentIDs))
			for index, componentID := range overview.AllowedComponentIDs {
				componentPlaceholders[index] = "?"
				markArgs = append(markArgs, componentID)
			}
			marksQuery += fmt.Sprintf(" AND sm.assessment_component_id IN (%s)", strings.Join(componentPlaceholders, ","))
		}

		marksQuery += " ORDER BY sm.student_id, sm.assessment_component_id"
		markRows, markErr := db.DB.Query(marksQuery, markArgs...)
		if markErr != nil {
			continue
		}

		for markRows.Next() {
			var studentID int
			var componentID int
			var componentName string
			var marks float64
			if scanErr := markRows.Scan(&studentID, &componentID, &componentName, &marks); scanErr != nil {
				continue
			}

			meta, exists := studentInfo[studentID]
			if !exists {
				continue
			}

			rows = append(rows, detailedMarkEntryRow{
				DepartmentName: overview.DepartmentName,
				Semester:       overview.Semester,
				WindowName: func() string {
					if overview.WindowName == nil {
						return ""
					}
					return strings.TrimSpace(*overview.WindowName)
				}(),
				TeacherID:     overview.TeacherID,
				TeacherName:   overview.TeacherName,
				CourseCode:    overview.CourseCode,
				CourseName:    overview.CourseName,
				StudentID:     studentID,
				EnrollmentNo:  meta.EnrollmentNo,
				StudentName:   meta.StudentName,
				ComponentID:   componentID,
				ComponentName: componentName,
				Marks:         marks,
			})
		}
		markRows.Close()
	}

	return rows, nil
}

func exportTeacherWiseMarkRows(departmentID int, semester int) ([][]string, error) {
	detailedRows, err := buildDetailedMarkEntryRows(departmentID, semester)
	if err != nil {
		return nil, err
	}
	return buildPivotedDetailedRecords(detailedRows, true), nil
}

func buildDetailedMarkEntryRowsAllDepartments(semester int) ([]detailedMarkEntryRow, error) {
	assignments, err := getAssignmentsForWindowMonitor(semester, nil)
	if err != nil {
		return nil, err
	}

	departmentSet := map[int]bool{}
	departmentIDs := make([]int, 0)
	for _, assignment := range assignments {
		if assignment.DepartmentID <= 0 {
			continue
		}
		if !departmentSet[assignment.DepartmentID] {
			departmentSet[assignment.DepartmentID] = true
			departmentIDs = append(departmentIDs, assignment.DepartmentID)
		}
	}

	sort.Ints(departmentIDs)
	allRows := make([]detailedMarkEntryRow, 0)
	for _, departmentID := range departmentIDs {
		rows, rowsErr := buildDetailedMarkEntryRows(departmentID, semester)
		if rowsErr != nil {
			continue
		}
		allRows = append(allRows, rows...)
	}

	return allRows, nil
}

func exportTeacherWiseMarkRowsAllDepartments(semester int) ([][]string, error) {
	detailedRows, err := buildDetailedMarkEntryRowsAllDepartments(semester)
	if err != nil {
		return nil, err
	}
	return buildPivotedDetailedRecords(detailedRows, true), nil
}

func exportCourseWiseMarkRows(departmentID int, semester int) ([][]string, error) {
	detailedRows, err := buildDetailedMarkEntryRows(departmentID, semester)
	if err != nil {
		return nil, err
	}
	return buildPivotedDetailedRecords(detailedRows, false), nil
}

func exportCourseWiseMarkRowsAllDepartments(semester int) ([][]string, error) {
	detailedRows, err := buildDetailedMarkEntryRowsAllDepartments(semester)
	if err != nil {
		return nil, err
	}
	return buildPivotedDetailedRecords(detailedRows, false), nil
}

type pivotDetailedRow struct {
	DepartmentName string
	Semester       int
	WindowName     string
	FacultyCode    string
	FacultyName    string
	CourseCode     string
	CourseName     string
	StudentID      int
	EnrollmentNo   string
	StudentName    string
	ComponentMarks map[string]float64
}

func splitMainAndSubComponent(componentName string) (string, string) {
	trimmed := strings.TrimSpace(componentName)
	if trimmed == "" {
		return "Component", ""
	}
	parts := strings.SplitN(trimmed, "->", 2)
	main := strings.TrimSpace(parts[0])
	if main == "" {
		main = "Component"
	}
	if len(parts) < 2 {
		return main, ""
	}
	return main, strings.TrimSpace(parts[1])
}

func subComponentLess(a string, b string) bool {
	an := strings.ToUpper(strings.TrimSpace(a))
	bn := strings.ToUpper(strings.TrimSpace(b))
	if strings.HasPrefix(an, "CO") && strings.HasPrefix(bn, "CO") {
		av, aErr := strconv.Atoi(strings.TrimSpace(an[2:]))
		bv, bErr := strconv.Atoi(strings.TrimSpace(bn[2:]))
		if aErr == nil && bErr == nil {
			return av < bv
		}
	}
	return an < bn
}

func buildPivotedDetailedRecords(detailedRows []detailedMarkEntryRow, teacherMode bool) [][]string {
	pivotByKey := make(map[string]*pivotDetailedRow)
	mainToSubs := make(map[string]map[string]bool)

	for _, row := range detailedRows {
		mainTest, sub := splitMainAndSubComponent(row.ComponentName)
		if _, exists := mainToSubs[mainTest]; !exists {
			mainToSubs[mainTest] = map[string]bool{}
		}
		if sub != "" {
			mainToSubs[mainTest][sub] = true
		}

		key := strings.Join([]string{
			row.DepartmentName,
			strconv.Itoa(row.Semester),
			row.WindowName,
			row.TeacherID,
			row.TeacherName,
			row.CourseCode,
			row.CourseName,
			strconv.Itoa(row.StudentID),
			row.EnrollmentNo,
			row.StudentName,
		}, "|")

		pivot, exists := pivotByKey[key]
		if !exists {
			pivot = &pivotDetailedRow{
				DepartmentName: row.DepartmentName,
				Semester:       row.Semester,
				WindowName:     row.WindowName,
				FacultyCode:    row.TeacherID,
				FacultyName:    row.TeacherName,
				CourseCode:     row.CourseCode,
				CourseName:     row.CourseName,
				StudentID:      row.StudentID,
				EnrollmentNo:   row.EnrollmentNo,
				StudentName:    row.StudentName,
				ComponentMarks: map[string]float64{},
			}
			pivotByKey[key] = pivot
		}

		columnName := mainTest
		if sub != "" {
			columnName = fmt.Sprintf("%s - %s", mainTest, sub)
		}
		pivot.ComponentMarks[columnName] += row.Marks

		if len(mainToSubs[mainTest]) > 0 {
			totalColumn := fmt.Sprintf("%s - Total", mainTest)
			pivot.ComponentMarks[totalColumn] += row.Marks
		}
	}

	orderedMains := make([]string, 0, len(mainToSubs))
	for main := range mainToSubs {
		orderedMains = append(orderedMains, main)
	}
	sort.Strings(orderedMains)

	orderedComponentColumns := make([]string, 0)
	for _, main := range orderedMains {
		subs := make([]string, 0, len(mainToSubs[main]))
		for sub := range mainToSubs[main] {
			subs = append(subs, sub)
		}
		sort.Slice(subs, func(i, j int) bool { return subComponentLess(subs[i], subs[j]) })

		if len(subs) == 0 {
			orderedComponentColumns = append(orderedComponentColumns, main)
			continue
		}

		for _, sub := range subs {
			orderedComponentColumns = append(orderedComponentColumns, fmt.Sprintf("%s - %s", main, sub))
		}
		orderedComponentColumns = append(orderedComponentColumns, fmt.Sprintf("%s - Total", main))
	}

	orderedRows := make([]*pivotDetailedRow, 0, len(pivotByKey))
	for _, row := range pivotByKey {
		orderedRows = append(orderedRows, row)
	}
	sort.Slice(orderedRows, func(i, j int) bool {
		a := orderedRows[i]
		b := orderedRows[j]
		if teacherMode {
			if a.FacultyCode != b.FacultyCode {
				return a.FacultyCode < b.FacultyCode
			}
			if a.CourseCode != b.CourseCode {
				return a.CourseCode < b.CourseCode
			}
		} else {
			if a.CourseCode != b.CourseCode {
				return a.CourseCode < b.CourseCode
			}
			if a.FacultyCode != b.FacultyCode {
				return a.FacultyCode < b.FacultyCode
			}
		}
		if a.EnrollmentNo != b.EnrollmentNo {
			return a.EnrollmentNo < b.EnrollmentNo
		}
		return a.StudentID < b.StudentID
	})

	headers := []string{"Department", "Semester", "Window Name"}
	if teacherMode {
		headers = append(headers, "Faculty Code", "Faculty", "Course Code", "Course Name")
	} else {
		headers = append(headers, "Course Code", "Course Name", "Faculty Code", "Faculty")
	}
	headers = append(headers, "Student ID", "Enrollment No", "Student Name")
	headers = append(headers, orderedComponentColumns...)

	records := make([][]string, 0, len(orderedRows)+1)
	records = append(records, headers)

	for _, row := range orderedRows {
		base := []string{row.DepartmentName, strconv.Itoa(row.Semester), row.WindowName}
		if teacherMode {
			base = append(base, row.FacultyCode, row.FacultyName, row.CourseCode, row.CourseName)
		} else {
			base = append(base, row.CourseCode, row.CourseName, row.FacultyCode, row.FacultyName)
		}
		base = append(base, strconv.Itoa(row.StudentID), row.EnrollmentNo, row.StudentName)
		for _, componentColumn := range orderedComponentColumns {
			if mark, exists := row.ComponentMarks[componentColumn]; exists {
				base = append(base, fmt.Sprintf("%.2f", mark))
			} else {
				base = append(base, "")
			}
		}
		records = append(records, base)
	}

	return records
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

func sanitizeSheetName(name string, fallback string, used map[string]int) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = fallback
	}

	replacer := strings.NewReplacer("\\", " ", "/", " ", "?", "", "*", "", "[", "(", "]", ")", ":", " - ")
	clean = strings.TrimSpace(replacer.Replace(clean))
	if clean == "" {
		clean = fallback
	}

	runes := []rune(clean)
	if len(runes) > 31 {
		clean = strings.TrimSpace(string(runes[:31]))
	}
	if clean == "" {
		clean = fallback
	}

	base := clean
	if count, exists := used[base]; exists {
		for {
			count++
			suffix := fmt.Sprintf(" (%d)", count)
			maxBaseRunes := 31 - len([]rune(suffix))
			if maxBaseRunes < 1 {
				maxBaseRunes = 1
			}
			baseRunes := []rune(base)
			nameBase := base
			if len(baseRunes) > maxBaseRunes {
				nameBase = strings.TrimSpace(string(baseRunes[:maxBaseRunes]))
			}
			candidate := strings.TrimSpace(nameBase + suffix)
			if _, taken := used[candidate]; !taken {
				used[base] = count
				used[candidate] = 1
				return candidate
			}
		}
	}

	used[base] = 1
	return base
}

func writeGroupedXLSXAttachment(w http.ResponseWriter, filename string, originalRecords [][]string, selectedRecords [][]string, groupByHeader string, suffixHeaders []string) error {
	if len(originalRecords) == 0 || len(selectedRecords) == 0 {
		return writeXLSXAttachment(w, filename, selectedRecords)
	}

	groupByIndex := -1
	for index, header := range originalRecords[0] {
		if strings.EqualFold(strings.TrimSpace(header), strings.TrimSpace(groupByHeader)) {
			groupByIndex = index
			break
		}
	}
	if groupByIndex < 0 {
		return writeXLSXAttachment(w, filename, selectedRecords)
	}

	suffixIndexes := make([]int, 0, len(suffixHeaders))
	for _, headerName := range suffixHeaders {
		suffixIndex := -1
		for index, header := range originalRecords[0] {
			if strings.EqualFold(strings.TrimSpace(header), strings.TrimSpace(headerName)) {
				suffixIndex = index
				break
			}
		}
		if suffixIndex >= 0 {
			suffixIndexes = append(suffixIndexes, suffixIndex)
		}
	}

	file := excelize.NewFile()
	if err := file.DeleteSheet("Sheet1"); err != nil {
		// ignore
	}

	groups := make(map[string][][]string)
	groupSheetLabel := make(map[string]string)
	order := make([]string, 0)
	maxRows := len(originalRecords)
	if len(selectedRecords) < maxRows {
		maxRows = len(selectedRecords)
	}
	for rowIndex := 1; rowIndex < maxRows; rowIndex++ {
		groupValue := "Unassigned"
		if groupByIndex < len(originalRecords[rowIndex]) {
			candidate := strings.TrimSpace(originalRecords[rowIndex][groupByIndex])
			if candidate != "" {
				groupValue = candidate
			}
		}
		if _, exists := groups[groupValue]; !exists {
			order = append(order, groupValue)
			groups[groupValue] = make([][]string, 0)
			identifierParts := []string{}
			for _, suffixIndex := range suffixIndexes {
				if suffixIndex >= 0 && suffixIndex < len(originalRecords[rowIndex]) {
					suffixValue := strings.TrimSpace(originalRecords[rowIndex][suffixIndex])
					if suffixValue != "" {
						duplicate := false
						for _, existing := range identifierParts {
							if strings.EqualFold(existing, suffixValue) {
								duplicate = true
								break
							}
						}
						if !duplicate {
							identifierParts = append(identifierParts, suffixValue)
						}
					}
				}
			}
			labelParts := make([]string, 0, len(identifierParts)+1)
			labelParts = append(labelParts, identifierParts...)
			if strings.TrimSpace(groupValue) != "" {
				labelParts = append(labelParts, groupValue)
			}
			groupSheetLabel[groupValue] = strings.Join(labelParts, " - ")
		}
		groups[groupValue] = append(groups[groupValue], selectedRecords[rowIndex])
	}

	usedNames := map[string]int{}
	for idx, groupValue := range order {
		label := groupSheetLabel[groupValue]
		if strings.TrimSpace(label) == "" {
			label = groupValue
		}
		sheetName := sanitizeSheetName(label, fmt.Sprintf("Sheet %d", idx+1), usedNames)
		sheetIndex, err := file.NewSheet(sheetName)
		if err != nil {
			return err
		}
		if idx == 0 {
			file.SetActiveSheet(sheetIndex)
		}

		for colIdx, value := range selectedRecords[0] {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
			_ = file.SetCellValue(sheetName, cell, value)
		}

		for dataRowIndex, row := range groups[groupValue] {
			for colIdx, value := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, dataRowIndex+2)
				_ = file.SetCellValue(sheetName, cell, value)
			}
		}
	}

	if len(order) == 0 {
		sheetIndex, err := file.NewSheet("Report")
		if err != nil {
			return err
		}
		file.SetActiveSheet(sheetIndex)
		for rowIdx, row := range selectedRecords {
			for colIdx, value := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				_ = file.SetCellValue("Report", cell, value)
			}
		}
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

func writeDepartmentSheetsXLSXAttachment(w http.ResponseWriter, filename string, originalRecords [][]string, reportType string, fieldsParam string) error {
	if len(originalRecords) == 0 {
		return writeXLSXAttachment(w, filename, originalRecords)
	}

	selectedRecords := applySelectedExportFields(originalRecords, reportType, fieldsParam)
	if len(selectedRecords) == 0 {
		selectedRecords = originalRecords
	}

	departmentIndex := -1
	for index, header := range originalRecords[0] {
		if strings.EqualFold(strings.TrimSpace(header), "Department") {
			departmentIndex = index
			break
		}
	}
	if departmentIndex < 0 {
		return writeXLSXAttachment(w, filename, selectedRecords)
	}

	groups := map[string][][]string{}
	order := make([]string, 0)
	maxRows := len(originalRecords)
	if len(selectedRecords) < maxRows {
		maxRows = len(selectedRecords)
	}
	for rowIndex := 1; rowIndex < maxRows; rowIndex++ {
		row := originalRecords[rowIndex]
		departmentName := "Unmapped"
		if departmentIndex >= 0 && departmentIndex < len(row) {
			candidate := strings.TrimSpace(row[departmentIndex])
			if candidate != "" {
				departmentName = candidate
			}
		}
		if _, exists := groups[departmentName]; !exists {
			groups[departmentName] = make([][]string, 0)
			order = append(order, departmentName)
		}
		groups[departmentName] = append(groups[departmentName], selectedRecords[rowIndex])
	}

	sort.SliceStable(order, func(i, j int) bool {
		return strings.ToLower(order[i]) < strings.ToLower(order[j])
	})

	file := excelize.NewFile()
	if err := file.DeleteSheet("Sheet1"); err != nil {
		// ignore
	}

	usedNames := map[string]int{}
	for index, departmentName := range order {
		sheetName := sanitizeSheetName(departmentName, fmt.Sprintf("Dept %d", index+1), usedNames)
		sheetIndex, err := file.NewSheet(sheetName)
		if err != nil {
			return err
		}
		if index == 0 {
			file.SetActiveSheet(sheetIndex)
		}

		selectedRows := make([][]string, 0, len(groups[departmentName])+1)
		selectedRows = append(selectedRows, selectedRecords[0])
		selectedRows = append(selectedRows, groups[departmentName]...)

		for rowIdx, row := range selectedRows {
			for colIdx, value := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				_ = file.SetCellValue(sheetName, cell, value)
			}
		}
	}

	if len(order) == 0 {
		sheetIndex, err := file.NewSheet("Report")
		if err != nil {
			return err
		}
		file.SetActiveSheet(sheetIndex)
		for rowIdx, row := range selectedRecords {
			for colIdx, value := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				_ = file.SetCellValue("Report", cell, value)
			}
		}
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

func getExportFieldHeaders(reportType string) map[string]string {
	switch reportType {
	case "teacher":
		return map[string]string{
			"student_name":  "Student Name",
			"enrollment_no": "Enrollment No",
			"course_code":   "Course Code",
			"course_name":   "Course Name",
			"components":    "Components",
			"faculty_code":  "Faculty Code",
			"teacher_name":  "Faculty",
			"window_name":   "Window Name",
			"semester":      "Semester",
			"department":    "Department",
		}
	case "course":
		return map[string]string{
			"student_name":  "Student Name",
			"enrollment_no": "Enrollment No",
			"course_code":   "Course Code",
			"course_name":   "Course Name",
			"components":    "Components",
			"faculty_code":  "Faculty Code",
			"teacher_name":  "Faculty",
			"window_name":   "Window Name",
			"semester":      "Semester",
			"department":    "Department",
		}
	case "course_summary":
		return map[string]string{
			"course_code":        "Course Code",
			"course_name":        "Course Name",
			"teacher_name":       "Teacher Name",
			"assigned_students":  "Assigned Students",
			"completed_students": "Completed Students",
			"completion_percent": "Completion %",
			"window_name":        "Window Name",
			"window_status":      "Window Status",
			"late":               "Late",
			"extension_status":   "Extension Status",
		}
	default:
		return map[string]string{}
	}
}

func applySelectedExportFields(records [][]string, reportType string, fieldsParam string) [][]string {
	if len(records) == 0 || strings.TrimSpace(fieldsParam) == "" {
		return records
	}

	fieldHeaderMap := getExportFieldHeaders(reportType)
	if len(fieldHeaderMap) == 0 {
		return records
	}

	headerRow := records[0]
	headerIndex := make(map[string]int, len(headerRow))
	for index, header := range headerRow {
		headerIndex[strings.TrimSpace(header)] = index
	}

	selectedIndexes := make([]int, 0)
	selectedHeaders := make([]string, 0)
	seenHeader := map[string]bool{}
	includeComponents := false
	for _, rawField := range strings.Split(fieldsParam, ",") {
		field := strings.ToLower(strings.TrimSpace(rawField))
		if field == "" {
			continue
		}
		if field == "components" {
			includeComponents = true
			continue
		}
		header, exists := fieldHeaderMap[field]
		if !exists || seenHeader[header] {
			continue
		}
		columnIndex, found := headerIndex[header]
		if !found {
			continue
		}
		seenHeader[header] = true
		selectedIndexes = append(selectedIndexes, columnIndex)
		selectedHeaders = append(selectedHeaders, header)
	}

	if len(selectedIndexes) == 0 {
		return records
	}

	if includeComponents {
		knownHeaders := map[string]bool{}
		for _, known := range fieldHeaderMap {
			knownHeaders[strings.TrimSpace(known)] = true
		}
		for index, header := range headerRow {
			trimmed := strings.TrimSpace(header)
			if trimmed == "" || knownHeaders[trimmed] {
				continue
			}
			already := false
			for _, selected := range selectedIndexes {
				if selected == index {
					already = true
					break
				}
			}
			if !already {
				selectedIndexes = append(selectedIndexes, index)
				selectedHeaders = append(selectedHeaders, header)
			}
		}
	}

	filtered := make([][]string, 0, len(records))
	filtered = append(filtered, selectedHeaders)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		row := records[rowIndex]
		selectedRow := make([]string, 0, len(selectedIndexes))
		for _, columnIndex := range selectedIndexes {
			if columnIndex >= 0 && columnIndex < len(row) {
				selectedRow = append(selectedRow, row[columnIndex])
			} else {
				selectedRow = append(selectedRow, "")
			}
		}
		filtered = append(filtered, selectedRow)
	}

	return filtered
}

func filterExportRecordsByScope(records [][]string, teacherID string, courseCode string, windowName string) [][]string {
	if len(records) == 0 {
		return records
	}

	teacherID = strings.TrimSpace(strings.ToLower(teacherID))
	courseCode = strings.TrimSpace(strings.ToLower(courseCode))
	windowName = strings.TrimSpace(strings.ToLower(windowName))

	if teacherID == "" && courseCode == "" && windowName == "" {
		return records
	}

	headers := records[0]
	teacherIdx := -1
	courseIdx := -1
	windowIdx := -1
	for index, header := range headers {
		switch strings.TrimSpace(strings.ToLower(header)) {
		case "faculty code":
			teacherIdx = index
		case "course code":
			courseIdx = index
		case "window name":
			windowIdx = index
		}
	}

	filtered := make([][]string, 0, len(records))
	filtered = append(filtered, headers)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		row := records[rowIndex]

		if teacherID != "" {
			if teacherIdx < 0 || teacherIdx >= len(row) || strings.TrimSpace(strings.ToLower(row[teacherIdx])) != teacherID {
				continue
			}
		}

		if courseCode != "" {
			if courseIdx < 0 || courseIdx >= len(row) || strings.TrimSpace(strings.ToLower(row[courseIdx])) != courseCode {
				continue
			}
		}

		if windowName != "" {
			if windowIdx < 0 || windowIdx >= len(row) || strings.TrimSpace(strings.ToLower(row[windowIdx])) != windowName {
				continue
			}
		}

		filtered = append(filtered, row)
	}

	return filtered
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
	departmentIDStr := strings.TrimSpace(r.URL.Query().Get("department_id"))
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
	requestPath := strings.ToLower(strings.TrimSpace(r.URL.Path))
	isAdminEndpoint := strings.Contains(requestPath, "/api/admin/")
	if role == "" && isAdminEndpoint {
		role = "admin"
	}
	semesterStr := strings.TrimSpace(r.URL.Query().Get("semester"))
	reportType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("report_type")))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	fields := strings.TrimSpace(r.URL.Query().Get("fields"))
	teacherIDFilter := strings.TrimSpace(r.URL.Query().Get("teacher_id"))
	courseCodeFilter := strings.TrimSpace(r.URL.Query().Get("course_code"))
	windowNameFilter := strings.TrimSpace(r.URL.Query().Get("window_name"))

	if semesterStr == "" || reportType == "" || format == "" {
		http.Error(w, "semester, report_type and format are required", http.StatusBadRequest)
		return
	}

	semester, err := strconv.Atoi(semesterStr)
	if err != nil || semester <= 0 {
		http.Error(w, "invalid semester", http.StatusBadRequest)
		return
	}

	var departmentID int
	var departmentName string
	allDepartments := false
	if departmentIDStr != "" {
		departmentID, err = strconv.Atoi(departmentIDStr)
		if err != nil || departmentID <= 0 {
			http.Error(w, "invalid department_id", http.StatusBadRequest)
			return
		}
		if scanErr := db.DB.QueryRow(`SELECT department_name FROM departments WHERE id = ?`, departmentID).Scan(&departmentName); scanErr != nil {
			http.Error(w, "department not found", http.StatusBadRequest)
			return
		}
	} else {
		if username == "" {
			if role == "admin" || role == "coe" {
				allDepartments = true
				departmentName = "all_departments"
			} else {
				http.Error(w, "username or department_id is required", http.StatusBadRequest)
				return
			}
		}
		if !allDepartments {
			departmentID, departmentName, err = getDepartmentContextFromUsername(username)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	var records [][]string
	normalizedReportType := reportType
	switch reportType {
	case "teacher", "teacher-wise", "teacher_wise", "student":
		normalizedReportType = "teacher"
		if allDepartments {
			records, err = exportTeacherWiseMarkRowsAllDepartments(semester)
		} else {
			records, err = exportTeacherWiseMarkRows(departmentID, semester)
		}
	case "course", "course-wise", "course_wise":
		normalizedReportType = "course"
		if allDepartments {
			records, err = exportCourseWiseMarkRowsAllDepartments(semester)
		} else {
			records, err = exportCourseWiseMarkRows(departmentID, semester)
		}
	case "course-summary":
		normalizedReportType = "course_summary"
		if allDepartments {
			http.Error(w, "course_summary export requires a specific department", http.StatusBadRequest)
			return
		}
		records, err = exportCourseWiseRows(departmentID, semester)
	default:
		http.Error(w, "report_type must be teacher or course", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Error generating report records: %v", err)
		http.Error(w, "failed to generate report", http.StatusInternalServerError)
		return
	}

	originalRecords := records
	originalRecords = filterExportRecordsByScope(originalRecords, teacherIDFilter, courseCodeFilter, windowNameFilter)
	log.Printf("[DownloadMarkEntryReport] report_type=%s dept_id=%d semester=%d base_rows=%d fields=%q", normalizedReportType, departmentID, semester, len(originalRecords), fields)
	records = applySelectedExportFields(originalRecords, normalizedReportType, fields)
	log.Printf("[DownloadMarkEntryReport] filtered_rows=%d", len(records))

	baseName := fmt.Sprintf("mark_entry_%s_sem%d_dept%s", normalizedReportType, semester, strings.ReplaceAll(strings.ToLower(departmentName), " ", "_"))
	switch format {
	case "csv":
		writeCSVAttachment(w, baseName+".csv", records)
	case "xlsx":
		if allDepartments && (normalizedReportType == "teacher" || normalizedReportType == "course") {
			if writeErr := writeDepartmentSheetsXLSXAttachment(w, baseName+".xlsx", originalRecords, normalizedReportType, fields); writeErr != nil {
				http.Error(w, "failed to generate xlsx", http.StatusInternalServerError)
			}
			return
		}
		if normalizedReportType == "teacher" {
			if writeErr := writeGroupedXLSXAttachment(w, baseName+".xlsx", originalRecords, records, "Faculty", []string{"Course Code", "Faculty Code"}); writeErr != nil {
				http.Error(w, "failed to generate xlsx", http.StatusInternalServerError)
			}
			return
		}
		if normalizedReportType == "course" {
			if writeErr := writeGroupedXLSXAttachment(w, baseName+".xlsx", originalRecords, records, "Course Name", []string{"Course Code", "Faculty Code"}); writeErr != nil {
				http.Error(w, "failed to generate xlsx", http.StatusInternalServerError)
			}
			return
		}
		if writeErr := writeXLSXAttachment(w, baseName+".xlsx", records); writeErr != nil {
			http.Error(w, "failed to generate xlsx", http.StatusInternalServerError)
		}
	case "pdf":
		title := fmt.Sprintf("Mark Entry %s Report - Semester %d - %s", strings.Title(strings.ReplaceAll(normalizedReportType, "_", " ")), semester, departmentName)
		if writeErr := writePDFAttachment(w, baseName+".pdf", title, records); writeErr != nil {
			http.Error(w, "failed to generate pdf", http.StatusInternalServerError)
		}
	default:
		http.Error(w, "format must be csv, xlsx or pdf", http.StatusBadRequest)
	}
}
