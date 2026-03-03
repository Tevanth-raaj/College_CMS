package curriculum

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server/db"

	"github.com/gorilla/mux"
	"github.com/xuri/excelize/v2"
)

// ExamAbsentee represents an absentee record
type ExamAbsentee struct {
	ID             int        `json:"id"`
	WindowID       int        `json:"window_id"`
	CourseID       int        `json:"course_id"`
	CourseCode     string     `json:"course_code,omitempty"`
	CourseName     string     `json:"course_name,omitempty"`
	StudentID      int        `json:"student_id"`
	StudentName    string     `json:"student_name,omitempty"`
	RegisterNo     string     `json:"register_no,omitempty"`
	MarkCategoryID int        `json:"mark_category_id"`
	CategoryName   string     `json:"category_name,omitempty"`
	LearningModeID int        `json:"learning_mode_id"`
	CreatedAt      time.Time  `json:"created_at"`
	WindowEndAt    *time.Time `json:"window_end_at,omitempty"`
}

type absenteeRow struct {
	examID     int
	courseCode string
	registerNo string
}

// parseExcelAbsentees reads an xlsx file and returns rows.
// Expected columns (any order, case-insensitive): exam_id, course_id, student_id
func parseExcelAbsentees(file io.Reader) ([]absenteeRow, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("cannot open xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("cannot read rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file must have a header row and at least one data row")
	}

	// Parse header row
	headerMap := map[string]int{}
	for idx, cell := range rows[0] {
		headerMap[strings.TrimSpace(strings.ToLower(cell))] = idx
	}

	examIdx, ok1 := headerMap["exam_id"]
	courseIdx, ok2 := headerMap["course_id"]
	studentIdx, ok3 := headerMap["student_id"]
	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("missing required columns: exam_id, course_id, student_id")
	}

	getCell := func(row []string, idx int) string {
		if idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var result []absenteeRow
	for _, row := range rows[1:] {
		examIDStr := getCell(row, examIdx)
		courseCode := getCell(row, courseIdx)
		registerNo := getCell(row, studentIdx)

		if examIDStr == "" && courseCode == "" && registerNo == "" {
			continue // skip blank rows
		}

		examID, err := strconv.Atoi(examIDStr)
		if err != nil || examID == 0 {
			return nil, fmt.Errorf("invalid exam_id value '%s': must be a positive integer", examIDStr)
		}

		result = append(result, absenteeRow{
			examID:     examID,
			courseCode: courseCode,
			registerNo: registerNo,
		})
	}
	return result, nil
}

