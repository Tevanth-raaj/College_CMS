package studentteacher

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

	"github.com/gorilla/mux"
)

// GetStudents retrieves all students from the database
func GetStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	yearLevelParam := strings.TrimSpace(r.URL.Query().Get("year_level"))

	var (
		yearLevel int
		err       error
	)
	if yearLevelParam != "" {
		yearLevel, err = strconv.Atoi(yearLevelParam)
		if err != nil || yearLevel < 1 || yearLevel > 4 {
			http.Error(w, "year_level must be between 1 and 4", http.StatusBadRequest)
			return
		}
	}

	query := `
			SELECT 
				s.id, 
				COALESCE(s.enrollment_no, ''), 
				COALESCE(s.register_no, ''), 
				COALESCE(s.dte_reg_no, ''), 
				COALESCE(s.application_no, ''), 
				COALESCE(s.admission_no, ''), 
				s.student_name, 
				COALESCE(s.gender, ''), 
				COALESCE(CAST(s.dob AS CHAR), ''), 
				COALESCE(s.age, 0),
				COALESCE(s.father_name, ''), 
				COALESCE(s.mother_name, ''), 
				COALESCE(s.guardian_name, ''), 
				COALESCE(s.religion, ''), 
				COALESCE(s.nationality, ''),
				COALESCE(s.community, ''), 
				COALESCE(s.mother_tongue, ''), 
				COALESCE(s.blood_group, ''), 
				COALESCE(s.aadhar_no, ''), 
				COALESCE(s.parent_occupation, ''),
				COALESCE(s.designation, ''), 
				COALESCE(s.place_of_work, ''), 
				COALESCE(s.parent_income, 0), 
				COALESCE(s.status, 1),
				s.learning_mode_id,
				COALESCE(NULLIF(TRIM(lm.code), ''), NULLIF(TRIM(lm.name), ''), '') AS learning_mode,
				s.department_id,
				COALESCE(ac.year_level, 0) AS year,
				COALESCE(d.department_code, '') AS department_code,
				COALESCE(NULLIF(TRIM(cd.student_email), ''), NULLIF(TRIM(cd.official_email), ''), NULLIF(TRIM(s.email), ''), '') AS mail_id
			FROM students s
			LEFT JOIN academic_details ad ON ad.student_id = s.id
			LEFT JOIN academic_calendar ac ON ac.id = s.year
			LEFT JOIN learning_modes lm ON lm.id = s.learning_mode_id
			LEFT JOIN departments d ON d.id = s.department_id
			LEFT JOIN contact_details cd ON cd.student_id = s.id
			WHERE s.status = 1
	`

	args := []interface{}{}
	if yearLevelParam != "" {
		query += "\n\t\t\tAND ac.year_level = ?"
		args = append(args, yearLevel)
	}

	query += `
			ORDER BY s.id DESC
		`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error querying students: %v", err)
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var students []models.Student
	for rows.Next() {
		var student models.Student
		err := rows.Scan(
			&student.StudentID, &student.EnrollmentNo, &student.RegisterNo,
			&student.DTERegNo, &student.ApplicationNo, &student.AdmissionNo,
			&student.StudentName, &student.Gender, &student.DOB, &student.Age,
			&student.FatherName, &student.MotherName, &student.GuardianName,
			&student.Religion, &student.Nationality, &student.Community,
			&student.MotherTongue, &student.BloodGroup, &student.AadharNo,
			&student.ParentOccupation, &student.Designation, &student.PlaceOfWork,
			&student.ParentIncome, &student.Status,
			&student.LearningModeID, &student.LearningMode, &student.DepartmentID,
			&student.Year, &student.DepartmentCode, &student.MailID,
		)
		if err != nil {
			log.Printf("Error scanning student row: %v", err)
			continue
		}
		students = append(students, student)
	}

	if students == nil {
		students = []models.Student{}
	}

	json.NewEncoder(w).Encode(students)
}

