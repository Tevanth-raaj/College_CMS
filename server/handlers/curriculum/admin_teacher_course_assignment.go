package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

type adminTeacherInfo struct {
	TeacherID      int    `json:"teacher_id"`
	FacultyID      string `json:"faculty_id"`
	TeacherName    string `json:"teacher_name"`
	DepartmentID   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
}

type adminTeacherCourseOption struct {
	CourseID   int    `json:"course_id"`
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Category   string `json:"category"`
	CourseType string `json:"course_type"`
	Source     string `json:"source"`
	IsAssigned bool   `json:"is_assigned"`
}

type adminTeacherCourseUpdateRequest struct {
	CourseIDs []int `json:"course_ids"`
}

func resolveTeacherForAdmin(teacherRef string) (*adminTeacherInfo, error) {
	teacherRef = strings.TrimSpace(teacherRef)
	if teacherRef == "" {
		return nil, sql.ErrNoRows
	}

	var teacher adminTeacherInfo
	err := db.DB.QueryRow(`
		SELECT
			t.id,
			COALESCE(t.faculty_id, ''),
			COALESCE(t.name, ''),
			COALESCE(CAST(d.id AS SIGNED), 0) AS department_id,
			COALESCE(d.department_name, '')
		FROM teachers t
		LEFT JOIN departments d ON CAST(d.id AS CHAR) = CAST(t.dept AS CHAR)
		WHERE t.id = ? OR t.faculty_id = ?
		LIMIT 1
	`, teacherRef, teacherRef).Scan(
		&teacher.TeacherID,
		&teacher.FacultyID,
		&teacher.TeacherName,
		&teacher.DepartmentID,
		&teacher.DepartmentName,
	)
	if err != nil {
		return nil, err
	}

	return &teacher, nil
}

func getEligibleTeacherCoursesForDepartment(departmentID int) ([]adminTeacherCourseOption, map[int]bool, error) {
	courseMap := make(map[int]adminTeacherCourseOption)

	coreQuery := `
		SELECT DISTINCT
			c.id,
			COALESCE(c.course_code, ''),
			COALESCE(c.course_name, ''),
			COALESCE(c.category, 'Core') AS category,
			COALESCE(ct.course_type, 'theory') AS course_type
		FROM departments d
		INNER JOIN curriculum cur ON cur.id = d.current_curriculum_id
		INNER JOIN normal_cards nc ON nc.curriculum_id = cur.id
		INNER JOIN curriculum_courses cc ON cc.semester_id = nc.id
		INNER JOIN courses c ON c.id = cc.course_id
		LEFT JOIN course_type ct
			ON CAST(ct.id AS CHAR) = CAST(c.course_type AS CHAR)
			OR LOWER(TRIM(ct.course_type)) COLLATE utf8mb4_general_ci = LOWER(TRIM(CAST(c.course_type AS CHAR))) COLLATE utf8mb4_general_ci
		WHERE d.id = ?
		  AND d.status = 1
		  AND c.status = 1
		  AND nc.status = 1
		  AND LOWER(COALESCE(nc.card_type, '')) = 'semester'
		  AND (
			c.category IS NULL
			OR c.category NOT IN ('PE - Professional Elective', 'OE - Open Elective', 'Elective', 'Open Elective', 'Honour', 'Minor', 'Addon')
		  )
	`
	coreRows, err := db.DB.Query(coreQuery, departmentID)
	if err != nil {
		return nil, nil, err
	}
	defer coreRows.Close()

	for coreRows.Next() {
		var row adminTeacherCourseOption
		if err := coreRows.Scan(&row.CourseID, &row.CourseCode, &row.CourseName, &row.Category, &row.CourseType); err != nil {
			continue
		}
		row.Source = "core"
		courseMap[row.CourseID] = row
	}

	extraQuery := `
		SELECT DISTINCT
			c.id,
			COALESCE(c.course_code, ''),
			COALESCE(c.course_name, ''),
			COALESCE(c.category, 'Elective') AS category,
			COALESCE(ct.course_type, 'theory') AS course_type,
			COALESCE(hes.slot_name, '') AS slot_name
		FROM students s
		INNER JOIN student_elective_choices sec ON sec.student_id = s.id
		INNER JOIN hod_elective_selections hes ON hes.id = sec.hod_selection_id
		INNER JOIN courses c ON c.id = hes.course_id
		LEFT JOIN course_type ct
			ON CAST(ct.id AS CHAR) = CAST(c.course_type AS CHAR)
			OR LOWER(TRIM(ct.course_type)) COLLATE utf8mb4_general_ci = LOWER(TRIM(CAST(c.course_type AS CHAR))) COLLATE utf8mb4_general_ci
		WHERE s.department_id = ?
		  AND c.status = 1
		  AND hes.status = 'ACTIVE'
	`
	extraRows, err := db.DB.Query(extraQuery, departmentID)
	if err != nil {
		return nil, nil, err
	}
	defer extraRows.Close()

	for extraRows.Next() {
		var row adminTeacherCourseOption
		var slotName string
		if err := extraRows.Scan(&row.CourseID, &row.CourseCode, &row.CourseName, &row.Category, &row.CourseType, &slotName); err != nil {
			continue
		}
		row.Source = "extra"
		if strings.TrimSpace(slotName) != "" {
			row.Category = strings.TrimSpace(slotName)
		}

		existing, exists := courseMap[row.CourseID]
		if !exists || existing.Source != "core" {
			courseMap[row.CourseID] = row
		}
	}

	courses := make([]adminTeacherCourseOption, 0, len(courseMap))
	eligibleSet := make(map[int]bool, len(courseMap))
	for _, course := range courseMap {
		courses = append(courses, course)
		eligibleSet[course.CourseID] = true
	}

	sort.Slice(courses, func(i, j int) bool {
		if courses[i].Source != courses[j].Source {
			return courses[i].Source < courses[j].Source
		}
		return courses[i].CourseCode < courses[j].CourseCode
	})

	return courses, eligibleSet, nil
}

