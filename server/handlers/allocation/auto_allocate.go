package allocation

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"server/db"
	"sort"
	"time"
)

// CourseEnrollment represents a course's enrollment data
type CourseEnrollment struct {
	CourseID        int
	CourseCode      string
	CourseName      string
	Semester        int
	Batch           string
	TotalStudents   int
	RequiredSections int
}

// TeacherPreference represents a teacher's course preference
type TeacherPreference struct {
	PreferenceID     int    // ID from teacher_course_preferences table
	FacultyID        string
	TeacherName      string
	CourseInternalID int
	CourseCode       string
	CourseName       string
	Semester         int
	Batch            string
	CourseType       string
	Priority         int
	MaxCount         int
}

// AllocationResult represents the outcome of an allocation
type AllocationResult struct {
	PreferenceID int       `json:"preference_id"` // Link to teacher_course_preferences
	FacultyID    string    `json:"faculty_id"`
	TeacherName  string    `json:"teacher_name"`
	CourseID     int       `json:"course_id"`
	CourseCode   string    `json:"course_code"`
	CourseName   string    `json:"course_name"`
	Section      string    `json:"section"`
	AllocatedAt  time.Time `json:"allocated_at"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// calculateTeachersNeeded calculates required teacher count based on 60-student rule
// remainder <= 30: round down | remainder > 30: round up
func calculateTeachersNeeded(studentCount int) int {
	if studentCount == 0 {
		return 0
	}

	quotient := studentCount / 60
	remainder := studentCount % 60

	// If remainder <= 30, round down; if > 30, round up
	if remainder <= 30 {
		return quotient
	}
	return quotient + 1
}

// RunAutoAllocation - Main endpoint to trigger automatic allocation
func RunAutoAllocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	startTime := time.Now()

	// Get the current academic year
	var academicYear string
	err := db.DB.QueryRow(`
		SELECT academic_year
		FROM academic_calendar 
		WHERE is_current = 1
		LIMIT 1
	`).Scan(&academicYear)

	if err != nil {
		log.Printf("No calendar entry found for allocation: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No academic year found",
		})
		return
	}

	log.Printf("🚀 Starting allocation for Academic Year: %s", academicYear)

	// Get all distinct semesters that have preferences for this academic year
	semestersQuery := `
		SELECT DISTINCT semester 
		FROM teacher_course_preferences 
		WHERE academic_year = ?
		AND status IN ('approved', 'pending')
		ORDER BY semester
	`
	
	rows, err := db.DB.Query(semestersQuery, academicYear)
	if err != nil {
		log.Printf("Error fetching semesters: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to fetch semesters",
		})
		return
	}
	defer rows.Close()

	var semesters []int
	for rows.Next() {
		var sem int
		if err := rows.Scan(&sem); err != nil {
			continue
		}
		semesters = append(semesters, sem)
	}

	if len(semesters) == 0 {
		log.Printf("No preferences found for academic year %s", academicYear)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No preferences found to process",
		})
		return
	}

	log.Printf("Found preferences for %d semesters: %v", len(semesters), semesters)

	totalAllocations := 0
	allResults := []map[string]interface{}{}

	// Process each semester
	for _, semester := range semesters {
		log.Printf("📚 Processing Semester %d", semester)

		// Step 1: Calculate course enrollments and required sections
		enrollments, err := calculateCourseEnrollments(academicYear, semester)
		if err != nil {
			log.Printf("Error calculating enrollments for semester %d: %v", semester, err)
			continue
		}

		log.Printf("Found %d courses needing allocation in semester %d", len(enrollments), semester)

		// Step 2: Get approved teacher preferences
		preferences, err := getTeacherPreferences(academicYear, semester)
		if err != nil {
			log.Printf("Error fetching preferences for semester %d: %v", semester, err)
			continue
		}

		log.Printf("Found preferences for %d teachers in semester %d", len(preferences), semester)

		// Step 3: Run allocation algorithm
		allocations := performAllocation(enrollments, preferences)

		log.Printf("Allocation algorithm completed with %d results for semester %d", len(allocations), semester)

		// Step 4: Save allocations to database
		successCount, failCount := saveAllocations(allocations, academicYear, semester)

		log.Printf("✓ Semester %d: %d successful, %d failed allocations", semester, successCount, failCount)
		
		totalAllocations += successCount
		
		allResults = append(allResults, map[string]interface{}{
			"semester":           semester,
			"total_allocations":  successCount,
			"failed_allocations": failCount,
			"total_courses":      len(enrollments),
			"total_teachers":     len(preferences),
		})
	}

	executionTime := time.Since(startTime).Seconds()

	log.Printf("✓ All semesters completed: %d total allocations in %.2f seconds", totalAllocations, executionTime)

	// Return response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"academic_year":  academicYear,
		"semesters":      allResults,
		"total_allocations": totalAllocations,
		"execution_time": executionTime,
	})
}

// calculateCourseEnrollments gets all curriculum courses with student enrollment data
func calculateCourseEnrollments(academicYear string, semester int) (map[string]*CourseEnrollment, error) {
	// Get ALL courses from curriculum (not just from preferences)
	// Join with student_courses to get actual student counts
	query := `
		SELECT 
			c.id,
			c.course_code,
			c.course_name,
			nc.semester_number,
			'' as batch,
			COALESCE(NULLIF(COUNT(DISTINCT sc.student_id), 0), 60) as total_students
		FROM curriculum_courses cc
		JOIN normal_cards nc ON cc.semester_id = nc.id
		JOIN courses c ON cc.course_id = c.id
		LEFT JOIN student_courses sc ON c.id = sc.course_id
		WHERE nc.semester_number = ?
		AND nc.card_type = 'semester'
		GROUP BY c.id, c.course_code, c.course_name, nc.semester_number
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, semester)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	enrollments := make(map[string]*CourseEnrollment)

	for rows.Next() {
		var enrollment CourseEnrollment
		err := rows.Scan(
			&enrollment.CourseID,
			&enrollment.CourseCode,
			&enrollment.CourseName,
			&enrollment.Semester,
			&enrollment.Batch,
			&enrollment.TotalStudents,
		)
		if err != nil {
			log.Printf("Error scanning enrollment: %v", err)
			continue
		}

		// Calculate required teachers using the 60-student rule
		enrollment.RequiredSections = calculateTeachersNeeded(enrollment.TotalStudents)

		log.Printf("  Course %s (%s): %d students → %d teachers needed", 
			enrollment.CourseCode, enrollment.CourseName, enrollment.TotalStudents, enrollment.RequiredSections)

		enrollments[enrollment.CourseCode] = &enrollment
	}

	return enrollments, nil
}

