package studentteacher

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"server/db"
)

const (
	electiveExemptionTypeNPTEL      = "NPTEL"
	electiveExemptionTypeInternship = "INTERNSHIP"
	maxElectiveExemptionUploadSize  = 10 << 20
)

type electiveExemptionRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      int64  `json:"id,omitempty"`
}

type studentElectiveExemptionRequestItem struct {
	ID                     int    `json:"id"`
	RequestType            string `json:"request_type"`
	ProfessionalElectiveNo int    `json:"professional_elective_no"`
	OnlineCourseName       string `json:"online_course_name"`
	CourseType             string `json:"course_type"`
	IndustryName           string `json:"industry_name"`
	IndustryContact        string `json:"industry_contact"`
	Sector                 string `json:"sector"`
	StartDate              string `json:"start_date"`
	EndDate                string `json:"end_date"`
	CertificateFilePath    string `json:"certificate_file_path"`
	CertificateURL         string `json:"certificate_url"`
	Status                 string `json:"status"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

func CreateElectiveExemptionRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseMultipartForm(maxElectiveExemptionUploadSize); err != nil {
		log.Printf("Error parsing elective exemption form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	studentEmail := strings.TrimSpace(r.FormValue("student_email"))
	requestType := strings.ToUpper(strings.TrimSpace(r.FormValue("request_type")))
	professionalElectiveValue := strings.TrimSpace(r.FormValue("professional_elective_no"))
	certificateURL := strings.TrimSpace(r.FormValue("certificate_url"))
	hasCertificateFile := false
	if r.MultipartForm != nil {
		if uploadedFiles, ok := r.MultipartForm.File["certificate_file"]; ok && len(uploadedFiles) > 0 {
			hasCertificateFile = true
		}
	}

	if studentEmail == "" {
		http.Error(w, "student_email is required", http.StatusBadRequest)
		return
	}

	if requestType != electiveExemptionTypeNPTEL && requestType != electiveExemptionTypeInternship {
		http.Error(w, "invalid request_type", http.StatusBadRequest)
		return
	}

	studentID, err := resolveStudentIDByEmail(studentEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Student not found with this email", http.StatusNotFound)
			return
		}
		log.Printf("Error resolving student by email %s: %v", studentEmail, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if certificateURL == "" && !hasCertificateFile {
		http.Error(w, "certificate proof is required", http.StatusBadRequest)
		return
	}

	formValues := electiveExemptionFormValues{
		ProfessionalNumber:  professionalElectiveValue,
		OnlineCourseName:    strings.TrimSpace(r.FormValue("online_course_name")),
		CourseType:          strings.TrimSpace(r.FormValue("course_type")),
		IndustryName:        strings.TrimSpace(r.FormValue("industry_name")),
		IndustryContact:     strings.TrimSpace(r.FormValue("industry_contact")),
		Sector:              strings.TrimSpace(r.FormValue("sector")),
		IndustryAddress:     strings.TrimSpace(r.FormValue("industry_address")),
		City:                strings.TrimSpace(r.FormValue("city")),
		State:               strings.TrimSpace(r.FormValue("state")),
		PostalCode:          strings.TrimSpace(r.FormValue("postal_code")),
		Country:             strings.TrimSpace(r.FormValue("country")),
		IndustryWebsiteURL:  strings.TrimSpace(r.FormValue("industry_website_url")),
		StartDate:           strings.TrimSpace(r.FormValue("start_date")),
		EndDate:             strings.TrimSpace(r.FormValue("end_date")),
		CourseDurationWeeks: strings.TrimSpace(r.FormValue("course_duration_weeks")),
		DaysAttended:        strings.TrimSpace(r.FormValue("number_of_days_attended")),
		StipendAmount:       strings.TrimSpace(r.FormValue("stipend_amount")),
	}

	validatedValues, err := validateElectiveExemptionForm(requestType, formValues)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	certificateFilePath, err := saveElectiveExemptionCertificate(r)
	if err != nil {
		log.Printf("Error saving elective exemption certificate: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := db.DB.Exec(`
		INSERT INTO student_elective_exemption_requests (
			student_id,
			student_email,
			request_type,
			professional_elective_no,
			status,
			online_course_name,
			course_type,
			industry_name,
			industry_contact,
			sector,
			industry_address,
			city,
			state,
			postal_code,
			country,
			industry_website_url,
			start_date,
			end_date,
			course_duration_weeks,
			number_of_days_attended,
			stipend_amount,
			certificate_file_path,
			certificate_url
		) VALUES (?, ?, ?, ?, 'submitted', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		studentID,
		studentEmail,
		requestType,
		nullableInt(validatedValues.ProfessionalNumber),
		nullableString(validatedValues.OnlineCourseName),
		nullableString(validatedValues.CourseType),
		nullableString(validatedValues.IndustryName),
		nullableString(validatedValues.IndustryContact),
		nullableString(validatedValues.Sector),
		nullableString(validatedValues.IndustryAddress),
		nullableString(validatedValues.City),
		nullableString(validatedValues.State),
		nullableString(validatedValues.PostalCode),
		nullableString(validatedValues.Country),
		nullableString(validatedValues.IndustryWebsiteURL),
		nullableDate(validatedValues.StartDate),
		nullableDate(validatedValues.EndDate),
		nullableInt(validatedValues.CourseDurationWeeks),
		nullableInt(validatedValues.NumberOfDaysAttended),
		nullableFloat(validatedValues.StipendAmount),
		certificateFilePath,
		nullableString(certificateURL),
	)
	if err != nil {
		if certificateFilePath != nil {
			_ = os.Remove("." + *certificateFilePath)
		}
		log.Printf("Error creating elective exemption request: %v", err)
		http.Error(w, "Failed to save request", http.StatusInternalServerError)
		return
	}

	requestID, _ := result.LastInsertId()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(electiveExemptionRequestResponse{
		Success: true,
		Message: "Elective exemption request submitted successfully",
		ID:      requestID,
	})
}

func GetStudentElectiveExemptionRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	studentEmail := strings.TrimSpace(r.URL.Query().Get("email"))
	if studentEmail == "" {
		http.Error(w, "email parameter required", http.StatusBadRequest)
		return
	}

	studentID, err := resolveStudentIDByEmail(studentEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Student not found with this email", http.StatusNotFound)
			return
		}
		log.Printf("Error resolving student by email %s: %v", studentEmail, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rows, err := db.DB.Query(`
		SELECT
			id,
			COALESCE(request_type, ''),
			COALESCE(professional_elective_no, 0),
			COALESCE(online_course_name, ''),
			COALESCE(course_type, ''),
			COALESCE(industry_name, ''),
			COALESCE(industry_contact, ''),
			COALESCE(sector, ''),
			COALESCE(DATE_FORMAT(start_date, '%Y-%m-%d'), ''),
			COALESCE(DATE_FORMAT(end_date, '%Y-%m-%d'), ''),
			COALESCE(certificate_file_path, ''),
			COALESCE(certificate_url, ''),
			COALESCE(status, 'submitted'),
			COALESCE(DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s'), ''),
			COALESCE(DATE_FORMAT(updated_at, '%Y-%m-%d %H:%i:%s'), '')
		FROM student_elective_exemption_requests
		WHERE student_id = ?
		ORDER BY created_at DESC
	`, studentID)
	if err != nil {
		log.Printf("Error fetching student elective exemption requests: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	requests := make([]studentElectiveExemptionRequestItem, 0)
	for rows.Next() {
		var item studentElectiveExemptionRequestItem
		if err := rows.Scan(
			&item.ID,
			&item.RequestType,
			&item.ProfessionalElectiveNo,
			&item.OnlineCourseName,
			&item.CourseType,
			&item.IndustryName,
			&item.IndustryContact,
			&item.Sector,
			&item.StartDate,
			&item.EndDate,
			&item.CertificateFilePath,
			&item.CertificateURL,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning student exemption request row: %v", err)
			continue
		}
		requests = append(requests, item)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"requests": requests,
	})
}

type electiveExemptionFormValues struct {
	ProfessionalNumber  string
	OnlineCourseName    string
	CourseType          string
	IndustryName        string
	IndustryContact     string
	Sector              string
	IndustryAddress     string
	City                string
	State               string
	PostalCode          string
	Country             string
	IndustryWebsiteURL  string
	StartDate           string
	EndDate             string
	CourseDurationWeeks string
	DaysAttended        string
	StipendAmount       string
}

type validatedElectiveExemptionValues struct {
	ProfessionalNumber   *int
	OnlineCourseName     string
	CourseType           string
	IndustryName         string
	IndustryContact      string
	Sector               string
	IndustryAddress      string
	City                 string
	State                string
	PostalCode           string
	Country              string
	IndustryWebsiteURL   string
	StartDate            *time.Time
	EndDate              *time.Time
	CourseDurationWeeks  *int
	NumberOfDaysAttended *int
	StipendAmount        *float64
}

func validateElectiveExemptionForm(requestType string, values electiveExemptionFormValues) (validatedElectiveExemptionValues, error) {
	startDate, err := parseElectiveExemptionDate(values.StartDate, "start_date")
	if err != nil {
		return validatedElectiveExemptionValues{}, err
	}

	endDate, err := parseElectiveExemptionDate(values.EndDate, "end_date")
	if err != nil {
		return validatedElectiveExemptionValues{}, err
	}

	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return validatedElectiveExemptionValues{}, fmt.Errorf("end_date must be on or after start_date")
	}

	validated := validatedElectiveExemptionValues{
		OnlineCourseName:   values.OnlineCourseName,
		CourseType:         values.CourseType,
		IndustryName:       values.IndustryName,
		IndustryContact:    values.IndustryContact,
		Sector:             values.Sector,
		IndustryAddress:    values.IndustryAddress,
		City:               values.City,
		State:              values.State,
		PostalCode:         values.PostalCode,
		Country:            values.Country,
		IndustryWebsiteURL: values.IndustryWebsiteURL,
		StartDate:          startDate,
		EndDate:            endDate,
	}

	switch requestType {
	case electiveExemptionTypeNPTEL:
		professionalNumber, err := parseRangedIntField(values.ProfessionalNumber, "professional_elective_no", 1, 9)
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		validated.ProfessionalNumber = professionalNumber

		if values.OnlineCourseName == "" || values.CourseType == "" || startDate == nil || endDate == nil {
			return validatedElectiveExemptionValues{}, fmt.Errorf("online course name, course type, start date, and end date are required")
		}

		courseDurationWeeks, err := parsePositiveIntField(values.CourseDurationWeeks, "course_duration_weeks")
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		validated.CourseDurationWeeks = courseDurationWeeks

	case electiveExemptionTypeInternship:
		professionalNumber, err := parseRangedIntField(values.ProfessionalNumber, "professional_elective_no", 1, 9)
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		validated.ProfessionalNumber = professionalNumber

		if values.IndustryName == "" || values.IndustryContact == "" || values.Sector == "" || values.IndustryAddress == "" || values.City == "" || values.State == "" || values.PostalCode == "" || values.Country == "" || values.IndustryWebsiteURL == "" || startDate == nil || endDate == nil {
			return validatedElectiveExemptionValues{}, fmt.Errorf("industry name, industry contact, sector, address, city, state, postal code, country, website, start date, and end date are required")
		}

		if !isValidInternshipSector(values.Sector) {
			return validatedElectiveExemptionValues{}, fmt.Errorf("sector must be Government or Private")
		}

		daysAttended, err := parsePositiveIntField(values.DaysAttended, "number_of_days_attended")
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		stipendAmount, err := parseNonNegativeFloatField(values.StipendAmount, "stipend_amount")
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		validated.NumberOfDaysAttended = daysAttended
		validated.StipendAmount = stipendAmount

	default:
		return validatedElectiveExemptionValues{}, fmt.Errorf("unsupported request type")
	}

	return validated, nil
}

