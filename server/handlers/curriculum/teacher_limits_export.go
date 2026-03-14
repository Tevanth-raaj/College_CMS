package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// courseEntry holds one assigned course entry for a teacher
type courseEntry struct {
	Semester     string
	Department   string
	CourseCode   string
	Title        string
	CourseType   string // Core / Elective / Open Elective / Honour / Minor / Addon etc.
	CourseNature string // Theory / Lab / Theory with Lab
}

// courseDetails holds rich metadata from the courses table for expanded exports.
type courseDetails struct {
	Credit            int
	LectureHrs        int
	TutorialHrs       int
	PracticalHrs      int
	ActivityHrs       int
	TwSlHrs           int
	TheoryTotalHrs    int
	TutorialTotalHrs  int
	PracticalTotalHrs int
	ActivityTotalHrs  int
	TotalHrs          int
	CIAMarks          int
	SEEMarks          int
	TotalMarks        int
}

// fetchCourseDetailsBatch returns a map[courseCode]courseDetails for the given codes.
func fetchCourseDetailsBatch(codes []string) map[string]courseDetails {
	result := map[string]courseDetails{}
	if len(codes) == 0 {
		return result
	}
	placeholders := strings.Repeat("?,", len(codes))
	placeholders = placeholders[:len(placeholders)-1]
	query := `
		SELECT
			course_code,
			COALESCE(credit, 0),
			COALESCE(lecture_hrs, 0),
			COALESCE(tutorial_hrs, 0),
			COALESCE(practical_hrs, 0),
			COALESCE(activity_hrs, 0),
			COALESCE(` + "`tw/sl`" + `, 0),
			COALESCE(theory_total_hrs, 0),
			COALESCE(tutorial_total_hrs, 0),
			COALESCE(practical_total_hrs, 0),
			COALESCE(activity_total_hrs, 0),
			COALESCE(total_hrs, 0),
			COALESCE(cia_marks, 0),
			COALESCE(see_marks, 0),
			COALESCE(total_marks, 0)
		FROM courses
		WHERE course_code IN (` + placeholders + `)
	`
	args := make([]interface{}, len(codes))
	for i, c := range codes {
		args[i] = c
	}
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("⚠️  fetchCourseDetailsBatch error: %v", err)
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var d courseDetails
		if err := rows.Scan(&code,
			&d.Credit, &d.LectureHrs, &d.TutorialHrs, &d.PracticalHrs, &d.ActivityHrs, &d.TwSlHrs,
			&d.TheoryTotalHrs, &d.TutorialTotalHrs, &d.PracticalTotalHrs, &d.ActivityTotalHrs, &d.TotalHrs,
			&d.CIAMarks, &d.SEEMarks, &d.TotalMarks,
		); err == nil {
			result[code] = d
		}
	}
	return result
}

// TeacherLimitExportRow represents a row in the export
type TeacherLimitExportRow struct {
	ID                     string
	FacultyName            string
	Designation            string
	WorkloadLimitTheory    string
	WorkloadLimitLab       string
	WorkloadLimitTheoryLab string
	SubjectsAlloted        string
	LabsAlloted            string
	Courses                []courseEntry
}

// AllocationWindow represents a distinct allocation window from teacher_course_history.
type AllocationWindow struct {
	WindowStart  string `json:"window_start"`
	WindowEnd    string `json:"window_end"`
	SemesterType string `json:"semester_type"`
	AcademicYear string `json:"academic_year"`
	Label        string `json:"label"`
}

// GetTeacherLimitWindows returns distinct allocation windows from teacher_course_history.
func GetTeacherLimitWindows(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))

	query := `
		SELECT DISTINCT
			DATE_FORMAT(tch.window_start, '%Y-%m-%d') as window_start,
			DATE_FORMAT(tch.window_end,   '%Y-%m-%d') as window_end,
			COALESCE(tch.semester_type, '') as semester_type,
			COALESCE(tch.academic_year, '') as academic_year
		FROM teacher_course_history tch
	`
	args := []interface{}{}
	if departmentID != "" {
		query += `
			JOIN teachers t ON t.faculty_id = tch.teacher_id
			LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
			WHERE tch.record_type = 'course' AND (
				t.dept = ?
				OR CAST(d.id AS CHAR) = ?
				OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
			)
		`
		args = append(args, departmentID, departmentID, departmentID)
	} else {
		query += `
			WHERE tch.record_type = 'course'
		`
	}
	query += `
		ORDER BY window_start DESC
	`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Error fetching allocation windows: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch windows"})
		return
	}
	defer rows.Close()

	var windows []AllocationWindow
	for rows.Next() {
		var win AllocationWindow
		if err := rows.Scan(&win.WindowStart, &win.WindowEnd, &win.SemesterType, &win.AcademicYear); err != nil {
			continue
		}
		label := win.AcademicYear
		if win.SemesterType != "" {
			label += " – " + win.SemesterType
		}
		if win.WindowStart != "" {
			label += " (" + win.WindowStart + " → " + win.WindowEnd + ")"
		}
		win.Label = label
		windows = append(windows, win)
	}
	if windows == nil {
		windows = []AllocationWindow{}
	}
	json.NewEncoder(w).Encode(windows)
}

