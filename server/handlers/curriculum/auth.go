package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	user, err := getActiveUserByUsername(loginReq.Username)
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

	if user.Password != loginReq.Password {
		log.Printf("Password verification failed")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	writeLoginSuccessResponse(w, user)
}

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
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

	var req models.GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	idToken := strings.TrimSpace(req.IDToken)
	if idToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "id_token is required",
		})
		return
	}

	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if googleClientID == "" {
		log.Println("GOOGLE_CLIENT_ID is not configured on server")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Google sign-in is not configured",
		})
		return
	}

	email, err := verifyGoogleIDToken(idToken, googleClientID)
	if err != nil {
		log.Printf("Invalid Google ID token: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Invalid Google token",
		})
		return
	}

	user, err := getActiveUserByEmail(email)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Your account is not registered in this system",
		})
		return
	}
	if err != nil {
		log.Printf("Error querying user by email: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.LoginResponse{
			Success: false,
			Message: "Internal server error",
		})
		return
	}

	writeLoginSuccessResponse(w, user)
}

func getActiveUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, password, email, role, is_active, created_at, updated_at, last_login
	          FROM users WHERE username = ? AND is_active = TRUE`

	err := db.DB.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getActiveUserByEmail(email string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, password, email, role, is_active, created_at, updated_at, last_login
	          FROM users WHERE email = ? AND is_active = TRUE LIMIT 1`
	err := db.DB.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func writeLoginSuccessResponse(w http.ResponseWriter, user *models.User) {
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
		User:    user,
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

type googleTokenInfoResponse struct {
	Aud              string `json:"aud"`
	Email            string `json:"email"`
	EmailVerified    string `json:"email_verified"`
	Iss              string `json:"iss"`
	Exp              string `json:"exp"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func verifyGoogleIDToken(idToken string, expectedAudience string) (string, error) {
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var tokenInfo googleTokenInfoResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&tokenInfo); decodeErr != nil {
		return "", decodeErr
	}

	if response.StatusCode != http.StatusOK {
		if strings.TrimSpace(tokenInfo.ErrorDescription) != "" {
			return "", fmt.Errorf(tokenInfo.ErrorDescription)
		}
		if strings.TrimSpace(tokenInfo.Error) != "" {
			return "", fmt.Errorf(tokenInfo.Error)
		}
		return "", fmt.Errorf("token validation failed with status %d", response.StatusCode)
	}

	if strings.TrimSpace(tokenInfo.Aud) != strings.TrimSpace(expectedAudience) {
		return "", fmt.Errorf("token audience mismatch")
	}

	issuer := strings.TrimSpace(tokenInfo.Iss)
	if issuer != "" && issuer != "accounts.google.com" && issuer != "https://accounts.google.com" {
		return "", fmt.Errorf("invalid token issuer")
	}

	email := strings.TrimSpace(tokenInfo.Email)
	if email == "" {
		return "", fmt.Errorf("token email missing")
	}

	if !strings.EqualFold(strings.TrimSpace(tokenInfo.EmailVerified), "true") {
		return "", fmt.Errorf("token email is not verified")
	}

	if expRaw := strings.TrimSpace(tokenInfo.Exp); expRaw != "" {
		expUnix, parseErr := strconv.ParseInt(expRaw, 10, 64)
		if parseErr == nil && time.Now().Unix() >= expUnix {
			return "", fmt.Errorf("token is expired")
		}
	}

	return strings.ToLower(email), nil
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