// getTeacherPreferences fetches approved teacher preferences with faculty_id
func getTeacherPreferences(academicYear string, semester int) (map[string][]TeacherPreference, error) {
	query := `
		SELECT 
			tcp.id,
			t.faculty_id,
			t.name as teacher_name,
			c.id as course_internal_id,
			c.course_code,
			c.course_name,
			tcp.semester,
			tcp.batch,
			ct.course_type,
			tcp.priority,
			COALESCE(tcl.max_count, 2) as max_count
		FROM teacher_course_preferences tcp
		JOIN teachers t ON tcp.teacher_id = t.id
		JOIN courses c ON tcp.course_id = c.course_code
		JOIN course_type ct ON tcp.course_type = ct.id
		LEFT JOIN teacher_course_limits tcl 
			ON t.id = tcl.teacher_id 
			AND tcp.course_type = tcl.course_type_id
		WHERE tcp.academic_year = ? 
		AND tcp.semester = ?
		AND tcp.status IN ('approved', 'pending')
		AND tcp.is_active = 1
		ORDER BY t.faculty_id, tcp.priority
	`

	rows, err := db.DB.Query(query, academicYear, semester)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	preferences := make(map[string][]TeacherPreference)

	for rows.Next() {
		var pref TeacherPreference
		err := rows.Scan(
			&pref.PreferenceID,
			&pref.FacultyID,
			&pref.TeacherName,
			&pref.CourseInternalID,
			&pref.CourseCode,
			&pref.CourseName,
			&pref.Semester,
			&pref.Batch,
			&pref.CourseType,
			&pref.Priority,
			&pref.MaxCount,
		)
		if err != nil {
			log.Printf("Error scanning preference: %v", err)
			continue
		}

		preferences[pref.FacultyID] = append(preferences[pref.FacultyID], pref)
	}

	return preferences, nil
}

