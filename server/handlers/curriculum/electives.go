package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"strconv"
	"strings"
	"time"
)

// GetHODProfile retrieves HOD's department and curriculum information
func GetHODProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	// Get user email from query or session (for now using query param)
	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Query to get HOD's department and curriculum
	query := `
		SELECT 
			u.id as user_id,
			u.username,
			u.email,
			u.role,
			d.id as dept_id,
			d.department_name,
			d.department_code,
			c.id as curr_id,
			c.name as curr_name,
			c.academic_year
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		LEFT JOIN curriculum c ON dca.curriculum_id = c.id
		WHERE u.email = ? AND u.role = 'hod' AND u.is_active = 1
		LIMIT 1
	`

	var response models.HODProfileResponse
	var deptID int
	var deptName string
	var deptCode sql.NullString
	var currID sql.NullInt64
	var currName, currYear sql.NullString

	err := db.DB.QueryRow(query, email).Scan(
		&response.UserID,
		&response.FullName,
		&response.Email,
		&response.Role,
		&deptID,
		&deptName,
		&deptCode,
		&currID,
		&currName,
		&currYear,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "HOD profile not found. Please contact admin to link your account.",
			})
			return
		}
		log.Println("Error fetching HOD profile:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}

	// Initialize nested structs
	response.Department = &models.DepartmentInfo{
		ID:   deptID,
		Name: deptName,
	}
	if deptCode.Valid {
		response.Department.Code = deptCode.String
	}

	if currID.Valid {
		response.Curriculum = &models.CurriculumInfo{
			ID:           int(currID.Int64),
			Name:         currName.String,
			AcademicYear: currYear.String,
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAvailableElectives retrieves available elective courses for a semester
func GetAvailableElectives(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	// Get parameters
	email := r.URL.Query().Get("email")
	batch := r.URL.Query().Get("batch")
	academicYear := r.URL.Query().Get("academic_year")

	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get HOD's department and curriculum
	var departmentID, curriculumID int
	deptQuery := `
		SELECT d.id, COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err := db.DB.QueryRow(deptQuery, email).Scan(&departmentID, &curriculumID)
	if err != nil {
		log.Println("Error fetching department:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Department not found",
		})
		return
	}

	// Query vertical card courses with semester assignments
	query := `
		SELECT 
			c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			nc.id as card_id,
			nc.id as vertical_id,
			COALESCE(nc.vertical_name, '') as vertical_name,
			nc.semester_number as vertical_semester,
			nc.card_type,
			CASE WHEN hes.id IS NOT NULL THEN 1 ELSE 0 END as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM courses c
		INNER JOIN curriculum_courses cc ON c.id = cc.course_id
		INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		LEFT JOIN hod_elective_selections hes ON (
			hes.course_id = c.id
			AND hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
		)
		LEFT JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE cc.curriculum_id = ?
			AND nc.card_type IN ('vertical', 'open_elective')
			AND c.status = 1
			AND nc.status = 1
		ORDER BY nc.semester_number IS NULL, nc.semester_number, c.course_type, c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, academicYear, batch, batch, curriculumID)
	if err != nil {
		log.Println("Error fetching electives:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	electives := []models.ElectiveCourse{}
	for rows.Next() {
		var course models.ElectiveCourse
		var isSelected int
		var assignedSemester sql.NullInt64
		var assignedSlotID sql.NullInt64
		var assignedSlot sql.NullString
		err := rows.Scan(
			&course.ID,
			&course.CourseCode,
			&course.CourseName,
			&course.CourseType,
			&course.Category,
			&course.Credit,
			&course.CardID,
			&course.VerticalID,
			&course.VerticalName,
			&course.VerticalSemester,
			&course.CardType,
			&isSelected,
			&assignedSemester,
			&assignedSlotID,
			&assignedSlot,
		)
		if err != nil {
			log.Println("Error scanning course:", err)
			continue
		}
		course.IsSelected = isSelected == 1
		if assignedSemester.Valid {
			sem := int(assignedSemester.Int64)
			course.AssignedSemester = &sem
		}
		if assignedSlotID.Valid {
			slotID := int(assignedSlotID.Int64)
			course.AssignedSlotID = &slotID
		}
		if assignedSlot.Valid {
			slot := assignedSlot.String
			course.AssignedSlot = &slot
		}
		electives = append(electives, course)
	}

	// Fetch department-specific semester courses for Add On electives
	deptCoursesQuery := `
		SELECT DISTINCT c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			nc.id as card_id,
			nc.id as vertical_id,
			'Department Courses' as vertical_name,
			nc.semester_number as vertical_semester,
			'department_course' as card_type,
			CASE WHEN hes.id IS NOT NULL THEN 1 ELSE 0 END as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM courses c
		INNER JOIN curriculum_courses cc ON c.id = cc.course_id
		INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		LEFT JOIN hod_elective_selections hes ON (
			hes.course_id = c.id
			AND hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
			AND hes.slot_id IN (SELECT id FROM elective_semester_slots WHERE slot_name = 'Add On')
		)
		LEFT JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE cc.curriculum_id = ?
			AND nc.card_type NOT IN ('vertical', 'elective', 'open_elective', 'honour')
			AND c.status = 1
			AND nc.status = 1
		ORDER BY nc.semester_number, c.course_code
	`

	deptRows, err := db.DB.Query(deptCoursesQuery, departmentID, academicYear, batch, batch, curriculumID)
	if err != nil {
		log.Println("Error fetching department courses:", err)
	} else {
		defer deptRows.Close()
		for deptRows.Next() {
			var course models.ElectiveCourse
			var isSelected int
			var assignedSemester sql.NullInt64
			var assignedSlotID sql.NullInt64
			var assignedSlot sql.NullString
			err := deptRows.Scan(
				&course.ID,
				&course.CourseCode,
				&course.CourseName,
				&course.CourseType,
				&course.Category,
				&course.Credit,
				&course.CardID,
				&course.VerticalID,
				&course.VerticalName,
				&course.VerticalSemester,
				&course.CardType,
				&isSelected,
				&assignedSemester,
				&assignedSlotID,
				&assignedSlot,
			)
			if err != nil {
				log.Println("Error scanning department course:", err)
				continue
			}
			course.IsSelected = isSelected == 1
			if assignedSemester.Valid {
				sem := int(assignedSemester.Int64)
				course.AssignedSemester = &sem
			}
			if assignedSlotID.Valid {
				slotID := int(assignedSlotID.Int64)
				course.AssignedSlotID = &slotID
			}
			if assignedSlot.Valid {
				slot := assignedSlot.String
				course.AssignedSlot = &slot
			}
			electives = append(electives, course)
		}
	}

	// Fetch courses from OTHER departments' PE slot selections (for Minor slot options)
	minorEligibleCourses := []models.MinorEligibleCourse{}
	minorQuery := `
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			d.id as department_id,
			d.department_name,
			COALESCE(d.department_code, '') as department_code,
			ess.slot_name,
			hes.semester
		FROM hod_elective_selections hes
		INNER JOIN courses c ON hes.course_id = c.id
		INNER JOIN departments d ON hes.department_id = d.id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE hes.department_id != ?
			AND hes.academic_year = ?
			AND hes.status = 'ACTIVE'
			AND LOWER(ess.slot_name) LIKE '%professional elective%'
		ORDER BY d.department_name, hes.semester, c.course_code
	`

	minorRows, err := db.DB.Query(minorQuery, departmentID, academicYear)
	if err != nil {
		log.Println("Error fetching minor eligible courses:", err)
	} else {
		defer minorRows.Close()
		for minorRows.Next() {
			var mc models.MinorEligibleCourse
			err := minorRows.Scan(
				&mc.ID,
				&mc.CourseCode,
				&mc.CourseName,
				&mc.CourseType,
				&mc.Category,
				&mc.Credit,
				&mc.DepartmentID,
				&mc.DepartmentName,
				&mc.DepartmentCode,
				&mc.SlotName,
				&mc.Semester,
			)
			if err != nil {
				log.Println("Error scanning minor eligible course:", err)
				continue
			}
			minorEligibleCourses = append(minorEligibleCourses, mc)
		}
	}

	// Fetch previously-assigned Minor courses for this department that aren't in the curriculum's verticals
	// (these are courses from other departments that were assigned to Minor slots)
	minorAssignedQuery := `
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			0 as card_id,
			0 as vertical_id,
			CONCAT(d_src.department_name, ' Minor') as vertical_name,
			NULL as vertical_semester,
			'minor_assigned' as card_type,
			1 as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM hod_elective_selections hes
		INNER JOIN courses c ON hes.course_id = c.id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		LEFT JOIN (
			SELECT hes2.course_id, d2.department_name
			FROM hod_elective_selections hes2
			INNER JOIN departments d2 ON hes2.department_id = d2.id
			INNER JOIN elective_semester_slots ess2 ON hes2.slot_id = ess2.id
			WHERE hes2.department_id != ?
				AND hes2.academic_year = ?
				AND hes2.status = 'ACTIVE'
				AND LOWER(ess2.slot_name) LIKE '%professional elective%'
		) d_src ON c.id = d_src.course_id
		WHERE hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
			AND LOWER(ess.slot_name) = 'minor'
			AND c.id NOT IN (
				SELECT cc2.course_id FROM curriculum_courses cc2
				INNER JOIN normal_cards nc2 ON cc2.semester_id = nc2.id
				WHERE cc2.curriculum_id = ? AND nc2.card_type IN ('vertical', 'open_elective') AND nc2.status = 1
			)
		ORDER BY c.course_code
	`

	minorAssignedRows, err := db.DB.Query(minorAssignedQuery, departmentID, academicYear, departmentID, academicYear, batch, batch, curriculumID)
	if err != nil {
		log.Println("Error fetching minor assigned courses:", err)
	} else {
		defer minorAssignedRows.Close()
		for minorAssignedRows.Next() {
			var course models.ElectiveCourse
			var isSelected int
			var assignedSemester sql.NullInt64
			var assignedSlotID sql.NullInt64
			var assignedSlot sql.NullString
			var verticalSemester sql.NullInt64
			err := minorAssignedRows.Scan(
				&course.ID,
				&course.CourseCode,
				&course.CourseName,
				&course.CourseType,
				&course.Category,
				&course.Credit,
				&course.CardID,
				&course.VerticalID,
				&course.VerticalName,
				&verticalSemester,
				&course.CardType,
				&isSelected,
				&assignedSemester,
				&assignedSlotID,
				&assignedSlot,
			)
			if err != nil {
				log.Println("Error scanning minor assigned course:", err)
				continue
			}
			course.IsSelected = isSelected == 1
			if verticalSemester.Valid {
				sem := int(verticalSemester.Int64)
				course.VerticalSemester = &sem
			}
			if assignedSemester.Valid {
				sem := int(assignedSemester.Int64)
				course.AssignedSemester = &sem
			}
			if assignedSlotID.Valid {
				slotID := int(assignedSlotID.Int64)
				course.AssignedSlotID = &slotID
			}
			if assignedSlot.Valid {
				slot := assignedSlot.String
				course.AssignedSlot = &slot
			}
			electives = append(electives, course)
		}
	}

	response := models.AvailableElectivesResponse{
		Semester:             0, // Not applicable - showing all vertical courses
		Batch:               batch,
		AcademicYear:        academicYear,
		AvailableElectives:  electives,
		MinorEligibleCourses: minorEligibleCourses,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SaveHODSelections saves HOD's elective course selections
func SaveHODSelections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	// Get email from query (in production, get from JWT token)
	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	var req models.SaveElectivesRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Error decoding request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// Validate semester assignments
	if len(req.CourseAssignments) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No course assignments provided",
		})
		return
	}

	for _, assignment := range req.CourseAssignments {
		if assignment.Semester < 4 || assignment.Semester > 8 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid semester (must be 4-8)",
			})
			return
		}
		if assignment.SlotID == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Slot ID is required for each assignment",
			})
			return
		}
	}

	// Honour courses validation
	// Get slot information to identify honour slots
	slotMap := make(map[int]string)
	slotQuery := "SELECT id, slot_name FROM elective_semester_slots WHERE id IN ("
	slotIDs := make([]interface{}, 0)
	for i, assignment := range req.CourseAssignments {
		if i > 0 {
			slotQuery += ","
		}
		slotQuery += "?"
		slotIDs = append(slotIDs, assignment.SlotID)
	}
	slotQuery += ")"

	if len(slotIDs) > 0 {
		slotRows, err := db.DB.Query(slotQuery, slotIDs...)
		if err != nil {
			log.Println("Error fetching slot names:", err)
		} else {
			defer slotRows.Close()
			for slotRows.Next() {
				var slotID int
				var slotName string
				if err := slotRows.Scan(&slotID, &slotName); err == nil {
					slotMap[slotID] = slotName
				}
			}
		}
	}

	// Count honour course assignments by semester and collect course IDs per semester
	honourCourseCountBySemester := make(map[int]int)
	honourCourseIDsBySemester := make(map[int][]int)
	professionalCourseIDsBySemester := make(map[int][]int)
	addOnCourseIDsBySemester := make(map[int][]int)

	for _, assignment := range req.CourseAssignments {
		slotName := slotMap[assignment.SlotID]
		sem := assignment.Semester
		if strings.Contains(strings.ToLower(slotName), "honour slot") {
			honourCourseCountBySemester[sem]++
			honourCourseIDsBySemester[sem] = append(honourCourseIDsBySemester[sem], assignment.CourseID)
		} else if strings.Contains(strings.ToLower(slotName), "professional elective") {
			professionalCourseIDsBySemester[sem] = append(professionalCourseIDsBySemester[sem], assignment.CourseID)
		} else if strings.Contains(strings.ToLower(slotName), "add on") {
			addOnCourseIDsBySemester[sem] = append(addOnCourseIDsBySemester[sem], assignment.CourseID)
		}
	}

	// Validate max 2 honour courses per semester
	for semester, count := range honourCourseCountBySemester {
		if count > 2 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Maximum 2 honour courses allowed per semester. Semester %d has %d honour courses.", semester, count),
			})
			return
		}
	}

	// Per-semester validations: no overlap between slot types within the same semester
	for sem := 4; sem <= 8; sem++ {
		honourIDs := honourCourseIDsBySemester[sem]
		profIDs := professionalCourseIDsBySemester[sem]
		addOnIDs := addOnCourseIDsBySemester[sem]

		// Honour vs Professional
		for _, hID := range honourIDs {
			for _, pID := range profIDs {
				if hID == pID {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": fmt.Sprintf("Semester %d: Honour and Professional Elective cannot share the same course. Please select different courses.", sem),
					})
					return
				}
			}
		}

		// Add On vs Professional
		for _, aID := range addOnIDs {
			for _, pID := range profIDs {
				if aID == pID {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": fmt.Sprintf("Semester %d: Add On and Professional Elective cannot share the same course. Please select different courses.", sem),
					})
					return
				}
			}
		}

		// Add On vs Honour
		for _, aID := range addOnIDs {
			for _, hID := range honourIDs {
				if aID == hID {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": fmt.Sprintf("Semester %d: Add On and Honour cannot share the same course. Please select different courses.", sem),
					})
					return
				}
			}
		}
	}

	// Get HOD's user ID, department, and curriculum
	var userID, departmentID, curriculumID int
	deptQuery := `
		SELECT u.id, d.id, COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err = db.DB.QueryRow(deptQuery, email).Scan(&userID, &departmentID, &curriculumID)
	if err != nil {
		log.Println("Error fetching HOD info:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "HOD profile not found",
		})
		return
	}

	if curriculumID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No active curriculum configured for your department",
		})
		return
	}

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer tx.Rollback()

	// Delete existing selections for this department/batch/year for all relevant semesters
	semesterSet := make(map[int]bool)
	for _, assignment := range req.CourseAssignments {
		semesterSet[assignment.Semester] = true
	}

	for semester := range semesterSet {
		deleteQuery := `
			DELETE FROM hod_elective_selections 
			WHERE department_id = ? 
			AND academic_year = ?
			AND semester = ?
			AND (batch = ? OR (batch IS NULL AND ? = ''))
		`
		_, err = tx.Exec(deleteQuery, departmentID, req.AcademicYear, semester, req.Batch, req.Batch)
		if err != nil {
			log.Println("Error deleting old selections for semester", semester, ":", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error clearing previous selections",
			})
			return
		}
	}

	// Insert user-provided selections with semester assignments
	if len(req.CourseAssignments) > 0 {
		insertQuery := `
			INSERT INTO hod_elective_selections 
			(department_id, curriculum_id, semester, course_id, slot_id, slot_name, academic_year, batch, approved_by_user_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, (SELECT slot_name FROM elective_semester_slots WHERE id = ?), ?, ?, ?, ?, ?, ?)
		`

		now := time.Now()
		status := req.Status
		if status == "" {
			status = "ACTIVE"
		}

		for _, assignment := range req.CourseAssignments {
			var batchVal interface{}
			if req.Batch != "" {
				batchVal = req.Batch
			} else {
				batchVal = nil
			}

			_, err = tx.Exec(insertQuery,
				departmentID,
				curriculumID,
				assignment.Semester,
				assignment.CourseID,
				assignment.SlotID,
				assignment.SlotID,
				req.AcademicYear,
				batchVal,
				userID,
				status,
				now,
				now,
			)
			if err != nil {
				log.Println("Error inserting selection:", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Error saving selections",
				})
				return
			}
		}
	}

	// Auto-add open elective courses to the last PE slot in semesters with >1 PE
	autoAllocatedCount := 0
	{
		// Get unique semesters from course assignments
		semesterSet := make(map[int]bool)
		for _, assignment := range req.CourseAssignments {
			semesterSet[assignment.Semester] = true
		}

		// Fetch open elective courses from curriculum
		openElectivesQuery := `
			SELECT DISTINCT c.id
			FROM courses c
			INNER JOIN curriculum_courses cc ON c.id = cc.course_id
			INNER JOIN normal_cards nc ON cc.semester_id = nc.id
			WHERE cc.curriculum_id = ?
				AND nc.card_type = 'open_elective'
				AND c.status = 1
				AND nc.status = 1
		`
		openElectiveRows, err := tx.Query(openElectivesQuery, curriculumID)
		if err != nil {
			log.Println("Error fetching open electives for auto-add:", err)
		} else {
			defer openElectiveRows.Close()

			openElectiveCourseIDs := []int{}
			for openElectiveRows.Next() {
				var courseID int
				if err := openElectiveRows.Scan(&courseID); err != nil {
					log.Println("Error scanning open elective course ID:", err)
					continue
				}
				openElectiveCourseIDs = append(openElectiveCourseIDs, courseID)
			}

			if len(openElectiveCourseIDs) > 0 {
				for semester := range semesterSet {
					// Fetch PE slots for this semester
					slotsQuery := `
						SELECT id, slot_name, slot_order
						FROM elective_semester_slots
						WHERE semester = ? AND LOWER(slot_name) LIKE '%professional elective%'
						ORDER BY slot_order
					`
					slotRows, err := tx.Query(slotsQuery, semester)
					if err != nil {
						log.Println("Error fetching PE slots:", err)
						continue
					}

					var peSlots []struct {
						ID        int
						SlotName  string
						SlotOrder int
					}
					for slotRows.Next() {
						var slotID, slotOrder int
						var slotName string
						if err := slotRows.Scan(&slotID, &slotName, &slotOrder); err != nil {
							log.Println("Error scanning PE slot:", err)
							continue
						}
						peSlots = append(peSlots, struct {
							ID        int
							SlotName  string
							SlotOrder int
						}{slotID, slotName, slotOrder})
					}
					slotRows.Close()

					// Only auto-add if >1 PE slot exists
					if len(peSlots) <= 1 {
						continue
					}

					// Find the last PE slot (highest slot_order)
					lastPE := peSlots[0]
					for _, slot := range peSlots {
						if slot.SlotOrder > lastPE.SlotOrder {
							lastPE = slot
						}
					}

					// Collect course IDs already assigned to this last PE slot
					existingCourses := make(map[int]bool)
					for _, assignment := range req.CourseAssignments {
						if assignment.Semester == semester && assignment.SlotID == lastPE.ID {
							existingCourses[assignment.CourseID] = true
						}
					}

					// Insert open elective courses into the last PE slot (skip duplicates)
					autoInsertQuery := `
						INSERT INTO hod_elective_selections 
						(department_id, curriculum_id, semester, course_id, slot_id, slot_name, academic_year, batch, approved_by_user_id, status, created_at, updated_at)
						VALUES (?, ?, ?, ?, ?, (SELECT slot_name FROM elective_semester_slots WHERE id = ?), ?, ?, ?, ?, ?, ?)
					`
					autoNow := time.Now()
					autoStatus := req.Status
					if autoStatus == "" {
						autoStatus = "ACTIVE"
					}
					var autoBatchVal interface{}
					if req.Batch != "" {
						autoBatchVal = req.Batch
					} else {
						autoBatchVal = nil
					}

					for _, oeCourseID := range openElectiveCourseIDs {
						if existingCourses[oeCourseID] {
							continue // Already assigned by HOD
						}
						_, err = tx.Exec(autoInsertQuery,
							departmentID,
							curriculumID,
							semester,
							oeCourseID,
							lastPE.ID,
							lastPE.ID,
							req.AcademicYear,
							autoBatchVal,
							userID,
							autoStatus,
							autoNow,
							autoNow,
						)
						if err != nil {
							log.Println("Error auto-adding open elective to last PE:", err)
							continue
						}
						autoAllocatedCount++
					}
				}
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Println("Error committing transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error saving selections",
		})
		return
	}

	message := "Elective selections saved successfully"
	if len(req.CourseAssignments) > 0 {
		message = fmt.Sprintf("%d elective courses assigned to semesters", len(req.CourseAssignments))
		if autoAllocatedCount > 0 {
			message = fmt.Sprintf("%s (%d open elective courses auto-added to last PE)", message, autoAllocatedCount)
		}
	} else {
		message = "All elective selections cleared"
	}

	response := models.SaveElectivesResponse{
		Success: true,
		Message: message,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetHODBatches retrieves all batches for the HOD's department
func GetHODBatches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get batches from academic_details table
	query := `
		SELECT DISTINCT ad.batch
		FROM academic_details ad
		INNER JOIN departments d ON ad.department = d.department_name
		INNER JOIN department_teachers dt ON d.id = dt.department_id
		INNER JOIN teachers t ON dt.teacher_id = t.faculty_id
		INNER JOIN users u ON t.email = u.email
		WHERE u.email = ? AND u.role = 'hod' AND ad.batch IS NOT NULL AND ad.batch != ''
		ORDER BY ad.batch DESC
	`

	rows, err := db.DB.Query(query, email)
	if err != nil {
		log.Println("Error fetching batches:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	var batches []string
	for rows.Next() {
		var batch string
		err := rows.Scan(&batch)
		if err != nil {
			log.Println("Error scanning batch:", err)
			continue
		}
		// Clean batch string
		batch = strings.TrimSpace(batch)
		if batch != "" {
			batches = append(batches, batch)
		}
	}

	// If no batches found, provide a default
	if len(batches) == 0 {
		batches = []string{"2024-2028", "2025-2029"}
	}

	response := models.BatchesResponse{
		Batches: batches,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetElectiveSemesterSlots returns available elective slots by semester
func GetElectiveSemesterSlots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	semesterParam := r.URL.Query().Get("semester")
	query := `
		SELECT id, semester, slot_name, slot_order
		FROM elective_semester_slots
		WHERE is_active = 1
	`
	args := []interface{}{}
	if semesterParam != "" {
		semester, err := strconv.Atoi(semesterParam)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Invalid semester",
			})
			return
		}
		query += " AND semester = ?"
		args = append(args, semester)
	}
	query += " ORDER BY semester, slot_order"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Println("Error fetching elective slots:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	slots := []models.ElectiveSemesterSlot{}
	for rows.Next() {
		var slot models.ElectiveSemesterSlot
		if err := rows.Scan(&slot.ID, &slot.Semester, &slot.SlotName, &slot.SlotOrder); err != nil {
			log.Println("Error scanning elective slot:", err)
			continue
		}
		slots = append(slots, slot)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"slots":   slots,
	})
}

// GetCurrentAcademicCalendar retrieves the current academic calendar
func GetCurrentAcademicCalendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	query := `
		SELECT id, academic_year, current_semester, semester_start_date, semester_end_date,
		       elective_selection_start, elective_selection_end, is_current, created_at, updated_at
		FROM academic_calendar
		WHERE is_current = 1
		ORDER BY current_semester
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Println("Error fetching academic calendar:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	calendars := []models.AcademicCalendar{}
	currentSemesters := []int{}
	for rows.Next() {
		var calendar models.AcademicCalendar
		if err := rows.Scan(
			&calendar.ID,
			&calendar.AcademicYear,
			&calendar.CurrentSemester,
			&calendar.SemesterStartDate,
			&calendar.SemesterEndDate,
			&calendar.ElectiveSelectionStart,
			&calendar.ElectiveSelectionEnd,
			&calendar.IsCurrent,
			&calendar.CreatedAt,
			&calendar.UpdatedAt,
		); err != nil {
			log.Println("Error scanning academic calendar:", err)
			continue
		}
		calendars = append(calendars, calendar)
		currentSemesters = append(currentSemesters, calendar.CurrentSemester)
	}

	if len(calendars) == 0 {
		// Return default if no calendar found
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"academic_year":    "2025-2026",
			"current_semester": 4,
		})
		return
	}

	if len(calendars) == 1 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(calendars[0])
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"academic_year":     calendars[0].AcademicYear,
		"current_semesters": currentSemesters,
		"calendars":         calendars,
	})
}

