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
        ORDER BY COALESCE(dca.curriculum_id, 0) DESC
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
			AND nc.card_type IN ('vertical')
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

	// Fetch courses offered TO this department by other departments via Minor Program Management
	minorEligibleCourses := []models.MinorEligibleCourse{}
	deptIDStr := fmt.Sprintf("%d", departmentID)

	// Fetch own curriculum Minor card courses (configured during curriculum setup).
	minorCardQuery := `
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			hv.id as card_id,
			hv.id as vertical_id,
			CONCAT(hc.title, ' - ', hv.name) as vertical_name,
			hc.title,
			hv.name,
			NULL as vertical_semester,
			'minor_card' as card_type,
			CASE WHEN hes.id IS NOT NULL THEN 1 ELSE 0 END as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM honour_verticals hv
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		INNER JOIN honour_vertical_courses hvc ON hv.id = hvc.honour_vertical_id AND hvc.status = 1
		INNER JOIN courses c ON hvc.course_id = c.id
		LEFT JOIN hod_elective_selections hes ON (
			hes.course_id = c.id
			AND hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
		)
		LEFT JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE hc.curriculum_id = ?
			AND hc.status = 1
			AND hv.status = 1
			AND UPPER(hc.title) LIKE '%MINOR%'
			AND c.status = 1
		ORDER BY hc.title, hv.name, c.course_code
	`

	minorCardRows, err := db.DB.Query(minorCardQuery, departmentID, academicYear, batch, batch, curriculumID)
	if err != nil {
		log.Println("Error fetching minor card courses:", err)
	} else {
		defer minorCardRows.Close()
		for minorCardRows.Next() {
			var course models.ElectiveCourse
			var isSelected int
			var assignedSemester sql.NullInt64
			var assignedSlotID sql.NullInt64
			var assignedSlot sql.NullString
			var verticalSemester sql.NullInt64
			var cardTitle sql.NullString
			var verticalName sql.NullString
			err := minorCardRows.Scan(
				&course.ID,
				&course.CourseCode,
				&course.CourseName,
				&course.CourseType,
				&course.Category,
				&course.Credit,
				&course.CardID,
				&course.VerticalID,
				&course.VerticalName,
				&cardTitle,
				&verticalName,
				&verticalSemester,
				&course.CardType,
				&isSelected,
				&assignedSemester,
				&assignedSlotID,
				&assignedSlot,
			)
			if err != nil {
				log.Println("Error scanning minor card course:", err)
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

	// Fetch honour card courses (from honour_vertical_courses under HONOUR cards)
	honourCardQuery := `
		SELECT DISTINCT
			c.id,
			c.course_code,
			c.course_name,
			c.course_type,
			COALESCE(c.category, '') as category,
			COALESCE(c.credit, 0) as credit,
			hv.id as card_id,
			hv.id as vertical_id,
			CONCAT(hc.title, ' - ', hv.name) as vertical_name,
			hc.title,
			hv.name,
			NULL as vertical_semester,
			'honour_card' as card_type,
			CASE WHEN hes.id IS NOT NULL THEN 1 ELSE 0 END as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM honour_verticals hv
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		INNER JOIN honour_vertical_courses hvc ON hv.id = hvc.honour_vertical_id AND hvc.status = 1
		INNER JOIN courses c ON hvc.course_id = c.id
		LEFT JOIN hod_elective_selections hes ON (
			hes.course_id = c.id
			AND hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
		)
		LEFT JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE hc.curriculum_id = ?
			AND hc.status = 1
			AND hv.status = 1
			AND UPPER(hc.title) NOT LIKE '%MINOR%'
			AND c.status = 1
		ORDER BY hc.title, hv.name, c.course_code
	`

	honourCardRows, err := db.DB.Query(honourCardQuery, departmentID, academicYear, batch, batch, curriculumID)
	if err != nil {
		log.Println("Error fetching honour card courses:", err)
	} else {
		defer honourCardRows.Close()
		for honourCardRows.Next() {
			var course models.ElectiveCourse
			var isSelected int
			var assignedSemester sql.NullInt64
			var assignedSlotID sql.NullInt64
			var assignedSlot sql.NullString
			var verticalSemester sql.NullInt64
			var cardTitle sql.NullString
			var verticalName sql.NullString
			err := honourCardRows.Scan(
				&course.ID,
				&course.CourseCode,
				&course.CourseName,
				&course.CourseType,
				&course.Category,
				&course.Credit,
				&course.CardID,
				&course.VerticalID,
				&course.VerticalName,
				&cardTitle,
				&verticalName,
				&verticalSemester,
				&course.CardType,
				&isSelected,
				&assignedSemester,
				&assignedSlotID,
				&assignedSlot,
			)
			if err != nil {
				log.Println("Error scanning honour card course:", err)
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
			'Minor Offering' as slot_name,
			hms.semester
		FROM hod_minor_selections hms
		INNER JOIN courses c ON hms.course_id = c.id
		INNER JOIN departments d ON hms.department_id = d.id
		WHERE hms.department_id != ?
			AND hms.academic_year = ?
			AND hms.status = 'ACTIVE'
			AND JSON_CONTAINS(hms.allowed_dept_ids, ?)
		ORDER BY d.department_name, hms.semester, c.course_code
	`

	minorRows, err := db.DB.Query(minorQuery, departmentID, academicYear, deptIDStr)
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

	// Fetch previously-assigned Minor Slot courses for this department
	// (courses from other departments that were assigned to Minor Slot 1/2)
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
			COALESCE(
				(SELECT CONCAT(d2.department_name, ' Minor')
				 FROM hod_minor_selections hms2
				 INNER JOIN departments d2 ON hms2.department_id = d2.id
				 WHERE hms2.course_id = c.id
				   AND hms2.department_id != ?
				   AND hms2.status = 'ACTIVE'
				 LIMIT 1),
				'Minor Offering'
			) as vertical_name,
			NULL as vertical_semester,
			'minor_assigned' as card_type,
			1 as is_selected,
			hes.semester as assigned_semester,
			hes.slot_id as assigned_slot_id,
			ess.slot_name as assigned_slot
		FROM hod_elective_selections hes
		INNER JOIN courses c ON hes.course_id = c.id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE hes.department_id = ?
			AND hes.academic_year = ?
			AND (hes.batch = ? OR hes.batch IS NULL OR ? = '')
			AND hes.status = 'ACTIVE'
			AND LOWER(ess.slot_name) LIKE '%minor slot%'
			AND c.id NOT IN (
				SELECT cc2.course_id FROM curriculum_courses cc2
				INNER JOIN normal_cards nc2 ON cc2.semester_id = nc2.id
				WHERE cc2.curriculum_id = ? AND nc2.card_type IN ('vertical', 'open_elective') AND nc2.status = 1
			)
		ORDER BY c.course_code
	`

	minorAssignedRows, err := db.DB.Query(minorAssignedQuery, departmentID, departmentID, academicYear, batch, batch, curriculumID)
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
	minorCourseCountBySemester := make(map[int]int)
	minorCourseIDsBySemester := make(map[int][]int)
	professionalCourseIDsBySemester := make(map[int][]int)
	addOnCourseIDsBySemester := make(map[int][]int)

	// Track slot usage for honour and minor slots (each specific slot can have max 1 course)
	slotCourseCount := make(map[int]int)

	for _, assignment := range req.CourseAssignments {
		slotName := slotMap[assignment.SlotID]
		sem := assignment.Semester
		if strings.Contains(strings.ToLower(slotName), "honour slot") {
			honourCourseCountBySemester[sem]++
			honourCourseIDsBySemester[sem] = append(honourCourseIDsBySemester[sem], assignment.CourseID)
			
			// Track individual honour slot usage
			slotCourseCount[assignment.SlotID]++
			if slotCourseCount[assignment.SlotID] > 1 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": fmt.Sprintf("%s can have only 1 course", slotName),
				})
				return
			}
		} else if strings.Contains(strings.ToLower(slotName), "minor slot") {
			minorCourseCountBySemester[sem]++
			minorCourseIDsBySemester[sem] = append(minorCourseIDsBySemester[sem], assignment.CourseID)
			
			// Track individual minor slot usage
			slotCourseCount[assignment.SlotID]++
			if slotCourseCount[assignment.SlotID] > 1 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": fmt.Sprintf("%s can have only 1 course", slotName),
				})
				return
			}
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

	// Validate max 2 minor courses per semester
	for semester, count := range minorCourseCountBySemester {
		if count > 2 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Maximum 2 minor courses allowed per semester. Semester %d has %d minor courses.", semester, count),
			})
			return
		}
	}

	// Per-semester validations: no overlap between slot types within the same semester
	for sem := 4; sem <= 8; sem++ {
		honourIDs := honourCourseIDsBySemester[sem]
		minorIDs := minorCourseIDsBySemester[sem]
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

		// Minor vs Professional
		for _, mID := range minorIDs {
			for _, pID := range profIDs {
				if mID == pID {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"message": fmt.Sprintf("Semester %d: Minor and Professional Elective cannot share the same course. Please select different courses.", sem),
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

	// ==================== Vertical Lock Enforcement ====================
	// For honour/minor slot courses, enforce that all semesters for the same batch
	// use the same vertical. Auto-create lock on first assignment, enforce on subsequent.

	// Step 1: Build semester→batch map from academic_calendar
	semBatchMap := make(map[int]string)
	batchRows, err := db.DB.Query(`
		SELECT current_semester + 1 as next_sem, batch
		FROM academic_calendar
		WHERE is_current = 1 AND batch IS NOT NULL AND batch != ''
	`)
	if err != nil {
		log.Println("Error fetching semester-batch map:", err)
	} else {
		defer batchRows.Close()
		for batchRows.Next() {
			var nextSem int
			var batch string
			if err := batchRows.Scan(&nextSem, &batch); err == nil {
				if nextSem >= 4 && nextSem <= 8 {
					semBatchMap[nextSem] = batch
				}
			}
		}
	}

	// Step 2: For each honour/minor course, look up its vertical and check/create lock
	type lockKey struct {
		batch    string
		lockType string
	}
	// Collect the vertical we expect for each (batch, lock_type) in this save request
	newLockExpectations := make(map[lockKey]struct {
		verticalID   int
		verticalName string
		semester     int
	})

	for _, assignment := range req.CourseAssignments {
		slotName := slotMap[assignment.SlotID]
		isMinor := strings.Contains(strings.ToLower(slotName), "minor slot")

		// Only enforce vertical locks for minor slots (not honour)
		if !isMinor {
			continue
		}

		batch := semBatchMap[assignment.Semester]
		if batch == "" {
			continue // No batch configured for this semester — skip lock enforcement
		}

		lockType := "minor"

		verticalID, verticalName, err := lookupCourseVertical(assignment.CourseID, curriculumID)
		if err != nil {
			// Course might be from another dept (minor offering) — skip lock for those
			continue
		}

		key := lockKey{batch: batch, lockType: lockType}
		if existing, ok := newLockExpectations[key]; ok {
			// Already have an expectation for this batch+type — must be same vertical
			if existing.verticalID != verticalID {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": fmt.Sprintf(
						"All %s courses for batch %s must be from the same vertical. "+
							"Semester %d uses '%s' but semester %d uses '%s'.",
						lockType, batch, existing.semester, existing.verticalName,
						assignment.Semester, verticalName),
				})
				return
			}
		} else {
			newLockExpectations[key] = struct {
				verticalID   int
				verticalName string
				semester     int
			}{verticalID, verticalName, assignment.Semester}
		}

		// Check existing lock in database
		var existingLockVerticalID int
		var existingLockVerticalName string
		lockErr := db.DB.QueryRow(`
			SELECT vertical_id, vertical_name FROM honour_minor_vertical_locks
			WHERE department_id = ? AND batch = ? AND lock_type = ?
		`, departmentID, batch, lockType).Scan(&existingLockVerticalID, &existingLockVerticalName)

		if lockErr == nil {
			// Lock exists — enforce it
			if existingLockVerticalID != verticalID {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": fmt.Sprintf(
						"Batch %s already has %s vertical locked to '%s'. "+
							"Cannot assign course from '%s' vertical. "+
							"All %s courses across semesters must be from the same vertical.",
						batch, lockType, existingLockVerticalName, verticalName, lockType),
				})
				return
			}
		}
		// If no lock exists (sql.ErrNoRows), we'll create one after successful save
	}

	// Step 3: Create any new locks that don't exist yet (inside the transaction)
	for key, expectation := range newLockExpectations {
		var existingID int
		lockErr := tx.QueryRow(`
			SELECT id FROM honour_minor_vertical_locks
			WHERE department_id = ? AND batch = ? AND lock_type = ?
		`, departmentID, key.batch, key.lockType).Scan(&existingID)

		if lockErr != nil {
			// No existing lock — create one
			_, err := tx.Exec(`
				INSERT INTO honour_minor_vertical_locks
				(department_id, batch, lock_type, vertical_id, vertical_name, locked_by_user_id, first_semester, first_academic_year)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, departmentID, key.batch, key.lockType, expectation.verticalID, expectation.verticalName,
				userID, expectation.semester, req.AcademicYear)
			if err != nil {
				log.Println("Error creating vertical lock:", err)
				// Non-fatal — continue saving
			}
		}
	}
	// ==================== End Vertical Lock Enforcement ====================

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

	// Auto-add OE courses offered TO this department into the "Open Elective" slot for the semester
	autoAllocatedCount := 0
	{
		// Get unique semesters from course assignments
		semesterSet := make(map[int]bool)
		for _, assignment := range req.CourseAssignments {
			semesterSet[assignment.Semester] = true
		}

		incomingOEQuery := `
			SELECT oe.semester, oe.course_id
			FROM hod_oe_offerings oe
			WHERE oe.department_id != ?
			AND oe.status = 'ACTIVE'
			AND oe.academic_year = ?
			AND (oe.batch = ? OR (oe.batch IS NULL AND ? = ''))
			AND JSON_CONTAINS(oe.allowed_dept_ids, CAST(? AS JSON))
		`
		incomingRows, err := tx.Query(incomingOEQuery, departmentID, req.AcademicYear, req.Batch, req.Batch, departmentID)
		if err != nil {
			log.Println("Error fetching incoming OE offerings for auto-add:", err)
		} else {
			defer incomingRows.Close()

			// Group incoming courses by semester
			incomingBySem := make(map[int][]int)
			for incomingRows.Next() {
				var sem, courseID int
				if err := incomingRows.Scan(&sem, &courseID); err != nil {
					log.Println("Error scanning incoming OE:", err)
					continue
				}
				incomingBySem[sem] = append(incomingBySem[sem], courseID)
			}

			for semester := range semesterSet {
				coursesForSem := incomingBySem[semester]
				if len(coursesForSem) == 0 {
					continue
				}

				// Find the "Open Elective" slot for this semester
				oeSlotQuery := `
					SELECT id, slot_name
					FROM elective_semester_slots
					WHERE semester = ? AND LOWER(slot_name) = 'open elective' AND is_active = 1
					LIMIT 1
				`
				var oeSlotID int
				var oeSlotName string
				err := tx.QueryRow(oeSlotQuery, semester).Scan(&oeSlotID, &oeSlotName)
				if err != nil {
					// No Open Elective slot for this semester, skip
					continue
				}

				// Collect courses already in this OE slot
				existingCourses := make(map[int]bool)
				for _, assignment := range req.CourseAssignments {
					if assignment.Semester == semester && assignment.SlotID == oeSlotID {
						existingCourses[assignment.CourseID] = true
					}
				}
				// Also check what's already saved in DB for this slot
				checkQuery := `
					SELECT course_id FROM hod_elective_selections
					WHERE department_id = ? AND semester = ? AND slot_id = ? AND academic_year = ?
					AND (batch = ? OR (batch IS NULL AND ? = ''))
				`
				checkRows, err := tx.Query(checkQuery, departmentID, semester, oeSlotID, req.AcademicYear, req.Batch, req.Batch)
				if err == nil {
					for checkRows.Next() {
						var cid int
						if err := checkRows.Scan(&cid); err == nil {
							existingCourses[cid] = true
						}
					}
					checkRows.Close()
				}

				incomingInsertQuery := `
					INSERT INTO hod_elective_selections 
					(department_id, curriculum_id, semester, course_id, slot_id, slot_name, academic_year, batch, approved_by_user_id, status, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				`
				now := time.Now()
				autoStatus := req.Status
				if autoStatus == "" {
					autoStatus = "ACTIVE"
				}
				var batchVal interface{}
				if req.Batch != "" {
					batchVal = req.Batch
				} else {
					batchVal = nil
				}

				for _, oeCourseID := range coursesForSem {
					if existingCourses[oeCourseID] {
						continue
					}
					_, err = tx.Exec(incomingInsertQuery,
						departmentID, curriculumID, semester, oeCourseID,
						oeSlotID, oeSlotName, req.AcademicYear, batchVal,
						userID, autoStatus, now, now,
					)
					if err != nil {
						log.Println("Error auto-adding incoming OE to Open Elective slot:", err)
						continue
					}
					autoAllocatedCount++
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
			message = fmt.Sprintf("%s (%d open elective courses auto-added to Open Elective slot)", message, autoAllocatedCount)
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
		SELECT id, academic_year, year_level, current_semester, batch, semester_start_date, semester_end_date,
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
	semesterBatchMap := make(map[int]string) // maps next_semester → batch
	for rows.Next() {
		var calendar models.AcademicCalendar
		if err := rows.Scan(
			&calendar.ID,
			&calendar.AcademicYear,
			&calendar.YearLevel,
			&calendar.CurrentSemester,
			&calendar.Batch,
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
		// Map the NEXT semester (what HOD assigns for) to the batch
		nextSem := calendar.CurrentSemester + 1
		if nextSem >= 4 && nextSem <= 8 && calendar.Batch != nil {
			semesterBatchMap[nextSem] = *calendar.Batch
		}
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
		"academic_year":      calendars[0].AcademicYear,
		"current_semesters":  currentSemesters,
		"calendars":          calendars,
		"semester_batch_map": semesterBatchMap,
	})
}

// ==================== Honour Program Management ====================

// GetHonourVerticals retrieves verticals from honour_cards with HONOUR in title
func GetHonourVerticals(w http.ResponseWriter, r *http.Request) {
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

	var curriculumID int
	currQuery := `
		SELECT COALESCE(dca.curriculum_id, 0)
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		LEFT JOIN department_curriculum_active dca ON d.id = dca.department_id AND dca.is_active = 1
		WHERE u.email = ? AND u.role = 'hod'
		ORDER BY COALESCE(dca.curriculum_id, 0) DESC
		LIMIT 1
	`

	err := db.DB.QueryRow(currQuery, email).Scan(&curriculumID)
	if err != nil || curriculumID == 0 {
		log.Println("Error fetching curriculum for honour verticals:", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Curriculum not found",
		})
		return
	}

	query := `
		SELECT hv.id, hv.honour_card_id, 
		       CONCAT(hc.title, ' - ', hv.name) as name,
		       COUNT(DISTINCT hvc.course_id) as course_count
		FROM honour_verticals hv
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		LEFT JOIN honour_vertical_courses hvc ON hv.id = hvc.honour_vertical_id AND hvc.status = 1
		WHERE hc.curriculum_id = ? AND hc.status = 1 AND hv.status = 1
		  AND UPPER(hc.title) LIKE '%HONOUR%'
		GROUP BY hv.id, hv.honour_card_id, hc.title, hv.name
		HAVING COUNT(DISTINCT hvc.course_id) >= 1
		ORDER BY hc.title, hv.name
	`

	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Println("Error fetching honour verticals:", err)
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
			log.Println("Error scanning honour vertical:", err)
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
        ORDER BY COALESCE(dca.curriculum_id, 0) DESC
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

	// Get minor verticals from honour_verticals under honour_cards with MINOR in title
	query := `
		SELECT hv.id, hv.honour_card_id, 
		       CONCAT(hc.title, ' - ', hv.name) as name,
		       COUNT(DISTINCT hvc.course_id) as course_count
		FROM honour_verticals hv
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		LEFT JOIN honour_vertical_courses hvc ON hv.id = hvc.honour_vertical_id AND hvc.status = 1
		WHERE hc.curriculum_id = ? AND hc.status = 1 AND hv.status = 1
		  AND UPPER(hc.title) LIKE '%MINOR%'
		GROUP BY hv.id, hv.honour_card_id, hc.title, hv.name
		HAVING COUNT(DISTINCT hvc.course_id) >= 1
		ORDER BY hc.title, hv.name
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

	// Check source param to determine where to look for courses
	source := r.URL.Query().Get("source")

	var query string
	if source == "honour_vertical" {
		// Fetch from honour_vertical_courses
		query = `
			SELECT c.id, c.course_code, c.course_name, c.credit
			FROM courses c
			INNER JOIN honour_vertical_courses hvc ON c.id = hvc.course_id
			WHERE hvc.honour_vertical_id = ? AND c.status = 1 AND hvc.status = 1
			ORDER BY c.course_code
		`
	} else {
		// Default: fetch from curriculum_courses (normal_cards)
		query = `
			SELECT c.id, c.course_code, c.course_name, c.credit
			FROM courses c
			INNER JOIN curriculum_courses cc ON c.id = cc.course_id
			WHERE cc.semester_id = ? AND c.status = 1
			ORDER BY c.course_code
		`
	}

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

	// No special validation needed for minor slots - they work like any other slots
	// Minor Slot 1 and Minor Slot 2 are regular slots in elective_semester_slots

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
		Message: fmt.Sprintf("Minor program saved successfully: %d courses offered", len(req.SemesterAssignments)),
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
		ORDER BY COALESCE(dca.curriculum_id, 0) DESC
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
	// Filter by type: "same" (default for offering workflow), "different", or "all"
	oeType := r.URL.Query().Get("type")
	var verticalFilter string
	switch oeType {
	case "different":
		verticalFilter = "AND nc.vertical_name = 'Different Dept'"
	case "all":
		verticalFilter = ""
	default:
		// Default: only Same Dept cards (for offering to other depts)
		verticalFilter = "AND nc.vertical_name = 'Same Dept'"
	}

	query := fmt.Sprintf(`
		SELECT nc.id,
		       COALESCE(nc.vertical_name, CONCAT('Open Elective - Semester ', nc.semester_number), 'Open Elective') as name,
		       nc.semester_number,
		       COALESCE(nc.vertical_name, '') as vertical_name,
		       COUNT(DISTINCT cc.course_id) as course_count
		FROM normal_cards nc
		LEFT JOIN curriculum_courses cc ON nc.id = cc.semester_id
		WHERE nc.curriculum_id = ? AND nc.card_type = 'open_elective' AND nc.status = 1
		%s
		GROUP BY nc.id, nc.semester_number, nc.vertical_name
		HAVING COUNT(DISTINCT cc.course_id) >= 1
		ORDER BY nc.semester_number IS NULL, nc.semester_number, nc.vertical_name
	`, verticalFilter)

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
		var verticalName sql.NullString
		err := rows.Scan(&card.ID, &cardName, &semesterNumber, &verticalName, &card.CourseCount)
		if err != nil {
			log.Println("Error scanning OE card:", err)
			continue
		}
		if cardName.Valid {
			card.Name = cardName.String
		} else {
			card.Name = "Open Elective"
		}
		if verticalName.Valid {
			card.VerticalName = verticalName.String
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

	// Validate that the selected OE card is a "Same Department" card
	var cardVerticalName string
	cardCheckQuery := `
		SELECT COALESCE(vertical_name, '') FROM normal_cards 
		WHERE id = ? AND curriculum_id = ? AND card_type = 'open_elective' AND status = 1
	`
	err = db.DB.QueryRow(cardCheckQuery, req.OECardID, curriculumID).Scan(&cardVerticalName)
	if err != nil {
		log.Println("Error validating OE card:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid OE card selected",
		})
		return
	}
	if cardVerticalName != "Same Dept" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Only 'Same Dept' OE cards can be offered to other departments",
		})
		return
	}

	// Validate that each offered course belongs to the selected Same Dept OE card.
	for _, assignment := range req.SemesterAssignments {
		var courseInCardCount int
		err := db.DB.QueryRow(`
			SELECT COUNT(*)
			FROM curriculum_courses cc
			INNER JOIN normal_cards nc ON cc.semester_id = nc.id
			WHERE cc.semester_id = ?
			  AND cc.course_id = ?
			  AND nc.curriculum_id = ?
			  AND nc.card_type = 'open_elective'
			  AND COALESCE(nc.vertical_name, '') = 'Same Dept'
			  AND nc.status = 1
		`, req.OECardID, assignment.CourseID, curriculumID).Scan(&courseInCardCount)
		if err != nil || courseInCardCount == 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "One or more selected courses are not part of the chosen Same Dept OE card",
			})
			return
		}
	}

	// Cross-department validation: verify each target department has an active curriculum
	// AND that each offered course exists in their "Different Dept" OE card
	type skippedDeptInfo struct {
		DeptName       string   `json:"dept_name"`
		Reason         string   `json:"reason"`
		MissingCourses []string `json:"missing_courses,omitempty"`
	}
	var skippedDepts []skippedDeptInfo
	var validDeptIDs []int

	for _, targetDeptID := range req.AllowedDeptIDs {
		var deptName string
		db.DB.QueryRow("SELECT COALESCE(department_name, CONCAT('ID:', id)) FROM departments WHERE id = ?", targetDeptID).Scan(&deptName)

		var targetCurrID int
		err := db.DB.QueryRow(`
			SELECT COALESCE(curriculum_id, 0) FROM department_curriculum_active 
			WHERE department_id = ? AND is_active = 1 LIMIT 1
		`, targetDeptID).Scan(&targetCurrID)
		if err != nil || targetCurrID == 0 {
			skippedDepts = append(skippedDepts, skippedDeptInfo{
				DeptName: deptName,
				Reason:   "does not have an active curriculum",
			})
			continue
		}

		// Check each offered course exists in the target dept's "Different Dept" OE card (match by course_code)
		var missingCourses []string
		for _, assignment := range req.SemesterAssignments {
			var count int
			err := db.DB.QueryRow(`
				SELECT COUNT(*) FROM curriculum_courses cc
				INNER JOIN normal_cards nc ON cc.semester_id = nc.id
				INNER JOIN courses c1 ON cc.course_id = c1.id
				INNER JOIN courses c2 ON TRIM(c1.course_code) = TRIM(c2.course_code)
				WHERE c2.id = ?
				  AND nc.curriculum_id = ?
				  AND nc.card_type = 'open_elective'
				  AND COALESCE(nc.vertical_name, '') = 'Different Dept'
				  AND nc.status = 1
			`, assignment.CourseID, targetCurrID).Scan(&count)
			if err != nil || count == 0 {
				var courseName string
				db.DB.QueryRow("SELECT COALESCE(CONCAT(course_code, ' - ', course_name), CONCAT('ID:', id)) FROM courses WHERE id = ?", assignment.CourseID).Scan(&courseName)
				missingCourses = append(missingCourses, courseName)
			}
		}

		if len(missingCourses) > 0 {
			skippedDepts = append(skippedDepts, skippedDeptInfo{
				DeptName:       deptName,
				Reason:         "courses not in their Different Dept OE card",
				MissingCourses: missingCourses,
			})
			continue
		}

		validDeptIDs = append(validDeptIDs, targetDeptID)
	}

	if len(validDeptIDs) == 0 {
		// All departments failed validation
		skippedMessages := make([]string, len(skippedDepts))
		for i, sd := range skippedDepts {
			if len(sd.MissingCourses) > 0 {
				skippedMessages[i] = fmt.Sprintf("%s: %s (%s)", sd.DeptName, sd.Reason, strings.Join(sd.MissingCourses, ", "))
			} else {
				skippedMessages[i] = fmt.Sprintf("%s: %s", sd.DeptName, sd.Reason)
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":             false,
			"message":             "No valid departments to offer courses to",
			"skipped_departments": skippedDepts,
			"details":             strings.Join(skippedMessages, "; "),
		})
		return
	}

	// Use only validated department IDs for the offering
	req.AllowedDeptIDs = validDeptIDs

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

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Open Elective offerings saved successfully: %d courses across semesters", len(req.SemesterAssignments)),
	}
	if len(skippedDepts) > 0 {
		skippedNames := make([]string, len(skippedDepts))
		for i, sd := range skippedDepts {
			skippedNames[i] = sd.DeptName
		}
		response["message"] = fmt.Sprintf("Open Elective offerings saved for %d department(s). Skipped %d department(s): %s",
			len(validDeptIDs), len(skippedDepts), strings.Join(skippedNames, ", "))
		response["skipped_departments"] = skippedDepts
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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

// GetOEOfferedToMyDept returns OE courses that other departments are offering TO the current HOD's department
func GetOEOfferedToMyDept(w http.ResponseWriter, r *http.Request) {
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

	// Find all OE offerings from other departments where my department is in the allowed_dept_ids
	// MySQL JSON_CONTAINS can check if departmentID is in the JSON array
	query := `
		SELECT oe.department_id, dep.department_name as offering_dept_name,
		       oe.semester, oe.course_id,
		       c.course_code, c.course_name, c.credit,
		       oe.academic_year
		FROM hod_oe_offerings oe
		INNER JOIN departments dep ON oe.department_id = dep.id
		INNER JOIN courses c ON oe.course_id = c.id
		WHERE oe.department_id != ?
		AND oe.status = 'ACTIVE'
		AND oe.academic_year = ?
		AND (oe.batch = ? OR (oe.batch IS NULL AND ? = ''))
		AND JSON_CONTAINS(oe.allowed_dept_ids, CAST(? AS JSON))
		ORDER BY dep.department_name, oe.semester, c.course_code
	`

	rows, err := db.DB.Query(query, departmentID, academicYear, batch, batch, departmentID)
	if err != nil {
		log.Println("Error fetching OE offerings to dept:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	type IncomingOECourse struct {
		OfferingDeptID   int    `json:"offering_dept_id"`
		OfferingDeptName string `json:"offering_dept_name"`
		Semester         int    `json:"semester"`
		CourseID         int    `json:"course_id"`
		CourseCode       string `json:"course_code"`
		CourseName       string `json:"course_name"`
		Credit           int    `json:"credit"`
		AcademicYear     string `json:"academic_year"`
	}

	offerings := []IncomingOECourse{}
	for rows.Next() {
		var o IncomingOECourse
		err := rows.Scan(&o.OfferingDeptID, &o.OfferingDeptName, &o.Semester,
			&o.CourseID, &o.CourseCode, &o.CourseName, &o.Credit, &o.AcademicYear)
		if err != nil {
			log.Println("Error scanning incoming OE offering:", err)
			continue
		}
		offerings = append(offerings, o)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"offerings": offerings,
	})
}

// ==================== Vertical Lock Handlers ====================

// GetVerticalLocks returns all honour/minor vertical locks for the HOD's department
func GetVerticalLocks(w http.ResponseWriter, r *http.Request) {
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

	// Fetch all vertical locks for this department
	query := `
		SELECT id, department_id, batch, lock_type, vertical_id, vertical_name,
		       locked_by_user_id, first_semester, first_academic_year
		FROM honour_minor_vertical_locks
		WHERE department_id = ?
		ORDER BY batch, lock_type
	`

	rows, err := db.DB.Query(query, departmentID)
	if err != nil {
		log.Println("Error fetching vertical locks:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Internal server error",
		})
		return
	}
	defer rows.Close()

	locks := []models.VerticalLock{}
	for rows.Next() {
		var lock models.VerticalLock
		if err := rows.Scan(
			&lock.ID,
			&lock.DepartmentID,
			&lock.Batch,
			&lock.LockType,
			&lock.VerticalID,
			&lock.VerticalName,
			&lock.LockedByUserID,
			&lock.FirstSemester,
			&lock.FirstAcademicYear,
		); err != nil {
			log.Println("Error scanning vertical lock:", err)
			continue
		}
		locks = append(locks, lock)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"locks":   locks,
	})
}