// CourseOption is a lightweight course item for the filter dropdown.
type CourseOption struct {
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Label      string `json:"label"`
}

// GetTeacherLimitCourses returns distinct courses for the filter dropdown.
// It unions three sources:
//  1. Core courses from teacher_course_history (teacher dept filter)
//  2. Elective courses from student_elective_choices (PE/OE) via hod_elective_selections
//  3. Honour/Minor courses from student_elective_choices via hod_elective_selections
//
// All three are optionally scoped to a department_id.
func GetTeacherLimitCourses(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))

	// ── Part 1: core courses taught by teachers (teacher_course_history) ──────
	coreQuery := `
		SELECT
			tch.course_code,
			COALESCE(c.course_name, tch.course_name, '') AS course_name
		FROM teacher_course_history tch
		LEFT JOIN courses c ON c.course_code = tch.course_code
		JOIN teachers t ON t.faculty_id = tch.teacher_id AND t.status = 1
	`
	coreArgs := []interface{}{}
	coreWhere := []string{"tch.record_type = 'course'"}
	if departmentID != "" {
		coreQuery += `
			LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
		`
		coreWhere = append(coreWhere, `(
			t.dept = ?
			OR CAST(d.id AS CHAR) = ?
			OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
		)`)
		coreArgs = append(coreArgs, departmentID, departmentID, departmentID)
	}
	coreQuery += "\nWHERE " + strings.Join(coreWhere, " AND ")

	// ── Part 2: elective / honour / minor courses chosen by students ──────────
	// hod_elective_selections.department_id is the owning department (INT).
	// We filter by it when departmentID is supplied.
	electiveQuery := `
		SELECT
			c.course_code,
			COALESCE(c.course_name, '') AS course_name
		FROM student_elective_choices sec
		JOIN hod_elective_selections hes ON hes.id = sec.hod_selection_id
		JOIN courses c ON c.id = hes.course_id
		WHERE hes.status = 'ACTIVE'
	`
	electiveArgs := []interface{}{}
	if departmentID != "" {
		electiveQuery += ` AND (
			CAST(hes.department_id AS CHAR) = ?
			OR LOWER(TRIM((SELECT department_name FROM departments WHERE id = hes.department_id LIMIT 1)))
			   = LOWER(TRIM(?))
		)`
		electiveArgs = append(electiveArgs, departmentID, departmentID)
	}

	// Full UNION query – wrap both as subqueries so we can deduplicate and sort
	allArgs := append(coreArgs, electiveArgs...)
	fullQuery := `
		SELECT DISTINCT course_code, course_name
		FROM (
			` + coreQuery + `
			UNION ALL
			` + electiveQuery + `
		) AS combined
		WHERE course_code != ''
		ORDER BY course_code ASC
	`

	rows, err := db.DB.Query(fullQuery, allArgs...)
	if err != nil {
		log.Printf("❌ GetTeacherLimitCourses query error: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch courses"})
		return
	}
	defer rows.Close()

	seen := map[string]bool{}
	var options []CourseOption
	for rows.Next() {
		var opt CourseOption
		if err := rows.Scan(&opt.CourseCode, &opt.CourseName); err != nil {
			continue
		}
		if seen[opt.CourseCode] {
			continue
		}
		seen[opt.CourseCode] = true
		if opt.CourseName != "" {
			opt.Label = opt.CourseCode + " – " + opt.CourseName
		} else {
			opt.Label = opt.CourseCode
		}
		options = append(options, opt)
	}
	if options == nil {
		options = []CourseOption{}
	}
	json.NewEncoder(w).Encode(options)
}

