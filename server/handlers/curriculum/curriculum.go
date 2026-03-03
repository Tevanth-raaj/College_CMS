package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// cascadeSoftDeleteCourse sets status=0 for all child records of a course
// If tx is provided, uses transaction; otherwise uses direct DB connection
func cascadeSoftDeleteCourse(courseID int, tx *sql.Tx) error {
	execFunc := func(query string, args ...interface{}) (sql.Result, error) {
		if tx != nil {
			return tx.Exec(query, args...)
		}
		return db.DB.Exec(query, args...)
	}

	// Soft delete objectives
	_, err := execFunc("UPDATE course_objectives SET status = 0 WHERE course_id = ? AND status = 1", courseID)
	if err != nil {
		log.Printf("Error soft-deleting objectives for course %d: %v", courseID, err)
		return err
	}

	// Soft delete outcomes
	_, err = execFunc("UPDATE course_outcomes SET status = 0 WHERE course_id = ? AND status = 1", courseID)
	if err != nil {
		log.Printf("Error soft-deleting outcomes for course %d: %v", courseID, err)
		return err
	}

	// Soft delete references
	_, err = execFunc("UPDATE course_references SET status = 0 WHERE course_id = ? AND status = 1", courseID)
	if err != nil {
		log.Printf("Error soft-deleting references for course %d: %v", courseID, err)
		return err
	}

	// Soft delete syllabus topics (cascade from titles)
	_, err = execFunc(`UPDATE syllabus_topics SET status = 0 
		WHERE title_id IN (SELECT id FROM syllabus_titles WHERE model_id IN (SELECT id FROM syllabus WHERE course_id = ?)) 
		AND status = 1`, courseID)
	if err != nil {
		log.Printf("Error soft-deleting syllabus topics for course %d: %v", courseID, err)
		return err
	}

	// Soft delete syllabus titles (cascade from models)
	_, err = execFunc(`UPDATE syllabus_titles SET status = 0 
		WHERE model_id IN (SELECT id FROM syllabus WHERE course_id = ?) 
		AND status = 1`, courseID)
	if err != nil {
		log.Printf("Error soft-deleting syllabus titles for course %d: %v", courseID, err)
		return err
	}

	// Soft delete syllabus models
	_, err = execFunc("UPDATE syllabus SET status = 0 WHERE course_id = ? AND status = 1", courseID)
	if err != nil {
		log.Printf("Error soft-deleting syllabus models for course %d: %v", courseID, err)
		return err
	}

	// Soft delete experiment topics (cascade from experiments)
	_, err = execFunc(`UPDATE course_experiment_topics SET status = 0 
		WHERE experiment_id IN (SELECT id FROM course_experiments WHERE course_id = ?) 
		AND status = 1`, courseID)
	if err != nil {
		log.Printf("Error soft-deleting experiment topics for course %d: %v", courseID, err)
		return err
	}

	// Soft delete experiments
	_, err = execFunc("UPDATE course_experiments SET status = 0 WHERE course_id = ? AND status = 1", courseID)
	if err != nil {
		log.Printf("Error soft-deleting experiments for course %d: %v", courseID, err)
		return err
	}

	log.Printf("Successfully cascaded soft-delete for course %d", courseID)
	return nil
}

// GetSemesters retrieves all semesters for a regulation
func GetSemesters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	query := "SELECT id, curriculum_id, semester_number, COALESCE(card_type, 'semester') as card_type, COALESCE(vertical_name, '') as vertical_name FROM normal_cards WHERE curriculum_id = ? AND (status = 1 OR status IS NULL) ORDER BY COALESCE(semester_number, 999), id"
	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Println("Error querying semesters:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch semesters"})
		return
	}
	defer rows.Close()

	var semesters []models.Semester = make([]models.Semester, 0)
	for rows.Next() {
		var sem models.Semester
		var semesterNum sql.NullInt64
		err := rows.Scan(&sem.ID, &sem.CurriculumID, &semesterNum, &sem.CardType, &sem.VerticalName)
		if err != nil {
			log.Println("Error scanning semester:", err)
			continue
		}
		if semesterNum.Valid {
			val := int(semesterNum.Int64)
			sem.SemesterNumber = &val
		}
		semesters = append(semesters, sem)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(semesters)
}

