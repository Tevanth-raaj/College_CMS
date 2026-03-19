package curriculum

import (
	"database/sql"
	"errors"
	"log"
	"server/db"
	"server/models"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func isMySQLUnknownColumnError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1054
}

func isMySQLNoDefaultValueForField(err error, field string) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	if mysqlErr.Number != 1364 {
		return false
	}
	return strings.Contains(strings.ToLower(mysqlErr.Message), strings.ToLower(field))
}

func isMySQLDuplicateEntryError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// Helper functions for normalized syllabus data access
// All tables reference course_id directly (course-centric design)

// fetchObjectives retrieves all objectives for a course ordered by position
func fetchObjectives(courseID int) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT objective 
		FROM course_objectives 
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position`, courseID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	objectives := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			objectives = append(objectives, text)
		}
	}
	return objectives, nil
}

// fetchOutcomes retrieves all outcomes for a course ordered by position
func fetchOutcomes(courseID int) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT outcome 
		FROM course_outcomes 
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position`, courseID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	outcomes := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			outcomes = append(outcomes, text)
		}
	}
	return outcomes, nil
}

// fetchReferences retrieves all references for a course ordered by position
func fetchReferences(courseID int) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT reference_text 
		FROM course_references 
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position`, courseID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	references := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			references = append(references, text)
		}
	}
	return references, nil
}

// fetchTextbookReferences retrieves all textbook references for a course ordered by position
func fetchTextbookReferences(courseID int) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT textbook
		FROM course_textbook_reference
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position`, courseID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	textbooks := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			textbooks = append(textbooks, text)
		}
	}
	return textbooks, nil
}

