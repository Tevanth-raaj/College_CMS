package models

type Teamwork struct {
	Activities []string `json:"activities"`
	Hours      int      `json:"hours"`
}

type SelfLearningInternal struct {
	Main     string   `json:"main"`
	Internal []string `json:"internal"`
}

type SelfLearning struct {
	MainInputs []SelfLearningInternal `json:"main_inputs"`
	Hours      int                    `json:"hours"`
}

// Syllabus represents the complete syllabus for API responses
// Data is fetched from normalized tables but presented in the same JSON structure
type Syllabus struct {
	ID                    int           `json:"id"`
	CourseID              int           `json:"course_id"`
	Objectives            []string      `json:"objectives"`              // from course_objectives table
	Outcomes              []string      `json:"outcomes"`                // from course_outcomes table
	ReferenceList         []string      `json:"reference_list"`          // from course_references table
	TextbookReferenceList []string      `json:"textbook_reference_list"` // from course_textbook_reference table
	Prerequisites         []string      `json:"prerequisites"`           // from course_prerequisites table
	Teamwork              *Teamwork     `json:"teamwork,omitempty"`      // from course_teamwork + course_teamwork_activities
	SelfLearning          *SelfLearning `json:"selflearning,omitempty"`  // from course_selflearning + course_selflearning_topics + course_selflearning_resources
}