// PreviewAbsentee is the resolved row returned to the UI before any DB write.
type PreviewAbsentee struct {
	RowNum         int    `json:"row_num"`
	ExamID         int    `json:"exam_id"`
	CourseID       int    `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	StudentID      int    `json:"student_id"`
	StudentName    string `json:"student_name"`
	RegisterNo     string `json:"register_no"`
	MarkCategoryID int    `json:"mark_category_id"`
	CategoryName   string `json:"category_name"`
}

type PreviewError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// PreviewAbsentees parses an XL file and returns resolved rows WITHOUT inserting anything.
// Accepts the same form fields as UploadAbsentees.
func PreviewAbsentees(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form (max 10 MB)", http.StatusBadRequest)
		return
	}

	markCategoryIDsStr := r.FormValue("mark_category_ids")
	if markCategoryIDsStr == "" {
		markCategoryIDsStr = r.FormValue("mark_category_id")
	}
	if markCategoryIDsStr == "" {
		http.Error(w, "mark_category_ids is required", http.StatusBadRequest)
		return
	}

	seen := map[int]bool{}
	var markCategoryIDs []int
	for _, part := range strings.Split(markCategoryIDsStr, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id == 0 {
			http.Error(w, fmt.Sprintf("Invalid mark_category_id: '%s'", part), http.StatusBadRequest)
			return
		}
		if !seen[id] {
			seen[id] = true
			markCategoryIDs = append(markCategoryIDs, id)
		}
	}

	learningModeID, err := strconv.Atoi(r.FormValue("learning_mode_id"))
	if err != nil || (learningModeID != 1 && learningModeID != 2) {
		http.Error(w, "Invalid learning_mode_id", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	records, parseErr := parseExcelAbsentees(file.(io.Reader))
	if parseErr != nil {
		http.Error(w, "Failed to parse Excel file: "+parseErr.Error(), http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// For each selected ID, look up its name then find ALL sibling IDs sharing
	// that name+learning_mode (one ID per course type: theory, theory&lab, etc.)
	// orderedNames keeps the display order; nameToAllIDs is used by UploadAbsentees
	// to insert for every course type. Preview shows ONE row per unique name.
	type catGroup struct {
		name   string
		allIDs []int
	}
	var catGroups []catGroup
	seenNames := map[string]bool{}
	for _, catID := range markCategoryIDs {
		var catName string
		if err2 := database.QueryRow(`SELECT name FROM mark_category_types WHERE id = ?`, catID).Scan(&catName); err2 != nil {
			continue
		}
		if seenNames[catName] {
			continue
		}
		seenNames[catName] = true
		// Fetch all sibling IDs (same name, same learning_mode_id)
		sibRows, err2 := database.Query(
			`SELECT id FROM mark_category_types WHERE name = ? AND learning_mode_id = ? AND status = 1`,
			catName, learningModeID,
		)
		if err2 != nil {
			continue
		}
		var siblings []int
		for sibRows.Next() {
			var sid int
			sibRows.Scan(&sid)
			siblings = append(siblings, sid)
		}
		sibRows.Close()
		catGroups = append(catGroups, catGroup{name: catName, allIDs: siblings})
	}

	var rows []PreviewAbsentee
	var rowErrors []PreviewError

	for i, rec := range records {
		rowNum := i + 2

		// Validate exam window exists
		var windowExists int
		if err := database.QueryRow(
			`SELECT COUNT(*) FROM mark_entry_windows WHERE id = ?`, rec.examID,
		).Scan(&windowExists); err != nil || windowExists == 0 {
			rowErrors = append(rowErrors, PreviewError{rowNum, fmt.Sprintf("exam_id %d has no mark entry window — please check the window ID", rec.examID)})
			continue
		}

		// Resolve course
		var courseID int
		var courseName string
		if err := database.QueryRow(
			`SELECT id, COALESCE(course_name,'') FROM courses WHERE course_code = ?`, rec.courseCode,
		).Scan(&courseID, &courseName); err != nil {
			rowErrors = append(rowErrors, PreviewError{rowNum, fmt.Sprintf("course_code '%s' not found", rec.courseCode)})
			continue
		}

		// Resolve student
		var studentID int
		var studentName string
		if err := database.QueryRow(
			`SELECT id, COALESCE(student_name,'') FROM students WHERE register_no = ?`, rec.registerNo,
		).Scan(&studentID, &studentName); err != nil {
			rowErrors = append(rowErrors, PreviewError{rowNum, fmt.Sprintf("register_no '%s' not found", rec.registerNo)})
			continue
		}

		// One preview row per UNIQUE category name (the user sees 4 students × 3 names = 12)
		// The representative ID shown is the first sibling; all siblings are stored on upload.
		for _, grp := range catGroups {
			repID := grp.allIDs[0]
			rows = append(rows, PreviewAbsentee{
				RowNum:         rowNum,
				ExamID:         rec.examID,
				CourseID:       courseID,
				CourseCode:     rec.courseCode,
				CourseName:     courseName,
				StudentID:      studentID,
				StudentName:    studentName,
				RegisterNo:     rec.registerNo,
				MarkCategoryID: repID,
				CategoryName:   grp.name,
			})
		}
	}

	if rows == nil {
		rows = []PreviewAbsentee{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rows":       rows,
		"error_rows": rowErrors,
		"total_xl":   len(records),
	})
}

// UploadAbsentees handles multipart XL file upload to record exam absentees.
// Form fields: file (.xlsx), mark_category_ids (comma-separated ints), learning_mode_id (1=UAL, 2=PBL)
// XL columns:  exam_id (window PK), course_id (course_code), student_id (register_no)
func UploadAbsentees(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form (max 10 MB)", http.StatusBadRequest)
		return
	}

	// Accept either mark_category_ids (new, comma-separated) or mark_category_id (legacy single)
	markCategoryIDsStr := r.FormValue("mark_category_ids")
	if markCategoryIDsStr == "" {
		markCategoryIDsStr = r.FormValue("mark_category_id")
	}
	if markCategoryIDsStr == "" {
		http.Error(w, "mark_category_ids is required", http.StatusBadRequest)
		return
	}

	seenCats := map[int]bool{}
	var markCategoryIDs []int
	for _, part := range strings.Split(markCategoryIDsStr, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id == 0 {
			http.Error(w, fmt.Sprintf("Invalid mark_category_id value: '%s'", part), http.StatusBadRequest)
			return
		}
		if !seenCats[id] {
			seenCats[id] = true
			markCategoryIDs = append(markCategoryIDs, id)
		}
	}

	learningModeID, err := strconv.Atoi(r.FormValue("learning_mode_id"))
	if err != nil || (learningModeID != 1 && learningModeID != 2) {
		http.Error(w, "Invalid learning_mode_id: must be 1 (UAL) or 2 (PBL)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	records, parseErr := parseExcelAbsentees(file.(io.Reader))
	if parseErr != nil {
		http.Error(w, "Failed to parse Excel file: "+parseErr.Error(), http.StatusBadRequest)
		return
	}
	if len(records) == 0 {
		http.Error(w, "No data rows found in file", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// Expand each selected ID to all sibling IDs sharing the same name+learning_mode.
	// This ensures a row is inserted for every course-type variant (theory, theory&lab, etc.)
	// so mark entry is blocked regardless of which catalogue the student's course belongs to.
	type uploadCatGroup struct {
		name   string
		allIDs []int
	}
	var uploadCatGroups []uploadCatGroup
	seenUploadNames := map[string]bool{}
	for _, catID := range markCategoryIDs {
		var catName string
		if err2 := database.QueryRow(`SELECT name FROM mark_category_types WHERE id = ? AND status = 1`, catID).Scan(&catName); err2 != nil {
			http.Error(w, fmt.Sprintf("Mark category %d not found or inactive", catID), http.StatusBadRequest)
			return
		}
		if seenUploadNames[catName] {
			continue
		}
		seenUploadNames[catName] = true
		sibRows, err2 := database.Query(
			`SELECT id FROM mark_category_types WHERE name = ? AND learning_mode_id = ? AND status = 1`,
			catName, learningModeID,
		)
		if err2 != nil {
			http.Error(w, "Failed to expand category siblings: "+err2.Error(), http.StatusInternalServerError)
			return
		}
		var siblings []int
		for sibRows.Next() {
			var sid int
			sibRows.Scan(&sid)
			siblings = append(siblings, sid)
		}
		sibRows.Close()
		uploadCatGroups = append(uploadCatGroups, uploadCatGroup{name: catName, allIDs: siblings})
	}

	type RowError struct {
		Row     int    `json:"row"`
		Message string `json:"message"`
	}

	var inserted, skipped int
	var rowErrors []RowError

	for i, rec := range records {
		rowNum := i + 2 // 1-indexed + header row

		// Validate exam_id exists in mark_entry_windows
		var windowExists int
		if err := database.QueryRow(
			`SELECT COUNT(*) FROM mark_entry_windows WHERE id = ?`, rec.examID,
		).Scan(&windowExists); err != nil || windowExists == 0 {
			rowErrors = append(rowErrors, RowError{rowNum, fmt.Sprintf("exam_id %d not found in mark entry windows — please check the window ID", rec.examID)})
			continue
		}

		// Resolve course by course_code
		var courseID int
		if err := database.QueryRow(
			`SELECT id FROM courses WHERE course_code = ?`, rec.courseCode,
		).Scan(&courseID); err != nil {
			rowErrors = append(rowErrors, RowError{rowNum, fmt.Sprintf("course_code '%s' not found", rec.courseCode)})
			continue
		}

		// Resolve student by register_no
		var studentID int
		if err := database.QueryRow(
			`SELECT id FROM students WHERE register_no = ?`, rec.registerNo,
		).Scan(&studentID); err != nil {
			rowErrors = append(rowErrors, RowError{rowNum, fmt.Sprintf("student register_no '%s' not found", rec.registerNo)})
			continue
		}

		// Insert ONE canonical record per unique category name.
		// (allIDs[0] is the MIN id representative; IsStudentAbsentForComponent
		//  checks by name so this covers every course-type sibling correctly.)
		rowHadError := false
		for _, grp := range uploadCatGroups {
			canonicalID := grp.allIDs[0]
			res, err := database.Exec(`
				INSERT INTO exam_absentees (window_id, course_id, student_id, mark_category_id, learning_mode_id)
				VALUES (?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE learning_mode_id = VALUES(learning_mode_id)
			`, rec.examID, courseID, studentID, canonicalID, learningModeID)
			if err != nil {
				log.Printf("absentees: insert error row %d cat %d: %v", rowNum, canonicalID, err)
				rowErrors = append(rowErrors, RowError{rowNum, fmt.Sprintf("database insert error for category '%s'", grp.name)})
				rowHadError = true
				continue
			}
			ra, _ := res.RowsAffected()
			if ra > 0 {
				inserted++
			} else {
				skipped++
			}
		}
		_ = rowHadError
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    len(rowErrors) == 0,
		"inserted":   inserted,
		"skipped":    skipped,
		"error_rows": rowErrors,
		"total_rows": len(records),
	})
}

// GetExamAbsentees returns absentee records for the COE management view.
// Query params: window_id, learning_mode_id, mark_category_id (all optional)
func GetExamAbsentees(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "GET, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	var conditions []string
	var args []interface{}

	if v := r.URL.Query().Get("window_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			conditions = append(conditions, "ea.window_id = ?")
			args = append(args, id)
		}
	}
	if v := r.URL.Query().Get("learning_mode_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			conditions = append(conditions, "ea.learning_mode_id = ?")
			args = append(args, id)
		}
	}
	if v := r.URL.Query().Get("mark_category_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			conditions = append(conditions, "ea.mark_category_id = ?")
			args = append(args, id)
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT
			MIN(ea.id)                              AS id,
			ea.window_id,
			ea.course_id,
			COALESCE(c.course_code, '')             AS course_code,
			COALESCE(c.course_name, '')             AS course_name,
			ea.student_id,
			COALESCE(s.student_name, '')            AS student_name,
			COALESCE(s.register_no, '')             AS register_no,
			ea.mark_category_id,
			COALESCE(mct.name, '')                  AS category_name,
			ea.learning_mode_id,
			MIN(ea.created_at)                      AS created_at,
			MAX(mew.end_at)                         AS window_end_at
		FROM exam_absentees ea
		LEFT JOIN courses c ON ea.course_id = c.id
		LEFT JOIN students s ON ea.student_id = s.id
		LEFT JOIN mark_category_types mct ON ea.mark_category_id = mct.id
		LEFT JOIN mark_entry_windows mew ON ea.window_id = mew.id
		%s
		GROUP BY ea.window_id, ea.course_id, ea.student_id, ea.mark_category_id,
		         ea.learning_mode_id, c.course_code, c.course_name,
		         s.student_name, s.register_no, mct.name
		ORDER BY MIN(ea.created_at) DESC
	`, where)

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("absentees: query error: %v", err)
		http.Error(w, "Failed to fetch absentees", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result []ExamAbsentee
	for rows.Next() {
		var a ExamAbsentee
		if err := rows.Scan(
			&a.ID, &a.WindowID, &a.CourseID, &a.CourseCode, &a.CourseName,
			&a.StudentID, &a.StudentName, &a.RegisterNo,
			&a.MarkCategoryID, &a.CategoryName, &a.LearningModeID, &a.CreatedAt,
			&a.WindowEndAt,
		); err != nil {
			log.Printf("absentees: scan error: %v", err)
			continue
		}
		result = append(result, a)
	}
	if result == nil {
		result = []ExamAbsentee{}
	}
	json.NewEncoder(w).Encode(result)
}

// GetCourseWindowAbsentees returns absentee entries for a course's currently active window(s).
// Used by the teacher's MarkEntryPage to render cells as blocked.
// Path: /api/course/{courseId}/exam-absentees
func GetCourseWindowAbsentees(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "GET, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil || courseID == 0 {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	teacherID := r.URL.Query().Get("teacher_id")

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	log.Printf("[ABSENTEES ENDPOINT] Fetching absentees for courseID=%d, teacherID=%s", courseID, teacherID)

	// Simply return all absentees for this course
	// The blocking logic in SaveStudentMarks will match against the specific window ID
	// This avoids timezone issues with SQL NOW() vs Go time
	query := `
		SELECT ea.student_id, ea.mark_category_id, ea.id, ea.window_id
		FROM exam_absentees ea
		WHERE ea.course_id = ?
	`

	log.Printf("[ABSENTEES ENDPOINT] Executing query for courseID=%d", courseID)

	rows, err := database.Query(query, courseID)
	if err != nil {
		log.Printf("[ABSENTEES ENDPOINT] Query ERROR: %v", err)
		http.Error(w, "Failed to fetch absentees", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AbsenteeCell struct {
		StudentID      int `json:"student_id"`
		MarkCategoryID int `json:"mark_category_id"`
		ID             int `json:"id"`
		WindowID       int `json:"window_id"`
	}

	var result []AbsenteeCell
	for rows.Next() {
		var a AbsenteeCell
		if err := rows.Scan(&a.StudentID, &a.MarkCategoryID, &a.ID, &a.WindowID); err != nil {
			log.Printf("[ABSENTEES ENDPOINT] Scan error: %v", err)
			continue
		}
		result = append(result, a)
		log.Printf("[ABSENTEES ENDPOINT] Found absentee: studentID=%d, categoryID=%d, windowID=%d", a.StudentID, a.MarkCategoryID, a.WindowID)
	}

	if result == nil {
		result = []AbsenteeCell{}
	}

	log.Printf("[ABSENTEES ENDPOINT] Returning %d absentee records", len(result))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeleteExamAbsentee removes a single absentee record by ID.
func DeleteExamAbsentee(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || id == 0 {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	if _, err := database.Exec(`DELETE FROM exam_absentees WHERE id = ?`, id); err != nil {
		log.Printf("absentees: delete error: %v", err)
		http.Error(w, "Failed to delete absentee", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteExamAbsenteesByWindow removes ALL absentee records for a given window_id.
func DeleteExamAbsenteesByWindow(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	windowID, err := strconv.Atoi(vars["windowId"])
	if err != nil || windowID == 0 {
		http.Error(w, "Invalid window ID", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	result, err := database.Exec(`DELETE FROM exam_absentees WHERE window_id = ?`, windowID)
	if err != nil {
		log.Printf("absentees: bulk delete error: %v", err)
		http.Error(w, "Failed to delete absentees", http.StatusInternalServerError)
		return
	}

	deleted, _ := result.RowsAffected()
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted": deleted})
}

// GetMarkCategoriesByLearningMode returns mark categories filtered by learning mode.
// Used to populate the component dropdown on the Absentees page.
// Query params: learning_mode_id (1=UAL, 2=PBL)
func GetMarkCategoriesByLearningMode(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, "GET, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	learningModeID, err := strconv.Atoi(r.URL.Query().Get("learning_mode_id"))
	if err != nil || (learningModeID != 1 && learningModeID != 2) {
		http.Error(w, "learning_mode_id must be 1 (UAL) or 2 (PBL)", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	rows, err := database.Query(`
		SELECT MIN(id) AS id, name, MAX(max_marks) AS max_marks, learning_mode_id, MIN(position) AS position
		FROM mark_category_types
		WHERE learning_mode_id = ? AND status = 1
		GROUP BY name, learning_mode_id
		ORDER BY MIN(position), MIN(id)
	`, learningModeID)
	if err != nil {
		log.Printf("absentees: mark categories query error: %v", err)
		http.Error(w, "Failed to fetch mark categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type MarkCat struct {
		ID             int     `json:"id"`
		Name           string  `json:"name"`
		MaxMarks       float64 `json:"max_marks"`
		LearningModeID int     `json:"learning_mode_id"`
		Position       int     `json:"position"`
	}

	var result []MarkCat
	for rows.Next() {
		var mc MarkCat
		if err := rows.Scan(&mc.ID, &mc.Name, &mc.MaxMarks, &mc.LearningModeID, &mc.Position); err != nil {
			continue
		}
		result = append(result, mc)
	}
	if result == nil {
		result = []MarkCat{}
	}
	json.NewEncoder(w).Encode(result)
}

// setCORSHeaders is a small utility to set common CORS + content-type headers.
func setCORSHeaders(w http.ResponseWriter, methods string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", methods+", OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

// IsStudentAbsentForComponent checks exam_absentees for an active window matching
// the given course + student + mark component (or any sibling sharing the same name).
// This handles the case where the DB stores one canonical ID but the save uses a
// different sibling ID for a different course type (theory vs theory&lab).
func IsStudentAbsentForComponent(windowID, courseID, studentID, markCategoryID int) (bool, error) {
	database := db.DB
	if database == nil {
		return false, fmt.Errorf("database not initialised")
	}
	var count int
	// Match on category name so any stored sibling ID counts as absent in this specific window.
	query := `
		SELECT COUNT(*)
		FROM exam_absentees ea
		INNER JOIN mark_category_types mct_stored  ON ea.mark_category_id = mct_stored.id
		INNER JOIN mark_category_types mct_current ON mct_current.id = ?
		WHERE ea.window_id = ?
		  AND ea.course_id  = ?
		  AND ea.student_id = ?
		  AND mct_stored.name = mct_current.name
	`
	log.Printf("[ABSENTEE QUERY] Query params: markCategoryID=%d, windowID=%d, courseID=%d, studentID=%d", markCategoryID, windowID, courseID, studentID)
	err := database.QueryRow(query, markCategoryID, windowID, courseID, studentID).Scan(&count)
	log.Printf("[ABSENTEE QUERY] Result: count=%d, error=%v", count, err)
	return count > 0, err
}

// multipartFileToReader casts a multipart.File to io.Reader — helper kept for clarity.
func multipartFileToReader(f multipart.File) io.Reader { return f }