// CreateSemester creates a new semester for a regulation
func CreateSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	var card models.Semester
	err = json.NewDecoder(r.Body).Decode(&card)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	card.CurriculumID = curriculumID

	// Set default card type if not provided
	if card.CardType == "" {
		card.CardType = "semester"
	}

	// Check if the number already exists within the same card type
	// Semester numbers must be unique among semester cards
	// Vertical numbers must be unique among vertical cards
	if card.SemesterNumber != nil {
		var existingCount int
		err = db.DB.QueryRow("SELECT COUNT(*) FROM normal_cards WHERE curriculum_id = ? AND semester_number = ? AND card_type = ? AND (status = 1 OR status IS NULL)",
			curriculumID, *card.SemesterNumber, card.CardType).Scan(&existingCount)
		if err != nil {
			log.Println("Error checking for duplicate number:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate number"})
			return
		}
		if existingCount > 0 {
			if card.CardType == "vertical" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Vertical %d already exists in this curriculum", *card.SemesterNumber)})
			} else {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Semester %d already exists in this curriculum", *card.SemesterNumber)})
			}
			return
		}
	}

	query := "INSERT INTO normal_cards (curriculum_id, semester_number, card_type, vertical_name) VALUES (?, ?, ?, ?)"
	var verticalName interface{}
	if card.CardType == "vertical" && card.VerticalName != "" {
		verticalName = card.VerticalName
	} else {
		verticalName = nil
	}
	result, err := db.DB.Exec(query, card.CurriculumID, card.SemesterNumber, card.CardType, verticalName)
	if err != nil {
		log.Println("Error inserting semester:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create semester"})
		return
	}

	id, _ := result.LastInsertId()
	card.ID = int(id)

	// Log the activity
	logName := ""
	if card.SemesterNumber != nil {
		cardTypeLabel := card.CardType
		if cardTypeLabel == "" {
			cardTypeLabel = "semester"
		}
		logName = strings.Title(cardTypeLabel) + " " + strconv.Itoa(*card.SemesterNumber)
	} else {
		logName = "New Card"
	}
	LogCurriculumActivity(curriculumID, "Card Added",
		"Added "+logName, "System")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(card)
}

// DeleteSemester deletes a semester
func DeleteSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	semesterID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid semester ID"})
		return
	}

	// Start a transaction for soft-delete cascade
	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("Error starting transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Soft delete the normal card (semester)
	query := "UPDATE normal_cards SET status = 0 WHERE id = ? AND status = 1"
	result, err := tx.Exec(query, semesterID)
	if err != nil {
		log.Println("Error deleting semester:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete semester"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Semester not found"})
		return
	}

	// Soft-delete all courses linked to this semester (keep curriculum_courses mappings intact)
	rows, err := tx.Query(`
		SELECT course_id FROM curriculum_courses WHERE semester_id = ?
	`, semesterID)
	if err != nil {
		log.Println("Error fetching semester courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch semester courses"})
		return
	}

	courseIDs := make([]int, 0)
	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			rows.Close()
			log.Println("Error scanning course ID:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch semester courses"})
			return
		}
		courseIDs = append(courseIDs, courseID)
	}
	rows.Close()

	// Soft-delete each course and cascade to its children
	for _, courseID := range courseIDs {
		_, err = tx.Exec("UPDATE courses SET status = 0 WHERE id = ? AND status = 1", courseID)
		if err != nil {
			log.Println("Error soft-deleting course:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to courses"})
			return
		}
		// Cascade soft-delete to course children (in DeleteSemester transaction)
		if err := cascadeSoftDeleteCourse(courseID, tx); err != nil {
			log.Println("Error cascading course soft-delete:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to course children"})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Println("Error committing transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit transaction"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Semester deleted successfully"})
}

