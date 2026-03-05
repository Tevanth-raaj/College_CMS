package models

import "time"

// AcademicCalendar represents the current academic calendar
type AcademicCalendar struct {
	ID                     int        `json:"id"`
	AcademicYear           string     `json:"academic_year"`
	YearLevel              int        `json:"year_level"`
	CurrentSemester        int        `json:"current_semester"`
	Batch                  *string    `json:"batch,omitempty"`
	SemesterStartDate      time.Time  `json:"semester_start_date"`
	SemesterEndDate        time.Time  `json:"semester_end_date"`
	ElectiveSelectionStart *time.Time `json:"elective_selection_start,omitempty"`
	ElectiveSelectionEnd   *time.Time `json:"elective_selection_end,omitempty"`
	IsCurrent              bool       `json:"is_current"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// VerticalLock represents a locked honour/minor vertical for a department+batch
type VerticalLock struct {
	ID                int    `json:"id"`
	DepartmentID      int    `json:"department_id"`
	Batch             string `json:"batch"`
	LockType          string `json:"lock_type"` // "honour" or "minor"
	VerticalID        int    `json:"vertical_id"`
	VerticalName      string `json:"vertical_name"`
	LockedByUserID    int    `json:"locked_by_user_id"`
	FirstSemester     int    `json:"first_semester"`
	FirstAcademicYear string `json:"first_academic_year"`
}

// HODElectiveSelection represents HOD's elective course selections
type HODElectiveSelection struct {
	ID               int       `json:"id"`
	DepartmentID     int       `json:"department_id"`
	CurriculumID     int       `json:"curriculum_id"`
	Semester         int       `json:"semester"`
	CourseID         int       `json:"course_id"`
	AcademicYear     string    `json:"academic_year"`
	Batch            *string   `json:"batch,omitempty"`
	MaxStudents      *int      `json:"max_students,omitempty"`
	ApprovedByUserID int       `json:"approved_by_user_id"`
	Status           string    `json:"status"` // ACTIVE, INACTIVE, DRAFT
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// StudentElectiveChoice represents student's elective selections
type StudentElectiveChoice struct {
	ID             int        `json:"id"`
	StudentID      int        `json:"student_id"`
	HODSelectionID int        `json:"hod_selection_id"`
	Semester       int        `json:"semester"`
	AcademicYear   string     `json:"academic_year"`
	ChoiceOrder    int        `json:"choice_order"`
	Status         string     `json:"status"` // PENDING, CONFIRMED, REJECTED, WAITLISTED
	SelectedAt     time.Time  `json:"selected_at"`
	ConfirmedAt    *time.Time `json:"confirmed_at,omitempty"`
}

// DepartmentCurriculumActive maps departments to active curricula
type DepartmentCurriculumActive struct {
	ID           int       `json:"id"`
	DepartmentID int       `json:"department_id"`
	CurriculumID int       `json:"curriculum_id"`
	AcademicYear string    `json:"academic_year"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// ==================== DTOs for API Responses ====================

// HODProfileResponse - Response for HOD profile API
type HODProfileResponse struct {
	UserID     int             `json:"user_id"`
	FullName   string          `json:"full_name"`
	Email      string          `json:"email"`
	Role       string          `json:"role"`
	Department *DepartmentInfo `json:"department"`
	Curriculum *CurriculumInfo `json:"curriculum"`
}

type DepartmentInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}

type CurriculumInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	AcademicYear string `json:"academic_year"`
}

// ElectiveCourse - Course with selection status for HOD
type ElectiveCourse struct {
	ID               int     `json:"id"`
	CourseCode       string  `json:"course_code"`
	CourseName       string  `json:"course_name"`
	CourseType       string  `json:"course_type"`
	Category         string  `json:"category"`
	Credit           int     `json:"credit"`
	CardID           int     `json:"card_id"`
	VerticalID       int     `json:"vertical_id"`
	VerticalName     string  `json:"vertical_name"`
	VerticalSemester *int    `json:"vertical_semester,omitempty"`
	CardType         string  `json:"card_type"`
	IsSelected       bool    `json:"is_selected"`
	AssignedSemester *int    `json:"assigned_semester,omitempty"`
	AssignedSlotID   *int    `json:"assigned_slot_id,omitempty"`
	AssignedSlot     *string `json:"assigned_slot,omitempty"`
}

