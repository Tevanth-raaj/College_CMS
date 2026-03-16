package allocation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"server/db"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// StudentAllocationMapping represents a student-teacher-course allocation
type StudentAllocationMapping struct {
	StudentID      int    `json:"student_id"`
	StudentName    string `json:"student_name"`
	CourseID       int    `json:"course_id"`
	TeacherID      string `json:"teacher_id"`
	LearningModeID int    `json:"learning_mode_id"`
}

// ExistingAllocation represents current teacher allocations
type ExistingAllocation struct {
	TeacherID      string
	LearningModeID int
	StudentCount   int
}

// TeacherLoad for sorting by current workload
type TeacherLoad struct {
	TeacherID string
	Load      int
}

// CourseAllocationResult represents the result for a single course
type CourseAllocationResult struct {
	CourseID         int                      `json:"course_id"`
	CourseName       string                   `json:"course_name"`
	Success          bool                     `json:"success"`
	AllocatedCount   int                      `json:"allocated_count"`
	UALCount         int                      `json:"ual_count"`
	PBLCount         int                      `json:"pbl_count"`
	TeachersUsed     int                      `json:"teachers_used"`
	Error            string                   `json:"error,omitempty"`
	ErrorCode        string                   `json:"error_code,omitempty"`
	StudentsWithNull []StudentNullModeWarning `json:"students_with_null_mode,omitempty"`
	ReallocationInfo *ReallocationInfo        `json:"reallocation_info,omitempty"`
}

// StudentNullModeWarning represents students without learning mode
type StudentNullModeWarning struct {
	StudentID      int    `json:"student_id"`
	StudentName    string `json:"student_name"`
	EnrollmentNo   string `json:"enrollment_no"`
	RegisterNo     string `json:"register_no"`
	DefaultedToUAL bool   `json:"defaulted_to_ual"`
}

// ReallocationInfo tracks inactive teacher reallocations
type ReallocationInfo struct {
	InactiveTeachers  []InactiveTeacherInfo `json:"inactive_teachers"`
	StudentsRelocated int                   `json:"students_relocated"`
}

// InactiveTeacherInfo represents info about inactive teacher
type InactiveTeacherInfo struct {
	TeacherID    string `json:"teacher_id"`
	TeacherName  string `json:"teacher_name"`
	StudentCount int    `json:"student_count"`
}

// AllocationSummary represents the overall allocation result
type AllocationSummary struct {
	Success                bool                     `json:"success"`
	TotalCourses           int                      `json:"total_courses"`
	SuccessfulCourses      int                      `json:"successful_courses"`
	FailedCourses          int                      `json:"failed_courses"`
	TotalStudentsAllocated int                      `json:"total_students_allocated"`
	ExecutionTime          float64                  `json:"execution_time_seconds"`
	CourseResults          []CourseAllocationResult `json:"course_results"`
	Timestamp              time.Time                `json:"timestamp"`
}

// AllocateStudentsToTeachers - Main endpoint for full student allocation
func AllocateStudentsToTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	forceRebalance := parseForceRebalance(r)

	startTime := time.Now()
	log.Println("🎓 STARTING STUDENT-TEACHER-COURSE ALLOCATION")

	// Get all courses that have teachers allocated
	courses, err := getCoursesWithTeachers()
	if err != nil {
		log.Printf("❌ Error fetching courses: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to fetch courses",
		})
		return
	}

	log.Printf("📚 Found %d courses with teacher allocations", len(courses))

	summary := AllocationSummary{
		Timestamp:     time.Now(),
		TotalCourses:  len(courses),
		CourseResults: []CourseAllocationResult{},
	}

	// Process each course
	for _, course := range courses {
		log.Printf("\n📖 Processing Course: %s (ID: %d)", course.CourseName, course.CourseID)

		result := allocateCourse(course.CourseID, course.CourseName, forceRebalance)
		summary.CourseResults = append(summary.CourseResults, result)

		if result.Success {
			summary.SuccessfulCourses++
			summary.TotalStudentsAllocated += result.AllocatedCount
		} else {
			summary.FailedCourses++
		}
	}

	summary.Success = summary.FailedCourses == 0
	summary.ExecutionTime = time.Since(startTime).Seconds()

	log.Printf("\n✅ ALLOCATION COMPLETE")
	log.Printf("   Total Courses: %d", summary.TotalCourses)
	log.Printf("   Successful: %d", summary.SuccessfulCourses)
	log.Printf("   Failed: %d", summary.FailedCourses)
	log.Printf("   Total Students Allocated: %d", summary.TotalStudentsAllocated)
	log.Printf("   Execution Time: %.2f seconds\n", summary.ExecutionTime)

	json.NewEncoder(w).Encode(summary)
}

