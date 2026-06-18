package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"server/db"
	"server/models"

	"github.com/gorilla/mux"
)

// UpdateCurriculum updates curriculum name and max_credits
func UpdateCurriculum(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPut {
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

	var updateData struct {
		Name               string `json:"name"`
		AcademicYear       string `json:"academic_year"`
		MaxCredits         int    `json:"max_credits"`
		CurriculumTemplate string `json:"curriculum_template"`
	}

	err = json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Fetch old values for diff
	var oldName string
	var oldAcademicYear string
	var oldMaxCredits int
	var oldTemplate string
	err = db.DB.QueryRow("SELECT name, academic_year, max_credits, curriculum_template FROM curriculum WHERE id = ?", curriculumID).Scan(&oldName, &oldAcademicYear, &oldMaxCredits, &oldTemplate)
	if err != nil {
		log.Println("Error fetching old curriculum data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch curriculum data"})
		return
	}

	// Update curriculum
	if updateData.CurriculumTemplate == "" {
		updateData.CurriculumTemplate = oldTemplate
	}

	// Prevent switching template when courses already exist
	if oldTemplate != "" && oldTemplate != updateData.CurriculumTemplate {
		var courseCount int
		_ = db.DB.QueryRow("SELECT COUNT(*) FROM curriculum_courses WHERE curriculum_id = ?", curriculumID).Scan(&courseCount)
		var honourCourseCount int
		_ = db.DB.QueryRow(`
			SELECT COUNT(*) FROM honour_vertical_courses hvc
			INNER JOIN honour_verticals hv ON hv.id = hvc.honour_vertical_id
			INNER JOIN honour_cards hc ON hc.id = hv.honour_card_id
			WHERE hc.curriculum_id = ?`, curriculumID).Scan(&honourCourseCount)

		if courseCount+honourCourseCount > 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Cannot change template after courses have been added"})
			return
		}
	}

	query := "UPDATE curriculum SET name = ?, academic_year = ?, max_credits = ?, curriculum_template = ? WHERE id = ?"
	_, err = db.DB.Exec(query, updateData.Name, updateData.AcademicYear, updateData.MaxCredits, updateData.CurriculumTemplate, curriculumID)
	if err != nil {
		log.Println("Error updating curriculum:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update curriculum"})
		return
	}

	// Generate diff and log changes
	diff := make(map[string]map[string]interface{})
	if oldName != updateData.Name {
		diff["name"] = map[string]interface{}{"old": oldName, "new": updateData.Name}
	}
	if oldAcademicYear != updateData.AcademicYear {
		diff["academic_year"] = map[string]interface{}{"old": oldAcademicYear, "new": updateData.AcademicYear}
	}
	if oldMaxCredits != updateData.MaxCredits {
		diff["max_credits"] = map[string]interface{}{"old": oldMaxCredits, "new": updateData.MaxCredits}
	}
	if oldTemplate != updateData.CurriculumTemplate {
		diff["curriculum_template"] = map[string]interface{}{"old": oldTemplate, "new": updateData.CurriculumTemplate}
	}

	if len(diff) > 0 {
		LogCurriculumActivityWithDiff(curriculumID, "Curriculum Updated",
			"Updated curriculum details", "System", diff)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Curriculum updated successfully"})
}

// UpdateSemester updates semester name/number
func UpdateSemester(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPut {
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

	var updateData struct {
		SemesterNumber int `json:"semester_number"`
	}

	err = json.NewDecoder(r.Body).Decode(&updateData)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Fetch old values and curriculum_id for diff
	var oldSemesterNumber int
	var curriculumID int
	var cardType string
	err = db.DB.QueryRow("SELECT semester_number, curriculum_id, card_type FROM normal_cards WHERE id = ?", semesterID).Scan(&oldSemesterNumber, &curriculumID, &cardType)
	if err != nil {
		log.Println("Error fetching old semester data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch semester data"})
		return
	}

	// Check if the new number already exists within the same card type (excluding current card)
	if oldSemesterNumber != updateData.SemesterNumber {
		var existingCount int
		err = db.DB.QueryRow("SELECT COUNT(*) FROM normal_cards WHERE curriculum_id = ? AND semester_number = ? AND id != ? AND card_type = ?",
			curriculumID, updateData.SemesterNumber, semesterID, cardType).Scan(&existingCount)
		if err != nil {
			log.Println("Error checking for duplicate number:", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to validate number"})
			return
		}
		if existingCount > 0 {
			if cardType == "vertical" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Vertical %d already exists in this curriculum", updateData.SemesterNumber)})
			} else {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Semester %d already exists in this curriculum", updateData.SemesterNumber)})
			}
			return
		}
	}

	// Update semester
	query := "UPDATE normal_cards SET semester_number = ? WHERE id = ?"
	_, err = db.DB.Exec(query, updateData.SemesterNumber, semesterID)
	if err != nil {
		log.Println("Error updating semester:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update semester"})
		return
	}

	// Generate diff and log changes
	if oldSemesterNumber != updateData.SemesterNumber {
		diff := map[string]map[string]interface{}{
			"semester_number": {"old": oldSemesterNumber, "new": updateData.SemesterNumber},
		}
		LogCurriculumActivityWithDiff(curriculumID, "Semester Updated",
			fmt.Sprintf("Updated Semester %d to Semester %d", oldSemesterNumber, updateData.SemesterNumber), "System", diff)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Semester updated successfully"})
}

