package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"server/db"
)

// SubmitMarkAppeal lets a teacher submit an appeal for a missed mark entry window.
// POST /api/mark-appeals
func SubmitMarkAppeal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		TeacherID string `json:"teacher_id"`
		CourseID  int    `json:"course_id"`
		WindowID  int    `json:"window_id"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.TeacherID = normalizeFacultyIdentifier(strings.TrimSpace(req.TeacherID))
	if req.TeacherID == "" || req.CourseID == 0 || req.WindowID == 0 {
		http.Error(w, "teacher_id, course_id and window_id are required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		http.Error(w, "reason is required", http.StatusBadRequest)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	_, err := database.Exec(`
		INSERT INTO mark_appeal_requests (teacher_id, course_id, window_id, reason, status)
		VALUES (?, ?, ?, ?, 'pending')
		ON DUPLICATE KEY UPDATE reason = VALUES(reason), status = 'pending', created_at = CURRENT_TIMESTAMP
	`, req.TeacherID, req.CourseID, req.WindowID, strings.TrimSpace(req.Reason))
	if err != nil {
		log.Printf("[SubmitMarkAppeal] Error: %v", err)
		http.Error(w, "Failed to save appeal", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Appeal submitted successfully"})
}

// GetMarkAppeals returns appeal requests.
// GET /api/mark-appeals?status=pending&window_id=X&teacher_id=Y&course_id=Z
func GetMarkAppeals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	windowIDStr := r.URL.Query().Get("window_id")
	teacherID := strings.TrimSpace(r.URL.Query().Get("teacher_id"))
	courseIDStr := r.URL.Query().Get("course_id")

	query := `
		SELECT
			ar.id, ar.teacher_id, COALESCE(t.name, ar.teacher_id) as teacher_name,
			ar.course_id, COALESCE(c.course_code, '') as course_code, COALESCE(c.course_name, '') as course_name,
			ar.window_id, COALESCE(mew.window_name, '') as window_name,
			ar.reason, ar.status, ar.created_at, ar.resolved_at, ar.resolved_by
		FROM mark_appeal_requests ar
		LEFT JOIN teachers t ON t.faculty_id = ar.teacher_id
		LEFT JOIN courses c ON c.id = ar.course_id
		LEFT JOIN mark_entry_windows mew ON mew.id = ar.window_id
		WHERE 1=1
	`
	args := []interface{}{}

	if statusFilter != "" {
		query += " AND ar.status = ?"
		args = append(args, statusFilter)
	}
	if windowIDStr != "" {
		wid, err := strconv.Atoi(windowIDStr)
		if err == nil {
			query += " AND ar.window_id = ?"
			args = append(args, wid)
		}
	}
	if teacherID != "" {
		// Normalize to faculty_id for lookup
		normalized := normalizeFacultyIdentifier(teacherID)
		if normalized != "" {
			teacherID = normalized
		}
		query += " AND ar.teacher_id = ?"
		args = append(args, teacherID)
	}
	if courseIDStr != "" {
		cid, err := strconv.Atoi(courseIDStr)
		if err == nil {
			query += " AND ar.course_id = ?"
			args = append(args, cid)
		}
	}
	query += " ORDER BY ar.created_at DESC"

	rows, err := database.Query(query, args...)
	if err != nil {
		log.Printf("[GetMarkAppeals] Error: %v", err)
		http.Error(w, "Failed to fetch appeals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Appeal struct {
		ID          int    `json:"id"`
		TeacherID   string `json:"teacher_id"`
		TeacherName string `json:"teacher_name"`
		CourseID    int    `json:"course_id"`
		CourseCode  string `json:"course_code"`
		CourseName  string `json:"course_name"`
		WindowID    int    `json:"window_id"`
		WindowName  string `json:"window_name"`
		Reason      string `json:"reason"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
		ResolvedAt  string `json:"resolved_at,omitempty"`
		ResolvedBy  string `json:"resolved_by,omitempty"`
	}

	var appeals []Appeal
	for rows.Next() {
		var a Appeal
		var createdAt time.Time
		var resolvedAt sql.NullTime
		var resolvedBy sql.NullString
		if err := rows.Scan(
			&a.ID, &a.TeacherID, &a.TeacherName,
			&a.CourseID, &a.CourseCode, &a.CourseName,
			&a.WindowID, &a.WindowName,
			&a.Reason, &a.Status, &createdAt, &resolvedAt, &resolvedBy,
		); err != nil {
			log.Printf("[GetMarkAppeals] Scan error: %v", err)
			continue
		}
		a.CreatedAt = createdAt.Format(time.RFC3339)
		if resolvedAt.Valid {
			a.ResolvedAt = resolvedAt.Time.Format(time.RFC3339)
		}
		if resolvedBy.Valid {
			a.ResolvedBy = resolvedBy.String
		}
		appeals = append(appeals, a)
	}
	if appeals == nil {
		appeals = []Appeal{}
	}
	json.NewEncoder(w).Encode(appeals)
}

// ResolveMarkAppeal marks an appeal as resolved or rejected.
// POST /api/mark-appeals/{id}/resolve
func ResolveMarkAppeal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract ID from path: /api/mark-appeals/{id}/resolve
	parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
	var idStr string
	for i, p := range parts {
		if p == "resolve" && i > 0 {
			idStr = parts[i-1]
			break
		}
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid appeal ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ResolvedBy       string `json:"resolved_by"`
		Status           string `json:"status"`            // "resolved" or "rejected"
		DeleteSubmission bool   `json:"delete_submission"` // if true, remove mark_submissions record (active-window appeal)
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Status == "" {
		req.Status = "resolved"
	}

	database := db.DB
	if database == nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	// If this is an active-window appeal approval, delete the mark_submission so the teacher can re-enter
	if req.DeleteSubmission && req.Status == "resolved" {
		// Look up the appeal details first
		var teacherID string
		var courseID, windowID int
		err = database.QueryRow(`
			SELECT teacher_id, course_id, window_id FROM mark_appeal_requests WHERE id = ?
		`, id).Scan(&teacherID, &courseID, &windowID)
		if err != nil {
			log.Printf("[ResolveMarkAppeal] Could not fetch appeal details for submission delete: %v", err)
		} else {
			res, delErr := database.Exec(`
				DELETE FROM mark_submissions WHERE teacher_id = ? AND course_id = ? AND window_id = ?
			`, teacherID, courseID, windowID)
			if delErr != nil {
				log.Printf("[ResolveMarkAppeal] Error deleting mark submission: %v", delErr)
			} else {
				rows, _ := res.RowsAffected()
				log.Printf("[ResolveMarkAppeal] Deleted %d mark_submission row(s) for teacher=%s course=%d window=%d", rows, teacherID, courseID, windowID)
			}
		}
	}

	_, err = database.Exec(`
		UPDATE mark_appeal_requests
		SET status = ?, resolved_at = CURRENT_TIMESTAMP, resolved_by = ?
		WHERE id = ?
	`, req.Status, req.ResolvedBy, id)
	if err != nil {
		log.Printf("[ResolveMarkAppeal] Error: %v", err)
		http.Error(w, "Failed to resolve appeal", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}
