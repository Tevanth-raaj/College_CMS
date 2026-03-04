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
	Subject1Semester       string
	Subject1Department     string
	Subject1CourseCode     string
	Subject1Title          string
	Subject1CourseType     string
	Subject1CourseNature   string
	Subject2Semester       string
	Subject2Department     string
	Subject2CourseCode     string
	Subject2Title          string
	Subject2CourseType     string
	Subject2CourseNature   string
	Subject3Semester       string
	Subject3Department     string
	Subject3CourseCode     string
	Subject3Title          string
	Subject3CourseType     string
	Subject3CourseNature   string
	Subject4Semester       string
	Subject4Department     string
	Subject4CourseCode     string
	Subject4Title          string
	Subject4CourseType     string
	Subject4CourseNature   string
	Subject5Semester       string
	Subject5Department     string
	Subject5CourseCode     string
	Subject5Title          string
	Subject5CourseType     string
	Subject5CourseNature   string
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

	query := `
		SELECT DISTINCT
			DATE_FORMAT(window_start, '%Y-%m-%d') as window_start,
			DATE_FORMAT(window_end,   '%Y-%m-%d') as window_end,
			COALESCE(semester_type, '') as semester_type,
			COALESCE(academic_year, '') as academic_year
		FROM teacher_course_history
		WHERE record_type = 'course'
		ORDER BY window_start DESC
	`

	rows, err := db.DB.Query(query)
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

// ExportTeacherLimits generates Excel export of teacher assignments and limits
func ExportTeacherLimits(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Optional window filter from query params
	windowStart := r.URL.Query().Get("window_start")
	windowEnd := r.URL.Query().Get("window_end")

	// Fetch all active teachers
	query := `
		SELECT 
			t.faculty_id,
			t.name,
			COALESCE(t.desg, '') as designation,
			COALESCE(t.dept, '') as dept
		FROM teachers t
		WHERE t.status = 1
		ORDER BY t.faculty_id ASC
	`

	rows, err := db.DB.Query(query)
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

		// Merge: core first, then special, capped at 5 slots
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
		}

		for i, c := range allCourses {
			if i >= 5 {
				break
			}
			fillCourseEntry(c, &exportRow, i+1)
		}

		exportRows = append(exportRows, exportRow)
	}

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

	headers := []string{
		"Faculty ID",
		"Faculty Name",
		"Designation",
		"Workload Limit - Theory",
		"Workload Limit - Lab",
		"Workload Limit - Theory+Lab",
		"Subjects Allotted",
		"Labs Allotted",
		"Semester - Subject 1",
		"Department - Subject 1",
		"Course Code - Subject 1",
		"Course Title - Subject 1",
		"Course Type - Subject 1",
		"Course Nature - Subject 1",
		"Semester - Subject 2",
		"Department - Subject 2",
		"Course Code - Subject 2",
		"Course Title - Subject 2",
		"Course Type - Subject 2",
		"Course Nature - Subject 2",
		"Semester - Subject 3",
		"Department - Subject 3",
		"Course Code - Subject 3",
		"Course Title - Subject 3",
		"Course Type - Subject 3",
		"Course Nature - Subject 3",
		"Semester - Subject 4",
		"Department - Subject 4",
		"Course Code - Subject 4",
		"Course Title - Subject 4",
		"Course Type - Subject 4",
		"Course Nature - Subject 4",
		"Semester - Subject 5",
		"Department - Subject 5",
		"Course Code - Subject 5",
		"Course Title - Subject 5",
		"Course Type - Subject 5",
		"Course Nature - Subject 5",
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
			row.Subject1Semester,
			row.Subject1Department,
			row.Subject1CourseCode,
			row.Subject1Title,
			row.Subject1CourseType,
			row.Subject1CourseNature,
			row.Subject2Semester,
			row.Subject2Department,
			row.Subject2CourseCode,
			row.Subject2Title,
			row.Subject2CourseType,
			row.Subject2CourseNature,
			row.Subject3Semester,
			row.Subject3Department,
			row.Subject3CourseCode,
			row.Subject3Title,
			row.Subject3CourseType,
			row.Subject3CourseNature,
			row.Subject4Semester,
			row.Subject4Department,
			row.Subject4CourseCode,
			row.Subject4Title,
			row.Subject4CourseType,
			row.Subject4CourseNature,
			row.Subject5Semester,
			row.Subject5Department,
			row.Subject5CourseCode,
			row.Subject5Title,
			row.Subject5CourseType,
			row.Subject5CourseNature,
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
		LEFT JOIN course_type ct ON c.course_type = ct.id
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
		LIMIT 5
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
		LEFT JOIN course_type ct ON c.course_type = ct.id
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
		ORDER BY hes.semester ASC, tch.course_code ASC
		LIMIT 5
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

// fillCourseEntry copies a courseEntry into the numbered subject slot of an export row.
func fillCourseEntry(c courseEntry, row *TeacherLimitExportRow, index int) {
	switch index {
	case 1:
		row.Subject1Semester = c.Semester
		row.Subject1Department = c.Department
		row.Subject1CourseCode = c.CourseCode
		row.Subject1Title = c.Title
		row.Subject1CourseType = c.CourseType
		row.Subject1CourseNature = c.CourseNature
	case 2:
		row.Subject2Semester = c.Semester
		row.Subject2Department = c.Department
		row.Subject2CourseCode = c.CourseCode
		row.Subject2Title = c.Title
		row.Subject2CourseType = c.CourseType
		row.Subject2CourseNature = c.CourseNature
	case 3:
		row.Subject3Semester = c.Semester
		row.Subject3Department = c.Department
		row.Subject3CourseCode = c.CourseCode
		row.Subject3Title = c.Title
		row.Subject3CourseType = c.CourseType
		row.Subject3CourseNature = c.CourseNature
	case 4:
		row.Subject4Semester = c.Semester
		row.Subject4Department = c.Department
		row.Subject4CourseCode = c.CourseCode
		row.Subject4Title = c.Title
		row.Subject4CourseType = c.CourseType
		row.Subject4CourseNature = c.CourseNature
	case 5:
		row.Subject5Semester = c.Semester
		row.Subject5Department = c.Department
		row.Subject5CourseCode = c.CourseCode
		row.Subject5Title = c.Title
		row.Subject5CourseType = c.CourseType
		row.Subject5CourseNature = c.CourseNature
	}
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
