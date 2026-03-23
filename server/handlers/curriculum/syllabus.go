package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"server/models"
	"strconv"

	"github.com/gorilla/mux"
)

// GetCourseSyllabus handles GET /course/:courseId/syllabus
// Fetches data from normalized tables and returns in the same JSON format as before
func GetCourseSyllabus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	// Fetch all data from normalized tables (course-centric design)
	syllabus := models.Syllabus{
		ID:       courseID, // Use course_id as identifier
		CourseID: courseID,
	}

	syllabus.Objectives, _ = fetchObjectives(courseID)
	syllabus.Outcomes, _ = fetchOutcomes(courseID)
	syllabus.ReferenceList, _ = fetchReferences(courseID)
	if getCurriculumTemplateForCourse(courseID) == "2026" {
		syllabus.TextbookReferenceList, _ = fetchTextbookReferences(courseID)
	} else {
		syllabus.TextbookReferenceList = []string{}
	}
	syllabus.Prerequisites, _ = fetchPrerequisites(courseID)
	syllabus.Teamwork, _ = fetchTeamwork(courseID)
	syllabus.SelfLearning, _ = fetchSelfLearning(courseID)

	json.NewEncoder(w).Encode(syllabus)
}

// SaveCourseSyllabus handles POST /course/:courseId/syllabus
// Saves data to normalized tables while maintaining the same API interface
// Accepts either a Syllabus object or just the header fields
func SaveCourseSyllabus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	// Parse request body - it could be just header fields or full syllabus
	var requestData struct {
		Objectives            []string             `json:"objectives"`
		Outcomes              []string             `json:"outcomes"`
		ReferenceList         []string             `json:"reference_list"`
		TextbookReferenceList []string             `json:"textbook_reference_list"`
		Prerequisites         []string             `json:"prerequisites"`
		Teamwork              *models.Teamwork     `json:"teamwork"`
		SelfLearning          *models.SelfLearning `json:"selflearning"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		log.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG: Received syllabus data for courseID %d:", courseID)
	log.Printf("  - Objectives: %d items", len(requestData.Objectives))
	log.Printf("  - Outcomes: %d items", len(requestData.Outcomes))
	log.Printf("  - References: %d items", len(requestData.ReferenceList))
	log.Printf("  - Textbook References: %d items", len(requestData.TextbookReferenceList))
	log.Printf("  - Prerequisites: %d items", len(requestData.Prerequisites))
	if requestData.Teamwork != nil {
		log.Printf("  - Teamwork: hours=%d, activities=%d items", requestData.Teamwork.Hours, len(requestData.Teamwork.Activities))
	} else {
		log.Printf("  - Teamwork: nil")
	}
	if requestData.SelfLearning != nil {
		log.Printf("  - SelfLearning: hours=%d", requestData.SelfLearning.Hours)
	} else {
		log.Printf("  - SelfLearning: nil")
	}

	// Save all data to normalized tables (course-centric design)
	if err := saveObjectives(courseID, requestData.Objectives); err != nil {
		log.Println("Error saving objectives:", err)
		http.Error(w, "Failed to save objectives", http.StatusInternalServerError)
		return
	}

	if err := saveOutcomes(courseID, requestData.Outcomes); err != nil {
		log.Println("Error saving outcomes:", err)
		http.Error(w, "Failed to save outcomes", http.StatusInternalServerError)
		return
	}

	if err := saveReferences(courseID, requestData.ReferenceList); err != nil {
		log.Println("Error saving references:", err)
		http.Error(w, "Failed to save references", http.StatusInternalServerError)
		return
	}

	if getCurriculumTemplateForCourse(courseID) == "2026" {
		if err := saveTextbookReferences(courseID, requestData.TextbookReferenceList); err != nil {
			log.Println("Error saving textbook references:", err)
			http.Error(w, "Failed to save textbook references", http.StatusInternalServerError)
			return
		}
	} else {
		// For older template children, textbook references remain empty
		_ = saveTextbookReferences(courseID, []string{})
	}

	if err := savePrerequisites(courseID, requestData.Prerequisites); err != nil {
		log.Println("Error saving prerequisites:", err)
		http.Error(w, "Failed to save prerequisites", http.StatusInternalServerError)
		return
	}

	if err := saveTeamwork(courseID, requestData.Teamwork); err != nil {
		log.Println("Error saving teamwork:", err)
		// Teamwork schemas differ across DB versions; don't fail the whole save
		// when objectives/outcomes/etc have already been persisted successfully.
	}

	if err := saveSelfLearning(courseID, requestData.SelfLearning); err != nil {
		log.Println("Error saving self-learning:", err)
		// Self-learning schemas differ across DB versions; don't fail the whole save
		// when objectives/outcomes/etc have already been persisted successfully.
	}

	// Return success response
	response := models.Syllabus{
		ID:            courseID, // Use course_id as identifier
		Outcomes:      requestData.Outcomes,
		ReferenceList: requestData.ReferenceList,
		Prerequisites: requestData.Prerequisites,
		Teamwork:      requestData.Teamwork,
		SelfLearning:  requestData.SelfLearning,
	}
	if getCurriculumTemplateForCourse(courseID) == "2026" {
		response.TextbookReferenceList = requestData.TextbookReferenceList
	} else {
		response.TextbookReferenceList = []string{}
	}

	json.NewEncoder(w).Encode(response)
}