// GetStudent retrieves a single student by ID with all details
func GetStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	studentID := vars["id"]

	log.Printf("GetStudent called for ID: %s", studentID)

	// 1. Fetch Basic Details
	query := `
			SELECT 
				id, 
				COALESCE(enrollment_no, ''), 
				COALESCE(register_no, ''), 
				COALESCE(dte_reg_no, ''), 
				COALESCE(application_no, ''), 
				COALESCE(admission_no, ''), 
				student_name, 
				COALESCE(gender, ''), 
				COALESCE(CAST(dob AS CHAR), ''), 
				COALESCE(age, 0),
				COALESCE(father_name, ''), 
				COALESCE(mother_name, ''), 
				COALESCE(guardian_name, ''), 
				COALESCE(religion, ''), 
				COALESCE(nationality, ''),
				COALESCE(community, ''), 
				COALESCE(mother_tongue, ''), 
				COALESCE(blood_group, ''), 
				COALESCE(aadhar_no, ''), 
				COALESCE(parent_occupation, ''),
				COALESCE(designation, ''), 
				COALESCE(place_of_work, ''), 
				COALESCE(parent_income, 0), 
				COALESCE(status, 1)
			FROM students
			WHERE id = ?
		`

	var student models.Student
	err := db.DB.QueryRow(query, studentID).Scan(
		&student.StudentID, &student.EnrollmentNo, &student.RegisterNo,
		&student.DTERegNo, &student.ApplicationNo, &student.AdmissionNo,
		&student.StudentName, &student.Gender, &student.DOB, &student.Age,
		&student.FatherName, &student.MotherName, &student.GuardianName,
		&student.Religion, &student.Nationality, &student.Community,
		&student.MotherTongue, &student.BloodGroup, &student.AadharNo,
		&student.ParentOccupation, &student.Designation, &student.PlaceOfWork,
		&student.ParentIncome, &student.Status,
	)

	if err == sql.ErrNoRows {
		log.Printf("Student ID %s not found in students table", studentID)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying student: %v", err)
		http.Error(w, "Failed to fetch student", http.StatusInternalServerError)
		return
	}

	fullStudent := models.FullStudent{
		Student: &student,
	}

	// 2. Fetch Academic Details
	var acad models.AcademicDetails
	acadQuery := `
		SELECT 
			COALESCE(batch, ''), COALESCE(year, 0), COALESCE(semester, 0), COALESCE(degree_level, ''),
			COALESCE(section, ''), COALESCE(department, ''), COALESCE(student_category, ''),
			COALESCE(branch_type, ''), COALESCE(seat_category, ''), COALESCE(regulation, ''),
			COALESCE(quota, ''), COALESCE(university, ''), COALESCE(year_of_admission, 0),
			COALESCE(year_of_completion, 0), COALESCE(student_status, ''), COALESCE(curriculum_id, 0)
		FROM academic_details WHERE student_id = ?`

	err = db.DB.QueryRow(acadQuery, student.StudentID).Scan(
		&acad.Batch, &acad.Year, &acad.Semester, &acad.DegreeLevel,
		&acad.Section, &acad.Department, &acad.StudentCategory,
		&acad.BranchType, &acad.SeatCategory, &acad.Regulation,
		&acad.Quota, &acad.University, &acad.YearOfAdmission,
		&acad.YearOfCompletion, &acad.StudentStatus, &acad.CurriculumID,
	)
	if err == nil {
		acad.StudentID = student.StudentID
		fullStudent.AcademicDetails = &acad
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching academic_details for student %d: %v", student.StudentID, err)
	}

	// 3. Fetch Address
	var addr models.Address
	addrQuery := `
		SELECT COALESCE(permanent_address, ''), COALESCE(present_address, ''), COALESCE(residence_location, '')
		FROM address WHERE student_id = ?`
	err = db.DB.QueryRow(addrQuery, student.StudentID).Scan(
		&addr.PermanentAddress, &addr.PresentAddress, &addr.ResidenceLocation,
	)
	if err == nil {
		addr.StudentID = student.StudentID
		fullStudent.Address = &addr
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching address for student %d: %v", student.StudentID, err)
	}

	// 4. Fetch Admission Payment
	var pay models.AdmissionPayment
	payQuery := `
		SELECT 
			COALESCE(dte_register_no, ''), COALESCE(dte_admission_no, ''), COALESCE(receipt_no, ''),
			COALESCE(CAST(receipt_date AS CHAR), ''), COALESCE(amount, 0), COALESCE(bank_name, '')
		FROM admission_payment WHERE student_id = ?`
	err = db.DB.QueryRow(payQuery, student.StudentID).Scan(
		&pay.DTERegisterNo, &pay.DTEAdmissionNo, &pay.ReceiptNo,
		&pay.ReceiptDate, &pay.Amount, &pay.BankName,
	)
	if err == nil {
		pay.StudentID = student.StudentID
		fullStudent.AdmissionPayment = &pay
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching admission_payment for student %d: %v", student.StudentID, err)
	}

	// 5. Fetch Contact Details
	var contact models.ContactDetails
	contactQuery := `
		SELECT 
			COALESCE(parent_mobile, ''), COALESCE(student_mobile, ''), COALESCE(student_email, ''),
			COALESCE(parent_email, ''), COALESCE(official_email, '')
		FROM contact_details WHERE student_id = ?`
	err = db.DB.QueryRow(contactQuery, student.StudentID).Scan(
		&contact.ParentMobile, &contact.StudentMobile, &contact.StudentEmail,
		&contact.ParentEmail, &contact.OfficialEmail,
	)
	if err == nil {
		contact.StudentID = student.StudentID
		fullStudent.ContactDetails = &contact
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching contact_details for student %d: %v", student.StudentID, err)
	}

	// 6. Fetch Hostel Details
	var hostel models.HostelDetails
	hostelQuery := `
		SELECT 
			COALESCE(hosteller_type, ''), COALESCE(hostel_name, ''), COALESCE(room_no, ''),
			COALESCE(room_capacity, 0), COALESCE(room_type, ''), COALESCE(floor_no, 0),
			COALESCE(warden_name, ''), COALESCE(alternate_warden, ''), COALESCE(class_advisor, ''),
			COALESCE(status, 1)
		FROM hostel_details WHERE student_id = ?`
	err = db.DB.QueryRow(hostelQuery, student.StudentID).Scan(
		&hostel.HostellerType, &hostel.HostelName, &hostel.RoomNo,
		&hostel.RoomCapacity, &hostel.RoomType, &hostel.FloorNo,
		&hostel.WardenName, &hostel.AlternateWarden, &hostel.ClassAdvisor,
		&hostel.Status,
	)
	if err == nil {
		hostel.StudentID = student.StudentID
		fullStudent.HostelDetails = &hostel
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching hostel_details for student %d: %v", student.StudentID, err)
	}

	// 7. Fetch Insurance Details
	var ins models.InsuranceDetails
	insQuery := `
		SELECT COALESCE(nominee_name, ''), COALESCE(relationship, ''), COALESCE(nominee_age, 0), COALESCE(status, 1)
		FROM insurance_details WHERE student_id = ?`
	err = db.DB.QueryRow(insQuery, student.StudentID).Scan(
		&ins.NomineeName, &ins.Relationship, &ins.NomineeAge, &ins.Status,
	)
	if err == nil {
		ins.StudentID = student.StudentID
		fullStudent.InsuranceDetails = &ins
	} else if err != sql.ErrNoRows {
		log.Printf("Error fetching insurance_details for student %d: %v", student.StudentID, err)
	}

	// 8. Fetch School Details (Multiple)
	schoolQuery := `
		SELECT 
			id, school_name, board, year_of_pass, state, tc_no, COALESCE(CAST(tc_date AS CHAR), ''), total_marks, status
		FROM school_details WHERE student_id = ?`
	rows, err := db.DB.Query(schoolQuery, student.StudentID)
	if err == nil {
		defer rows.Close()
		var schools []models.SchoolDetails
		for rows.Next() {
			var s models.SchoolDetails
			// Use temporary variables to handle potential NULLs safely
			var sn, sb, st, tn, td sql.NullString
			var yp sql.NullInt64
			var tm sql.NullFloat64
			var stat sql.NullInt64

			if err := rows.Scan(&s.ID, &sn, &sb, &yp, &st, &tn, &td, &tm, &stat); err == nil {
				s.StudentID = student.StudentID
				s.SchoolName = sn.String
				s.Board = sb.String
				s.YearOfPass = int(yp.Int64)
				s.State = st.String
				s.TCNo = tn.String
				s.TCDate = td.String
				s.TotalMarks = tm.Float64
				s.Status = int(stat.Int64)
				schools = append(schools, s)
			} else {
				log.Printf("Error scanning school row: %v", err)
			}
		}
		fullStudent.SchoolDetails = schools
	} else {
		log.Printf("Error querying school_details for student %d: %v", student.StudentID, err)
	}

	json.NewEncoder(w).Encode(fullStudent)
}

