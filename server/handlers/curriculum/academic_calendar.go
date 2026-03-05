package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"strconv"

	"github.com/gorilla/mux"
)

// GetAllAcademicCalendars returns all academic calendar rows (admin view)
func GetAllAcademicCalendars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	query := `
		SELECT id, academic_year, year_level, current_semester,
		       COALESCE(batch, '') as batch,
		       semester_start_date, semester_end_date,
		       elective_selection_start, elective_selection_end,
		       is_current, created_at, updated_at
		FROM academic_calendar
		ORDER BY is_current DESC, academic_year DESC, year_level ASC
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		log.Println("Error fetching academic calendars:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to fetch academic calendars",
		})
		return
	}
	defer rows.Close()

	calendars := []models.AcademicCalendar{}
	for rows.Next() {
		var cal models.AcademicCalendar
		var batch string
		var electiveStart, electiveEnd, semStart, semEnd sql.NullTime
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&cal.ID,
			&cal.AcademicYear,
			&cal.YearLevel,
			&cal.CurrentSemester,
			&batch,
			&semStart,
			&semEnd,
			&electiveStart,
			&electiveEnd,
			&cal.IsCurrent,
			&createdAt,
			&updatedAt,
		); err != nil {
			log.Println("Error scanning academic calendar row:", err)
			continue
		}
		if batch != "" {
			cal.Batch = &batch
		}
		if semStart.Valid {
			cal.SemesterStartDate = semStart.Time
		}
		if semEnd.Valid {
			cal.SemesterEndDate = semEnd.Time
		}
		if electiveStart.Valid {
			cal.ElectiveSelectionStart = &electiveStart.Time
		}
		if electiveEnd.Valid {
			cal.ElectiveSelectionEnd = &electiveEnd.Time
		}
		if createdAt.Valid {
			cal.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			cal.UpdatedAt = updatedAt.Time
		}
		calendars = append(calendars, cal)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"calendars": calendars,
	})
}

// CreateAcademicCalendar creates a new academic calendar row
func CreateAcademicCalendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		AcademicYear           string  `json:"academic_year"`
		YearLevel              int     `json:"year_level"`
		CurrentSemester        int     `json:"current_semester"`
		Batch                  *string `json:"batch"`
		SemesterStartDate      string  `json:"semester_start_date"`
		SemesterEndDate        string  `json:"semester_end_date"`
		ElectiveSelectionStart *string `json:"elective_selection_start"`
		ElectiveSelectionEnd   *string `json:"elective_selection_end"`
		IsCurrent              bool    `json:"is_current"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	if req.AcademicYear == "" || req.YearLevel < 1 || req.YearLevel > 4 || req.CurrentSemester < 1 || req.CurrentSemester > 8 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Academic year, valid year_level (1-4), and current_semester (1-8) are required",
		})
		return
	}

	result, err := db.DB.Exec(`
		INSERT INTO academic_calendar
		(academic_year, year_level, current_semester, batch, semester_start_date, semester_end_date,
		 elective_selection_start, elective_selection_end, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.AcademicYear, req.YearLevel, req.CurrentSemester, req.Batch,
		req.SemesterStartDate, req.SemesterEndDate,
		req.ElectiveSelectionStart, req.ElectiveSelectionEnd, req.IsCurrent,
	)
	if err != nil {
		log.Println("Error creating academic calendar:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to create academic calendar: " + err.Error(),
		})
		return
	}

	id, _ := result.LastInsertId()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Academic calendar created",
		"id":      id,
	})
}

// UpdateAcademicCalendar updates an existing academic calendar row
func UpdateAcademicCalendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid calendar ID",
		})
		return
	}

	var req struct {
		AcademicYear           string  `json:"academic_year"`
		YearLevel              int     `json:"year_level"`
		CurrentSemester        int     `json:"current_semester"`
		Batch                  *string `json:"batch"`
		SemesterStartDate      string  `json:"semester_start_date"`
		SemesterEndDate        string  `json:"semester_end_date"`
		ElectiveSelectionStart *string `json:"elective_selection_start"`
		ElectiveSelectionEnd   *string `json:"elective_selection_end"`
		IsCurrent              bool    `json:"is_current"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE academic_calendar
		SET academic_year = ?, year_level = ?, current_semester = ?, batch = ?,
		    semester_start_date = ?, semester_end_date = ?,
		    elective_selection_start = ?, elective_selection_end = ?,
		    is_current = ?, updated_at = NOW()
		WHERE id = ?`,
		req.AcademicYear, req.YearLevel, req.CurrentSemester, req.Batch,
		req.SemesterStartDate, req.SemesterEndDate,
		req.ElectiveSelectionStart, req.ElectiveSelectionEnd,
		req.IsCurrent, id,
	)
	if err != nil {
		log.Println("Error updating academic calendar:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to update academic calendar: " + err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Academic calendar updated",
	})
}

// DeleteAcademicCalendar deletes an academic calendar row
func DeleteAcademicCalendar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Invalid calendar ID",
		})
		return
	}

	_, err = db.DB.Exec("DELETE FROM academic_calendar WHERE id = ?", id)
	if err != nil {
		log.Println("Error deleting academic calendar:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to delete academic calendar: " + err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Academic calendar deleted",
	})
}

// AdvanceAcademicYear handles advancing all calendars to the next academic year
// This marks old rows as not current and creates new rows with incremented semesters
func AdvanceAcademicYear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		NewAcademicYear   string `json:"new_academic_year"`
		SemesterStartDate string `json:"semester_start_date"`
		SemesterEndDate   string `json:"semester_end_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewAcademicYear == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "new_academic_year is required",
		})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Get current calendars
	rows, err := tx.Query(`
		SELECT id, academic_year, year_level, current_semester, COALESCE(batch, '') as batch
		FROM academic_calendar WHERE is_current = 1
		ORDER BY year_level
	`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to fetch current calendars",
		})
		return
	}

	type calRow struct {
		ID              int
		AcademicYear    string
		YearLevel       int
		CurrentSemester int
		Batch           string
	}
	var current []calRow
	for rows.Next() {
		var c calRow
		rows.Scan(&c.ID, &c.AcademicYear, &c.YearLevel, &c.CurrentSemester, &c.Batch)
		current = append(current, c)
	}
	rows.Close()

	// Mark all current rows as not current
	_, err = tx.Exec("UPDATE academic_calendar SET is_current = 0 WHERE is_current = 1")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to update old calendars",
		})
		return
	}

	// Create new rows: advance semesters, shift year_levels
	// Year level 4 graduates out, year levels 1-3 move up, new year level 1 enters
	created := 0

	// Advance existing batches (year 1→2, 2→3, 3→4)
	for _, c := range current {
		if c.YearLevel >= 4 {
			continue // year 4 students graduate
		}
		newYearLevel := c.YearLevel + 1
		newSemester := c.CurrentSemester + 1
		if newSemester > 8 {
			continue
		}

		_, err = tx.Exec(`
			INSERT INTO academic_calendar
			(academic_year, year_level, current_semester, batch, semester_start_date, semester_end_date, is_current)
			VALUES (?, ?, ?, ?, ?, ?, 1)`,
			req.NewAcademicYear, newYearLevel, newSemester, c.Batch,
			req.SemesterStartDate, req.SemesterEndDate,
		)
		if err != nil {
			log.Println("Error creating advanced calendar:", err)
			continue
		}
		created++
	}

	// Create new year level 1 entry (new batch)
	// Derive batch from academic year: e.g., "2026-2027" → new batch "2026-2030"
	newBatch := ""
	if len(req.NewAcademicYear) >= 4 {
		startYear := req.NewAcademicYear[:4]
		if yr, err := strconv.Atoi(startYear); err == nil {
			newBatch = strconv.Itoa(yr) + "-" + strconv.Itoa(yr+4)
		}
	}

	_, err = tx.Exec(`
		INSERT INTO academic_calendar
		(academic_year, year_level, current_semester, batch, semester_start_date, semester_end_date, is_current)
		VALUES (?, 1, 1, ?, ?, ?, 1)`,
		req.NewAcademicYear, newBatch, req.SemesterStartDate, req.SemesterEndDate,
	)
	if err != nil {
		log.Println("Error creating new year 1 calendar:", err)
	} else {
		created++
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Failed to commit transaction",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Academic year advanced successfully",
		"created": created,
	})
}