// fetchPrerequisites retrieves all prerequisites for a course ordered by position
func fetchPrerequisites(courseID int) ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT prerequisite 
		FROM course_prerequisites 
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position`, courseID)
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	prerequisites := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			prerequisites = append(prerequisites, text)
		}
	}
	return prerequisites, nil
}

// fetchTeamwork retrieves teamwork data for a course
func fetchTeamwork(courseID int) (*models.Teamwork, error) {
	// Get teamwork_id and total_hours from course_teamwork using course_id
	var teamworkID int
	var hours int
	err := db.DB.QueryRow(`
		SELECT id, total_hours
		FROM course_teamwork
		WHERE course_id = ?`, courseID).Scan(&teamworkID, &hours)

	if err == sql.ErrNoRows {
		return nil, nil // No teamwork data
	}
	if err != nil {
		log.Printf("Error fetching teamwork for courseID %d: %v", courseID, err)
		return nil, err
	}

	// Fetch activities from course_teamwork_activities using teamwork_id
	rows, err := db.DB.Query(`
		SELECT activity
		FROM course_teamwork_activities
		WHERE teamwork_id = ?
		ORDER BY position`, teamworkID)
	if err != nil {
		log.Printf("Error fetching teamwork activities for teamwork_id %d: %v", teamworkID, err)
		return &models.Teamwork{Hours: hours, Activities: []string{}}, nil
	}
	defer rows.Close()

	activities := []string{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err == nil {
			activities = append(activities, text)
		}
	}

	return &models.Teamwork{
		Hours:      hours,
		Activities: activities,
	}, nil
}

// fetchSelfLearning retrieves self-learning data for a course
func fetchSelfLearning(courseID int) (*models.SelfLearning, error) {
	var selfLearningID int
	var hours int
	err := db.DB.QueryRow(`
		SELECT id, total_hours 
		FROM course_selflearning 
		WHERE course_id = ?
		LIMIT 1`, courseID).Scan(&selfLearningID, &hours)

	if err == sql.ErrNoRows {
		return nil, nil // No self-learning data
	}
	if err != nil {
		return nil, err
	}

	// Fetch main topics
	rows, err := db.DB.Query(`
		SELECT id, main_text 
		FROM course_selflearning_topics 
		WHERE selflearning_id = ? 
		ORDER BY position`, selfLearningID)
	if err != nil {
		return &models.SelfLearning{Hours: hours, MainInputs: []models.SelfLearningInternal{}}, nil
	}
	defer rows.Close()

	mainInputs := []models.SelfLearningInternal{}
	for rows.Next() {
		var mainID int
		var mainText string
		if err := rows.Scan(&mainID, &mainText); err == nil {
			// Fetch internal resources for this main topic
			internalRows, err := db.DB.Query(`
				SELECT internal_text 
				FROM course_selflearning_resources 
				WHERE main_id = ? 
				ORDER BY position`, mainID)

			internal := []string{}
			if err == nil {
				for internalRows.Next() {
					var text string
					if err := internalRows.Scan(&text); err == nil {
						internal = append(internal, text)
					}
				}
				internalRows.Close()
			}

			mainInputs = append(mainInputs, models.SelfLearningInternal{
				Main:     mainText,
				Internal: internal,
			})
		}
	}

	return &models.SelfLearning{
		Hours:      hours,
		MainInputs: mainInputs,
	}, nil
}

// saveObjectives saves objectives for a course, replacing existing ones
func saveObjectives(courseID int, objectives []string) error {
	log.Printf("DEBUG saveObjectives: courseID=%d, count=%d", courseID, len(objectives))

	// Normalize input (keep order; skip empty)
	incoming := make([]string, 0, len(objectives))
	for _, o := range objectives {
		norm := strings.TrimSpace(o)
		if norm == "" {
			continue
		}
		incoming = append(incoming, norm)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("ERROR saveObjectives begin tx: %v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	type existingObjective struct {
		ID       int
		Text     string
		Position int
	}

	existing := make([]existingObjective, 0)
	rows, err := tx.Query(`
		SELECT id, objective, position
		FROM course_objectives
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)
	if err != nil {
		log.Printf("ERROR saveObjectives load existing: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var o existingObjective
		if err := rows.Scan(&o.ID, &o.Text, &o.Position); err == nil {
			existing = append(existing, o)
		}
	}

	// Case 1: edit-only (same count) => update same records by index
	if len(incoming) == len(existing) {
		for i := range incoming {
			if _, err := tx.Exec(
				"UPDATE course_objectives SET objective = ?, position = ?, status = 1 WHERE id = ?",
				incoming[i], i, existing[i].ID,
			); err != nil {
				log.Printf("ERROR saveObjectives update id=%d: %v", existing[i].ID, err)
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("ERROR saveObjectives commit: %v", err)
			return err
		}
		log.Printf("DEBUG saveObjectives: updated %d objectives (edit-only)", len(incoming))
		return nil
	}

	// Case 2: add/remove => diff by text, shift positions without overwriting texts
	existingTexts := make([]string, 0, len(existing))
	for _, o := range existing {
		existingTexts = append(existingTexts, strings.TrimSpace(o.Text))
	}
	n := len(existingTexts)
	m := len(incoming)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if existingTexts[i-1] == incoming[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	matchedExisting := make(map[int]int)
	matchedIncoming := make(map[int]bool)
	i := n
	j := m
	for i > 0 && j > 0 {
		if existingTexts[i-1] == incoming[j-1] {
			matchedExisting[i-1] = j - 1
			matchedIncoming[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Soft delete removed
	for idx, o := range existing {
		if _, ok := matchedExisting[idx]; !ok {
			if _, err := tx.Exec("UPDATE course_objectives SET status = 0 WHERE id = ?", o.ID); err != nil {
				log.Printf("ERROR saveObjectives soft delete id=%d: %v", o.ID, err)
				return err
			}
		}
	}

	// Avoid unique(course_id,position) collisions while shifting positions
	if _, err := tx.Exec(
		"UPDATE course_objectives SET position = position + 10000 WHERE course_id = ? AND (status = 1 OR status IS NULL)",
		courseID,
	); err != nil {
		log.Printf("ERROR saveObjectives temp bump: %v", err)
		return err
	}

	// Shift kept objectives (position only)
	for oldIdx, newIdx := range matchedExisting {
		id := existing[oldIdx].ID
		if _, err := tx.Exec("UPDATE course_objectives SET position = ?, status = 1 WHERE id = ?", newIdx, id); err != nil {
			log.Printf("ERROR saveObjectives shift id=%d: %v", id, err)
			return err
		}
	}

	// Insert new objectives
	for idx, text := range incoming {
		if matchedIncoming[idx] {
			continue
		}
		if _, err := tx.Exec(
			"INSERT INTO course_objectives (course_id, objective, position, status) VALUES (?, ?, ?, 1)",
			courseID, text, idx,
		); err != nil {
			log.Printf("ERROR saveObjectives insert position=%d: %v", idx, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR saveObjectives commit: %v", err)
		return err
	}
	log.Printf("DEBUG saveObjectives: successfully saved %d objectives", len(incoming))
	return nil
}

// saveOutcomes saves outcomes for a course, replacing existing ones
func saveOutcomes(courseID int, outcomes []string) error {
	log.Printf("DEBUG saveOutcomes: courseID=%d, count=%d", courseID, len(outcomes))

	// Normalize input (keep order; skip empty)
	incoming := make([]string, 0, len(outcomes))
	for _, o := range outcomes {
		norm := strings.TrimSpace(o)
		if norm == "" {
			continue
		}
		incoming = append(incoming, norm)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("ERROR saveOutcomes begin tx: %v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	type existingOutcome struct {
		ID       int
		Text     string
		Position int
	}

	existing := make([]existingOutcome, 0)
	rows, err := tx.Query(`
		SELECT id, outcome, position
		FROM course_outcomes
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)
	if err != nil {
		log.Printf("ERROR saveOutcomes load existing: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var o existingOutcome
		if err := rows.Scan(&o.ID, &o.Text, &o.Position); err == nil {
			existing = append(existing, o)
		}
	}

	// Case 1: edit-only (same count) => update same records by index
	if len(incoming) == len(existing) {
		for i := range incoming {
			if _, err := tx.Exec(
				"UPDATE course_outcomes SET outcome = ?, position = ?, status = 1 WHERE id = ?",
				incoming[i], i, existing[i].ID,
			); err != nil {
				log.Printf("ERROR saveOutcomes update id=%d: %v", existing[i].ID, err)
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("ERROR saveOutcomes commit: %v", err)
			return err
		}
		log.Printf("DEBUG saveOutcomes: updated %d outcomes (edit-only)", len(incoming))
		return nil
	}

	// Case 2: add/remove => diff by text, shift positions without overwriting texts
	existingTexts := make([]string, 0, len(existing))
	for _, o := range existing {
		existingTexts = append(existingTexts, strings.TrimSpace(o.Text))
	}
	n := len(existingTexts)
	m := len(incoming)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if existingTexts[i-1] == incoming[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	matchedExisting := make(map[int]int)
	matchedIncoming := make(map[int]bool)
	i := n
	j := m
	for i > 0 && j > 0 {
		if existingTexts[i-1] == incoming[j-1] {
			matchedExisting[i-1] = j - 1
			matchedIncoming[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Soft delete removed
	for idx, o := range existing {
		if _, ok := matchedExisting[idx]; !ok {
			if _, err := tx.Exec("UPDATE course_outcomes SET status = 0 WHERE id = ?", o.ID); err != nil {
				log.Printf("ERROR saveOutcomes soft delete id=%d: %v", o.ID, err)
				return err
			}
		}
	}

	// Avoid unique(course_id,position) collisions while shifting positions
	if _, err := tx.Exec(
		"UPDATE course_outcomes SET position = position + 10000 WHERE course_id = ? AND (status = 1 OR status IS NULL)",
		courseID,
	); err != nil {
		log.Printf("ERROR saveOutcomes temp bump: %v", err)
		return err
	}

	// Shift kept outcomes (position only)
	for oldIdx, newIdx := range matchedExisting {
		id := existing[oldIdx].ID
		if _, err := tx.Exec("UPDATE course_outcomes SET position = ?, status = 1 WHERE id = ?", newIdx, id); err != nil {
			log.Printf("ERROR saveOutcomes shift id=%d: %v", id, err)
			return err
		}
	}

	// Insert new outcomes
	for idx, text := range incoming {
		if matchedIncoming[idx] {
			continue
		}
		if _, err := tx.Exec(
			"INSERT INTO course_outcomes (course_id, outcome, position, status) VALUES (?, ?, ?, 1)",
			courseID, text, idx,
		); err != nil {
			log.Printf("ERROR saveOutcomes insert position=%d: %v", idx, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR saveOutcomes commit: %v", err)
		return err
	}
	log.Printf("DEBUG saveOutcomes: successfully saved %d outcomes", len(incoming))
	return nil
}

// saveReferences saves references for a course, replacing existing ones
func saveReferences(courseID int, references []string) error {
	log.Printf("DEBUG saveReferences: courseID=%d, count=%d", courseID, len(references))

	// Normalize input (keep order; skip empty)
	incoming := make([]string, 0, len(references))
	for _, r := range references {
		norm := strings.TrimSpace(r)
		if norm == "" {
			continue
		}
		incoming = append(incoming, norm)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("ERROR saveReferences begin tx: %v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	type existingRef struct {
		ID       int
		Text     string
		Position int
	}

	existing := make([]existingRef, 0)
	rows, err := tx.Query(`
		SELECT id, reference_text, position
		FROM course_references
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)
	if err != nil {
		log.Printf("ERROR saveReferences load existing: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r existingRef
		if err := rows.Scan(&r.ID, &r.Text, &r.Position); err == nil {
			existing = append(existing, r)
		}
	}

	// Case 1: edit-only (same count) => update same records by index
	if len(incoming) == len(existing) {
		for i := range incoming {
			if _, err := tx.Exec(
				"UPDATE course_references SET reference_text = ?, position = ?, status = 1 WHERE id = ?",
				incoming[i], i, existing[i].ID,
			); err != nil {
				log.Printf("ERROR saveReferences update id=%d: %v", existing[i].ID, err)
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("ERROR saveReferences commit: %v", err)
			return err
		}
		log.Printf("DEBUG saveReferences: updated %d references (edit-only)", len(incoming))
		return nil
	}

	// Case 2: add/remove => diff by text, shift positions without overwriting texts
	existingTexts := make([]string, 0, len(existing))
	for _, r := range existing {
		existingTexts = append(existingTexts, strings.TrimSpace(r.Text))
	}
	n := len(existingTexts)
	m := len(incoming)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if existingTexts[i-1] == incoming[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	matchedExisting := make(map[int]int)
	matchedIncoming := make(map[int]bool)
	i := n
	j := m
	for i > 0 && j > 0 {
		if existingTexts[i-1] == incoming[j-1] {
			matchedExisting[i-1] = j - 1
			matchedIncoming[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Soft delete removed
	for idx, r := range existing {
		if _, ok := matchedExisting[idx]; !ok {
			if _, err := tx.Exec("UPDATE course_references SET status = 0 WHERE id = ?", r.ID); err != nil {
				log.Printf("ERROR saveReferences soft delete id=%d: %v", r.ID, err)
				return err
			}
		}
	}

	// Avoid unique(course_id,position) collisions while shifting positions
	if _, err := tx.Exec(
		"UPDATE course_references SET position = position + 10000 WHERE course_id = ? AND (status = 1 OR status IS NULL)",
		courseID,
	); err != nil {
		log.Printf("ERROR saveReferences temp bump: %v", err)
		return err
	}

	// Shift kept refs (position only)
	for oldIdx, newIdx := range matchedExisting {
		id := existing[oldIdx].ID
		if _, err := tx.Exec("UPDATE course_references SET position = ?, status = 1 WHERE id = ?", newIdx, id); err != nil {
			log.Printf("ERROR saveReferences shift id=%d: %v", id, err)
			return err
		}
	}

	// Insert new refs
	for idx, text := range incoming {
		if matchedIncoming[idx] {
			continue
		}
		if _, err := tx.Exec(
			"INSERT INTO course_references (course_id, reference_text, position, status) VALUES (?, ?, ?, 1)",
			courseID, text, idx,
		); err != nil {
			log.Printf("ERROR saveReferences insert position=%d: %v", idx, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR saveReferences commit: %v", err)
		return err
	}
	log.Printf("DEBUG saveReferences: successfully saved %d references", len(incoming))
	return nil
}

// saveTextbookReferences saves textbook references for a course, replacing existing ones
func saveTextbookReferences(courseID int, references []string) error {
	log.Printf("DEBUG saveTextbookReferences: courseID=%d, count=%d", courseID, len(references))

	incoming := make([]string, 0, len(references))
	for _, r := range references {
		norm := strings.TrimSpace(r)
		if norm == "" {
			continue
		}
		incoming = append(incoming, norm)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("ERROR saveTextbookReferences begin tx: %v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	type existingRef struct {
		ID       int
		Text     string
		Position int
	}

	existing := make([]existingRef, 0)
	rows, err := tx.Query(`
		SELECT id, textbook, position
		FROM course_textbook_reference
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)
	if err != nil {
		log.Printf("ERROR saveTextbookReferences load existing: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r existingRef
		if err := rows.Scan(&r.ID, &r.Text, &r.Position); err == nil {
			existing = append(existing, r)
		}
	}

	if len(incoming) == len(existing) {
		for i := range incoming {
			if _, err := tx.Exec(
				"UPDATE course_textbook_reference SET textbook = ?, position = ?, status = 1 WHERE id = ?",
				incoming[i], i, existing[i].ID,
			); err != nil {
				log.Printf("ERROR saveTextbookReferences update id=%d: %v", existing[i].ID, err)
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("ERROR saveTextbookReferences commit: %v", err)
			return err
		}
		log.Printf("DEBUG saveTextbookReferences: updated %d references (edit-only)", len(incoming))
		return nil
	}

	existingTexts := make([]string, 0, len(existing))
	for _, r := range existing {
		existingTexts = append(existingTexts, strings.TrimSpace(r.Text))
	}
	n := len(existingTexts)
	m := len(incoming)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if existingTexts[i-1] == incoming[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	matchedExisting := make(map[int]int)
	matchedIncoming := make(map[int]bool)
	i := n
	j := m
	for i > 0 && j > 0 {
		if existingTexts[i-1] == incoming[j-1] {
			matchedExisting[i-1] = j - 1
			matchedIncoming[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	for idx, r := range existing {
		if _, ok := matchedExisting[idx]; !ok {
			if _, err := tx.Exec("UPDATE course_textbook_reference SET status = 0 WHERE id = ?", r.ID); err != nil {
				log.Printf("ERROR saveTextbookReferences soft delete id=%d: %v", r.ID, err)
				return err
			}
		}
	}

	if _, err := tx.Exec(
		"UPDATE course_textbook_reference SET position = position + 10000 WHERE course_id = ? AND (status = 1 OR status IS NULL)",
		courseID,
	); err != nil {
		log.Printf("ERROR saveTextbookReferences temp bump: %v", err)
		return err
	}

	for oldIdx, newIdx := range matchedExisting {
		id := existing[oldIdx].ID
		if _, err := tx.Exec("UPDATE course_textbook_reference SET position = ?, status = 1 WHERE id = ?", newIdx, id); err != nil {
			log.Printf("ERROR saveTextbookReferences shift id=%d: %v", id, err)
			return err
		}
	}

	for idx, text := range incoming {
		if matchedIncoming[idx] {
			continue
		}
		if _, err := tx.Exec(
			"INSERT INTO course_textbook_reference (course_id, textbook, position, status) VALUES (?, ?, ?, 1)",
			courseID, text, idx,
		); err != nil {
			log.Printf("ERROR saveTextbookReferences insert position=%d: %v", idx, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR saveTextbookReferences commit: %v", err)
		return err
	}
	log.Printf("DEBUG saveTextbookReferences: successfully saved %d references", len(incoming))
	return nil
}

// savePrerequisites saves prerequisites for a course with soft-delete semantics
func savePrerequisites(courseID int, prerequisites []string) error {
	log.Printf("DEBUG savePrerequisites: courseID=%d, count=%d", courseID, len(prerequisites))

	// Normalize input (keep order; skip empty)
	incoming := make([]string, 0, len(prerequisites))
	for _, p := range prerequisites {
		norm := strings.TrimSpace(p)
		if norm == "" {
			continue
		}
		incoming = append(incoming, norm)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("ERROR savePrerequisites begin tx: %v", err)
		return err
	}
	defer func() { _ = tx.Rollback() }()

	type existingPrereq struct {
		ID       int
		Text     string
		Position int
	}

	existing := make([]existingPrereq, 0)
	rows, err := tx.Query(`
		SELECT id, prerequisite, position
		FROM course_prerequisites
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)
	if err != nil {
		log.Printf("ERROR savePrerequisites load existing: %v", err)
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p existingPrereq
		if err := rows.Scan(&p.ID, &p.Text, &p.Position); err == nil {
			existing = append(existing, p)
		}
	}

	// Case 1: edit-only (same count) => update same records by index
	if len(incoming) == len(existing) {
		for i := range incoming {
			if _, err := tx.Exec(
				"UPDATE course_prerequisites SET prerequisite = ?, position = ?, status = 1 WHERE id = ?",
				incoming[i], i, existing[i].ID,
			); err != nil {
				log.Printf("ERROR savePrerequisites update id=%d: %v", existing[i].ID, err)
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			log.Printf("ERROR savePrerequisites commit: %v", err)
			return err
		}
		log.Printf("DEBUG savePrerequisites: updated %d prerequisites (edit-only)", len(incoming))
		return nil
	}

	// Case 2: add/remove => diff by text, shift positions without overwriting texts
	existingTexts := make([]string, 0, len(existing))
	for _, p := range existing {
		existingTexts = append(existingTexts, strings.TrimSpace(p.Text))
	}
	n := len(existingTexts)
	m := len(incoming)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if existingTexts[i-1] == incoming[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] >= dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	matchedExisting := make(map[int]int)
	matchedIncoming := make(map[int]bool)
	i := n
	j := m
	for i > 0 && j > 0 {
		if existingTexts[i-1] == incoming[j-1] {
			matchedExisting[i-1] = j - 1
			matchedIncoming[j-1] = true
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	// Soft delete removed
	for idx, p := range existing {
		if _, ok := matchedExisting[idx]; !ok {
			if _, err := tx.Exec("UPDATE course_prerequisites SET status = 0 WHERE id = ?", p.ID); err != nil {
				log.Printf("ERROR savePrerequisites soft delete id=%d: %v", p.ID, err)
				return err
			}
		}
	}

	// Avoid unique(course_id,position) collisions while shifting positions
	if _, err := tx.Exec(
		"UPDATE course_prerequisites SET position = position + 10000 WHERE course_id = ? AND (status = 1 OR status IS NULL)",
		courseID,
	); err != nil {
		log.Printf("ERROR savePrerequisites temp bump: %v", err)
		return err
	}

	// Shift kept prerequisites (position only)
	for oldIdx, newIdx := range matchedExisting {
		id := existing[oldIdx].ID
		if _, err := tx.Exec("UPDATE course_prerequisites SET position = ?, status = 1 WHERE id = ?", newIdx, id); err != nil {
			log.Printf("ERROR savePrerequisites shift id=%d: %v", id, err)
			return err
		}
	}

	// Insert new prerequisites
	for idx, text := range incoming {
		if matchedIncoming[idx] {
			continue
		}
		if _, err := tx.Exec(
			"INSERT INTO course_prerequisites (course_id, prerequisite, position, status) VALUES (?, ?, ?, 1)",
			courseID, text, idx,
		); err != nil {
			log.Printf("ERROR savePrerequisites insert position=%d: %v", idx, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR savePrerequisites commit: %v", err)
		return err
	}
	log.Printf("DEBUG savePrerequisites: successfully saved %d prerequisites", len(incoming))
	return nil
}

// saveTeamwork saves teamwork data for a course
func saveTeamwork(courseID int, teamwork *models.Teamwork) error {
	if teamwork == nil {
		log.Printf("DEBUG: saveTeamwork called with nil teamwork for courseID %d", courseID)
		// Delete if nil
		if _, err := db.DB.Exec("DELETE FROM course_teamwork_activities WHERE teamwork_id = ?", courseID); err != nil {
			if isMySQLUnknownColumnError(err) {
				_, _ = db.DB.Exec("DELETE FROM course_teamwork_activities WHERE course_id = ?", courseID)
			} else {
				return err
			}
		}
		if _, err := db.DB.Exec("DELETE FROM course_teamwork WHERE id = ?", courseID); err != nil {
			if isMySQLUnknownColumnError(err) {
				_, _ = db.DB.Exec("DELETE FROM course_teamwork WHERE course_id = ?", courseID)
			} else {
				return err
			}
		}
		return nil
	}

	log.Printf("DEBUG: saveTeamwork called for courseID %d with hours=%d, activities=%v", courseID, teamwork.Hours, teamwork.Activities)

	// Upsert total_hours in course_teamwork
	// Schema: (id PRIMARY KEY, course_id, total_hours)
	var teamworkID int64

	// First, try to get existing teamwork_id for this course
	err := db.DB.QueryRow("SELECT id FROM course_teamwork WHERE course_id = ?", courseID).Scan(&teamworkID)

	var result sql.Result

	if err == sql.ErrNoRows {
		// No existing record, insert new one
		result, err = db.DB.Exec(`
			INSERT INTO course_teamwork (course_id, total_hours)
			VALUES (?, ?)`,
			courseID, teamwork.Hours)
		if err != nil {
			log.Printf("ERROR: Failed to insert course_teamwork: %v", err)
			return err
		}
		teamworkID, _ = result.LastInsertId()
		log.Printf("DEBUG: Created new teamwork record with id=%d", teamworkID)
	} else if err == nil {
		// Existing record, update it
		_, err = db.DB.Exec(`
			UPDATE course_teamwork SET total_hours = ? WHERE id = ?`,
			teamwork.Hours, teamworkID)
		if err != nil {
			log.Printf("ERROR: Failed to update course_teamwork: %v", err)
			return err
		}
		log.Printf("DEBUG: Updated teamwork record id=%d with hours=%d", teamworkID, teamwork.Hours)
	} else {
		log.Printf("ERROR: Failed to query course_teamwork: %v", err)
		return err
	}

	// Delete existing activities using teamwork_id
	if _, err := db.DB.Exec("DELETE FROM course_teamwork_activities WHERE teamwork_id = ?", teamworkID); err != nil {
		log.Printf("ERROR: Failed to delete existing teamwork activities: %v", err)
		return err
	}

	// Insert new activities using the correct teamwork_id
	log.Printf("DEBUG: Inserting %d teamwork activities for teamwork_id=%d", len(teamwork.Activities), teamworkID)
	for i, text := range teamwork.Activities {
		if text == "" {
			log.Printf("DEBUG: Skipping empty activity at position %d", i)
			continue
		}
		result, err := db.DB.Exec(`
			INSERT INTO course_teamwork_activities (teamwork_id, course_id, activity, position, status)
			VALUES (?, ?, ?, ?, 1)`, teamworkID, courseID, text, i)
		if err != nil {
			log.Printf("ERROR: Failed to insert teamwork activity at position %d: %v", i, err)
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		log.Printf("DEBUG: Inserted teamwork activity at position %d, affected %d rows", i, rowsAffected)
	}
	log.Printf("DEBUG: Successfully saved teamwork data for courseID %d", courseID)
	return nil
}

// saveSelfLearning saves self-learning data for a course
func saveSelfLearning(courseID int, selflearning *models.SelfLearning) error {
	if selflearning == nil {
		// Delete if nil
		db.DB.Exec("DELETE FROM course_selflearning_resources WHERE main_id IN (SELECT id FROM course_selflearning_topics WHERE selflearning_id IN (SELECT id FROM course_selflearning WHERE course_id = ?))", courseID)
		db.DB.Exec("DELETE FROM course_selflearning_topics WHERE selflearning_id IN (SELECT id FROM course_selflearning WHERE course_id = ?)", courseID)
		db.DB.Exec("DELETE FROM course_selflearning WHERE course_id = ?", courseID)
		return nil
	}

	var selfLearningID int
	err := db.DB.QueryRow(`
		SELECT id
		FROM course_selflearning
		WHERE course_id = ?
		ORDER BY id ASC
		LIMIT 1`, courseID).Scan(&selfLearningID)
	if err == sql.ErrNoRows {
		result, insertErr := db.DB.Exec(`
			INSERT INTO course_selflearning (course_id, total_hours)
			VALUES (?, ?)`,
			courseID, selflearning.Hours)

		if isMySQLNoDefaultValueForField(insertErr, "id") {
			if _, alterErr := db.DB.Exec("ALTER TABLE course_selflearning MODIFY COLUMN id INT NOT NULL AUTO_INCREMENT"); alterErr != nil {
				log.Printf("WARN: failed to set AUTO_INCREMENT on course_selflearning.id: %v", alterErr)
			}
			result, insertErr = db.DB.Exec(`
				INSERT INTO course_selflearning (course_id, total_hours)
				VALUES (?, ?)`,
				courseID, selflearning.Hours)

			if isMySQLNoDefaultValueForField(insertErr, "id") {
				for attempt := 0; attempt < 3; attempt++ {
					_, insertErr = db.DB.Exec(`
						INSERT INTO course_selflearning (id, course_id, total_hours)
						SELECT COALESCE(MAX(id), 0) + 1, ?, ?
						FROM course_selflearning`,
						courseID, selflearning.Hours,
					)
					if insertErr == nil {
						break
					}
					if !isMySQLDuplicateEntryError(insertErr) {
						break
					}
				}
			}
		}

		if insertErr != nil {
			return insertErr
		}

		if id, idErr := result.LastInsertId(); idErr == nil && id > 0 {
			selfLearningID = int(id)
		} else {
			err = db.DB.QueryRow(`
				SELECT id
				FROM course_selflearning
				WHERE course_id = ?
				ORDER BY id ASC
				LIMIT 1`, courseID).Scan(&selfLearningID)
			if err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	} else {
		if _, err = db.DB.Exec(
			"UPDATE course_selflearning SET total_hours = ?, status = 1 WHERE id = ?",
			selflearning.Hours, selfLearningID,
		); err != nil {
			if _, err = db.DB.Exec(
				"UPDATE course_selflearning SET total_hours = ? WHERE id = ?",
				selflearning.Hours, selfLearningID,
			); err != nil {
				return err
			}
		}
	}

	// Keep exactly one self-learning header row per course.
	db.DB.Exec("DELETE FROM course_selflearning_resources WHERE main_id IN (SELECT id FROM course_selflearning_topics WHERE selflearning_id IN (SELECT id FROM course_selflearning WHERE course_id = ? AND id <> ?))", courseID, selfLearningID)
	db.DB.Exec("DELETE FROM course_selflearning_topics WHERE selflearning_id IN (SELECT id FROM course_selflearning WHERE course_id = ? AND id <> ?)", courseID, selfLearningID)
	db.DB.Exec("DELETE FROM course_selflearning WHERE course_id = ? AND id <> ?", courseID, selfLearningID)

	// Delete existing main topics and their internals
	db.DB.Exec("DELETE FROM course_selflearning_resources WHERE main_id IN (SELECT id FROM course_selflearning_topics WHERE selflearning_id = ?)", selfLearningID)
	db.DB.Exec("DELETE FROM course_selflearning_topics WHERE selflearning_id = ?", selfLearningID)

	// Insert new main topics
	for i, mainInput := range selflearning.MainInputs {
		if strings.TrimSpace(mainInput.Main) == "" {
			continue
		}

		result, err := db.DB.Exec(`
			INSERT INTO course_selflearning_topics (selflearning_id, main_text, position, status) 
			VALUES (?, ?, ?, 1)`, selfLearningID, strings.TrimSpace(mainInput.Main), i)
		if err != nil {
			return err
		}

		mainID, _ := result.LastInsertId()

		// Insert internal resources for this main topic
		for j, text := range mainInput.Internal {
			if strings.TrimSpace(text) == "" {
				continue
			}
			_, err := db.DB.Exec(`
				INSERT INTO course_selflearning_resources (main_id, internal_text, position, status) 
				VALUES (?, ?, ?, 1)`, mainID, strings.TrimSpace(text), j)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
