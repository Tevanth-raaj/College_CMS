package routes

import (
	"net/http"
	"server/handlers/allocation"
	curriculum "server/handlers/curriculum"
	studentteacher "server/handlers/student-teacher_entry"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all application routes
func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Course types
	router.HandleFunc("/api/course-types", curriculum.GetCourseTypes).Methods("GET", "OPTIONS")

	// Academic Calendar routes
	// router.HandleFunc("/api/academic-calendar/current", curriculum.GetCurrentAcademicCalendar).Methods("GET", "OPTIONS")

	// Courses routes
	router.HandleFunc("/api/courses/by-semester/{semester}", curriculum.GetCoursesBySemester).Methods("GET", "OPTIONS")

	// Department routes
	router.HandleFunc("/api/departments", curriculum.GetDepartments).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/departments/{departmentId}/curriculum/semester/{semester}/courses", curriculum.GetDepartmentCurriculumCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/all-departments/semester/{semester}/courses", curriculum.GetAllDepartmentsCourses).Methods("GET", "OPTIONS")

	// Course Type routes
	router.HandleFunc("/api/course-types", curriculum.GetCourseTypes).Methods("GET", "OPTIONS")

	// Curriculum routes
	router.HandleFunc("/api/curriculum", curriculum.GetRegulations).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/create", curriculum.CreateRegulation).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/curriculum/delete", curriculum.DeleteRegulation).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}", curriculum.UpdateCurriculum).Methods("PUT", "OPTIONS")

	// NEW Regulation Management routes (isolated from curriculum)
	router.HandleFunc("/api/regulations", curriculum.GetRegulationsNew).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/regulations", curriculum.CreateRegulationNew).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/regulations/{id}", curriculum.GetRegulationByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/regulations/{id}", curriculum.UpdateRegulationNew).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/regulations/{id}", curriculum.DeleteRegulationNew).Methods("DELETE", "OPTIONS")

	// Regulation Clauses routes
	router.HandleFunc("/api/regulations/{id}/clauses", curriculum.GetRegulationClauses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/regulations/{id}/clauses", curriculum.CreateRegulationClause).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/regulations/clauses/{clauseId}", curriculum.UpdateRegulationClause).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/regulations/clauses/{clauseId}", curriculum.DeleteRegulationClause).Methods("DELETE", "OPTIONS")

	// Regulation Editor routes (structured editing)
	router.HandleFunc("/api/regulations/{id}/structure", curriculum.GetRegulationStructure).Methods("GET", "OPTIONS")

	// Section management
	router.HandleFunc("/api/regulations/{id}/sections", curriculum.CreateSection).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/regulations/sections/{sectionId}", curriculum.UpdateSection).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/regulations/sections/{sectionId}", curriculum.DeleteSection).Methods("DELETE", "OPTIONS")

	// Clause management
	router.HandleFunc("/api/regulations/sections/{sectionId}/clauses", curriculum.CreateClause).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/regulations/clauses/{clauseId}", curriculum.UpdateClause).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/regulations/clauses/{clauseId}", curriculum.DeleteClause).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/regulations/clauses/{clauseId}/history", curriculum.GetClauseHistory).Methods("GET", "OPTIONS")

	// Department Overview routes
	router.HandleFunc("/api/curriculum/{id}/overview", curriculum.GetDepartmentOverview).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/overview", curriculum.SaveDepartmentOverview).Methods("POST", "OPTIONS")

	// Semester routes
	router.HandleFunc("/api/curriculum/{id}/semesters", curriculum.GetSemesters).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/semester", curriculum.CreateSemester).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/semester/{id}", curriculum.UpdateSemester).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/semester/{id}", curriculum.DeleteSemester).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/semester/{semId}/courses", curriculum.GetSemesterCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/semester/{semId}/course", curriculum.AddCourseToSemester).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/course/{id}", curriculum.GetCourse).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/course/{id}", curriculum.UpdateCourse).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/curriculum-course/{id}", curriculum.UpdateCurriculumCourse).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/semester/{semId}/course/{courseId}", curriculum.RemoveCourseFromSemester).Methods("DELETE", "OPTIONS")

	// Honour Card routes
	router.HandleFunc("/api/curriculum/{id}/honour-cards", curriculum.GetHonourCards).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/honour-card", curriculum.CreateHonourCard).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/honour-card/{cardId}", curriculum.DeleteHonourCard).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/honour-card/{cardId}/vertical", curriculum.CreateHonourVertical).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/honour-vertical/{verticalId}", curriculum.DeleteHonourVertical).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/honour-vertical/{verticalId}/course", curriculum.AddCourseToVertical).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/honour-vertical/{verticalId}/course/{courseId}", curriculum.RemoveCourseFromVertical).Methods("DELETE", "OPTIONS")

	// Syllabus routes
	// Return nested syllabus (header + models/titles/topics)
	router.HandleFunc("/api/course/{courseId}/syllabus", curriculum.GetCourseSyllabusNested).Methods("GET", "OPTIONS")
	// Save header-only fields (outcomes, resources, prerequisites)
	router.HandleFunc("/api/course/{courseId}/syllabus", curriculum.SaveCourseSyllabus).Methods("POST", "OPTIONS")

	// Relational CRUD
	router.HandleFunc("/api/course/{courseId}/syllabus/model", curriculum.CreateModel).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/syllabus/model/{modelId}", curriculum.UpdateModel).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/syllabus/model/{modelId}", curriculum.DeleteModel).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/syllabus/model/{modelId}/title", curriculum.CreateTitle).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/syllabus/title/{titleId}", curriculum.UpdateTitle).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/syllabus/title/{titleId}", curriculum.DeleteTitle).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/syllabus/title/{titleId}/topic", curriculum.CreateTopic).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/syllabus/topic/{topicId}", curriculum.UpdateTopic).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/syllabus/topic/{topicId}", curriculum.DeleteTopic).Methods("DELETE", "OPTIONS")

	// CO-PO and CO-PSO Mapping routes
	router.HandleFunc("/api/course/{courseId}/mapping", curriculum.GetCourseMapping).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/course/{courseId}/mapping", curriculum.SaveCourseMapping).Methods("POST", "OPTIONS")

	// PEO-PO Mapping routes
	router.HandleFunc("/api/curriculum/{id}/peo-po-mapping", curriculum.GetPEOPOMapping).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/peo-po-mapping", curriculum.SavePEOPOMapping).Methods("POST", "OPTIONS")

	// Experiments routes (2022 template)
	router.HandleFunc("/api/course/{courseId}/experiments", curriculum.GetCourseExperiments).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/course/{courseId}/experiments", curriculum.CreateExperiment).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/experiments/{expId}", curriculum.UpdateExperiment).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/experiments/{expId}", curriculum.DeleteExperiment).Methods("DELETE", "OPTIONS")

	// Curriculum Logs routes
	router.HandleFunc("/api/curriculum/{id}/log", curriculum.CreateCurriculumLog).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/curriculum/{id}/logs", curriculum.GetCurriculumLogs).Methods("GET", "OPTIONS")

	// PDF Generation route
	router.HandleFunc("/api/curriculum/{id}/pdf", curriculum.GenerateRegulationPDFHTML).Methods("GET", "OPTIONS")

	// Course Allocation routes
	router.HandleFunc("/api/allocations", curriculum.GetCourseAllocations).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/allocations", curriculum.CreateAllocation).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/allocations/{id}", curriculum.UpdateAllocation).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/allocations/{id}", curriculum.DeleteAllocation).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/allocations/unassigned", curriculum.GetUnassignedCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/allocations/summary", curriculum.GetAllocationSummary).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers/{id}/courses", curriculum.GetTeacherCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/courses/{id}", curriculum.GetCourse).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/courses/{id}/teachers", curriculum.GetCourseTeachers).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/department/{departmentId}/semester/{semester}/courses", curriculum.GetDepartmentSemesterCourses).Methods("GET", "OPTIONS")

	// Cluster Management routes
	router.HandleFunc("/api/clusters", curriculum.GetClusters).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/clusters", curriculum.CreateCluster).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/clusters/available-departments", curriculum.GetAvailableDepartments).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/cluster/{id}", curriculum.DeleteCluster).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/cluster/{id}/departments", curriculum.GetClusterDepartments).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/cluster/{id}/department", curriculum.AddDepartmentToCluster).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/cluster/{id}/department/{deptId}", curriculum.RemoveDepartmentFromCluster).Methods("DELETE", "OPTIONS")

	// Sharing Management routes
	router.HandleFunc("/api/curriculum/{id}/sharing", curriculum.GetDepartmentSharingInfo).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/sharing/visibility", curriculum.UpdateItemVisibility).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/sharing/{item_type}/{item_id}/departments", curriculum.GetItemSharedDepartments).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/cluster/{id}/shared-content", curriculum.GetClusterSharedContent).Methods("GET", "OPTIONS")

	// Mark Entry routes
	router.HandleFunc("/api/mark-entry-window", curriculum.GetMarkEntryWindow).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry-window", curriculum.SaveMarkEntryWindow).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/applicable-windows", curriculum.GetApplicableMarkEntryWindows).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows", curriculum.GetAllMarkEntryWindows).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows/stats", curriculum.GetMarkEntryStats).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/stats", curriculum.GetMarkEntryStats).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows/pending-submissions", curriculum.GetWindowsPendingSubmissions).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows/extend-for-teachers", curriculum.ExtendWindowForTeachers).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows/{id}", curriculum.UpdateMarkEntryWindow).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/mark-entry-windows/{id}", curriculum.DeleteMarkEntryWindow).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/course/{courseId}/mark-categories", curriculum.GetMarkCategoriesForCourse).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-categories-by-type/{courseTypeId}", curriculum.GetMarkCategoriesByType).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/course/{courseId}/student-marks", curriculum.GetStudentMarks).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/student-marks/save", curriculum.SaveStudentMarks).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/hod/mark-entry/overview", curriculum.GetHODMarkEntryOverview).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/mark-entry/window-monitor", curriculum.GetHODWindowMonitor).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/admin/mark-entry/window-monitor", curriculum.GetAdminWindowMonitor).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/mark-entry/teacher-students", curriculum.GetTeacherEnteredStudents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/result-analysis", curriculum.GetResultAnalysis).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/mark-entry/download", curriculum.DownloadMarkEntryReport).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/extensions/request", curriculum.CreateMarkEntryExtensionRequest).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/extensions", curriculum.GetMarkEntryExtensionRequests).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/extensions/{id}/approve", curriculum.UpdateMarkEntryExtensionRequestStatus).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/extensions/{id}/reject", curriculum.UpdateMarkEntryExtensionRequestStatus).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/extensions/analytics", curriculum.GetMarkEntryExtensionAnalytics).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-submissions", curriculum.SubmitMarks).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-submissions/check", curriculum.CheckMarkSubmission).Methods("GET", "OPTIONS")

	// Exam Absentees routes (COE role)
	router.HandleFunc("/api/exam-absentees/preview", curriculum.PreviewAbsentees).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/exam-absentees/upload", curriculum.UploadAbsentees).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/exam-absentees", curriculum.GetExamAbsentees).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/exam-absentees/by-window/{windowId}", curriculum.DeleteExamAbsenteesByWindow).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/exam-absentees/{id}", curriculum.DeleteExamAbsentee).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/course/{courseId}/exam-absentees", curriculum.GetCourseWindowAbsentees).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-categories/by-learning-mode", curriculum.GetMarkCategoriesByLearningMode).Methods("GET", "OPTIONS")

	// Student-specific Mark Entry Permission routes
	router.HandleFunc("/api/mark-entry/available-users", curriculum.GetAvailableUsersForAssignment).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/available-students", curriculum.GetStudentsForAssignment).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/create-user-window", curriculum.CreateUserStudentWindow).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/assign-students", curriculum.AssignStudentsToUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/mark-entry/user-assigned-students", curriculum.GetUserAssignedStudents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/remove-student-assignment", curriculum.RemoveStudentAssignment).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/users/{userId}/courses", curriculum.GetUserCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/mark-entry/check-user-windows", curriculum.CheckUserHasAssignedWindows).Methods("GET", "OPTIONS")

	// Authentication routes
	router.HandleFunc("/api/auth/login", curriculum.Login).Methods("POST", "OPTIONS")

	// Elective Management routes
	router.HandleFunc("/api/hod/profile", curriculum.GetHODProfile).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/electives/available", curriculum.GetAvailableElectives).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/electives/save", curriculum.SaveHODSelections).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/hod/batches", curriculum.GetHODBatches).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/elective-slots", curriculum.GetElectiveSemesterSlots).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/honour-minor-eligibility/template", curriculum.DownloadHonourMinorEligibilityTemplate).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/honour-minor-eligibility/import", curriculum.ImportHonourMinorEligibility).Methods("POST", "OPTIONS")

	// Separate honour and minor routes
	router.HandleFunc("/api/hod/honour-eligibility/template", curriculum.DownloadHonourTemplate).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/honour-eligibility/import", curriculum.ImportHonourEligibility).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/hod/minor-eligibility/template", curriculum.DownloadMinorTemplate).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/minor-eligibility/import", curriculum.ImportMinorEligibility).Methods("POST", "OPTIONS")

	router.HandleFunc("/api/hod/teacher-limits/export", curriculum.ExportTeacherLimits).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/academic-calendar/current", curriculum.GetCurrentAcademicCalendar).Methods("GET", "OPTIONS")

	// Minor Program Management routes
	// TODO: Implement these handlers in curriculum package
	router.HandleFunc("/api/hod/minor-verticals", curriculum.GetMinorVerticals).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/vertical-courses", curriculum.GetVerticalCourses).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/minor-selections", curriculum.GetHODMinorSelections).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hod/minor-selections", curriculum.SaveHODMinorSelections).Methods("POST", "OPTIONS")

	// User Management routes
	router.HandleFunc("/api/users", curriculum.GetUsers).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/users", curriculum.CreateUser).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/users/{id}", curriculum.GetUser).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/users/{id}", curriculum.UpdateUser).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/users/{id}", curriculum.DeleteUser).Methods("DELETE", "OPTIONS")
	router.HandleFunc("/api/users/{id}/password", curriculum.ChangePassword).Methods("PUT", "OPTIONS")

	// Student-Teacher Entry routes
	router.HandleFunc("/api/students", studentteacher.GetStudents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/students/{id}", studentteacher.GetStudent).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/students", studentteacher.CreateStudent).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/students/{id}", studentteacher.UpdateStudent).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/students/{id}", studentteacher.DeleteStudent).Methods("DELETE", "OPTIONS")

	// Teacher routes
	router.HandleFunc("/api/teachers", studentteacher.GetTeachers).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers/by-email", studentteacher.GetTeacherByEmail).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers/allocation", studentteacher.GetTeacherAllocation).Methods("GET", "OPTIONS")        // ?faculty_id= - MUST be before /{id}
	router.HandleFunc("/api/teachers/allocation/active", studentteacher.GetActiveAllocations).Methods("GET", "OPTIONS") // ?faculty_id= (only is_active=1)
	router.HandleFunc("/api/teachers/{id}", studentteacher.GetTeacherByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers", studentteacher.CreateTeacher).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/teachers/{id}", studentteacher.UpdateTeacher).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/teachers/{id}", studentteacher.DeleteTeacher).Methods("DELETE", "OPTIONS")

	// Student-Teacher Mapping routes
	router.HandleFunc("/api/student-teacher-mapping/filters", studentteacher.GetMappingFilters).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/student-teacher-mapping/data", studentteacher.GetMappingData).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/student-teacher-mapping/assign", studentteacher.AssignStudentsToTeachers).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/student-teacher-mapping/clear", studentteacher.ClearMappings).Methods("DELETE", "OPTIONS")

	// Student Elective Selection routes
	router.HandleFunc("/api/students/electives/available", studentteacher.GetAvailableElectives).Methods("GET", "OPTIONS")         // ?email=
	router.HandleFunc("/api/students/electives/selections", studentteacher.SaveElectiveSelections).Methods("POST", "OPTIONS")      // ?email=
	router.HandleFunc("/api/students/electives/selections", studentteacher.GetStudentElectiveSelections).Methods("GET", "OPTIONS") // ?email=

	// HR routes
	router.HandleFunc("/api/hr/faculty", studentteacher.GetAllFaculty).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hr/faculty/subject-counts", studentteacher.UpdateFacultySubjectCounts).Methods("PUT", "OPTIONS")

	// Teacher Course Preferences routes
	router.HandleFunc("/api/teachers/{teacher_id}/allocation-summary", studentteacher.GetTeacherAllocationSummary).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers/{teacher_id}/course-preferences", studentteacher.GetTeacherCoursePreferences).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/teachers/course-preferences", studentteacher.SaveTeacherCoursePreferences).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/teachers/course-window/{academic_year}", studentteacher.GetTeacherCourseWindow).Methods("GET", "OPTIONS")

	// Teacher Course Appeal routes
	router.HandleFunc("/api/teachers/appeals", studentteacher.CreateCourseAppeal).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/teachers/appeals/pending", studentteacher.GetTeacherPendingAppeal).Methods("GET", "OPTIONS") // ?faculty_id=
	router.HandleFunc("/api/teachers/appeals/history", studentteacher.GetTeacherAppealHistory).Methods("GET", "OPTIONS") // ?faculty_id=
	router.HandleFunc("/api/hr/appeals", studentteacher.GetAllAppeals).Methods("GET", "OPTIONS")                         // ?status=pending|resolved
	router.HandleFunc("/api/hr/appeals/{appeal_id}", studentteacher.GetAppealByID).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/hr/appeals/{appeal_id}/resolve", studentteacher.UpdateAppealStatus).Methods("PUT", "OPTIONS")
	router.HandleFunc("/api/admin/allocations/deactivate", studentteacher.DeactivateAllocationsForAcademicYear).Methods("PUT", "OPTIONS") // ?academic_year=

	// Teacher -> Department -> Semester courses (auto-map department/curriculum)
	router.HandleFunc("/api/teachers/{teacherId}/semester/{semester}/courses", curriculum.GetCoursesForTeacherSemester).Methods("GET", "OPTIONS")

	// Automatic Allocation routes
	router.HandleFunc("/api/allocations/run", allocation.RunAutoAllocation).Methods("POST", "OPTIONS")

	return router
}
