package studentteacher

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"strconv"
	"strings"
	"time"
)

// ElectiveOption represents an available elective course
type ElectiveOption struct {
	HodSelectionID int    `json:"hod_selection_id"` // hod_elective_selections.id — use this as the key when submitting
	CourseID        int    `json:"course_id"`
	CourseCode      string `json:"course_code"`
	CourseName      string `json:"course_name"`
	Credits         int    `json:"credits"`
	Category        string `json:"category"`
	SlotID          int    `json:"slot_id"`
	SlotName        string `json:"slot_name"`
}

// ElectiveSlot represents a group of electives for a specific slot
type ElectiveSlot struct {
	SlotID      int               `json:"slot_id"`
	SlotName    string            `json:"slot_name"`
	SlotType    string            `json:"slot_type"` // "PROFESSIONAL", "OPEN", or "MIXED"
	Courses     []ElectiveOption  `json:"courses"`
	IsActive    bool              `json:"is_active"`
}

// ElectivesBySlot represents electives organized by slots
type ElectivesBySlot struct {
	StudentID          int             `json:"student_id"`
	DepartmentID       int             `json:"department_id"`
	CurrentSemester    int             `json:"current_semester"`
	NextSemester       int             `json:"next_semester"`
	Batch              string          `json:"batch"`
	Slots              []ElectiveSlot  `json:"slots"`
	ExistingSelections map[string]int  `json:"existing_selections"` // slot_name -> hod_selection_id
	WindowOpen         bool            `json:"window_open"`
	WindowStart        string          `json:"window_start"` // YYYY-MM-DD
	WindowEnd          string          `json:"window_end"`   // YYYY-MM-DD
}

// ElectiveSelection represents a student's elective choice
type ElectiveSelection struct {
	StudentID  int    `json:"student_id"`
	SemNo      int    `json:"sem_no"`
	CardType   string `json:"card_type"`
	CourseID   int    `json:"course_id"`
	CardID     int    `json:"card_id"`
}

func isOpenElectiveCategory(category, slotName string) bool {
	cat := strings.ToLower(strings.TrimSpace(category))
	slot := strings.ToLower(strings.TrimSpace(slotName))
	return strings.Contains(cat, "open elective") || strings.Contains(cat, "oe - open elective") || strings.Contains(slot, "open elective")
}

func shiftAcademicYearForOddSemester(academicYear string, semester int) string {
	trimmedAcademicYear := strings.TrimSpace(academicYear)
	if semester%2 == 0 {
		return trimmedAcademicYear
	}

	parts := strings.Split(trimmedAcademicYear, "-")
	if len(parts) != 2 {
		return trimmedAcademicYear
	}

	startYear, errStart := strconv.Atoi(strings.TrimSpace(parts[0]))
	endYear, errEnd := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errStart != nil || errEnd != nil {
		return trimmedAcademicYear
	}

	return fmt.Sprintf("%d-%d", startYear+1, endYear+1)
}

func getEligibleOpenElectiveCourseSet(curriculumID int) (map[int]struct{}, error) {
	eligible := make(map[int]struct{})
	if curriculumID <= 0 {
		return eligible, nil
	}

	query := `
		SELECT DISTINCT c.id
		FROM courses c
		INNER JOIN curriculum_courses cc ON c.id = cc.course_id
		INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		WHERE cc.curriculum_id = ?
		  AND LOWER(nc.card_type) = 'open_elective'
		  AND c.status = 1
		  AND nc.status = 1
	`

	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			continue
		}
		eligible[courseID] = struct{}{}
	}

	return eligible, nil
}

func resolveStudentCurriculumID(studentID, departmentID int) (int, error) {
	var curriculum sql.NullInt64
	err := db.DB.QueryRow(`
		SELECT curriculum_id
		FROM academic_details
		WHERE student_id = ?
	`, studentID).Scan(&curriculum)
	if err != nil {
		return 0, err
	}

	if curriculum.Valid && curriculum.Int64 > 0 {
		return int(curriculum.Int64), nil
	}

	var departmentCurriculum sql.NullInt64
	err = db.DB.QueryRow(`
		SELECT current_curriculum_id
		FROM departments
		WHERE id = ?
	`, departmentID).Scan(&departmentCurriculum)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	if departmentCurriculum.Valid && departmentCurriculum.Int64 > 0 {
		return int(departmentCurriculum.Int64), nil
	}

	return 0, nil
}

