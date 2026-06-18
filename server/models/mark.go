package models

// MarkCategoryType represents a mark assessment component/category for courses
type MarkCategoryType struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	MaxMarks           int     `json:"max_marks"`
	PerEntryMaxMarks   int     `json:"per_entry_max_marks,omitempty"`
	EntryRepeatCount   int     `json:"entry_repeat_count,omitempty"`
	EffectiveMaxMarks  int     `json:"effective_max_marks,omitempty"`
	ExperimentCountTWL int     `json:"experiment_count_theorywithlab,omitempty"`
	ConversionMarks    float64 `json:"conversion_marks"`
	Position           int     `json:"position"`
	CourseTypeID       int     `json:"course_type_id"`
	CourseTypeName     string  `json:"course_type_name"`
	CategoryNameID     int     `json:"category_name_id"`
	CategoryName       string  `json:"category_name"`
	LearningModeID     int     `json:"learning_mode_id"`
	Status             int     `json:"status"`
}

// MarkCategoryName represents the name/label of a mark category
type MarkCategoryName struct {
	ID           int    `json:"id"`
	CategoryName string `json:"category_name"`
	Status       int    `json:"status"`
}

// TestTypeOption represents a selectable assessment component for CO-PO attainment.
type TestTypeOption struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	MaxMarks int    `json:"max_marks"`
	Position int    `json:"position"`
}

// COMarkColumn represents a dynamic CO/component column for the attainment grid.
type COMarkColumn struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	ComponentID   int    `json:"component_id"`
	ComponentName string `json:"component_name"`
	Position      int    `json:"position"`
	MaxMarks      int    `json:"max_marks"`
}

// POAttainmentColumn represents a dynamic PO column for computed attainment.
type POAttainmentColumn struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	POIndex int    `json:"po_index"`
}

// POAttainmentSummary represents aggregate PO attainment across present students.
type POAttainmentSummary struct {
	Key               string  `json:"key"`
	Label             string  `json:"label"`
	POIndex           int     `json:"po_index"`
	AttainmentPercent float64 `json:"attainment_percent"`
}

// COPOAttainmentStudentRow represents a course-wise student row for the CO-PO attainment table.
type COPOAttainmentStudentRow struct {
	StudentID    int                 `json:"student_id"`
	RegisterNo   string              `json:"register_no"`
	StudentName  string              `json:"student_name"`
	LearningMode string              `json:"learning_mode"`
	CourseID     int                 `json:"course_id"`
	CourseCode   string              `json:"course_code"`
	CourseName   string              `json:"course_name"`
	TestTypeID   int                 `json:"test_type_id"`
	TestTypeName string              `json:"test_type_name"`
	COMarks      map[string]*float64 `json:"co_marks"`
	POAttainment map[string]*float64 `json:"po_attainment,omitempty"`
	TotalMarks   *float64            `json:"total_marks,omitempty"`
}

// COPOAttainmentResponse represents the full table response with dynamic CO columns.
type COPOAttainmentResponse struct {
	Columns       []COMarkColumn             `json:"columns"`
	POColumns     []POAttainmentColumn       `json:"po_columns,omitempty"`
	Students      []COPOAttainmentStudentRow `json:"students"`
	POSummary     []POAttainmentSummary      `json:"po_summary,omitempty"`
	TargetPercent float64                    `json:"target_percent"`
	PresentCount  int                        `json:"present_count"`
	AbsentCount   int                        `json:"absent_count"`
}

// StudentMark represents a student's mark entry for an assessment component
type StudentMark struct {
	ID                    int     `json:"id"`
	StudentID             int     `json:"student_id"`
	CourseID              int     `json:"course_id"`
	FacultyID             string  `json:"faculty_id"`
	AssessmentComponentID int     `json:"assessment_component_id"`
	ObtainedMarks         float64 `json:"obtained_marks"`
	ConvertedMarks        float64 `json:"converted_marks"`
	Status                int     `json:"status"`
}

// StudentMarkEntry is used for bulk save requests
type StudentMarkEntry struct {
	StudentID             int     `json:"student_id"`
	CourseID              int     `json:"course_id"`
	AssessmentComponentID int     `json:"assessment_component_id"`
	ObtainedMarks         float64 `json:"obtained_marks"`
}

// StudentMarkDeleteEntry is used for bulk delete requests when marks are cleared.
type StudentMarkDeleteEntry struct {
	StudentID             int `json:"student_id"`
	CourseID              int `json:"course_id"`
	AssessmentComponentID int `json:"assessment_component_id"`
}

// MarkEntrySaveRequest contains batch mark entries to save
type MarkEntrySaveRequest struct {
	CourseID      int                      `json:"course_id"`
	FacultyID     string                   `json:"faculty_id"`
	WindowID      int                      `json:"window_id,omitempty"`
	MarkEntries   []StudentMarkEntry       `json:"mark_entries"`
	DeleteEntries []StudentMarkDeleteEntry `json:"delete_entries,omitempty"`
}

// MarkEntrySaveResponse is returned after saving marks
type MarkEntrySaveResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	SavedCount int    `json:"saved_count"`
}

// MarkEntryPermissionCategory represents a mark category with enabled state for a course and teacher.
type MarkEntryPermissionCategory struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	MaxMarks        int     `json:"max_marks"`
	ConversionMarks float64 `json:"conversion_marks"`
	Position        int     `json:"position"`
	CourseTypeID    int     `json:"course_type_id"`
	CategoryNameID  int     `json:"category_name_id"`
	LearningModeID  int     `json:"learning_mode_id"`
	Status          int     `json:"status"`
	Enabled         bool    `json:"enabled"`
}

