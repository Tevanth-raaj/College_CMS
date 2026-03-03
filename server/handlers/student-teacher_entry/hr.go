package studentteacher

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"server/db"
)

// FacultyWithDepartment represents a faculty member with department details
type FacultyWithDepartment struct {
	ID                         uint64         `json:"id"`
	FacultyID                  string         `json:"faculty_id"`
	Name                       string         `json:"name"`
	Email                      string         `json:"email"`
	Phone                      sql.NullString `json:"phone"`
	ProfileImg                 sql.NullString `json:"profile_img"`
	Dept                       sql.NullString `json:"dept"`
	Desg                       sql.NullString `json:"desg"`
	LastLogin                  sql.NullTime   `json:"last_login"`
	Status                     bool           `json:"status"`
	DepartmentID               sql.NullInt64  `json:"department_id"`
	DepartmentCode             sql.NullString `json:"department_code"`
	DepartmentName             sql.NullString `json:"department_name"`
	CourseLimits               []TeacherLimit `json:"course_limits"`
}

type TeacherLimit struct {
	CourseTypeID int    `json:"course_type_id"`
	TypeName     string `json:"type_name"`
	MaxCount     int    `json:"max_count"`
}

// GetAllFaculty retrieves all faculty members with their department information
func GetAllFaculty(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := `
		SELECT 
			t.id,
			t.faculty_id,
			t.name,
			t.email,
			t.phone,
			t.profile_img,
			t.dept,
			t.desg,
			t.last_login,
			t.status,
			d.id as department_id,
			d.department_code,
			d.department_name
		FROM teachers t
		LEFT JOIN departments d ON t.dept = d.id
		ORDER BY t.name ASC
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Printf("Error querying faculty: %v", err)
		http.Error(w, "Failed to retrieve faculty data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var faculty []FacultyWithDepartment
	for rows.Next() {
		var f FacultyWithDepartment
		err := rows.Scan(
			&f.ID,
			&f.FacultyID,
			&f.Name,
			&f.Email,
			&f.Phone,
			&f.ProfileImg,
			&f.Dept,
			&f.Desg,
			&f.LastLogin,
			&f.Status,
			&f.DepartmentID,
			&f.DepartmentCode,
			&f.DepartmentName,
		)
		if err != nil {
			log.Printf("Error scanning faculty row: %v", err)
			continue
		}

		// Fetch limits for this faculty using alphanumeric faculty_id
		f.CourseLimits = getTeacherLimits(f.FacultyID)
		
		faculty = append(faculty, f)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating faculty rows: %v", err)
		http.Error(w, "Failed to process faculty data", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully retrieved %d faculty members", len(faculty))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"faculty": faculty,
		"count":   len(faculty),
	})
}

func getTeacherLimits(facultyID string) []TeacherLimit {
	query := `
		SELECT ct.id, ct.course_type, COALESCE(tcl.max_count, 0)
		FROM course_type ct
		LEFT JOIN teacher_course_limits tcl ON ct.id = tcl.course_type_id AND tcl.teacher_id = ?
		ORDER BY ct.id
	`
	rows, err := db.DB.Query(query, facultyID)
	if err != nil {
		log.Printf("Error fetching teacher limits: %v", err)
		return []TeacherLimit{}
	}
	defer rows.Close()

	var limits []TeacherLimit
	for rows.Next() {
		var l TeacherLimit
		if err := rows.Scan(&l.CourseTypeID, &l.TypeName, &l.MaxCount); err != nil {
			continue
		}
		limits = append(limits, l)
	}
	return limits
}

// UpdateFacultySubjectCounts updates the theory subject counts for a faculty member
func UpdateFacultySubjectCounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		FacultyID    string         `json:"faculty_id"` // Changed to string for alphanumeric ID
		CourseLimits []TeacherLimit `json:"course_limits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, limit := range req.CourseLimits {
		query := `
			INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE max_count = VALUES(max_count)
		`
		_, err := tx.Exec(query, req.FacultyID, limit.CourseTypeID, limit.MaxCount)
		if err != nil {
			log.Printf("Error updating limit: %v", err)
			http.Error(w, "Failed to update limits", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to commit changes", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated subject counts for faculty ID %d", req.FacultyID)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Subject counts updated successfully",
	})
}