// performAllocation allocates teachers to courses based on curriculum demand and preferences
func performAllocation(enrollments map[string]*CourseEnrollment, preferences map[string][]TeacherPreference) []AllocationResult {
	var results []AllocationResult

	log.Printf("📊 Total courses to allocate: %d", len(enrollments))
	log.Printf("📊 Total teachers with preferences: %d", len(preferences))

	// Build reverse map: course_code -> list of teachers who prefer it
	courseToTeachers := make(map[string][]string) // course_code -> []faculty_id
	for facultyID, prefs := range preferences {
		for _, pref := range prefs {
			courseToTeachers[pref.CourseCode] = append(courseToTeachers[pref.CourseCode], facultyID)
		}
	}

	// Build list of all faculty IDs for random selection
	var allTeachers []string
	for facultyID := range preferences {
		allTeachers = append(allTeachers, facultyID)
	}
	sort.Strings(allTeachers)

	// Track allocations per teacher per course type
	teacherAllocationCount := make(map[string]map[string]int)
	for facultyID := range preferences {
		teacherAllocationCount[facultyID] = make(map[string]int)
	}

	// Track which courses each teacher has been allocated
	teacherAllocatedCourses := make(map[string]map[string]bool)
	for facultyID := range preferences {
		teacherAllocatedCourses[facultyID] = make(map[string]bool)
	}

	// Initialize max counts per teacher from preferences
	teacherMaxCounts := make(map[string]map[string]int) // faculty_id -> course_type -> max
	for facultyID, prefs := range preferences {
		if teacherMaxCounts[facultyID] == nil {
			teacherMaxCounts[facultyID] = make(map[string]int)
		}
		for _, pref := range prefs {
			if teacherMaxCounts[facultyID][pref.CourseType] == 0 {
				teacherMaxCounts[facultyID][pref.CourseType] = pref.MaxCount
			}
		}
	}

	log.Println("🔄 Starting curriculum-based allocation with random teacher assignment...")

	// For each course that needs allocation
	allocatedCount := 0
	for courseCode, enrollment := range enrollments {
		if enrollment.RequiredSections == 0 {
			continue
		}

		log.Printf("\n📚 Allocating %s: needs %d teachers (%d students)", 
			courseCode, enrollment.RequiredSections, enrollment.TotalStudents)

		// Step 1: Collect teachers who prefer this course
		preferringTeachers := courseToTeachers[courseCode]
		log.Printf("   Teachers who prefer %s: %d", courseCode, len(preferringTeachers))

		// Shuffle preferring teachers for randomness
		rand.Shuffle(len(preferringTeachers), func(i, j int) {
			preferringTeachers[i], preferringTeachers[j] = preferringTeachers[j], preferringTeachers[i]
		})

		// Step 2: Try to allocate from teachers who prefer this course
		needCount := enrollment.RequiredSections
		allocatedThisCourse := 0

		for _, facultyID := range preferringTeachers {
			if needCount == 0 {
				break
			}

			// Skip if teacher already has this course
			if teacherAllocatedCourses[facultyID][courseCode] {
				continue
			}

			// Find the course type for this teacher's preference for this course
			var courseType string
			var prefID int
			for _, pref := range preferences[facultyID] {
				if pref.CourseCode == courseCode {
					courseType = pref.CourseType
					prefID = pref.PreferenceID
					break
				}
			}

			// Check if teacher has capacity for this course type
			maxForType := teacherMaxCounts[facultyID][courseType]
			currentForType := teacherAllocationCount[facultyID][courseType]

			if currentForType >= maxForType {
				log.Printf("     ⚠️ %s at limit for %s (%d/%d)", facultyID, courseType, currentForType, maxForType)
				continue
			}

			// Allocate!
			results = append(results, AllocationResult{
				PreferenceID: prefID,
				FacultyID:   facultyID,
				TeacherName: "", // Will be filled from preference
				CourseID:    enrollment.CourseID,
				CourseCode:  courseCode,
				CourseName:  enrollment.CourseName,
				Section:     "A", // Placeholder
				AllocatedAt: time.Now(),
				Success:     true,
			})

			teacherAllocationCount[facultyID][courseType]++
			teacherAllocatedCourses[facultyID][courseCode] = true
			needCount--
			allocatedThisCourse++

			log.Printf("     ✓ Allocated %s (prefers: YES)", facultyID)
		}

		// Step 3: If still need more teachers, randomly select from those who don't prefer it
		if needCount > 0 {
			log.Printf("   ⚠️ Need %d more teachers for %s - randomly assigning from others", needCount, courseCode)

			// Get teachers not yet allocated to this course
			unallocatedTeachers := []string{}
			for _, facultyID := range allTeachers {
				if !teacherAllocatedCourses[facultyID][courseCode] {
					unallocatedTeachers = append(unallocatedTeachers, facultyID)
				}
			}

			// Shuffle for randomness
			rand.Shuffle(len(unallocatedTeachers), func(i, j int) {
				unallocatedTeachers[i], unallocatedTeachers[j] = unallocatedTeachers[j], unallocatedTeachers[i]
			})

			for _, facultyID := range unallocatedTeachers {
				if needCount == 0 {
					break
				}

				// Get any preference from this teacher to use their data (or use default values)
				var courseType string
				var maxCourses int
				if len(preferences[facultyID]) > 0 {
					courseType = preferences[facultyID][0].CourseType
					maxCourses = preferences[facultyID][0].MaxCount
				} else {
					courseType = "theory"
					maxCourses = 2
				}

				// Check capacity
				currentForType := teacherAllocationCount[facultyID][courseType]
				if currentForType >= maxCourses {
					continue
				}

				// Allocate!
				var prefID int
				// Try to find preference for this course, if exists
				for _, pref := range preferences[facultyID] {
					if pref.CourseCode == courseCode {
						prefID = pref.PreferenceID
						break
					}
				}

				results = append(results, AllocationResult{
					PreferenceID: prefID,
					FacultyID:   facultyID,
					TeacherName: "",
					CourseID:    enrollment.CourseID,
					CourseCode:  courseCode,
					CourseName:  enrollment.CourseName,
					Section:     "A",
					AllocatedAt: time.Now(),
					Success:     true,
				})

				teacherAllocationCount[facultyID][courseType]++
				teacherAllocatedCourses[facultyID][courseCode] = true
				needCount--
				allocatedThisCourse++

				log.Printf("     ✓ Assigned %s (prefers: NO - randomly selected)", facultyID)
			}

			if needCount > 0 {
				log.Printf("     ❌ Could not allocate %d teachers for %s (not enough available capacity)", needCount, courseCode)
			}
		}

		allocatedCount += allocatedThisCourse
		log.Printf("   ✓ %s: Successfully allocated %d/%d teachers needed", courseCode, allocatedThisCourse, enrollment.RequiredSections)
	}

	log.Printf("\n✅ Allocation complete: %d total allocations made", allocatedCount)
	return results
}