// ExportTeacherLimits generates Excel export of teacher assignments and limits
func ExportTeacherLimits(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Optional window filter from query params
	windowStart := r.URL.Query().Get("window_start")
	windowEnd := r.URL.Query().Get("window_end")
	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))
	courseCodeFilter := strings.TrimSpace(r.URL.Query().Get("course_code"))

	// Fetch all active teachers in scope so teachers with no allocations are also
	// included (with 0 subjects/labs) instead of being skipped.
	query := `
		SELECT 
			t.faculty_id,
			COALESCE(t.name, '') as name,
			COALESCE(t.desg, '') as designation,
			COALESCE(t.dept, '') as dept
		FROM teachers t
	`
	args := []interface{}{}
	whereClauses := []string{"t.status = 1"}
	if departmentID != "" {
		query += `
			LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
		`
		whereClauses = append(whereClauses, `(
			t.dept = ?
			OR CAST(d.id AS CHAR) = ?
			OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
		)`) 
		args = append(args, departmentID, departmentID, departmentID)
	}
	query += "\nWHERE " + strings.Join(whereClauses, " AND ")
	query += `
		ORDER BY t.faculty_id ASC
	`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("❌ Error fetching teachers: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch teachers"})
		return
	}
	defer rows.Close()

	var exportRows []TeacherLimitExportRow

	for rows.Next() {
		var facultyID, name, designation, dept string

		if err := rows.Scan(&facultyID, &name, &designation, &dept); err != nil {
			log.Printf("❌ Error scanning teacher row: %v\n", err)
			continue
		}

		// Limits come from teacher_course_history record_type='limit'
		limitTheory := fetchTeacherLimit(facultyID, 1, windowStart, windowEnd)    // theory
		limitLab := fetchTeacherLimit(facultyID, 2, windowStart, windowEnd)       // lab
		limitTheoryLab := fetchTeacherLimit(facultyID, 3, windowStart, windowEnd) // theory_with_lab

		// Core course assignments from teacher_course_history (non-special categories)
		coreCourses, err := fetchCoreCourses(facultyID, windowStart, windowEnd)
		if err != nil {
			log.Printf("⚠️  Error fetching core courses for %s: %v\n", facultyID, err)
		}

		// Special courses from teacher_course_history (special categories only),
		// semester resolved via hod_elective_selections for teacher's dept
		var specialCourses []courseEntry
		if dept != "" {
			specialCourses, err = fetchSpecialCourses(facultyID, dept, windowStart, windowEnd)
			if err != nil {
				log.Printf("⚠️  Error fetching special courses for %s (dept=%s): %v\n", facultyID, dept, err)
			}
		}

		// Merge: core first, then special (no cap)
		allCourses := append(coreCourses, specialCourses...)

		subjectCount := 0
		labCount := 0
		for _, c := range allCourses {
			if c.CourseNature == "Lab" {
				labCount++
			} else {
				subjectCount++
			}
		}

		exportRow := TeacherLimitExportRow{
			ID:                     facultyID,
			FacultyName:            name,
			Designation:            designation,
			WorkloadLimitTheory:    formatInt(limitTheory),
			WorkloadLimitLab:       formatInt(limitLab),
			WorkloadLimitTheoryLab: formatInt(limitTheoryLab),
			SubjectsAlloted:        formatInt(subjectCount),
			LabsAlloted:            formatInt(labCount),
			Courses:                allCourses,
		}

		exportRows = append(exportRows, exportRow)
	}

	// ── MODE SELECTION ─────────────────────────────────────────────────────────
	// courseCodeFilter == ""       → normal wide format (one row per teacher)
	// courseCodeFilter == "__ALL__" → expanded format (one row per teacher-course, all)
	// courseCodeFilter == "<CODE>" → expanded format (one row per teacher, filtered to course)

	if courseCodeFilter != "" {
		// ── EXPANDED FORMAT ──────────────────────────────────────────────────
		// ── collect course codes for batch detail lookup ─────────────────────
		uniqCodes := map[string]bool{}
		for _, row := range exportRows {
			for _, c := range row.Courses {
				if courseCodeFilter == "__ALL__" || strings.EqualFold(c.CourseCode, courseCodeFilter) {
					uniqCodes[c.CourseCode] = true
				}
			}
		}
		codeSlice := make([]string, 0, len(uniqCodes))
		for code := range uniqCodes {
			codeSlice = append(codeSlice, code)
		}
		detailsMap := fetchCourseDetailsBatch(codeSlice)

		type expandedRow struct {
			FacultyID   string
			Name        string
			Designation string
			LimitTheory string
			LimitLab    string
			LimitTL     string
			Course      courseEntry
			Details     courseDetails
		}
		var expanded []expandedRow
		for _, row := range exportRows {
			for _, c := range row.Courses {
				if courseCodeFilter == "__ALL__" || strings.EqualFold(c.CourseCode, courseCodeFilter) {
					expanded = append(expanded, expandedRow{
						FacultyID:   row.ID,
						Name:        row.FacultyName,
						Designation: row.Designation,
						LimitTheory: row.WorkloadLimitTheory,
						LimitLab:    row.WorkloadLimitLab,
						LimitTL:     row.WorkloadLimitTheoryLab,
						Course:      c,
						Details:     detailsMap[c.CourseCode],
					})
				}
			}
		}

		f := excelize.NewFile()
		defer func() { _ = f.Close() }()
		sheetName := "Teacher Course Mapping"
		index, err := f.NewSheet(sheetName)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create  sheet"})
			return
		}
		headers := []string{
			"Faculty ID", "Faculty Name", "Designation",
			"Workload Limit - Theory", "Workload Limit - Lab", "Workload Limit - Theory+Lab",
			"Course Code", "Course Name", "Category", "Course Nature", "Semester", "Department",
			"Credits",
			"Lecture Hrs", "Tutorial Hrs", "Practical Hrs", "Activity Hrs", "TW/SL Hrs",
			"Theory Total Hrs", "Tutorial Total Hrs", "Practical Total Hrs", "Activity Total Hrs", "Total Hrs",
			"CIA Marks", "SEE Marks", "Total Marks",
		}
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
		}
		for ri, ex := range expanded {
			rowNum := ri + 2
			d := ex.Details
			record := []interface{}{
				ex.FacultyID, ex.Name, ex.Designation,
				ex.LimitTheory, ex.LimitLab, ex.LimitTL,
				ex.Course.CourseCode, ex.Course.Title,
				ex.Course.CourseType, ex.Course.CourseNature,
				ex.Course.Semester, ex.Course.Department,
				d.Credit,
				d.LectureHrs, d.TutorialHrs, d.PracticalHrs, d.ActivityHrs, d.TwSlHrs,
				d.TheoryTotalHrs, d.TutorialTotalHrs, d.PracticalTotalHrs, d.ActivityTotalHrs, d.TotalHrs,
				d.CIAMarks, d.SEEMarks, d.TotalMarks,
			}
			for ci, val := range record {
				cell, _ := excelize.CoordinatesToCellName(ci+1, rowNum)
				f.SetCellValue(sheetName, cell, val)
			}
		}
		f.SetActiveSheet(index)
		f.DeleteSheet("Sheet1")
		buffer, err := f.WriteToBuffer()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate Excel file"})
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=\"teacher_course_mapping_"+strings.ReplaceAll(getCurrentDateTime(), " ", "_")+".xlsx\"")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write(buffer.Bytes())
		log.Printf("✅ Expanded teacher-course mapping export served (%d rows)\n", len(expanded))
		return
	}

	// ── NORMAL FORMAT (existing behaviour) ───────────────────────────────────
	// Generate Excel file
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("❌ Error closing Excel file: %v\n", err)
		}
	}()

	sheetName := "Teacher Limits"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		log.Printf("❌ Error creating sheet: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create Excel sheet"})
		return
	}

	baseHeaders := []string{
		"Faculty ID",
		"Faculty Name",
		"Designation",
		"Workload Limit - Theory",
		"Workload Limit - Lab",
		"Workload Limit - Theory+Lab",
		"Subjects Allotted",
		"Labs Allotted",
	}

	maxSubjects := 0
	for _, row := range exportRows {
		if len(row.Courses) > maxSubjects {
			maxSubjects = len(row.Courses)
		}
	}

	headers := make([]string, 0, len(baseHeaders)+(maxSubjects*6))
	headers = append(headers, baseHeaders...)
	for i := 1; i <= maxSubjects; i++ {
		headers = append(headers,
			"Semester - Subject "+strconv.Itoa(i),
			"Department - Subject "+strconv.Itoa(i),
			"Course Code - Subject "+strconv.Itoa(i),
			"Course Title - Subject "+strconv.Itoa(i),
			"Course Type - Subject "+strconv.Itoa(i),
			"Course Nature - Subject "+strconv.Itoa(i),
		)
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for rowIndex, row := range exportRows {
		rowNum := rowIndex + 2
		record := []interface{}{
			row.ID,
			row.FacultyName,
			row.Designation,
			row.WorkloadLimitTheory,
			row.WorkloadLimitLab,
			row.WorkloadLimitTheoryLab,
			row.SubjectsAlloted,
			row.LabsAlloted,
		}

		for i := 0; i < maxSubjects; i++ {
			if i < len(row.Courses) {
				c := row.Courses[i]
				record = append(record,
					c.Semester,
					c.Department,
					c.CourseCode,
					c.Title,
					c.CourseType,
					c.CourseNature,
				)
			} else {
				record = append(record, "", "", "", "", "", "")
			}
		}

		for colIndex, value := range record {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowNum)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	f.SetActiveSheet(index)
	f.DeleteSheet("Sheet1")

	buffer, err := f.WriteToBuffer()
	if err != nil {
		log.Printf("❌ Excel generation error: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate Excel file"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"teacher_limits_"+strings.ReplaceAll(getCurrentDateTime(), " ", "_")+".xlsx\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Printf("❌ Error writing response: %v\n", err)
		return
	}

	log.Printf("✅ Teacher limits export served successfully\n")
}

// fetchTeacherLimit returns the latest max_count from teacher_course_history
// where record_type = 'limit' for the given teacher and course_type_id.
// If windowStart/windowEnd are non-empty the query is scoped to that window.
func fetchTeacherLimit(facultyID string, courseTypeID int, windowStart, windowEnd string) int {
	var maxCount int
	if windowStart != "" && windowEnd != "" {
		query := `
			SELECT max_count
			FROM teacher_course_history
			WHERE teacher_id = ? AND course_type_id = ? AND record_type = 'limit'
			  AND window_start = ? AND window_end = ?
			ORDER BY created_at DESC
			LIMIT 1
		`
		if err := db.DB.QueryRow(query, facultyID, courseTypeID, windowStart, windowEnd).Scan(&maxCount); err != nil {
			return 0
		}
	} else {
		query := `
			SELECT max_count
			FROM teacher_course_history
			WHERE teacher_id = ? AND course_type_id = ? AND record_type = 'limit'
			ORDER BY created_at DESC
			LIMIT 1
		`
		if err := db.DB.QueryRow(query, facultyID, courseTypeID).Scan(&maxCount); err != nil {
			return 0
		}
	}
	return maxCount
}

// fetchCoreCourses retrieves core (non-special) courses assigned to a teacher via
// teacher_course_history.  Special categories (elective / honour / minor / addon /
// open elective) are excluded because their semester is sourced from hod_elective_selections.
// Semester and department come from normal_cards → curriculum_courses → curriculum.
// If windowStart/windowEnd are provided, results are scoped to that allocation window.
func fetchCoreCourses(facultyID, windowStart, windowEnd string) ([]courseEntry, error) {
	windowFilter := ""
	args := []interface{}{facultyID}
	if windowStart != "" && windowEnd != "" {
		windowFilter = " AND tch.window_start = ? AND tch.window_end = ?"
		args = append(args, windowStart, windowEnd)
	}

	query := `
		SELECT 
			tch.course_code,
			tch.course_name,
			COALESCE(c.category, 'Core') as category,
			COALESCE(ct.course_type, 'theory') as course_nature,
			COALESCE(
				GROUP_CONCAT(DISTINCT nc.semester_number ORDER BY nc.semester_number SEPARATOR ','),
				''
			) as semesters,
			COALESCE(MAX(cur.name), '') as department
		FROM teacher_course_history tch
		LEFT JOIN courses c ON tch.course_code = c.course_code
		LEFT JOIN course_type ct ON c.course_type = ct.course_type
		LEFT JOIN curriculum_courses cc ON c.id = cc.course_id
		LEFT JOIN normal_cards nc ON cc.semester_id = nc.id
		LEFT JOIN curriculum cur ON nc.curriculum_id = cur.id
		WHERE tch.teacher_id = ?
		  AND tch.record_type = 'course'
		  AND (
		        c.category IS NULL
		        OR c.category NOT IN (
		            'PE - Professional Elective', 'OE - Open Elective',
		            'Elective', 'Open Elective', 'Honour', 'Minor', 'Addon'
		        )
		  )` + windowFilter + `
		GROUP BY tch.course_code, tch.course_name, c.category, ct.course_type
		ORDER BY tch.course_code ASC
	`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []courseEntry
	for rows.Next() {
		var code, name, category, nature, semesters, department string
		if err := rows.Scan(&code, &name, &category, &nature, &semesters, &department); err != nil {
			log.Printf("⚠️  Error scanning core course: %v\n", err)
			continue
		}
		entries = append(entries, courseEntry{
			Semester:     formatSemesterList(semesters),
			Department:   department,
			CourseCode:   code,
			Title:        name,
			CourseType:   category,
			CourseNature: formatCourseNature(nature),
		})
	}
	return entries, rows.Err()
}

// fetchSpecialCourses retrieves elective / open-elective / honour / minor / addon courses
// that are actually assigned to the teacher in teacher_course_history (record_type='course').
// The semester is resolved from hod_elective_selections using the teacher's department_id
// and the course_id from the history record.
// If windowStart/windowEnd are provided, results are scoped to that allocation window.
func fetchSpecialCourses(facultyID string, teacherDept string, windowStart, windowEnd string) ([]courseEntry, error) {
	windowFilter := ""
	args := []interface{}{teacherDept, teacherDept, facultyID}
	if windowStart != "" && windowEnd != "" {
		windowFilter = " AND tch.window_start = ? AND tch.window_end = ?"
		args = append(args, windowStart, windowEnd)
	}

	query := `
		SELECT DISTINCT
			tch.course_code,
			tch.course_name,
			COALESCE(hes.slot_name, c.category, 'Elective') as category,
			COALESCE(ct.course_type, 'theory') as course_nature,
			COALESCE(hes.semester, 0) as semester,
			COALESCE(d.department_name, '') as department
		FROM teacher_course_history tch
		JOIN courses c ON tch.course_code = c.course_code
		LEFT JOIN course_type ct ON c.course_type = ct.course_type
		LEFT JOIN hod_elective_selections hes
			ON hes.course_id = c.id
			AND hes.department_id = CAST(? AS UNSIGNED)
		LEFT JOIN departments d ON d.id = CAST(? AS UNSIGNED)
		WHERE tch.teacher_id = ?
		  AND tch.record_type = 'course'
		  AND c.category IN (
			'PE - Professional Elective', 'OE - Open Elective',
			'Elective', 'Open Elective', 'Honour', 'Minor', 'Addon'
		  )` + windowFilter + `
		ORDER BY semester ASC, tch.course_code ASC
	`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []courseEntry
	for rows.Next() {
		var code, name, category, nature, department string
		var semester int
		if err := rows.Scan(&code, &name, &category, &nature, &semester, &department); err != nil {
			log.Printf("⚠️  Error scanning special course: %v\n", err)
			continue
		}
		semStr := ""
		if semester > 0 {
			semStr = "Sem " + strconv.Itoa(semester)
		}
		entries = append(entries, courseEntry{
			Semester:     semStr,
			Department:   department,
			CourseCode:   code,
			Title:        name,
			CourseType:   formatSlotName(category),
			CourseNature: formatCourseNature(nature),
		})
	}
	return entries, rows.Err()
}

// formatCourseNature converts raw course_type.course_type to a display label.
func formatCourseNature(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lab":
		return "Lab"
	case "theory_with_lab":
		return "Theory with Lab"
	default:
		return "Theory"
	}
}

// formatSlotName normalises slot_name / category strings to clean display values.
func formatSlotName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "elective", "pe - professional elective":
		return "Elective"
	case "open elective", "oe - open elective":
		return "Open Elective"
	case "honour":
		return "Honour"
	case "minor":
		return "Minor"
	case "addon":
		return "Addon"
	default:
		return raw
	}
}

// formatSemesterList converts "3,5" → "Sem 3, Sem 5".
func formatSemesterList(raw string) string {
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	labelled := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			labelled = append(labelled, "Sem "+p)
		}
	}
	return strings.Join(labelled, ", ")
}

// formatInt converts int to string.
func formatInt(val int) string {
	return strconv.Itoa(val)
}

// getCurrentDateTime returns a formatted datetime string for filenames.
func getCurrentDateTime() string {
	return time.Now().Format("2006-01-02_15-04-05")
}
