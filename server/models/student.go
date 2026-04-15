package models

import "time"

// Student - Basic student information
type Student struct {
	StudentID        int       `json:"student_id"`
	EnrollmentNo     string    `json:"enrollment_no"`
	RegisterNo       string    `json:"register_no"`
	DTERegNo         string    `json:"dte_reg_no"`
	ApplicationNo    string    `json:"application_no"`
	AdmissionNo      string    `json:"admission_no"`
	StudentName      string    `json:"student_name"`
	Gender           string    `json:"gender"`
	DOB              string    `json:"dob"`
	Age              int       `json:"age"`
	FatherName       string    `json:"father_name"`
	MotherName       string    `json:"mother_name"`
	GuardianName     string    `json:"guardian_name"`
	Religion         string    `json:"religion"`
	Nationality      string    `json:"nationality"`
	Community        string    `json:"community"`
	MotherTongue     string    `json:"mother_tongue"`
	BloodGroup       string    `json:"blood_group"`
	AadharNo         string    `json:"aadhar_no"`
	ParentOccupation string    `json:"parent_occupation"`
	Designation      string    `json:"designation"`
	PlaceOfWork      string    `json:"place_of_work"`
	ParentIncome     float64   `json:"parent_income"`
	Status           int       `json:"status"`
	LearningModeID   *int      `json:"learning_mode_id"`
	LearningMode     string    `json:"learning_mode"`
	DepartmentID     *int      `json:"department_id"`
	Year             int       `json:"year"`
	DepartmentCode   string    `json:"department_code"`
	MailID           string    `json:"mail_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AcademicDetails - Academic information
type AcademicDetails struct {
	StudentID        int    `json:"student_id"`
	Batch            string `json:"batch"`
	Year             int    `json:"year"`
	Semester         int    `json:"semester"`
	DegreeLevel      string `json:"degree_level"`
	Section          string `json:"section"`
	Department       string `json:"department"`
	StudentCategory  string `json:"student_category"`
	BranchType       string `json:"branch_type"`
	SeatCategory     string `json:"seat_category"`
	Regulation       string `json:"regulation"`
	Quota            string `json:"quota"`
	University       string `json:"university"`
	YearOfAdmission  int    `json:"year_of_admission"`
	YearOfCompletion int    `json:"year_of_completion"`
	StudentStatus    string `json:"student_status"`
	CurriculumID     int    `json:"curriculum_id"`
}

// Address - Address information
type Address struct {
	StudentID         int    `json:"student_id"`
	PermanentAddress  string `json:"permanent_address"`
	PresentAddress    string `json:"present_address"`
	ResidenceLocation string `json:"residence_location"`
}

// AdmissionPayment - Admission payment information
type AdmissionPayment struct {
	StudentID      int     `json:"student_id"`
	DTERegisterNo  string  `json:"dte_register_no"`
	DTEAdmissionNo string  `json:"dte_admission_no"`
	ReceiptNo      string  `json:"receipt_no"`
	ReceiptDate    string  `json:"receipt_date"`
	Amount         float64 `json:"amount"`
	BankName       string  `json:"bank_name"`
}

// ContactDetails - Contact information
type ContactDetails struct {
	StudentID     int    `json:"student_id"`
	ParentMobile  string `json:"parent_mobile"`
	StudentMobile string `json:"student_mobile"`
	StudentEmail  string `json:"student_email"`
	ParentEmail   string `json:"parent_email"`
	OfficialEmail string `json:"official_email"`
}

// HostelDetails - Hostel information
type HostelDetails struct {
	StudentID       int    `json:"student_id"`
	HostellerType   string `json:"hosteller_type"`
	HostelName      string `json:"hostel_name"`
	RoomNo          string `json:"room_no"`
	RoomCapacity    int    `json:"room_capacity"`
	RoomType        string `json:"room_type"`
	FloorNo         int    `json:"floor_no"`
	WardenName      string `json:"warden_name"`
	AlternateWarden string `json:"alternate_warden"`
	ClassAdvisor    string `json:"class_advisor"`
	Status          int    `json:"status"`
}

// InsuranceDetails - Insurance information
type InsuranceDetails struct {
	StudentID    int    `json:"student_id"`
	NomineeName  string `json:"nominee_name"`
	Relationship string `json:"relationship"`
	NomineeAge   int    `json:"nominee_age"`
	Status       int    `json:"status"`
}

// SchoolDetails - School information
type SchoolDetails struct {
	ID         int     `json:"id"`
	StudentID  int     `json:"student_id"`
	SchoolName string  `json:"school_name"`
	Board      string  `json:"board"`
	YearOfPass int     `json:"year_of_pass"`
	State      string  `json:"state"`
	TCNo       string  `json:"tc_no"`
	TCDate     string  `json:"tc_date"`
	TotalMarks float64 `json:"total_marks"`
	Status     int     `json:"status"`
}

// FullStudent - Complete student with all related information
type FullStudent struct {
	Student          *Student          `json:"student"`
	AcademicDetails  *AcademicDetails  `json:"academic_details"`
	Address          *Address          `json:"address"`
	AdmissionPayment *AdmissionPayment `json:"admission_payment"`
	ContactDetails   *ContactDetails   `json:"contact_details"`
	HostelDetails    *HostelDetails    `json:"hostel_details"`
	InsuranceDetails *InsuranceDetails `json:"insurance_details"`
	SchoolDetails    []SchoolDetails   `json:"school_details"`
}

// CreateStudentRequest - Request to create student with all related details
type CreateStudentRequest struct {
	// Basic Student Fields
	EnrollmentNo     string `json:"enrollment_no"`
	RegisterNo       string `json:"register_no"`
	DTERegNo         string `json:"dte_reg_no"`
	ApplicationNo    string `json:"application_no"`
	AdmissionNo      string `json:"admission_no"`
	StudentName      string `json:"student_name"`
	Gender           string `json:"gender"`
	DOB              string `json:"dob"`
	Age              string `json:"age"`
	FatherName       string `json:"father_name"`
	MotherName       string `json:"mother_name"`
	GuardianName     string `json:"guardian_name"`
	Religion         string `json:"religion"`
	Nationality      string `json:"nationality"`
	Community        string `json:"community"`
	MotherTongue     string `json:"mother_tongue"`
	BloodGroup       string `json:"blood_group"`
	AadharNo         string `json:"aadhar_no"`
	ParentOccupation string `json:"parent_occupation"`
	Designation      string `json:"designation"`
	PlaceOfWork      string `json:"place_of_work"`
	ParentIncome     string `json:"parent_income"`

	// Academic Details Fields
	Batch            string `json:"batch"`
	Year             string `json:"year"`
	Semester         string `json:"semester"`
	DegreeLevel      string `json:"degree_level"`
	Section          string `json:"section"`
	Department       string `json:"department"`
	StudentCategory  string `json:"student_category"`
	BranchType       string `json:"branch_type"`
	SeatCategory     string `json:"seat_category"`
	Regulation       string `json:"regulation"`
	Quota            string `json:"quota"`
	University       string `json:"university"`
	YearOfAdmission  string `json:"year_of_admission"`
	YearOfCompletion string `json:"year_of_completion"`
	StudentStatus    string `json:"student_status"`
	CurriculumID     string `json:"curriculum_id"`

	// Address Fields
	PermanentAddress  string `json:"permanent_address"`
	PresentAddress    string `json:"present_address"`
	ResidenceLocation string `json:"residence_location"`

	// Admission Payment Fields
	DTERegisterNo  string `json:"dte_register_no"`
	DTEAdmissionNo string `json:"dte_admission_no"`
	ReceiptNo      string `json:"receipt_no"`
	ReceiptDate    string `json:"receipt_date"`
	Amount         string `json:"amount"`
	BankName       string `json:"bank_name"`

	// Contact Details Fields
	ParentMobile  string `json:"parent_mobile"`
	StudentMobile string `json:"student_mobile"`
	StudentEmail  string `json:"student_email"`
	ParentEmail   string `json:"parent_email"`
	OfficialEmail string `json:"official_email"`

	// Hostel Details Fields
	HostellerType   string `json:"hosteller_type"`
	HostelName      string `json:"hostel_name"`
	RoomNo          string `json:"room_no"`
	RoomCapacity    string `json:"room_capacity"`
	RoomType        string `json:"room_type"`
	FloorNo         string `json:"floor_no"`
	WardenName      string `json:"warden_name"`
	AlternateWarden string `json:"alternate_warden"`
	ClassAdvisor    string `json:"class_advisor"`

	// Insurance Details Fields
	NomineeName  string `json:"nominee_name"`
	Relationship string `json:"relationship"`
	NomineeAge   string `json:"nominee_age"`

	// School Details - Array for multiple schools
	SchoolDetails []SchoolDetailsRequest `json:"school_details"`
}

// SchoolDetailsRequest - School details for creation
type SchoolDetailsRequest struct {
	SchoolName string `json:"school_name"`
	Board      string `json:"board"`
	YearOfPass string `json:"year_of_pass"`
	State      string `json:"state"`
	TCNo       string `json:"tc_no"`
	TCDate     string `json:"tc_date"`
	TotalMarks string `json:"total_marks"`
}
