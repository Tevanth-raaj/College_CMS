package studentteacher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"server/db"

	"github.com/xuri/excelize/v2"
)

// DownloadStudentImportTemplate serves an xlsx template for bulk student import.
func DownloadStudentImportTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: Students
	sheet1 := "Students"
	f.SetSheetName("Sheet1", sheet1)

	headers := []string{
		"register_no", "enrollment_no", "student_name", "email",
		"department", "semester", "learning_mode",
	}
	colWidths := []float64{18, 18, 28, 32, 14, 12, 16}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "1D4ED8", Style: 1},
			{Type: "right", Color: "1D4ED8", Style: 1},
			{Type: "bottom", Color: "1D4ED8", Style: 2},
		},
	})
	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"EFF6FF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})

	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "1"
		f.SetCellValue(sheet1, cell, h)
		f.SetCellStyle(sheet1, cell, cell, headerStyle)
		f.SetColWidth(sheet1, col, col, colWidths[i])
	}
	f.SetRowHeight(sheet1, 1, 28)

	examples := []string{"RN2024001", "EN2024001", "John Doe", "john.doe@student.edu", "CS", "1", "UAL"}
	for i, val := range examples {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := col + "2"
		f.SetCellValue(sheet1, cell, val)
		f.SetCellStyle(sheet1, cell, cell, exampleStyle)
	}
	f.SetRowHeight(sheet1, 2, 20)
	f.SetPanes(sheet1, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	// Sheet 2: Field Guide
	sheet2 := "Field Guide"
	f.NewSheet(sheet2)

	guideHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"0F172A"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	requiredStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "DC2626", Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	optionalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "6B7280"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	f.SetCellValue(sheet2, "A1", "Field")
	f.SetCellValue(sheet2, "B1", "Required")
	f.SetCellValue(sheet2, "C1", "Description / Allowed Values")
	f.SetCellStyle(sheet2, "A1", "C1", guideHeaderStyle)
	f.SetColWidth(sheet2, "A", "A", 20)
	f.SetColWidth(sheet2, "B", "B", 12)
	f.SetColWidth(sheet2, "C", "C", 65)

	guide := [][]string{
		{"register_no", "Yes", "University register number (e.g. RN2024001)"},
		{"enrollment_no", "Yes", "College enrollment number (e.g. EN2024001)"},
		{"student_name", "Yes", "Full name of the student"},
		{"email", "No", "Student email address"},
		{"department", "No", "Department code (e.g. CS, EC, EE, ME, CB, AI, IT, AG)"},
		{"semester", "No", "Current semester as a number (1 to 8)"},
		{"learning_mode", "No", "UAL  or  PBL  (case-insensitive)"},
	}
	for i, row := range guide {
		rn := strconv.Itoa(i + 2)
		f.SetCellValue(sheet2, "A"+rn, row[0])
		f.SetCellValue(sheet2, "B"+rn, row[1])
		f.SetCellValue(sheet2, "C"+rn, row[2])
		if row[1] == "Yes" {
			f.SetCellStyle(sheet2, "B"+rn, "B"+rn, requiredStyle)
		} else {
			f.SetCellStyle(sheet2, "B"+rn, "B"+rn, optionalStyle)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("Error generating template: %v", err)
		http.Error(w, "Failed to generate template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=student_import_template.xlsx")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

// DownloadComprehensiveStudentImportTemplate serves a comprehensive xlsx template with all student fields.
func DownloadComprehensiveStudentImportTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: Students
	sheet1 := "Students"
	f.SetSheetName("Sheet1", sheet1)

	headers := []string{
		// Identification
		"enrollment_no", "register_no", "dte_reg_no", "application_no", "admission_no",
		// Basic info
		"student_name", "gender", "dob", "age",
		"father_name", "mother_name", "guardian_name",
		"religion", "nationality", "community", "mother_tongue",
		"blood_group", "aadhar_no",
		"parent_occupation", "designation", "place_of_work", "parent_income",
		// Academic
		"batch", "year", "semester", "degree_level", "section",
		"department", "student_category", "branch_type", "seat_category",
		"regulation", "quota", "university",
		"year_of_admission", "year_of_completion", "student_status", "curriculum_id",
		// Address
		"permanent_address", "present_address", "residence_location",
		// Admission payment
		"dte_register_no", "dte_admission_no", "receipt_no", "receipt_date", "amount", "bank_name",
		// Contact
		"parent_mobile", "student_mobile", "student_email", "parent_email", "official_email",
		// Hostel
		"hosteller_type", "hostel_name", "room_no", "room_capacity",
		"room_type", "floor_no", "warden_name", "alternate_warden", "class_advisor",
		// Insurance
		"nominee_name", "relationship", "nominee_age",
		// School (single record per row)
		"school_name", "board", "year_of_pass", "state", "tc_no", "tc_date", "total_marks",
	}

	// Style: bold header
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"EFF6FF"}, Pattern: 1},
	})

	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		f.SetCellValue(sheet1, cell, h)
		f.SetCellStyle(sheet1, cell, cell, headerStyle)
		colName, _ := excelize.ColumnNumberToName(colIdx + 1)
		f.SetColWidth(sheet1, colName, colName, 22)
	}
	f.SetRowHeight(sheet1, 1, 30)

	// Example data row
	example := []string{
		// Identification
		"EN2024001", "RN2024001", "DTE2024001", "APP2024001", "ADM2024001",
		// Basic info
		"John Doe", "Male", "2003-05-15", "21",
		"James Doe", "Jane Doe", "Guardian Name",
		"Hindu", "Indian", "OC", "Tamil",
		"O+", "123456789012",
		"Engineer", "Senior Engineer", "Chennai", "500000",
		// Academic
		"2024-2028", "1", "1", "B.E.", "A",
		"CS", "Regular", "Main", "Management",
		"R2021", "Management", "Anna University",
		"2024", "2028", "Active", "1",
		// Address
		"123 Main St, Chennai 600001", "45 College View, Coimbatore 641001", "Urban",
		// Admission payment
		"DTE2024001", "ADM2024001", "RCT001", "2024-08-01", "75000", "State Bank of India",
		// Contact
		"9876543210", "9123456789", "john.doe@student.edu", "james.doe@email.com", "john.2024@college.edu",
		// Hostel
		"Hosteller", "Boys Hostel Block A", "101", "2",
		"Double", "1", "Mr. Warden", "Mr. Alt Warden", "Dr. Advisor",
		// Insurance
		"Jane Doe", "Mother", "47",
		// School
		"ABC Higher Secondary School", "State Board", "2022", "Tamil Nadu", "TC001", "2022-05-30", "485",
	}

	for colIdx, val := range example {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 2)
		f.SetCellValue(sheet1, cell, val)
		f.SetCellStyle(sheet1, cell, cell, exampleStyle)
	}
	f.SetRowHeight(sheet1, 2, 20)

	// Freeze header row
	f.SetPanes(sheet1, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// ── Sheet 2: Field Guide ───────────────────────────────────────────────────

	sheet2 := "Field Guide"
	f.NewSheet(sheet2)

	guideHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"0F172A"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	sectionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "1D4ED8"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DBEAFE"}, Pattern: 1},
	})

	f.SetCellValue(sheet2, "A1", "Field Name")
	f.SetCellValue(sheet2, "B1", "Description / Allowed Values")
	f.SetCellStyle(sheet2, "A1", "B1", guideHeaderStyle)
	f.SetColWidth(sheet2, "A", "A", 30)
	f.SetColWidth(sheet2, "B", "B", 70)

	guide := [][]string{
		// ----- section -----
		{"── IDENTIFICATION ──", ""},
		{"enrollment_no", "College enrollment number (Required)"},
		{"register_no", "University register number (Required)"},
		{"dte_reg_no", "DTE registration number"},
		{"application_no", "Application number"},
		{"admission_no", "Admission number"},
		// ----- section -----
		{"── BASIC INFO ──", ""},
		{"student_name", "Full name of the student (Required)"},
		{"gender", "Male / Female / Other"},
		{"dob", "Date of birth in YYYY-MM-DD format (e.g. 2003-05-15)"},
		{"age", "Age in years (integer)"},
		{"father_name", "Father's full name"},
		{"mother_name", "Mother's full name"},
		{"guardian_name", "Guardian's name (if applicable)"},
		{"religion", "Religion (e.g. Hindu, Christian, Muslim, Others)"},
		{"nationality", "Nationality (default: Indian)"},
		{"community", "Community (e.g. OC, BC, MBC, SC, ST)"},
		{"mother_tongue", "Mother tongue language"},
		{"blood_group", "Blood group (e.g. O+, A-, B+, AB+)"},
		{"aadhar_no", "12-digit Aadhar number"},
		{"parent_occupation", "Parent's occupation"},
		{"designation", "Parent's job designation"},
		{"place_of_work", "Parent's place of work"},
		{"parent_income", "Annual parent income (number, e.g. 500000)"},
		// ----- section -----
		{"── ACADEMIC DETAILS ──", ""},
		{"batch", "Academic batch/year range (e.g. 2024-2028)"},
		{"year", "Current year of study (1 / 2 / 3 / 4)"},
		{"semester", "Current semester (1–8)"},
		{"degree_level", "Degree level (e.g. B.E., B.Tech, M.E., M.Tech)"},
		{"section", "Class section (e.g. A, B, C)"},
		{"department", "Department code (e.g. CS, EC, EE, ME, CB, AI, IT)"},
		{"student_category", "Category (Regular / Lateral Entry / Exchange)"},
		{"branch_type", "Branch type (Main / Aided / SF)"},
		{"seat_category", "Seat category (Management / Government / NRI)"},
		{"regulation", "Regulation code (e.g. R2021, R2019)"},
		{"quota", "Admission quota (Management / Government / NRI)"},
		{"university", "Affiliated university name"},
		{"year_of_admission", "Year of joining (e.g. 2024)"},
		{"year_of_completion", "Expected year of completion (e.g. 2028)"},
		{"student_status", "Active / Detained / Dropped / Passed Out"},
		{"curriculum_id", "Numeric ID of the assigned curriculum (leave blank if unknown)"},
		// ----- section -----
		{"── ADDRESS ──", ""},
		{"permanent_address", "Full permanent address"},
		{"present_address", "Current residential address"},
		{"residence_location", "Urban / Rural / Semi-Urban"},
		// ----- section -----
		{"── ADMISSION PAYMENT ──", ""},
		{"dte_register_no", "DTE register number on payment receipt"},
		{"dte_admission_no", "DTE admission number on payment receipt"},
		{"receipt_no", "Payment receipt number"},
		{"receipt_date", "Receipt date in YYYY-MM-DD format"},
		{"amount", "Amount paid (number, e.g. 75000)"},
		{"bank_name", "Name of the bank"},
		// ----- section -----
		{"── CONTACT DETAILS ──", ""},
		{"parent_mobile", "Parent's 10-digit mobile number"},
		{"student_mobile", "Student's 10-digit mobile number"},
		{"student_email", "Student's personal email address"},
		{"parent_email", "Parent's email address"},
		{"official_email", "Official college-issued email address"},
		// ----- section -----
		{"── HOSTEL DETAILS ──", "(fill only if student is a hosteller)"},
		{"hosteller_type", "Hosteller / Day Scholar"},
		{"hostel_name", "Name of the hostel block"},
		{"room_no", "Room number"},
		{"room_capacity", "Number of occupants allowed in the room"},
		{"room_type", "Single / Double / Triple"},
		{"floor_no", "Floor number (integer)"},
		{"warden_name", "Warden's name"},
		{"alternate_warden", "Alternate warden's name"},
		{"class_advisor", "Class advisor's name"},
		// ----- section -----
		{"── INSURANCE ──", ""},
		{"nominee_name", "Insurance nominee's full name"},
		{"relationship", "Nominee's relationship (Father / Mother / Sibling etc.)"},
		{"nominee_age", "Nominee's age (integer)"},
		// ----- section -----
		{"── SCHOOL DETAILS ── (last qualification)", "(one school record per student row)"},
		{"school_name", "Name of the school / junior college"},
		{"board", "Exam board (e.g. State Board, CBSE, ICSE)"},
		{"year_of_pass", "Year of passing (e.g. 2022)"},
		{"state", "State where school is located"},
		{"tc_no", "Transfer certificate number"},
		{"tc_date", "TC issue date in YYYY-MM-DD format"},
		{"total_marks", "Total marks scored (number, e.g. 485)"},
	}

	for rowIdx, row := range guide {
		cellA, _ := excelize.CoordinatesToCellName(1, rowIdx+2)
		cellB, _ := excelize.CoordinatesToCellName(2, rowIdx+2)
		f.SetCellValue(sheet2, cellA, row[0])
		f.SetCellValue(sheet2, cellB, row[1])
		if strings.HasPrefix(row[0], "──") {
			f.SetCellStyle(sheet2, cellA, cellB, sectionStyle)
		}
	}

	// ── Write to response ──────────────────────────────────────────────────────

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		log.Printf("Error generating student import template: %v", err)
		http.Error(w, "Failed to generate template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=student_import_template.xlsx")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

