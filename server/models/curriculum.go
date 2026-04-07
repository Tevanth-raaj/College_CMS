package models

type CourseType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Semester struct {
	ID             int    `json:"id"`
	CurriculumID   int    `json:"curriculum_id"`
	SemesterNumber *int   `json:"semester_number"`
	CardType       string `json:"card_type"`
	VerticalName   string `json:"vertical_name,omitempty"`
}

type Course struct {
	CourseID           int    `json:"id"`
	CourseCode         string `json:"course_code"`
	CourseName         string `json:"course_name"`
	CourseType         string `json:"course_type"`
	ExperimentCountTWL int    `json:"experiment_count_theorywithlab"`
	Category           string `json:"category"`
	Credit             int    `json:"credit"`
	LectureHrs         int    `json:"lecture_hrs"`
	TutorialHrs        int    `json:"tutorial_hrs"`
	PracticalHrs       int    `json:"practical_hrs"`
	ActivityHrs        int    `json:"activity_hrs"`
	TwSlHrs            int    `json:"tw_sl_hrs"`
	TheoryTotalHrs     int    `json:"theory_total_hrs"`
	TutorialTotalHrs   int    `json:"tutorial_total_hrs"`
	ActivityTotalHrs   int    `json:"activity_total_hrs"`
	PracticalTotalHrs  int    `json:"practical_total_hrs"`
	TotalHrs           int    `json:"total_hrs"`
	CIAMarks           int    `json:"cia_marks"`
	SEEMarks           int    `json:"see_marks"`
	TotalMarks         int    `json:"total_marks"`
	CountTowardsLimit  *bool  `json:"count_towards_limit,omitempty"`
	CurriculumTemplate string `json:"curriculum_template,omitempty"`
}

type RegulationCourse struct {
	ID           int `json:"id"`
	CurriculumID int `json:"curriculum_id"`
	SemesterID   int `json:"semester_id"`
	CourseID     int `json:"course_id"`
}

type CourseWithDetails struct {
	CourseID           int    `json:"id"`
	CourseCode         string `json:"course_code"`
	CourseName         string `json:"course_name"`
	CourseType         string `json:"course_type"`
	ExperimentCountTWL int    `json:"experiment_count_theorywithlab"`
	Category           string `json:"category"`
	Credit             int    `json:"credit"`
	LectureHrs         int    `json:"lecture_hrs"`
	TutorialHrs        int    `json:"tutorial_hrs"`
	PracticalHrs       int    `json:"practical_hrs"`
	ActivityHrs        int    `json:"activity_hrs"`
	TwSlHrs            int    `json:"tw_sl_hrs"`
	TheoryTotalHrs     int    `json:"theory_total_hrs"`
	TutorialTotalHrs   int    `json:"tutorial_total_hrs"`
	ActivityTotalHrs   int    `json:"activity_total_hrs"`
	PracticalTotalHrs  int    `json:"practical_total_hrs"`
	TotalHrs           int    `json:"total_hrs"`
	CIAMarks           int    `json:"cia_marks"`
	SEEMarks           int    `json:"see_marks"`
	TotalMarks         int    `json:"total_marks"`
	CountTowardsLimit  *bool  `json:"count_towards_limit,omitempty"`
	RegCourseID        int    `json:"reg_course_id"`
	CurriculumTemplate string `json:"curriculum_template,omitempty"`
}

type HonourCard struct {
	ID           int    `json:"id"`
	CurriculumID int    `json:"curriculum_id"`
	Title        string `json:"title"`
	Number       *int   `json:"number,omitempty"`
}

type HonourVertical struct {
	ID           int    `json:"id"`
	HonourCardID int    `json:"honour_card_id"`
	Name         string `json:"name"`
}

type HonourVerticalWithCourses struct {
	ID           int                 `json:"id"`
	HonourCardID int                 `json:"honour_card_id"`
	Name         string              `json:"name"`
	Courses      []CourseWithDetails `json:"courses"`
}

type HonourCardWithVerticals struct {
	ID           int                         `json:"id"`
	CurriculumID int                         `json:"curriculum_id"`
	Title        string                      `json:"title"`
	Number       *int                        `json:"number,omitempty"`
	Verticals    []HonourVerticalWithCourses `json:"verticals"`
}

// Curriculum model with template support
type Curriculum struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	AcademicYear       string `json:"academic_year"`
	MaxCredits         int    `json:"max_credits"`
	CurriculumTemplate string `json:"curriculum_template"` // "2022" or "2026"
}

// Experiment models for 2022 template
type Experiment struct {
	ID               int      `json:"id"`
	CourseID         int      `json:"course_id"`
	ExperimentNumber int      `json:"experiment_number"`
	ExperimentName   string   `json:"experiment_name"`
	Hours            int      `json:"hours"`
	Topics           []string `json:"topics"` // List of topics for this experiment
}

type ExperimentTopic struct {
	ID           int    `json:"id"`
	ExperimentID int    `json:"experiment_id"`
	TopicText    string `json:"topic_text"`
	TopicOrder   int    `json:"topic_order"`
}
