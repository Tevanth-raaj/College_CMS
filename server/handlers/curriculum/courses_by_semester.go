package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"server/db"
)

// Course represents a course in the curriculum
type Course struct {
	CourseID     string  `json:"course_id"`
	CourseName   string  `json:"course_name"`
	CourseType   string  `json:"course_type"` // theory, theory_with_lab, elective, open_elective, honour, minor
	Semester     int     `json:"semester"`
	Credits      float64 `json:"credits"`
	DepartmentID *int    `json:"department_id,omitempty"`
}

// GetCoursesBySemester returns all courses for a given semester
// This includes main courses, electives, open electives, honours, and minors
func GetCoursesBySemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	semesterStr := vars["semester"]
	semester, err := strconv.Atoi(semesterStr)
	if err != nil || semester < 1 || semester > 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid semester number. Must be between 1 and 8",
		})
		return
	}

	academicYear := r.URL.Query().Get("academic_year")
	log.Printf("Fetching courses for semester %d, academic year: %s", semester, academicYear)

	// Query to get all courses from the semester
	// This query fetches from the normal_cards (renamed from semesters) and courses tables
	query := `
		SELECT DISTINCT
			c.course_id,
			c.course_name,
			CASE 
				WHEN c.course_id LIKE '%L' OR c.course_id LIKE '%Lab%' THEN 'theory_with_lab'
				WHEN nc.card_type = 'Elective' THEN 'elective'
				WHEN nc.card_type = 'Open Elective' THEN 'open_elective'
				WHEN nc.card_type LIKE '%Honour%' THEN 'honour'
				WHEN nc.card_type LIKE '%Minor%' THEN 'minor'
				ELSE 'theory'
			END as course_type,
			nc.semester_no as semester,
			COALESCE(c.credits, 0) as credits,
			nc.department_id
		FROM normal_cards nc
		INNER JOIN courses c ON nc.id = c.card_id
		WHERE nc.semester_no = ?
			AND nc.is_deleted = 0
			AND c.is_deleted = 0
		ORDER BY c.course_id
	`

	rows, err := db.DB.Query(query, semester)
	if err != nil {
		log.Printf("Error fetching courses: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to fetch courses",
		})
		return
	}
	defer rows.Close()

	courses := []Course{}
	for rows.Next() {
		var course Course
		err := rows.Scan(
			&course.CourseID,
			&course.CourseName,
			&course.CourseType,
			&course.Semester,
			&course.Credits,
			&course.DepartmentID,
		)
		if err != nil {
			log.Printf("Error scanning course: %v", err)
			continue
		}
		courses = append(courses, course)
	}

	log.Printf("Found %d courses for semester %d", len(courses), semester)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"semester": semester,
		"count":    len(courses),
		"courses":  courses,
	})
}