// ==================== Minor Program Management ====================

// GetMinorVerticals retrieves vertical cards from normal_cards for minor program selection
func GetMinorVerticals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get HOD's curriculum
	var curriculumID int
	currQuery := `
		SELECT COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err := db.DB.QueryRow(currQuery, email).Scan(&curriculumID)
	if err != nil || curriculumID == 0 {
		log.Println("Error fetching curriculum:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Curriculum not found",
		})
		return
	}

	// Get vertical cards from normal_cards with course counts
	query := `
		SELECT nc.id, 0 as honour_card_id, 
		       COALESCE(nc.vertical_name, '') as name,
		       COUNT(DISTINCT cc.course_id) as course_count
		FROM normal_cards nc
		LEFT JOIN curriculum_courses cc ON nc.id = cc.semester_id
		WHERE nc.curriculum_id = ? AND nc.card_type = 'vertical' AND nc.status = 1
		GROUP BY nc.id, nc.semester_number, nc.vertical_name
		HAVING COUNT(DISTINCT cc.course_id) >= 6
		ORDER BY nc.semester_number IS NULL, nc.semester_number, nc.vertical_name
	`

	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Println("Error fetching minor verticals:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	verticals := []models.MinorVerticalInfo{}
	for rows.Next() {
		var vertical models.MinorVerticalInfo
		err := rows.Scan(&vertical.ID, &vertical.HonourCardID, &vertical.Name, &vertical.CourseCount)
		if err != nil {
			log.Println("Error scanning vertical:", err)
			continue
		}
		verticals = append(verticals, vertical)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"verticals": verticals,
	})
}

// GetVerticalCourses retrieves courses for a specific vertical
func GetVerticalCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	verticalIDParam := r.URL.Query().Get("vertical_id")
	if verticalIDParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "vertical_id parameter required",
		})
		return
	}

	verticalID, err := strconv.Atoi(verticalIDParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid vertical_id",
		})
		return
	}

	query := `
		SELECT c.id, c.course_code, c.course_name, c.credit
		FROM courses c
		INNER JOIN curriculum_courses cc ON c.id = cc.course_id
		WHERE cc.semester_id = ? AND c.status = 1
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, verticalID)
	if err != nil {
		log.Println("Error fetching vertical courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	courses := []models.MinorCourse{}
	for rows.Next() {
		var course models.MinorCourse
		err := rows.Scan(&course.ID, &course.CourseCode, &course.CourseName, &course.Credit)
		if err != nil {
			log.Println("Error scanning course:", err)
			continue
		}
		courses = append(courses, course)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"courses": courses,
	})
}

