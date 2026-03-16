package curriculum

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type TeacherPreferencePeriod struct {
	AcademicYear string `json:"academic_year"`
	SemesterType string `json:"semester_type"`
	Label        string `json:"label"`
}

type TeacherPreferenceTeacher struct {
	TeacherID      int    `json:"teacher_id"`
	FacultyID      string `json:"faculty_id"`
	TeacherName    string `json:"teacher_name"`
	DepartmentID   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	Label          string `json:"label"`
}

type TeacherPreferenceExportRow struct {
	TeacherID       int
	FacultyID       string
	TeacherName     string
	DepartmentName  string
	AcademicYear    string
	SemesterType    string
	CourseCode      string
	CourseName      string
	Semester        int
	Batch           string
	CourseType      string
	Priority        int
	Status          string
	IsActive        int
	SubmittedAt     string
}

// GetTeacherPreferencePeriods returns distinct academic year + semester type values for teacher preferences.
func GetTeacherPreferencePeriods(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))

	query := `
		SELECT DISTINCT
			COALESCE(tcp.academic_year, '') AS academic_year,
			COALESCE(LOWER(TRIM(tcp.current_semester_type)), '') AS semester_type
		FROM teacher_course_preferences tcp
		JOIN teachers t ON t.id = tcp.teacher_id
		LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
		WHERE 1 = 1
	`
	args := []interface{}{}
	if departmentID != "" {
		query += `
			AND (
				CAST(t.dept AS CHAR) = ?
				OR CAST(d.id AS CHAR) = ?
				OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
			)
		`
		args = append(args, departmentID, departmentID, departmentID)
	}
	query += ` ORDER BY academic_year DESC, semester_type DESC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("GetTeacherPreferencePeriods error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch teacher preference periods"})
		return
	}
	defer rows.Close()

	result := []TeacherPreferencePeriod{}
	for rows.Next() {
		var p TeacherPreferencePeriod
		if err := rows.Scan(&p.AcademicYear, &p.SemesterType); err != nil {
			continue
		}
		if strings.TrimSpace(p.AcademicYear) == "" && strings.TrimSpace(p.SemesterType) == "" {
			continue
		}
		p.Label = strings.TrimSpace(p.AcademicYear)
		if p.SemesterType != "" {
			if p.Label != "" {
				p.Label += " – "
			}
			p.Label += strings.Title(p.SemesterType)
		}
		result = append(result, p)
	}
	if result == nil {
		result = []TeacherPreferencePeriod{}
	}

	json.NewEncoder(w).Encode(result)
}

// GetTeacherPreferenceTeachers returns teachers for teacher preference export filters.
// Optional filters: department_id, q (name/faculty search)
func GetTeacherPreferenceTeachers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))
	search := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))

	query := `
		SELECT
			t.id,
			COALESCE(t.faculty_id, ''),
			COALESCE(t.name, ''),
			COALESCE(CAST(d.id AS SIGNED), 0) AS department_id,
			COALESCE(d.department_name, '')
		FROM teachers t
		LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
		WHERE COALESCE(t.status, 1) = 1
	`
	args := []interface{}{}
	if departmentID != "" {
		query += `
			AND (
				CAST(t.dept AS CHAR) = ?
				OR CAST(d.id AS CHAR) = ?
				OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
			)
		`
		args = append(args, departmentID, departmentID, departmentID)
	}
	if search != "" {
		query += `
			AND (
				LOWER(TRIM(t.name)) LIKE ?
				OR LOWER(TRIM(t.faculty_id)) LIKE ?
			)
		`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	query += ` ORDER BY t.name ASC LIMIT 300`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("GetTeacherPreferenceTeachers error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch teachers"})
		return
	}
	defer rows.Close()

	result := []TeacherPreferenceTeacher{}
	for rows.Next() {
		var t TeacherPreferenceTeacher
		if err := rows.Scan(&t.TeacherID, &t.FacultyID, &t.TeacherName, &t.DepartmentID, &t.DepartmentName); err != nil {
			continue
		}
		t.Label = strings.TrimSpace(t.FacultyID + " - " + t.TeacherName + " (" + t.DepartmentName + ")")
		result = append(result, t)
	}

	if result == nil {
		result = []TeacherPreferenceTeacher{}
	}
	json.NewEncoder(w).Encode(result)
}

// ExportTeacherPreferences exports teacher-submitted course preference data with optional filters.
func ExportTeacherPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")

	academicYear := strings.TrimSpace(r.URL.Query().Get("academic_year"))
	semesterType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("semester_type")))
	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))
	courseCode := strings.TrimSpace(r.URL.Query().Get("course_code"))
	teacherID := strings.TrimSpace(r.URL.Query().Get("teacher_id"))

	query := `
		SELECT
			t.id,
			COALESCE(t.faculty_id, ''),
			COALESCE(t.name, ''),
			COALESCE(d.department_name, ''),
			COALESCE(tcp.academic_year, ''),
			COALESCE(LOWER(TRIM(tcp.current_semester_type)), ''),
			COALESCE(tcp.course_id, ''),
			COALESCE(c.course_name, ''),
			COALESCE(tcp.semester, 0),
			COALESCE(tcp.batch, ''),
			COALESCE(ct.course_type, ''),
			COALESCE(tcp.priority, 0),
			COALESCE(tcp.status, ''),
			COALESCE(tcp.is_active, 1),
			COALESCE(DATE_FORMAT(tcp.created_at, '%Y-%m-%d %H:%i:%s'), '')
		FROM teacher_course_preferences tcp
		JOIN teachers t ON t.id = tcp.teacher_id
		LEFT JOIN departments d ON d.id = CAST(t.dept AS UNSIGNED)
		LEFT JOIN courses c ON c.course_code = tcp.course_id
		LEFT JOIN course_type ct ON ct.id = tcp.course_type
		WHERE 1 = 1
	`
	args := []interface{}{}
	if academicYear != "" {
		query += ` AND tcp.academic_year = ?`
		args = append(args, academicYear)
	}
	if semesterType != "" {
		query += ` AND LOWER(TRIM(COALESCE(tcp.current_semester_type, ''))) = ?`
		args = append(args, semesterType)
	}
	if departmentID != "" {
		query += `
			AND (
				CAST(t.dept AS CHAR) = ?
				OR CAST(d.id AS CHAR) = ?
				OR LOWER(TRIM(d.department_name)) = LOWER(TRIM(?))
			)
		`
		args = append(args, departmentID, departmentID, departmentID)
	}
	if courseCode != "" {
		query += ` AND LOWER(TRIM(tcp.course_id)) = LOWER(TRIM(?))`
		args = append(args, courseCode)
	}
	if teacherID != "" {
		query += ` AND CAST(t.id AS CHAR) = ?`
		args = append(args, teacherID)
	}
	query += ` ORDER BY tcp.academic_year DESC, tcp.current_semester_type DESC, d.department_name ASC, t.name ASC, tcp.priority ASC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("ExportTeacherPreferences query error: %v", err)
		http.Error(w, "Failed to export teacher preference data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []TeacherPreferenceExportRow{}
	for rows.Next() {
		var row TeacherPreferenceExportRow
		if err := rows.Scan(
			&row.TeacherID,
			&row.FacultyID,
			&row.TeacherName,
			&row.DepartmentName,
			&row.AcademicYear,
			&row.SemesterType,
			&row.CourseCode,
			&row.CourseName,
			&row.Semester,
			&row.Batch,
			&row.CourseType,
			&row.Priority,
			&row.Status,
			&row.IsActive,
			&row.SubmittedAt,
		); err != nil {
			continue
		}
		result = append(result, row)
	}

	file := excelize.NewFile()
	sheet := "Teacher Preferences"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Teacher ID", "Faculty ID", "Teacher Name", "Department", "Academic Year",
		"Semester Type", "Course Code", "Course Name", "Semester", "Batch",
		"Course Type", "Priority", "Status", "Is Active", "Submitted At",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheet, cell, h)
	}

	for idx, row := range result {
		r := idx + 2
		values := []interface{}{
			row.TeacherID,
			row.FacultyID,
			row.TeacherName,
			row.DepartmentName,
			row.AcademicYear,
			strings.Title(row.SemesterType),
			row.CourseCode,
			row.CourseName,
			row.Semester,
			row.Batch,
			row.CourseType,
			row.Priority,
			row.Status,
			row.IsActive,
			row.SubmittedAt,
		}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			file.SetCellValue(sheet, cell, v)
		}
	}

	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		file.SetColWidth(sheet, col, col, 18)
	}

	headStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	file.SetCellStyle(sheet, "A1", "O1", headStyle)
	file.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	buf, err := file.WriteToBuffer()
	if err != nil {
		log.Printf("ExportTeacherPreferences buffer error: %v", err)
		http.Error(w, "Failed to generate export", http.StatusInternalServerError)
		return
	}

	stamp := time.Now().Format("20060102_150405")
	parts := []string{"teacher_preferences"}
	if academicYear != "" {
		parts = append(parts, academicYear)
	}
	if semesterType != "" {
		parts = append(parts, semesterType)
	}
	filename := fmt.Sprintf("%s_%s.xlsx", strings.Join(parts, "_"), stamp)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