func getTeacherAssignedCourseIDs(facultyID string) (map[int]bool, error) {
	assigned := make(map[int]bool)
	rows, err := db.DB.Query(`
		SELECT DISTINCT course_id
		FROM teacher_course_history
		WHERE teacher_id = ?
		  AND record_type = 'course'
		  AND archived_at IS NULL
	`, facultyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			continue
		}
		assigned[courseID] = true
	}
	return assigned, nil
}

// GetAdminTeacherCourseAssignmentContext returns eligible and assigned courses for one teacher.
func GetAdminTeacherCourseAssignmentContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	teacherRef := mux.Vars(r)["teacherId"]
	teacher, err := resolveTeacherForAdmin(teacherRef)
	if err == sql.ErrNoRows {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetAdminTeacherCourseAssignmentContext: failed to resolve teacher: %v", err)
		http.Error(w, "Failed to resolve teacher", http.StatusInternalServerError)
		return
	}

	if teacher.DepartmentID <= 0 {
		http.Error(w, "Teacher has no mapped department", http.StatusBadRequest)
		return
	}

	courses, _, err := getEligibleTeacherCoursesForDepartment(teacher.DepartmentID)
	if err != nil {
		log.Printf("GetAdminTeacherCourseAssignmentContext: failed to get eligible courses: %v", err)
		http.Error(w, "Failed to fetch eligible courses", http.StatusInternalServerError)
		return
	}

	assignedSet, err := getTeacherAssignedCourseIDs(teacher.FacultyID)
	if err != nil {
		log.Printf("GetAdminTeacherCourseAssignmentContext: failed to fetch assignments: %v", err)
		http.Error(w, "Failed to fetch existing assignments", http.StatusInternalServerError)
		return
	}

	for i := range courses {
		courses[i].IsAssigned = assignedSet[courses[i].CourseID]
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"teacher": teacher,
		"courses": courses,
	})
}

