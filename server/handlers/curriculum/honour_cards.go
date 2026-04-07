package curriculum

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"server/db"
	"server/models"

	"github.com/gorilla/mux"
)

// GetHonourCards retrieves all honour cards for a regulation
func GetHonourCards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	curriculumID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum ID"})
		return
	}

	query := "SELECT id, curriculum_id, title FROM honour_cards WHERE curriculum_id = ? AND status = 1 ORDER BY id"
	rows, err := db.DB.Query(query, curriculumID)
	if err != nil {
		log.Println("Error querying honour cards:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch honour cards"})
		return
	}
	defer rows.Close()

	var honourCards []models.HonourCardWithVerticals = make([]models.HonourCardWithVerticals, 0)
	for rows.Next() {
		var card models.HonourCardWithVerticals
		err := rows.Scan(&card.ID, &card.CurriculumID, &card.Title)
		if err != nil {
			log.Println("Error scanning honour card:", err)
			continue
		}

		// Fetch verticals for this honour card
		card.Verticals = fetchVerticalsForCard(card.ID)
		honourCards = append(honourCards, card)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(honourCards)
}

// fetchVerticalsForCard retrieves all verticals and their courses for a given honour card
func fetchVerticalsForCard(honourCardID int) []models.HonourVerticalWithCourses {
	query := "SELECT id, honour_card_id, name FROM honour_verticals WHERE honour_card_id = ? AND status = 1 ORDER BY id"
	rows, err := db.DB.Query(query, honourCardID)
	if err != nil {
		log.Println("Error querying verticals:", err)
		return []models.HonourVerticalWithCourses{}
	}
	defer rows.Close()

	var verticals []models.HonourVerticalWithCourses = make([]models.HonourVerticalWithCourses, 0)
	for rows.Next() {
		var vertical models.HonourVerticalWithCourses
		err := rows.Scan(&vertical.ID, &vertical.HonourCardID, &vertical.Name)
		if err != nil {
			log.Println("Error scanning vertical:", err)
			continue
		}

		// Fetch courses for this vertical
		vertical.Courses = fetchCoursesForVertical(vertical.ID)
		verticals = append(verticals, vertical)
	}

	return verticals
}

// fetchCoursesForVertical retrieves all courses for a given vertical
func fetchCoursesForVertical(verticalID int) []models.CourseWithDetails {
	query := `SELECT c.id, c.course_code, c.course_name, ct.course_type, c.category, 
	       c.credit, c.lecture_hrs, c.tutorial_hrs, c.practical_hrs, c.activity_hrs, COALESCE(c.` + "`tw/sl`" + `, 0) as tw_sl,
	       COALESCE(c.theory_total_hrs, 0), COALESCE(c.tutorial_total_hrs, 0), COALESCE(c.practical_total_hrs, 0), COALESCE(c.activity_total_hrs, 0), COALESCE(c.total_hrs, 0),
	       c.cia_marks, c.see_marks, c.total_marks,
	       hvc.id as honour_vertical_course_id
		FROM courses c
		INNER JOIN honour_vertical_courses hvc ON c.id = hvc.course_id
		LEFT JOIN course_type ct ON c.course_type = ct.id
		WHERE hvc.honour_vertical_id = ? AND hvc.status = 1 AND c.status = 1
		ORDER BY c.course_code`
	rows, err := db.DB.Query(query, verticalID)
	if err != nil {
		log.Println("Error querying courses for vertical:", err)
		return []models.CourseWithDetails{}
	}
	defer rows.Close()

	var courses []models.CourseWithDetails = make([]models.CourseWithDetails, 0)
	for rows.Next() {
		var course models.CourseWithDetails
		err := rows.Scan(
			&course.CourseID, &course.CourseCode, &course.CourseName, &course.CourseType,
			&course.Category, &course.Credit, &course.LectureHrs, &course.TutorialHrs, &course.PracticalHrs, &course.ActivityHrs, &course.TwSlHrs,
			&course.TheoryTotalHrs, &course.TutorialTotalHrs, &course.PracticalTotalHrs, &course.ActivityTotalHrs, &course.TotalHrs,
			&course.CIAMarks, &course.SEEMarks, &course.TotalMarks,
			&course.RegCourseID,
		)
		if err != nil {
			log.Println("Error scanning course:", err)
			continue
		}
		courses = append(courses, course)
	}

	return courses
}

