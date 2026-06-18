package copoattainment

import (
	"fmt"
	"strings"

	"server/models"
	repository "server/repositories/copoattainment"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetTestTypes(courseID *int) ([]models.TestTypeOption, error) {
	return s.repo.GetTestTypes(courseID)
}

func (s *Service) GetStudents(courseID, testTypeID, windowID int, targetPercent float64) (*models.COPOAttainmentResponse, error) {
	response, err := s.repo.GetStudentsByCourseAndTestType(courseID, testTypeID, windowID, targetPercent)
	if err != nil {
		return nil, err
	}

	rows := response.Students

	// Safety net: dedupe by student+course in case the allocation table contains repeated mappings.
	deduped := make([]models.COPOAttainmentStudentRow, 0, len(rows))
	indexByKey := make(map[string]int, len(rows))
	for _, row := range rows {
		key := fmt.Sprintf("%d:%d:%s", row.StudentID, row.CourseID, strings.ToLower(strings.TrimSpace(row.RegisterNo)))
		if idx, exists := indexByKey[key]; exists {
			for columnKey, mark := range row.COMarks {
				if deduped[idx].COMarks == nil {
					deduped[idx].COMarks = make(map[string]*float64)
				}
				if _, alreadyExists := deduped[idx].COMarks[columnKey]; !alreadyExists && mark != nil {
					deduped[idx].COMarks[columnKey] = mark
				}
			}
			for poKey, po := range row.POAttainment {
				if deduped[idx].POAttainment == nil {
					deduped[idx].POAttainment = make(map[string]*float64)
				}
				if _, alreadyExists := deduped[idx].POAttainment[poKey]; !alreadyExists && po != nil {
					deduped[idx].POAttainment[poKey] = po
				}
			}
			if deduped[idx].TotalMarks == nil && row.TotalMarks != nil {
				deduped[idx].TotalMarks = row.TotalMarks
			}
			continue
		}

		indexByKey[key] = len(deduped)
		deduped = append(deduped, row)
	}

	response.Students = deduped
	response.PresentCount = len(deduped)
	return response, nil
}