// lookupCourseVertical finds the vertical_id and vertical_name for a given course_id.
// First checks curriculum_courses → normal_cards (for regular verticals),
// then falls back to honour_vertical_courses → honour_cards (for minor/honour verticals).
func lookupCourseVertical(courseID int, curriculumID int) (int, string, error) {
	var verticalID int
	var verticalName string

	// Try normal_cards first (regular verticals)
	query := `
		SELECT nc.id, nc.vertical_name
		FROM curriculum_courses cc
		INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		WHERE cc.course_id = ? AND cc.curriculum_id = ?
		AND nc.card_type IN ('vertical', 'open_elective')
		AND nc.status = 1
		LIMIT 1
	`
	err := db.DB.QueryRow(query, courseID, curriculumID).Scan(&verticalID, &verticalName)
	if err == nil {
		return verticalID, verticalName, nil
	}

	// Fallback: check honour_vertical_courses → honour_verticals → honour_cards (for minor/honour courses)
	honourQuery := `
		SELECT hv.id, CONCAT(hc.title, ' - ', hv.name)
		FROM honour_vertical_courses hvc
		INNER JOIN honour_verticals hv ON hvc.honour_vertical_id = hv.id
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		WHERE hvc.course_id = ? AND hc.curriculum_id = ?
		AND COALESCE(hvc.status, 1) = 1
		AND COALESCE(hv.status, 1) = 1
		AND COALESCE(hc.status, 1) = 1
		LIMIT 1
	`
	err = db.DB.QueryRow(honourQuery, courseID, curriculumID).Scan(&verticalID, &verticalName)
	return verticalID, verticalName, err
}