// GetSemesterCourses retrieves all courses for a specific semester
func GetSemesterCourses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	semesterID, err := strconv.Atoi(vars["semId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid semester ID"})
		return
	}

	curriculumTemplate := getCurriculumTemplateByRegulation(curriculumID)

	query := `
		SELECT c.id, c.course_code, c.course_name, ct.course_type, c.category, c.credit, 
		       c.lecture_hrs, c.tutorial_hrs, c.practical_hrs, c.activity_hrs, COALESCE(c.` + "`tw/sl`" + `, 0) as tw_sl,
		       COALESCE(c.theory_total_hrs, 0), COALESCE(c.tutorial_total_hrs, 0), COALESCE(c.practical_total_hrs, 0), COALESCE(c.activity_total_hrs, 0), COALESCE(c.total_hrs, 0),
		       c.cia_marks, c.see_marks, c.total_marks,
		       rc.id as reg_course_id,
		       COALESCE(rc.count_towards_limit, 1) as count_towards_limit
		FROM courses c
		INNER JOIN curriculum_courses rc ON c.id = rc.course_id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE rc.curriculum_id = ? AND rc.semester_id = ? AND c.status = 1
		ORDER BY c.course_code
	`

	rows, err := db.DB.Query(query, curriculumID, semesterID)
	if err != nil {
		log.Println("Error querying courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch courses"})
		return
	}
	defer rows.Close()

	var courses []models.CourseWithDetails = make([]models.CourseWithDetails, 0)
	for rows.Next() {
		var course models.CourseWithDetails
		var countTowardsLimitInt int
		err := rows.Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &course.CourseType, &course.Category, &course.Credit,
			&course.LectureHrs, &course.TutorialHrs, &course.PracticalHrs, &course.ActivityHrs, &course.TwSlHrs,
			&course.TheoryTotalHrs, &course.TutorialTotalHrs, &course.PracticalTotalHrs, &course.ActivityTotalHrs, &course.TotalHrs,
			&course.CIAMarks, &course.SEEMarks, &course.TotalMarks,
			&course.RegCourseID, &countTowardsLimitInt)
		if err != nil {
			log.Println("Error scanning course:", err)
			continue
		}
		// Convert int to bool pointer
		countTowardsLimit := countTowardsLimitInt == 1
		course.CountTowardsLimit = &countTowardsLimit
		course.CurriculumTemplate = curriculumTemplate
		courses = append(courses, course)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(courses)
}

