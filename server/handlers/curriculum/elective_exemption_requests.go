package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"server/db"

	"github.com/gorilla/mux"
)

type hodElectiveExemptionRequest struct {
	ID                     int    `json:"id"`
	StudentID              int    `json:"student_id"`
	StudentName            string `json:"student_name"`
	EnrollmentNo           string `json:"enrollment_no"`
	RegisterNo             string `json:"register_no"`
	DepartmentName         string `json:"department_name"`
	StudentEmail           string `json:"student_email"`
	RequestType            string `json:"request_type"`
	ProfessionalElectiveNo int    `json:"professional_elective_no"`
	OnlineCourseName       string `json:"online_course_name"`
	CourseType             string `json:"course_type"`
	IndustryName           string `json:"industry_name"`
	IndustryContact        string `json:"industry_contact"`
	IndustryAddress        string `json:"industry_address"`
	City                   string `json:"city"`
	State                  string `json:"state"`
	PostalCode             string `json:"postal_code"`
	Country                string `json:"country"`
	Sector                 string `json:"sector"`
	IndustryWebsiteURL     string `json:"industry_website_url"`
	NumberOfDaysAttended   string `json:"number_of_days_attended"`
	StipendAmount          string `json:"stipend_amount"`
	CertificateFilePath    string `json:"certificate_file_path"`
	CertificateURL         string `json:"certificate_url"`
	StartDate              string `json:"start_date"`
	EndDate                string `json:"end_date"`
	Status                 string `json:"status"`
	CreatedAt              string `json:"created_at"`
}

func GetHODElectiveExemptionRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	email := strings.TrimSpace(r.URL.Query().Get("email"))
	if email == "" {
		http.Error(w, "email parameter required", http.StatusBadRequest)
		return
	}

	departmentID, err := resolveHODDepartmentID(email)
	if err != nil {
		writeHODExemptionError(w, err)
		return
	}

	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	allowedStatus := map[string]bool{"submitted": true, "approved": true, "rejected": true}

	query := `
		SELECT
			seer.id,
			seer.student_id,
			COALESCE(s.student_name, ''),
			COALESCE(s.enrollment_no, ''),
			COALESCE(s.register_no, ''),
			COALESCE(d.department_name, ''),
			COALESCE(seer.student_email, ''),
			COALESCE(seer.request_type, ''),
			COALESCE(seer.professional_elective_no, 0),
			COALESCE(seer.online_course_name, ''),
			COALESCE(seer.course_type, ''),
			COALESCE(seer.industry_name, ''),
			COALESCE(seer.industry_contact, ''),
			COALESCE(seer.industry_address, ''),
			COALESCE(seer.city, ''),
			COALESCE(seer.state, ''),
			COALESCE(seer.postal_code, ''),
			COALESCE(seer.country, ''),
			COALESCE(seer.sector, ''),
			COALESCE(seer.industry_website_url, ''),
			COALESCE(CAST(seer.number_of_days_attended AS CHAR), ''),
			COALESCE(CAST(seer.stipend_amount AS CHAR), ''),
			COALESCE(seer.certificate_file_path, ''),
			COALESCE(seer.certificate_url, ''),
			COALESCE(DATE_FORMAT(seer.start_date, '%Y-%m-%d'), ''),
			COALESCE(DATE_FORMAT(seer.end_date, '%Y-%m-%d'), ''),
			COALESCE(seer.status, 'submitted'),
			COALESCE(DATE_FORMAT(seer.created_at, '%Y-%m-%d %H:%i:%s'), '')
		FROM student_elective_exemption_requests seer
		INNER JOIN students s ON seer.student_id = s.id
		LEFT JOIN departments d ON s.department_id = d.id
		WHERE s.department_id = ?
	`

	args := []interface{}{departmentID}
	if allowedStatus[statusFilter] {
		query += ` AND LOWER(seer.status) = ?`
		args = append(args, statusFilter)
	}
	query += ` ORDER BY FIELD(LOWER(seer.status), 'submitted', 'approved', 'rejected'), seer.created_at DESC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching HOD elective exemption requests: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	requests := make([]hodElectiveExemptionRequest, 0)
	for rows.Next() {
		var item hodElectiveExemptionRequest
		if err := rows.Scan(
			&item.ID,
			&item.StudentID,
			&item.StudentName,
			&item.EnrollmentNo,
			&item.RegisterNo,
			&item.DepartmentName,
			&item.StudentEmail,
			&item.RequestType,
			&item.ProfessionalElectiveNo,
			&item.OnlineCourseName,
			&item.CourseType,
			&item.IndustryName,
			&item.IndustryContact,
			&item.IndustryAddress,
			&item.City,
			&item.State,
			&item.PostalCode,
			&item.Country,
			&item.Sector,
			&item.IndustryWebsiteURL,
			&item.NumberOfDaysAttended,
			&item.StipendAmount,
			&item.CertificateFilePath,
			&item.CertificateURL,
			&item.StartDate,
			&item.EndDate,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			log.Printf("Error scanning HOD exemption request row: %v", err)
			continue
		}
		requests = append(requests, item)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"requests": requests,
	})
}

func UpdateHODElectiveExemptionRequestStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	requestID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || requestID <= 0 {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}

	var body struct {
		Email  string `json:"email"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	departmentID, err := resolveHODDepartmentID(strings.TrimSpace(body.Email))
	if err != nil {
		writeHODExemptionError(w, err)
		return
	}

	status := strings.ToLower(strings.TrimSpace(body.Status))
	if status != "approved" && status != "rejected" {
		http.Error(w, "status must be approved or rejected", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("Error starting HOD exemption transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var studentID, requestDeptID, professionalElectiveNo int
	var currentStatus string
	err = tx.QueryRow(`
		SELECT seer.student_id, s.department_id, COALESCE(seer.professional_elective_no, 0), COALESCE(LOWER(seer.status), 'submitted')
		FROM student_elective_exemption_requests seer
		INNER JOIN students s ON seer.student_id = s.id
		WHERE seer.id = ?
		LIMIT 1
	`, requestID).Scan(&studentID, &requestDeptID, &professionalElectiveNo, &currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "request not found", http.StatusNotFound)
			return
		}
		log.Printf("Error fetching exemption request %d: %v", requestID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if requestDeptID != departmentID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if currentStatus != "submitted" {
		http.Error(w, "request already processed", http.StatusConflict)
		return
	}

	removedChoices := 0
	if status == "approved" {
		removedChoices, err = removeStudentChoiceForProfessionalElective(tx, studentID, professionalElectiveNo)
		if err != nil {
			log.Printf("Error removing approved student PE choice: %v", err)
			http.Error(w, "Failed to update student elective choice", http.StatusInternalServerError)
			return
		}
	}

	if _, err := tx.Exec(`UPDATE student_elective_exemption_requests SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, requestID); err != nil {
		log.Printf("Error updating exemption request status: %v", err)
		http.Error(w, "Failed to update request status", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing HOD exemption status update: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"message":         fmt.Sprintf("Request %s successfully", status),
		"removed_choices": removedChoices,
	})
}

func resolveHODDepartmentID(email string) (int, error) {
	if strings.TrimSpace(email) == "" {
		return 0, fmt.Errorf("email parameter required")
	}

	var departmentID int
	err := db.DB.QueryRow(`
		SELECT d.id
		FROM users u
		INNER JOIN teachers t ON u.email = t.email
		INNER JOIN department_teachers dt ON t.faculty_id = dt.teacher_id
		INNER JOIN departments d ON dt.department_id = d.id
		WHERE u.email = ? AND u.role = 'hod'
		ORDER BY d.id
		LIMIT 1
	`, email).Scan(&departmentID)
	if err != nil {
		return 0, err
	}

	return departmentID, nil
}

func removeStudentChoiceForProfessionalElective(tx *sql.Tx, studentID, professionalElectiveNo int) (int, error) {
	if professionalElectiveNo <= 0 {
		return 0, nil
	}

	normalizedShort := fmt.Sprintf("pe%d", professionalElectiveNo)
	normalizedLong := fmt.Sprintf("professionalelective%d", professionalElectiveNo)

	rows, err := tx.Query(`
		SELECT sec.id, hes.course_id
		FROM student_elective_choices sec
		INNER JOIN hod_elective_selections hes ON sec.hod_selection_id = hes.id
		INNER JOIN elective_semester_slots ess ON hes.slot_id = ess.id
		WHERE sec.student_id = ?
		  AND REPLACE(REPLACE(REPLACE(LOWER(ess.slot_name), ' ', ''), '-', ''), '_', '') IN (?, ?)
	`, studentID, normalizedShort, normalizedLong)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	choiceIDs := make([]int, 0)
	courseIDs := make([]int, 0)
	for rows.Next() {
		var choiceID, courseID int
		if err := rows.Scan(&choiceID, &courseID); err != nil {
			return 0, err
		}
		choiceIDs = append(choiceIDs, choiceID)
		courseIDs = append(courseIDs, courseID)
	}

	removed := 0
	for index, choiceID := range choiceIDs {
		if _, err := tx.Exec(`DELETE FROM student_elective_choices WHERE id = ?`, choiceID); err != nil {
			return removed, err
		}
		if _, err := tx.Exec(`DELETE FROM student_courses WHERE student_id = ? AND course_id = ?`, studentID, courseIDs[index]); err != nil {
			return removed, err
		}
		removed++
	}

	return removed, nil
}

func writeHODExemptionError(w http.ResponseWriter, err error) {
	if err == sql.ErrNoRows {
		http.Error(w, "hod department not found", http.StatusNotFound)
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "email") {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("HOD exemption handler error: %v", err)
	http.Error(w, "Database error", http.StatusInternalServerError)
}