// ValidateOECourseEligibility checks which courses are eligible for each target department's "Different Dept" OE card
func ValidateOECourseEligibility(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Method not allowed"})
		return
	}

	courseIDsStr := r.URL.Query().Get("course_ids")
	deptIDsStr := r.URL.Query().Get("dept_ids")
	if courseIDsStr == "" || deptIDsStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "course_ids and dept_ids parameters required"})
		return
	}

	// Parse course IDs
	var courseIDs []int
	for _, s := range strings.Split(courseIDsStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		courseIDs = append(courseIDs, id)
	}

	// Parse department IDs
	var deptIDs []int
	for _, s := range strings.Split(deptIDsStr, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		deptIDs = append(deptIDs, id)
	}

	if len(courseIDs) == 0 || len(deptIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Valid course_ids and dept_ids required"})
		return
	}

	type deptResult struct {
		DeptID          int      `json:"dept_id"`
		DeptName        string   `json:"dept_name"`
		HasCurriculum   bool     `json:"has_curriculum"`
		EligibleCourses []string `json:"eligible_courses"`
		MissingCourses  []string `json:"missing_courses"`
		Eligible        bool     `json:"eligible"`
	}

	var results []deptResult

	for _, deptID := range deptIDs {
		var deptName string
		db.DB.QueryRow("SELECT COALESCE(department_name, CONCAT('ID:', id)) FROM departments WHERE id = ?", deptID).Scan(&deptName)

		var targetCurrID int
		err := db.DB.QueryRow(`
			SELECT COALESCE(curriculum_id, 0) FROM department_curriculum_active 
			WHERE department_id = ? AND is_active = 1 LIMIT 1
		`, deptID).Scan(&targetCurrID)

		if err != nil || targetCurrID == 0 {
			results = append(results, deptResult{
				DeptID:        deptID,
				DeptName:      deptName,
				HasCurriculum: false,
				Eligible:      false,
			})
			continue
		}

		var eligibleCourses, missingCourses []string
		for _, courseID := range courseIDs {
			var count int
			db.DB.QueryRow(`
				SELECT COUNT(*) FROM curriculum_courses cc
				INNER JOIN normal_cards nc ON cc.semester_id = nc.id
				INNER JOIN courses c1 ON cc.course_id = c1.id
				INNER JOIN courses c2 ON TRIM(c1.course_code) = TRIM(c2.course_code)
				WHERE c2.id = ?
				  AND nc.curriculum_id = ?
				  AND nc.card_type = 'open_elective'
				  AND COALESCE(nc.vertical_name, '') = 'Different Dept'
				  AND nc.status = 1
			`, courseID, targetCurrID).Scan(&count)

			var courseName string
			db.DB.QueryRow("SELECT COALESCE(CONCAT(course_code, ' - ', course_name), CONCAT('ID:', id)) FROM courses WHERE id = ?", courseID).Scan(&courseName)

			if count > 0 {
				eligibleCourses = append(eligibleCourses, courseName)
			} else {
				missingCourses = append(missingCourses, courseName)
			}
		}

		results = append(results, deptResult{
			DeptID:          deptID,
			DeptName:        deptName,
			HasCurriculum:   true,
			EligibleCourses: eligibleCourses,
			MissingCourses:  missingCourses,
			Eligible:        len(missingCourses) == 0,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"departments": results,
	})
}
