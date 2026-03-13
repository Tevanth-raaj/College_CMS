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
	electiveSemesterValue := strings.TrimSpace(r.FormValue("elective_semester_no"))
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

	electiveSemesterNo, err := strconv.Atoi(electiveSemesterValue)
	if err != nil {
		http.Error(w, "invalid elective_semester_no", http.StatusBadRequest)
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
		OnlineCourseName:    strings.TrimSpace(r.FormValue("online_course_name")),
		CourseType:          strings.TrimSpace(r.FormValue("course_type")),
		IndustryName:        strings.TrimSpace(r.FormValue("industry_name")),
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
			elective_semester_no,
			status,
			online_course_name,
			course_type,
			industry_name,
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
		) VALUES (?, ?, ?, ?, 'submitted', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		studentID,
		studentEmail,
		requestType,
		electiveSemesterNo,
		nullableString(validatedValues.OnlineCourseName),
		nullableString(validatedValues.CourseType),
		nullableString(validatedValues.IndustryName),
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

type electiveExemptionFormValues struct {
	OnlineCourseName    string
	CourseType          string
	IndustryName        string
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
	OnlineCourseName     string
	CourseType           string
	IndustryName         string
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
		if values.OnlineCourseName == "" || values.CourseType == "" || startDate == nil || endDate == nil {
			return validatedElectiveExemptionValues{}, fmt.Errorf("online course name, course type, start date, and end date are required")
		}

		courseDurationWeeks, err := parsePositiveIntField(values.CourseDurationWeeks, "course_duration_weeks")
		if err != nil {
			return validatedElectiveExemptionValues{}, err
		}
		validated.CourseDurationWeeks = courseDurationWeeks

	case electiveExemptionTypeInternship:
		if values.IndustryName == "" || values.Sector == "" || values.IndustryAddress == "" || values.City == "" || values.State == "" || values.PostalCode == "" || values.Country == "" || values.IndustryWebsiteURL == "" || startDate == nil || endDate == nil {
			return validatedElectiveExemptionValues{}, fmt.Errorf("industry name, sector, address, city, state, postal code, country, website, start date, and end date are required")
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

func resolveStudentIDByEmail(email string) (int, error) {
	var studentID int
	err := db.DB.QueryRow(`
		SELECT student_id
		FROM contact_details
		WHERE student_email = ?
	`, email).Scan(&studentID)
	if err != nil {
		return 0, err
	}

	return studentID, nil
}

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