// saveAllocations inserts allocations into teacher_course_allocation table with upsert logic
func saveAllocations(allocations []AllocationResult, academicYear string, semester int) (int, int) {
	successCount := 0
	failCount := 0

	// Begin transaction for atomic updates
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("❌ Failed to start transaction: %v", err)
		return 0, len(allocations)
	}
	defer tx.Rollback()

	// Step 1: Deactivate all existing allocations for this academic year and semester type
	semesterType := "odd"
	if semester%2 == 0 {
		semesterType = "even"
	}

	log.Printf("🔄 Deactivating existing allocations for academic year %s, semester type %s", academicYear, semesterType)

	// Deactivate previous allocations by setting is_active = 0
	updateQuery := `
		UPDATE teacher_course_allocation tca
		JOIN teacher_course_preferences tcp ON tca.teacher_course_preferences_id = tcp.id
		SET tca.is_active = 0
		WHERE tcp.academic_year = ? AND tcp.current_semester_type = ?
	`
	result, err := tx.Exec(updateQuery, academicYear, semesterType)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to deactivate existing allocations: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("✓ Deactivated %d existing allocations", rowsAffected)
	}

	// Step 2: Insert or Update allocations using UPSERT
	log.Printf("📝 Processing %d allocations (insert or update)", len(allocations))

	for i := range allocations {
		// Use INSERT ... ON DUPLICATE KEY UPDATE to handle unique constraint on (course_id, teacher_id)
		// If record exists, update it; otherwise insert new
		_, err := tx.Exec(`
			INSERT INTO teacher_course_allocation 
			(course_id, teacher_id, teacher_course_preferences_id, is_active)
			VALUES (?, ?, ?, 1)
			ON DUPLICATE KEY UPDATE
				teacher_course_preferences_id = VALUES(teacher_course_preferences_id),
				is_active = 1
		`, allocations[i].CourseID, allocations[i].FacultyID, allocations[i].PreferenceID)

		if err != nil {
			log.Printf("❌ Failed to save allocation for %s → %s: %v",
				allocations[i].FacultyID, allocations[i].CourseCode, err)
			allocations[i].Success = false
			allocations[i].ErrorMessage = err.Error()
			failCount++
		} else {
			log.Printf("  ✓ Saved: %s → %s (pref_id=%d)", 
				allocations[i].FacultyID, allocations[i].CourseCode, allocations[i].PreferenceID)
			successCount++
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Failed to commit transaction: %v", err)
		return 0, len(allocations)
	}

	log.Printf("✓ Transaction committed: %d successful, %d failed", successCount, failCount)
	return successCount, failCount
}