// CreateHonourCard creates a new honour card for a regulation
func CreateHonourCard(w http.ResponseWriter, r *http.Request) {
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

	var card models.HonourCard
	err = json.NewDecoder(r.Body).Decode(&card)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	card.CurriculumID = curriculumID

	query := "INSERT INTO honour_cards (curriculum_id, title) VALUES (?, ?)"
	result, err := db.DB.Exec(query, card.CurriculumID, card.Title)
	if err != nil {
		log.Println("Error inserting honour card:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create honour card"})
		return
	}

	id, _ := result.LastInsertId()
	card.ID = int(id)

	// Log the activity
	LogCurriculumActivity(curriculumID, "Honour Card Added",
		"Added Honour Card: "+card.Title, "System")

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(card)
}

// CreateHonourVertical creates a new vertical within an honour card
func CreateHonourVertical(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	honourCardID, err := strconv.Atoi(vars["cardId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid honour card ID"})
		return
	}

	var vertical models.HonourVertical
	err = json.NewDecoder(r.Body).Decode(&vertical)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	vertical.HonourCardID = honourCardID

	query := "INSERT INTO honour_verticals (honour_card_id, name) VALUES (?, ?)"
	result, err := db.DB.Exec(query, vertical.HonourCardID, vertical.Name)
	if err != nil {
		log.Println("Error inserting vertical:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create vertical"})
		return
	}

	id, _ := result.LastInsertId()
	vertical.ID = int(id)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vertical)
}