// AddCourseToSemester adds a new course and links it to a semester
func AddCourseToSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	semesterID, err := strconv.Atoi(vars["semId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid semester ID"})
		return
	}

	course, err := decodeCourseRequest(r)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Get curriculum's max_credits
	var maxCredits int
	err = db.DB.QueryRow("SELECT max_credits FROM curriculum WHERE id = ? AND status = 1", curriculumID).Scan(&maxCredits)
	if err != nil {
		log.Println("Error fetching curriculum max_credits:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate credits"})
		return
	}

	// Check if the current semester is a semester card type
	var cardType string
	err = db.DB.QueryRow("SELECT COALESCE(card_type, 'semester') FROM normal_cards WHERE id = ? AND (status = 1 OR status IS NULL)", semesterID).Scan(&cardType)
	if err != nil {
		log.Println("Error fetching semester card type:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate semester"})
		return
	}

	// Only validate credit limit if this is a semester card and count_towards_limit is true
	countTowardsLimit := true
	if course.CountTowardsLimit != nil {
		countTowardsLimit = *course.CountTowardsLimit
	}

	if cardType == "semester" && countTowardsLimit {
		// Calculate current total credits for this curriculum (across all semester cards with count_towards_limit=true)
		var currentCredits sql.NullInt64
		creditQuery := `SELECT SUM(c.credit) FROM courses c 
		                INNER JOIN curriculum_courses cc ON c.id = cc.course_id 
		                INNER JOIN normal_cards nc ON cc.semester_id = nc.id
		                WHERE cc.curriculum_id = ? 
		                AND nc.card_type = 'semester'
		                AND (cc.count_towards_limit IS NULL OR cc.count_towards_limit = 1)`
		err = db.DB.QueryRow(creditQuery, curriculumID).Scan(&currentCredits)
		if err != nil {
			log.Println("Error calculating current credits:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate credits"})
			return
		}

		totalCredits := 0
		if currentCredits.Valid {
			totalCredits = int(currentCredits.Int64)
		}

		// Check if adding this course would exceed curriculum's max_credits
		if totalCredits+course.Credit > maxCredits {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Total credits exceed the curriculum's maximum allowed limit"})
			return
		}
	}
	// Get curriculum template
	var curriculumTemplate string
	err = db.DB.QueryRow("SELECT curriculum_template FROM curriculum WHERE id = ? AND status = 1", curriculumID).Scan(&curriculumTemplate)
	if err != nil {
		log.Println("Error fetching curriculum template:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch curriculum template"})
		return
	}

	// Use total hours from frontend (already calculated)
	theoryTotal := course.TheoryTotalHrs
	tutorialTotal := course.TutorialTotalHrs
	practicalTotal := course.PracticalTotalHrs
	activityTotal := course.ActivityTotalHrs

	// First check: prevent duplicate course codes in the same curriculum (active courses only)
	// Skip this check if course_code is "NA"
	if course.CourseCode != "NA" {
		var duplicateCheck int
		duplicateQuery := `SELECT c.id FROM courses c 
		                   INNER JOIN curriculum_courses cc ON c.id = cc.course_id 
		                   WHERE c.course_code = ? AND cc.curriculum_id = ? AND c.status = 1`
		duplicateErr := db.DB.QueryRow(duplicateQuery, course.CourseCode, curriculumID).Scan(&duplicateCheck)
		if duplicateErr == nil {
			// Course with this code already exists in curriculum (active)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "A course with code " + course.CourseCode + " already exists in this curriculum"})
			return
		} else if duplicateErr != sql.ErrNoRows {
			log.Println("Error checking duplicate course code:", duplicateErr)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate course"})
			return
		}
	}

	// Check if course code already exists in this curriculum
	// Skip this check if course_code is "NA" to allow multiple NA courses
	var existingCourseID int
	var courseID int
	var wasReused bool

	courseTypeID, err := resolveCourseTypeID(course.CourseType)
	if err != nil {
		log.Println("Error resolving course_type:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course type"})
		return
	}

	if course.CourseCode == "NA" {
		// For NA courses, always create a new course entry
		insertCourseQuery := `INSERT INTO courses (course_code, course_name, course_type, category, credit, 
		                      lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, ` + "`tw/sl`" + `,
		                      theory_total_hrs, tutorial_total_hrs, practical_total_hrs, activity_total_hrs,
	                      cia_marks, see_marks, status) 
	                      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`
		result, err := db.DB.Exec(insertCourseQuery, course.CourseCode, course.CourseName, courseTypeID, course.Category, course.Credit,
			course.LectureHrs, course.TutorialHrs, course.PracticalHrs, course.ActivityHrs, course.TwSlHrs,
			theoryTotal, tutorialTotal, practicalTotal, activityTotal,
			course.CIAMarks, course.SEEMarks)
		if err != nil {
			log.Println("Error inserting NA course:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
			return
		}
		id, _ := result.LastInsertId()
		courseID = int(id)
		wasReused = false
	} else {
		// For non-NA courses, check for existing courses (only active ones)
		checkQuery := `SELECT c.id FROM courses c 
		               INNER JOIN curriculum_courses cc ON c.id = cc.course_id 
		               WHERE c.course_code = ? AND cc.curriculum_id = ? AND c.status = 1`
		err = db.DB.QueryRow(checkQuery, course.CourseCode, curriculumID).Scan(&existingCourseID)

		if err == sql.ErrNoRows {
			// Course code doesn't exist in this curriculum, check if it exists globally (only active ones)
			var globalCourseID int
			globalCheckQuery := "SELECT id FROM courses WHERE course_code = ? AND status = 1"
			globalErr := db.DB.QueryRow(globalCheckQuery, course.CourseCode).Scan(&globalCourseID)

			if globalErr == sql.ErrNoRows {
				// Course doesn't exist globally - create new course
				insertCourseQuery := `INSERT INTO courses (course_code, course_name, course_type, category, credit, 
				                      lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, ` + "`tw/sl`" + `,
				                      theory_total_hrs, tutorial_total_hrs, practical_total_hrs, activity_total_hrs,
			                      cia_marks, see_marks, status) 
			                      VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`
				result, err := db.DB.Exec(insertCourseQuery, course.CourseCode, course.CourseName, courseTypeID, course.Category, course.Credit,
					course.LectureHrs, course.TutorialHrs, course.PracticalHrs, course.ActivityHrs, course.TwSlHrs,
					theoryTotal, tutorialTotal, practicalTotal, activityTotal,
					course.CIAMarks, course.SEEMarks)
				if err != nil {
					log.Println("Error inserting course:", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
					return
				}
				id, _ := result.LastInsertId()
				courseID = int(id)
				wasReused = false
			} else if globalErr != nil {
				log.Println("Error checking global course:", globalErr)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
				return
			} else {
				// Course exists globally but not in this curriculum - reuse the existing course
				courseID = globalCourseID
				// Reactivate the course if it was soft-deleted
				db.DB.Exec("UPDATE courses SET status = 1 WHERE id = ?", globalCourseID)
				wasReused = true
				log.Printf("Reusing existing course %s (ID: %d) for curriculum %d", course.CourseCode, globalCourseID, curriculumID)
			}
		} else if err != nil {
			log.Println("Error checking existing course in curriculum:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
			return
		} else {
			// Course code already exists in this curriculum - return error
			log.Printf("Course with code %s already exists in curriculum %d (ID: %d)", course.CourseCode, curriculumID, existingCourseID)
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "A course with this course code already exists in this curriculum. Please use a different course code."})
			return
		}
	}

	// Link course to curriculum and semester (countTowardsLimit already set above)
	linkQuery := "INSERT INTO curriculum_courses (curriculum_id, semester_id, course_id, count_towards_limit) VALUES (?, ?, ?, ?)"
	result, err := db.DB.Exec(linkQuery, curriculumID, semesterID, courseID, countTowardsLimit)
	if err != nil {
		log.Println("Error linking course:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to add course to semester"})
		return
	}

	regCourseID, _ := result.LastInsertId()
	course.CourseID = courseID

	// Fetch the complete course details including computed fields
	var fullCourse models.CourseWithDetails
	fetchQuery := `SELECT id, course_code, course_name, course_type, category, credit, 
	               lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, COALESCE(` + "`tw/sl`" + `, 0) as tw_sl,
	               COALESCE(theory_total_hrs, 0), COALESCE(tutorial_total_hrs, 0), COALESCE(practical_total_hrs, 0), COALESCE(activity_total_hrs, 0), COALESCE(total_hrs, 0),
	               cia_marks, see_marks, total_marks 
	               FROM courses WHERE id = ?`
	err = db.DB.QueryRow(fetchQuery, courseID).Scan(&fullCourse.CourseID, &fullCourse.CourseCode, &fullCourse.CourseName,
		&fullCourse.CourseType, &fullCourse.Category, &fullCourse.Credit,
		&fullCourse.LectureHrs, &fullCourse.TutorialHrs, &fullCourse.PracticalHrs, &fullCourse.ActivityHrs, &fullCourse.TwSlHrs,
		&fullCourse.TheoryTotalHrs, &fullCourse.TutorialTotalHrs, &fullCourse.PracticalTotalHrs, &fullCourse.ActivityTotalHrs, &fullCourse.TotalHrs,
		&fullCourse.CIAMarks, &fullCourse.SEEMarks, &fullCourse.TotalMarks)
	if err != nil {
		log.Println("Error fetching full course details:", err)
		// Fallback to sent values
		fullCourse = models.CourseWithDetails{
			CourseID:     courseID,
			CourseCode:   course.CourseCode,
			CourseName:   course.CourseName,
			CourseType:   course.CourseType,
			Category:     course.Category,
			Credit:       course.Credit,
			LectureHrs:   course.LectureHrs,
			TutorialHrs:  course.TutorialHrs,
			PracticalHrs: course.PracticalHrs,
			ActivityHrs:  course.ActivityHrs,
			CIAMarks:     course.CIAMarks,
			SEEMarks:     course.SEEMarks,
			TotalMarks:   course.CIAMarks + course.SEEMarks,
		}
	}
	fullCourse.RegCourseID = int(regCourseID)

	// Log the activity
	LogCurriculumActivity(curriculumID, "Course Added",
		"Added course "+course.CourseCode+" - "+course.CourseName+" to Semester "+strconv.Itoa(semesterID), "System")

	w.WriteHeader(http.StatusCreated)

	// Return response with optional message about reused course
	response := map[string]interface{}{
		"course": fullCourse,
	}
	if wasReused {
		response["message"] = fmt.Sprintf("Course %s already exists in another curriculum and has been reused. The course details from the database have been applied.", course.CourseCode)
	}
	json.NewEncoder(w).Encode(response)
}

