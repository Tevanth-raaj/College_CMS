package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"server/models"

	"github.com/gorilla/mux"
)

// GetCourseAllocations retrieves courses for a specific semester and academic year with their faculty assignments
func GetCourseAllocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	semesterID := r.URL.Query().Get("semester_id")
	academicYear := r.URL.Query().Get("academic_year")

	if semesterID == "" || academicYear == "" {
		http.Error(w, "semester_id and academic_year are required", http.StatusBadRequest)
		return
	}

	// 1. Fetch all courses linked to this semester
	courseQuery := `
		SELECT c.id, c.course_code, c.course_name, ct.course_type, c.credit
		FROM courses c
		JOIN curriculum_courses cc ON c.id = cc.course_id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE cc.semester_id = ? AND c.status = 1
	`
	rows, err := db.DB.Query(courseQuery, semesterID)
	if err != nil {
		log.Printf("Error fetching courses for allocation: %v", err)
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []models.CourseWithAllocations
	for rows.Next() {
		var c models.CourseWithAllocations
		if err := rows.Scan(&c.CourseID, &c.CourseCode, &c.CourseName, &c.CourseType, &c.Credit); err != nil {
			log.Printf("Error scanning course row: %v", err)
			continue
		}
		c.Allocations = []models.CourseAllocation{}
		courses = append(courses, c)
	}

	// 2. Fetch all allocations for these courses
	allocationQuery := `
		SELECT ca.id, ca.course_id, ca.teacher_id, t.name
		FROM teacher_course_allocation ca
		JOIN teachers t ON ca.teacher_id = t.id
	`
	aRows, err := db.DB.Query(allocationQuery)
	if err != nil {
		log.Printf("Error fetching allocations: %v", err)
		// We still return courses even if allocations fetch fails
	} else {
		defer aRows.Close()
		allocMap := make(map[int][]models.CourseAllocation)
		for aRows.Next() {
			var a models.CourseAllocation
			if err := aRows.Scan(&a.ID, &a.CourseID, &a.TeacherID, &a.TeacherName); err != nil {
				continue
			}
			allocMap[a.CourseID] = append(allocMap[a.CourseID], a)
		}

		// 3. Merge allocations into courses
		for i := range courses {
			if allocs, ok := allocMap[courses[i].CourseID]; ok {
				courses[i].Allocations = allocs
			}
		}
	}

	json.NewEncoder(w).Encode(courses)
}

