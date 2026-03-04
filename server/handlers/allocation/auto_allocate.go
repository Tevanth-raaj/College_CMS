package allocation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"strings"
	"time"
)

// DepartmentAllocation represents allocation data per department
type DepartmentAllocation struct {
	DepartmentID   int
	DepartmentName string
	AcademicYear   string
	Semester       int
	TotalStudents  int
	Courses[]CourseAllocationDetail
	Teachers       map[string]*TeacherAllocationCount
}

// CourseAllocationDetail represents a course and its allocation details
type CourseAllocationDetail struct {
	CourseID          int
	CourseCode        string
	CourseName        string
	CourseTypeID      int
	StudentCount      int
	RequiredTeachers  int
	AllocatedTeachers []map[string]interface{} // {"id": teacher_id, "faculty_id": faculty_id, "max_count": limit}
}

// TeacherAllocationCount tracks how many courses a teacher is allocated
type TeacherAllocationCount struct {
	TeacherID      int
	FacultyID      string
	TeacherName    string
	DepartmentID   int
	MaxCourses     int
	AllocatedCount int
}

// calculateTeachersNeeded calculates required teacher count based on 60-student rule
func calculateTeachersNeeded(studentCount int) int {
	if studentCount == 0 {
		return 0
	}
	quotient := studentCount / 60
	remainder := studentCount % 60
	if remainder > 30 {
		return quotient + 1
	}
	return quotient
}

// isAssigned checks if a faculty is already assigned to this specific course
func isAssigned(allocated []map[string]interface{}, facultyID string) bool {
	for _, t := range allocated {
		if t["faculty_id"] == facultyID {
			return true
		}
	}
	return false
}

// buildInClause builds a parameterised SQL IN clause string like "(?,?,?)" and matching args.
func buildInClause(vals []int) (string, []interface{}) {
	placeholders := make([]string, len(vals))
	args := make([]interface{}, len(vals))
	for i, v := range vals {
		placeholders[i] = "?"
		args[i] = v
	}
	return "(" + strings.Join(placeholders, ",") + ")", args
}

