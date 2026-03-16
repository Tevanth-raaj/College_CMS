package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"server/db"
	"server/models"
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

			if len(currentSemesters) > 0 {
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

				placeholders := strings.TrimRight(strings.Repeat("?,", len(currentSemesters)), ",")
				semWiseBaseQuery := `
					SELECT DISTINCT
						c.id, tch.course_id, c.course_code, c.course_name, COALESCE(ct.course_type, ''),
						c.credit, COALESCE(c.category, 'General'),
						COALESCE(tch.academic_year, ''), COALESCE(LOWER(TRIM(tch.semester_type)), '')
					FROM teacher_course_history tch
					JOIN courses c ON tch.course_id = c.id
					LEFT JOIN course_type ct ON c.course_type = ct.id
					WHERE tch.teacher_id = ?
					  AND tch.record_type = 'course'
					  AND tch.academic_year = ?
					  AND EXISTS (
						SELECT 1
						FROM curriculum_courses cc
						JOIN normal_cards nc ON cc.semester_id = nc.id
						WHERE cc.course_id = tch.course_id
						  AND nc.card_type = 'semester'
						  AND nc.status = 1
						  AND nc.semester_number IN (` + placeholders + `)
					  )
				`
				if derivedCalendarSemesterType != "" {
					semWiseBaseQuery += ` AND COALESCE(LOWER(TRIM(tch.semester_type)), '') = ?`
				}

				semWiseArgs := []interface{}{facultyID, currentAcademicYear}
				for _, sem := range currentSemesters {
					semWiseArgs = append(semWiseArgs, sem)
				}
				if derivedCalendarSemesterType != "" {
					semWiseArgs = append(semWiseArgs, derivedCalendarSemesterType)
				}

				var semWiseCount int
				countQuery := `SELECT COUNT(*) FROM (` + semWiseBaseQuery + `) q`
				countErr := db.DB.QueryRow(countQuery, semWiseArgs...).Scan(&semWiseCount)
				if countErr != nil {
					log.Printf("Teacher dashboard sem-wise count failed for faculty_id=%s, academic_year=%s: %v", facultyID, currentAcademicYear, countErr)
				} else if semWiseCount > 0 {
					finalQuery := semWiseBaseQuery + ` ORDER BY c.course_code`
					candidateRows, queryErr := db.DB.Query(finalQuery, semWiseArgs...)
					if queryErr != nil {
						log.Printf("Teacher dashboard sem-wise query failed for faculty_id=%s, academic_year=%s: %v", facultyID, currentAcademicYear, queryErr)
					} else {
						rows = candidateRows
						log.Printf("Teacher dashboard sem-wise mapping applied for faculty_id=%s, academic_year=%s, semesters=%v", facultyID, currentAcademicYear, currentSemesters)
					}
				}
			}
		}
	} else {
		log.Printf("Teacher dashboard academic_calendar year lookup unavailable: %v", calendarYearErr)
	}

	if rows == nil {
		log.Printf("No teacher history courses matched academic_calendar mapping for faculty_id='%s'", facultyID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	defer rows.Close()

	type StudentEnrollment struct {
		StudentID      int    `json:"student_id"`
		StudentName    string `json:"student_name"`
		LearningModeID *int   `json:"learning_mode_id"`
		DepartmentID   *int   `json:"department_id"`
		DepartmentName string `json:"department_name"`
	}

	type DepartmentInfo struct {
		DepartmentID   *int   `json:"department_id"`
		DepartmentName string `json:"department_name"`
		Semester       *int   `json:"semester"`
		CurriculumName string `json:"curriculum_name"`
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
		MappedSemesters           []int               `json:"mapped_semesters"`
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
			SELECT DISTINCT d.id, d.department_name, nc.semester_number, cur.name
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
				if err := deptRows.Scan(&depID, &depName, &depSem, &curName); err != nil {
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
				course.Departments = append(course.Departments, info)
			}
			deptRows.Close()
		}
		if course.Departments == nil {
			course.Departments = []DepartmentInfo{}
		}

		mappedSemSet := map[int]struct{}{}
		for _, dept := range course.Departments {
			if dept.Semester == nil {
				continue
			}
			if _, ok := activeSemesterMap[*dept.Semester]; ok {
				mappedSemSet[*dept.Semester] = struct{}{}
			}
		}
		if len(mappedSemSet) == 0 {
			for _, sem := range currentSemesters {
				mappedSemSet[sem] = struct{}{}
			}
		}
		course.MappedSemesters = make([]int, 0, len(mappedSemSet))
		for sem := range mappedSemSet {
			course.MappedSemesters = append(course.MappedSemesters, sem)
		}
		sort.Ints(course.MappedSemesters)

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

		// Fetch allocated students for this course and teacher
		studentQuery := `
			SELECT DISTINCT s.id, s.student_name, s.learning_mode_id, s.department_id, COALESCE(d.department_name, '')
			FROM course_student_teacher_allocation csta
			JOIN students s ON csta.student_id = s.id
			LEFT JOIN departments d ON s.department_id = d.id
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
				if err := sRows.Scan(&enrollment.StudentID, &enrollment.StudentName, &enrollment.LearningModeID, &enrollment.DepartmentID, &enrollment.DepartmentName); err != nil {
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

// resolveExpiredMarkEntryWindows finds expired mark entry windows for a teacher+course using
// the same department/semester matching logic as resolveMarkEntryWindow.
// Returns (missedWindow, submittedWindow, error).
// missedWindow: expired window with no submission by this teacher for this course.
// submittedWindow: expired window where the teacher submitted before expiry.
func resolveExpiredMarkEntryWindows(courseID int, facultyID string) (*MarkEntryWindow, *MarkEntryWindow, error) {
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
	semRows, err := db.DB.Query(`
		SELECT DISTINCT nc.semester_number
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		WHERE cc.course_id = ? AND COALESCE(nc.card_type, 'semester') = 'semester' AND nc.semester_number IS NOT NULL
	`, courseID)
	if err == nil {
		for semRows.Next() {
			var sem int64
			if semRows.Scan(&sem) == nil {
				courseSemesters = append(courseSemesters, sem)
			}
		}
		semRows.Close()
	}

	log.Printf("[resolveExpired] courseID=%d facultyID=%s teacherDepts=%v courseDepts=%v allDepts=%v sems=%v",
		courseID, facultyID, teacherDeptIDs, courseDeptIDs, allDeptIDs, courseSemesters)

	// 4. Build dynamic WHERE clauses (same pattern as resolveMarkEntryWindow)
	var queryArgs []interface{}
	queryArgs = append(queryArgs, facultyID, courseID)

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

	var semClause string
	if len(courseSemesters) == 0 {
		semClause = "1=1"
	} else {
		ph := make([]string, len(courseSemesters))
		for i, s := range courseSemesters {
			ph[i] = "?"
			queryArgs = append(queryArgs, s)
		}
		semClause = fmt.Sprintf("(w.semester IS NULL OR w.semester = 0 OR w.semester IN (%s))", strings.Join(ph, ","))
	}

	// Base: expired windows matching this course+teacher (ownership check via teacher_id/dept/semester)
	baseSQL := fmt.Sprintf(`
		SELECT w.id, w.start_at, w.end_at
		FROM mark_entry_windows w
		WHERE (w.teacher_id IS NULL OR w.teacher_id = ?)
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
		WHERE (w.teacher_id IS NULL OR w.teacher_id = ?)
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
