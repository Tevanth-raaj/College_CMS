package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"server/db"
	"server/models"
)

// GetCoursesForTeacherSemester returns core + extra courses (split by source)
// Core: from curriculum_courses
// Extra: only courses students in department chose via student_elective_choices
func GetCoursesForTeacherSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	teacherIDStr := vars["teacherId"]
	semStr := vars["semester"]

	teacherID, err := strconv.Atoi(teacherIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid teacher ID"})
		return
	}
	semesterNum, err := strconv.Atoi(semStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid semester number"})
		return
	}

	// Get teacher's department
	var deptRaw sql.NullString
	var facultyID sql.NullString
	err = db.DB.QueryRow("SELECT dept, faculty_id FROM teachers WHERE id = ?", teacherID).Scan(&deptRaw, &facultyID)
	if err != nil {
		log.Printf("Error fetching teacher data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch teacher data"})
		return
	}

	var deptID int
	if deptRaw.Valid && deptRaw.String != "" {
		if id, convErr := strconv.Atoi(deptRaw.String); convErr == nil {
			deptID = id
		}
	}

	if deptID == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not determine teacher's department"})
		return
	}

	log.Printf("Getting core + extra courses for teacher in department %d, semester %d", deptID, semesterNum)

	requestedAcademicYear := strings.TrimSpace(r.URL.Query().Get("academic_year"))
	currentAcademicYear := requestedAcademicYear
	if currentAcademicYear == "" {
		err = db.DB.QueryRow(`
			SELECT academic_year
			FROM academic_calendar
			WHERE is_current = 1
			ORDER BY updated_at DESC, id DESC
			LIMIT 1
		`).Scan(&currentAcademicYear)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Warning: could not resolve academic year from academic_calendar: %v", err)
			}
			currentAcademicYear = ""
		}
	}

	candidateAcademicYears := make([]string, 0, 3)
	seenAcademicYears := make(map[string]struct{})
	appendAcademicYear := func(year string) {
		year = strings.TrimSpace(year)
		if year == "" {
			return
		}
		if _, exists := seenAcademicYears[year]; exists {
			return
		}
		seenAcademicYears[year] = struct{}{}
		candidateAcademicYears = append(candidateAcademicYears, year)
	}
	appendAcademicYear(requestedAcademicYear)
	appendAcademicYear(currentAcademicYear)
	if len(currentAcademicYear) >= 9 && currentAcademicYear[4] == '-' {
		endYearStr := currentAcademicYear[5:9]
		if endYear, convErr := strconv.Atoi(endYearStr); convErr == nil {
			nextStartYear := endYear
			nextEndYear := endYear + 1
			appendAcademicYear(strconv.Itoa(nextStartYear) + "-" + strconv.Itoa(nextEndYear))
		}
	}
	if len(candidateAcademicYears) == 0 {
		appendAcademicYear("2025-2026")
	}
	log.Printf("📅 Academic year candidates for extra courses: %v", candidateAcademicYears)

	// ===== CATEGORY 1: CORE COURSES from curriculum =====
	// Get curriculum for this department and semester
	// Core courses are those in curriculum_courses for this department
	coreCourseQuery := `
		SELECT DISTINCT c.id, c.course_code, c.course_name, c.course_type as course_type_id, ct.course_type,
		       c.category, c.credit, 
		       c.lecture_hrs, c.tutorial_hrs, c.practical_hrs, c.activity_hrs, COALESCE(c.` + "`tw/sl`" + `, 0) as tw_sl,
		       COALESCE(c.theory_total_hrs,0), COALESCE(c.tutorial_total_hrs,0), COALESCE(c.practical_total_hrs,0), COALESCE(c.activity_total_hrs,0), COALESCE(c.total_hrs,0),
		       c.cia_marks, c.see_marks, c.total_marks,
		       cc.id as reg_course_id, 1 as count_towards_limit,
		       'core' as card_type
		FROM courses c
		LEFT JOIN course_type ct ON c.course_type = ct.id
		INNER JOIN curriculum_courses cc ON c.id = cc.course_id
		INNER JOIN curriculum cur ON cc.curriculum_id = cur.id
		INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		INNER JOIN departments d ON cur.id = d.current_curriculum_id
		WHERE d.id = ?
		  AND nc.semester_number = ?
		  AND nc.card_type = 'semester'
		  AND nc.status = 1
		  AND (c.status = 1 OR c.status IS NULL)
		  AND c.category NOT LIKE '%Elective%'
		  AND c.category NOT LIKE '%Open%'
		  AND c.category NOT LIKE '%Honour%'
		  AND cur.status = 1
		ORDER BY c.course_code
	`

	log.Printf("Executing core course query for department %d, semester %d", deptID, semesterNum)
	rows, err := db.DB.Query(coreCourseQuery, deptID, semesterNum)
	if err != nil {
		log.Printf("❌ Error querying core courses: %v", err)
		// Continue with empty list
	} else {
		log.Printf("✅ Core course query executed successfully")
		defer rows.Close()
	}

	// Initialize as empty slice so JSON returns [] not null
	coreCourses := make([]models.CourseWithDetails, 0)
	if rows != nil {
		for rows.Next() {
			var course models.CourseWithDetails
			var countTowardsLimitInt int
			var cardType string
			var courseTypeID sql.NullInt64
			err := rows.Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &courseTypeID, &course.CourseType, &course.Category, &course.Credit,
				&course.LectureHrs, &course.TutorialHrs, &course.PracticalHrs, &course.ActivityHrs, &course.TwSlHrs,
				&course.TheoryTotalHrs, &course.TutorialTotalHrs, &course.PracticalTotalHrs, &course.ActivityTotalHrs, &course.TotalHrs,
				&course.CIAMarks, &course.SEEMarks, &course.TotalMarks,
				&course.RegCourseID, &countTowardsLimitInt, &cardType)
			if err != nil {
				log.Printf("Error scanning core course: %v", err)
				continue
			}
			countTowardsLimit := countTowardsLimitInt == 1
			course.CountTowardsLimit = &countTowardsLimit
			coreCourses = append(coreCourses, course)
		}
	}
	log.Printf("📘 Found %d core courses for semester %d", len(coreCourses), semesterNum)

	// ===== CATEGORY 2: EXTRA COURSES (electives, honours, minors, open electives, add-ons) =====
	// Filter strictly from student_elective_choices (dept + semester + academic year).
	// hod_elective_selections is used only to map hod_selection_id -> course details.
	extraCourseQuery := `
		SELECT DISTINCT c.id, c.course_code, c.course_name, c.course_type as course_type_id, ct.course_type,
		       c.category, c.credit, 
		       c.lecture_hrs, c.tutorial_hrs, c.practical_hrs, c.activity_hrs, COALESCE(c.` + "`tw/sl`" + `, 0) as tw_sl,
		       COALESCE(c.theory_total_hrs,0), COALESCE(c.tutorial_total_hrs,0), COALESCE(c.practical_total_hrs,0), COALESCE(c.activity_total_hrs,0), COALESCE(c.total_hrs,0),
		       c.cia_marks, c.see_marks, c.total_marks,
		       hes.id as reg_course_id, 1 as count_towards_limit,
		       hes.slot_name as card_type
		FROM courses c
		LEFT JOIN course_type ct ON c.course_type = ct.id
		INNER JOIN hod_elective_selections hes ON c.id = hes.course_id
		INNER JOIN student_elective_choices sec ON hes.id = sec.hod_selection_id
		WHERE hes.department_id = ? 
		  AND sec.semester = ?
		  AND sec.academic_year = ?
		  AND hes.status = 'ACTIVE'
		  AND (c.status = 1 OR c.status IS NULL)
		ORDER BY c.course_code
	`

	fetchExtraCourses := func(academicYear string) ([]models.CourseWithDetails, int, error) {
		log.Printf("Executing extra course query for department %d, semester %d, academic year %s", deptID, semesterNum, academicYear)
		log.Printf("🔍 Query will join: courses -> hod_elective_selections -> student_elective_choices")
		log.Printf("🔍 Looking for: hes.department_id = %d, sec.semester = %d, sec.academic_year = %s", deptID, semesterNum, academicYear)

		var debugCount int
		if err := db.DB.QueryRow(`
			SELECT COUNT(DISTINCT sec.hod_selection_id)
			FROM student_elective_choices sec
			INNER JOIN hod_elective_selections hes ON sec.hod_selection_id = hes.id
			WHERE hes.department_id = ?
			  AND sec.semester = ?
			  AND sec.academic_year = ?
			  AND hes.status = 'ACTIVE'
		`, deptID, semesterNum, academicYear).Scan(&debugCount); err != nil {
			log.Printf("Warning: could not count student elective choices for %s: %v", academicYear, err)
		}
		log.Printf("🔍 DEBUG: Found %d student elective choices for dept %d, semester %d, year %s", debugCount, deptID, semesterNum, academicYear)

		rows2, queryErr := db.DB.Query(extraCourseQuery, deptID, semesterNum, academicYear)
		if queryErr != nil {
			return nil, debugCount, queryErr
		}
		defer rows2.Close()

		extraCoursesForYear := make([]models.CourseWithDetails, 0)
		for rows2.Next() {
			var course models.CourseWithDetails
			var countTowardsLimitInt int
			var cardType string
			var courseTypeID sql.NullInt64
			err := rows2.Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &courseTypeID, &course.CourseType, &course.Category, &course.Credit,
				&course.LectureHrs, &course.TutorialHrs, &course.PracticalHrs, &course.ActivityHrs, &course.TwSlHrs,
				&course.TheoryTotalHrs, &course.TutorialTotalHrs, &course.PracticalTotalHrs, &course.ActivityTotalHrs, &course.TotalHrs,
				&course.CIAMarks, &course.SEEMarks, &course.TotalMarks,
				&course.RegCourseID, &countTowardsLimitInt, &cardType)
			if err != nil {
				log.Printf("Error scanning extra course: %v", err)
				continue
			}
			countTowardsLimit := countTowardsLimitInt == 1
			course.CountTowardsLimit = &countTowardsLimit
			extraCoursesForYear = append(extraCoursesForYear, course)
		}

		return extraCoursesForYear, debugCount, nil
	}

	// Initialize as empty slice so JSON returns [] not null
	extraCourses := make([]models.CourseWithDetails, 0)
	extraCoursesAcademicYear := ""
	for _, academicYear := range candidateAcademicYears {
		coursesForYear, debugCount, queryErr := fetchExtraCourses(academicYear)
		if queryErr != nil {
			log.Printf("❌ Error querying extra courses for academic year %s: %v", academicYear, queryErr)
			continue
		}
		log.Printf("✅ Extra course query executed successfully for %s", academicYear)
		if len(coursesForYear) > 0 || debugCount > 0 {
			extraCourses = coursesForYear
			extraCoursesAcademicYear = academicYear
			break
		}
	}
	log.Printf("✨ Found %d extra courses for department %d, semester %d", len(extraCourses), deptID, semesterNum)

	log.Printf("📊 Total courses: %d core + %d extra = %d", len(coreCourses), len(extraCourses), len(coreCourses)+len(extraCourses))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"coreCourses":              coreCourses,
		"extraCourses":             extraCourses,
		"extraCoursesAcademicYear": extraCoursesAcademicYear,
		"message":                  "Core courses required for all; Extra courses are only those students in your department have enrolled in",
	})
}