// AllocateSingleCourse - Endpoint to allocate/reallocate a specific course
func AllocateSingleCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	forceRebalance := parseForceRebalance(r)

	vars := mux.Vars(r)
	courseIDStr := vars["course_id"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid course ID",
		})
		return
	}

	// Get course name
	var courseName string
	err = db.DB.QueryRow("SELECT course_name FROM courses WHERE id = ?", courseID).Scan(&courseName)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Course not found",
		})
		return
	}

	log.Printf("🎓 Allocating single course: %s (ID: %d)", courseName, courseID)

	result := allocateCourse(courseID, courseName, forceRebalance)
	json.NewEncoder(w).Encode(result)
}

// allocateCourse - Core allocation logic for a single course
func allocateCourse(courseID int, courseName string, forceRebalance bool) CourseAllocationResult {
	result := CourseAllocationResult{
		CourseID:   courseID,
		CourseName: courseName,
		Success:    false,
	}

	// Step 1: Check for students with NULL learning_mode_id
	nullModeStudents, err := checkNullLearningMode(courseID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to check learning modes: %v", err)
		result.ErrorCode = "VALIDATION_ERROR"
		return result
	}

	if len(nullModeStudents) > 0 {
		log.Printf("⚠️  Found %d students with NULL learning_mode_id", len(nullModeStudents))
		result.StudentsWithNull = nullModeStudents
		result.Error = fmt.Sprintf("%d students have NULL learning_mode_id. Please update student records.", len(nullModeStudents))
		result.ErrorCode = "NULL_LEARNING_MODE"
		return result
	}

	// Step 2: Get available teachers for this course
	availableTeachers, err := getActiveTeachersForCourse(courseID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to fetch teachers: %v", err)
		result.ErrorCode = "DATABASE_ERROR"
		return result
	}

	if len(availableTeachers) == 0 {
		log.Printf("❌ No teachers allocated for course %d", courseID)
		result.Error = "teacher_course_allocation is empty for this course. Please allocate teachers first."
		result.ErrorCode = "NO_TEACHERS_ALLOCATED"
		return result
	}

	log.Printf("👨‍🏫 Found %d active teachers", len(availableTeachers))

	// Step 3: Handle inactive teachers - reallocate their students
	reallocationInfo, err := handleInactiveTeachers(courseID)
	if err != nil {
		log.Printf("⚠️  Error handling inactive teachers: %v", err)
	} else if reallocationInfo != nil && len(reallocationInfo.InactiveTeachers) > 0 {
		log.Printf("♻️  Reallocated %d students from %d inactive teachers",
			reallocationInfo.StudentsRelocated, len(reallocationInfo.InactiveTeachers))
		result.ReallocationInfo = reallocationInfo
	}

	// Step 4: Get existing allocations
	existingAllocations, err := getExistingAllocations(courseID)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to fetch existing allocations: %v", err)
		result.ErrorCode = "DATABASE_ERROR"
		return result
	}

	// Step 5: Get unallocated students by learning mode
	unallocatedUAL, err := getUnallocatedStudents(courseID, 1)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to fetch UAL students: %v", err)
		result.ErrorCode = "DATABASE_ERROR"
		return result
	}

	unallocatedPBL, err := getUnallocatedStudents(courseID, 2)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to fetch PBL students: %v", err)
		result.ErrorCode = "DATABASE_ERROR"
		return result
	}

	log.Printf("📊 Unallocated students: %d UAL, %d PBL", len(unallocatedUAL), len(unallocatedPBL))

	// Step 5b: Optionally rebalance fully allocated courses.
	// This handles the common case where new teachers are added to an already fully allocated course.
	if shouldRebalanceCourse(forceRebalance, availableTeachers, existingAllocations, len(unallocatedUAL), len(unallocatedPBL)) {
		log.Printf("♻️  Rebalancing course %d (force=%v)", courseID, forceRebalance)

		if err := deactivateCourseAllocations(courseID); err != nil {
			result.Error = fmt.Sprintf("Failed to reset course allocations for rebalance: %v", err)
			result.ErrorCode = "DATABASE_ERROR"
			return result
		}

		// Refresh allocation context after reset.
		existingAllocations = []ExistingAllocation{}

		unallocatedUAL, err = getUnallocatedStudents(courseID, 1)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to fetch UAL students after rebalance reset: %v", err)
			result.ErrorCode = "DATABASE_ERROR"
			return result
		}

		unallocatedPBL, err = getUnallocatedStudents(courseID, 2)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to fetch PBL students after rebalance reset: %v", err)
			result.ErrorCode = "DATABASE_ERROR"
			return result
		}

		log.Printf("📊 Rebalance reset complete: %d UAL, %d PBL now unallocated", len(unallocatedUAL), len(unallocatedPBL))
	}

	// If no students to allocate, return success
	if len(unallocatedUAL) == 0 && len(unallocatedPBL) == 0 {
		log.Printf("✅ No unallocated students for course %d", courseID)
		result.Success = true
		result.AllocatedCount = 0
		return result
	}

	// Step 6: Check if we need both modes but only have 1 teacher
	needsBothModes := len(unallocatedUAL) > 0 && len(unallocatedPBL) > 0
	if needsBothModes && len(availableTeachers) < 2 {
		log.Printf("❌ Insufficient teachers: need 2 (UAL + PBL) but only have %d", len(availableTeachers))
		result.Error = fmt.Sprintf("Insufficient teachers. Course needs both UAL and PBL teachers but only %d teacher(s) available. Please add at least 1 more teacher to this course.", len(availableTeachers))
		result.ErrorCode = "INSUFFICIENT_TEACHERS"
		result.UALCount = len(unallocatedUAL)
		result.PBLCount = len(unallocatedPBL)
		return result
	}

	// Step 7: Build teacher mode mapping from existing allocations
	teacherModes := make(map[string]int) // teacher_id -> learning_mode_id
	teacherLoads := make(map[string]int) // teacher_id -> current student count

	for _, alloc := range existingAllocations {
		teacherModes[alloc.TeacherID] = alloc.LearningModeID
		teacherLoads[alloc.TeacherID] = alloc.StudentCount
	}

	// Step 8: Separate teachers by availability
	availableForUAL := []string{}
	availableForPBL := []string{}
	freeTeachers := []string{}

	for _, teacher := range availableTeachers {
		mode, hasAssignment := teacherModes[teacher]
		if !hasAssignment {
			freeTeachers = append(freeTeachers, teacher)
		} else if mode == 1 {
			availableForUAL = append(availableForUAL, teacher)
		} else if mode == 2 {
			availableForPBL = append(availableForPBL, teacher)
		}
	}

	log.Printf("👥 Teacher availability: %d free, %d UAL-locked, %d PBL-locked",
		len(freeTeachers), len(availableForUAL)-len(freeTeachers), len(availableForPBL)-len(freeTeachers))

	// Step 9: Assign free teachers to modes based on student ratios
	if len(freeTeachers) > 0 {
		totalUnallocated := len(unallocatedUAL) + len(unallocatedPBL)
		if totalUnallocated > 0 {
			ualRatio := float64(len(unallocatedUAL)) / float64(totalUnallocated)
			ualTeachersNeeded := int(math.Ceil(ualRatio * float64(len(freeTeachers))))

			// Ensure at least 1 teacher per mode if students exist
			if len(unallocatedUAL) > 0 && ualTeachersNeeded == 0 {
				ualTeachersNeeded = 1
			}
			if len(unallocatedPBL) > 0 && (len(freeTeachers)-ualTeachersNeeded) == 0 {
				ualTeachersNeeded = len(freeTeachers) - 1
			}

			log.Printf("🎯 Assigning %d free teachers: %d to UAL, %d to PBL",
				len(freeTeachers), ualTeachersNeeded, len(freeTeachers)-ualTeachersNeeded)

			for i, teacher := range freeTeachers {
				if i < ualTeachersNeeded {
					availableForUAL = append(availableForUAL, teacher)
				} else {
					availableForPBL = append(availableForPBL, teacher)
				}
			}
		}
	}

	// Step 10: Validate sufficient teachers
	// If one mode is starved (common after partial manual deletes), reset and reallocate this course.
	if len(existingAllocations) > 0 {
		modeStarved := (len(unallocatedUAL) > 0 && len(availableForUAL) == 0) || (len(unallocatedPBL) > 0 && len(availableForPBL) == 0)
		if modeStarved {
			log.Printf("♻️  Mode starvation detected for course %d; resetting active allocations and retrying", courseID)
			if err := deactivateCourseAllocations(courseID); err != nil {
				result.Error = fmt.Sprintf("Failed to reset allocations after mode starvation: %v", err)
				result.ErrorCode = "DATABASE_ERROR"
				return result
			}

			return allocateCourse(courseID, courseName, false)
		}
	}

	if len(unallocatedUAL) > 0 && len(availableForUAL) == 0 {
		result.Error = "No teachers available for UAL students. Please allocate more teachers to this course."
		result.ErrorCode = "INSUFFICIENT_TEACHERS"
		return result
	}
	if len(unallocatedPBL) > 0 && len(availableForPBL) == 0 {
		result.Error = "No teachers available for PBL students. Please allocate more teachers to this course."
		result.ErrorCode = "INSUFFICIENT_TEACHERS"
		return result
	}

	// Step 11: Distribute students
	allMappings := []StudentAllocationMapping{}

	if len(unallocatedUAL) > 0 {
		ualMappings := distributeStudentsToTeachers(unallocatedUAL, availableForUAL, teacherLoads, courseID, 1)
		allMappings = append(allMappings, ualMappings...)
		log.Printf("✅ Distributed %d UAL students to %d teachers", len(unallocatedUAL), len(availableForUAL))
	}

	if len(unallocatedPBL) > 0 {
		pblMappings := distributeStudentsToTeachers(unallocatedPBL, availableForPBL, teacherLoads, courseID, 2)
		allMappings = append(allMappings, pblMappings...)
		log.Printf("✅ Distributed %d PBL students to %d teachers", len(unallocatedPBL), len(availableForPBL))
	}

	// Step 12: Insert mappings into database
	if len(allMappings) > 0 {
		err = insertAllocations(allMappings)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to save allocations: %v", err)
			result.ErrorCode = "DATABASE_ERROR"
			return result
		}
	}

	// Success!
	result.Success = true
	result.AllocatedCount = len(allMappings)
	result.UALCount = len(unallocatedUAL)
	result.PBLCount = len(unallocatedPBL)
	result.TeachersUsed = len(availableTeachers)

	return result
}