// ─────────────────────────────────────────────────────────────────────────────

// ImportStudentsResult is returned by the import endpoint.
type ImportStudentsResult struct {
	Inserted int                     `json:"inserted"`
	Skipped  int                     `json:"skipped"`
	Errors   []ImportStudentRowError `json:"errors"`
}

// ImportStudentRowError describes a per-row error during import.
type ImportStudentRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ImportStudents handles POST /api/students/import.
// Accepts a multipart form with field "file" containing an xlsx spreadsheet.
func ImportStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not parse form: " + err.Error()})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded (field name must be 'file')"})
		return
	}
	defer file.Close()

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot open xlsx: " + err.Error()})
		return
	}
	defer xlsx.Close()

	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No sheets found in the file"})
		return
	}

	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Cannot read rows: " + err.Error()})
		return
	}

	if len(rows) < 2 {
		json.NewEncoder(w).Encode(ImportStudentsResult{Inserted: 0, Skipped: 0, Errors: []ImportStudentRowError{}})
		return
	}

	// Build header → column index map (case-insensitive)
	colIdx := map[string]int{}
	for i, cell := range rows[0] {
		colIdx[strings.ToLower(strings.TrimSpace(cell))] = i
	}

	get := func(row []string, field string) string {
		i, ok := colIdx[field]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	result := ImportStudentsResult{Errors: []ImportStudentRowError{}}

	for rowNum, row := range rows[1:] {
		xlsxRow := rowNum + 2 // 1-based, accounting for header

		registerNo := get(row, "register_no")
		enrollmentNo := get(row, "enrollment_no")
		studentName := get(row, "student_name")

		// Skip completely blank rows
		if registerNo == "" && enrollmentNo == "" && studentName == "" {
			result.Skipped++
			continue
		}
		if studentName == "" {
			result.Errors = append(result.Errors, ImportStudentRowError{Row: xlsxRow, Message: "student_name is required"})
			continue
		}
		if enrollmentNo == "" {
			result.Errors = append(result.Errors, ImportStudentRowError{Row: xlsxRow, Message: "enrollment_no is required"})
			continue
		}
		if registerNo == "" {
			result.Errors = append(result.Errors, ImportStudentRowError{Row: xlsxRow, Message: "register_no is required"})
			continue
		}

		email := get(row, "email")
		department := get(row, "department")
		semesterStr := get(row, "semester")
		learningMode := get(row, "learning_mode")

		if err := insertSimpleStudent(registerNo, enrollmentNo, studentName, email, department, semesterStr, learningMode); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "Duplicate entry") {
				if strings.Contains(msg, "enrollment_no") {
					msg = fmt.Sprintf("enrollment_no '%s' already exists", enrollmentNo)
				} else if strings.Contains(msg, "register_no") {
					msg = fmt.Sprintf("register_no '%s' already exists", registerNo)
				}
			}
			result.Errors = append(result.Errors, ImportStudentRowError{Row: xlsxRow, Message: msg})
			continue
		}
		result.Inserted++
	}

	json.NewEncoder(w).Encode(result)
}

