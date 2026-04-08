package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// MarkEntryWindow holds display info for an active or expired mark entry window.
type MarkEntryWindow struct {
	ID             int       `json:"id"`
	DepartmentName string    `json:"department_name"`
	Semester       *int      `json:"semester"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	ComponentNames []string  `json:"component_names"`
	SubmittedAt    string    `json:"submitted_at,omitempty"`
}

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

	// 2. Fetch all allocations for these courses from history table
	allocationQuery := `
		SELECT tch.id, tch.course_id, tch.teacher_id, t.name
		FROM teacher_course_history tch
		JOIN teachers t ON tch.teacher_id = t.faculty_id
		WHERE tch.record_type = 'course' AND tch.archived_at IS NULL
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

	var rows *sql.Rows
	var currentAcademicYear string
	currentSemesters := []int{}
	derivedCalendarSemesterType := ""
	calendarYearErr := db.DB.QueryRow(`
		SELECT academic_year
		FROM academic_calendar
		WHERE is_current = 1
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	`).Scan(&currentAcademicYear)

	if calendarYearErr == nil && strings.TrimSpace(currentAcademicYear) != "" {
		semesterRows, semesterErr := db.DB.Query(`
			SELECT DISTINCT current_semester
			FROM academic_calendar
			WHERE is_current = 1
			  AND academic_year = ?
			ORDER BY current_semester ASC
		`, currentAcademicYear)
		if semesterErr != nil {
			log.Printf("Teacher dashboard semester mapping lookup failed for academic_year=%s: %v", currentAcademicYear, semesterErr)
		} else {
			for semesterRows.Next() {
				var sem int
				if err := semesterRows.Scan(&sem); err == nil && sem > 0 {
					currentSemesters = append(currentSemesters, sem)
				}
			}
			semesterRows.Close()
		}
	}

	if len(currentSemesters) == 0 {
		log.Printf("Teacher dashboard semester mapping unavailable for faculty_id='%s', academic_year='%s' (continuing with teacher_course_history data)", facultyID, currentAcademicYear)
	}

	oddCount := 0
	evenCount := 0
	for _, sem := range currentSemesters {
		if sem%2 == 0 {
			evenCount++
		} else {
			oddCount++
		}
	}
	if oddCount > 0 && evenCount == 0 {
		derivedCalendarSemesterType = "odd"
	} else if evenCount > 0 && oddCount == 0 {
		derivedCalendarSemesterType = "even"
	}

	semWiseBaseQuery := `
		SELECT DISTINCT
			c.id, tca.course_id, c.course_code, c.course_name, COALESCE(ct.course_type, ''),
			c.credit, COALESCE(c.category, 'General'),
			COALESCE(tca.academic_year, ''), LOWER(COALESCE(tca.semester_type, ''))
		FROM teacher_course_history tca
		INNER JOIN (
			SELECT course_id, MAX(id) AS latest_id
			FROM teacher_course_history
			WHERE teacher_id = ?
			  AND record_type = 'course'
			  AND course_id IS NOT NULL
			GROUP BY course_id
		) latest ON latest.latest_id = tca.id
		JOIN courses c ON tca.course_id = c.id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE c.status = 1
	`

	semWiseArgs := []interface{}{facultyID}

	finalQuery := semWiseBaseQuery + ` ORDER BY c.course_code`
	candidateRows, queryErr := db.DB.Query(finalQuery, semWiseArgs...)
	if queryErr != nil {
		log.Printf("Teacher dashboard sem-wise query failed for faculty_id=%s, academic_year=%s: %v", facultyID, currentAcademicYear, queryErr)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	rows = candidateRows
	defer rows.Close()

	type StudentEnrollment struct {
		StudentID      int    `json:"student_id"`
		StudentName    string `json:"student_name"`
		EnrollmentNo   string `json:"enrollment_no"`
		RegisterNo     string `json:"register_no"`
		LearningModeID *int   `json:"learning_mode_id"`
		DepartmentID   *int   `json:"department_id"`
		DepartmentName string `json:"department_name"`
	}

	type DepartmentInfo struct {
		DepartmentID   *int   `json:"department_id"`
		DepartmentName string `json:"department_name"`
		Semester       *int   `json:"semester"`
		CurriculumName string `json:"curriculum_name"`
		CardType       string `json:"card_type"`
	}

	type TeacherCourse struct {
		ID                        int                 `json:"id"`
		CourseID                  int                 `json:"course_id"`
		CourseCode                string              `json:"course_code"`
		CourseName                string              `json:"course_name"`
		CourseType                string              `json:"course_type"`
		Credit                    int                 `json:"credit"`
		Category                  string              `json:"category"`
		AcademicYear              string              `json:"academic_year"`
		SemesterType              string              `json:"semester_type"`
		CurriculumID              *int                `json:"curriculum_id"`
		Semester                  *int                `json:"semester"`
		CurriculumName            string              `json:"curriculum_name"`
		DepartmentID              *int                `json:"department_id"`
		Departments               []DepartmentInfo    `json:"departments"`
		Enrollments               []StudentEnrollment `json:"enrollments"`
		HasWindow                 bool                `json:"has_window"`
		Window                    *MarkEntryWindow    `json:"window"`
		IsSubmitted               bool                `json:"is_submitted"`
		SubmittedAt               string              `json:"submitted_at,omitempty"`
		HasMissedSubmission       bool                `json:"has_missed_submission"`
		MissedWindow              *MarkEntryWindow    `json:"missed_window"`
		HasSubmittedExpiredWindow bool                `json:"has_submitted_expired_window"`
		SubmittedExpiredWindow    *MarkEntryWindow    `json:"submitted_expired_window"`
		MappedSemesters           []int               `json:"mapped_semesters"`
		MappedSemesterLabels      []string            `json:"mapped_semester_labels"`
	}

	var courses []TeacherCourse
	activeSemesterMap := make(map[int]struct{}, len(currentSemesters))
	for _, sem := range currentSemesters {
		activeSemesterMap[sem] = struct{}{}
	}
	log.Printf("Starting to iterate courses...")
	courseCount := 0
	for rows.Next() {
		var course TeacherCourse
		err := rows.Scan(&course.ID, &course.CourseID, &course.CourseCode, &course.CourseName,
			&course.CourseType, &course.Credit, &course.Category, &course.AcademicYear, &course.SemesterType)
		if err != nil {
			log.Printf("Error scanning course row: %v", err)
			continue
		}
		if strings.TrimSpace(course.SemesterType) == "" {
			course.SemesterType = derivedCalendarSemesterType
		}
		courseCount++
		log.Printf("Found course #%d: ID=%d, Code=%s, Name=%s", courseCount, course.CourseID, course.CourseCode, course.CourseName)

		deptQuery := `
			SELECT DISTINCT d.id, d.department_name, nc.semester_number, cur.name, COALESCE(nc.card_type, 'semester')
			FROM curriculum_courses cc
			JOIN normal_cards nc ON cc.semester_id = nc.id
			JOIN curriculum cur ON cc.curriculum_id = cur.id
			JOIN departments d ON d.current_curriculum_id = cur.id
			WHERE cc.course_id = ?
		`
		deptRows, deptErr := db.DB.Query(deptQuery, course.CourseID)
		if deptErr != nil {
			log.Printf("Error fetching departments for course %d: %v", course.CourseID, deptErr)
		} else {
			course.Departments = []DepartmentInfo{}
			for deptRows.Next() {
				var info DepartmentInfo
				var depID sql.NullInt64
				var depName sql.NullString
				var depSem sql.NullInt64
				var curName sql.NullString
				var cardType sql.NullString
				if err := deptRows.Scan(&depID, &depName, &depSem, &curName, &cardType); err != nil {
					continue
				}
				if depID.Valid {
					v := int(depID.Int64)
					info.DepartmentID = &v
				}
				if depName.Valid {
					info.DepartmentName = depName.String
				}
				if depSem.Valid {
					v := int(depSem.Int64)
					info.Semester = &v
				}
				if curName.Valid {
					info.CurriculumName = curName.String
				}
				if cardType.Valid {
					info.CardType = strings.ToLower(strings.TrimSpace(cardType.String))
				}
				course.Departments = append(course.Departments, info)
			}
			deptRows.Close()
		}
		if course.Departments == nil {
			course.Departments = []DepartmentInfo{}
		}

		mappedSemSet := map[int]struct{}{}
		mappedLabelSet := map[string]struct{}{}
		hasVerticalMapping := false
		for _, dept := range course.Departments {
			if dept.CardType == "vertical" {
				mappedLabelSet["vertical"] = struct{}{}
				hasVerticalMapping = true
				continue
			}

			if dept.Semester == nil {
				continue
			}
			if len(activeSemesterMap) == 0 {
				mappedSemSet[*dept.Semester] = struct{}{}
				mappedLabelSet[fmt.Sprintf("Sem %d", *dept.Semester)] = struct{}{}
				continue
			}
			if _, ok := activeSemesterMap[*dept.Semester]; ok {
				mappedSemSet[*dept.Semester] = struct{}{}
				mappedLabelSet[fmt.Sprintf("Sem %d", *dept.Semester)] = struct{}{}
			}
		}
		if !hasVerticalMapping && len(mappedSemSet) == 0 {
			for _, sem := range currentSemesters {
				mappedSemSet[sem] = struct{}{}
				mappedLabelSet[fmt.Sprintf("Sem %d", sem)] = struct{}{}
			}
		}
		if hasVerticalMapping {
			// For vertical courses, avoid mixed labels like "Sem 4, vertical".
			mappedSemSet = map[int]struct{}{}
			mappedLabelSet = map[string]struct{}{"vertical": {}}
		}
		if len(mappedLabelSet) == 0 {
			mappedLabelSet["Sem -"] = struct{}{}
		}
		course.MappedSemesters = make([]int, 0, len(mappedSemSet))
		for sem := range mappedSemSet {
			course.MappedSemesters = append(course.MappedSemesters, sem)
		}
		sort.Ints(course.MappedSemesters)
		course.MappedSemesterLabels = make([]string, 0, len(mappedLabelSet))
		for label := range mappedLabelSet {
			course.MappedSemesterLabels = append(course.MappedSemesterLabels, label)
		}
		sort.Slice(course.MappedSemesterLabels, func(i, j int) bool {
			a := course.MappedSemesterLabels[i]
			b := course.MappedSemesterLabels[j]
			if strings.HasPrefix(a, "Sem ") && strings.HasPrefix(b, "Sem ") {
				var ai, bi int
				if _, err := fmt.Sscanf(a, "Sem %d", &ai); err == nil {
					if _, err2 := fmt.Sscanf(b, "Sem %d", &bi); err2 == nil {
						return ai < bi
					}
				}
			}
			if strings.HasPrefix(a, "Sem ") {
				return true
			}
			if strings.HasPrefix(b, "Sem ") {
				return false
			}
			return a < b
		})

		// Check if there's an active mark entry window for this course
		windowOpen, windowIDs, windowComponents, err := resolveMarkEntryWindow(course.CourseID, facultyID)
		if err != nil {
			log.Printf("Error checking mark entry window for course %d: %v", course.CourseID, err)
		} else if windowOpen && len(windowIDs) > 0 {
			course.HasWindow = true

			// Fetch full window details from the first window (for display)
			windowID := windowIDs[0]
			var startAt, endAt time.Time
			var deptName sql.NullString
			var semester sql.NullInt64
			windowQuery := `
				SELECT w.start_at, w.end_at, d.department_name, w.semester
				FROM mark_entry_windows w
				LEFT JOIN departments d ON w.department_id = d.id
				WHERE w.id = ?
			`
			err := db.DB.QueryRow(windowQuery, windowID).Scan(&startAt, &endAt, &deptName, &semester)

			// If multiple windows, find the widest time range
			if err == nil && len(windowIDs) > 1 {
				for _, wid := range windowIDs[1:] {
					var s, e time.Time
					var dn sql.NullString
					var sem sql.NullInt64
					if err2 := db.DB.QueryRow(windowQuery, wid).Scan(&s, &e, &dn, &sem); err2 == nil {
						if s.Before(startAt) {
							startAt = s
						}
						if e.After(endAt) {
							endAt = e
						}
					}
				}
			}
			if err != nil {
				log.Printf("Error fetching window details for window %d: %v", windowID, err)
			} else {
				windowDetails := &MarkEntryWindow{
					ID:      windowID,
					StartAt: startAt,
					EndAt:   endAt,
				}
				if deptName.Valid {
					windowDetails.DepartmentName = deptName.String
				}
				if semester.Valid {
					sem := int(semester.Int64)
					windowDetails.Semester = &sem
				}

				// Fetch component names from mark_category_types
				if len(windowComponents) > 0 {
					componentNames := []string{}
					seenPT1 := false
					seenPT2 := false

					log.Printf("Processing %d window components for course %s", len(windowComponents), course.CourseCode)

					for _, compID := range windowComponents {
						var name string
						compQuery := `SELECT name FROM mark_category_types WHERE id = ?`
						err := db.DB.QueryRow(compQuery, compID).Scan(&name)
						if err == nil {
							log.Printf("Component ID %d: name = '%s'", compID, name)

							// Normalize for comparison
							nameUpper := strings.ToUpper(strings.TrimSpace(name))

							// Check for "Periodical Test 1" or "PT-1" or "PT - 1" or "PT 1"
							// Examples: "Periodical Test 1 -> CO - 1", "PT - 1 - CO 1", "PT-1-CO1", etc.
							isPT1 := strings.Contains(nameUpper, "PERIODICAL TEST 1") ||
								strings.Contains(nameUpper, "PERIODICALTEST1") ||
								(strings.Contains(nameUpper, "PERIODICAL") && strings.Contains(nameUpper, "TEST") && strings.Contains(nameUpper, "1") && !strings.Contains(nameUpper, "2"))

							// Also check for PT-1 format
							if !isPT1 {
								normalized := strings.ReplaceAll(nameUpper, " ", "")
								isPT1 = strings.HasPrefix(normalized, "PT-1") || strings.HasPrefix(normalized, "PT1")
							}

							if isPT1 {
								if !seenPT1 {
									componentNames = append(componentNames, "PT - 1")
									seenPT1 = true
									log.Printf("✓ Grouped as PT - 1")
								} else {
									log.Printf("✓ Skipped duplicate PT - 1")
								}
								continue
							}

							// Check for "Periodical Test 2" or "PT-2" or "PT - 2" or "PT 2"
							isPT2 := strings.Contains(nameUpper, "PERIODICAL TEST 2") ||
								strings.Contains(nameUpper, "PERIODICALTEST2") ||
								(strings.Contains(nameUpper, "PERIODICAL") && strings.Contains(nameUpper, "TEST") && strings.Contains(nameUpper, "2"))

							// Also check for PT-2 format
							if !isPT2 {
								normalized := strings.ReplaceAll(nameUpper, " ", "")
								isPT2 = strings.HasPrefix(normalized, "PT-2") || strings.HasPrefix(normalized, "PT2")
							}

							if isPT2 {
								if !seenPT2 {
									componentNames = append(componentNames, "PT - 2")
									seenPT2 = true
									log.Printf("✓ Grouped as PT - 2")
								} else {
									log.Printf("✓ Skipped duplicate PT - 2")
								}
								continue
							}

							// For all other components, show the full name
							componentNames = append(componentNames, name)
							log.Printf("→ Added full name: '%s'", name)
						} else {
							log.Printf("Error fetching component name for ID %d: %v", compID, err)
						}
					}

					log.Printf("Final component names for course %s: %v", course.CourseCode, componentNames)
					windowDetails.ComponentNames = componentNames
				} else {
					windowDetails.ComponentNames = []string{}
				}

				course.Window = windowDetails
				log.Printf("Window details for course %s: ID=%d, Start=%s, End=%s, Components=%v",
					course.CourseCode, windowID, startAt, endAt, windowDetails.ComponentNames)

				// Check if teacher has already submitted for this active window
				var subAt time.Time
				subErr := db.DB.QueryRow(`
					SELECT submitted_at FROM mark_submissions
					WHERE teacher_id = ? AND course_id = ? AND window_id = ?
					LIMIT 1
				`, facultyID, course.CourseID, windowID).Scan(&subAt)
				if subErr == nil {
					course.IsSubmitted = true
					course.SubmittedAt = subAt.Format(time.RFC3339)
				}
			}
		} else {
			course.HasWindow = false

			// Use the same dept+semester resolution logic as resolveMarkEntryWindow
			// to find expired windows (missed or submitted-then-expired).
			missedWin, submittedWin, resolveErr := resolveExpiredMarkEntryWindows(course.CourseID, facultyID)
			if resolveErr != nil {
				log.Printf("Error resolving expired windows for teacher=%s course=%d: %v", facultyID, course.CourseID, resolveErr)
			}
			if missedWin != nil {
				log.Printf("Missed window %d found for teacher=%s course=%d", missedWin.ID, facultyID, course.CourseID)
				course.HasMissedSubmission = true
				course.MissedWindow = missedWin
			}
			if submittedWin != nil {
				log.Printf("Submitted expired window %d found for teacher=%s course=%d", submittedWin.ID, facultyID, course.CourseID)
				course.HasSubmittedExpiredWindow = true
				course.SubmittedExpiredWindow = submittedWin
			}
			if missedWin == nil && submittedWin == nil {
				log.Printf("No expired windows found for teacher=%s course=%d", facultyID, course.CourseID)
			}
		}

		// Fetch all allocated students for this course and teacher.
		// Course details should always show the complete allocation list.
		studentQuery := `
			SELECT DISTINCT s.id, s.student_name, COALESCE(s.enrollment_no, ''), COALESCE(s.register_no, ''), s.learning_mode_id, s.department_id, COALESCE(d.department_name, '')
			FROM course_student_teacher_allocation csta
			JOIN students s ON csta.student_id = s.id
			LEFT JOIN departments d ON s.department_id = d.id
			WHERE csta.course_id = ? AND csta.teacher_id = ? AND s.status = 1
		`
		studentArgs := []interface{}{course.CourseID, facultyID}
		studentQuery += ` ORDER BY s.student_name`

		log.Printf("Querying students for courseID=%d, faculty_id='%s'", course.CourseID, facultyID)
		sRows, err := db.DB.Query(studentQuery, studentArgs...)
		if err != nil {
			log.Printf("Error fetching students for course %d: %v", course.CourseID, err)
		} else {
			defer sRows.Close()
			course.Enrollments = []StudentEnrollment{}
			studentCount := 0
			for sRows.Next() {
				studentCount++
				var enrollment StudentEnrollment
				if err := sRows.Scan(&enrollment.StudentID, &enrollment.StudentName, &enrollment.EnrollmentNo, &enrollment.RegisterNo, &enrollment.LearningModeID, &enrollment.DepartmentID, &enrollment.DepartmentName); err != nil {
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

		hasNonSemesterMapping := false
		for _, dept := range course.Departments {
			if strings.TrimSpace(dept.CardType) != "" && !strings.EqualFold(strings.TrimSpace(dept.CardType), "semester") {
				hasNonSemesterMapping = true
				break
			}
		}
		if !hasNonSemesterMapping {
			var nonSemesterCount int
			_ = db.DB.QueryRow(`
				SELECT COUNT(1)
				FROM curriculum_courses cc
				JOIN normal_cards nc ON cc.semester_id = nc.id
				WHERE cc.course_id = ?
				  AND LOWER(TRIM(COALESCE(nc.card_type, 'semester'))) <> 'semester'
			`, course.CourseID).Scan(&nonSemesterCount)
			hasNonSemesterMapping = nonSemesterCount > 0
		}

		// Mark-entry windows are semester-scoped. Keep course/student visibility intact,
		// but hide window actions when no allocated student is currently in the window semester.
		// Non-semester cards (vertical/elective/etc.) are semester-agnostic and must not be suppressed.
		if !hasNonSemesterMapping && course.HasWindow && course.Window != nil && course.Window.Semester != nil && *course.Window.Semester > 0 {
			var allocationEligibleCount int
			allocationEligibleErr := db.DB.QueryRow(`
				SELECT COUNT(DISTINCT s.id)
				FROM course_student_teacher_allocation csta
				JOIN students s ON csta.student_id = s.id
				LEFT JOIN academic_calendar ac ON ac.id = s.year
				WHERE csta.course_id = ?
				  AND csta.teacher_id = ?
				  AND s.status = 1
				  AND ac.current_semester = ?
			`, course.CourseID, facultyID, *course.Window.Semester).Scan(&allocationEligibleCount)

			var assignedEligibleCount int
			assignedEligibleErr := db.DB.QueryRow(`
				SELECT COUNT(DISTINCT s.id)
				FROM mark_entry_student_permissions mesp
				JOIN mark_entry_windows w ON w.id = mesp.window_id
				JOIN students s ON s.id = mesp.student_id
				LEFT JOIN academic_calendar ac ON ac.id = s.year
				LEFT JOIN student_courses sc ON sc.student_id = s.id
				WHERE mesp.window_id = ?
				  AND s.status = 1
				  AND (
					w.course_id IS NULL OR w.course_id = 0
					OR w.course_id = ?
					OR sc.course_id = ?
				  )
				  AND (
					COALESCE(w.semester, 0) = 0
					OR ac.current_semester = w.semester
				  )
			`, course.Window.ID, course.CourseID, course.CourseID).Scan(&assignedEligibleCount)

			if allocationEligibleErr != nil {
				log.Printf("Error checking allocation semester-eligible students for course %d window %d: %v", course.CourseID, course.Window.ID, allocationEligibleErr)
			}
			if assignedEligibleErr != nil {
				log.Printf("Error checking window-assigned semester-eligible students for course %d window %d: %v", course.CourseID, course.Window.ID, assignedEligibleErr)
			}

			if allocationEligibleErr == nil && assignedEligibleErr == nil && allocationEligibleCount == 0 && assignedEligibleCount == 0 {
				log.Printf("Suppressing mark entry window for course %d window %d: no students in semester %d", course.CourseID, course.Window.ID, *course.Window.Semester)
				course.HasWindow = false
				course.Window = nil
				course.IsSubmitted = false
				course.SubmittedAt = ""
			}
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

// resolveExpiredMarkEntryWindows finds expired mark entry windows for a teacher+course using
// the same department/semester matching logic as resolveMarkEntryWindow.
// Returns (missedWindow, submittedWindow, error).
// missedWindow: expired window with no submission by this teacher for this course.
// submittedWindow: expired window where the teacher submitted before expiry.
func resolveExpiredMarkEntryWindows(courseID int, facultyID string) (*MarkEntryWindow, *MarkEntryWindow, error) {
	var numericUserID sql.NullInt64
	if resolvedUserID, userErr := resolveNumericUserID(db.DB, facultyID); userErr == nil && resolvedUserID > 0 {
		numericUserID = sql.NullInt64{Int64: int64(resolvedUserID), Valid: true}
	}

	// 1. Teacher's departments from department_teachers
	var teacherDeptIDs []int64
	tRows, err := db.DB.Query(`SELECT department_id FROM department_teachers WHERE teacher_id = ? AND status = 1`, facultyID)
	if err == nil {
		for tRows.Next() {
			var did int64
			if tRows.Scan(&did) == nil {
				teacherDeptIDs = append(teacherDeptIDs, did)
			}
		}
		tRows.Close()
	}

	// 2. Course's departments via curriculum chain (same as resolveMarkEntryWindow)
	var courseDeptIDs []int64
	cRows, err := db.DB.Query(`
		SELECT DISTINCT d.id
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN departments d ON d.current_curriculum_id = nc.curriculum_id
		WHERE cc.course_id = ?
		  AND d.id IS NOT NULL
	`, courseID)
	if err == nil {
		for cRows.Next() {
			var did int64
			if cRows.Scan(&did) == nil {
				courseDeptIDs = append(courseDeptIDs, did)
			}
		}
		cRows.Close()
	}

	// Merge + deduplicate department IDs
	deptSeen := make(map[int64]bool)
	var allDeptIDs []int64
	for _, did := range append(courseDeptIDs, teacherDeptIDs...) {
		if !deptSeen[did] {
			allDeptIDs = append(allDeptIDs, did)
			deptSeen[did] = true
		}
	}

	// 3. Course semesters via curriculum chain
	var courseSemesters []int64
	studentSemesterSet := make(map[int64]bool)
	var hasNonSemesterCard bool
	var hasSemesterCard bool
	semRows, err := db.DB.Query(`
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
				if !strings.EqualFold(strings.TrimSpace(cType), "semester") {
					hasNonSemesterCard = true
				} else {
					hasSemesterCard = true
					if semNum.Valid {
						courseSemesters = append(courseSemesters, semNum.Int64)
					}
				}
			}
		}
		semRows.Close()
	}

	studentSemRows, studentSemErr := db.DB.Query(`
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
			if studentSemRows.Scan(&currentSem) == nil && currentSem > 0 {
				studentSemesterSet[currentSem] = true
			}
		}
		studentSemRows.Close()
	}

	for sem := range studentSemesterSet {
		courseSemesters = append(courseSemesters, sem)
	}

	log.Printf("[resolveExpired] courseID=%d facultyID=%s teacherDepts=%v courseDepts=%v allDepts=%v sems=%v",
		courseID, facultyID, teacherDeptIDs, courseDeptIDs, allDeptIDs, courseSemesters)

	// 4. Build dynamic WHERE clauses (same pattern as resolveMarkEntryWindow)
	var queryArgs []interface{}
	userIDValue := interface{}(nil)
	if numericUserID.Valid {
		userIDValue = numericUserID.Int64
	}
	queryArgs = append(queryArgs, facultyID, userIDValue, courseID)

	var deptClause string
	if len(allDeptIDs) == 0 {
		deptClause = "1=1"
	} else {
		ph := make([]string, len(allDeptIDs))
		for i, did := range allDeptIDs {
			ph[i] = "?"
			queryArgs = append(queryArgs, did)
		}
		deptClause = fmt.Sprintf("(w.department_id IS NULL OR w.department_id = 0 OR w.department_id IN (%s))", strings.Join(ph, ","))
	}

	var semesterOnlyClause string
	if !hasSemesterCard && hasNonSemesterCard {
		semesterOnlyClause = "(w.semester IS NULL OR w.semester = 0)"
	} else if len(courseSemesters) == 0 {
		semesterOnlyClause = "(w.semester IS NULL OR w.semester = 0)"
	} else {
		ph := make([]string, len(courseSemesters))
		for i, s := range courseSemesters {
			ph[i] = "?"
			queryArgs = append(queryArgs, s)
		}
		semesterOnlyClause = fmt.Sprintf("(w.semester IS NULL OR w.semester = 0 OR w.semester IN (%s))", strings.Join(ph, ","))
	}

	nonSemesterByDeptClause := "0=1"
	if hasNonSemesterCard {
		nonSemesterByDeptClause = `EXISTS (
			SELECT 1
			FROM curriculum_courses ccx
			JOIN normal_cards ncx ON ccx.semester_id = ncx.id
			JOIN departments dx ON dx.current_curriculum_id = ncx.curriculum_id
			WHERE ccx.course_id = ?
			  AND LOWER(COALESCE(ncx.card_type, 'semester')) <> 'semester'
			  AND (COALESCE(w.department_id, 0) = 0 OR dx.id = w.department_id)
			  AND (
				COALESCE(w.semester, 0) = 0
				OR EXISTS (
					SELECT 1
					FROM course_student_teacher_allocation csta_val
					JOIN students s_val ON csta_val.student_id = s_val.id
					LEFT JOIN academic_calendar ac_val ON ac_val.id = s_val.year
					WHERE csta_val.course_id = ?
					  AND csta_val.teacher_id = ?
					  AND s_val.status = 1
					  AND COALESCE(ac_val.current_semester, 0) = w.semester
				)
			  )
		)`
		queryArgs = append(queryArgs, courseID, courseID, facultyID)
	}
	semClause := fmt.Sprintf("(%s OR %s)", semesterOnlyClause, nonSemesterByDeptClause)

	// Base: expired windows matching this course+teacher (ownership check via teacher_id/dept/semester)
	baseSQL := fmt.Sprintf(`
		SELECT w.id, w.start_at, w.end_at
		FROM mark_entry_windows w
		WHERE ((w.teacher_id IS NULL AND w.user_id IS NULL) OR w.teacher_id = ? OR w.user_id = ?)
		  AND (w.course_id IS NULL OR w.course_id = ?)
		  AND %s
		  AND %s
		  AND w.enabled = 1
		  AND w.end_at < NOW()
	`, deptClause, semClause)

	// 5. Check for a submission for any matching expired window
	submittedArgs := make([]interface{}, len(queryArgs))
	copy(submittedArgs, queryArgs)
	submittedArgs = append(submittedArgs, facultyID, courseID)
	submittedSQL := fmt.Sprintf(`
		SELECT w.id, w.start_at, w.end_at, ms.submitted_at
		FROM mark_submissions ms
		JOIN mark_entry_windows w ON w.id = ms.window_id
		WHERE ((w.teacher_id IS NULL AND w.user_id IS NULL) OR w.teacher_id = ? OR w.user_id = ?)
		  AND (w.course_id IS NULL OR w.course_id = ?)
		  AND %s
		  AND %s
		  AND w.enabled = 1
		  AND w.end_at < NOW()
		  AND ms.teacher_id = ?
		  AND ms.course_id = ?
		ORDER BY w.end_at DESC
		LIMIT 1
	`, deptClause, semClause)

	var submittedWin *MarkEntryWindow
	var swID int
	var swStart, swEnd, swSubAt time.Time
	if err := db.DB.QueryRow(submittedSQL, submittedArgs...).Scan(&swID, &swStart, &swEnd, &swSubAt); err == nil {
		submittedWin = &MarkEntryWindow{
			ID:          swID,
			StartAt:     swStart,
			EndAt:       swEnd,
			SubmittedAt: swSubAt.Format(time.RFC3339),
		}
		log.Printf("[resolveExpired] submitted window %d found for teacher=%s course=%d", swID, facultyID, courseID)
	} else if err != sql.ErrNoRows {
		log.Printf("[resolveExpired] submitted query error for teacher=%s course=%d: %v", facultyID, courseID, err)
	}

	// 6. Check for missed window (expired with no submission)
	// Exclude windows that have a submission (handled above).
	missedArgs := make([]interface{}, len(queryArgs))
	copy(missedArgs, queryArgs)
	missedArgs = append(missedArgs, facultyID, courseID)
	missedSQL := baseSQL + `
		  AND NOT EXISTS (
			SELECT 1 FROM mark_submissions ms2
			WHERE ms2.teacher_id = ? AND ms2.course_id = ? AND ms2.window_id = w.id
		  )
		ORDER BY w.end_at DESC
		LIMIT 1
	`
	var missedWin *MarkEntryWindow
	var mwID int
	var mwStart, mwEnd time.Time
	if err := db.DB.QueryRow(missedSQL, missedArgs...).Scan(&mwID, &mwStart, &mwEnd); err == nil {
		missedWin = &MarkEntryWindow{
			ID:      mwID,
			StartAt: mwStart,
			EndAt:   mwEnd,
		}
		log.Printf("[resolveExpired] missed window %d found for teacher=%s course=%d", mwID, facultyID, courseID)
	} else if err != sql.ErrNoRows {
		log.Printf("[resolveExpired] missed query error for teacher=%s course=%d: %v", facultyID, courseID, err)
	}

	return missedWin, submittedWin, nil
}

// GetCourseTeachers retrieves all teachers assigned to a specific course
func GetCourseTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	courseID := vars["id"]

	query := `
		SELECT 
			tch.id, tch.teacher_id, t.name, t.email, t.dept, d.department_name
		FROM teacher_course_history tch
		JOIN teachers t ON tch.teacher_id = t.faculty_id
		LEFT JOIN departments d ON t.dept = d.id
		WHERE tch.course_id = ?
		  AND tch.record_type = 'course'
		  AND tch.archived_at IS NULL
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

	var err error
	academicYear := strings.TrimSpace(r.URL.Query().Get("academic_year"))
	if academicYear == "" {
		err = db.DB.QueryRow(`
			SELECT academic_year
			FROM academic_calendar
			WHERE is_current = 1
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		`).Scan(&academicYear)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Warning: could not resolve current academic year: %v", err)
			}
			academicYear = ""
		}
	}

	query := `
		SELECT DISTINCT c.id, c.course_code, c.course_name
		FROM teacher_course_history tch
		JOIN curriculum_courses cc ON tch.course_id = cc.course_id
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN courses c ON tch.course_id = c.id
		JOIN department_teachers dt ON dt.teacher_id = tch.teacher_id
		WHERE dt.department_id = ?
			AND dt.status = 1
			AND nc.semester_number = ?
			AND tch.record_type = 'course'
			AND tch.archived_at IS NULL
		UNION
		SELECT DISTINCT c.id, c.course_code, c.course_name
		FROM courses c
		JOIN hod_elective_selections hes ON c.id = hes.course_id
		JOIN student_elective_choices sec ON sec.hod_selection_id = hes.id
		WHERE hes.department_id = ?
			AND sec.semester = ?
			AND sec.academic_year = ?
			AND hes.status = 'ACTIVE'
			AND (c.status = 1 OR c.status IS NULL)
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, semester, departmentID, semester, academicYear)
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
			SELECT 1 FROM teacher_course_history tch
			WHERE tch.course_id = c.id
			  AND tch.record_type = 'course'
			  AND tch.archived_at IS NULL
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
		SELECT COUNT(DISTINCT tch.course_id)
		FROM teacher_course_history tch
		JOIN curriculum_courses cc ON tch.course_id = cc.course_id
		WHERE cc.semester_id = ?
		  AND tch.record_type = 'course'
		  AND tch.archived_at IS NULL
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
		SELECT COUNT(DISTINCT tch.teacher_id)
		FROM teacher_course_history tch
		JOIN curriculum_courses cc ON tch.course_id = cc.course_id
		WHERE cc.semester_id = ?
		  AND tch.record_type = 'course'
		  AND tch.archived_at IS NULL
	`, semesterID).Scan(&summary.ActiveTeachers)
	if err != nil {
		log.Printf("Error counting active teachers: %v", err)
	}

	json.NewEncoder(w).Encode(summary)
}
