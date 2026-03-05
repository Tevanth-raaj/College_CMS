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
	"server/db"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Teacher represents the teacher model
type Teacher struct {
	ID         int64     `json:"id"`
	FacultyID  string    `json:"faculty_id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      *string   `json:"phone"`
	ProfileImg *string   `json:"profile_img"`
	Dept       *string   `json:"dept"` // VARCHAR in database
	Department *string   `json:"department"` // For display purposes
	Desg       *string   `json:"designation"`
	LastLogin  time.Time `json:"last_login"`
	Status     int       `json:"status"` // 1 = active, 0 = deleted
}

// TeacherInput represents the input for creating/updating a teacher
type TeacherInput struct {
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Phone      *string `json:"phone"`
	ProfileImg *string `json:"profile_img"`
	Department string  `json:"department"` // Department name from frontend
	Desg       *string `json:"designation"`
}

// GetTeachers retrieves all teachers from the database
func GetTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	query := `
		SELECT 
			t.id, t.faculty_id, t.name, t.email, t.phone, t.profile_img, 
			t.dept, d.department_name as department, t.desg, 
			t.last_login, t.status
		FROM teachers t
		LEFT JOIN departments d ON t.dept = d.id
		WHERE t.status = 1
		ORDER BY t.id DESC
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Printf("Error querying teachers: %v", err)
		http.Error(w, "Failed to fetch teachers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var teachers []Teacher
	for rows.Next() {
		var teacher Teacher
		err := rows.Scan(
			&teacher.ID, &teacher.FacultyID, &teacher.Name, &teacher.Email, &teacher.Phone,
			&teacher.ProfileImg, &teacher.Dept, &teacher.Department, &teacher.Desg,
			&teacher.LastLogin, &teacher.Status,
		)
		if err != nil {
			log.Printf("Error scanning teacher row: %v", err)
			continue
		}
		teachers = append(teachers, teacher)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating teacher rows: %v", err)
		http.Error(w, "Failed to process teachers", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(teachers)
}

// GetTeacherByID retrieves a single teacher by ID
func GetTeacherByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			t.id, t.faculty_id, t.name, t.email, t.phone, t.profile_img, 
			t.dept, d.department_name as department, t.desg, 
			t.last_login, t.status
		FROM teachers t
		LEFT JOIN departments d ON t.dept = d.id
		WHERE t.id = ? AND t.status = 1
	`

	var teacher Teacher
	err = db.DB.QueryRow(query, id).Scan(
		&teacher.ID, &teacher.FacultyID, &teacher.Name, &teacher.Email, &teacher.Phone,
		&teacher.ProfileImg, &teacher.Dept, &teacher.Department, &teacher.Desg,
		&teacher.LastLogin, &teacher.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying teacher: %v", err)
		http.Error(w, "Failed to fetch teacher", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(teacher)
}

// GetTeacherByEmail retrieves a single teacher by email
func GetTeacherByEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle OPTIONS request for CORS
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		log.Printf("GetTeacherByEmail called without email parameter")
		http.Error(w, "Email parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Fetching teacher by email: %s", email)

	query := `
		SELECT 
			t.id, t.faculty_id, t.name, t.email, t.phone, t.profile_img, 
			t.dept, d.department_name as department, t.desg, 
			t.last_login, t.status
		FROM teachers t
		LEFT JOIN departments d ON t.dept = d.id
		WHERE t.email = ? AND t.status = 1
	`

	var teacher Teacher

	err := db.DB.QueryRow(query, email).Scan(
		&teacher.ID, &teacher.FacultyID, &teacher.Name, &teacher.Email, &teacher.Phone,
		&teacher.ProfileImg, &teacher.Dept, &teacher.Department, &teacher.Desg,
		&teacher.LastLogin, &teacher.Status,
	)

	if err == sql.ErrNoRows {
		log.Printf("Teacher not found with email: %s", email)
		http.Error(w, fmt.Sprintf("Teacher not found with email: %s", email), http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error querying teacher by email: %v", err)
		http.Error(w, "Failed to fetch teacher", http.StatusInternalServerError)
		return
	}

	log.Printf("Teacher found: ID=%d, Name=%s, FacultyID=%s", teacher.ID, teacher.Name, teacher.FacultyID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"teacher": teacher,
	})
}

// CreateTeacher creates a new teacher
func CreateTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle OPTIONS request for CORS
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse multipart form data (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form fields
	name := r.FormValue("name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")
	department := r.FormValue("department")
	designation := r.FormValue("designation")

	// Validate required fields
	if name == "" || email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Handle file upload
	var profileImgPath *string
	file, header, err := r.FormFile("profile_img")
	if err == nil {
		defer file.Close()

		// Create uploads directory if it doesn't exist
		uploadDir := "./uploads/teachers"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			log.Printf("Error creating upload directory: %v", err)
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Generate unique filename
		ext := filepath.Ext(header.Filename)
		filename := fmt.Sprintf("teacher_%d%s", time.Now().Unix(), ext)
		filepath := filepath.Join(uploadDir, filename)

		// Create the file
		dst, err := os.Create(filepath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination
		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Error copying file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Store the relative path to save in database
		relativePath := "/uploads/teachers/" + filename
		profileImgPath = &relativePath
		log.Printf("File uploaded successfully: %s", relativePath)
	} else if err != http.ErrMissingFile {
		log.Printf("Error retrieving file: %v", err)
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		return
	}

	// Get department ID (frontend now sends department ID)
	var deptID *string
	if department != "" {
		// Store as string since dept column is VARCHAR
		deptID = &department
		log.Printf("Using department value: %s", department)
	}

	// Prepare optional fields
	var phonePtr *string
	if phone != "" {
		phonePtr = &phone
	}
	var desgPtr *string
	if designation != "" {
		desgPtr = &designation
	}

	// Insert teacher with status = 1 (active) by default
	query := `
		INSERT INTO teachers (name, email, phone, profile_img, dept, desg, status)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`

	result, err := db.DB.Exec(
		query,
		name,
		email,
		phonePtr,
		profileImgPath,
		deptID,
		desgPtr,
	)

	if err != nil {
		log.Printf("Error creating teacher: %v", err)
		if strings.Contains(err.Error(), "Duplicate entry") {
			http.Error(w, "Teacher with this email already exists", http.StatusConflict)
		} else {
			http.Error(w, "Failed to create teacher", http.StatusInternalServerError)
		}
		return
	}

	teacherID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting last insert ID: %v", err)
		http.Error(w, "Failed to get teacher ID", http.StatusInternalServerError)
		return
	}

	// Note: dept is stored as VARCHAR directly in teachers table
	// department_teachers junction table is managed separately

	// Fetch and return the created teacher
	createdTeacher := Teacher{
		ID:         teacherID,
		Name:       name,
		Email:      email,
		Phone:      phonePtr,
		ProfileImg: profileImgPath,
		Dept:       deptID,
		Department: &department,
		Desg:       desgPtr,
		Status:     1,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdTeacher)
}

// UpdateTeacher updates an existing teacher
func UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle OPTIONS request for CORS
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	// Parse multipart form data (max 10MB)
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form fields
	name := r.FormValue("name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")
	department := r.FormValue("department")
	designation := r.FormValue("designation")

	// Validate required fields
	if name == "" || email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Get existing teacher to check current profile_img
	var existingProfileImg *string
	err = db.DB.QueryRow("SELECT profile_img FROM teachers WHERE id = ? AND status = 1", id).Scan(&existingProfileImg)
	if err == sql.ErrNoRows {
		log.Printf("Teacher with ID %d not found or inactive", id)
		http.Error(w, fmt.Sprintf("Teacher with ID %d not found or has been deleted", id), http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error fetching existing teacher: %v", err)
		http.Error(w, "Failed to fetch teacher", http.StatusInternalServerError)
		return
	}

	// Handle file upload
	profileImgPath := existingProfileImg
	file, header, err := r.FormFile("profile_img")
	if err == nil {
		defer file.Close()

		// Delete old profile image if exists
		if existingProfileImg != nil && *existingProfileImg != "" {
			oldFilePath := "." + *existingProfileImg
			if err := os.Remove(oldFilePath); err != nil {
				log.Printf("Warning: Failed to delete old profile image: %v", err)
			}
		}

		// Create uploads directory if it doesn't exist
		uploadDir := "./uploads/teachers"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			log.Printf("Error creating upload directory: %v", err)
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}

		// Generate unique filename
		ext := filepath.Ext(header.Filename)
		filename := fmt.Sprintf("teacher_%d%s", time.Now().Unix(), ext)
		filepath := filepath.Join(uploadDir, filename)

		// Create the file
		dst, err := os.Create(filepath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy the uploaded file to the destination
		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Error copying file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// Store the relative path to save in database
		relativePath := "/uploads/teachers/" + filename
		profileImgPath = &relativePath
		log.Printf("File uploaded successfully: %s", relativePath)
	} else if err != http.ErrMissingFile {
		log.Printf("Error retrieving file: %v", err)
		http.Error(w, "Failed to retrieve file", http.StatusBadRequest)
		return
	}

	// Get department ID (frontend now sends department ID)
	var deptID *string
	if department != "" {
		// Store as string since dept column is VARCHAR
		deptID = &department
		log.Printf("Using department value: %s", department)
	}

	// Prepare optional fields
	var phonePtr *string
	if phone != "" {
		phonePtr = &phone
	}
	var desgPtr *string
	if designation != "" {
		desgPtr = &designation
	}

	// Update teacher
	query := `
		UPDATE teachers 
		SET name = ?, email = ?, phone = ?, profile_img = ?, dept = ?, desg = ?
		WHERE id = ? AND status = 1
	`

	result, err := db.DB.Exec(
		query,
		name,
		email,
		phonePtr,
		profileImgPath,
		deptID,
		desgPtr,
		id,
	)

	if err != nil {
		log.Printf("Error updating teacher: %v", err)
		http.Error(w, "Failed to update teacher", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Failed to update teacher", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}

	// Note: dept is stored as VARCHAR directly in teachers table
	// department_teachers junction table is managed separately

	// Fetch and return the updated teacher
	updatedTeacher := Teacher{
		ID:         id,
		Name:       name,
		Email:      email,
		Phone:      phonePtr,
		ProfileImg: profileImgPath,
		Dept:       deptID,
		Department: &department,
		Desg:       desgPtr,
		Status:     1,
	}

	json.NewEncoder(w).Encode(updatedTeacher)
}

// DeleteTeacher soft deletes a teacher by setting status to 0
func DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid teacher ID", http.StatusBadRequest)
		return
	}

	// Soft delete: set status to 0
	query := "UPDATE teachers SET status = 0 WHERE id = ? AND status = 1"
	result, err := db.DB.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting teacher: %v", err)
		http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		http.Error(w, "Failed to delete teacher", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Teacher not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Teacher deleted successfully"})
}