// SaveHODMinorSelections saves HOD's minor program selections
func SaveHODMinorSelections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	var req models.SaveMinorRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Error decoding request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// Validate request
	if req.VerticalID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "vertical_id is required",
		})
		return
	}

	if len(req.AllowedDeptIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "At least one department must be allowed to take this minor",
		})
		return
	}

	if len(req.SemesterAssignments) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No semester assignments provided",
		})
		return
	}

	// Validate semester assignments (must be sem 5, 6, 7 with 2 courses each)
	semesterCounts := make(map[int]int)
	for _, assignment := range req.SemesterAssignments {
		if assignment.Semester < 5 || assignment.Semester > 7 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Minor courses must be in semesters 5, 6, or 7",
			})
			return
		}
		semesterCounts[assignment.Semester]++
	}

	for sem := 5; sem <= 7; sem++ {
		if semesterCounts[sem] != 2 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Each semester (5, 6, 7) must have exactly 2 courses. Semester %d has %d courses.", sem, semesterCounts[sem]),
			})
			return
		}
	}

	// Get HOD's info
	var userID, departmentID, curriculumID int
	deptQuery := `
		SELECT u.id, d.id, COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err = db.DB.QueryRow(deptQuery, email).Scan(&userID, &departmentID, &curriculumID)
	if err != nil {
		log.Println("Error fetching HOD info:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "HOD profile not found",
		})
		return
	}

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer tx.Rollback()

	// Delete existing minor selections for this dept/year/batch
	deleteQuery := `
		DELETE FROM hod_minor_selections 
		WHERE department_id = ? 
		AND academic_year = ?
		AND (batch = ? OR (batch IS NULL AND ? = ''))
	`
	_, err = tx.Exec(deleteQuery, departmentID, req.AcademicYear, req.Batch, req.Batch)
	if err != nil {
		log.Println("Error deleting old minor selections:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error clearing previous selections",
		})
		return
	}

	// Insert new minor selections
	insertQuery := `
		INSERT INTO hod_minor_selections 
		(department_id, curriculum_id, vertical_id, semester, course_id, allowed_dept_ids, academic_year, batch, approved_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	status := req.Status
	if status == "" {
		status = "ACTIVE"
	}

	// Convert allowed_dept_ids to JSON (same for all courses in this minor)
	deptIDsJSON, err := json.Marshal(req.AllowedDeptIDs)
	if err != nil {
		log.Println("Error marshaling dept IDs:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error processing department IDs",
		})
		return
	}

	for _, assignment := range req.SemesterAssignments {
		var batchVal interface{}
		if req.Batch != "" {
			batchVal = req.Batch
		} else {
			batchVal = nil
		}

		_, err = tx.Exec(insertQuery,
			departmentID,
			curriculumID,
			req.VerticalID,
			assignment.Semester,
			assignment.CourseID,
			string(deptIDsJSON),
			req.AcademicYear,
			batchVal,
			userID,
			status,
			now,
			now,
		)
		if err != nil {
			log.Println("Error inserting minor selection:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error saving minor selections",
			})
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Println("Error committing transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error saving minor selections",
		})
		return
	}

	response := models.SaveMinorResponse{
		Success: true,
		Message: fmt.Sprintf("Minor program saved successfully: %d courses across semesters 5-7", len(req.SemesterAssignments)),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetHODMinorSelections retrieves HOD's saved minor program selections
func GetHODMinorSelections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	academicYear := r.URL.Query().Get("academic_year")
	batch := r.URL.Query().Get("batch")

	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get HOD's department
	var departmentID int
	deptQuery := `
		SELECT d.id
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err := db.DB.QueryRow(deptQuery, email).Scan(&departmentID)
	if err != nil {
		log.Println("Error fetching department:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Department not found",
		})
		return
	}

	// Get minor selections
	query := `
		SELECT hms.vertical_id, 
		       CASE 
		           WHEN nc.semester_number IS NOT NULL THEN CONCAT('Semester ', nc.semester_number, ' - Vertical')
		           ELSE 'Vertical Card'
		       END as name,
		       hms.semester, hms.course_id, 
		       c.course_code, c.course_name, c.credit, hms.allowed_dept_ids
		FROM hod_minor_selections hms
		INNER JOIN normal_cards nc ON hms.vertical_id = nc.id
		INNER JOIN courses c ON hms.course_id = c.id
		WHERE hms.department_id = ?
		AND hms.academic_year = ?
		AND (hms.batch = ? OR (hms.batch IS NULL AND ? = ''))
		AND hms.status = 'ACTIVE'
		ORDER BY hms.semester, c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, academicYear, batch, batch)
	if err != nil {
		log.Println("Error fetching minor selections:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	var verticalID int
	var verticalName string
	assignments := []models.MinorAssignmentInfo{}

	for rows.Next() {
		var tempVerticalID int
		var tempVerticalName string
		var assignment models.MinorAssignmentInfo
		var deptIDsJSON string

		err := rows.Scan(&tempVerticalID, &tempVerticalName, &assignment.Semester, &assignment.CourseID,
			&assignment.CourseCode, &assignment.CourseName, &assignment.Credit, &deptIDsJSON)
		if err != nil {
			log.Println("Error scanning minor selection:", err)
			continue
		}

		// Parse JSON allowed_dept_ids
		if err := json.Unmarshal([]byte(deptIDsJSON), &assignment.AllowedDeptIDs); err != nil {
			log.Println("Error parsing dept IDs:", err)
			assignment.AllowedDeptIDs = []int{}
		}

		verticalID = tempVerticalID
		verticalName = tempVerticalName
		assignments = append(assignments, assignment)
	}

	response := models.MinorSelectionResponse{
		VerticalID:   verticalID,
		VerticalName: verticalName,
		AcademicYear: academicYear,
		Batch:        batch,
		Assignments:  assignments,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"selection": response,
	})
}