// Helper Functions

// getCourseInfo represents course basic info
type CourseInfo struct {
	CourseID   int
	CourseName string
}

// getCoursesWithTeachers fetches all courses that have teachers allocated
func getCoursesWithTeachers() ([]CourseInfo, error) {
	query := `
		SELECT DISTINCT c.id, c.course_name
		FROM courses c
		INNER JOIN teacher_course_history tca ON c.id = tca.course_id
		WHERE tca.record_type = 'course'
			AND tca.archived_at IS NULL
		ORDER BY c.course_name
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := []CourseInfo{}
	for rows.Next() {
		var course CourseInfo
		if err := rows.Scan(&course.CourseID, &course.CourseName); err != nil {
			continue
		}
		courses = append(courses, course)
	}

	return courses, nil
}

// checkNullLearningMode finds students with NULL learning_mode_id
func checkNullLearningMode(courseID int) ([]StudentNullModeWarning, error) {
	query := `
		SELECT 
			s.id,
			s.student_name,
			COALESCE(s.enrollment_no, '') as enrollment_no,
			COALESCE(s.register_no, '') as register_no
		FROM students s
		INNER JOIN student_courses sc ON s.id = sc.student_id
		WHERE sc.course_id = ?
			AND s.status = 1
			AND (s.learning_mode_id IS NULL OR s.learning_mode_id NOT IN (1, 2))
		ORDER BY s.student_name
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []StudentNullModeWarning{}
	for rows.Next() {
		var student StudentNullModeWarning
		if err := rows.Scan(&student.StudentID, &student.StudentName, &student.EnrollmentNo, &student.RegisterNo); err != nil {
			continue
		}
		students = append(students, student)
	}

	return students, nil
}