// UpdateCourse updates course details
func UpdateCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID"})
		return
	}

	course, err := decodeCourseRequest(r)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Fetch old course data for diff
	var oldCourse models.Course
	err = db.DB.QueryRow(`SELECT course_code, course_name, course_type, COALESCE(experiment_count_theorywithlab, 0), category, credit, 
		lecture_hrs, tutorial_hrs, practical_hrs, activity_hrs, COALESCE(`+"`tw/sl`"+`, 0) as tw_sl, cia_marks, see_marks 
		FROM courses WHERE id = ?`, courseID).Scan(
		&oldCourse.CourseCode, &oldCourse.CourseName, &oldCourse.CourseType, &oldCourse.ExperimentCountTWL, &oldCourse.Category,
		&oldCourse.Credit, &oldCourse.LectureHrs, &oldCourse.TutorialHrs, &oldCourse.PracticalHrs, &oldCourse.ActivityHrs, &oldCourse.TwSlHrs,
		&oldCourse.CIAMarks, &oldCourse.SEEMarks)
	if err != nil {
		log.Println("Error fetching old course data:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch course data"})
		return
	}

	// Get curriculum_id from curriculum_courses
	var curriculumID int
	err = db.DB.QueryRow("SELECT curriculum_id FROM curriculum_courses WHERE course_id = ? LIMIT 1", courseID).Scan(&curriculumID)
	if err != nil && err != sql.ErrNoRows {
		log.Println("Error fetching curriculum_id:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch curriculum ID"})
		return
	}

	// Get curriculum template
	var curriculumTemplate string
	if curriculumID > 0 {
		err = db.DB.QueryRow("SELECT curriculum_template FROM curriculum WHERE id = ?", curriculumID).Scan(&curriculumTemplate)
		if err != nil {
			log.Println("Error fetching curriculum template:", err)
			// Don't fail, just use default calculation
			curriculumTemplate = "2022"
		}
	} else {
		curriculumTemplate = "2022"
	}

	// Calculate total hours based on template and course type
	var theoryTotal, tutorialTotal, practicalTotal, activityTotal int
	if curriculumTemplate == "2026" {
		switch course.CourseType {
		case "Theory": // Theory
			theoryTotal = course.LectureHrs * 15
			tutorialTotal = course.TutorialHrs * 15
			activityTotal = course.ActivityHrs * 15
			practicalTotal = 0
		case "Lab": // Lab
			theoryTotal = 0
			tutorialTotal = 0
			activityTotal = 0
			practicalTotal = course.PracticalHrs * 15
		case "Theory with Lab": // Theory with Lab
			theoryTotal = course.LectureHrs * 15
			tutorialTotal = course.TutorialHrs * 15
			practicalTotal = course.PracticalHrs * 15
			activityTotal = 0
		default:
			// Default to calculating all
			theoryTotal = course.LectureHrs * 15
			tutorialTotal = course.TutorialHrs * 15
			practicalTotal = course.PracticalHrs * 15
			activityTotal = course.ActivityHrs * 15
		}
	} else {
		// For 2022 or other templates, calculate all as before
		theoryTotal = course.LectureHrs * 15
		tutorialTotal = course.TutorialHrs * 15
		practicalTotal = course.PracticalHrs * 15
		activityTotal = course.ActivityHrs * 15
	}

	// Update course - calculate total hours (total_hrs and total_marks are GENERATED columns)
	courseTypeID, err := resolveCourseTypeID(course.CourseType)
	if err != nil {
		log.Println("Error resolving course_type:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course type"})
		return
	}
	query := `UPDATE courses SET course_code = ?, course_name = ?, course_type = ?, category = ?, 
		experiment_count_theorywithlab = ?,
		credit = ?, lecture_hrs = ?, tutorial_hrs = ?, practical_hrs = ?, activity_hrs = ?, ` + "`tw/sl`" + ` = ?,
		theory_total_hrs = ?, tutorial_total_hrs = ?, practical_total_hrs = ?, activity_total_hrs = ?,
		cia_marks = ?, see_marks = ? WHERE id = ?`

	_, err = db.DB.Exec(query, course.CourseCode, course.CourseName, courseTypeID, course.Category,
		course.ExperimentCountTWL,
		course.Credit, course.LectureHrs, course.TutorialHrs, course.PracticalHrs, course.ActivityHrs, course.TwSlHrs,
		theoryTotal, tutorialTotal, practicalTotal, activityTotal,
		course.CIAMarks, course.SEEMarks, courseID)
	if err != nil {
		log.Println("Error updating course:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update course"})
		return
	}

	// Generate diff and log changes
	diff := make(map[string]map[string]interface{})
	if oldCourse.CourseCode != course.CourseCode {
		diff["course_code"] = map[string]interface{}{"old": oldCourse.CourseCode, "new": course.CourseCode}
	}
	if oldCourse.CourseName != course.CourseName {
		diff["course_name"] = map[string]interface{}{"old": oldCourse.CourseName, "new": course.CourseName}
	}
	if oldCourse.CourseType != course.CourseType {
		diff["course_type"] = map[string]interface{}{"old": oldCourse.CourseType, "new": course.CourseType}
	}
	if oldCourse.ExperimentCountTWL != course.ExperimentCountTWL {
		diff["experiment_count_theorywithlab"] = map[string]interface{}{"old": oldCourse.ExperimentCountTWL, "new": course.ExperimentCountTWL}
	}
	if oldCourse.Category != course.Category {
		diff["category"] = map[string]interface{}{"old": oldCourse.Category, "new": course.Category}
	}
	if oldCourse.Credit != course.Credit {
		diff["credit"] = map[string]interface{}{"old": oldCourse.Credit, "new": course.Credit}
	}
	if oldCourse.LectureHrs != course.LectureHrs {
		diff["lecture_hrs"] = map[string]interface{}{"old": oldCourse.LectureHrs, "new": course.LectureHrs}
	}
	if oldCourse.TutorialHrs != course.TutorialHrs {
		diff["tutorial_hrs"] = map[string]interface{}{"old": oldCourse.TutorialHrs, "new": course.TutorialHrs}
	}
	if oldCourse.PracticalHrs != course.PracticalHrs {
		diff["practical_hrs"] = map[string]interface{}{"old": oldCourse.PracticalHrs, "new": course.PracticalHrs}
	}
	if oldCourse.ActivityHrs != course.ActivityHrs {
		diff["activity_hrs"] = map[string]interface{}{"old": oldCourse.ActivityHrs, "new": course.ActivityHrs}
	}
	if oldCourse.CIAMarks != course.CIAMarks {
		diff["cia_marks"] = map[string]interface{}{"old": oldCourse.CIAMarks, "new": course.CIAMarks}
	}
	if oldCourse.SEEMarks != course.SEEMarks {
		diff["see_marks"] = map[string]interface{}{"old": oldCourse.SEEMarks, "new": course.SEEMarks}
	}

	if len(diff) > 0 && curriculumID > 0 {
		LogCurriculumActivityWithDiff(curriculumID, "Course Updated",
			fmt.Sprintf("Updated course: %s - %s", course.CourseCode, course.CourseName), "System", diff)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Course updated successfully"})
}