// MarkEntryPermissionUpdate represents a single permission update entry.
type MarkEntryPermissionUpdate struct {
	AssessmentComponentID int  `json:"assessment_component_id"`
	Enabled               bool `json:"enabled"`
}

// MarkEntryPermissionUpdateRequest represents a request to update permissions.
type MarkEntryPermissionUpdateRequest struct {
	CourseID    int                         `json:"course_id"`
	TeacherID   string                      `json:"teacher_id"`
	Permissions []MarkEntryPermissionUpdate `json:"permissions"`
}

// MarkEntryWindow represents a mark entry open window rule.
type MarkEntryWindow struct {
	ID           int     `json:"id"`
	ScopeType    string  `json:"scope_type,omitempty"`
	TeacherID    *string `json:"teacher_id,omitempty"`
	DepartmentID *int    `json:"department_id,omitempty"`
	Semester     *int    `json:"semester,omitempty"`
	CourseID     *int    `json:"course_id,omitempty"`
	StartAt      string  `json:"start_at"`
	EndAt        string  `json:"end_at"`
	Enabled      bool    `json:"enabled"`
	WindowName   string  `json:"window_name,omitempty"`
	ComponentIDs []int   `json:"component_ids,omitempty"` // Empty = all components allowed
}

// MarkEntryWindowRequest represents a create/update window request.
type MarkEntryWindowRequest struct {
	ScopeType     string   `json:"scope_type,omitempty"`
	TeacherID     *string  `json:"teacher_id,omitempty"`
	UserID        *string  `json:"user_id,omitempty"` // For user-based windows
	UserIDs       []string `json:"user_ids,omitempty"`
	DepartmentID  *int     `json:"department_id,omitempty"`
	DepartmentIDs []int    `json:"department_ids,omitempty"`
	Semester      *int     `json:"semester,omitempty"`
	CourseID      *int     `json:"course_id,omitempty"`
	StartAt       string   `json:"start_at"`
	EndAt         string   `json:"end_at"`
	Enabled       bool     `json:"enabled"`
	ComponentIDs  []int    `json:"component_ids,omitempty"` // Empty = all components allowed
	WindowName    string   `json:"window_name,omitempty"`
}

// StudentMarkPermission represents student-specific mark entry permission
type StudentMarkPermission struct {
	ID        int    `json:"id"`
	WindowID  int    `json:"window_id"`
	UserID    string `json:"user_id"`
	StudentID int    `json:"student_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	CreatedBy string `json:"created_by,omitempty"`
}

// CreateUserStudentWindowRequest represents a request to create a user-student mark entry window
type CreateUserStudentWindowRequest struct {
	UserID        string   `json:"user_id"`
	UserIDs       []string `json:"user_ids,omitempty"`
	DepartmentID  *int     `json:"department_id,omitempty"`
	DepartmentIDs []int    `json:"department_ids,omitempty"`
	Semester      *int     `json:"semester,omitempty"`
	CourseID      *int     `json:"course_id,omitempty"`
	StudentIDs    []int    `json:"student_ids"` // Specific students for this window
	StartAt       string   `json:"start_at"`
	EndAt         string   `json:"end_at"`
	ComponentIDs  []int    `json:"component_ids,omitempty"` // PBL/UAL components
	CreatedBy     string   `json:"created_by,omitempty"`
	WindowName    string   `json:"window_name,omitempty"`
}

// CreateUserStudentWindowResponse represents the response after creating user-student window
type CreateUserStudentWindowResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	WindowID           int    `json:"window_id"`
	WindowIDs          []int  `json:"window_ids,omitempty"`
	WindowsCreated     int    `json:"windows_created,omitempty"`
	AssignmentsCreated int    `json:"assignments_created"`
}

// AssignStudentsToUserRequest represents a request to assign students to a user for mark entry
type AssignStudentsToUserRequest struct {
	WindowID   int    `json:"window_id"`
	UserID     string `json:"user_id"`
	StudentIDs []int  `json:"student_ids"`
	CreatedBy  string `json:"created_by,omitempty"`
}

// AssignStudentsToUserResponse represents the response after assigning students
type AssignStudentsToUserResponse struct {
	Success            bool   `json:"success"`
	Message            string `json:"message"`
	AssignmentsCreated int    `json:"assignments_created"`
}

// UserAssignedStudentsRequest represents a request to get assigned students for a user
type UserAssignedStudentsRequest struct {
	UserID   string `json:"user_id"`
	CourseID *int   `json:"course_id,omitempty"`
}

// AssignedStudentInfo represents detailed information about an assigned student
type AssignedStudentInfo struct {
	StudentID      int    `json:"student_id"`
	EnrollmentNo   string `json:"enrollment_no"`
	RegisterNo     string `json:"register_no"`
	StudentName    string `json:"student_name"`
	Department     string `json:"department"`
	Year           int    `json:"year"`
	WindowID       int    `json:"window_id"`
	WindowStart    string `json:"window_start"`
	WindowEnd      string `json:"window_end"`
	CourseID       *int   `json:"course_id,omitempty"`
	CourseCode     string `json:"course_code,omitempty"`
	CourseName     string `json:"course_name,omitempty"`
	LearningModeID *int   `json:"learning_mode_id"`
}
