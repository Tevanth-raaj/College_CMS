package curriculum

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"server/db"
	"server/models"
	repository "server/repositories/copoattainment"
	service "server/services/copoattainment"
)

func buildCOPOAttainmentService() *service.Service {
	return service.New(repository.New(db.DB))
}

func setCOPOAttainmentHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

// GetTestTypes handles GET /api/test-types and returns assessment components for the selected course type.
func GetTestTypes(w http.ResponseWriter, r *http.Request) {
	setCOPOAttainmentHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	svc := buildCOPOAttainmentService()
	if svc == nil {
		writeJSONError(w, "Service unavailable", http.StatusInternalServerError)
		return
	}

	var courseID *int
	if rawCourseID := strings.TrimSpace(r.URL.Query().Get("courseId")); rawCourseID != "" {
		parsedCourseID, err := strconv.Atoi(rawCourseID)
		if err != nil || parsedCourseID <= 0 {
			writeJSONError(w, "Invalid courseId", http.StatusBadRequest)
			return
		}
		courseID = &parsedCourseID
	}

	testTypes, err := svc.GetTestTypes(courseID)
	if err != nil {
		log.Printf("GetTestTypes error: %v", err)
		writeJSONError(w, "Failed to fetch test types", http.StatusInternalServerError)
		return
	}

	if testTypes == nil {
		testTypes = []models.TestTypeOption{}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(testTypes)
}

// GetCOPOAttainmentStudents handles GET /api/co-po-attainment/students and returns course-wise mapped students.
func GetCOPOAttainmentStudents(w http.ResponseWriter, r *http.Request) {
	setCOPOAttainmentHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	rawTestTypeID := strings.TrimSpace(r.URL.Query().Get("testTypeId"))
	if rawTestTypeID == "" {
		writeJSONError(w, "testTypeId is required", http.StatusBadRequest)
		return
	}

	rawCourseID := strings.TrimSpace(r.URL.Query().Get("courseId"))
	if rawCourseID == "" {
		writeJSONError(w, "courseId is required", http.StatusBadRequest)
		return
	}

	rawWindowID := strings.TrimSpace(r.URL.Query().Get("windowId"))
	rawTargetPercent := strings.TrimSpace(r.URL.Query().Get("targetPercent"))

	testTypeID, err := strconv.Atoi(rawTestTypeID)
	if err != nil || testTypeID <= 0 {
		writeJSONError(w, "Invalid testTypeId", http.StatusBadRequest)
		return
	}

	courseID, err := strconv.Atoi(rawCourseID)
	if err != nil || courseID <= 0 {
		writeJSONError(w, "Invalid courseId", http.StatusBadRequest)
		return
	}

	windowID := 0
	if rawWindowID != "" {
		parsedWindowID, parseErr := strconv.Atoi(rawWindowID)
		if parseErr != nil || parsedWindowID <= 0 {
			writeJSONError(w, "Invalid windowId", http.StatusBadRequest)
			return
		}
		windowID = parsedWindowID
	}

	targetPercent := 0.0
	if rawTargetPercent != "" {
		parsedTargetPercent, parseErr := strconv.ParseFloat(rawTargetPercent, 64)
		if parseErr != nil || parsedTargetPercent < 0 || parsedTargetPercent > 100 {
			writeJSONError(w, "Invalid targetPercent", http.StatusBadRequest)
			return
		}
		targetPercent = parsedTargetPercent
	}

	svc := buildCOPOAttainmentService()
	if svc == nil {
		writeJSONError(w, "Service unavailable", http.StatusInternalServerError)
		return
	}

	response, err := svc.GetStudents(courseID, testTypeID, windowID, targetPercent)
	if err != nil {
		log.Printf("GetCOPOAttainmentStudents error: %v", err)
		writeJSONError(w, "Failed to fetch CO-PO attainment students", http.StatusInternalServerError)
		return
	}

	if response == nil {
		response = &models.COPOAttainmentResponse{
			Columns:       []models.COMarkColumn{},
			POColumns:     []models.POAttainmentColumn{},
			Students:      []models.COPOAttainmentStudentRow{},
			POSummary:     []models.POAttainmentSummary{},
			TargetPercent: targetPercent,
			PresentCount:  0,
			AbsentCount:   0,
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
