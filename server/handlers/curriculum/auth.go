package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"server/db"
	"server/models"
)

// Login handles user authentication
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	var loginReq models.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if err != nil {
		log.Println("Error decoding login request:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	log.Printf("Login attempt for username: %s", loginReq.Username)

	// Query user from database
	var user models.User
	query := `SELECT id, username, password, email, role, is_active, created_at, updated_at, last_login 
	          FROM users WHERE username = ? AND is_active = TRUE`

	err = db.DB.QueryRow(query, loginReq.Username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err == sql.ErrNoRows {
		log.Printf("User not found or inactive: %s", loginReq.Username)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	} else if err != nil {
		log.Println("Error querying user:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Internal server error",
		})
		return
	}

	log.Printf("User found: %s, verifying password", user.Username)
	log.Printf("Password from DB: %s", user.Password)
	log.Printf("Password provided: %s", loginReq.Password)

	// Verify password
	if user.Password != loginReq.Password {
		log.Printf("Password verification failed")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	log.Printf("Login successful for user: %s", user.Username)

	// Update last login time
	_, _ = db.DB.Exec("UPDATE users SET last_login = ? WHERE id = ?", time.Now(), user.ID)

	// If user is a teacher or hod, fetch teacher details from teachers table
	var teacherData map[string]interface{}
	if user.Role == "teacher" || user.Role == "hod" {
		log.Printf("Fetching teacher details for role %s, email: %s", user.Role, user.Email)
		
		var teacherID int64
		var facultyID, name, email, phone, dept, desg sql.NullString
		var profileImg sql.NullString
		var status sql.NullInt32
		
		// Query without theory_subject_count columns (they may not exist in all schemas)
		teacherQuery := `SELECT id, faculty_id, name, email, phone, profile_img, dept, desg, status
		                 FROM teachers WHERE email = ? AND status = 1`
		
		err := db.DB.QueryRow(teacherQuery, user.Email).Scan(
			&teacherID, &facultyID, &name, &email, &phone, &profileImg, 
			&dept, &desg, &status,
		)
		
		if err == nil {
			// Teacher found, add teacher data to response
			teacherData = map[string]interface{}{
				"teacher_id":  teacherID,
				"faculty_id":  nullStringToString(facultyID),
				"name":        nullStringToString(name),
				"email":       nullStringToString(email),
				"phone":       nullStringToString(phone),
				"profile_img": nullStringToString(profileImg),
				"dept":        nullStringToString(dept),
				"designation": nullStringToString(desg),
				"status":      nullIntToInt(status),
			}
			log.Printf("Teacher data found: ID=%d, Name=%s, FacultyID=%s", teacherID, nullStringToString(name), nullStringToString(facultyID))
		} else if err == sql.ErrNoRows {
			log.Printf("Warning: User is teacher role but not found in teachers table: %s", user.Email)
		} else {
			log.Printf("Error fetching teacher data: %v", err)
		}
	}

	// Return success response
	response := models.LoginResponse{
		Success: true,
		Message: "Login successful",
		User:    &user,
		Token:   "dummy-token", // In production, generate a proper JWT token
	}
	
	// Add teacher data if available
	if teacherData != nil {
		responseMap := map[string]interface{}{
			"success":      response.Success,
			"message":      response.Message,
			"user":         response.User,
			"token":        response.Token,
			"teacher_data": teacherData,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responseMap)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper functions to handle NULL values
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullIntToInt(ni sql.NullInt32) int {
	if ni.Valid {
		return int(ni.Int32)
	}
	return 0
}
