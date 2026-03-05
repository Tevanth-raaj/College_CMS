import React from "react";
import { Routes, Route } from "react-router-dom";
import LoginPage from "../pages/curriculum/loginPage";
import Dashboard from "../pages/curriculum/dashboard";
import TeacherDashboardPage from "../pages/curriculum/TeacherDashboardPage";
import CurriculumMainPage from "../pages/curriculum/curriculumMainPage";
import DepartmentOverviewPage from "../pages/curriculum/departmentOverviewPage";
import ManageCurriculumPage from "../pages/curriculum/manageCurriculumPage";
import SemesterDetailPage from "../pages/curriculum/semesterDetailPage";
import HonourCardPage from "../pages/curriculum/honourCardPage";
import SyllabusPage from "../pages/curriculum/syllabusPage";
import MappingPage from "../pages/curriculum/mappingPage";
import PEOPOMappingPage from "../pages/curriculum/peoPOMappingPage";
import ClusterManagementPage from "../pages/curriculum/clusterManagementPage";
import SharingManagementPage from "../pages/curriculum/sharingManagementPage";
import RegulationPage from "../pages/regulation/regulationPage";
import RegulationEditorPage from "../pages/regulation/regulationEditorPage";
import UsersPage from "../pages/curriculum/usersPage";
import StudentDetailsPage from "../pages/student-teacher_entry/studentDetailsPage";
import TeacherStudentDashboard from "../pages/student-teacher_entry/TeacherStudentDashboard";
import TeacherDetailsPage from "../pages/student-teacher_entry/TeacherDetailsPage";
import TeacherStudentMappingPage from "../pages/student-teacher_entry/TeacherStudentMappingPage";
import CourseAllocationPage from "../pages/curriculum/CourseAllocationPage";
import ElectiveSelectionPage from "../pages/student/ElectiveSelectionPage";
import TeacherCourseSelectionPage from "../pages/teacher/TeacherCourseSelectionPage";
import HRFacultyPage from "../pages/hr/HRFacultyPage";
import HRAppealsReviewPage from "../pages/hr/HRAppealsReviewPage";
import ElectiveManagementPage from "../pages/curriculum/ElectiveManagementPage";
import HODElectivePage from "../pages/curriculum/HODElectivePage";
import HODHonourMinorEligibilityPage from "../pages/curriculum/HODHonourMinorEligibilityPage";
import TeacherCoursesPage from "../pages/curriculum/TeacherCoursesPage";
import TeacherCourseStudentsPage from "../pages/curriculum/TeacherCourseStudentsPage";
import MarkEntryPage from "../pages/curriculum/MarkEntryPage";
import MarkEntryPermissionsPage from "../pages/curriculum/MarkEntryPermissionsPage";
import MyAssignedStudentsPage from "../pages/curriculum/MyAssignedStudentsPage";
import AcademicCalendarPage from "../pages/curriculum/AcademicCalendarPage";

//404 page
import NotFoundPage from "../components/NotFoundPage";

// Layout components
import PrivateRoute from "../components/PrivateRoute";
import AppShell from "../components/AppShell";

function App() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />

      <Route element={<PrivateRoute />}>
        <Route element={<AppShell />}>
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="teacher-dashboard" element={<TeacherDashboardPage />} />

          <Route path="/Student_details" element={<StudentDetailsPage />} />
          <Route
            path="/student-teacher-dashboard"
            element={<TeacherStudentDashboard />}
          />
          <Route path="/teacher-details" element={<TeacherDetailsPage />} />
          <Route
            path="/teacher-student-mapping"
            element={<TeacherStudentMappingPage />}
          />

          <Route path="Student_details" element={<StudentDetailsPage />} />
          <Route
            path="student-teacher-dashboard"
            element={<TeacherStudentDashboard />}
          />
          <Route path="teacher-details" element={<TeacherDetailsPage />} />
          <Route
            path="teacher-student-mapping"
            element={<TeacherStudentMappingPage />}
          />

          <Route path="teacher-courses" element={<TeacherCoursesPage />} />
          <Route
            path="teacher-course/:courseId/students"
            element={<TeacherCourseStudentsPage />}
          />

          <Route path="mark-entry" element={<MarkEntryPage />} />
          <Route
            path="mark-entry-permissions"
            element={<MarkEntryPermissionsPage />}
          />
          <Route
            path="my-assigned-students"
            element={<MyAssignedStudentsPage />}
          />

          <Route path="course-allocation" element={<CourseAllocationPage />} />
          <Route
            path="elective-management"
            element={<ElectiveManagementPage />}
          />
          <Route path="hod/elective-management" element={<HODElectivePage />} />
          <Route path="academic-calendar" element={<AcademicCalendarPage />} />

          <Route path="regulations" element={<RegulationPage />} />
          <Route
            path="curriculum/:id/editor"
            element={<RegulationEditorPage />}
          />

          <Route path="curriculum" element={<CurriculumMainPage />} />
          <Route
            path="curriculum/:id/overview"
            element={<DepartmentOverviewPage />}
          />
          <Route
            path="curriculum/:id/curriculum"
            element={<ManageCurriculumPage />}
          />
          <Route
            path="curriculum/:id/curriculum/semester/:semId"
            element={<SemesterDetailPage />}
          />
          <Route
            path="curriculum/:id/curriculum/honour/:cardId"
            element={<HonourCardPage />}
          />

          <Route path="course/:courseId/syllabus" element={<SyllabusPage />} />
          <Route path="course/:courseId/mapping" element={<MappingPage />} />
          <Route
            path="curriculum/:id/peo-po-mapping"
            element={<PEOPOMappingPage />}
          />

          <Route path="clusters" element={<ClusterManagementPage />} />
          <Route path="sharing" element={<SharingManagementPage />} />

          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Route>
    </Routes>
  );
}

export default App;
