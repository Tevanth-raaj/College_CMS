package curriculum

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"server/db"
	"server/models"
	"strings"
)

type coursePayload struct {
	CourseID           int             `json:"id"`
	CourseCode         string          `json:"course_code"`
	CourseName         string          `json:"course_name"`
	CourseType         json.RawMessage `json:"course_type"`
	ExperimentCountTWL int             `json:"experiment_count_theorywithlab"`
	Category           string          `json:"category"`
	Credit             int             `json:"credit"`
	LectureHrs         int             `json:"lecture_hrs"`
	TutorialHrs        int             `json:"tutorial_hrs"`
	PracticalHrs       int             `json:"practical_hrs"`
	ActivityHrs        int             `json:"activity_hrs"`
	TwSlHrs            int             `json:"tw_sl_hrs"`
	TheoryTotalHrs     int             `json:"theory_total_hrs"`
	TutorialTotalHrs   int             `json:"tutorial_total_hrs"`
	ActivityTotalHrs   int             `json:"activity_total_hrs"`
	PracticalTotalHrs  int             `json:"practical_total_hrs"`
	TotalHrs           int             `json:"total_hrs"`
	CIAMarks           int             `json:"cia_marks"`
	SEEMarks           int             `json:"see_marks"`
	TotalMarks         int             `json:"total_marks"`
	CountTowardsLimit  *bool           `json:"count_towards_limit,omitempty"`
	CurriculumTemplate string          `json:"curriculum_template,omitempty"`
}

func decodeCourseRequest(r *http.Request) (models.Course, error) {
	var payload coursePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return models.Course{}, err
	}

	courseType, err := resolveCourseType(payload.CourseType)
	if err != nil {
		return models.Course{}, err
	}

	course := models.Course{
		CourseID:           payload.CourseID,
		CourseCode:         strings.TrimSpace(payload.CourseCode),
		CourseName:         strings.TrimSpace(payload.CourseName),
		CourseType:         courseType,
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
		ActivityTotalHrs:   payload.ActivityTotalHrs,
		PracticalTotalHrs:  payload.PracticalTotalHrs,
		TotalHrs:           payload.TotalHrs,
		CIAMarks:           payload.CIAMarks,
		SEEMarks:           payload.SEEMarks,
		TotalMarks:         payload.TotalMarks,
		CountTowardsLimit:  payload.CountTowardsLimit,
		CurriculumTemplate: payload.CurriculumTemplate,
	}

	return course, nil
}

func resolveCourseType(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var courseTypeString string
	if err := json.Unmarshal(raw, &courseTypeString); err == nil {
		return courseTypeString, nil
	}

	var courseTypeID int
	if err := json.Unmarshal(raw, &courseTypeID); err == nil {
		var courseTypeName sql.NullString
		err := db.DB.QueryRow("SELECT course_type FROM course_type WHERE id = ?", courseTypeID).Scan(&courseTypeName)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", fmt.Errorf("invalid course_type id")
			}
			return "", err
		}
		return courseTypeName.String, nil
	}

	return "", fmt.Errorf("invalid course_type format")
}

func resolveCourseTypeID(courseType string) (int, error) {
	if courseType == "" {
		return 0, fmt.Errorf("course_type is required")
	}

	var numericID int
	if _, err := fmt.Sscanf(courseType, "%d", &numericID); err == nil {
		return numericID, nil
	}

	var courseTypeID int
	err := db.DB.QueryRow("SELECT id FROM course_type WHERE course_type = ?", courseType).Scan(&courseTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("invalid course_type")
		}
		return 0, err
	}

	return courseTypeID, nil
}