// getActiveTeachersForCourse gets all active teachers allocated to a course
func getActiveTeachersForCourse(courseID int) ([]string, error) {
	query := `
		SELECT DISTINCT tca.teacher_id
		FROM teacher_course_history tca
		INNER JOIN teachers t ON tca.teacher_id = t.faculty_id
		WHERE tca.course_id = ?
			AND tca.record_type = 'course'
			AND tca.archived_at IS NULL
			AND t.status = 1
		ORDER BY tca.teacher_id
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teachers := []string{}
	for rows.Next() {
		var teacherID string
		if err := rows.Scan(&teacherID); err != nil {
			continue
		}
		teachers = append(teachers, teacherID)
	}

	return teachers, nil
}

// handleInactiveTeachers finds and reallocates students from inactive teachers
func handleInactiveTeachers(courseID int) (*ReallocationInfo, error) {
	// Find inactive teachers with student allocations
	query := `
		SELECT 
			csta.teacher_id,
			t.name,
			COUNT(*) as student_count
		FROM course_student_teacher_allocation csta
		LEFT JOIN teachers t ON csta.teacher_id = t.faculty_id
		WHERE csta.course_id = ?
			AND csta.status = 1
			AND (t.status = 0 OR t.status IS NULL)
		GROUP BY csta.teacher_id, t.name
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inactiveTeachers := []InactiveTeacherInfo{}
	for rows.Next() {
		var info InactiveTeacherInfo
		var teacherName sql.NullString
		if err := rows.Scan(&info.TeacherID, &teacherName, &info.StudentCount); err != nil {
			continue
		}
		if teacherName.Valid {
			info.TeacherName = teacherName.String
		} else {
			info.TeacherName = "Unknown"
		}
		inactiveTeachers = append(inactiveTeachers, info)
	}

	if len(inactiveTeachers) == 0 {
		return nil, nil
	}

	log.Printf("♻️  Found %d inactive teachers with students", len(inactiveTeachers))

	// Get their students grouped by learning mode
	studentsToReallocate := make(map[int][]int) // learning_mode_id -> student_ids

	for _, inactiveTeacher := range inactiveTeachers {
		studentQuery := `
			SELECT csta.student_id, s.learning_mode_id
			FROM course_student_teacher_allocation csta
			INNER JOIN students s ON csta.student_id = s.id
			WHERE csta.course_id = ?
				AND csta.teacher_id = ?
				AND csta.status = 1
		`

		studentRows, err := db.DB.Query(studentQuery, courseID, inactiveTeacher.TeacherID)
		if err != nil {
			continue
		}

		for studentRows.Next() {
			var studentID, learningModeID int
			if err := studentRows.Scan(&studentID, &learningModeID); err != nil {
				continue
			}
			studentsToReallocate[learningModeID] = append(studentsToReallocate[learningModeID], studentID)
		}
		studentRows.Close()

		// Mark old allocations as inactive
		_, err = db.DB.Exec(`
			UPDATE course_student_teacher_allocation 
			SET status = 0 
			WHERE course_id = ? AND teacher_id = ?
		`, courseID, inactiveTeacher.TeacherID)
		if err != nil {
			log.Printf("⚠️  Failed to deactivate allocations for %s: %v", inactiveTeacher.TeacherID, err)
		}
	}

	// Get active teachers for reallocation
	activeTeachers, err := getActiveTeachersForCourse(courseID)
	if err != nil || len(activeTeachers) == 0 {
		return &ReallocationInfo{
			InactiveTeachers:  inactiveTeachers,
			StudentsRelocated: 0,
		}, fmt.Errorf("no active teachers available for reallocation")
	}

	// Get existing allocations for load balancing
	existingAllocations, _ := getExistingAllocations(courseID)
	teacherLoads := make(map[string]int)
	teacherModes := make(map[string]int)
	for _, alloc := range existingAllocations {
		teacherLoads[alloc.TeacherID] = alloc.StudentCount
		teacherModes[alloc.TeacherID] = alloc.LearningModeID
	}

	totalRelocated := 0

	// Reallocate UAL students
	if ualStudents, ok := studentsToReallocate[1]; ok && len(ualStudents) > 0 {
		// Find teachers available for UAL
		ualTeachers := []string{}
		for _, t := range activeTeachers {
			if mode, exists := teacherModes[t]; !exists || mode == 1 {
				ualTeachers = append(ualTeachers, t)
			}
		}

		if len(ualTeachers) > 0 {
			mappings := distributeStudentsToTeachers(ualStudents, ualTeachers, teacherLoads, courseID, 1)
			if err := insertAllocations(mappings); err == nil {
				totalRelocated += len(mappings)
				log.Printf("✅ Reallocated %d UAL students", len(mappings))
			}
		}
	}

	// Reallocate PBL students
	if pblStudents, ok := studentsToReallocate[2]; ok && len(pblStudents) > 0 {
		// Find teachers available for PBL
		pblTeachers := []string{}
		for _, t := range activeTeachers {
			if mode, exists := teacherModes[t]; !exists || mode == 2 {
				pblTeachers = append(pblTeachers, t)
			}
		}

		if len(pblTeachers) > 0 {
			mappings := distributeStudentsToTeachers(pblStudents, pblTeachers, teacherLoads, courseID, 2)
			if err := insertAllocations(mappings); err == nil {
				totalRelocated += len(mappings)
				log.Printf("✅ Reallocated %d PBL students", len(mappings))
			}
		}
	}

	return &ReallocationInfo{
		InactiveTeachers:  inactiveTeachers,
		StudentsRelocated: totalRelocated,
	}, nil
}