// UpdateAdminTeacherCourseAssignments syncs teacher-course mappings inside teacher's eligible department scope.
func UpdateAdminTeacherCourseAssignments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	teacherRef := mux.Vars(r)["teacherId"]
	teacher, err := resolveTeacherForAdmin(teacherRef)
	if err == sql.ErrNoRows {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("UpdateAdminTeacherCourseAssignments: failed to resolve teacher: %v", err)
		http.Error(w, "Failed to resolve teacher", http.StatusInternalServerError)
		return
	}
	if teacher.DepartmentID <= 0 {
		http.Error(w, "Teacher has no mapped department", http.StatusBadRequest)
		return
	}

	var req adminTeacherCourseUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, eligibleSet, err := getEligibleTeacherCoursesForDepartment(teacher.DepartmentID)
	if err != nil {
		log.Printf("UpdateAdminTeacherCourseAssignments: failed to load eligible courses: %v", err)
		http.Error(w, "Failed to validate courses", http.StatusInternalServerError)
		return
	}

	requested := make(map[int]bool)
	invalidIDs := make([]int, 0)
	for _, id := range req.CourseIDs {
		if id <= 0 {
			continue
		}
		if !eligibleSet[id] {
			invalidIDs = append(invalidIDs, id)
			continue
		}
		requested[id] = true
	}

	if len(invalidIDs) > 0 {
		http.Error(w, "Request includes courses outside teacher department/student scope", http.StatusBadRequest)
		return
	}

	existing, err := getTeacherAssignedCourseIDs(teacher.FacultyID)
	if err != nil {
		log.Printf("UpdateAdminTeacherCourseAssignments: failed to fetch existing assignments: %v", err)
		http.Error(w, "Failed to fetch existing assignments", http.StatusInternalServerError)
		return
	}

	toRemove := make([]int, 0)
	for courseID := range existing {
		if eligibleSet[courseID] && !requested[courseID] {
			toRemove = append(toRemove, courseID)
		}
	}

	toAdd := make([]int, 0)
	for courseID := range requested {
		if !existing[courseID] {
			toAdd = append(toAdd, courseID)
		}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start update", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	removed := 0
	for _, courseID := range toRemove {
		result, delErr := tx.Exec(`
			DELETE FROM teacher_course_allocation
			WHERE teacher_id = ? AND course_id = ?
		`, teacher.FacultyID, courseID)
		if delErr != nil {
			log.Printf("UpdateAdminTeacherCourseAssignments: delete failed for %s -> %d: %v", teacher.FacultyID, courseID, delErr)
			http.Error(w, "Failed to update assignments", http.StatusInternalServerError)
			return
		}
		affected, _ := result.RowsAffected()
		removed += int(affected)
	}

	added := 0
	for _, courseID := range toAdd {
		_, insErr := tx.Exec(`
			INSERT INTO teacher_course_allocation (course_id, teacher_id, is_active)
			SELECT ?, ?, 1
			WHERE NOT EXISTS (
				SELECT 1 FROM teacher_course_allocation
				WHERE course_id = ? AND teacher_id = ?
			)
		`, courseID, teacher.FacultyID, courseID, teacher.FacultyID)
		if insErr != nil {
			log.Printf("UpdateAdminTeacherCourseAssignments: insert failed for %s -> %d: %v", teacher.FacultyID, courseID, insErr)
			http.Error(w, "Failed to update assignments", http.StatusInternalServerError)
			return
		}
		added++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit update", http.StatusInternalServerError)
		return
	}

	assignedIDs := make([]int, 0, len(requested))
	for courseID := range requested {
		assignedIDs = append(assignedIDs, courseID)
	}
	sort.Ints(assignedIDs)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"teacher_id":          teacher.TeacherID,
		"faculty_id":          teacher.FacultyID,
		"department_id":       teacher.DepartmentID,
		"department_name":     teacher.DepartmentName,
		"added_count":         added,
		"removed_count":       removed,
		"assigned_course_ids": assignedIDs,
		"message":             "Teacher course assignments updated",
	})
}

// GetAdminTeacherAssignmentTeachers returns active teachers for dropdown.
func GetAdminTeacherAssignmentTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	departmentID := strings.TrimSpace(r.URL.Query().Get("department_id"))

	query := `
		SELECT
			t.id,
			COALESCE(t.faculty_id, ''),
			COALESCE(t.name, ''),
			COALESCE(CAST(d.id AS SIGNED), 0) AS department_id,
			COALESCE(d.department_name, '')
		FROM teachers t
		LEFT JOIN departments d ON CAST(d.id AS CHAR) = CAST(t.dept AS CHAR)
		WHERE t.status = 1
	`
	args := make([]interface{}, 0)

	if departmentID != "" {
		query += ` AND CAST(d.id AS CHAR) = ?`
		args = append(args, departmentID)
	}
	if search != "" {
		query += ` AND (LOWER(t.name) LIKE ? OR LOWER(t.faculty_id) LIKE ?)`
		s := "%" + strings.ToLower(search) + "%"
		args = append(args, s, s)
	}

	query += ` ORDER BY t.name ASC LIMIT 200`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("GetAdminTeacherAssignmentTeachers: query failed: %v", err)
		http.Error(w, "Failed to fetch teachers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var teacherID, deptID int
		var facultyID, name, deptName string
		if err := rows.Scan(&teacherID, &facultyID, &name, &deptID, &deptName); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"teacher_id":      teacherID,
			"faculty_id":      facultyID,
			"teacher_name":    name,
			"department_id":   deptID,
			"department_name": deptName,
			"label":           strings.TrimSpace(facultyID + " - " + name + " (" + deptName + ")"),
		})
	}

	json.NewEncoder(w).Encode(results)
}