func parseElectiveExemptionDate(value, fieldName string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s", fieldName)
	}

	return &parsed, nil
}

func isValidInternshipSector(value string) bool {
	sector := strings.TrimSpace(strings.ToLower(value))
	return sector == "government" || sector == "private"
}

func parsePositiveIntField(value, fieldName string) (*int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is required", fieldName)
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("%s must be a positive number", fieldName)
	}

	return &parsed, nil
}

func parseRangedIntField(value, fieldName string, minValue, maxValue int) (*int, error) {
	parsed, err := parsePositiveIntField(value, fieldName)
	if err != nil {
		return nil, err
	}
	if *parsed < minValue || *parsed > maxValue {
		return nil, fmt.Errorf("%s must be between %d and %d", fieldName, minValue, maxValue)
	}
	return parsed, nil
}

func parseNonNegativeFloatField(value, fieldName string) (*float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is required", fieldName)
	}

	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || parsed < 0 {
		return nil, fmt.Errorf("%s must be zero or greater", fieldName)
	}

	return &parsed, nil
}

func saveElectiveExemptionCertificate(r *http.Request) (*string, error) {
	file, header, err := r.FormFile("certificate_file")
	if err == http.ErrMissingFile {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve certificate file")
	}
	defer file.Close()

	uploadDir := "./uploads/elective-exemptions"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".bin"
	}

	fileName := fmt.Sprintf("elective_exemption_%d%s", time.Now().UnixNano(), ext)
	fullPath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save certificate file")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("failed to save certificate file")
	}

	relativePath := "/uploads/elective-exemptions/" + fileName
	return &relativePath, nil
}

<<<<<<< HEAD
func resolveStudentIDByEmail(email string) (int, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" {
		return 0, sql.ErrNoRows
	}

	var studentID int
	err := db.DB.QueryRow(`
		SELECT student_id
		FROM contact_details
		WHERE LOWER(TRIM(student_email)) = ?
		   OR LOWER(TRIM(official_email)) = ?
		LIMIT 1
	`, normalizedEmail, normalizedEmail).Scan(&studentID)
	if err != nil {
		return 0, err
	}

	return studentID, nil
}

=======
>>>>>>> 30f43088941d7e5d6f462a7b0f960f278a66a9d8
func nullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableDate(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}