// GetAvailableElectives returns electives available for a student based on HOD selections
func GetAvailableElectives(w http.ResponseWriter, r *http.Request) {
	// Get email from query parameter (from logged in user)
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Fetching electives for email: %s", email)

	// Step 1: Get student_id from contact_details using student_email (email from users table)
	var studentID int
	err := db.DB.QueryRow(`
		SELECT student_id 
		FROM contact_details 
		WHERE student_email = ?
	`, email).Scan(&studentID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Student not found with this email", http.StatusNotFound)
		} else {
			log.Printf("Error fetching student from contact_details: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Found student_id: %d", studentID)

	// Step 2: Get student's department_id from students table
	var departmentID int
	err = db.DB.QueryRow(`
		SELECT s.department_id 
		FROM students s
		WHERE s.id = ?
	`, studentID).Scan(&departmentID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Student not found in students table with id: %d", studentID)
			http.Error(w, fmt.Sprintf("Student not found in students table with id: %d", studentID), http.StatusNotFound)
		} else {
			log.Printf("Error fetching student department: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Found department_id: %d", departmentID)

	// Step 3: Get semester and batch from academic_details (curriculum can be NULL)
	var currentSemester int
	var batch string
	err = db.DB.QueryRow(`
		SELECT semester, batch
		FROM academic_details 
		WHERE student_id = ?
	`, studentID).Scan(&currentSemester, &batch)
	
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Academic details not found for student_id: %d", studentID)
			http.Error(w, fmt.Sprintf("Academic details not found for student_id: %d", studentID), http.StatusNotFound)
		} else {
			log.Printf("Error fetching current semester and batch: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	studentCurriculumID, err := resolveStudentCurriculumID(studentID, departmentID)
	if err != nil {
		log.Printf("Error resolving curriculum_id for student %d: %v", studentID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	nextSemester := currentSemester + 1
	log.Printf("Student department_id: %d, curriculum_id: %d, current semester: %d, next semester: %d, batch: %s", departmentID, studentCurriculumID, currentSemester, nextSemester, batch)

	eligibleOpenElectiveCourseIDs, err := getEligibleOpenElectiveCourseSet(studentCurriculumID)
	if err != nil {
		log.Printf("Error fetching eligible open electives for curriculum_id %d: %v", studentCurriculumID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	log.Printf("Eligible open-elective course count for curriculum %d: %d", studentCurriculumID, len(eligibleOpenElectiveCourseIDs))

	// Step 3b: Check elective selection window from academic_calendar
	var windowStart, windowEnd string
	windowOpen := false
	err = db.DB.QueryRow(`
		SELECT
			COALESCE(DATE_FORMAT(elective_selection_start,'%Y-%m-%d'), '') as ws,
			COALESCE(DATE_FORMAT(elective_selection_end,  '%Y-%m-%d'), '') as we
		FROM academic_calendar
		WHERE current_semester = ? AND is_current = 1
		LIMIT 1
	`, currentSemester).Scan(&windowStart, &windowEnd)
	if err != nil {
		log.Printf("Warning: could not fetch elective window from academic_calendar: %v", err)
	}
	if windowStart != "" && windowEnd != "" {
		const dateFmt = "2006-01-02"
		today := time.Now().Format(dateFmt)
		windowOpen = today >= windowStart && today <= windowEnd
		log.Printf("Elective window: %s to %s, today: %s, open: %v", windowStart, windowEnd, today, windowOpen)
	}

	// Step 4: Check student eligibility for Honour/Minor courses by type
	eligibilityMap := make(map[string]bool)
	
	// Check for HONOUR eligibility
	var hasHonourEligibility bool
	err = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM student_eligible_honour_minor 
			WHERE student_email = ? AND (type = 'HONOUR')
		)
	`, email).Scan(&hasHonourEligibility)
	
	if err != nil {
		log.Printf("Error checking honour eligibility: %v", err)
		hasHonourEligibility = false
	}
	eligibilityMap["HONOUR"] = hasHonourEligibility
	
	// Check for MINOR eligibility
	var hasMinorEligibility bool
	err = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM student_eligible_honour_minor 
			WHERE student_email = ? AND (type = 'MINOR')
		)
	`, email).Scan(&hasMinorEligibility)
	
	if err != nil {
		log.Printf("Error checking minor eligibility: %v", err)
		hasMinorEligibility = false
	}
	eligibilityMap["MINOR"] = hasMinorEligibility
	
	log.Printf("Student eligible for Honour: %v, Minor: %v", hasHonourEligibility, hasMinorEligibility)

	// Step 5: Get ALL elective slots for this semester with their HOD-assigned courses
	query := `
		SELECT 
			COALESCE(c.id, 0) as course_id,
			COALESCE(c.course_code, '') as course_code,
			COALESCE(c.course_name, '') as course_name,
			COALESCE(c.credit, 0) as credits,
			COALESCE(c.category, '') as category,
			ess.id as slot_id,
			ess.slot_name,
			ess.is_active,
			ess.slot_order,
			COALESCE(hes.id, 0) as hod_selection_id,
			COALESCE(hes.department_id, 0) as hod_department_id
		FROM elective_semester_slots ess
		LEFT JOIN hod_elective_selections hes ON ess.id = hes.slot_id 
			AND hes.semester = ?
			AND (hes.batch = ? OR hes.batch IS NULL)
			AND hes.status = 'ACTIVE'
		LEFT JOIN courses c ON hes.course_id = c.id
		WHERE ess.semester = ?
		AND ess.is_active = 1
		ORDER BY ess.slot_order, ess.slot_name, c.category, c.course_code
	`

	log.Printf("Executing electives query with params: semester=%d, batch=%s, studentID=%d",
		nextSemester, batch, studentID)

	rows, err := db.DB.Query(query, nextSemester, batch, nextSemester)
	if err != nil {
		log.Printf("Error fetching available electives: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Map to group courses by slot
	slotMap := make(map[int]*ElectiveSlot)
	var slotOrderList []int // To maintain order
	courseCount := 0
	rowCount := 0

	for rows.Next() {
		rowCount++
		var elective ElectiveOption
		var hodDepartmentID int
		var isActive bool
		var slotOrder int
		err := rows.Scan(
			&elective.CourseID,
			&elective.CourseCode,
			&elective.CourseName,
			&elective.Credits,
			&elective.Category,
			&elective.SlotID,
			&elective.SlotName,
			&isActive,
			&slotOrder,
			&elective.HodSelectionID,
			&hodDepartmentID,
		)
		if err != nil {
			log.Printf("Error scanning elective: %v", err)
			continue
		}

		if elective.CourseID > 0 {
			if isOpenElectiveCategory(elective.Category, elective.SlotName) {
				if _, ok := eligibleOpenElectiveCourseIDs[elective.CourseID]; !ok {
					log.Printf("Skipping OE course %d (%s) for student %d: not in student curriculum %d", elective.CourseID, elective.CourseCode, studentID, studentCurriculumID)
					continue
				}
			} else {
				if hodDepartmentID != departmentID {
					log.Printf("Skipping non-OE course %d (%s) for student %d: HOD dept %d != student dept %d", elective.CourseID, elective.CourseCode, studentID, hodDepartmentID, departmentID)
					continue
				}
			}
		}

		// Log each row for debugging
		log.Printf("Row %d: slot_id=%d (%s), course_id=%d (%s), hod_selection_id=%d",
			rowCount, elective.SlotID, elective.SlotName, elective.CourseID, elective.CourseCode, elective.HodSelectionID)

		// Create slot if it doesn't exist
		if _, exists := slotMap[elective.SlotID]; !exists {
			slotType := determineSlotType(elective.SlotName, elective.Category)
			slotMap[elective.SlotID] = &ElectiveSlot{
				SlotID:   elective.SlotID,
				SlotName: elective.SlotName,
				SlotType: slotType,
				Courses:  []ElectiveOption{},
				IsActive: isActive,
			}
			slotOrderList = append(slotOrderList, elective.SlotID)
			log.Printf("  -> Created slot: %s (slot_id: %d, slot_type: %s)", elective.SlotName, elective.SlotID, slotType)
		}

		// Only add course if it exists (course_id > 0)
		if elective.CourseID > 0 {
			courseCount++
			log.Printf("  -> Adding course #%d: %s - %s to slot %d", 
				courseCount, elective.CourseCode, elective.CourseName, elective.SlotID)
			// Add course to the slot
			slotMap[elective.SlotID].Courses = append(slotMap[elective.SlotID].Courses, elective)
		} else {
			log.Printf("  -> Empty slot (no courses assigned)")
		}
	}

	// Convert map to ordered slice
	var slots []ElectiveSlot
	for _, slotID := range slotOrderList {
		slot := slotMap[slotID]

		// Post-process: if a slot is named like a professional slot but ALL its courses
		// belong to a different category (e.g. HOD put Add-On courses in "Professional Elective 9"),
		// reclassify it based on the dominant course category.
		if slot.SlotType == "PROFESSIONAL" && len(slot.Courses) > 0 {
			allAddon := true
			allOpen  := true
			for _, c := range slot.Courses {
				cat := strings.ToLower(c.Category)
				if !strings.Contains(cat, "addon") && !strings.Contains(cat, "add-on") && !strings.Contains(cat, "add on") {
					allAddon = false
				}
				if !strings.Contains(cat, "open") {
					allOpen = false
				}
			}
			if allAddon {
				log.Printf("Reclassifying slot '%s' from PROFESSIONAL to ADDON (all courses are Add-On category)", slot.SlotName)
				slot.SlotType = "ADDON"
			} else if allOpen {
				log.Printf("Reclassifying slot '%s' from PROFESSIONAL to OPEN (all courses are Open category)", slot.SlotName)
				slot.SlotType = "OPEN"
			}
		}

		// Filter out slots based on eligibility
		if slot.SlotType == "HONOR" && !eligibilityMap["HONOUR"] {
			log.Printf("Filtering out HONOUR slot '%s' - student not eligible", slot.SlotName)
			continue
		}
		if slot.SlotType == "MINOR" && !eligibilityMap["MINOR"] {
			log.Printf("Filtering out MINOR slot '%s' - student not eligible", slot.SlotName)
			continue
		}
		
		slots = append(slots, *slot)
	}

	// Handle mixed slots (last professional elective can include open electives)
	slots = handleMixedSlots(slots, nextSemester)

	// Fetch existing selections for this student and semester
	existingSelections := make(map[string]int)
	selectionRows, err := db.DB.Query(`
		SELECT ess.slot_name, sec.hod_selection_id
		FROM student_elective_choices sec
		INNER JOIN hod_elective_selections hes ON sec.hod_selection_id = hes.id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE sec.student_id = ? AND sec.semester = ?
	`, studentID, nextSemester)
	
	if err != nil {
		log.Printf("Warning: Could not fetch existing selections: %v", err)
	} else {
		defer selectionRows.Close()
		for selectionRows.Next() {
			var slotName string
			var hodSelID int
			if err := selectionRows.Scan(&slotName, &hodSelID); err != nil {
				log.Printf("Error scanning selection: %v", err)
				continue
			}
			existingSelections[slotName] = hodSelID
		}
		log.Printf("Found %d existing selections for student", len(existingSelections))
	}

	log.Printf("Processed %d rows from query, found %d courses in %d elective slots for next semester %d, batch %s", rowCount, courseCount, len(slots), nextSemester, batch)
	for i, slot := range slots {
		log.Printf("Slot %d: %s (%s) with %d courses", i+1, slot.SlotName, slot.SlotType, len(slot.Courses))
	}

	response := ElectivesBySlot{
		StudentID:          studentID,
		DepartmentID:       departmentID,
		CurrentSemester:    currentSemester,
		NextSemester:       nextSemester,
		Batch:              batch,
		Slots:              slots,
		ExistingSelections: existingSelections,
		WindowOpen:         windowOpen,
		WindowStart:        windowStart,
		WindowEnd:          windowEnd,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully sent response with %d slots", len(slots))
}

// SaveElectiveSelections saves a student's elective choices
func SaveElectiveSelections(w http.ResponseWriter, r *http.Request) {
	// Get email from query parameter
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Saving elective selections for email: %s", email)

	// Get student_id
	var studentID int
	err := db.DB.QueryRow(`SELECT student_id FROM contact_details WHERE student_email = ?`, email).Scan(&studentID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Student not found with this email", http.StatusNotFound)
		} else {
			log.Printf("Error fetching student: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Get current semester, department and batch (curriculum can be NULL)
	var currentSemester, departmentID int
	var batch string
	err = db.DB.QueryRow(`
		SELECT ad.semester, s.department_id, ad.batch
		FROM students s
		INNER JOIN academic_details ad ON s.id = ad.student_id
		WHERE s.id = ?
	`, studentID).Scan(&currentSemester, &departmentID, &batch)
	if err != nil {
		log.Printf("Error fetching student details: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	studentCurriculumID, err := resolveStudentCurriculumID(studentID, departmentID)
	if err != nil {
		log.Printf("Error resolving curriculum_id for student %d: %v", studentID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Get academic year + check elective selection window
	var academicYear, winStart, winEnd string
	_ = db.DB.QueryRow(`
		SELECT
			COALESCE(academic_year, ''),
			COALESCE(DATE_FORMAT(elective_selection_start,'%Y-%m-%d'), ''),
			COALESCE(DATE_FORMAT(elective_selection_end,  '%Y-%m-%d'), '')
		FROM academic_calendar
		WHERE current_semester = ? AND is_current = 1
		LIMIT 1
	`, currentSemester).Scan(&academicYear, &winStart, &winEnd)

	if winStart != "" && winEnd != "" {
		today := time.Now().Format("2006-01-02")
		if today < winStart || today > winEnd {
			log.Printf("Elective selection window closed for student %d (window: %s to %s, today: %s)", studentID, winStart, winEnd, today)
			http.Error(w, fmt.Sprintf("Elective selection window is closed (open %s to %s)", winStart, winEnd), http.StatusForbidden)
			return
		}
	}

	var requestBody struct {
		SelectionIDs []int                  `json:"selection_ids"` // hod_selection_id list from frontend
		Selections   map[string]interface{} `json:"selections"`    // backward compatibility
		Semester     int                    `json:"semester"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if requestBody.Semester == 0 {
		requestBody.Semester = currentSemester + 1
	}

	selectionAcademicYear := shiftAcademicYearForOddSemester(academicYear, requestBody.Semester)
	log.Printf("Saving student elective choices for student_id=%d semester=%d academic_year(base=%s, stored=%s)", studentID, requestBody.Semester, academicYear, selectionAcademicYear)

	selectionIDs := requestBody.SelectionIDs
	if len(selectionIDs) == 0 && len(requestBody.Selections) > 0 {
		for _, val := range requestBody.Selections {
			switch v := val.(type) {
			case float64:
				id := int(v)
				if id > 0 {
					selectionIDs = append(selectionIDs, id)
				}
			case string:
				parsed, parseErr := strconv.Atoi(v)
				if parseErr == nil && parsed > 0 {
					selectionIDs = append(selectionIDs, parsed)
				}
			}
		}
	}

	if len(selectionIDs) == 0 {
		http.Error(w, "No selections provided", http.StatusBadRequest)
		return
	}

	eligibleOpenElectiveCourseIDs, err := getEligibleOpenElectiveCourseSet(studentCurriculumID)
	if err != nil {
		log.Printf("Error fetching eligible open electives for curriculum_id %d: %v", studentCurriculumID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete existing PENDING selections for this student+semester (allow re-save during window)
	_, err = tx.Exec(`DELETE FROM student_elective_choices WHERE student_id = ? AND semester = ?`, studentID, requestBody.Semester)
	if err != nil {
		log.Printf("Error deleting existing selections: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Prepare: validate hod_selection_id belongs to this student's semester/batch and fetch metadata
	// and fetch the course_id so we can also record it in student_courses
	validateStmt, err := tx.Prepare(`
		SELECT hes.course_id, hes.department_id, COALESCE(c.category, ''), COALESCE(ess.slot_name, '')
		FROM hod_elective_selections hes
		INNER JOIN courses c ON c.id = hes.course_id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE hes.id = ?
		  AND hes.semester = ?
		  AND (hes.batch = ? OR hes.batch IS NULL)
		  AND hes.status = 'ACTIVE'
		  AND ess.is_active = 1
		LIMIT 1
	`)
	if err != nil {
		log.Printf("Error preparing validate statement: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer validateStmt.Close()

	insertStmt, err := tx.Prepare(`
		INSERT INTO student_elective_choices (student_id, hod_selection_id, semester, academic_year, status)
		VALUES (?, ?, ?, ?, 'PENDING')
	`)
	if err != nil {
		log.Printf("Error preparing insert statement: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer insertStmt.Close()

	courseStmt, err := tx.Prepare(`INSERT IGNORE INTO student_courses (student_id, course_id) VALUES (?, ?)`)
	if err != nil {
		log.Printf("Error preparing student_courses statement: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer courseStmt.Close()

	successCount := 0
	seenSelectionIDs := make(map[int]struct{})
	for _, hodSelID := range selectionIDs {
		if hodSelID == 0 {
			log.Printf("Skipping zero hod_selection_id")
			continue
		}
		if _, exists := seenSelectionIDs[hodSelID]; exists {
			continue
		}
		seenSelectionIDs[hodSelID] = struct{}{}

		// Validate: ensure this hod_selection_id is valid for this student's dept/batch/semester
		var courseID, hodDepartmentID int
		var courseCategory, slotName string
		err := validateStmt.QueryRow(hodSelID, requestBody.Semester, batch).Scan(&courseID, &hodDepartmentID, &courseCategory, &slotName)
		if err == sql.ErrNoRows {
			log.Printf("Invalid hod_selection_id %d (sem %d, batch %s)", hodSelID, requestBody.Semester, batch)
			http.Error(w, "Invalid selection", http.StatusBadRequest)
			return
		} else if err != nil {
			log.Printf("Error validating hod_selection_id %d: %v", hodSelID, err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if isOpenElectiveCategory(courseCategory, slotName) {
			if _, ok := eligibleOpenElectiveCourseIDs[courseID]; !ok {
				log.Printf("Rejected OE selection hod_selection_id=%d course_id=%d for student %d: not in curriculum %d", hodSelID, courseID, studentID, studentCurriculumID)
				http.Error(w, "Invalid Open Elective selection: not offered in your curriculum", http.StatusBadRequest)
				return
			}
		} else {
			if hodDepartmentID != departmentID {
				log.Printf("Rejected non-OE selection hod_selection_id=%d for student %d: HOD dept %d != student dept %d", hodSelID, studentID, hodDepartmentID, departmentID)
				http.Error(w, "Invalid selection: course not available for your department", http.StatusBadRequest)
				return
			}
		}

		// Insert into student_elective_choices
		if _, err = insertStmt.Exec(studentID, hodSelID, requestBody.Semester, selectionAcademicYear); err != nil {
			log.Printf("Error inserting selection hod_sel_id=%d: %v", hodSelID, err)
			http.Error(w, fmt.Sprintf("Failed to save selection: %v", err), http.StatusInternalServerError)
			return
		}
		// Record in student_courses
		if _, err = courseStmt.Exec(studentID, courseID); err != nil {
			log.Printf("Error inserting student_courses course_id=%d: %v", courseID, err)
			http.Error(w, fmt.Sprintf("Failed to add course to student record: %v", err), http.StatusInternalServerError)
			return
		}
		successCount++
		log.Printf("Saved slot=%s hod_sel_id=%d course_id=%d for student %d", slotName, hodSelID, courseID, studentID)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully saved %d elective selections for student_id %d", successCount, studentID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Selections saved successfully",
		"courses_saved": successCount,
	})
}

// GetStudentElectiveSelections retrieves a student's saved elective choices
func GetStudentElectiveSelections(w http.ResponseWriter, r *http.Request) {
	// Get email from query parameter
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Fetching saved selections for email: %s", email)

	// Get student_id from contact_details using student_email
	var studentID int
	err := db.DB.QueryRow(`
		SELECT student_id 
		FROM contact_details 
		WHERE student_email = ?
	`, email).Scan(&studentID)
	
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Student not found with this email", http.StatusNotFound)
		} else {
			log.Printf("Error fetching student: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	query := `
		SELECT 
			sec.choice_id,
			sec.course_id,
			sec.semester,
			c.course_code,
			c.course_name,
			c.credits,
			c.category,
			sec.selected_at
		FROM student_elective_choices sec
		INNER JOIN courses c ON sec.course_id = c.course_id
		WHERE sec.student_id = ?
		ORDER BY sec.semester, c.course_code
	`

	rows, err := db.DB.Query(query, studentID)
	if err != nil {
		log.Printf("Error fetching student selections: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var selections []map[string]interface{}
	for rows.Next() {
		var (
			choiceID   int
			courseID   int
			semester   int
			courseCode string
			courseName string
			credits    int
			category   string
			selectedAt string
		)

		err := rows.Scan(
			&choiceID, &courseID, &semester, &courseCode, 
			&courseName, &credits, &category, &selectedAt,
		)
		if err != nil {
			log.Printf("Error scanning selection: %v", err)
			continue
		}

		selections = append(selections, map[string]interface{}{
			"choice_id":   choiceID,
			"course_id":   courseID,
			"semester":    semester,
			"course_code": courseCode,
			"course_name": courseName,
			"credits":     credits,
			"category":    category,
			"selected_at": selectedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(selections)
}

// Helper function to determine slot type based on slot name and category
func determineSlotType(slotName, category string) string {
	slotNameLower := strings.ToLower(strings.TrimSpace(slotName))

	log.Printf("Determining slot type for: '%s' (category: '%s')", slotName, category)

	// Check slot name — explicit slot names always win
	if strings.Contains(slotNameLower, "honour") || strings.Contains(slotNameLower, "honor") {
		log.Printf("  -> Detected as HONOR")
		return "HONOR"
	}
	if strings.Contains(slotNameLower, "minor") {
		log.Printf("  -> Detected as MINOR")
		return "MINOR"
	}
	if strings.Contains(slotNameLower, "addon") || strings.Contains(slotNameLower, "add-on") || strings.Contains(slotNameLower, "add on") {
		log.Printf("  -> Detected as ADDON")
		return "ADDON"
	}
	if strings.Contains(slotNameLower, "open") {
		log.Printf("  -> Detected as OPEN")
		return "OPEN"
	}

	// Everything else defaults to PROFESSIONAL.
	// Note: if all courses in the slot share a different category (e.g. Add-On),
	// the caller reclassifies after all courses are loaded.
	log.Printf("  -> Defaulting to PROFESSIONAL")
	return "PROFESSIONAL"
}

// Helper function to handle mixed slots
func handleMixedSlots(slots []ElectiveSlot, semester int) []ElectiveSlot {
	if len(slots) == 0 {
		return slots
	}

	var professionalSlots []*ElectiveSlot
	var dedicatedOpenExists bool

	for i := range slots {
		if slots[i].SlotType == "PROFESSIONAL" {
			professionalSlots = append(professionalSlots, &slots[i])
		} else if slots[i].SlotType == "OPEN" {
			dedicatedOpenExists = true
		}
	}

	if len(professionalSlots) > 1 && !dedicatedOpenExists {
		lastProfSlot := professionalSlots[len(professionalSlots)-1]
		hasOpenElectives := false
		for _, course := range lastProfSlot.Courses {
			if strings.Contains(strings.ToLower(course.Category), "open") {
				hasOpenElectives = true
				break
			}
		}
		if hasOpenElectives {
			lastProfSlot.SlotType = "MIXED"
			log.Printf("Marked last professional slot as MIXED (%s) for semester %d", lastProfSlot.SlotName, semester)
		} else {
			log.Printf("Last PE slot has no open electives, keeping separate for semester %d", semester)
		}
	} else if dedicatedOpenExists {
		log.Printf("Keeping dedicated Open Elective slot separate for semester %d", semester)
	} else {
		log.Printf("Only 1 PE slot, keeping all %d slots separate for semester %d", len(slots), semester)
	}

	return slots
}
		