// CreateStudent creates a new student record with all related details in transaction
func CreateStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		http.Error(w, "Failed to create student", http.StatusInternalServerError)
		return
	}

	// INSERT into students table
	insertStudentQuery := `
			INSERT INTO students (
				enrollment_no, register_no, dte_reg_no, application_no,
				admission_no, student_name, gender, dob, age, father_name, mother_name,
				guardian_name, religion, nationality, community, mother_tongue,
				blood_group, aadhar_no, parent_occupation, designation, place_of_work,
				parent_income, status
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)
		`

	result, err := tx.Exec(
		insertStudentQuery,
		req.EnrollmentNo, req.RegisterNo, req.DTERegNo, req.ApplicationNo,
		req.AdmissionNo, req.StudentName, req.Gender, req.DOB, parseInt(req.Age),
		req.FatherName, req.MotherName, req.GuardianName, req.Religion,
		req.Nationality, req.Community, req.MotherTongue, req.BloodGroup,
		req.AadharNo, req.ParentOccupation, req.Designation, req.PlaceOfWork,
		parseFloat(req.ParentIncome), 1,
	)

	if err != nil {
		tx.Rollback()
		log.Printf("Error inserting student: %v", err)
		http.Error(w, "Failed to create student", http.StatusInternalServerError)
		return
	}

	studentID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Printf("Error getting student ID: %v", err)
		http.Error(w, "Failed to create student", http.StatusInternalServerError)
		return
	}

	// INSERT into academic_details if provided
	if req.Department != "" || req.Batch != "" {
		acadQuery := `
				INSERT INTO academic_details (
					student_id, batch, year, semester, degree_level, section, department,
					student_category, branch_type, seat_category, regulation, quota,
					university, year_of_admission, year_of_completion, student_status, curriculum_id
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
		_, err := tx.Exec(
			acadQuery,
			studentID, req.Batch, parseInt(req.Year), parseInt(req.Semester), req.DegreeLevel,
			req.Section, req.Department, req.StudentCategory, req.BranchType,
			req.SeatCategory, req.Regulation, req.Quota, req.University,
			parseInt(req.YearOfAdmission), parseInt(req.YearOfCompletion), req.StudentStatus, parseNullableInt(req.CurriculumID),
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting academic details: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into address if provided
	if req.PermanentAddress != "" || req.PresentAddress != "" {
		addrQuery := `
				INSERT INTO address (student_id, permanent_address, present_address, residence_location)
				VALUES (?, ?, ?, ?)
			`
		_, err := tx.Exec(addrQuery, studentID, req.PermanentAddress, req.PresentAddress, req.ResidenceLocation)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting address: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into admission_payment if provided
	if req.ReceiptNo != "" || req.Amount != "" {
		payQuery := `
				INSERT INTO admission_payment (
					student_id, dte_register_no, dte_admission_no, receipt_no, receipt_date, amount, bank_name
				) VALUES (?, ?, ?, ?, ?, ?, ?)
			`
		_, err := tx.Exec(
			payQuery, studentID, req.DTERegisterNo, req.DTEAdmissionNo,
			req.ReceiptNo, req.ReceiptDate, parseFloat(req.Amount), req.BankName,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting admission payment: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into contact_details if provided
	if req.StudentEmail != "" || req.ParentMobile != "" {
		contactQuery := `
				INSERT INTO contact_details (
					student_id, parent_mobile, student_mobile, student_email, parent_email, official_email
				) VALUES (?, ?, ?, ?, ?, ?)
			`
		_, err := tx.Exec(
			contactQuery, studentID, req.ParentMobile, req.StudentMobile, req.StudentEmail,
			req.ParentEmail, req.OfficialEmail,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting contact details: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into hostel_details if provided
	if req.HostelName != "" {
		hostelQuery := `
				INSERT INTO hostel_details (
					student_id, hosteller_type, hostel_name, room_no, room_capacity,
					room_type, floor_no, warden_name, alternate_warden, class_advisor, status
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
		_, err := tx.Exec(
			hostelQuery, studentID, req.HostellerType, req.HostelName, req.RoomNo,
			parseInt(req.RoomCapacity), req.RoomType, parseInt(req.FloorNo), req.WardenName,
			req.AlternateWarden, req.ClassAdvisor, 1,
		)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting hostel details: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into insurance_details if provided
	if req.NomineeName != "" {
		insQuery := `
				INSERT INTO insurance_details (student_id, nominee_name, relationship, nominee_age, status)
				VALUES (?, ?, ?, ?, ?)
			`
		_, err := tx.Exec(insQuery, studentID, req.NomineeName, req.Relationship, parseInt(req.NomineeAge), 1)
		if err != nil {
			tx.Rollback()
			log.Printf("Error inserting insurance details: %v", err)
			http.Error(w, "Failed to create student", http.StatusInternalServerError)
			return
		}
	}

	// INSERT into school_details for each school if provided
	if len(req.SchoolDetails) > 0 {
		schoolQuery := `
				INSERT INTO school_details (
					student_id, school_name, board, year_of_pass, state, tc_no, tc_date, total_marks, status
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
		for _, school := range req.SchoolDetails {
			if school.SchoolName != "" {
				_, err := tx.Exec(
					schoolQuery, studentID, school.SchoolName, school.Board, parseInt(school.YearOfPass),
					school.State, school.TCNo, school.TCDate, parseFloat(school.TotalMarks), 1,
				)
				if err != nil {
					tx.Rollback()
					log.Printf("Error inserting school details: %v", err)
					http.Error(w, "Failed to create student", http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Auto-enroll student in courses from curriculum and semester
	if req.CurriculumID != "" && req.Semester != "" {
		curriculumID := parseNullableInt(req.CurriculumID)
		semester := parseInt(req.Semester)

		if curriculumID != nil && semester != 0 {
			// 1. Get distinct semester_ids
			var semesterIDs []int
			semesterQuery := `
            SELECT DISTINCT semester_id 
            FROM curriculum_courses 
            WHERE curriculum_id = ? 
            ORDER BY semester_id
        `
			sRows, err := tx.Query(semesterQuery, *curriculumID)
			if err != nil {
				log.Printf("Warning: Error fetching semester_ids: %v", err)
			} else {
				for sRows.Next() {
					var sid int
					if err := sRows.Scan(&sid); err == nil {
						semesterIDs = append(semesterIDs, sid)
					}
				}
				sRows.Close() // Close immediately after reading into the slice
			}

			// 2. Process courses if the requested semester index exists
			if len(semesterIDs) >= semester {
				semesterID := semesterIDs[semester-1]

				// Fetch course IDs into a slice first to free the connection
				var courseIDs []int
				coursesQuery := `SELECT DISTINCT course_id FROM curriculum_courses WHERE semester_id = ?`

				cRows, err := tx.Query(coursesQuery, semesterID)
				if err != nil {
					log.Printf("Warning: Error fetching courses: %v", err)
				} else {
					for cRows.Next() {
						var cid int
						if err := cRows.Scan(&cid); err == nil {
							courseIDs = append(courseIDs, cid)
						}
					}
					cRows.Close() // Close immediately
				}

				// 3. Now perform the inserts using the collected slice
				enrollQuery := `
                INSERT INTO student_courses (student_id, course_id) 
                VALUES (?, ?)
                ON DUPLICATE KEY UPDATE student_id = student_id
            `
				coursesEnrolled := 0
				for _, cid := range courseIDs {
					_, err := tx.Exec(enrollQuery, studentID, cid)
					if err != nil {
						// Log specific course error but don't necessarily kill the whole process
						log.Printf("Warning: Error auto-enrolling course %d for student %d: %v", cid, studentID, err)
						continue
					}
					coursesEnrolled++
				}

				log.Printf("Auto-enrolled student %d in %d courses for semester_id %d", studentID, coursesEnrolled, semesterID)
			} else {
				log.Printf("Warning: Semester %d not found for curriculum %d", semester, *curriculumID)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to create student", http.StatusInternalServerError)
		return
	}

	// Return created student
	student := models.Student{
		StudentID:        int(studentID),
		EnrollmentNo:     req.EnrollmentNo,
		RegisterNo:       req.RegisterNo,
		DTERegNo:         req.DTERegNo,
		ApplicationNo:    req.ApplicationNo,
		AdmissionNo:      req.AdmissionNo,
		StudentName:      req.StudentName,
		Gender:           req.Gender,
		DOB:              req.DOB,
		Age:              parseInt(req.Age),
		FatherName:       req.FatherName,
		MotherName:       req.MotherName,
		GuardianName:     req.GuardianName,
		Religion:         req.Religion,
		Nationality:      req.Nationality,
		Community:        req.Community,
		MotherTongue:     req.MotherTongue,
		BloodGroup:       req.BloodGroup,
		AadharNo:         req.AadharNo,
		ParentOccupation: req.ParentOccupation,
		Designation:      req.Designation,
		PlaceOfWork:      req.PlaceOfWork,
		ParentIncome:     parseFloat(req.ParentIncome),
		Status:           1,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

// UpdateStudent updates an existing student record and all optional related tables
func UpdateStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	studentID := vars["id"]

	// Log the student ID being updated
	log.Printf("UpdateStudent called with studentID: %s (type: string)", studentID)

	var req models.CreateStudentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse studentID to integer and verify it's valid
	studentIDInt := parseInt(studentID)
	if studentIDInt <= 0 {
		log.Printf("Invalid student ID: %s", studentID)
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}
	log.Printf("Parsed studentID to integer: %d", studentIDInt)

	// First, verify the student exists
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM students WHERE id = ?)", studentIDInt).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if student exists: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		log.Printf("Student ID %d not found in database", studentIDInt)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	log.Printf("Student ID %d exists, proceeding with update", studentIDInt)

	// Begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}

	// Defer rollback only if we haven't committed yet
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// UPDATE students table
	updateQuery := `
        UPDATE students SET
            enrollment_no = ?, register_no = ?, dte_reg_no = ?,
            application_no = ?, admission_no = ?, student_name = ?, gender = ?,
            dob = ?, age = ?, father_name = ?, mother_name = ?,
            guardian_name = ?, religion = ?, nationality = ?, community = ?,
            mother_tongue = ?, blood_group = ?, aadhar_no = ?,
            parent_occupation = ?, designation = ?, place_of_work = ?,
            parent_income = ?
        WHERE id = ?
    `

	log.Printf("Attempting to update student %d with: name=%s, enrollment=%s, income=%s",
		studentIDInt, req.StudentName, req.EnrollmentNo, req.ParentIncome)

	result, err := tx.Exec(
		updateQuery,
		req.EnrollmentNo, req.RegisterNo, req.DTERegNo, req.ApplicationNo,
		req.AdmissionNo, req.StudentName, req.Gender, req.DOB, parseInt(req.Age),
		req.FatherName, req.MotherName, req.GuardianName, req.Religion,
		req.Nationality, req.Community, req.MotherTongue, req.BloodGroup,
		req.AadharNo, req.ParentOccupation, req.Designation, req.PlaceOfWork,
		parseFloat(req.ParentIncome), studentIDInt,
	)

	if err != nil {
		log.Printf("Error executing UPDATE for student %d: %v", studentIDInt, err)
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting RowsAffected for student %d: %v", studentIDInt, err)
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}

	// Note: rowsAffected == 0 is OK if the data hasn't changed
	// MySQL returns 0 rows affected if no data actually changed
	log.Printf("UPDATE executed for student %d - %d rows affected", studentIDInt, rowsAffected)

	// Helper function to handle optional field updates
	updateOptionalFields := func() error {
		// UPDATE academic_details if provided
		if req.Department != "" || req.Batch != "" {
			// First check if record exists
			var acadExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM academic_details WHERE student_id = ?)", studentIDInt).Scan(&acadExists)
			if err != nil {
				return fmt.Errorf("checking academic_details: %v", err)
			}

			if acadExists {
				// Update existing record
				acadQuery := `
                    UPDATE academic_details SET
                        batch = ?, year = ?, semester = ?, degree_level = ?, section = ?,
                        department = ?, student_category = ?, branch_type = ?, seat_category = ?,
                        regulation = ?, quota = ?, university = ?, year_of_admission = ?,
                        year_of_completion = ?, student_status = ?, curriculum_id = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(acadQuery,
					req.Batch, parseInt(req.Year), parseInt(req.Semester), req.DegreeLevel,
					req.Section, req.Department, req.StudentCategory, req.BranchType,
					req.SeatCategory, req.Regulation, req.Quota, req.University,
					parseInt(req.YearOfAdmission), parseInt(req.YearOfCompletion),
					req.StudentStatus, parseNullableInt(req.CurriculumID), studentIDInt,
				)
				if err != nil {
					return fmt.Errorf("updating academic_details: %v", err)
				}
			} else {
				// Insert new record
				acadQuery := `
                    INSERT INTO academic_details (
                        student_id, batch, year, semester, degree_level, section, department,
                        student_category, branch_type, seat_category, regulation, quota,
                        university, year_of_admission, year_of_completion, student_status, curriculum_id
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                `
				_, err := tx.Exec(acadQuery,
					studentIDInt, req.Batch, parseInt(req.Year), parseInt(req.Semester), req.DegreeLevel,
					req.Section, req.Department, req.StudentCategory, req.BranchType,
					req.SeatCategory, req.Regulation, req.Quota, req.University,
					parseInt(req.YearOfAdmission), parseInt(req.YearOfCompletion),
					req.StudentStatus, parseNullableInt(req.CurriculumID),
				)
				if err != nil {
					return fmt.Errorf("inserting academic_details: %v", err)
				}
			}
		}

		// UPDATE address if provided
		if req.PermanentAddress != "" || req.PresentAddress != "" {
			var addrExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM address WHERE student_id = ?)", studentIDInt).Scan(&addrExists)
			if err != nil {
				return fmt.Errorf("checking address: %v", err)
			}

			if addrExists {
				addrQuery := `
                    UPDATE address SET
                        permanent_address = ?, present_address = ?, residence_location = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(addrQuery,
					req.PermanentAddress, req.PresentAddress, req.ResidenceLocation, studentIDInt)
				if err != nil {
					return fmt.Errorf("updating address: %v", err)
				}
			} else {
				addrQuery := `
                    INSERT INTO address (student_id, permanent_address, present_address, residence_location)
                    VALUES (?, ?, ?, ?)
                `
				_, err := tx.Exec(addrQuery, studentIDInt, req.PermanentAddress, req.PresentAddress, req.ResidenceLocation)
				if err != nil {
					return fmt.Errorf("inserting address: %v", err)
				}
			}
		}

		// UPDATE admission_payment if provided
		if req.ReceiptNo != "" || req.Amount != "" {
			var payExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM admission_payment WHERE student_id = ?)", studentIDInt).Scan(&payExists)
			if err != nil {
				return fmt.Errorf("checking admission_payment: %v", err)
			}

			if payExists {
				payQuery := `
                    UPDATE admission_payment SET
                        dte_register_no = ?, dte_admission_no = ?, receipt_no = ?,
                        receipt_date = ?, amount = ?, bank_name = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(payQuery,
					req.DTERegisterNo, req.DTEAdmissionNo, req.ReceiptNo,
					req.ReceiptDate, parseFloat(req.Amount), req.BankName, studentIDInt)
				if err != nil {
					return fmt.Errorf("updating admission_payment: %v", err)
				}
			} else {
				payQuery := `
                    INSERT INTO admission_payment (
                        student_id, dte_register_no, dte_admission_no, receipt_no, receipt_date, amount, bank_name
                    ) VALUES (?, ?, ?, ?, ?, ?, ?)
                `
				_, err := tx.Exec(payQuery,
					studentIDInt, req.DTERegisterNo, req.DTEAdmissionNo, req.ReceiptNo,
					req.ReceiptDate, parseFloat(req.Amount), req.BankName)
				if err != nil {
					return fmt.Errorf("inserting admission_payment: %v", err)
				}
			}
		}

		// UPDATE contact_details if provided
		if req.StudentEmail != "" || req.ParentMobile != "" {
			var contactExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM contact_details WHERE student_id = ?)", studentIDInt).Scan(&contactExists)
			if err != nil {
				return fmt.Errorf("checking contact_details: %v", err)
			}

			if contactExists {
				contactQuery := `
                    UPDATE contact_details SET
                        parent_mobile = ?, student_mobile = ?, student_email = ?,
                        parent_email = ?, official_email = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(contactQuery,
					req.ParentMobile, req.StudentMobile, req.StudentEmail,
					req.ParentEmail, req.OfficialEmail, studentIDInt)
				if err != nil {
					return fmt.Errorf("updating contact_details: %v", err)
				}
			} else {
				contactQuery := `
                    INSERT INTO contact_details (
                        student_id, parent_mobile, student_mobile, student_email, parent_email, official_email
                    ) VALUES (?, ?, ?, ?, ?, ?)
                `
				_, err := tx.Exec(contactQuery,
					studentIDInt, req.ParentMobile, req.StudentMobile, req.StudentEmail,
					req.ParentEmail, req.OfficialEmail)
				if err != nil {
					return fmt.Errorf("inserting contact_details: %v", err)
				}
			}
		}

		// UPDATE hostel_details if provided
		if req.HostelName != "" {
			var hostelExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM hostel_details WHERE student_id = ?)", studentIDInt).Scan(&hostelExists)
			if err != nil {
				return fmt.Errorf("checking hostel_details: %v", err)
			}

			if hostelExists {
				hostelQuery := `
                    UPDATE hostel_details SET
                        hosteller_type = ?, hostel_name = ?, room_no = ?, room_capacity = ?,
                        room_type = ?, floor_no = ?, warden_name = ?, alternate_warden = ?, class_advisor = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(hostelQuery,
					req.HostellerType, req.HostelName, req.RoomNo, parseInt(req.RoomCapacity),
					req.RoomType, parseInt(req.FloorNo), req.WardenName, req.AlternateWarden,
					req.ClassAdvisor, studentIDInt)
				if err != nil {
					return fmt.Errorf("updating hostel_details: %v", err)
				}
			} else {
				hostelQuery := `
                    INSERT INTO hostel_details (
                        student_id, hosteller_type, hostel_name, room_no, room_capacity,
                        room_type, floor_no, warden_name, alternate_warden, class_advisor, status
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
                `
				_, err := tx.Exec(hostelQuery,
					studentIDInt, req.HostellerType, req.HostelName, req.RoomNo,
					parseInt(req.RoomCapacity), req.RoomType, parseInt(req.FloorNo),
					req.WardenName, req.AlternateWarden, req.ClassAdvisor)
				if err != nil {
					return fmt.Errorf("inserting hostel_details: %v", err)
				}
			}
		}

		// UPDATE insurance_details if provided
		if req.NomineeName != "" {
			var insExists bool
			err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM insurance_details WHERE student_id = ?)", studentIDInt).Scan(&insExists)
			if err != nil {
				return fmt.Errorf("checking insurance_details: %v", err)
			}

			if insExists {
				insQuery := `
                    UPDATE insurance_details SET
                        nominee_name = ?, relationship = ?, nominee_age = ?
                    WHERE student_id = ?
                `
				_, err := tx.Exec(insQuery,
					req.NomineeName, req.Relationship, parseInt(req.NomineeAge), studentIDInt)
				if err != nil {
					return fmt.Errorf("updating insurance_details: %v", err)
				}
			} else {
				insQuery := `
                    INSERT INTO insurance_details (student_id, nominee_name, relationship, nominee_age, status)
                    VALUES (?, ?, ?, ?, 1)
                `
				_, err := tx.Exec(insQuery,
					studentIDInt, req.NomineeName, req.Relationship, parseInt(req.NomineeAge))
				if err != nil {
					return fmt.Errorf("inserting insurance_details: %v", err)
				}
			}
		}

		// Handle school_details - only update if provided
		if len(req.SchoolDetails) > 0 {
			// First, delete existing records (optional - you might want to keep them)
			_, err := tx.Exec(`DELETE FROM school_details WHERE student_id = ?`, studentIDInt)
			if err != nil {
				return fmt.Errorf("deleting school_details: %v", err)
			}

			// Insert new records
			schoolQuery := `
                INSERT INTO school_details (
                    student_id, school_name, board, year_of_pass, state, tc_no, tc_date, total_marks, status
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
            `
			for _, school := range req.SchoolDetails {
				if school.SchoolName != "" {
					_, err := tx.Exec(
						schoolQuery, studentIDInt, school.SchoolName, school.Board,
						parseInt(school.YearOfPass), school.State, school.TCNo,
						school.TCDate, parseFloat(school.TotalMarks),
					)
					if err != nil {
						return fmt.Errorf("inserting school_details for %s: %v", school.SchoolName, err)
					}
				}
			}
		}

		return nil
	}

	// Update optional fields
	if err := updateOptionalFields(); err != nil {
		log.Printf("Error updating optional fields: %v", err)

		// Return 400 instead of 404 for related field errors
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Failed to update related records",
			"details": err.Error(),
		})
		return
	}

	// Auto-enroll/re-enroll student in courses if curriculum/semester changed
	if req.CurriculumID != "" && req.Semester != "" {
		curriculumID := parseNullableInt(req.CurriculumID)
		semester := parseInt(req.Semester)

		if curriculumID != nil && semester != 0 {
			// Get distinct semester_ids for this curriculum, ordered
			var semesterIDs []int
			semesterQuery := `
				SELECT DISTINCT semester_id 
				FROM curriculum_courses 
				WHERE curriculum_id = ? 
				ORDER BY semester_id
			`
			semesterRows, err := tx.Query(semesterQuery, *curriculumID)
			if err != nil {
				log.Printf("Warning: Error fetching semester_ids: %v", err)
			} else {
				for semesterRows.Next() {
					var sid int
					if err := semesterRows.Scan(&sid); err == nil {
						semesterIDs = append(semesterIDs, sid)
					}
				}
				semesterRows.Close()
			}

			// Check if we have enough semesters and pick the correct semester_id
			if len(semesterIDs) >= semester {
				semesterID := semesterIDs[semester-1] // semester is 1-indexed

				// First, remove all existing course enrollments for this student
				_, err := tx.Exec(`DELETE FROM student_courses WHERE student_id = ?`, studentIDInt)
				if err != nil {
					log.Printf("Warning: Error removing existing enrollments for student %d: %v", studentIDInt, err)
				}

				// Get all courses for this semester_id
				coursesQuery := `
					SELECT DISTINCT course_id 
					FROM curriculum_courses 
					WHERE semester_id = ?
				`
				courseRows, err := tx.Query(coursesQuery, semesterID)
				if err != nil {
					log.Printf("Warning: Error fetching courses for auto-enrollment: %v", err)
				} else {
					// Insert courses into student_courses
					enrollQuery := `
						INSERT INTO student_courses (student_id, course_id) 
						VALUES (?, ?)
					`

					coursesEnrolled := 0
					for courseRows.Next() {
						var courseID int
						if err := courseRows.Scan(&courseID); err != nil {
							log.Printf("Warning: Error scanning course_id: %v", err)
							continue
						}

						_, err := tx.Exec(enrollQuery, studentIDInt, courseID)
						if err != nil {
							log.Printf("Warning: Error auto-enrolling course %d for student %d: %v", courseID, studentIDInt, err)
						} else {
							coursesEnrolled++
						}
					}
					courseRows.Close()

					log.Printf("Re-enrolled student %d in %d courses for semester_id %d (curriculum %d, semester %d)", studentIDInt, coursesEnrolled, semesterID, *curriculumID, semester)
				}
			} else {
				log.Printf("Warning: Semester %d not found for curriculum %d (only %d semesters available)", semester, *curriculumID, len(semesterIDs))
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}
	tx = nil // Prevent defer from rolling back

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":    "Student updated successfully",
		"student_id": fmt.Sprintf("%d", studentIDInt),
	})
}

// DeleteStudent deletes a student record (Soft Delete)
func DeleteStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	studentID := vars["id"]

	query := `UPDATE students SET status = 0 WHERE id = ?`
	result, err := db.DB.Exec(query, studentID)
	if err != nil {
		log.Printf("Error deleting student: %v", err)
		http.Error(w, "Failed to delete student", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Failed to delete student", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Student deleted successfully",
	})
}

// Helper functions to parse string values from form input
func parseString(s string) string {
	return s
}

// parseNullableString returns nil if string is empty, otherwise returns the string
// Useful for optional DATE or VARCHAR columns that should be NULL instead of empty
func parseNullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, _ := strconv.Atoi(s)
	return val
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0.0
	}
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// parseNullableInt converts string to *int, returns nil if empty or zero
func parseNullableInt(s string) *int {
	val := parseInt(s)
	if val == 0 {
		return nil
	}
	return &val
}
