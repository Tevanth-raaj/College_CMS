package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"strconv"

	"github.com/gorilla/mux"
)

func GetCourseSyllabusNested(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	curriculumTemplate := getCurriculumTemplateForCourse(courseID)

	var resp models.CourseSyllabusResponse
	resp.CurriculumTemplate = curriculumTemplate

	// 1. Fetch header data from normalized tables
	resp.Header.ID = courseID // Use course_id as the identifier
	resp.Header.CourseID = courseID
	resp.Header.Objectives, _ = fetchObjectives(courseID)
	resp.Header.Outcomes, _ = fetchOutcomes(courseID)
	resp.Header.ReferenceList, _ = fetchReferences(courseID)
	resp.Header.TextbookReferenceList, _ = fetchTextbookReferences(courseID)
	resp.Header.Prerequisites, _ = fetchPrerequisites(courseID)
	resp.Header.Teamwork, _ = fetchTeamwork(courseID)
	resp.Header.SelfLearning, _ = fetchSelfLearning(courseID)

	// 2. Get models linked via course_id directly
	modelRows, err := db.DB.Query(`
		SELECT id, course_id, model_name, position 
		FROM syllabus 
		WHERE course_id = ? AND (status = 1 OR status IS NULL)
		ORDER BY position, id`, courseID)

	if err != nil {
		log.Println("Error fetching models:", err)
		http.Error(w, "Failed to fetch models", http.StatusInternalServerError)
		return
	}
	defer modelRows.Close()

	modelsList := []models.SyllabusModel{}
	for modelRows.Next() {
		var model models.SyllabusModel
		if err := modelRows.Scan(&model.ID, &model.CourseID, &model.ModelName, &model.Position); err != nil {
			log.Println("Error scanning model:", err)
			continue
		}

		// 3. Get titles for this model
		titleRows, err := db.DB.Query(`
			SELECT id, model_id, title, hours, position 
			FROM syllabus_titles 
			WHERE model_id = ? AND (status = 1 OR status IS NULL)
			ORDER BY position, id`, model.ID)

		if err != nil {
			log.Println("Error fetching titles for model", model.ID, ":", err)
			model.Titles = []models.SyllabusTitle{}
			modelsList = append(modelsList, model)
			continue
		}

		titlesList := []models.SyllabusTitle{}
		for titleRows.Next() {
			var title models.SyllabusTitle
			if err := titleRows.Scan(&title.ID, &title.ModelID, &title.TitleName, &title.Hours, &title.Position); err != nil {
				log.Println("Error scanning title:", err)
				continue
			}

			// 4. Get topics for this title
			topicRows, err := db.DB.Query(`
				SELECT id, title_id, topic, position 
				FROM syllabus_topics 
				WHERE title_id = ? AND (status = 1 OR status IS NULL)
				ORDER BY position, id`, title.ID)

			if err != nil {
				log.Println("Error fetching topics for title", title.ID, ":", err)
				title.Topics = []models.SyllabusTopic{}
				titlesList = append(titlesList, title)
				continue
			}

			topicsList := []models.SyllabusTopic{}
			for topicRows.Next() {
				var topic models.SyllabusTopic
				if err := topicRows.Scan(&topic.ID, &topic.TitleID, &topic.Topic, &topic.Position); err != nil {
					log.Println("Error scanning topic:", err)
					continue
				}
				topicsList = append(topicsList, topic)
			}
			topicRows.Close()

			title.Topics = topicsList
			titlesList = append(titlesList, title)
		}
		titleRows.Close()

		model.Titles = titlesList
		modelsList = append(modelsList, model)
	}

	resp.Models = modelsList

	// Fetch experiments for 2022 template
	if curriculumTemplate == "2022" {
		experiments := []models.Experiment{}
		expRows, err := db.DB.Query(`
			SELECT id, course_id, experiment_number, experiment_name, hours
			FROM course_experiments
			WHERE course_id = ? AND status = 1
			ORDER BY experiment_number`, courseID)
		if err == nil {
			defer expRows.Close()
			for expRows.Next() {
				var exp models.Experiment
				if err := expRows.Scan(&exp.ID, &exp.CourseID, &exp.ExperimentNumber, &exp.ExperimentName, &exp.Hours); err != nil {
					continue
				}

				// Fetch topics
				topicRows, err := db.DB.Query(`
					SELECT topic_text
					FROM course_experiment_topics
					WHERE experiment_id = ? AND status = 1
					ORDER BY topic_order`, exp.ID)
				if err != nil {
					exp.Topics = []string{}
				} else {
					topics := []string{}
					for topicRows.Next() {
						var topic string
						if err := topicRows.Scan(&topic); err == nil {
							topics = append(topics, topic)
						}
					}
					topicRows.Close()
					exp.Topics = topics
				}

				experiments = append(experiments, exp)
			}
		}
		resp.Experiments = experiments
	}

	json.NewEncoder(w).Encode(resp)
}

