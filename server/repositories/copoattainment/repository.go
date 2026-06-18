package copoattainment

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"server/models"
)

type Repository struct {
	db *sql.DB
}

type componentDefinition struct {
	ID    int
	Name  string
	Key   string
	Label string
	Order int
	Max   int
}

type coPOMappingRecord struct {
	COIndex      int
	POIndex      int
	MappingValue int
}

var (
	coNumberPattern   = regexp.MustCompile(`(?i)\bCO\s*-?\s*(\d+)\b`)
	partNumberPattern = regexp.MustCompile(`\((\d+)\)`)
	unitTestPattern   = regexp.MustCompile(`(?i)\bUNIT\s*TEST\s*([1-5])(?:\s*\(?([A-B])\)?)?\b`)
	testFamilyPattern = regexp.MustCompile(`(?i)\b(?:PERIODICAL\s*TEST|UNIT\s*TEST|PT)\s*-?\s*([12])\b`)
	ipFamilyPattern   = regexp.MustCompile(`(?i)\b(?:INNOVATIVE\s*PRACTICE|IP)\s*-?\s*([12])\b`)
)

func New(database *sql.DB) *Repository {
	return &Repository{db: database}
}

func mapCourseCategoryToTypeID(category string) int {
	categoryLower := strings.ToLower(strings.TrimSpace(category))
	if categoryLower == "" {
		return 0
	}
	if strings.Contains(categoryLower, "theory") && strings.Contains(categoryLower, "lab") {
		return 3
	}
	if strings.Contains(categoryLower, "lab") {
		return 2
	}
	return 1
}

func (r *Repository) resolveCourseTypeID(courseID int) (int, error) {
	var rawCourseType sql.NullString
	var courseCategory sql.NullString

	err := r.db.QueryRow(`
		SELECT COALESCE(CAST(course_type AS CHAR), ''), COALESCE(category, '')
		FROM courses
		WHERE id = ?
	`, courseID).Scan(&rawCourseType, &courseCategory)
	if err != nil {
		return 0, err
	}

	if rawCourseType.Valid {
		raw := strings.TrimSpace(rawCourseType.String)
		if raw != "" {
			if parsedID, convErr := strconv.Atoi(raw); convErr == nil && parsedID > 0 {
				return parsedID, nil
			}

			var resolvedID sql.NullInt64
			err = r.db.QueryRow(`
				SELECT id
				FROM course_type
				WHERE LOWER(TRIM(course_type)) = LOWER(TRIM(?))
				LIMIT 1
			`, raw).Scan(&resolvedID)
			if err == nil && resolvedID.Valid && resolvedID.Int64 > 0 {
				return int(resolvedID.Int64), nil
			}
		}
	}

	return mapCourseCategoryToTypeID(courseCategory.String), nil
}

func (r *Repository) GetTestTypes(courseID *int) ([]models.TestTypeOption, error) {
	testTypes := make([]models.TestTypeOption, 0)

	query := `
		SELECT
			mcn.id,
			COALESCE(mcn.category_name, ''),
			COALESCE(SUM(mct.max_marks), 0),
			COALESCE(MIN(mct.position), 0)
		FROM mark_category_types mct
		INNER JOIN mark_category_name mcn
			ON mcn.id = mct.category_name_id
		WHERE mct.status = 1
		GROUP BY mcn.id, mcn.category_name
		ORDER BY MIN(mct.position) ASC, mcn.category_name ASC
	`

	args := []interface{}{}
	if courseID != nil {
		courseTypeID, err := r.resolveCourseTypeID(*courseID)
		if err != nil {
			return nil, err
		}
		if courseTypeID <= 0 {
			return testTypes, nil
		}

		query = `
			SELECT
				mcn.id,
				COALESCE(mcn.category_name, ''),
				COALESCE(SUM(mct.max_marks), 0),
				COALESCE(MIN(mct.position), 0)
			FROM mark_category_types mct
			INNER JOIN mark_category_name mcn
				ON mcn.id = mct.category_name_id
			WHERE mct.course_type_id = ?
			  AND mct.status = 1
			GROUP BY mcn.id, mcn.category_name
			ORDER BY MIN(mct.position) ASC, mcn.category_name ASC
		`
		args = append(args, courseTypeID)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.TestTypeOption
		if err := rows.Scan(&item.ID, &item.Name, &item.MaxMarks, &item.Position); err != nil {
			return nil, err
		}
		testTypes = append(testTypes, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return testTypes, nil
}

func deriveComponentLabel(name string, fallbackIndex int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if matches := coNumberPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		return fmt.Sprintf("CO%s", matches[1])
	}
	if matches := unitTestPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		switch matches[1] {
		case "1":
			return "CO1"
		case "2":
			return "CO2"
		case "3":
			return "CO3"
		case "4":
			return "CO4"
		case "5":
			return "CO5"
		}
	}
	if matches := partNumberPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		return fmt.Sprintf("Part %s", matches[1])
	}
	if trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("Part %d", fallbackIndex)
}

