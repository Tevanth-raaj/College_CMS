package curriculum

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"server/db"
	"strconv"
	"strings"
)

// DownloadHonourMinorEligibilityTemplate serves a CSV template for student_eligible_honour_minor import.
// type query parameter: 'honour' or 'minor'
func DownloadHonourMinorEligibilityTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	eligibilityType := strings.ToUpper(r.URL.Query().Get("type"))
	if eligibilityType == "" {
		eligibilityType = "HONOUR"
	}

	// Validate type
	if eligibilityType != "HONOUR" && eligibilityType != "MINOR" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid type. Use 'HONOUR' or 'MINOR'",
		})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	filename := "student_eligible_" + strings.ToLower(eligibilityType) + "_template.csv"
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"student_email"})
	_ = writer.Write([]string{"student1@example.com"})
	_ = writer.Write([]string{"student2@example.com"})
	writer.Flush()
}

// DownloadHonourTemplate serves a CSV template specifically for honour eligibility
func DownloadHonourTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=student_eligible_honour_template.csv")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"student_email"})
	_ = writer.Write([]string{"student1@example.com"})
	_ = writer.Write([]string{"student2@example.com"})
	writer.Flush()
}

// DownloadMinorTemplate serves a CSV template specifically for minor eligibility
func DownloadMinorTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=student_eligible_minor_template.csv")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"student_email"})
	_ = writer.Write([]string{"student1@example.com"})
	_ = writer.Write([]string{"student2@example.com"})
	writer.Flush()
}

// ImportHonourMinorEligibility imports CSV data into student_eligible_honour_minor.
// Expected CSV header: student_email
// type form field: 'HONOUR' or 'MINOR'
func ImportHonourMinorEligibility(w http.ResponseWriter, r *http.Request) {
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid form data",
		})
		return
	}

	eligibilityType := strings.ToUpper(r.FormValue("type"))
	if eligibilityType == "" {
		eligibilityType = "HONOUR"
	}

	// Validate type
	if eligibilityType != "HONOUR" && eligibilityType != "MINOR" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid type. Use 'HONOUR' or 'MINOR'",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "CSV file is required (field name: file)",
		})
		return
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to read CSV header",
		})
		return
	}

	emailCol := -1
	for idx, col := range header {
		if strings.EqualFold(strings.TrimSpace(col), "student_email") {
			emailCol = idx
			break
		}
	}
	if emailCol == -1 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "CSV must contain 'student_email' column",
		})
		return
	}

	inserted := 0
	skipped := 0
	errors := []string{}
	lineNo := 1

	for {
		lineNo++
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": invalid CSV row")
			continue
		}

		if emailCol >= len(record) {
			skipped++
			continue
		}

		email := strings.TrimSpace(strings.ToLower(record[emailCol]))
		if email == "" {
			skipped++
			continue
		}

		// Check if this (email, type) combination already exists
		var existingID int
		checkErr := db.DB.QueryRow(`
			SELECT id FROM student_eligible_honour_minor 
			WHERE student_email = ? AND type = ?
		`, email, eligibilityType).Scan(&existingID)

		if checkErr == nil {
			// Row WAS found (Scan succeeded) - this is a duplicate, skip it
			skipped++
			continue
		}

		// Check if it's actually "no rows found" error (expected case)
		if checkErr != sql.ErrNoRows {
			// Some actual database error occurred - log it and skip this row
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": database error for "+email)
			continue
		}

		// checkErr == sql.ErrNoRows means row doesn't exist, so insert it now
		result, execErr := db.DB.Exec(`
			INSERT INTO student_eligible_honour_minor (student_email, type, created_at, updated_at)
			VALUES (?, ?, NOW(), NOW())
		`, email, eligibilityType)
		if execErr != nil {
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": failed to insert "+email)
			continue
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			inserted++
		} else {
			skipped++
		}
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"inserted": inserted,
			"skipped":  skipped,
			"errors":   errors,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"inserted": inserted,
		"skipped":  skipped,
	})
}

// ImportHonourEligibility imports CSV data for honour eligibility
// Creates records with type='HONOUR'
// Same student can have multiple types: (email, 'HONOUR') and (email, 'MINOR') as separate records
func ImportHonourEligibility(w http.ResponseWriter, r *http.Request) {
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid form data",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "CSV file is required (field name: file)",
		})
		return
	}
	defer file.Close()

	importCSVData(w, file, "HONOUR")
}

// ImportMinorEligibility imports CSV data for minor eligibility
// Creates records with type='MINOR'
// Same student can have multiple types: (email, 'HONOUR') and (email, 'MINOR') as separate records
func ImportMinorEligibility(w http.ResponseWriter, r *http.Request) {
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Method not allowed",
		})
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid form data",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "CSV file is required (field name: file)",
		})
		return
	}
	defer file.Close()

	importCSVData(w, file, "MINOR")
}

// importCSVData is a helper function to import CSV data with specified type
// 
// DUAL-RECORD HANDLING:
// This function properly handles the case where same student has multiple eligibility types.
// 
// Examples:
//   - John imported as HONOUR + John imported as MINOR = 2 records (both types)
//   - Jane imported as HONOUR twice = 1 record (duplicate within same type is ignored)
//
// Database Behavior:
//   - If (email, 'HONOUR') exists and importing HONOUR: SKIPPED (duplicate)
//   - If (email, 'HONOUR') exists and importing MINOR: SUCCESS (different type, new record)
//   - UNIQUE constraint on (student_email, type) prevents duplicates within same type
//   - Same (email) can exist multiple times with different types
func importCSVData(w http.ResponseWriter, file io.ReadCloser, eligibilityType string) {
	reader := csv.NewReader(bufio.NewReader(file))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to read CSV header",
		})
		return
	}

	emailCol := -1
	for idx, col := range header {
		if strings.EqualFold(strings.TrimSpace(col), "student_email") {
			emailCol = idx
			break
		}
	}
	if emailCol == -1 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "CSV must contain 'student_email' column",
		})
		return
	}

	inserted := 0
	skipped := 0
	errors := []string{}
	lineNo := 1

	for {
		lineNo++
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": invalid CSV row")
			continue
		}

		if emailCol >= len(record) {
			skipped++
			continue
		}

		email := strings.TrimSpace(strings.ToLower(record[emailCol]))
		if email == "" {
			skipped++
			continue
		}

		// Check if this (email, type) combination already exists
		var existingID int
		checkErr := db.DB.QueryRow(`
			SELECT id FROM student_eligible_honour_minor 
			WHERE student_email = ? AND type = ?
		`, email, eligibilityType).Scan(&existingID)

		if checkErr == nil {
			// Row WAS found (Scan succeeded) - this is a duplicate, skip it
			skipped++
			continue
		}

		// Check if it's actually "no rows found" error (expected case)
		if checkErr != sql.ErrNoRows {
			// Some actual database error occurred - log it and skip this row
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": database error for "+email)
			continue
		}

		// checkErr == sql.ErrNoRows means row doesn't exist, so insert it now
		result, execErr := db.DB.Exec(`
			INSERT INTO student_eligible_honour_minor (student_email, type, created_at, updated_at)
			VALUES (?, ?, NOW(), NOW())
		`, email, eligibilityType)
		if execErr != nil {
			errors = append(errors, "Line "+strconv.Itoa(lineNo)+": failed to insert "+email)
			continue
		}

		affected, _ := result.RowsAffected()
		if affected > 0 {
			inserted++
		} else {
			skipped++
		}
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"inserted": inserted,
			"skipped":  skipped,
			"errors":   errors,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"inserted": inserted,
		"skipped":  skipped,
	})
}