// CreateAllocation assigns a teacher to a course
func CreateAllocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var alloc models.CourseAllocation
	if err := json.NewDecoder(r.Body).Decode(&alloc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if alloc.CourseID == 0 || alloc.TeacherID == 0 {
		http.Error(w, "CourseID and TeacherID are required", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO teacher_course_allocation (course_id, teacher_id)
		VALUES (?, ?)
	`
	_, err := db.DB.Exec(query, alloc.CourseID, alloc.TeacherID)
	if err != nil {
		log.Printf("Error creating allocation: %v", err)
		http.Error(w, "Failed to create allocation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Allocation successful"})
}

// DeleteAllocation performs a soft delete of an allocation
func DeleteAllocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	id := vars["id"]

	query := `DELETE FROM teacher_course_allocation WHERE id = ?`
	_, err := db.DB.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting allocation: %v", err)
		http.Error(w, "Failed to delete allocation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Allocation removed successfully"})
}

// UpdateAllocation updates an existing allocation
func UpdateAllocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	id := vars["id"]

	var alloc models.CourseAllocation
	if err := json.NewDecoder(r.Body).Decode(&alloc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE teacher_course_allocation 
		SET teacher_id = ?
		WHERE id = ?
	`
	_, err := db.DB.Exec(query, alloc.TeacherID, id)
	if err != nil {
		log.Printf("Error updating allocation: %v", err)
		http.Error(w, "Failed to update allocation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Allocation updated successfully"})
}

// GetTeacherCourses retrieves all courses assigned to a specific teacher
func GetTeacherCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	teacherID := vars["id"]

	log.Printf("=== GetTeacherCourses called with teacherID='%s' ===", teacherID)

	// Convert numeric ID to faculty_id if it's a number
	var facultyID string
	err := db.DB.QueryRow("SELECT faculty_id FROM teachers WHERE id = ? OR faculty_id = ?", teacherID, teacherID).Scan(&facultyID)
	if err != nil {
		log.Printf("Error fetching teacher faculty_id: %v", err)
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}

	log.Printf("Resolved teacherID '%s' to faculty_id '%s'", teacherID, facultyID)

	query := `
		SELECT 
			ca.id, ca.course_id, c.course_code, c.course_name, ct.course_type, 
			c.credit, COALESCE(c.category, 'General')
		FROM teacher_course_allocation ca
		JOIN courses c ON ca.course_id = c.id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE ca.teacher_id = ?
		ORDER BY c.course_code
	`

	log.Printf("Executing query with faculty_id='%s'", facultyID)
	rows, err := db.DB.Query(query, facultyID)
	if err != nil {
		log.Printf("Error fetching teacher courses: %v", err)
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type StudentEnrollment struct {
		StudentID      int    `json:"student_id"`
		StudentName    string `json:"student_name"`
		LearningModeID *int   `json:"learning_mode_id"`
	}

	type TeacherCourse struct {
		ID          int                 `json:"id"`
		CourseID    int                 `json:"course_id"`
		CourseCode  string              `json:"course_code"`
		CourseName  string              `json:"course_name"`
		CourseType  string              `json:"course_type"`
		Credit      int                 `json:"credit"`
		Category    string              `json:"category"`
		Enrollments []StudentEnrollment `json:"enrollments"`
	}

	var courses []TeacherCourse
	log.Printf("Starting to iterate courses...")
	courseCount := 0
	for rows.Next() {
		var course TeacherCourse
		err := rows.Scan(&course.ID, &course.CourseID, &course.CourseCode, &course.CourseName,
			&course.CourseType, &course.Credit, &course.Category)
		if err != nil {
			log.Printf("Error scanning course row: %v", err)
			continue
		}
		courseCount++
		log.Printf("Found course #%d: ID=%d, Code=%s, Name=%s", courseCount, course.CourseID, course.CourseCode, course.CourseName)

		// Fetch allocated students for this course and teacher
		studentQuery := `
			SELECT DISTINCT s.id, s.student_name, s.learning_mode_id
			FROM course_student_teacher_allocation csta
			JOIN students s ON csta.student_id = s.id
			WHERE csta.course_id = ? AND csta.teacher_id = ?
			ORDER BY s.student_name
		`
		log.Printf("Querying students for courseID=%d, faculty_id='%s'", course.CourseID, facultyID)
		sRows, err := db.DB.Query(studentQuery, course.CourseID, facultyID)
		if err != nil {
			log.Printf("Error fetching students for course %d: %v", course.CourseID, err)
		} else {
			defer sRows.Close()
			course.Enrollments = []StudentEnrollment{}
			studentCount := 0
			for sRows.Next() {
				studentCount++
				var enrollment StudentEnrollment
				if err := sRows.Scan(&enrollment.StudentID, &enrollment.StudentName, &enrollment.LearningModeID); err != nil {
					log.Printf("Error scanning student row: %v", err)
					continue
				}
				course.Enrollments = append(course.Enrollments, enrollment)
			}
			log.Printf("Found %d students for course %s (ID=%d)", studentCount, course.CourseCode, course.CourseID)
			sRows.Close()
		}

		if course.Enrollments == nil {
			course.Enrollments = []StudentEnrollment{}
			log.Printf("No students found for course %s, setting empty array", course.CourseCode)
		}

		courses = append(courses, course)
	}

	log.Printf("Total courses found: %d", len(courses))
	if courses == nil {
		courses = []TeacherCourse{}
	}

	log.Printf("Returning %d courses to client", len(courses))
	json.NewEncoder(w).Encode(courses)
}

// GetCourseTeachers retrieves all teachers assigned to a specific course
func GetCourseTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	courseID := vars["id"]

	query := `
		SELECT 
			ca.id, ca.teacher_id, t.name, t.email, t.dept, d.department_name
		FROM teacher_course_allocation ca
		JOIN teachers t ON ca.teacher_id = t.id
		LEFT JOIN departments d ON t.dept = d.id
		WHERE ca.course_id = ?
		ORDER BY t.name
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		log.Printf("Error fetching course teachers: %v", err)
		http.Error(w, "Failed to fetch teachers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type CourseTeacher struct {
		ID             int     `json:"id"`
		TeacherID      int     `json:"teacher_id"`
		TeacherName    string  `json:"teacher_name"`
		Email          string  `json:"email"`
		DeptID         *int    `json:"dept_id"`
		DepartmentName *string `json:"department_name"`
	}

	var teachers []CourseTeacher
	for rows.Next() {
		var teacher CourseTeacher
		err := rows.Scan(&teacher.ID, &teacher.TeacherID, &teacher.TeacherName, &teacher.Email,
			&teacher.DeptID, &teacher.DepartmentName)
		if err != nil {
			log.Printf("Error scanning teacher row: %v", err)
			continue
		}
		teachers = append(teachers, teacher)
	}

	if teachers == nil {
		teachers = []CourseTeacher{}
	}

	json.NewEncoder(w).Encode(teachers)
}

// GetDepartmentSemesterCourses returns distinct courses for a department and semester.
func GetDepartmentSemesterCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	departmentID := vars["departmentId"]
	semester := vars["semester"]

	query := `
		SELECT DISTINCT c.id, c.course_code, c.course_name
		FROM teacher_course_allocation tca
		JOIN curriculum_courses cc ON tca.course_id = cc.course_id
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN courses c ON tca.course_id = c.id
		JOIN department_teachers dt ON dt.teacher_id = tca.teacher_id
		WHERE dt.department_id = ?
			AND dt.status = 1
			AND nc.semester_number = ?
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, semester)
	if err != nil {
		log.Printf("Error fetching department courses: %v", err)
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DepartmentCourse struct {
		CourseID   int    `json:"course_id"`
		CourseCode string `json:"course_code"`
		CourseName string `json:"course_name"`
	}

	var courses []DepartmentCourse
	for rows.Next() {
		var course DepartmentCourse
		if err := rows.Scan(&course.CourseID, &course.CourseCode, &course.CourseName); err != nil {
			log.Printf("Error scanning department course: %v", err)
			continue
		}
		courses = append(courses, course)
	}

	if courses == nil {
		courses = []DepartmentCourse{}
	}

	json.NewEncoder(w).Encode(courses)
}

// GetUnassignedCourses retrieves courses without teacher assignments
func GetUnassignedCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	semesterID := r.URL.Query().Get("semester_id")
	academicYear := r.URL.Query().Get("academic_year")

	if semesterID == "" || academicYear == "" {
		http.Error(w, "semester_id and academic_year are required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT c.id, c.course_code, c.course_name, ct.course_type, c.credit
		FROM courses c
		JOIN curriculum_courses cc ON c.id = cc.course_id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE cc.semester_id = ? AND c.status = 1
		AND NOT EXISTS (
			SELECT 1 FROM teacher_course_allocation ca
			WHERE ca.course_id = c.id
		)
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, semesterID)
	if err != nil {
		log.Printf("Error fetching unassigned courses: %v", err)
		http.Error(w, "Failed to fetch unassigned courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []models.CourseWithAllocations
	for rows.Next() {
		var c models.CourseWithAllocations
		err := rows.Scan(&c.CourseID, &c.CourseCode, &c.CourseName, &c.CourseType, &c.Credit)
		if err != nil {
			log.Printf("Error scanning course row: %v", err)
			continue
		}
		c.Allocations = []models.CourseAllocation{}
		courses = append(courses, c)
	}

	if courses == nil {
		courses = []models.CourseWithAllocations{}
	}

	json.NewEncoder(w).Encode(courses)
}

// GetAllocationSummary retrieves allocation summary statistics
func GetAllocationSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	semesterID := r.URL.Query().Get("semester_id")
	academicYear := r.URL.Query().Get("academic_year")

	if semesterID == "" || academicYear == "" {
		http.Error(w, "semester_id and academic_year are required", http.StatusBadRequest)
		return
	}

	type Summary struct {
		TotalCourses      int `json:"total_courses"`
		AssignedCourses   int `json:"assigned_courses"`
		UnassignedCourses int `json:"unassigned_courses"`
		TotalTeachers     int `json:"total_teachers"`
		ActiveTeachers    int `json:"active_teachers"`
	}

	var summary Summary

	// Total courses
	err := db.DB.QueryRow(`
		SELECT COUNT(DISTINCT c.id)
		FROM courses c
		JOIN curriculum_courses cc ON c.id = cc.course_id
		WHERE cc.semester_id = ? AND c.status = 1
	`, semesterID).Scan(&summary.TotalCourses)
	if err != nil {
		log.Printf("Error counting total courses: %v", err)
	}

	// Assigned courses
	err = db.DB.QueryRow(`
		SELECT COUNT(DISTINCT ca.course_id)
		FROM teacher_course_allocation ca
		JOIN curriculum_courses cc ON ca.course_id = cc.course_id
		WHERE cc.semester_id = ?
	`, semesterID).Scan(&summary.AssignedCourses)
	if err != nil {
		log.Printf("Error counting assigned courses: %v", err)
	}

	summary.UnassignedCourses = summary.TotalCourses - summary.AssignedCourses

	// Total teachers
	err = db.DB.QueryRow(`SELECT COUNT(*) FROM teachers WHERE status = 1`).Scan(&summary.TotalTeachers)
	if err != nil {
		log.Printf("Error counting total teachers: %v", err)
	}

	// Active teachers (assigned to at least one course in this semester)
	err = db.DB.QueryRow(`
		SELECT COUNT(DISTINCT ca.teacher_id)
		FROM teacher_course_allocation ca
		JOIN curriculum_courses cc ON ca.course_id = cc.course_id
		WHERE cc.semester_id = ?
	`, semesterID).Scan(&summary.ActiveTeachers)
	if err != nil {
		log.Printf("Error counting active teachers: %v", err)
	}

	json.NewEncoder(w).Encode(summary)
}