// AddCourseToVertical adds a course to a vertical
func AddCourseToVertical(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	verticalID, err := strconv.Atoi(vars["verticalId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid vertical ID"})
		return
	}

	// Support two ways of adding a course:
	// 1) By existing course_id (legacy behaviour)
	// 2) By full course details (same as normal card flow)
	var payload struct {
		CourseID           *int   `json:"course_id,omitempty"`
		CourseCode         string `json:"course_code,omitempty"`
		CourseName         string `json:"course_name,omitempty"`
		CourseType         string `json:"course_type,omitempty"`
		ExperimentCountTWL int    `json:"experiment_count_theorywithlab,omitempty"`
		Category           string `json:"category,omitempty"`
		Credit             int    `json:"credit,omitempty"`
		LectureHrs         int    `json:"lecture_hrs,omitempty"`
		TutorialHrs        int    `json:"tutorial_hrs,omitempty"`
		PracticalHrs       int    `json:"practical_hrs,omitempty"`
		ActivityHrs        int    `json:"activity_hrs,omitempty"`
		TwSlHrs            int    `json:"tw_sl_hrs,omitempty"`
		TheoryTotalHrs     int    `json:"theory_total_hrs,omitempty"`
		TutorialTotalHrs   int    `json:"tutorial_total_hrs,omitempty"`
		PracticalTotalHrs  int    `json:"practical_total_hrs,omitempty"`
		ActivityTotalHrs   int    `json:"activity_total_hrs,omitempty"`
		TotalHrs           int    `json:"total_hrs,omitempty"`
		CIAMarks           int    `json:"cia_marks,omitempty"`
		SEEMarks           int    `json:"see_marks,omitempty"`
		TotalMarks         int    `json:"total_marks,omitempty"`
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Get curriculum ID and template from vertical
	var curriculumID int
	var curriculumTemplate string
	err = db.DB.QueryRow(`
		SELECT hc.curriculum_id, c.curriculum_template 
		FROM honour_verticals hv
		INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
		INNER JOIN curriculum c ON hc.curriculum_id = c.id
		WHERE hv.id = ?`, verticalID).Scan(&curriculumID, &curriculumTemplate)
	if err != nil {
		log.Println("Error fetching curriculum template for vertical:", err)
		// Don't fail, just use default calculation
		curriculumTemplate = "2022"
	}

	var courseID int
	var wasReused bool

	if payload.CourseID != nil && *payload.CourseID > 0 {
		// Legacy path: link an existing course by ID
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM courses WHERE id = ?)"
		err = db.DB.QueryRow(checkQuery, *payload.CourseID).Scan(&exists)
		if err != nil || !exists {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
			return
		}
		courseID = *payload.CourseID
	} else {
		// New path: create or reuse a course based on course_code (similar to AddCourseToSemester)
		if payload.CourseCode == "" || payload.CourseName == "" || payload.CourseType == "" || payload.Category == "" || payload.Credit < 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing required course fields"})
			return
		}

		// Use total hours from frontend (already calculated)
		theoryTotal := payload.TheoryTotalHrs
		tutorialTotal := payload.TutorialTotalHrs
		practicalTotal := payload.PracticalTotalHrs
		activityTotal := payload.ActivityTotalHrs

		courseTypeID, err := resolveCourseTypeID(payload.CourseType)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course type"})
			return
		}

		// First check: prevent duplicate course codes in the same curriculum (active courses only)
		// Check both curriculum_courses (normal cards) and honour_vertical_courses (honour cards)
		// Skip this check if course_code is "NA" (case-insensitive, trimmed)
		trimmedCourseCode := strings.TrimSpace(strings.ToUpper(payload.CourseCode))
		if trimmedCourseCode != "NA" {
			var duplicateCheck int
			duplicateQuery := `SELECT c.id FROM courses c 
			                   INNER JOIN curriculum_courses cc ON c.id = cc.course_id 
			                   WHERE c.course_code = ? AND cc.curriculum_id = ? AND c.status = 1`
			duplicateErr := db.DB.QueryRow(duplicateQuery, payload.CourseCode, curriculumID).Scan(&duplicateCheck)
			if duplicateErr == nil {
				// Course with this code already exists in curriculum (active)
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "A course with this course code already exists in this curriculum. Please use a different course code."})
				return
			} else if duplicateErr != sql.ErrNoRows {
				log.Println("Error checking duplicate course code:", duplicateErr)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate course"})
				return
			}

			// Also check in honour vertical courses for this curriculum
			var honourDuplicateCheck int
			honourDuplicateQuery := `SELECT c.id FROM courses c 
			                         INNER JOIN honour_vertical_courses hvc ON c.id = hvc.course_id 
			                         INNER JOIN honour_verticals hv ON hvc.honour_vertical_id = hv.id
			                         INNER JOIN honour_cards hc ON hv.honour_card_id = hc.id
			                         WHERE c.course_code = ? AND hc.curriculum_id = ? AND c.status = 1 AND hvc.status = 1`
			honourDuplicateErr := db.DB.QueryRow(honourDuplicateQuery, payload.CourseCode, curriculumID).Scan(&honourDuplicateCheck)
			if honourDuplicateErr == nil {
				// Course with this code already exists in honour verticals
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "A course with this course code already exists in this curriculum. Please use a different course code."})
				return
			} else if honourDuplicateErr != sql.ErrNoRows {
				log.Println("Error checking duplicate honour course code:", honourDuplicateErr)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate course"})
				return
			}
		}

		// Check if course with same code AND name already exists in this curriculum (only active ones)
		var existingCourseID int
		checkQuery := `SELECT c.id FROM courses c 
		               INNER JOIN curriculum_courses cc ON c.id = cc.course_id 
		               WHERE c.course_code = ? AND c.course_name = ? AND cc.curriculum_id = ? AND c.status = 1`
		err = db.DB.QueryRow(checkQuery, payload.CourseCode, payload.CourseName, curriculumID).Scan(&existingCourseID)

		if err == sql.ErrNoRows {
			// Course code doesn't exist in this curriculum, check if it exists globally with same name (only active ones)
			var globalCourseID int
			globalCheckQuery := "SELECT id FROM courses WHERE course_code = ? AND course_name = ? AND status = 1"
			globalErr := db.DB.QueryRow(globalCheckQuery, payload.CourseCode, payload.CourseName).Scan(&globalCourseID)

			if globalErr == sql.ErrNoRows {
				// Course doesn't exist globally with same code+name - create new course
				insertCourseQuery := `INSERT INTO courses (course_code, course_name, course_type, category, credit,
					experiment_count_theorywithlab,
					lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, ` + "`tw/sl`" + `,
					theory_total_hrs, tutorial_total_hrs, practical_total_hrs, activity_total_hrs, 
					cia_marks, see_marks, status) 
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`
				result, err := db.DB.Exec(insertCourseQuery,
					payload.CourseCode,
					payload.CourseName,
					courseTypeID,
					payload.Category,
					payload.Credit,
					payload.ExperimentCountTWL,
					payload.LectureHrs,
					payload.TutorialHrs,
					payload.PracticalHrs,
					payload.ActivityHrs,
					payload.TwSlHrs,
					theoryTotal,
					tutorialTotal,
					practicalTotal,
					activityTotal,
					payload.CIAMarks,
					payload.SEEMarks,
				)
				if err != nil {
					log.Println("Error inserting course for honour vertical:", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
					return
				}
				id, _ := result.LastInsertId()
				courseID = int(id)
				log.Printf("Created new course %s (ID: %d) for honour vertical", payload.CourseCode, courseID)
			} else if globalErr != nil {
				log.Println("Error checking global course for honour vertical:", globalErr)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
				return
			} else {
				// Course exists globally with same code+name - reuse the existing course
				courseID = globalCourseID
				wasReused = true
				// Reactivate the course if it was soft-deleted
				db.DB.Exec("UPDATE courses SET status = 1 WHERE id = ?", globalCourseID)
				log.Printf("Reusing existing course %s (ID: %d) for honour vertical curriculum %d", payload.CourseCode, globalCourseID, curriculumID)
			}
		} else if err != nil {
			log.Println("Error checking existing course in curriculum for honour vertical:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create course"})
			return
		} else {
			// Course code already exists in this curriculum - use the existing one
			courseID = existingCourseID
			wasReused = true
			log.Printf("Reusing existing course %s (ID: %d) in curriculum %d for honour vertical", payload.CourseCode, existingCourseID, curriculumID)
		}
	}

	// Note: Honour courses are NOT added to curriculum_courses table
	// They are tracked via honour_cards -> honour_verticals -> honour_vertical_courses
	// This is different from semester courses which use curriculum_courses

	// Insert into honour_vertical_courses mapping table
	query := `INSERT INTO honour_vertical_courses (honour_vertical_id, course_id, status)
	          VALUES (?, ?, 1)
	          ON DUPLICATE KEY UPDATE status = 1`
	_, err = db.DB.Exec(query, verticalID, courseID)
	if err != nil {
		log.Println("Error adding course to vertical:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to add course to vertical"})
		return
	}

	// Fetch the complete course details including computed fields (matching normal card behavior)
	var fullCourse models.CourseWithDetails
	fetchQuery := `SELECT id, course_code, course_name, course_type, category, credit, 
	               COALESCE(experiment_count_theorywithlab, 0),
	               lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, COALESCE(` + "`tw/sl`" + `, 0) as tw_sl,
	               COALESCE(theory_total_hrs, 0), COALESCE(tutorial_total_hrs, 0), COALESCE(practical_total_hrs, 0), COALESCE(activity_total_hrs, 0), COALESCE(total_hrs, 0),
	               cia_marks, see_marks, total_marks 
	               FROM courses WHERE id = ?`
	err = db.DB.QueryRow(fetchQuery, courseID).Scan(&fullCourse.CourseID, &fullCourse.CourseCode, &fullCourse.CourseName,
		&fullCourse.CourseType, &fullCourse.Category, &fullCourse.Credit, &fullCourse.ExperimentCountTWL,
		&fullCourse.LectureHrs, &fullCourse.TutorialHrs, &fullCourse.PracticalHrs, &fullCourse.ActivityHrs, &fullCourse.TwSlHrs,
		&fullCourse.TheoryTotalHrs, &fullCourse.TutorialTotalHrs, &fullCourse.PracticalTotalHrs, &fullCourse.ActivityTotalHrs, &fullCourse.TotalHrs,
		&fullCourse.CIAMarks, &fullCourse.SEEMarks, &fullCourse.TotalMarks)
	if err != nil {
		log.Println("Error fetching full course details:", err)
		// Fallback to sent values
		fullCourse = models.CourseWithDetails{
			CourseID:           courseID,
			CourseCode:         payload.CourseCode,
			CourseName:         payload.CourseName,
			CourseType:         payload.CourseType,
			ExperimentCountTWL: payload.ExperimentCountTWL,
			Category:           payload.Category,
			Credit:             payload.Credit,
			LectureHrs:         payload.LectureHrs,
			TutorialHrs:        payload.TutorialHrs,
			PracticalHrs:       payload.PracticalHrs,
			ActivityHrs:        payload.ActivityHrs,
			TwSlHrs:            payload.TwSlHrs,
			TheoryTotalHrs:     payload.TheoryTotalHrs,
			TutorialTotalHrs:   payload.TutorialTotalHrs,
			PracticalTotalHrs:  payload.PracticalTotalHrs,
			ActivityTotalHrs:   payload.ActivityTotalHrs,
			TotalHrs:           payload.TotalHrs,
			CIAMarks:           payload.CIAMarks,
			SEEMarks:           payload.SEEMarks,
			TotalMarks:         payload.TotalMarks,
		}
	}
	fullCourse.CurriculumTemplate = curriculumTemplate

	w.WriteHeader(http.StatusCreated)

	// Return course with optional message if it was reused
	if wasReused {
		response := map[string]interface{}{
			"course_id":                      fullCourse.CourseID,
			"course_code":                    fullCourse.CourseCode,
			"course_name":                    fullCourse.CourseName,
			"course_type":                    fullCourse.CourseType,
			"experiment_count_theorywithlab": fullCourse.ExperimentCountTWL,
			"category":                       fullCourse.Category,
			"credit":                         fullCourse.Credit,
			"lecture_hrs":                    fullCourse.LectureHrs,
			"tutorial_hrs":                   fullCourse.TutorialHrs,
			"practical_hrs":                  fullCourse.PracticalHrs,
			"activity_hrs":                   fullCourse.ActivityHrs,
			"tw_sl_hrs":                      fullCourse.TwSlHrs,
			"theory_total_hrs":               fullCourse.TheoryTotalHrs,
			"tutorial_total_hrs":             fullCourse.TutorialTotalHrs,
			"practical_total_hrs":            fullCourse.PracticalTotalHrs,
			"activity_total_hrs":             fullCourse.ActivityTotalHrs,
			"total_hrs":                      fullCourse.TotalHrs,
			"cia_marks":                      fullCourse.CIAMarks,
			"see_marks":                      fullCourse.SEEMarks,
			"total_marks":                    fullCourse.TotalMarks,
			"curriculum_template":            fullCourse.CurriculumTemplate,
			"message":                        "Course already exists in another curriculum and has been reused",
		}
		json.NewEncoder(w).Encode(response)
	} else {
		json.NewEncoder(w).Encode(fullCourse)
	}
}