// GetCourse retrieves a single course by ID
func GetCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid course ID"})
		return
	}

	var course models.Course
	err = db.DB.QueryRow(`
		SELECT id, course_code, course_name, course_type, COALESCE(experiment_count_theorywithlab, 0), category, credit, 
		       lecture_hrs, tutorial_hrs, practical_hrs, COALESCE(`+"`tw/sl`"+`, 0) as tw_sl, cia_marks, see_marks, total_marks
		FROM courses 
		WHERE id = ?`, courseID).
		Scan(&course.CourseID, &course.CourseCode, &course.CourseName, &course.CourseType,
			&course.ExperimentCountTWL, &course.Category, &course.Credit, &course.LectureHrs, &course.TutorialHrs,
			&course.PracticalHrs, &course.TwSlHrs, &course.CIAMarks, &course.SEEMarks, &course.TotalMarks)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Course not found"})
		return
	} else if err != nil {
		log.Println("Error fetching course:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch course"})
		return
	}

	json.NewEncoder(w).Encode(course)
}

// UpdateCurriculumCourse updates curriculum_courses link table (e.g., count_towards_limit)
func UpdateCurriculumCourse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	vars := mux.Vars(r)
	regCourseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid curriculum course ID"})
		return
	}

	var requestData struct {
		CountTowardsLimit bool `json:"count_towards_limit"`
	}
	err = json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		log.Println("Error decoding request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Update curriculum_courses record
	query := "UPDATE curriculum_courses SET count_towards_limit = ? WHERE id = ?"
	_, err = db.DB.Exec(query, requestData.CountTowardsLimit, regCourseID)
	if err != nil {
		log.Println("Error updating curriculum_courses:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update curriculum course link"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Curriculum course link updated successfully"})
}