// parseLearningModeID converts "UAL"/"PBL" (or "1"/"2") to 1 or 2. Returns 0 if unrecognised.
func parseLearningModeID(s string) int {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "UAL", "1":
		return 1
	case "PBL", "2":
		return 2
	}
	return 0
}

// insertSimpleStudent inserts a student using only the 7 simplified import fields.
func insertSimpleStudent(registerNo, enrollmentNo, studentName, email, department, semesterStr, learningMode string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	lmID := parseLearningModeID(learningMode)

	// students table
	var studentID int64
	switch {
	case lmID != 0 && email != "":
		r, e := tx.Exec(`INSERT INTO students (enrollment_no, register_no, student_name, email, learning_mode_id, status) VALUES (?, ?, ?, ?, ?, 1)`,
			enrollmentNo, registerNo, studentName, email, lmID)
		if e != nil {
			tx.Rollback()
			return fmt.Errorf("insert student: %w", e)
		}
		studentID, _ = r.LastInsertId()
	case lmID != 0:
		r, e := tx.Exec(`INSERT INTO students (enrollment_no, register_no, student_name, learning_mode_id, status) VALUES (?, ?, ?, ?, 1)`,
			enrollmentNo, registerNo, studentName, lmID)
		if e != nil {
			tx.Rollback()
			return fmt.Errorf("insert student: %w", e)
		}
		studentID, _ = r.LastInsertId()
	case email != "":
		r, e := tx.Exec(`INSERT INTO students (enrollment_no, register_no, student_name, email, status) VALUES (?, ?, ?, ?, 1)`,
			enrollmentNo, registerNo, studentName, email)
		if e != nil {
			tx.Rollback()
			return fmt.Errorf("insert student: %w", e)
		}
		studentID, _ = r.LastInsertId()
	default:
		r, e := tx.Exec(`INSERT INTO students (enrollment_no, register_no, student_name, status) VALUES (?, ?, ?, 1)`,
			enrollmentNo, registerNo, studentName)
		if e != nil {
			tx.Rollback()
			return fmt.Errorf("insert student: %w", e)
		}
		studentID, _ = r.LastInsertId()
	}

	// academic_details
	if department != "" || semesterStr != "" {
		semester := 0
		if semesterStr != "" {
			semester, _ = strconv.Atoi(semesterStr)
		}
		_, err = tx.Exec(
			`INSERT INTO academic_details (student_id, department, semester) VALUES (?, ?, ?)
			 ON DUPLICATE KEY UPDATE department = VALUES(department), semester = VALUES(semester)`,
			studentID, department, semester)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("insert academic_details: %w", err)
		}
	}

	// contact_details
	if email != "" {
		_, err = tx.Exec(
			`INSERT INTO contact_details (student_id, student_email) VALUES (?, ?)
			 ON DUPLICATE KEY UPDATE student_email = VALUES(student_email)`,
			studentID, email)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("insert contact_details: %w", err)
		}
	}

	return tx.Commit()
}