// getExistingAllocations fetches current allocations grouped by teacher and mode
func getExistingAllocations(courseID int) ([]ExistingAllocation, error) {
	query := `
		SELECT 
			csta.teacher_id,
			s.learning_mode_id,
			COUNT(*) as student_count
		FROM course_student_teacher_allocation csta
		INNER JOIN students s ON csta.student_id = s.id
		WHERE csta.course_id = ? 
			AND csta.status = 1
		GROUP BY csta.teacher_id, s.learning_mode_id
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allocations := []ExistingAllocation{}
	for rows.Next() {
		var alloc ExistingAllocation
		if err := rows.Scan(&alloc.TeacherID, &alloc.LearningModeID, &alloc.StudentCount); err != nil {
			continue
		}
		allocations = append(allocations, alloc)
	}

	return allocations, nil
}

// getUnallocatedStudents fetches students not yet allocated for a specific learning mode
func getUnallocatedStudents(courseID int, learningModeID int) ([]int, error) {
	query := `
		SELECT DISTINCT s.id
		FROM students s
		INNER JOIN student_courses sc ON s.id = sc.student_id
		LEFT JOIN course_student_teacher_allocation csta 
			ON csta.student_id = s.id AND csta.course_id = sc.course_id AND csta.status = 1
		WHERE sc.course_id = ?
			AND s.learning_mode_id = ?
			AND s.status = 1
			AND csta.id IS NULL
		ORDER BY s.id
	`

	rows, err := db.DB.Query(query, courseID, learningModeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []int{}
	for rows.Next() {
		var studentID int
		if err := rows.Scan(&studentID); err != nil {
			continue
		}
		students = append(students, studentID)
	}

	return students, nil
}

// distributeStudentsToTeachers distributes students evenly across teachers with load balancing
func distributeStudentsToTeachers(students []int, teachers []string, currentLoads map[string]int, courseID int, learningModeID int) []StudentAllocationMapping {
	if len(teachers) == 0 {
		return []StudentAllocationMapping{}
	}

	mappings := []StudentAllocationMapping{}

	// Sort teachers by current load (ascending) for better balance
	sortedTeachers := make([]TeacherLoad, len(teachers))
	for i, tid := range teachers {
		sortedTeachers[i] = TeacherLoad{
			TeacherID: tid,
			Load:      currentLoads[tid],
		}
	}
	sort.Slice(sortedTeachers, func(i, j int) bool {
		return sortedTeachers[i].Load < sortedTeachers[j].Load
	})

	// Round-robin distribution with load awareness
	for i, studentID := range students {
		teacherIdx := i % len(sortedTeachers)
		teacher := sortedTeachers[teacherIdx]

		mappings = append(mappings, StudentAllocationMapping{
			StudentID:      studentID,
			CourseID:       courseID,
			TeacherID:      teacher.TeacherID,
			LearningModeID: learningModeID,
		})

		// Update load for next iteration
		sortedTeachers[teacherIdx].Load++
	}

	return mappings
}

// insertAllocations batch inserts allocations into the database
func insertAllocations(mappings []StudentAllocationMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Batch insert in chunks of 500
	batchSize := 500
	for i := 0; i < len(mappings); i += batchSize {
		end := i + batchSize
		if end > len(mappings) {
			end = len(mappings)
		}

		batch := mappings[i:end]
		if err := insertBatch(tx, batch); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// insertBatch inserts a batch of allocations
func insertBatch(tx *sql.Tx, batch []StudentAllocationMapping) error {
	if len(batch) == 0 {
		return nil
	}

	// Build multi-value INSERT
	query := `
		INSERT INTO course_student_teacher_allocation 
		(student_id, course_id, teacher_id, status)
		VALUES `

	values := []interface{}{}
	for i, mapping := range batch {
		if i > 0 {
			query += ","
		}
		query += "(?, ?, ?, 1)"
		values = append(values, mapping.StudentID, mapping.CourseID, mapping.TeacherID)
	}

	query += `
		ON DUPLICATE KEY UPDATE 
			teacher_id = VALUES(teacher_id),
			status = VALUES(status)
	`

	_, err := tx.Exec(query, values...)
	return err
}

// parseForceRebalance returns true when query parameter force_rebalance is set to true/1/yes.
func parseForceRebalance(r *http.Request) bool {
	value := r.URL.Query().Get("force_rebalance")
	if value == "" {
		return false
	}

	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "true" || normalized == "1" || normalized == "yes"
}

// shouldRebalanceCourse decides whether to reset active allocations and redistribute.
func shouldRebalanceCourse(forceRebalance bool, availableTeachers []string, existingAllocations []ExistingAllocation, unallocatedUAL int, unallocatedPBL int) bool {
	if len(existingAllocations) == 0 {
		return false
	}

	if forceRebalance {
		return true
	}

	// Auto-rebalance only when course is fully allocated but has newly added teachers.
	if unallocatedUAL > 0 || unallocatedPBL > 0 {
		return false
	}

	allocatedTeacherSet := make(map[string]bool)
	for _, alloc := range existingAllocations {
		allocatedTeacherSet[alloc.TeacherID] = true
	}

	return len(availableTeachers) > len(allocatedTeacherSet)
}

// deactivateCourseAllocations marks all active allocations for the course as inactive.
func deactivateCourseAllocations(courseID int) error {
	_, err := db.DB.Exec(`
		UPDATE course_student_teacher_allocation
		SET status = 0
		WHERE course_id = ? AND status = 1
	`, courseID)
	return err
}

// GetAllocationByCourse - View current allocations for a course
func GetAllocationByCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	courseIDStr := vars["course_id"]
	courseID, err := strconv.Atoi(courseIDStr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid course ID",
		})
		return
	}

	query := `
		SELECT 
			c.course_name,
			t.name AS teacher_name,
			csta.teacher_id,
			CASE s.learning_mode_id 
				WHEN 1 THEN 'UAL'
				WHEN 2 THEN 'PBL'
				ELSE 'Unknown'
			END AS mode,
			s.learning_mode_id,
			COUNT(*) AS student_count
		FROM course_student_teacher_allocation csta
		INNER JOIN students s ON csta.student_id = s.id
		INNER JOIN teachers t ON csta.teacher_id = t.faculty_id
		INNER JOIN courses c ON csta.course_id = c.id
		WHERE csta.course_id = ? AND csta.status = 1
		GROUP BY c.id, t.id, s.learning_mode_id
		ORDER BY s.learning_mode_id, student_count DESC
	`

	rows, err := db.DB.Query(query, courseID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to fetch allocations",
		})
		return
	}
	defer rows.Close()

	type AllocationView struct {
		CourseName   string `json:"course_name"`
		TeacherName  string `json:"teacher_name"`
		TeacherID    string `json:"teacher_id"`
		Mode         string `json:"mode"`
		ModeID       int    `json:"mode_id"`
		StudentCount int    `json:"student_count"`
	}

	allocations := []AllocationView{}
	var courseName string

	for rows.Next() {
		var alloc AllocationView
		if err := rows.Scan(&courseName, &alloc.TeacherName, &alloc.TeacherID, &alloc.Mode, &alloc.ModeID, &alloc.StudentCount); err != nil {
			continue
		}
		alloc.CourseName = courseName
		allocations = append(allocations, alloc)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"course_id":   courseID,
		"course_name": courseName,
		"allocations": allocations,
	})
}