// RunAutoAllocation - Allocate teachers based on preferences and random round-robin
func RunAutoAllocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	startTime := time.Now()

	// Do not run when a new window is active
	var activeCount int
	err := db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM teacher_course_tracking
		WHERE window_start IS NOT NULL
		AND window_end IS NOT NULL
		AND window_start <= DATE(NOW())
		AND window_end >= DATE(NOW())
	`).Scan(&activeCount)
	if err != nil {
		log.Printf("❌ Failed to check active windows: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to check active windows"})
		return
	}
	if activeCount > 0 {
		log.Printf("ℹ️  Active window found - skipping allocation run")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Active window found"})
		return
	}

	// Get the most recent closed window
	var windowAcademicYear string
	var currentSemesterType string
	var windowStart, windowEnd time.Time

	err = db.DB.QueryRow(`
		SELECT academic_year, COALESCE(current_semester_type, 'even') as semester_type, window_start, window_end
		FROM teacher_course_tracking
		WHERE window_end IS NOT NULL
		AND window_end < DATE(NOW())
		ORDER BY window_end DESC
		LIMIT 1
	`).Scan(&windowAcademicYear, &currentSemesterType, &windowStart, &windowEnd)

	if err != nil {
		log.Printf("❌ No closed window found for allocation: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "No closed window found"})
		return
	}

	// Parse window academic year once for reuse
	var currentYearStart, currentYearEnd int
	fmt.Sscanf(windowAcademicYear, "%d-%d", &currentYearStart, &currentYearEnd)

	// extraCourseYear: for extra courses (honour/minor/elective/open-elective/addon)
	// the relevant hod_elective_selections are always in the NEXT academic year
	// because students advancing a semester enter a new cycle for these tracks.
	extraCourseYear := fmt.Sprintf("%d-%d", currentYearStart+1, currentYearEnd+1)

	var semestersToAllocate []int
	var allocationYear string

	if currentSemesterType == "even" {
		semestersToAllocate = []int{1, 3, 5, 7}
		allocationYear = windowAcademicYear
	} else {
		semestersToAllocate = []int{2, 4, 6, 8}
		allocationYear = fmt.Sprintf("%d-%d", currentYearStart+1, currentYearEnd+1)
	}

	var targetSemesterType string
	if currentSemesterType == "even" {
		targetSemesterType = "ODD"
	} else {
		targetSemesterType = "EVEN"
	}

	// Skip save if this window already has history entries
	var existingWindowHistory int
	err = db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM teacher_course_history
		WHERE window_start = ?
		AND window_end = ?
		AND semester_type = ?
		AND academic_year = ?
		AND record_type = 'course'
	`, windowStart, windowEnd, targetSemesterType, allocationYear).Scan(&existingWindowHistory)
	if err != nil {
		log.Printf("❌ Failed to check existing window history: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to check existing window history"})
		return
	}
	if existingWindowHistory > 0 {
		log.Printf("ℹ️  History already exists for window %s (%s). Skipping save.", allocationYear, targetSemesterType)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "skipped": true, "message": "History already exists for this window"})
		return
	}

	log.Printf("\n🚀 STARTING AUTO ALLOCATION")
	log.Printf("   Allocating for semesters: %v in %s (extra courses year: %s)", semestersToAllocate, allocationYear, extraCourseYear)

	// Step 0: Fetch Active Teacher Preferences globally
	prefRows, err := db.DB.Query(`SELECT teacher_id, course_id FROM teacher_course_preferences WHERE is_active = 1`)
	teacherPrefs := make(map[string]map[string]bool)
	if err == nil {
		defer prefRows.Close()
		for prefRows.Next() {
			var tID, cID string
			if err := prefRows.Scan(&tID, &cID); err == nil {
				if teacherPrefs[tID] == nil {
					teacherPrefs[tID] = make(map[string]bool)
				}
				teacherPrefs[tID][cID] = true
			}
		}
	} else {
		log.Printf("⚠️  Could not fetch preferences: %v", err)
	}

	allocationResults := []map[string]interface{}{}
	allAllocatedCourses := []CourseAllocationDetail{}

	// Step 1: Get all departments that have a current_curriculum_id set.
	// departments.current_curriculum_id is the direct FK → curriculum.id
	// (same field the admin curriculum page uses to browse dept → semesters → courses)
	type deptInfo struct {
		ID   int
		Name string
	}
	deptRows, err := db.DB.Query(`
		SELECT id, department_name
		FROM departments
		WHERE department_name IS NOT NULL
		  AND current_curriculum_id IS NOT NULL
		  AND status = 1
		ORDER BY department_name
	`)
	if err != nil {
		log.Printf("❌ Error fetching departments: %v", err)
		return
	}
	defer deptRows.Close()

	var departments []deptInfo
	for deptRows.Next() {
		var dep deptInfo
		if err := deptRows.Scan(&dep.ID, &dep.Name); err == nil {
			departments = append(departments, dep)
		}
	}

	// Build semester IN clause once – reused for both core and extra queries
	semInClause, semArgs := buildInClause(semestersToAllocate)

	// Step 2: Process each department
	for _, dep := range departments {
		deptName := dep.Name
		deptID := dep.ID
		log.Printf("🏢 Department: %s (id=%d)", deptName, deptID)

		var courses []CourseAllocationDetail

		// ── CORE courses ───────────────────────────────────────────────────────
		// Fetch via curriculum → normal_cards → curriculum_courses → courses
		// for the target semesters, excluding special/extra categories.
		// Use departments.current_curriculum_id — the same link the admin page uses
		coreQuery := fmt.Sprintf(`
			SELECT DISTINCT
				c.id, c.course_code, c.course_name,
				COALESCE(c.course_type, 1) AS course_type_id,
				COALESCE(sc_counts.student_count, 0) AS student_count
			FROM curriculum_courses cc
			JOIN normal_cards nc ON nc.id = cc.semester_id
			JOIN courses c ON c.id = cc.course_id
			INNER JOIN curriculum cur ON cur.id = nc.curriculum_id
			INNER JOIN departments d ON d.current_curriculum_id = cur.id
			LEFT JOIN (
				SELECT sc.course_id, COUNT(DISTINCT s.id) AS student_count
				FROM student_courses sc
				JOIN students s ON s.id = sc.student_id
				WHERE s.department_id = ?
				GROUP BY sc.course_id
			) sc_counts ON sc_counts.course_id = c.id
			WHERE d.id = ?
			  AND nc.semester_number IN %s
			  AND nc.card_type = 'semester'
			  AND nc.status = 1
			  AND c.category NOT IN (
			        'PE - Professional Elective', 'OE - Open Elective',
			        'Elective', 'Open Elective', 'Honour', 'Minor', 'Addon'
			      )
			GROUP BY c.id, c.course_code, c.course_name, c.course_type
			ORDER BY c.course_code
		`, semInClause)
		coreArgs := make([]interface{}, 0, 2+len(semArgs))
		coreArgs = append(coreArgs, deptID, deptID)
		coreArgs = append(coreArgs, semArgs...)
		coreRows, err := db.DB.Query(coreQuery, coreArgs...)
		if err != nil {
			log.Printf("   ⚠️  Core course query error for dept %s: %v", deptName, err)
		} else {
			for coreRows.Next() {
				var course CourseAllocationDetail
				if err := coreRows.Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &course.CourseTypeID, &course.StudentCount); err == nil {
					course.RequiredTeachers = calculateTeachersNeeded(course.StudentCount)
					courses = append(courses, course)
				}
			}
			coreRows.Close()
		}

		// ── EXTRA courses (elective / open-elective / honour / minor / addon) ──
		// Fetch from hod_elective_selections for this dept + target semesters.
		// Academic year = extraCourseYear (always next year relative to window).
		// Each hod_elective_selections row is a separate allocation slot.
		extraQuery := fmt.Sprintf(`
			SELECT
				c.id, c.course_code, c.course_name,
				COALESCE(c.course_type, 1) AS course_type_id,
				COUNT(DISTINCT sec.student_id) AS student_count
			FROM hod_elective_selections hes
			JOIN courses c ON c.id = hes.course_id
			LEFT JOIN student_elective_choices sec ON sec.hod_selection_id = hes.id
			WHERE hes.department_id = ?
			  AND hes.semester IN %s
			  AND hes.academic_year = ?
			GROUP BY hes.id, c.id, c.course_code, c.course_name, c.course_type
			HAVING COUNT(DISTINCT sec.student_id) > 0
			ORDER BY c.course_code
		`, semInClause)
		extraArgs := make([]interface{}, 0, 2+len(semArgs))
		extraArgs = append(extraArgs, deptID)
		extraArgs = append(extraArgs, semArgs...)
		extraArgs = append(extraArgs, extraCourseYear)
		extraRows, err := db.DB.Query(extraQuery, extraArgs...)
		if err != nil {
			log.Printf("   ⚠️  Extra course query error for dept %s: %v", deptName, err)
		} else {
			for extraRows.Next() {
				var course CourseAllocationDetail
				if err := extraRows.Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &course.CourseTypeID, &course.StudentCount); err == nil {
					course.RequiredTeachers = calculateTeachersNeeded(course.StudentCount)
					courses = append(courses, course)
				}
			}
			extraRows.Close()
		}

		if len(courses) == 0 {
			log.Printf("   ℹ️  No courses found for dept %s in semesters %v", deptName, semestersToAllocate)
			continue
		}

		// Fetch Teachers
		teacherRows, err := db.DB.Query(`
			SELECT DISTINCT t.id, t.faculty_id, t.name
			FROM teachers t
			WHERE t.dept = ? AND t.status = 1
			ORDER BY t.faculty_id
		`, deptID)
		
		if err != nil {
			continue
		}

		teachers := make(map[string]*TeacherAllocationCount)
		var teacherList []string

		for teacherRows.Next() {
			var teacherID int
			var facultyID, name string
			if err := teacherRows.Scan(&teacherID, &facultyID, &name); err == nil {
				teachers[facultyID] = &TeacherAllocationCount{
					TeacherID: teacherID, FacultyID: facultyID, TeacherName: name,
					MaxCourses: 0, AllocatedCount: 0,
				}
				teacherList = append(teacherList, facultyID)
			}
		}
		teacherRows.Close()

		if len(teachers) == 0 {
			continue
		}

		// Fetch teacher limits from teacher_course_history WHERE record_type='limit'.
		// Use windowAcademicYear — that is when teachers submitted their limits.
		// allocationYear may differ (e.g. odd window → even sems allocated in next year).
		// Ordered by created_at DESC so we take the most recent limit per course_type.
		teacherLimits := make(map[string]map[int]int)     // [faculty_id][course_type_id] = max_count
		teacherAllocations := make(map[string]map[int]int) // [faculty_id][course_type_id] = allocated_count

		for _, facultyID := range teacherList {
			limitRows, limitErr := db.DB.Query(`
				SELECT course_type_id, max_count
				FROM teacher_course_history
				WHERE teacher_id = ?
				  AND record_type = 'limit'
				  AND academic_year = ?
				ORDER BY course_type_id, created_at DESC
			`, facultyID, windowAcademicYear)

			if limitErr == nil {
				seenType := make(map[int]bool)
				for limitRows.Next() {
					var courseTypeID, maxCount int
					if err := limitRows.Scan(&courseTypeID, &maxCount); err == nil {
						if seenType[courseTypeID] {
							continue // already captured the latest row for this type
						}
						seenType[courseTypeID] = true
						if teacherLimits[facultyID] == nil {
							teacherLimits[facultyID] = make(map[int]int)
							teacherAllocations[facultyID] = make(map[int]int)
						}
						teacherLimits[facultyID][courseTypeID] = maxCount
						teacherAllocations[facultyID][courseTypeID] = 0
						log.Printf("   📊 Teacher %s - Year %s - Type %d: Limit=%d", facultyID, windowAcademicYear, courseTypeID, maxCount)
					}
				}
				limitRows.Close()
			}

			// If teacher has no limit records at all, give a default capacity of 3 per type
			// so allocation isn't completely blocked for teachers who haven't set limits yet.
			if teacherLimits[facultyID] == nil {
				teacherLimits[facultyID] = map[int]int{1: 3, 2: 3, 3: 3}
				teacherAllocations[facultyID] = map[int]int{1: 0, 2: 0, 3: 0}
				log.Printf("   📊 Teacher %s - no limits set, using default capacity 3 per type", facultyID)
			}
		}
		
		// Helper function to get teacher limit for a specific course type
		getTeacherLimit := func(facultyID string, courseTypeID int) int {
			if limits, ok := teacherLimits[facultyID]; ok {
				if limit, exists := limits[courseTypeID]; exists {
					return limit
				}
				// Limit record exists for this teacher but not this type — use default
				return 3
			}
			return 0
		}
		
		// Helper function to check if teacher has capacity for a course type
		hasCapacity := func(facultyID string, courseTypeID int) bool {
			limit := getTeacherLimit(facultyID, courseTypeID)
			allocated := 0
			if allocations, ok := teacherAllocations[facultyID]; ok {
				allocated = allocations[courseTypeID]
			}
			return allocated < limit
		}

		// ==========================================
		// ALLOCATION ALGORITHM
		// ==========================================

		// PASS 1A: Guarantee at least one preferred course per teacher (if possible)
		for _, facultyID := range teacherList {
			prefs, hasPrefs := teacherPrefs[facultyID]
			if !hasPrefs || len(prefs) == 0 {
				continue
			}

			mappedOnePreferred := false

			for courseIdx := range courses {
				course := &courses[courseIdx]
				if len(course.AllocatedTeachers) >= course.RequiredTeachers {
					continue
				}

				courseStrID := fmt.Sprintf("%d", course.CourseID)
				if !(prefs[courseStrID] || prefs[course.CourseCode]) {
					continue
				}

				if hasCapacity(facultyID, course.CourseTypeID) && !isAssigned(course.AllocatedTeachers, facultyID) {
					teacher := teachers[facultyID]
					teacherLimit := getTeacherLimit(facultyID, course.CourseTypeID)
					course.AllocatedTeachers = append(course.AllocatedTeachers, map[string]interface{}{
						"id":         teacher.TeacherID,
						"faculty_id": facultyID,
						"max_count":  teacherLimit,
					})
					if teacherAllocations[facultyID] == nil {
						teacherAllocations[facultyID] = make(map[int]int)
					}
					teacherAllocations[facultyID][course.CourseTypeID]++
					mappedOnePreferred = true
					log.Printf("   ✅ Guaranteed Preferred %s to %s (Type: %d, Limit: %d)", course.CourseCode, facultyID, course.CourseTypeID, teacherLimit)
					break
				}
			}

			if !mappedOnePreferred {
				log.Printf("   ℹ️  Could not map a preferred course for %s (no capacity/slot)", facultyID)
			}
		}

		// PASS 1B: Fill remaining slots based on preferences
		for courseIdx := range courses {
			course := &courses[courseIdx]
			courseStrID := fmt.Sprintf("%d", course.CourseID)
			remainingSlots := course.RequiredTeachers - len(course.AllocatedTeachers)

			for j := 0; j < remainingSlots; j++ {
				for _, facultyID := range teacherList {
					teacher := teachers[facultyID]
					
					if hasCapacity(facultyID, course.CourseTypeID) && !isAssigned(course.AllocatedTeachers, facultyID) {
						prefers := false
						if prefs, ok := teacherPrefs[facultyID]; ok {
							if prefs[courseStrID] || prefs[course.CourseCode] {
								prefers = true
							}
						}
						
						if prefers {
							teacherLimit := getTeacherLimit(facultyID, course.CourseTypeID)
							course.AllocatedTeachers = append(course.AllocatedTeachers, map[string]interface{}{
								"id":         teacher.TeacherID,
								"faculty_id": facultyID,
								"max_count":  teacherLimit,
							})
							if teacherAllocations[facultyID] == nil {
								teacherAllocations[facultyID] = make(map[int]int)
							}
							teacherAllocations[facultyID][course.CourseTypeID]++
							log.Printf("   ⭐ Preference Allocated %s to %s (Type: %d, Limit: %d)", course.CourseCode, facultyID, course.CourseTypeID, teacherLimit)
							break
						}
					}
				}
			}
		}

		// PASS 2: Random Round-Robin for REMAINING slots
		teacherIndex := 0
		for courseIdx := range courses {
			course := &courses[courseIdx]
			remainingSlots := course.RequiredTeachers - len(course.AllocatedTeachers)

			for j := 0; j < remainingSlots; j++ {
				if len(teacherList) == 0 { break }

				assigned := false
				teachersChecked := 0

				for teachersChecked < len(teacherList) {
					if teacherIndex >= len(teacherList) { teacherIndex = 0 }

					facultyID := teacherList[teacherIndex]
					teacher := teachers[facultyID]

					if hasCapacity(facultyID, course.CourseTypeID) && !isAssigned(course.AllocatedTeachers, facultyID) {
						teacherLimit := getTeacherLimit(facultyID, course.CourseTypeID)
						course.AllocatedTeachers = append(course.AllocatedTeachers, map[string]interface{}{
							"id":         teacher.TeacherID,
							"faculty_id": facultyID,
							"max_count":  teacherLimit,
						})
						// Increment allocation count for this course type
						if teacherAllocations[facultyID] == nil {
							teacherAllocations[facultyID] = make(map[int]int)
						}
						teacherAllocations[facultyID][course.CourseTypeID]++
						assigned = true
						log.Printf("   → Random Allocated %s to %s (Type: %d, Limit: %d)", course.CourseCode, facultyID, course.CourseTypeID, teacherLimit)
						teacherIndex++
						break
					}
					teacherIndex++
					teachersChecked++
				}

				if !assigned {
					log.Printf("   ⚠️  Ran out of teachers with capacity for %s (Type: %d)!", course.CourseCode, course.CourseTypeID)
					break
				}
			}
		}

		allAllocatedCourses = append(allAllocatedCourses, courses...)

		allocationResults = append(allocationResults, map[string]interface{}{
			"department": deptName, "total_courses": len(courses), "total_teachers": len(teachers),
		})
	}

	saveSuccess, saveFail := saveAllocationsForWindow(allAllocatedCourses, allocationYear, windowStart, windowEnd, targetSemesterType)
	log.Printf("✓ Saved %d allocations (%d failed)\n", saveSuccess, saveFail)
	log.Printf("   Departments processed: %d", len(allocationResults))

	// ==========================================
	// CLEANUP: Preferences & Limits
	// ==========================================
	log.Printf("🧹 Cleaning up preferences and resetting limits to 0...")

	// 1. Deactivate utilized preferences
	db.DB.Exec(`UPDATE teacher_course_preferences SET is_active = 0 WHERE is_active = 1`)

	// 2. Set limits to 0 for course types 1, 2, 3 and insert missing limits as 0
	db.DB.Exec(`
		INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count, academic_year, updated_at)
		SELECT t.faculty_id, ct.course_type_id, 0, ?, NOW()
		FROM teachers t
		CROSS JOIN (
			SELECT 1 AS course_type_id
			UNION ALL SELECT 2
			UNION ALL SELECT 3
		) ct
		WHERE t.status = 1
		ON DUPLICATE KEY UPDATE max_count = 0, updated_at = NOW()
	`, allocationYear)

	executionTime := time.Since(startTime).Seconds()
	log.Printf("✅ AUTO ALLOCATION COMPLETE (Time: %.2f seconds)\n", executionTime)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, "academic_year": allocationYear, "execution_time": executionTime,
		"departments_processed": len(allocationResults),
	})
}

