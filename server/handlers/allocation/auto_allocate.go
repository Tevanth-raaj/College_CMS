package allocation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"server/db"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type allocationRunRequest struct {
	WindowID     int    `json:"window_id"`
	AcademicYear string `json:"academic_year"`
	SemesterType string `json:"semester_type"`
	WindowStart  string `json:"window_start"`
	WindowEnd    string `json:"window_end"`
	Force        bool   `json:"force"`
	FullRerun    bool   `json:"full_rerun"`
}

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

// calculateTeachersNeeded calculates required teacher count based on 60-student rule:
//   0 students    → 0 teachers (course skipped)
//   1–60 students → 1 teacher
//   >60 students  → studentCount/60, +1 only if remainder > 30
func calculateTeachersNeeded(studentCount int) int {
	if studentCount == 0 {
		return 0
	}
	if studentCount <= 60 {
		return 1
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

	runReq := allocationRunRequest{}
	if r != nil && r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&runReq); err != nil && err.Error() != "EOF" {
			log.Printf("⚠️  Invalid allocation run payload, continuing with defaults: %v", err)
		}
	}
	forceRun := runReq.Force
	fullRerun := runReq.FullRerun

	// Do not run when a new window is active
	var activeCount int
	err := db.DB.QueryRow(`
		SELECT COUNT(*)
		FROM teacher_course_tracking
		WHERE window_start IS NOT NULL
		AND window_end IS NOT NULL
		AND DATE(window_start) <= DATE(NOW())
		AND DATE(window_end) >= DATE(NOW())
	`).Scan(&activeCount)
	if err != nil {
		log.Printf("❌ Failed to check active windows: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to check active windows"})
		return
	}
	if activeCount > 0 && !forceRun {
		log.Printf("ℹ️  Active window found - skipping allocation run")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Active window found"})
		return
	}

	// Resolve target window (specific window if provided, else latest closed window)
	var windowAcademicYear string
	var currentSemesterType string
	var windowStart, windowEnd time.Time

	if runReq.WindowID > 0 {
		err = db.DB.QueryRow(`
			SELECT academic_year, COALESCE(current_semester_type, 'even') as semester_type, window_start, window_end
			FROM teacher_course_tracking
			WHERE id = ?
			LIMIT 1
		`, runReq.WindowID).Scan(&windowAcademicYear, &currentSemesterType, &windowStart, &windowEnd)
	} else if strings.TrimSpace(runReq.WindowStart) != "" && strings.TrimSpace(runReq.WindowEnd) != "" {
		err = db.DB.QueryRow(`
			SELECT academic_year, COALESCE(current_semester_type, 'even') as semester_type, window_start, window_end
			FROM teacher_course_tracking
			WHERE window_start = ? AND window_end = ?
			ORDER BY id DESC
			LIMIT 1
		`, runReq.WindowStart, runReq.WindowEnd).Scan(&windowAcademicYear, &currentSemesterType, &windowStart, &windowEnd)
	} else {
		err = db.DB.QueryRow(`
			SELECT academic_year, COALESCE(current_semester_type, 'even') as semester_type, window_start, window_end
			FROM teacher_course_tracking
			WHERE window_end IS NOT NULL
			AND DATE(window_end) < DATE(NOW())
			ORDER BY window_end DESC
			LIMIT 1
		`).Scan(&windowAcademicYear, &currentSemesterType, &windowStart, &windowEnd)
	}

	if err != nil {
		log.Printf("❌ No closed window found for allocation: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "No closed window found"})
		return
	}

	// Parse window academic year once for reuse
	var currentYearStart, currentYearEnd int
	fmt.Sscanf(windowAcademicYear, "%d-%d", &currentYearStart, &currentYearEnd)

	// extraCourseAcademicYear: use the same academic year context as the target
	// semesters being allocated. Student elective selections are saved against
	// semester + academic_year and must be matched on those fields.
	extraCourseAcademicYear := windowAcademicYear

	var semestersToAllocate []int
	var allocationYear string

	if currentSemesterType == "even" {
		semestersToAllocate = []int{1, 3, 5, 7}
		allocationYear = fmt.Sprintf("%d-%d", currentYearStart+1, currentYearEnd+1)
		extraCourseAcademicYear = allocationYear
	} else {
		semestersToAllocate = []int{2, 4, 6, 8}
		allocationYear = windowAcademicYear
		extraCourseAcademicYear = allocationYear
	}

	// prefLookupYear: the academic year under which teacher_course_preferences were saved.
	// The save handler uses shiftAcademicYearForWindowType:
	//   even window → saves preferences under NEXT year
	//   odd  window → saves preferences under CURRENT year
	// This must mirror that logic so the TCP queries actually find rows.
	var prefLookupYear string
	if currentSemesterType == "even" {
		prefLookupYear = fmt.Sprintf("%d-%d", currentYearStart+1, currentYearEnd+1) // next year
	} else {
		prefLookupYear = windowAcademicYear // current year
	}
	log.Printf("   prefLookupYear=%s  allocationYear=%s  semType=%s", prefLookupYear, allocationYear, currentSemesterType)

	// Log teachers who have active preferences for this window (uses t.id join)
	diagRows, diagErr := db.DB.Query(`
		SELECT t.faculty_id, t.name, t.dept, t.status
		FROM teachers t
		INNER JOIN teacher_course_preferences tcp ON tcp.teacher_id = t.id
		WHERE tcp.is_active = 1
		  AND tcp.academic_year = ?
		  AND tcp.current_semester_type = ?
		GROUP BY t.faculty_id, t.name, t.dept, t.status
	`, prefLookupYear, currentSemesterType)
	if diagErr == nil {
		for diagRows.Next() {
			var fid, tname string
			var dept *string
			var status int
			if err := diagRows.Scan(&fid, &tname, &dept, &status); err == nil {
				deptVal := "<NULL>"
				if dept != nil {
					deptVal = *dept
				}
				log.Printf("   Teacher with prefs: faculty_id=%s name=%s dept=%s status=%d", fid, tname, deptVal, status)
			}
		}
		diagRows.Close()
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
	if existingWindowHistory > 0 && !forceRun {
		log.Printf("ℹ️  History already exists for window %s (%s). Skipping save.", allocationYear, targetSemesterType)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "skipped": true, "message": "History already exists for this window"})
		return
	}

	if forceRun && fullRerun {
		if _, delErr := db.DB.Exec(`
			DELETE FROM teacher_course_history
			WHERE window_start = ?
			  AND window_end = ?
			  AND semester_type = ?
			  AND academic_year = ?
			  AND record_type = 'course'
		`, windowStart, windowEnd, targetSemesterType, allocationYear); delErr != nil {
			log.Printf("⚠️  Failed to clear old course history for rerun window: %v", delErr)
		}
	}

	log.Printf("\n🚀 STARTING AUTO ALLOCATION")
	log.Printf("   Allocating for semesters: %v in %s (extra courses year: %s)", semestersToAllocate, allocationYear, extraCourseAcademicYear)

	// Step 0: Fetch Active Teacher Preferences globally, keyed by faculty_id.
	// teacher_course_preferences.teacher_id stores the INTEGER teachers.id.
	// JOIN teachers to get the varchar faculty_id used as the map key.
	// The map value stores both the integer course_id ("123") and the raw stored
	// course_id string so matches work regardless of how the frontend submitted it.
	prefRows, err := db.DB.Query(`
		SELECT t.faculty_id, tcp.course_id, COALESCE(c.course_code, '') AS course_code
		FROM teacher_course_preferences tcp
		JOIN teachers t ON t.id = tcp.teacher_id
		LEFT JOIN courses c ON c.id = tcp.course_id
		WHERE tcp.is_active = 1
		  AND tcp.academic_year = ?
		  AND tcp.current_semester_type = ?
	`, prefLookupYear, currentSemesterType)
	teacherPrefs := make(map[string]map[string]bool) // faculty_id → set of course_id strings & course_codes
	if err == nil {
		defer prefRows.Close()
		for prefRows.Next() {
			var facultyID, cID, cCode string
			if err := prefRows.Scan(&facultyID, &cID, &cCode); err == nil {
				if teacherPrefs[facultyID] == nil {
					teacherPrefs[facultyID] = make(map[string]bool)
				}
				teacherPrefs[facultyID][cID] = true // raw stored value
				if cCode != "" {
					teacherPrefs[facultyID][cCode] = true
				}
			}
		}
	} else {
		log.Printf("⚠️  Could not fetch preferences: %v", err)
	}
	log.Printf("   Loaded preferences for %d teachers (prefLookupYear: %s, semType: %s)", len(teacherPrefs), prefLookupYear, currentSemesterType)

	allocationResults := []map[string]interface{}{}
	allAllocatedCourses := []CourseAllocationDetail{}

	// Step 0.5: Snapshot current teacher_course_limits into teacher_course_history
	// with record_type='limit' for this window. This is the source of truth for
	// exports after the limits table is reset to 0 post-allocation.
	log.Printf("📸 Snapshotting teacher limits into history for window %s → %s (%s %s)",
		windowStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"), allocationYear, targetSemesterType)

	snapshotResult, snapshotErr := db.DB.Exec(`
		INSERT INTO teacher_course_history
			(teacher_id, course_type_id, max_count, allocated_count,
			 window_start, window_end, semester_type, academic_year, record_type, created_at)
		SELECT
			tcl.teacher_id,
			tcl.course_type_id,
			tcl.max_count,
			0,
			?, ?, ?, ?, 'limit', NOW()
		FROM teacher_course_limits tcl
		WHERE NOT EXISTS (
			SELECT 1 FROM teacher_course_history h
			WHERE h.teacher_id  = tcl.teacher_id
			  AND h.course_type_id = tcl.course_type_id
			  AND h.window_start   = ?
			  AND h.window_end     = ?
			  AND h.record_type    = 'limit'
		)
	`, windowStart, windowEnd, targetSemesterType, allocationYear,
		windowStart, windowEnd)
	if snapshotErr != nil {
		log.Printf("⚠️  Could not snapshot limits into history: %v", snapshotErr)
	} else {
		snapshotCount, _ := snapshotResult.RowsAffected()
		log.Printf("   ✅ Snapshotted %d limit rows into teacher_course_history", snapshotCount)
	}

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
	// numYearLevels = number of distinct year cohorts in this window (1 per semester for 4-year program)
	numYearLevels := len(semestersToAllocate)
	if numYearLevels == 0 {
		numYearLevels = 4
	}

	// Step 2: Process each department
	for _, dep := range departments {
		deptName := dep.Name
		deptID := dep.ID
		log.Printf("🏢 Department: %s (id=%d)", deptName, deptID)

		var courses []CourseAllocationDetail

		// ── CORE courses ───────────────────────────────────────────────────────
		// Student count: count active students in this dept, divide by number of
		// year-levels being allocated to get an approximate cohort size per semester.
		// Use CEIL so any positive dept strength contributes at least 1 student to
		// the semester cohort (prevents requiredTeachers from becoming 0 incorrectly).
		coreQuery := fmt.Sprintf(`
			SELECT DISTINCT
				c.id, c.course_code, c.course_name,
				COALESCE((
					SELECT ct.id
					FROM course_type ct
					WHERE CAST(ct.id AS CHAR) = CAST(c.course_type AS CHAR)
					   OR LOWER(TRIM(CONVERT(ct.course_type USING utf8mb4))) COLLATE utf8mb4_general_ci =
					      LOWER(TRIM(CONVERT(CAST(c.course_type AS CHAR) USING utf8mb4))) COLLATE utf8mb4_general_ci
					LIMIT 1
				), 1) AS course_type_id,
				CAST(CEIL((SELECT COUNT(*) FROM students s WHERE s.department_id = ? AND s.status = 1) / ?) AS UNSIGNED) AS student_count
			FROM curriculum_courses cc
			JOIN normal_cards nc ON nc.id = cc.semester_id
			JOIN courses c ON c.id = cc.course_id
			INNER JOIN curriculum cur ON cur.id = nc.curriculum_id
			INNER JOIN departments d ON d.current_curriculum_id = cur.id
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
		coreArgs := make([]interface{}, 0, 3+len(semArgs))
		coreArgs = append(coreArgs, deptID, numYearLevels, deptID)
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
		// Fetch from student_elective_choices (actual student selections) joined to
		// hod_elective_selections for course metadata.
		// Match on sec.semester + sec.academic_year so selected electives are counted.
		extraQuery := fmt.Sprintf(`
			SELECT
				c.id, c.course_code, c.course_name,
				COALESCE((
					SELECT ct.id
					FROM course_type ct
					WHERE CAST(ct.id AS CHAR) = CAST(c.course_type AS CHAR)
					   OR LOWER(TRIM(CONVERT(ct.course_type USING utf8mb4))) COLLATE utf8mb4_general_ci =
					      LOWER(TRIM(CONVERT(CAST(c.course_type AS CHAR) USING utf8mb4))) COLLATE utf8mb4_general_ci
					LIMIT 1
				), 1) AS course_type_id,
				COUNT(DISTINCT sec.student_id) AS student_count
			FROM hod_elective_selections hes
			JOIN courses c ON c.id = hes.course_id
			JOIN student_elective_choices sec ON sec.hod_selection_id = hes.id
			WHERE hes.department_id = ?
			  AND hes.status = 'ACTIVE'
			  AND sec.semester IN %s
			  AND sec.academic_year = ?
			GROUP BY hes.id, c.id, c.course_code, c.course_name, c.course_type
			ORDER BY c.course_code
		`, semInClause)
		extraArgs := make([]interface{}, 0, 2+len(semArgs))
		extraArgs = append(extraArgs, deptID)
		extraArgs = append(extraArgs, semArgs...)
		extraArgs = append(extraArgs, extraCourseAcademicYear)
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

		// Only include teachers who submitted active preferences for this exact window
		// (matching academic_year + current_semester_type). Teachers from other cycles
		// or who never submitted preferences are excluded.
		// teacher_course_preferences.teacher_id stores faculty_id (varchar).
		// teacher_course_preferences.teacher_id = integer teachers.id
		teacherRows, err := db.DB.Query(`
			SELECT DISTINCT t.id, t.faculty_id, t.name
			FROM teachers t
			INNER JOIN teacher_course_preferences tcp ON tcp.teacher_id = t.id
				AND tcp.is_active = 1
				AND tcp.academic_year = ?
				AND tcp.current_semester_type = ?
			WHERE CAST(t.dept AS UNSIGNED) = ? AND t.status = 1
			ORDER BY t.faculty_id
		`, prefLookupYear, currentSemesterType, deptID)
		
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
			log.Printf("   ⚠️  No teachers with prefs found for dept %s (id=%d, prefYear=%s)", deptName, deptID, prefLookupYear)
			continue
		}
		log.Printf("   ✅ Dept %s: %d teacher(s) with prefs", deptName, len(teachers))

		// Fetch teacher limits from teacher_course_limits — the LIVE table where
		// teachers submit their per-type max counts during the selection window.
		// A snapshot of these values has already been written to teacher_course_history
		// (record_type='limit') above, so exports continue to work after the limits
		// table is reset to 0 post-allocation.
		teacherLimits := make(map[string]map[int]int)      // [faculty_id][course_type_id] = max_count
		teacherAllocations := make(map[string]map[int]int) // [faculty_id][course_type_id] = allocated so far

		for _, facultyID := range teacherList {
			limitRows, limitErr := db.DB.Query(`
				SELECT course_type_id, max_count
				FROM teacher_course_limits
				WHERE teacher_id = ?
				ORDER BY course_type_id
			`, facultyID)

			if limitErr == nil {
				for limitRows.Next() {
					var courseTypeID, maxCount int
					if err := limitRows.Scan(&courseTypeID, &maxCount); err == nil {
						if teacherLimits[facultyID] == nil {
							teacherLimits[facultyID] = make(map[int]int)
							teacherAllocations[facultyID] = make(map[int]int)
						}
						teacherLimits[facultyID][courseTypeID] = maxCount
						teacherAllocations[facultyID][courseTypeID] = 0
						log.Printf("   📊 Teacher %s - Type %d: Limit=%d", facultyID, courseTypeID, maxCount)
					}
				}
				limitRows.Close()
			}

			// If teacher has no limit records at all, treat all limits as 0 — skip allocation.
			if teacherLimits[facultyID] == nil {
				teacherLimits[facultyID] = map[int]int{1: 0, 2: 0, 3: 0}
				teacherAllocations[facultyID] = map[int]int{1: 0, 2: 0, 3: 0}
				log.Printf("   ⛔ Teacher %s - NO limit records found in teacher_course_limits (0 for all types), will be SKIPPED", facultyID)
			} else {
				log.Printf("   📊 Teacher %s limits: theory=%d, lab=%d, theory_with_lab=%d",
					facultyID,
					teacherLimits[facultyID][1],
					teacherLimits[facultyID][2],
					teacherLimits[facultyID][3])
			}
		}
		
		// Helper function to get teacher limit for a specific course type
		getTeacherLimit := func(facultyID string, courseTypeID int) int {
			if limits, ok := teacherLimits[facultyID]; ok {
				if limit, exists := limits[courseTypeID]; exists {
					return limit
				}
				// Teacher has records but not for this type →
				// they didn't volunteer for this type, treat as 0.
				return 0
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

		// Log dept summary before allocation
		log.Printf("   Dept %s: %d course(s) to allocate", deptName, len(courses))
		for ci := range courses {
			log.Printf("     Course: id=%d code=%s name=%s type=%d requiredTeachers=%d",
				courses[ci].CourseID, courses[ci].CourseCode, courses[ci].CourseName,
				courses[ci].CourseTypeID, courses[ci].RequiredTeachers)
		}
		for _, fid := range teacherList {
			if prefs, ok := teacherPrefs[fid]; ok {
				prefKeys := make([]string, 0, len(prefs))
				for k := range prefs {
					prefKeys = append(prefKeys, k)
				}
				log.Printf("   Teacher %s prefers: %v", fid, prefKeys)
			}
		}

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
			if remainingSlots > 0 {
				log.Printf("   [1B] %s (type %d): need %d more teachers", course.CourseCode, course.CourseTypeID, remainingSlots)
			}

			for j := 0; j < remainingSlots; j++ {
				slotFilled := false
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
							log.Printf("   ⭐ [1B] Preference Allocated %s to %s (Type: %d, Limit: %d)", course.CourseCode, facultyID, course.CourseTypeID, teacherLimit)
							slotFilled = true
							break
						}
					}
				}
				if !slotFilled {
					log.Printf("   ⚠️  [1B] No preferred teacher with capacity for slot %d of %s (type %d)", j+1, course.CourseCode, course.CourseTypeID)
					break // no point trying further slots if no one has capacity
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

	saveSuccess, saveFail := saveAllocationsForWindow(allAllocatedCourses, allocationYear, windowStart, windowEnd, targetSemesterType, fullRerun)
	log.Printf("✓ Saved %d allocations (%d failed)\n", saveSuccess, saveFail)
	log.Printf("   Departments processed: %d", len(allocationResults))

	// ==========================================
	// CLEANUP: Preferences & Limits
	// ==========================================
	if saveSuccess > 0 {
		log.Printf("🧹 Cleaning up preferences and resetting limits to 0...")

		// Collect faculty IDs that were actually allocated
		allocatedFacultyIDs := make(map[string]bool)
		for _, course := range allAllocatedCourses {
			for _, teacherMap := range course.AllocatedTeachers {
				if fid, ok := teacherMap["faculty_id"].(string); ok {
					allocatedFacultyIDs[fid] = true
				}
			}
		}

		// 1. Deactivate ONLY the preferences of teachers who were allocated.
		//    tcp.teacher_id is the integer teachers.id; resolve from faculty_id.
		for fid := range allocatedFacultyIDs {
			db.DB.Exec(`
				UPDATE teacher_course_preferences tcp
				JOIN teachers t ON t.id = tcp.teacher_id
				SET tcp.is_active = 0
				WHERE t.faculty_id = ? AND tcp.is_active = 1
			`, fid)
		}
		log.Printf("   Deactivated preferences for %d allocated teachers", len(allocatedFacultyIDs))

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
	} else {
		log.Printf("⚠️  No allocations saved — skipping preference/limit cleanup to preserve data for retry")
	}

	executionTime := time.Since(startTime).Seconds()
	log.Printf("✅ AUTO ALLOCATION COMPLETE (Time: %.2f seconds)\n", executionTime)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":               true,
		"academic_year":         allocationYear,
		"execution_time":        executionTime,
		"departments_processed": len(allocationResults),
		"allocations_saved":     saveSuccess,
	})
}

// saveAllocationsForWindow saves history and updates teacher_course_allocation.
// If fullRefresh is true, existing allocations are replaced for rerun use-cases.
func saveAllocationsForWindow(courses[]CourseAllocationDetail, academicYear string, windowStart time.Time, windowEnd time.Time, semesterType string, fullRefresh bool) (int, int) {
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
		if fullRefresh {
			_, err = tx.Exec(`DELETE FROM teacher_course_allocation`)
			if err != nil {
				log.Printf("   ❌ Failed to clear teacher_course_allocation: %v", err)
				return 0, len(courses)
			}
		}

		for _, row := range allocationRows {
			_, err := tx.Exec(`
				INSERT INTO teacher_course_allocation (course_id, teacher_id, is_active)
				SELECT ?, ?, 1
				WHERE NOT EXISTS (
					SELECT 1 FROM teacher_course_allocation
					WHERE course_id = ? AND teacher_id = ?
				)
			`, row.CourseID, row.FacultyID, row.CourseID, row.FacultyID)
			if err != nil {
				log.Printf("   ❌ Failed to upsert allocation %d -> %s: %v", row.CourseID, row.FacultyID, err)
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

func shiftAcademicYearForward(year string) string {
	parts := strings.Split(strings.TrimSpace(year), "-")
	if len(parts) != 2 {
		return strings.TrimSpace(year)
	}
	startYear, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	endYear, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return strings.TrimSpace(year)
	}
	return fmt.Sprintf("%d-%d", startYear+1, endYear+1)
}

func targetSemesterTypeFromCurrent(currentSemesterType string) string {
	if strings.EqualFold(strings.TrimSpace(currentSemesterType), "even") {
		return "ODD"
	}
	return "EVEN"
}

func allocationYearForWindow(windowAcademicYear, currentSemesterType string) string {
	if strings.EqualFold(strings.TrimSpace(currentSemesterType), "even") {
		return shiftAcademicYearForward(windowAcademicYear)
	}
	return strings.TrimSpace(windowAcademicYear)
}

func prefLookupYearForWindow(windowAcademicYear, currentSemesterType string) string {
	if strings.EqualFold(strings.TrimSpace(currentSemesterType), "even") {
		return shiftAcademicYearForward(windowAcademicYear)
	}
	return strings.TrimSpace(windowAcademicYear)
}

// RerunAllocationForWindow reruns allocation for a specific admin-selected window.
// It restores limits from history and reactivates preferences for a full fresh allocation.
func RerunAllocationForWindow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	windowIDText := mux.Vars(r)["id"]
	windowID, err := strconv.Atoi(windowIDText)
	if err != nil || windowID <= 0 {
		http.Error(w, "Invalid window id", http.StatusBadRequest)
		return
	}

	var academicYear, currentSemesterType string
	var windowStart, windowEnd time.Time
	err = db.DB.QueryRow(`
		SELECT academic_year, COALESCE(current_semester_type, 'even'), window_start, window_end
		FROM teacher_course_tracking
		WHERE id = ?
		LIMIT 1
	`, windowID).Scan(&academicYear, &currentSemesterType, &windowStart, &windowEnd)
	if err == sql.ErrNoRows {
		http.Error(w, "Window not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to fetch window %d for rerun: %v", windowID, err)
		http.Error(w, "Failed to fetch window", http.StatusInternalServerError)
		return
	}

	prefLookupYear := prefLookupYearForWindow(academicYear, currentSemesterType)
	if _, err := db.DB.Exec(`
		UPDATE teacher_course_preferences
		SET is_active = 1
		WHERE academic_year = ? AND current_semester_type = ?
	`, prefLookupYear, strings.ToLower(strings.TrimSpace(currentSemesterType))); err != nil {
		log.Printf("⚠️ Failed to reactivate preferences for rerun: %v", err)
	}

	targetSemesterType := targetSemesterTypeFromCurrent(currentSemesterType)
	allocationYear := allocationYearForWindow(academicYear, currentSemesterType)
	if _, err := db.DB.Exec(`
		INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count, academic_year, updated_at)
		SELECT teacher_id, course_type_id, max_count, ?, NOW()
		FROM teacher_course_history
		WHERE window_start = ?
		  AND window_end = ?
		  AND semester_type = ?
		  AND academic_year = ?
		  AND record_type = 'limit'
		ON DUPLICATE KEY UPDATE
			max_count = VALUES(max_count),
			academic_year = VALUES(academic_year),
			updated_at = NOW()
	`, allocationYear, windowStart, windowEnd, targetSemesterType, allocationYear); err != nil {
		log.Printf("⚠️ Failed to restore limits for rerun: %v", err)
	}

	payload := allocationRunRequest{
		WindowID:  windowID,
		Force:     true,
		FullRerun: true,
	}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	childReq := httptest.NewRequest("POST", "/api/allocations/run", bytes.NewBuffer(body))
	childReq.Header.Set("Content-Type", "application/json")
	childReq = childReq.WithContext(context.WithValue(childReq.Context(), "user_id", 1))

	RunAutoAllocation(rr, childReq)

	w.WriteHeader(rr.Code)
	_, _ = w.Write(rr.Body.Bytes())
}