// ============================================================================
// MODEL CRUD OPERATIONS
// ============================================================================

// CreateModel creates a new module under a course syllabus
func CreateModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseId"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	var body struct {
		ModelName string `json:"model_name"`
		Position  int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println("CreateModel decode error:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG CreateModel: courseID=%d, ModelName=%s, Position=%d", courseID, body.ModelName, body.Position)

	// Insert model with course_id directly (course-centric design)
	result, err := db.DB.Exec(`
		INSERT INTO syllabus (course_id, model_name, name, position, status)
		VALUES (?, ?, ?, ?, 1)`, courseID, body.ModelName, body.ModelName, body.Position)

	if err != nil {
		log.Println("CreateModel error:", err)
		http.Error(w, "Failed to create model", http.StatusInternalServerError)
		return
	}

	modelID, _ := result.LastInsertId()
	log.Printf("DEBUG CreateModel: successfully created model with ID=%d", modelID)
	json.NewEncoder(w).Encode(map[string]int{"id": int(modelID)})
}

// UpdateModel updates a model's name and position
func UpdateModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID, err := strconv.Atoi(vars["modelId"])
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	var body struct {
		ModelName string `json:"model_name"`
		Position  int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE syllabus 
		SET model_name = ?, name = ?, position = ? 
		WHERE id = ? AND (status = 1 OR status IS NULL)`, body.ModelName, body.ModelName, body.Position, modelID)

	if err != nil {
		log.Println("UpdateModel error:", err)
		http.Error(w, "Failed to update model", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteModel deletes a model (cascades to titles and topics)
func DeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID, err := strconv.Atoi(vars["modelId"])
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("DeleteModel begin tx error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Get parent course + current position so we can shift the remaining models
	var courseID, pos int
	if err := tx.QueryRow(
		"SELECT course_id, position FROM syllabus WHERE id = ? AND (status = 1 OR status IS NULL)",
		modelID,
	).Scan(&courseID, &pos); err != nil {
		log.Println("DeleteModel lookup error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	// Soft delete the model
	if _, err := tx.Exec("UPDATE syllabus SET status = 0 WHERE id = ? AND (status = 1 OR status IS NULL)", modelID); err != nil {
		log.Println("DeleteModel soft delete error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	// Cascade soft delete titles and topics under this model
	if _, err := tx.Exec("UPDATE syllabus_titles SET status = 0 WHERE model_id = ? AND (status = 1 OR status IS NULL)", modelID); err != nil {
		log.Println("DeleteModel cascade titles error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(
		"UPDATE syllabus_topics SET status = 0 WHERE title_id IN (SELECT id FROM syllabus_titles WHERE model_id = ?) AND (status = 1 OR status IS NULL)",
		modelID,
	); err != nil {
		log.Println("DeleteModel cascade topics error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	// Shift positions of remaining active models
	if _, err := tx.Exec(
		"UPDATE syllabus SET position = position - 1 WHERE course_id = ? AND position > ? AND (status = 1 OR status IS NULL)",
		courseID, pos,
	); err != nil {
		log.Println("DeleteModel shift positions error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("DeleteModel commit error:", err)
		http.Error(w, "Failed to delete model", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// TITLE CRUD OPERATIONS
// ============================================================================

// CreateTitle creates a new title under a model
func CreateTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID, err := strconv.Atoi(vars["modelId"])
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	var body struct {
		TitleName string `json:"title_name"`
		Hours     int    `json:"hours"`
		Position  int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println("CreateTitle decode error:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG CreateTitle: modelID=%d, TitleName=%s, Hours=%d, Position=%d", modelID, body.TitleName, body.Hours, body.Position)

	result, err := db.DB.Exec(`
		INSERT INTO syllabus_titles (model_id, title, hours, position, status)
		VALUES (?, ?, ?, ?, 1)`, modelID, body.TitleName, body.Hours, body.Position)

	if err != nil {
		log.Println("CreateTitle error:", err)
		http.Error(w, "Failed to create title", http.StatusInternalServerError)
		return
	}

	titleID, _ := result.LastInsertId()
	log.Printf("DEBUG CreateTitle: successfully created title with ID=%d", titleID)
	json.NewEncoder(w).Encode(map[string]int{"id": int(titleID)})
}

// UpdateTitle updates a title's name, hours, and position
func UpdateTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	titleID, err := strconv.Atoi(vars["titleId"])
	if err != nil {
		http.Error(w, "Invalid title ID", http.StatusBadRequest)
		return
	}

	var body struct {
		TitleName string `json:"title_name"`
		Hours     int    `json:"hours"`
		Position  int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE syllabus_titles 
		SET title = ?, hours = ?, position = ? 
		WHERE id = ? AND (status = 1 OR status IS NULL)`, body.TitleName, body.Hours, body.Position, titleID)

	if err != nil {
		log.Println("UpdateTitle error:", err)
		http.Error(w, "Failed to update title", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteTitle deletes a title (cascades to topics)
func DeleteTitle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	titleID, err := strconv.Atoi(vars["titleId"])
	if err != nil {
		http.Error(w, "Invalid title ID", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("DeleteTitle begin tx error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Get parent model + current position so we can shift remaining titles
	var modelID, pos int
	if err := tx.QueryRow(
		"SELECT model_id, position FROM syllabus_titles WHERE id = ? AND (status = 1 OR status IS NULL)",
		titleID,
	).Scan(&modelID, &pos); err != nil {
		log.Println("DeleteTitle lookup error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}

	// Soft delete the title
	if _, err := tx.Exec("UPDATE syllabus_titles SET status = 0 WHERE id = ? AND (status = 1 OR status IS NULL)", titleID); err != nil {
		log.Println("DeleteTitle soft delete error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}

	// Cascade soft delete topics under this title
	if _, err := tx.Exec("UPDATE syllabus_topics SET status = 0 WHERE title_id = ? AND (status = 1 OR status IS NULL)", titleID); err != nil {
		log.Println("DeleteTitle cascade topics error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}

	// Shift positions of remaining active titles
	if _, err := tx.Exec(
		"UPDATE syllabus_titles SET position = position - 1 WHERE model_id = ? AND position > ? AND (status = 1 OR status IS NULL)",
		modelID, pos,
	); err != nil {
		log.Println("DeleteTitle shift positions error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("DeleteTitle commit error:", err)
		http.Error(w, "Failed to delete title", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// TOPIC CRUD OPERATIONS
// ============================================================================

// CreateTopic creates a new topic under a title
func CreateTopic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	titleID, err := strconv.Atoi(vars["titleId"])
	if err != nil {
		http.Error(w, "Invalid title ID", http.StatusBadRequest)
		return
	}

	var body struct {
		Topic    string `json:"topic"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Println("CreateTopic decode error:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG CreateTopic: titleID=%d, Topic=%s, Position=%d", titleID, body.Topic, body.Position)

	result, err := db.DB.Exec(`
		INSERT INTO syllabus_topics (title_id, topic, position, status)
		VALUES (?, ?, ?, 1)`, titleID, body.Topic, body.Position)

	if err != nil {
		log.Println("CreateTopic error:", err)
		http.Error(w, "Failed to create topic", http.StatusInternalServerError)
		return
	}

	topicID, _ := result.LastInsertId()
	log.Printf("DEBUG CreateTopic: successfully created topic with ID=%d", topicID)
	json.NewEncoder(w).Encode(map[string]int{"id": int(topicID)})
}

// UpdateTopic updates a topic's content and position
func UpdateTopic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topicID, err := strconv.Atoi(vars["topicId"])
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	var body struct {
		Topic    string `json:"topic"`
		Position int    `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = db.DB.Exec(`
		UPDATE syllabus_topics 
		SET topic = ?, position = ? 
		WHERE id = ? AND (status = 1 OR status IS NULL)`, body.Topic, body.Position, topicID)

	if err != nil {
		log.Println("UpdateTopic error:", err)
		http.Error(w, "Failed to update topic", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteTopic deletes a topic
func DeleteTopic(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topicID, err := strconv.Atoi(vars["topicId"])
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		log.Println("DeleteTopic begin tx error:", err)
		http.Error(w, "Failed to delete topic", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Get parent title + current position so we can shift remaining topics
	var titleID, pos int
	if err := tx.QueryRow(
		"SELECT title_id, position FROM syllabus_topics WHERE id = ? AND (status = 1 OR status IS NULL)",
		topicID,
	).Scan(&titleID, &pos); err != nil {
		log.Println("DeleteTopic lookup error:", err)
		http.Error(w, "Failed to delete topic", http.StatusInternalServerError)
		return
	}

	// Soft delete topic
	if _, err := tx.Exec("UPDATE syllabus_topics SET status = 0 WHERE id = ? AND (status = 1 OR status IS NULL)", topicID); err != nil {
		log.Println("DeleteTopic soft delete error:", err)
		http.Error(w, "Failed to delete topic", http.StatusInternalServerError)
		return
	}

	// Shift positions of remaining active topics
	if _, err := tx.Exec(
		"UPDATE syllabus_topics SET position = position - 1 WHERE title_id = ? AND position > ? AND (status = 1 OR status IS NULL)",
		titleID, pos,
	); err != nil {
		log.Println("DeleteTopic shift positions error:", err)
		http.Error(w, "Failed to delete topic", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("DeleteTopic commit error:", err)
		http.Error(w, "Failed to delete topic", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