func columnSortRank(label string, fallback int) int {
	if matches := coNumberPattern.FindStringSubmatch(label); len(matches) > 1 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			return value
		}
	}
	return 1000 + fallback
}

func normalizeAssessmentFamily(name string) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if trimmed == "" {
		return ""
	}
	if matches := ipFamilyPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		return fmt.Sprintf("IP%s", matches[1])
	}
	if matches := testFamilyPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
		return fmt.Sprintf("PT%s", matches[1])
	}
	return strings.ToUpper(trimmed)
}

func normalizeTargetPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func deriveCOIndex(label string) (int, bool) {
	matches := coNumberPattern.FindStringSubmatch(strings.TrimSpace(label))
	if len(matches) <= 1 {
		return 0, false
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func (r *Repository) studentMarksHasWindowColumn() bool {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'student_marks'
		  AND column_name = 'window_id'
	`).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func (r *Repository) getComponentDefinitions(courseTypeID, testTypeID int) ([]componentDefinition, error) {
	query := `
		SELECT
			mct.id,
			COALESCE(mct.name, ''),
			COALESCE(mct.position, 0),
			COALESCE(mct.max_marks, 0)
		FROM mark_category_types mct
		WHERE mct.course_type_id = ?
		  AND mct.category_name_id = ?
		  AND mct.status = 1
		ORDER BY mct.position ASC, mct.id ASC
	`

	rows, err := r.db.Query(query, courseTypeID, testTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	components := make([]componentDefinition, 0)
	index := 0
	for rows.Next() {
		index++
		var item componentDefinition
		if err := rows.Scan(&item.ID, &item.Name, &item.Order, &item.Max); err != nil {
			return nil, err
		}

		baseLabel := deriveComponentLabel(item.Name, index)
		key := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(baseLabel), " ", "_"))
		item.Label = baseLabel
		item.Key = key
		components = append(components, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.SliceStable(components, func(i, j int) bool {
		left := columnSortRank(components[i].Label, components[i].Order)
		right := columnSortRank(components[j].Label, components[j].Order)
		if left == right {
			return components[i].Order < components[j].Order
		}
		return left < right
	})

	return components, nil
}

func (r *Repository) GetStudentsByCourseAndTestType(courseID, testTypeID, windowID int, targetPercent float64) (*models.COPOAttainmentResponse, error) {
	targetPercent = normalizeTargetPercent(targetPercent)

	courseTypeID, err := r.resolveCourseTypeID(courseID)
	if err != nil {
		return nil, err
	}
	if courseTypeID <= 0 {
		return &models.COPOAttainmentResponse{
			Columns:       []models.COMarkColumn{},
			POColumns:     []models.POAttainmentColumn{},
			Students:      []models.COPOAttainmentStudentRow{},
			POSummary:     []models.POAttainmentSummary{},
			TargetPercent: targetPercent,
			PresentCount:  0,
			AbsentCount:   0,
		}, nil
	}

	components, err := r.getComponentDefinitions(courseTypeID, testTypeID)
	if err != nil {
		return nil, err
	}
	if len(components) == 0 {
		return &models.COPOAttainmentResponse{
			Columns:       []models.COMarkColumn{},
			POColumns:     []models.POAttainmentColumn{},
			Students:      []models.COPOAttainmentStudentRow{},
			POSummary:     []models.POAttainmentSummary{},
			TargetPercent: targetPercent,
			PresentCount:  0,
			AbsentCount:   0,
		}, nil
	}

	testTypeName := ""
	var selectedTestTypeName sql.NullString
	if err := r.db.QueryRow(`
		SELECT COALESCE(category_name, '')
		FROM mark_category_name
		WHERE id = ?
		LIMIT 1
	`, testTypeID).Scan(&selectedTestTypeName); err == nil && selectedTestTypeName.Valid {
		testTypeName = strings.TrimSpace(selectedTestTypeName.String)
	}
	componentIDs := make([]interface{}, 0, len(components))
	columns := make([]models.COMarkColumn, 0, len(components))
	componentByID := make(map[int]componentDefinition, len(components))
	columnIndexByKey := make(map[string]int)
	for _, component := range components {
		componentIDs = append(componentIDs, component.ID)
		componentByID[component.ID] = component
		if _, exists := columnIndexByKey[component.Key]; !exists {
			columnIndexByKey[component.Key] = len(columns)
			columns = append(columns, models.COMarkColumn{
				Key:           component.Key,
				Label:         component.Label,
				ComponentID:   component.ID,
				ComponentName: component.Name,
				Position:      component.Order,
				MaxMarks:      component.Max,
			})
		} else if component.Max > columns[columnIndexByKey[component.Key]].MaxMarks {
			columns[columnIndexByKey[component.Key]].MaxMarks = component.Max
		}
		if testTypeName == "" {
			testTypeName = component.Name
		}
	}

	sort.SliceStable(columns, func(i, j int) bool {
		left := columnSortRank(columns[i].Label, columns[i].Position)
		right := columnSortRank(columns[j].Label, columns[j].Position)
		if left == right {
			return columns[i].Position < columns[j].Position
		}
		return left < right
	})

	columnMaxByKey := make(map[string]float64, len(columns))
	componentKeysByCOIndex := make(map[int][]string)
	for _, column := range columns {
		columnMaxByKey[column.Key] = float64(column.MaxMarks)
		if coIndex, ok := deriveCOIndex(column.Label); ok {
			componentKeysByCOIndex[coIndex] = append(componentKeysByCOIndex[coIndex], column.Key)
		}
	}

	coPoMappings := make([]coPOMappingRecord, 0)
	poIndices := make([]int, 0)
	poIndexExists := make(map[int]bool)

	coPoRows, coPoErr := r.db.Query(`
		SELECT co_index, po_index, mapping_value
		FROM co_po_mapping
		WHERE course_id = ?
		  AND mapping_value > 0
		ORDER BY po_index ASC, co_index ASC
	`, courseID)
	if coPoErr != nil {
		return nil, coPoErr
	}
	defer coPoRows.Close()

	for coPoRows.Next() {
		var mapping coPOMappingRecord
		if err := coPoRows.Scan(&mapping.COIndex, &mapping.POIndex, &mapping.MappingValue); err != nil {
			return nil, err
		}
		if mapping.COIndex <= 0 || mapping.POIndex <= 0 || mapping.MappingValue <= 0 {
			continue
		}
		coPoMappings = append(coPoMappings, mapping)
		if !poIndexExists[mapping.POIndex] {
			poIndexExists[mapping.POIndex] = true
			poIndices = append(poIndices, mapping.POIndex)
		}
	}
	if err := coPoRows.Err(); err != nil {
		return nil, err
	}

	sort.Ints(poIndices)
	poColumns := make([]models.POAttainmentColumn, 0, len(poIndices))
	for _, poIndex := range poIndices {
		poColumns = append(poColumns, models.POAttainmentColumn{
			Key:     fmt.Sprintf("PO%d", poIndex),
			Label:   fmt.Sprintf("PO%d", poIndex),
			POIndex: poIndex,
		})
	}

	mappingsByPO := make(map[int][]coPOMappingRecord, len(poIndices))
	for _, mapping := range coPoMappings {
		mappingsByPO[mapping.POIndex] = append(mappingsByPO[mapping.POIndex], mapping)
	}

	placeholders := make([]string, len(componentIDs))
	for index := range componentIDs {
		placeholders[index] = "?"
	}

	hasStudentMarksWindowColumn := r.studentMarksHasWindowColumn()

	students := make([]models.COPOAttainmentStudentRow, 0)
	rowIndexByStudentID := make(map[int]int)

	loadStudentRows := func(query string, args ...interface{}) error {
		rows, err := r.db.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row models.COPOAttainmentStudentRow

			if err := rows.Scan(
				&row.StudentID,
				&row.RegisterNo,
				&row.StudentName,
				&row.LearningMode,
				&row.CourseID,
				&row.CourseCode,
				&row.CourseName,
			); err != nil {
				return err
			}

			if _, exists := rowIndexByStudentID[row.StudentID]; exists {
				continue
			}

			row.TestTypeID = testTypeID
			row.TestTypeName = testTypeName
			row.COMarks = make(map[string]*float64, len(columns))
			rowIndexByStudentID[row.StudentID] = len(students)
			students = append(students, row)
		}

		return rows.Err()
	}

	if windowID > 0 {
		windowRosterQuery := `
			SELECT DISTINCT
				s.id AS student_id,
				COALESCE(NULLIF(s.register_no, ''), NULLIF(s.enrollment_no, ''), CONCAT('Student-', s.id)) AS register_no,
				COALESCE(s.student_name, '') AS student_name,
				COALESCE(NULLIF(TRIM(lm.code), ''), NULLIF(TRIM(lm.name), ''), '-') AS learning_mode,
				c.id AS course_id,
				COALESCE(c.course_code, '') AS course_code,
				COALESCE(c.course_name, '') AS course_name
			FROM mark_entry_student_permissions mesp
			INNER JOIN students s
				ON s.id = mesp.student_id
			INNER JOIN student_courses sc
				ON sc.student_id = s.id
			   AND sc.course_id = ?
			INNER JOIN courses c
				ON c.id = sc.course_id
			LEFT JOIN learning_modes lm
				ON lm.id = s.learning_mode_id
			LEFT JOIN academic_calendar ac
				ON ac.id = s.year
			LEFT JOIN mark_entry_windows mew
				ON mew.id = mesp.window_id
			WHERE mesp.window_id = ?
			  AND s.status = 1
			  AND (
				COALESCE(mew.semester, 0) = 0
				OR ac.current_semester = mew.semester
			)
			ORDER BY
				register_no ASC,
				student_name ASC
		`

		if err := loadStudentRows(windowRosterQuery, courseID, windowID); err != nil {
			return nil, err
		}
	}

	if len(students) == 0 {
		courseRosterQuery := `
			SELECT DISTINCT
				s.id AS student_id,
				COALESCE(NULLIF(s.register_no, ''), NULLIF(s.enrollment_no, ''), CONCAT('Student-', s.id)) AS register_no,
				COALESCE(s.student_name, '') AS student_name,
				COALESCE(NULLIF(TRIM(lm.code), ''), NULLIF(TRIM(lm.name), ''), '-') AS learning_mode,
				c.id AS course_id,
				COALESCE(c.course_code, '') AS course_code,
				COALESCE(c.course_name, '') AS course_name
			FROM student_courses sc
			INNER JOIN students s
				ON s.id = sc.student_id
			INNER JOIN courses c
				ON c.id = sc.course_id
			LEFT JOIN learning_modes lm
				ON lm.id = s.learning_mode_id
			WHERE sc.course_id = ?
			  AND s.status = 1
			ORDER BY
				register_no ASC,
				student_name ASC
		`

		if err := loadStudentRows(courseRosterQuery, courseID); err != nil {
			return nil, err
		}
	}

	if len(students) == 0 {
		allocationRosterQuery := `
			SELECT DISTINCT
				s.id AS student_id,
				COALESCE(NULLIF(s.register_no, ''), NULLIF(s.enrollment_no, ''), CONCAT('Student-', s.id)) AS register_no,
				COALESCE(s.student_name, '') AS student_name,
				COALESCE(NULLIF(TRIM(lm.code), ''), NULLIF(TRIM(lm.name), ''), '-') AS learning_mode,
				c.id AS course_id,
				COALESCE(c.course_code, '') AS course_code,
				COALESCE(c.course_name, '') AS course_name
			FROM (
				SELECT DISTINCT student_id, course_id
				FROM course_student_teacher_allocation
				WHERE course_id = ?
				  AND status = 1
			) mapped
			INNER JOIN students s
				ON s.id = mapped.student_id
			LEFT JOIN learning_modes lm
				ON lm.id = s.learning_mode_id
			INNER JOIN courses c
				ON c.id = mapped.course_id
			WHERE s.status = 1
			ORDER BY
				register_no ASC,
				student_name ASC
		`

		if err := loadStudentRows(allocationRosterQuery, courseID); err != nil {
			return nil, err
		}
	}

	loadStudentsFromMarks := func(windowScoped bool) (int, error) {
		beforeCount := len(students)
		args := make([]interface{}, 0, len(componentIDs)+2)
		args = append(args, courseID)

		windowFilter := ""
		if hasStudentMarksWindowColumn && windowScoped && windowID > 0 {
			windowFilter = "AND (sm.window_id = ? OR sm.window_id = 0)"
			args = append(args, windowID)
		}
		args = append(args, componentIDs...)

		markStudentQuery := fmt.Sprintf(`
			SELECT DISTINCT
				s.id AS student_id,
				COALESCE(NULLIF(s.register_no, ''), NULLIF(s.enrollment_no, ''), CONCAT('Student-', s.id)) AS register_no,
				COALESCE(s.student_name, '') AS student_name,
				COALESCE(NULLIF(TRIM(lm.code), ''), NULLIF(TRIM(lm.name), ''), '-') AS learning_mode,
				c.id AS course_id,
				COALESCE(c.course_code, '') AS course_code,
				COALESCE(c.course_name, '') AS course_name
			FROM student_marks sm
			INNER JOIN students s
				ON s.id = sm.student_id
			INNER JOIN courses c
				ON c.id = sm.course_id
			LEFT JOIN learning_modes lm
				ON lm.id = s.learning_mode_id
			WHERE sm.course_id = ?
			  AND sm.status = 1
			  %s
			  AND sm.assessment_component_id IN (%s)
			  AND s.status = 1
			ORDER BY
				register_no ASC,
				student_name ASC
		`, windowFilter, strings.Join(placeholders, ","))

		if err := loadStudentRows(markStudentQuery, args...); err != nil {
			return 0, err
		}

		return len(students) - beforeCount, nil
	}

	if _, err := loadStudentsFromMarks(windowID > 0); err != nil {
		return nil, err
	}
	if windowID > 0 {
		if _, err := loadStudentsFromMarks(false); err != nil {
			return nil, err
		}
	}

	if len(students) == 0 {
		return &models.COPOAttainmentResponse{
			Columns:       columns,
			POColumns:     poColumns,
			Students:      []models.COPOAttainmentStudentRow{},
			POSummary:     []models.POAttainmentSummary{},
			TargetPercent: targetPercent,
			PresentCount:  0,
			AbsentCount:   0,
		}, nil
	}

	absentStudentIDs := make(map[int]bool)
	if selectedTestFamily := normalizeAssessmentFamily(testTypeName); selectedTestFamily != "" {
		absentQuery := `
			SELECT
				ea.window_id,
				ea.student_id,
				COALESCE(mct_stored.name, ''),
				COALESCE(mew.end_at, ea.created_at)
			FROM cms.exam_absentees ea
			INNER JOIN mark_category_types mct_stored
				ON mct_stored.id = ea.mark_category_id
			LEFT JOIN mark_entry_windows mew
				ON mew.id = ea.window_id
			WHERE ea.course_id = ?
		`

		absentRows, absentErr := r.db.Query(absentQuery, courseID)
		if absentErr != nil {
			return nil, absentErr
		}
		defer absentRows.Close()

		type absenteeWindowBucket struct {
			studentIDs map[int]bool
			sortAt     time.Time
		}

		buckets := make(map[int]*absenteeWindowBucket)
		for absentRows.Next() {
			var absenteeWindowID int
			var studentID int
			var storedCategoryName string
			var sortAt time.Time
			if err := absentRows.Scan(&absenteeWindowID, &studentID, &storedCategoryName, &sortAt); err != nil {
				return nil, err
			}
			if selectedTestFamily != "" && normalizeAssessmentFamily(storedCategoryName) != selectedTestFamily {
				continue
			}

			bucket := buckets[absenteeWindowID]
			if bucket == nil {
				bucket = &absenteeWindowBucket{
					studentIDs: make(map[int]bool),
					sortAt:     sortAt,
				}
				buckets[absenteeWindowID] = bucket
			}
			if sortAt.After(bucket.sortAt) {
				bucket.sortAt = sortAt
			}
			bucket.studentIDs[studentID] = true
		}

		if err := absentRows.Err(); err != nil {
			return nil, err
		}

		if len(buckets) > 0 {
			if bucket, exists := buckets[windowID]; exists && windowID > 0 {
				absentStudentIDs = bucket.studentIDs
			} else {
				var latestWindowID int
				var latestSortAt time.Time
				for candidateWindowID, bucket := range buckets {
					if latestWindowID == 0 || bucket.sortAt.After(latestSortAt) || (bucket.sortAt.Equal(latestSortAt) && candidateWindowID > latestWindowID) {
						latestWindowID = candidateWindowID
						latestSortAt = bucket.sortAt
					}
				}
				if latestWindowID > 0 {
					absentStudentIDs = buckets[latestWindowID].studentIDs
				}
			}
		}

		if len(absentStudentIDs) > 0 {
			cohortAbsentStudentIDs := make(map[int]bool, len(absentStudentIDs))
			for studentID := range absentStudentIDs {
				if _, exists := rowIndexByStudentID[studentID]; exists {
					cohortAbsentStudentIDs[studentID] = true
				}
			}
			absentStudentIDs = cohortAbsentStudentIDs
		}
	}

	loadMarks := func(windowScoped bool) (int, error) {
		markArgs := make([]interface{}, 0, len(componentIDs)+2)
		markArgs = append(markArgs, courseID)

		windowFilter := ""
		if hasStudentMarksWindowColumn && windowScoped && windowID > 0 {
			windowFilter = "AND (sm.window_id = ? OR sm.window_id = 0)"
			markArgs = append(markArgs, windowID)
		}
		markArgs = append(markArgs, componentIDs...)

		marksQuery := fmt.Sprintf(`
			SELECT
				sm.student_id,
				sm.assessment_component_id,
				sm.obtained_marks
			FROM student_marks sm
			WHERE sm.course_id = ?
			  AND sm.status = 1
			  %s
			  AND sm.assessment_component_id IN (%s)
			ORDER BY sm.id DESC
		`, windowFilter, strings.Join(placeholders, ","))

		markRows, err := r.db.Query(marksQuery, markArgs...)
		if err != nil {
			return 0, err
		}
		defer markRows.Close()

		loadedCount := 0
		for markRows.Next() {
			var studentID int
			var componentID int
			var obtainedMarks sql.NullFloat64
			if err := markRows.Scan(&studentID, &componentID, &obtainedMarks); err != nil {
				return 0, err
			}

			rowIndex, exists := rowIndexByStudentID[studentID]
			if !exists {
				continue
			}

			component, exists := componentByID[componentID]
			if !exists {
				continue
			}

			// Keep the first non-null mark because rows are ordered by latest id DESC.
			if students[rowIndex].COMarks[component.Key] != nil {
				continue
			}
			if obtainedMarks.Valid {
				value := obtainedMarks.Float64
				students[rowIndex].COMarks[component.Key] = &value
				loadedCount++
			}
		}

		if err := markRows.Err(); err != nil {
			return 0, err
		}
		return loadedCount, nil
	}

	if _, err := loadMarks(windowID > 0); err != nil {
		return nil, err
	}
	// Backward compatibility: keep selected-window marks first, then fill any
	// still-empty cells from legacy/non-windowed mark rows.
	if windowID > 0 {
		if _, err := loadMarks(false); err != nil {
			return nil, err
		}
	}

	for index := range students {
		var total float64
		hasTotal := false
		for _, mark := range students[index].COMarks {
			if mark == nil {
				continue
			}
			total += *mark
			hasTotal = true
		}
		if hasTotal {
			totalCopy := total
			students[index].TotalMarks = &totalCopy
		}
	}

	for index := range students {
		if len(poColumns) == 0 {
			continue
		}

		studentPO := make(map[string]*float64, len(poColumns))
		for _, poColumn := range poColumns {
			poMappings := mappingsByPO[poColumn.POIndex]
			if len(poMappings) == 0 {
				continue
			}

			weightedCOPercentTotal := 0.0
			attainedWeightTotal := 0.0
			weightTotal := 0.0
			for _, mapping := range poMappings {
				componentKeys := componentKeysByCOIndex[mapping.COIndex]
				if len(componentKeys) == 0 {
					continue
				}

				weight := float64(mapping.MappingValue)
				for _, componentKey := range componentKeys {
					obtained := students[index].COMarks[componentKey]
					maxMarks := columnMaxByKey[componentKey]
					if obtained == nil || maxMarks <= 0 {
						continue
					}

					coPercent := (*obtained / maxMarks) * 100
					weightedCOPercentTotal += coPercent * weight
					weightTotal += weight
					if targetPercent > 0 && coPercent >= targetPercent {
						attainedWeightTotal += weight
					}
				}
			}

			if weightTotal <= 0 {
				continue
			}

			poPercent := 0.0
			if targetPercent > 0 {
				poPercent = (attainedWeightTotal / weightTotal) * 100
			} else {
				poPercent = weightedCOPercentTotal / weightTotal
			}

			poPercentCopy := poPercent
			studentPO[poColumn.Key] = &poPercentCopy
		}

		if len(studentPO) > 0 {
			students[index].POAttainment = studentPO
		}
	}

	presentStudents := make([]models.COPOAttainmentStudentRow, 0, len(students))
	for _, student := range students {
		if absentStudentIDs[student.StudentID] {
			continue
		}
		presentStudents = append(presentStudents, student)
	}

	poSummary := make([]models.POAttainmentSummary, 0, len(poColumns))
	for _, poColumn := range poColumns {
		total := 0.0
		count := 0
		for _, student := range presentStudents {
			if student.POAttainment == nil {
				continue
			}
			if value := student.POAttainment[poColumn.Key]; value != nil {
				total += *value
				count++
			}
		}

		average := 0.0
		if count > 0 {
			average = total / float64(count)
		}

		poSummary = append(poSummary, models.POAttainmentSummary{
			Key:               poColumn.Key,
			Label:             poColumn.Label,
			POIndex:           poColumn.POIndex,
			AttainmentPercent: average,
		})
	}

	return &models.COPOAttainmentResponse{
		Columns:       columns,
		POColumns:     poColumns,
		Students:      presentStudents,
		POSummary:     poSummary,
		TargetPercent: targetPercent,
		PresentCount:  len(presentStudents),
		AbsentCount:   len(absentStudentIDs),
	}, nil
}