// saveAllocationsForWindow saves history and then fully refreshes teacher_course_allocation.
func saveAllocationsForWindow(courses[]CourseAllocationDetail, academicYear string, windowStart time.Time, windowEnd time.Time, semesterType string) (int, int) {
	successCount := 0
	failCount := 0
	historyInsertedCount := 0

	tx, err := db.DB.Begin()
	if err != nil {
		return 0, len(courses)
	}
	defer tx.Rollback()

	type allocationRow struct {
		CourseID  int
		FacultyID string
	}
	allocationRows := []allocationRow{}
	allocationSeen := make(map[string]bool)

	for _, course := range courses {
		for _, teacherMap := range course.AllocatedTeachers {
			facultyID := teacherMap["faculty_id"].(string)
			maxCount := teacherMap["max_count"].(int) // Fetched from earlier algorithm

			// Insert History
			historyResult, err := tx.Exec(`
				INSERT INTO teacher_course_history
				(teacher_id, course_id, course_code, course_name, course_type_id, max_count, allocated_count, window_start, window_end, semester_type, academic_year, record_type, allocated_date, created_at)
				SELECT ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, 'course', NOW(), NOW()
				WHERE NOT EXISTS (
					SELECT 1 FROM teacher_course_history
					WHERE teacher_id = ? AND course_id = ? AND window_start = ? AND window_end = ? AND record_type = 'course'
				)
			`, facultyID, course.CourseID, course.CourseCode, course.CourseName, course.CourseTypeID, maxCount, windowStart, windowEnd, semesterType, academicYear,
				facultyID, course.CourseID, windowStart, windowEnd)

			if err != nil {
				failCount++
				continue
			}

			affected, _ := historyResult.RowsAffected()
			if affected == 0 {
				continue // History already existed
			}

			historyInsertedCount++
			successCount++

			allocationKey := fmt.Sprintf("%d|%s", course.CourseID, facultyID)
			if !allocationSeen[allocationKey] {
				allocationSeen[allocationKey] = true
				allocationRows = append(allocationRows, allocationRow{CourseID: course.CourseID, FacultyID: facultyID})
			}
		}
	}

	if historyInsertedCount > 0 {
		_, err = tx.Exec(`DELETE FROM teacher_course_allocation`)
		if err != nil {
			log.Printf("   ❌ Failed to clear teacher_course_allocation: %v", err)
			return 0, len(courses)
		}

		for _, row := range allocationRows {
			_, err := tx.Exec(`
				INSERT INTO teacher_course_allocation
				(course_id, teacher_id, is_active)
				VALUES (?, ?, 1)
			`, row.CourseID, row.FacultyID)
			if err != nil {
				log.Printf("   ❌ Failed to repopulate allocation %d -> %s: %v", row.CourseID, row.FacultyID, err)
				failCount++
			}
		}
	} else {
		log.Printf("ℹ️  No history inserted for this window. Allocation table refresh skipped.")
	}

	if err := tx.Commit(); err != nil {
		return 0, len(courses)
	}

	return successCount, failCount
}