// RemoveCourseFromSemester removes a course from a semester
func RemoveCourseFromSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	semesterID, err := strconv.Atoi(vars["semId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid semester ID"})
		return
	}

	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID"})
		return
	}

	// Check if mapping exists
	var mappingExists bool
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM curriculum_courses WHERE curriculum_id = ? AND semester_id = ? AND course_id = ?)", curriculumID, semesterID, courseID).Scan(&mappingExists)
	if err != nil {
		log.Println("Error checking course mapping:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check course mapping"})
		return
	}

	if !mappingExists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course not found in semester"})
		return
	}

	// Get course name for logging
	var courseName string
	db.DB.QueryRow("SELECT course_name FROM courses WHERE id = ?", courseID).Scan(&courseName)

	// Soft-delete the course (keep the mapping intact)
	result, err := db.DB.Exec("UPDATE courses SET status = 0 WHERE id = ? AND status = 1", courseID)
	if err != nil {
		log.Println("Error soft-deleting course:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to remove course"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Course might already be soft-deleted, which is fine
		log.Printf("Course %d was already inactive", courseID)
	}

	// Cascade soft-delete to all course children
	if err := cascadeSoftDeleteCourse(courseID, nil); err != nil {
		log.Println("Error cascading course soft-delete:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to course children"})
		return
	}

	// Log the activity
	LogCurriculumActivity(curriculumID, "Course Removed",
		"Removed course "+courseName+" from Semester "+strconv.Itoa(semesterID), "System")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Course removed successfully"})
}