// ==================== Open Elective Offering Handlers ====================

// GetOECards retrieves open elective cards available for offering
func GetOECards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get HOD's curriculum
	var curriculumID int
	currQuery := `
		SELECT COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err := db.DB.QueryRow(currQuery, email).Scan(&curriculumID)
	if err != nil || curriculumID == 0 {
		log.Println("Error fetching curriculum:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Curriculum not found",
		})
		return
	}

	// Get open_elective cards with course counts
	query := `
		SELECT nc.id,
		       COALESCE(nc.vertical_name, CONCAT('Open Elective - Semester ', nc.semester_number), 'Open Elective') as name,
		       nc.semester_number,
		       COUNT(DISTINCT cc.course_id) as course_count
		FROM normal_cards nc
		LEFT JOIN curriculum_courses cc ON nc.id = cc.semester_id
		WHERE nc.curriculum_id = ? AND nc.card_type = 'open_elective' AND nc.status = 1
		GROUP BY nc.id, nc.semester_number, nc.vertical_name
		HAVING COUNT(DISTINCT cc.course_id) >= 1
		ORDER BY nc.semester_number IS NULL, nc.semester_number, nc.vertical_name
	`

	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Println("Error fetching OE cards:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	cards := []models.OECardInfo{}
	for rows.Next() {
		var card models.OECardInfo
		var semesterNumber sql.NullInt64
		var cardName sql.NullString
		err := rows.Scan(&card.ID, &cardName, &semesterNumber, &card.CourseCount)
		if err != nil {
			log.Println("Error scanning OE card:", err)
			continue
		}
		if cardName.Valid {
			card.Name = cardName.String
		} else {
			card.Name = "Open Elective"
		}
		if semesterNumber.Valid {
			sem := int(semesterNumber.Int64)
			card.Semester = &sem
		}
		cards = append(cards, card)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"cards":   cards,
	})
}

// SaveHODOEOfferings saves HOD's open elective offerings to other departments
func SaveHODOEOfferings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	var req models.SaveOEOfferingRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Error decoding request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	// Validate request
	if req.OECardID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "oe_card_id is required",
		})
		return
	}

	if len(req.AllowedDeptIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "At least one department must be allowed",
		})
		return
	}

	if len(req.SemesterAssignments) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No course assignments provided",
		})
		return
	}

	// Validate semester range (5-7)
	for _, assignment := range req.SemesterAssignments {
		if assignment.Semester < 5 || assignment.Semester > 7 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Open Elective courses must be offered in semesters 5, 6, or 7",
			})
			return
		}
	}

	// Get HOD's info
	var userID, departmentID, curriculumID int
	deptQuery := `
		SELECT u.id, d.id, COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err = db.DB.QueryRow(deptQuery, email).Scan(&userID, &departmentID, &curriculumID)
	if err != nil {
		log.Println("Error fetching HOD info:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "HOD profile not found",
		})
		return
	}

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer tx.Rollback()

	// Delete existing OE offerings for this dept/year/batch
	deleteQuery := `
		DELETE FROM hod_oe_offerings 
		WHERE department_id = ? 
		AND academic_year = ?
		AND (batch = ? OR (batch IS NULL AND ? = ''))
	`
	_, err = tx.Exec(deleteQuery, departmentID, req.AcademicYear, req.Batch, req.Batch)
	if err != nil {
		log.Println("Error deleting old OE offerings:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error clearing previous offerings",
		})
		return
	}

	// Insert new OE offerings
	insertQuery := `
		INSERT INTO hod_oe_offerings 
		(department_id, curriculum_id, oe_card_id, semester, course_id, allowed_dept_ids, academic_year, batch, approved_by_user_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	status := req.Status
	if status == "" {
		status = "ACTIVE"
	}

	// Convert allowed_dept_ids to JSON
	deptIDsJSON, err := json.Marshal(req.AllowedDeptIDs)
	if err != nil {
		log.Println("Error marshaling dept IDs:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error processing department IDs",
		})
		return
	}

	for _, assignment := range req.SemesterAssignments {
		var batchVal interface{}
		if req.Batch != "" {
			batchVal = req.Batch
		} else {
			batchVal = nil
		}

		_, err = tx.Exec(insertQuery,
			departmentID,
			curriculumID,
			req.OECardID,
			assignment.Semester,
			assignment.CourseID,
			string(deptIDsJSON),
			req.AcademicYear,
			batchVal,
			userID,
			status,
			now,
			now,
		)
		if err != nil {
			log.Println("Error inserting OE offering:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Error saving OE offerings",
			})
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Println("Error committing transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Error saving OE offerings",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Open Elective offerings saved successfully: %d courses across semesters", len(req.SemesterAssignments)),
	})
}

// GetHODOEOfferings retrieves HOD's saved open elective offerings
func GetHODOEOfferings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	email := r.URL.Query().Get("email")
	academicYear := r.URL.Query().Get("academic_year")
	batch := r.URL.Query().Get("batch")

	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Email parameter required",
		})
		return
	}

	// Get HOD's department
	var departmentID int
	deptQuery := `
		SELECT d.id
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		WHERE u.email = ? AND u.role = 'hod'
		LIMIT 1
	`

	err := db.DB.QueryRow(deptQuery, email).Scan(&departmentID)
	if err != nil {
		log.Println("Error fetching department:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Department not found",
		})
		return
	}

	// Get OE offerings
	query := `
		SELECT oe.oe_card_id, 
		       COALESCE(nc.vertical_name, CONCAT('Open Elective - Semester ', nc.semester_number), 'Open Elective') as card_name,
		       oe.semester, oe.course_id, 
		       c.course_code, c.course_name, c.credit, oe.allowed_dept_ids
		FROM hod_oe_offerings oe
		INNER JOIN normal_cards nc ON oe.oe_card_id = nc.id
		INNER JOIN courses c ON oe.course_id = c.id
		WHERE oe.department_id = ?
		AND oe.academic_year = ?
		AND (oe.batch = ? OR (oe.batch IS NULL AND ? = ''))
		AND oe.status = 'ACTIVE'
		ORDER BY oe.semester, c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, academicYear, batch, batch)
	if err != nil {
		log.Println("Error fetching OE offerings:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	var oeCardID int
	var oeCardName string
	assignments := []models.OEAssignmentInfo{}

	for rows.Next() {
		var tempCardID int
		var tempCardName string
		var assignment models.OEAssignmentInfo
		var deptIDsJSON string

		err := rows.Scan(&tempCardID, &tempCardName, &assignment.Semester, &assignment.CourseID,
			&assignment.CourseCode, &assignment.CourseName, &assignment.Credit, &deptIDsJSON)
		if err != nil {
			log.Println("Error scanning OE offering:", err)
			continue
		}

		// Parse JSON allowed_dept_ids
		if err := json.Unmarshal([]byte(deptIDsJSON), &assignment.AllowedDeptIDs); err != nil {
			log.Println("Error parsing dept IDs:", err)
			assignment.AllowedDeptIDs = []int{}
		}

		oeCardID = tempCardID
		oeCardName = tempCardName
		assignments = append(assignments, assignment)
	}

	response := models.OEOfferingResponse{
		OECardID:     oeCardID,
		OECardName:   oeCardName,
		AcademicYear: academicYear,
		Batch:        batch,
		Assignments:  assignments,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"offering": response,
	})
}