// RemoveCourseFromVertical removes a course from a vertical
func RemoveCourseFromVertical(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	verticalID, err := strconv.Atoi(vars["verticalId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid vertical ID"})
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
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM honour_vertical_courses WHERE honour_vertical_id = ? AND course_id = ? AND status = 1)", verticalID, courseID).Scan(&mappingExists)
	if err != nil {
		log.Println("Error checking course mapping:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check course mapping"})
		return
	}

	if !mappingExists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course not found in vertical"})
		return
	}

	// Soft-delete honour_vertical_courses mapping (keep the record)
	_, err = db.DB.Exec("UPDATE honour_vertical_courses SET status = 0 WHERE honour_vertical_id = ? AND course_id = ? AND status = 1", verticalID, courseID)
	if err != nil {
		log.Println("Error soft-deleting vertical course mapping:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to remove course"})
		return
	}

	// Soft-delete the course itself
	_, err = db.DB.Exec("UPDATE courses SET status = 0 WHERE id = ? AND status = 1", courseID)
	if err != nil {
		log.Println("Error soft-deleting course:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete course"})
		return
	}

	// Cascade soft-delete to all course children
	if err := cascadeSoftDeleteCourse(courseID, nil); err != nil {
		log.Println("Error cascading course soft-delete:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to course children"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Course removed successfully"})
}

// DeleteHonourVertical deletes a vertical
func DeleteHonourVertical(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	verticalID, err := strconv.Atoi(vars["verticalId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid vertical ID"})
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

	// Soft delete the honour vertical
	query := "UPDATE honour_verticals SET status = 0 WHERE id = ? AND status = 1"
	result, err := tx.Exec(query, verticalID)
	if err != nil {
		log.Println("Error deleting vertical:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete vertical"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Vertical not found"})
		return
	}

	// Soft delete honour_vertical_courses for this vertical
	_, err = tx.Exec("UPDATE honour_vertical_courses SET status = 0 WHERE honour_vertical_id = ? AND status = 1", verticalID)
	if err != nil {
		log.Println("Error soft-deleting honour_vertical_courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to honour_vertical_courses"})
		return
	}

	// Get course IDs and soft delete courses + their children
	rows, err := tx.Query("SELECT course_id FROM honour_vertical_courses WHERE honour_vertical_id = ?", verticalID)
	if err != nil {
		log.Println("Error fetching vertical courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch vertical courses"})
		return
	}

	courseIDs := make([]int, 0)
	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			rows.Close()
			log.Println("Error scanning course ID:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch vertical courses"})
			return
		}
		courseIDs = append(courseIDs, courseID)
	}
	rows.Close()

	for _, courseID := range courseIDs {
		_, err = tx.Exec("UPDATE courses SET status = 0 WHERE id = ? AND status = 1", courseID)
		if err != nil {
			log.Println("Error soft-deleting course:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to courses"})
			return
		}
		// Cascade soft-delete to course children (DeleteHonourVertical)
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
	json.NewEncoder(w).Encode(map[string]string{"message": "Vertical deleted successfully"})
}