// MinorEligibleCourse - Course from another department's PE selections, available for minor
type MinorEligibleCourse struct {
	ID             int    `json:"id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	CourseType     string `json:"course_type"`
	Category       string `json:"category"`
	Credit         int    `json:"credit"`
	DepartmentID   int    `json:"department_id"`
	DepartmentName string `json:"department_name"`
	DepartmentCode string `json:"department_code"`
	SlotName       string `json:"slot_name"`
	Semester       int    `json:"semester"`
}

// AvailableElectivesResponse - Response for available electives API
type AvailableElectivesResponse struct {
	Semester              int                   `json:"semester"`
	Batch                 string                `json:"batch,omitempty"`
	AcademicYear          string                `json:"academic_year"`
	AvailableElectives    []ElectiveCourse      `json:"available_electives"`
	MinorEligibleCourses  []MinorEligibleCourse `json:"minor_eligible_courses"`
}

// SaveElectivesRequest - Request to save HOD selections
type SaveElectivesRequest struct {
	Semester          int                `json:"semester"`
	Batch             string             `json:"batch,omitempty"`
	AcademicYear      string             `json:"academic_year"`
	SelectedCourses   []int              `json:"selected_courses"`
	CourseAssignments []CourseAssignment `json:"course_assignments"`
	Status            string             `json:"status"` // ACTIVE or DRAFT
}

// CourseAssignment - Maps course to semester
type CourseAssignment struct {
	CourseID int `json:"course_id"`
	Semester int `json:"semester"`
	SlotID   int `json:"slot_id"`
}

// SaveElectivesResponse - Response after saving HOD selections
type SaveElectivesResponse struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	Selections []HODElectiveSelection `json:"selections,omitempty"`
}

// BatchInfo - Batch information
type BatchInfo struct {
	Batch        string `json:"batch"`
	StudentCount int    `json:"student_count"`
}

// BatchesResponse - Response for batches API
type BatchesResponse struct {
	Batches []string `json:"batches"`
}

// ElectiveSemesterSlot represents available slots for a semester
type ElectiveSemesterSlot struct {
	ID        int    `json:"id"`
	Semester  int    `json:"semester"`
	SlotName  string `json:"slot_name"`
	SlotOrder int    `json:"slot_order"`
}

// SelectedElectivesResponse - Response for getting HOD's selected electives
type SelectedElectivesResponse struct {
	Semester        int              `json:"semester"`
	Batch           string           `json:"batch,omitempty"`
	AcademicYear    string           `json:"academic_year"`
	SelectedCourses []ElectiveCourse `json:"selected_courses"`
}

// StudentElectiveCourse - Course for student view
type StudentElectiveCourse struct {
	SelectionID        int    `json:"selection_id"`
	CourseCode         string `json:"course_code"`
	CourseName         string `json:"course_name"`
	CourseType         string `json:"course_type"`
	Category           string `json:"category"`
	Credit             int    `json:"credit"`
	MaxStudents        *int   `json:"max_students,omitempty"`
	CurrentEnrollments int    `json:"current_enrollments"`
	IsChosen           bool   `json:"is_chosen"`
}

// StudentAvailableElectivesResponse - Response for student available electives
type StudentAvailableElectivesResponse struct {
	Semester  int                     `json:"semester"`
	Electives []StudentElectiveCourse `json:"electives"`
}

// StudentElectiveSelectionRequest - Request for student to choose electives
type StudentElectiveSelectionRequest struct {
	Semester   int                        `json:"semester"`
	Selections []StudentElectiveSelection `json:"selections"`
}

type StudentElectiveSelection struct {
	HODSelectionID int `json:"hod_selection_id"`
	ChoiceOrder    int `json:"choice_order"`
}

// ==================== Minor Program Models ====================

// HODMinorSelection represents HOD's minor program selections
type HODMinorSelection struct {
	ID               int       `json:"id"`
	DepartmentID     int       `json:"department_id"`
	CurriculumID     int       `json:"curriculum_id"`
	VerticalID       int       `json:"vertical_id"`
	Semester         int       `json:"semester"`
	CourseID         int       `json:"course_id"`
	AllowedDeptIDs   []int     `json:"allowed_dept_ids"`
	AcademicYear     string    `json:"academic_year"`
	Batch            *string   `json:"batch,omitempty"`
	ApprovedByUserID int       `json:"approved_by_user_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// MinorVerticalInfo - Vertical information for minor program selection
type MinorVerticalInfo struct {
	ID           int           `json:"id"`
	HonourCardID int           `json:"honour_card_id"`
	Name         string        `json:"name"`
	CourseCount  int           `json:"course_count"`
	Courses      []MinorCourse `json:"courses,omitempty"`
}

// MinorCourse - Course info for minor programs
type MinorCourse struct {
	ID         int    `json:"id"`
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Credit     int    `json:"credit"`
}

// SaveMinorRequest - Request to save HOD minor program selections
type SaveMinorRequest struct {
	VerticalID          int                       `json:"vertical_id"`
	AllowedDeptIDs      []int                     `json:"allowed_dept_ids"`
	AcademicYear        string                    `json:"academic_year"`
	Batch               string                    `json:"batch,omitempty"`
	SemesterAssignments []MinorSemesterAssignment `json:"semester_assignments"`
	Status              string                    `json:"status"`
}

// MinorSemesterAssignment - Maps course to semester
type MinorSemesterAssignment struct {
	Semester int `json:"semester"`
	CourseID int `json:"course_id"`
}

// SaveMinorResponse - Response after saving HOD minor selections
type SaveMinorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MinorSelectionResponse - Response for getting HOD's minor selections
type MinorSelectionResponse struct {
	VerticalID   int                   `json:"vertical_id"`
	VerticalName string                `json:"vertical_name"`
	AcademicYear string                `json:"academic_year"`
	Batch        string                `json:"batch,omitempty"`
	Assignments  []MinorAssignmentInfo `json:"assignments"`
}

// MinorAssignmentInfo - Info about a minor course assignment
type MinorAssignmentInfo struct {
	Semester       int    `json:"semester"`
	CourseID       int    `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	Credit         int    `json:"credit"`
	AllowedDeptIDs []int  `json:"allowed_dept_ids"`
}

// ==================== Open Elective Offering Models ====================

// HODOEOffering represents HOD's open elective course offering to other departments
type HODOEOffering struct {
	ID               int       `json:"id"`
	DepartmentID     int       `json:"department_id"`
	CurriculumID     int       `json:"curriculum_id"`
	OECardID         int       `json:"oe_card_id"`
	Semester         int       `json:"semester"`
	CourseID         int       `json:"course_id"`
	AllowedDeptIDs   []int     `json:"allowed_dept_ids"`
	AcademicYear     string    `json:"academic_year"`
	Batch            *string   `json:"batch,omitempty"`
	ApprovedByUserID int       `json:"approved_by_user_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// OECardInfo - OE card information for open elective offering
type OECardInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	VerticalName string `json:"vertical_name,omitempty"`
	Semester     *int   `json:"semester,omitempty"`
	CourseCount  int    `json:"course_count"`
}

// SaveOEOfferingRequest - Request to save HOD OE offerings
type SaveOEOfferingRequest struct {
	OECardID            int                    `json:"oe_card_id"`
	AllowedDeptIDs      []int                  `json:"allowed_dept_ids"`
	AcademicYear        string                 `json:"academic_year"`
	Batch               string                 `json:"batch,omitempty"`
	SemesterAssignments []OESemesterAssignment `json:"semester_assignments"`
	Status              string                 `json:"status"`
}

// OESemesterAssignment - Maps OE course to semester
type OESemesterAssignment struct {
	Semester int `json:"semester"`
	CourseID int `json:"course_id"`
}

// SaveOEOfferingResponse - Response after saving OE offerings
type SaveOEOfferingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// OEOfferingResponse - Response for getting HOD's OE offerings
type OEOfferingResponse struct {
	OECardID     int                `json:"oe_card_id"`
	OECardName   string             `json:"oe_card_name"`
	AcademicYear string             `json:"academic_year"`
	Batch        string             `json:"batch,omitempty"`
	Assignments  []OEAssignmentInfo `json:"assignments"`
}

// OEAssignmentInfo - Info about an OE course offering assignment
type OEAssignmentInfo struct {
	Semester       int    `json:"semester"`
	CourseID       int    `json:"course_id"`
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	Credit         int    `json:"credit"`
	AllowedDeptIDs []int  `json:"allowed_dept_ids"`
}

// Department - Basic department info
type Department struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}
