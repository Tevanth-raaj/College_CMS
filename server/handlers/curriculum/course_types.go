package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"server/db"
)

type CourseType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func GetCourseTypes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rows, err := db.DB.Query("SELECT id, course_type FROM course_type ORDER BY id")
	if err != nil {
		log.Printf("Error fetching course types: %v", err)
		http.Error(w, "Failed to fetch course types", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var types []CourseType
	for rows.Next() {
		var t CourseType
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			log.Printf("Error scanning course type: %v", err)
			continue
		}
		types = append(types, t)
	}

	json.NewEncoder(w).Encode(types)
}