// DeleteHonourCard deletes an honour card and all its verticals
func DeleteHonourCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	cardID, err := strconv.Atoi(vars["cardId"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid card ID"})
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

	// Soft delete the honour card
	query := "UPDATE honour_cards SET status = 0 WHERE id = ? AND status = 1"
	result, err := tx.Exec(query, cardID)
	if err != nil {
		log.Println("Error deleting honour card:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete honour card"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Honour card not found"})
		return
	}

	// Soft delete honour_verticals for this card
	_, err = tx.Exec("UPDATE honour_verticals SET status = 0 WHERE honour_card_id = ? AND status = 1", cardID)
	if err != nil {
		log.Println("Error soft-deleting honour_verticals:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to honour_verticals"})
		return
	}

	// Soft delete honour_vertical_courses for all verticals in this card
	_, err = tx.Exec(`
		UPDATE honour_vertical_courses 
		SET status = 0 
		WHERE honour_vertical_id IN (
			SELECT id FROM honour_verticals WHERE honour_card_id = ?
		) AND status = 1
	`, cardID)
	if err != nil {
		log.Println("Error soft-deleting honour_vertical_courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to honour_vertical_courses"})
		return
	}

	// Get all course IDs from verticals in this card, soft-delete courses + their children
	rows, err := tx.Query(`
		SELECT DISTINCT course_id FROM honour_vertical_courses 
		WHERE honour_vertical_id IN (
			SELECT id FROM honour_verticals WHERE honour_card_id = ?
		)
	`, cardID)
	if err != nil {
		log.Println("Error fetching card courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch card courses"})
		return
	}

	courseIDs := make([]int, 0)
	for rows.Next() {
		var courseID int
		if err := rows.Scan(&courseID); err != nil {
			rows.Close()
			log.Println("Error scanning course ID:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch card courses"})
			return
		}
		courseIDs = append(courseIDs, courseID)
	}
	rows.Close()

	for _, courseID := range courseIDs {
		_, err = tx.Exec("UPDATE courses SET status = 0 WHERE id = ? AND status = 1", courseID)
		if err != nil {
			log.Println("Error soft-deleting course:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to cascade delete to courses"})
			return
		}
		// Cascade soft-delete to course children (DeleteHonourCard)
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
	json.NewEncoder(w).Encode(map[string]string{"message": "Honour card deleted successfully"})
}
