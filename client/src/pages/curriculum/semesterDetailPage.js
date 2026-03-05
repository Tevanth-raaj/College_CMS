import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

function SemesterDetailPage() {
  const { id, semId } = useParams();
  const navigate = useNavigate();

  const [courses, setCourses] = useState([]);
  const [semester, setSemester] = useState(null);
  const [curriculum, setCurriculum] = useState(null);
  const [curriculumTemplate, setCurriculumTemplate] = useState("2026");
  const [totalCurriculumCredits, setTotalCurriculumCredits] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showAddForm, setShowAddForm] = useState(false);
  const [courseTypes, setCourseTypes] = useState([]);
  const [courseTypeMap, setCourseTypeMap] = useState({}); // Map of ID to name
  const [newCourse, setNewCourse] = useState({
    course_code: "",
    course_name: "",
    course_type: 0,
    category: "",
    credit: "",
    lecture_hrs: 0,
    tutorial_hrs: 0,
    practical_hrs: 0,
    activity_hrs: 0,
    tw_sl_hrs: 0,
    cia_marks: 40,
    see_marks: 60,
    count_towards_limit: true,
  });
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingCourse, setEditingCourse] = useState(null);
  const [editCourseData, setEditCourseData] = useState({
    course_code: "",
    course_name: "",
    course_type: 0,
    category: "",
    credit: "",
    lecture_hrs: 0,
    tutorial_hrs: 0,
    practical_hrs: 0,
    activity_hrs: 0,
    tw_sl_hrs: 0,
    cia_marks: 40,
    see_marks: 60,
    theory_hours: 0,
    tutorial_hours: 0,
    practical_hours: 0,
    count_towards_limit: true,
  });

  useEffect(() => {
    fetchCourseTypes();
    fetchCurriculum();
    fetchSemester();
    fetchCourses();
    fetchTotalCredits();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, semId]);

  const fetchTotalCredits = async () => {
    try {
      // Fetch all semesters for this curriculum
      const semResponse = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semesters`,
      );
      if (!semResponse.ok) return;
      const semesters = await semResponse.json();

      // Fetch courses for all semesters and sum credits
      // Only count credits from semester card types and courses with count_towards_limit = true
      let totalCredits = 0;
      for (const sem of semesters) {
        // Only count credits for semester card types
        if (sem.card_type !== "semester") continue;

        const courseResponse = await fetch(
          `${API_BASE_URL}/curriculum/${id}/semester/${sem.id}/courses`,
        );
        if (courseResponse.ok) {
          const semCourses = await courseResponse.json();
          // Only sum credits where count_towards_limit is true (or undefined for backwards compatibility)
          totalCredits += semCourses.reduce((sum, c) => {
            const shouldCount =
              c.count_towards_limit === undefined ||
              c.count_towards_limit === true;
            return sum + (shouldCount ? c.credit || 0 : 0);
          }, 0);
        }
      }
      setTotalCurriculumCredits(totalCredits);
    } catch (err) {
      console.error("Error fetching total credits:", err);
    }
  };

  const fetchCourseTypes = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/course-types`);
      if (!response.ok) {
        throw new Error("Failed to fetch course types");
      }
      const data = await response.json();
      setCourseTypes(data || []);

      // Create mapping of ID to name
      const map = {};
      (data || []).forEach((ct) => {
        map[ct.id] = ct.name;
      });
      setCourseTypeMap(map);
    } catch (err) {
      console.error("Error fetching course types:", err);
    }
  };

  const fetchCurriculum = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/curriculum`);
      if (!response.ok) {
        throw new Error("Failed to fetch curriculum info");
      }
      const data = await response.json();
      const currentCurr = data.find((c) => c.id === parseInt(id));
      setCurriculum(currentCurr);
      if (currentCurr && currentCurr.curriculum_template) {
        setCurriculumTemplate(currentCurr.curriculum_template);
      }
    } catch (err) {
      console.error("Error fetching curriculum:", err);
    }
  };

  const fetchSemester = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semesters`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch semester info");
      }
      const data = await response.json();
      const currentSem = data.find((s) => s.id === parseInt(semId));
      setSemester(currentSem);
    } catch (err) {
      console.error("Error fetching semester:", err);
    }
  };

  const fetchCourses = async () => {
    try {
      setLoading(true);
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semester/${semId}/courses`,
      );
      if (!response.ok) {
        throw new Error("Failed to fetch courses");
      }
      const data = await response.json();
      // Reverse the array so most recently added courses appear at the top
      setCourses((data || []).reverse());
      setError("");
    } catch (err) {
      console.error("Error fetching courses:", err);
      setError("Failed to load courses");
    } finally {
      setLoading(false);
    }
  };

  const handleAddCourse = async (e) => {
    e.preventDefault();

    // Validate total marks
    const totalMarks =
      (parseInt(newCourse.cia_marks) || 0) +
      (parseInt(newCourse.see_marks) || 0);
    if (totalMarks > 100) {
      setError(
        "Total marks (CIA + SEE) cannot exceed 100. Please adjust the marks.",
      );
      setTimeout(() => setError(""), 5000);
      return;
    }

    try {
      const lectureHrs = parseInt(newCourse.lecture_hrs) || 0;
      const tutorialHrs = parseInt(newCourse.tutorial_hrs) || 0;
      const practicalHrs = parseInt(newCourse.practical_hrs) || 0;
      const activityHrs = parseInt(newCourse.activity_hrs) || 0;

      const courseData = {
        ...newCourse,
        credit: parseInt(newCourse.credit),
        lecture_hrs: lectureHrs,
        tutorial_hrs: tutorialHrs,
        practical_hrs: practicalHrs,
        activity_hrs: activityHrs,
        tw_sl_hrs: parseInt(newCourse.tw_sl_hrs) || 0,
        cia_marks:
          newCourse.cia_marks !== "" &&
          newCourse.cia_marks !== null &&
          newCourse.cia_marks !== undefined
            ? parseInt(newCourse.cia_marks)
            : 40,
        see_marks:
          newCourse.see_marks !== "" &&
          newCourse.see_marks !== null &&
          newCourse.see_marks !== undefined
            ? parseInt(newCourse.see_marks)
            : 60,
      };

      // Calculate total hours based on course type
      // Course Type IDs: 1=Theory, 2=Lab, 3=Theory with Lab
      if (newCourse.course_type === 2) {
        // Lab
        courseData.theory_total_hrs = 0;
        courseData.tutorial_total_hrs = 0;
        courseData.activity_total_hrs = 0;
        courseData.practical_total_hrs = practicalHrs * 15;
      } else if (newCourse.course_type === 1) {
        // Theory
        courseData.theory_total_hrs = lectureHrs * 15;
        courseData.tutorial_total_hrs = tutorialHrs * 15;
        courseData.activity_total_hrs = activityHrs * 15;
        courseData.practical_total_hrs = 0;
      } else if (newCourse.course_type === 3) {
        // Theory with Lab
        courseData.theory_total_hrs = lectureHrs * 15;
        courseData.tutorial_total_hrs = tutorialHrs * 15;
        courseData.practical_total_hrs = practicalHrs * 15;
        courseData.activity_total_hrs = 0;
      } else {
        // Default: calculate all
        courseData.theory_total_hrs = lectureHrs * 15;
        courseData.tutorial_total_hrs = tutorialHrs * 15;
        courseData.practical_total_hrs = practicalHrs * 15;
        courseData.activity_total_hrs = activityHrs * 15;
      }

      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semester/${semId}/course`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(courseData),
        },
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to add course");
      }

      const responseData = await response.json();

      // Check if course was reused and show info message
      if (responseData.message) {
        setSuccess(responseData.message);
        setTimeout(() => setSuccess(""), 5000);
      } else {
        setSuccess("Course added successfully!");
        setTimeout(() => setSuccess(""), 3000);
      }

      setNewCourse({
        course_code: "",
        course_name: "",
        course_type: "",
        category: "",
        credit: "",
        lecture_hrs: 0,
        tutorial_hrs: 0,
        practical_hrs: 0,
        activity_hrs: 0,
        tw_sl_hrs: 0,
        cia_marks: 40,
        see_marks: 60,
        count_towards_limit: true,
      });
      setShowAddForm(false);
      fetchCourses();
      fetchTotalCredits();
    } catch (err) {
      console.error("Error adding course:", err);
      setError(err.message || "Failed to add course");
      setTimeout(() => setError(""), 5000);
    }
  };

  const handleEditCourse = (course) => {
    setEditingCourse(course);

    setEditCourseData({
      course_code: course.course_code,
      course_name: course.course_name,
      course_type: course.course_type,
      category: course.category,
      credit: course.credit,
      lecture_hrs: course.lecture_hrs || 0,
      tutorial_hrs: course.tutorial_hrs || 0,
      practical_hrs: course.practical_hrs || 0,
      activity_hrs: course.activity_hrs || 0,
      tw_sl_hrs: course.tw_sl_hrs || 0,
      cia_marks:
        course.cia_marks !== null && course.cia_marks !== undefined
          ? course.cia_marks
          : 40,
      see_marks:
        course.see_marks !== null && course.see_marks !== undefined
          ? course.see_marks
          : 60,
      theory_hours: course.theory_total_hrs || 0,
      tutorial_hours: course.tutorial_total_hrs || 0,
      practical_hours: course.practical_total_hrs || 0,
      count_towards_limit:
        course.count_towards_limit === undefined
          ? true
          : course.count_towards_limit,
    });
    setShowEditModal(true);
  };

  const handleUpdateCourse = async (e) => {
    e.preventDefault();

    // Validate total marks
    const totalMarks =
      (parseInt(editCourseData.cia_marks) || 0) +
      (parseInt(editCourseData.see_marks) || 0);
    if (totalMarks > 100) {
      setError(
        "Total marks (CIA + SEE) cannot exceed 100. Please adjust the marks.",
      );
      setTimeout(() => setError(""), 5000);
      return;
    }

    try {
      const lectureHrs = parseInt(editCourseData.lecture_hrs) || 0;
      const tutorialHrs = parseInt(editCourseData.tutorial_hrs) || 0;
      const practicalHrs = parseInt(editCourseData.practical_hrs) || 0;
      const activityHrs = parseInt(editCourseData.activity_hrs) || 0;

      const courseData = {
        ...editCourseData,
        credit: parseInt(editCourseData.credit),
        lecture_hrs: lectureHrs,
        tutorial_hrs: tutorialHrs,
        practical_hrs: practicalHrs,
        activity_hrs: activityHrs,
        tw_sl_hrs: parseInt(editCourseData.tw_sl_hrs) || 0,
        cia_marks:
          editCourseData.cia_marks !== "" &&
          editCourseData.cia_marks !== null &&
          editCourseData.cia_marks !== undefined
            ? parseInt(editCourseData.cia_marks)
            : 40,
        see_marks:
          editCourseData.see_marks !== "" &&
          editCourseData.see_marks !== null &&
          editCourseData.see_marks !== undefined
            ? parseInt(editCourseData.see_marks)
            : 60,
      };

      // Calculate total hours based on course type (for 2026 template, auto-calculate; for 2022, use manual inputs)
      if (curriculumTemplate === "2026") {
        // Course Type IDs: 1=Theory, 2=Lab, 3=Theory with Lab
        if (editCourseData.course_type === 2) {
          // Lab
          courseData.theory_total_hrs = 0;
          courseData.tutorial_total_hrs = 0;
          courseData.activity_total_hrs = 0;
          courseData.practical_total_hrs = practicalHrs * 15;
        } else if (editCourseData.course_type === 1) {
          // Theory
          courseData.theory_total_hrs = lectureHrs * 15;
          courseData.tutorial_total_hrs = tutorialHrs * 15;
          courseData.activity_total_hrs = activityHrs * 15;
          courseData.practical_total_hrs = 0;
        } else if (editCourseData.course_type === 3) {
          // Theory with Lab
          courseData.theory_total_hrs = lectureHrs * 15;
          courseData.tutorial_total_hrs = tutorialHrs * 15;
          courseData.practical_total_hrs = practicalHrs * 15;
          courseData.activity_total_hrs = 0;
        } else {
          // Default: calculate all
          courseData.theory_total_hrs = lectureHrs * 15;
          courseData.tutorial_total_hrs = tutorialHrs * 15;
          courseData.practical_total_hrs = practicalHrs * 15;
          courseData.activity_total_hrs = activityHrs * 15;
        }
      } else {
        // For 2022 template, use manual input values
        courseData.theory_total_hrs =
          parseInt(editCourseData.theory_hours) || 0;
        courseData.tutorial_total_hrs =
          parseInt(editCourseData.tutorial_hours) || 0;
        courseData.practical_total_hrs =
          parseInt(editCourseData.practical_hours) || 0;
      }

      const response = await fetch(
        `${API_BASE_URL}/course/${editingCourse.id}`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(courseData),
        },
      );

      if (!response.ok) {
        throw new Error("Failed to update course");
      }

      // Update count_towards_limit in curriculum_courses if reg_course_id exists
      if (editingCourse.reg_course_id && semester?.card_type === "semester") {
        const linkUpdateResponse = await fetch(
          `${API_BASE_URL}/curriculum-course/${editingCourse.reg_course_id}`,
          {
            method: "PUT",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              count_towards_limit: editCourseData.count_towards_limit,
            }),
          },
        );

        if (!linkUpdateResponse.ok) {
          console.warn("Failed to update count_towards_limit flag");
        }
      }

      setSuccess("Course updated successfully!");
      setTimeout(() => setSuccess(""), 3000);
      setShowEditModal(false);
      setEditingCourse(null);
      fetchCourses();
      fetchTotalCredits();
    } catch (err) {
      console.error("Error updating course:", err);
      setError("Failed to update course");
    }
  };

  const handleRemoveCourse = async (courseId) => {
    if (
      !window.confirm(
        "Are you sure you want to remove this course from the semester?",
      )
    ) {
      return;
    }

    try {
      const response = await fetch(
        `${API_BASE_URL}/curriculum/${id}/semester/${semId}/course/${courseId}`,
        {
          method: "DELETE",
        },
      );

      if (!response.ok) {
        throw new Error("Failed to remove course");
      }

      setSuccess("Course removed successfully!");
      setTimeout(() => setSuccess(""), 3000);
      fetchCourses();
      fetchTotalCredits();
    } catch (err) {
      console.error("Error removing course:", err);
      setError("Failed to remove course");
    }
  };

  if (loading) {
    return (
      <MainLayout title="Semester Courses" subtitle="Loading...">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <svg
              className="animate-spin h-12 w-12 text-blue-600 mx-auto mb-4"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              ></circle>
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <p className="text-gray-600">Loading courses...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  // Helper function to get proper card type display name
  const getCardTypeName = () => {
    const cardType = semester?.card_type || "semester";
    if (cardType === "semester")
      return semester?.semester_number
        ? `Semester ${semester.semester_number}`
        : "Semester";
    if (cardType === "vertical")
      return semester?.semester_number
        ? `Vertical ${semester.semester_number}`
        : "Vertical";
    if (cardType === "language_elective") return "Language Elective";
    if (cardType === "open_elective") {
      const vn = semester?.vertical_name || "";
      if (vn === "Same Dept") return "Open Elective (Same Dept)";
      if (vn === "Different Dept") return "Open Elective (Diff Dept)";
      return "Open Elective";
    }
    if (cardType === "one_credit") return "One Credit";
    return `Card ${semId}`;
  };

  return (
    <MainLayout
      title={`${getCardTypeName()} - Courses`}
      subtitle={
        <div className="flex items-center space-x-4">
          <span>Curriculum ID: {id}</span>
          {curriculum && (
            <span
              className={`font-semibold ${totalCurriculumCredits > curriculum.max_credits ? "text-red-600" : totalCurriculumCredits === curriculum.max_credits ? "text-green-600" : "text-blue-600"}`}
            >
              Total Credits: {totalCurriculumCredits} /{" "}
              {curriculum.max_credits || 0}
            </span>
          )}
        </div>
      }
      actions={
        <div className="flex items-center space-x-3">
          <button
            onClick={() => navigate(`/curriculum/${id}/curriculum`)}
            className="btn-secondary-custom flex items-center space-x-2"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 19l-7-7m0 0l7-7m-7 7h18"
              />
            </svg>
            <span>Back</span>
          </button>
          <button
            onClick={() => setShowAddForm(!showAddForm)}
            className="btn-primary-custom flex items-center space-x-2"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
            <span>{showAddForm ? "Cancel" : "Add Course"}</span>
          </button>
        </div>
      }
    >
      <div className="min-w-screen mx-auto space-y-6">
        {/* Messages */}
        {error && (
          <div className="flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
            <svg
              className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
            <p className="text-sm font-medium text-red-600">{error}</p>
          </div>
        )}

        {success && (
          <div className="flex items-start space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
            <svg
              className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                clipRule="evenodd"
              />
            </svg>
            <p className="text-sm font-medium text-green-600">{success}</p>
          </div>
        )}

        {/* Add Course Form */}
        {showAddForm && (
          <div className="card-custom p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-bold text-gray-900">
                Add New Course
              </h2>
              {curriculum && semester?.card_type === "semester" && (
                <div
                  className={`px-4 py-2 rounded-lg font-semibold text-sm ${
                    totalCurriculumCredits >= curriculum.max_credits
                      ? "bg-red-100 text-red-700"
                      : newCourse.count_towards_limit &&
                          totalCurriculumCredits +
                            (parseInt(newCourse.credit) || 0) >
                            curriculum.max_credits
                        ? "bg-yellow-100 text-yellow-700"
                        : "bg-blue-100 text-blue-700"
                  }`}
                >
                  {totalCurriculumCredits >= curriculum.max_credits
                    ? "Maximum credits reached!"
                    : `Available: ${curriculum.max_credits - totalCurriculumCredits} credits`}
                </div>
              )}
            </div>
            <form
              onSubmit={handleAddCourse}
              className="grid grid-cols-1 md:grid-cols-2 gap-4"
            >
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Course Code *
                </label>
                <input
                  type="text"
                  value={newCourse.course_code}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, course_code: e.target.value })
                  }
                  placeholder="e.g., CS101"
                  required
                  className="input-custom"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Course Name *
                </label>
                <input
                  type="text"
                  value={newCourse.course_name}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, course_name: e.target.value })
                  }
                  placeholder="e.g., Introduction to Programming"
                  required
                  className="input-custom"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Course Type *
                </label>
                <select
                  value={newCourse.course_type}
                  onChange={(e) =>
                    setNewCourse({
                      ...newCourse,
                      course_type: parseInt(e.target.value) || 0,
                    })
                  }
                  required
                  className="input-custom"
                >
                  <option value="0">Select Type</option>
                  <option value="1">Theory</option>
                  <option value="2">Lab</option>
                  <option value="3">Theory with Lab</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Category *
                </label>
                <select
                  value={newCourse.category}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, category: e.target.value })
                  }
                  required
                  className="input-custom"
                >
                  <option value="">Select Category</option>
                  <option value="BS - Basic Sciences">
                    BS - Basic Sciences
                  </option>
                  <option value="ES - Engineering Sciences">
                    ES - Engineering Sciences
                  </option>
                  <option value="HSS - Humanities and Social Sciences">
                    HSS - Humanities and Social Sciences
                  </option>
                  <option value="PC - Professional Core">
                    PC - Professional Core
                  </option>
                  <option value="PE - Professional Elective">
                    PE - Professional Elective
                  </option>
                  <option value="OE - Open Elective">OE - Open Elective</option>
                  <option value="EEC - Employability Enhancement Course">
                    EEC - Employability Enhancement Course
                  </option>
                  <option value="NA">NA</option>
                </select>
              </div>

              {/* Common Fields - Hours per week */}
              {!(
                curriculumTemplate === "2022" && newCourse.course_type === 2
              ) && (
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Lecture (hrs per week) *
                  </label>
                  <input
                    type="number"
                    value={newCourse.lecture_hrs}
                    onChange={(e) =>
                      setNewCourse({
                        ...newCourse,
                        lecture_hrs: e.target.value,
                      })
                    }
                    placeholder="0"
                    required
                    min="0"
                    className="input-custom"
                  />
                </div>
              )}

              {!(
                curriculumTemplate === "2022" && newCourse.course_type === 2
              ) && (
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Tutorial (hrs per week)
                  </label>
                  <input
                    type="number"
                    value={newCourse.tutorial_hrs}
                    onChange={(e) =>
                      setNewCourse({
                        ...newCourse,
                        tutorial_hrs: e.target.value,
                      })
                    }
                    placeholder="0"
                    min="0"
                    className="input-custom"
                  />
                </div>
              )}

              {newCourse.course_type !== 1 && (
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Practical (hrs per week)
                  </label>
                  <input
                    type="number"
                    value={newCourse.practical_hrs}
                    onChange={(e) =>
                      setNewCourse({
                        ...newCourse,
                        practical_hrs: e.target.value,
                      })
                    }
                    placeholder="0"
                    min="0"
                    className="input-custom"
                  />
                </div>
              )}

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Credits *
                </label>
                <input
                  type="number"
                  value={newCourse.credit}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, credit: e.target.value })
                  }
                  placeholder="e.g., 3"
                  required
                  min="0"
                  className="input-custom"
                />
              </div>

              {curriculumTemplate !== "2022" && (
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Activity (hrs per week)
                  </label>
                  <input
                    type="number"
                    value={newCourse.activity_hrs}
                    onChange={(e) =>
                      setNewCourse({
                        ...newCourse,
                        activity_hrs: e.target.value,
                      })
                    }
                    placeholder="0"
                    min="0"
                    className="input-custom"
                  />
                </div>
              )}

              {/* Common Marks Fields */}
              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  CIA *
                </label>
                <input
                  type="number"
                  value={newCourse.cia_marks}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, cia_marks: e.target.value })
                  }
                  placeholder="40"
                  required
                  min="0"
                  max="100"
                  className="input-custom"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  SEE *
                </label>
                <input
                  type="number"
                  value={newCourse.see_marks}
                  onChange={(e) =>
                    setNewCourse({ ...newCourse, see_marks: e.target.value })
                  }
                  placeholder="60"
                  required
                  min="0"
                  max="100"
                  className="input-custom"
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  TOTAL SCORE (Auto)
                </label>
                <input
                  type="number"
                  value={
                    (parseInt(newCourse.cia_marks) || 0) +
                    (parseInt(newCourse.see_marks) || 0)
                  }
                  readOnly
                  className={`input-custom bg-gray-100 cursor-not-allowed ${(parseInt(newCourse.cia_marks) || 0) + (parseInt(newCourse.see_marks) || 0) > 100 ? "border-red-500 border-2" : ""}`}
                />
                {(parseInt(newCourse.cia_marks) || 0) +
                  (parseInt(newCourse.see_marks) || 0) >
                  100 && (
                  <p className="text-red-600 text-xs mt-1 font-medium">
                    ⚠ Total marks cannot exceed 100
                  </p>
                )}
              </div>

              {/* Course Type Specific Fields - Total Hours for whole semester */}
              {newCourse.course_type === 1 && (
                <>
                  <div className="md:col-span-2">
                    <h3 className="text-sm font-bold text-gray-900 mb-3 mt-4 pt-4 border-t border-gray-200">
                      Total Hours (for whole semester)
                    </h3>
                  </div>

                  {curriculumTemplate === "2026" ? (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          THEORY HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.lecture_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TUTORIAL HOURS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.tutorial_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          ACTIVITY HOURS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.activity_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TOTAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={
                            (parseInt(newCourse.lecture_hrs) || 0) * 15 +
                            (parseInt(newCourse.tutorial_hrs) || 0) * 15 +
                            (parseInt(newCourse.activity_hrs) || 0) * 15
                          }
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          THEORY HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.lecture_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TUTORIAL HOURS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.tutorial_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TOTAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={
                            (parseInt(newCourse.lecture_hrs) || 0) * 15 +
                            (parseInt(newCourse.tutorial_hrs) || 0) * 15
                          }
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  )}
                </>
              )}

              {newCourse.course_type === 3 && (
                <>
                  <div className="md:col-span-2">
                    <h3 className="text-sm font-bold text-gray-900 mb-3 mt-4 pt-4 border-t border-gray-200">
                      Total Hours (for whole semester)
                    </h3>
                  </div>

                  {curriculumTemplate === "2026" ? (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          THEORY HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.lecture_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TUTORIAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.tutorial_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          PRACTICAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.practical_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TOTAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={
                            (parseInt(newCourse.lecture_hrs) || 0) * 15 +
                            (parseInt(newCourse.tutorial_hrs) || 0) * 15 +
                            (parseInt(newCourse.practical_hrs) || 0) * 15
                          }
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          THEORY HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.lecture_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TUTORIAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.tutorial_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          PRACTICAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.practical_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TOTAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={
                            (parseInt(newCourse.lecture_hrs) || 0) * 15 +
                            (parseInt(newCourse.tutorial_hrs) || 0) * 15 +
                            (parseInt(newCourse.practical_hrs) || 0) * 15
                          }
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  )}
                </>
              )}

              {newCourse.course_type === 2 && (
                <>
                  <div className="md:col-span-2">
                    <h3 className="text-sm font-bold text-gray-900 mb-3 mt-4 pt-4 border-t border-gray-200">
                      Total Hours (for whole semester)
                    </h3>
                  </div>

                  {curriculumTemplate === "2026" ? (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          PRACTICAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.practical_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TW/SL HRS
                        </label>
                        <input
                          type="number"
                          value={newCourse.tw_sl_hrs}
                          onChange={(e) =>
                            setNewCourse({
                              ...newCourse,
                              tw_sl_hrs: e.target.value,
                            })
                          }
                          placeholder="0"
                          min="0"
                          className="input-custom"
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          TOTAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={
                            (parseInt(newCourse.practical_hrs) || 0) * 15 +
                            (parseInt(newCourse.tw_sl_hrs) || 0)
                          }
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  ) : (
                    <>
                      <div>
                        <label className="block text-sm font-semibold text-gray-700 mb-2">
                          PRACTICAL HRS (Auto)
                        </label>
                        <input
                          type="number"
                          value={(parseInt(newCourse.practical_hrs) || 0) * 15}
                          readOnly
                          className="input-custom bg-gray-100 cursor-not-allowed"
                        />
                      </div>
                    </>
                  )}
                </>
              )}

              {semester?.card_type === "semester" && (
                <>
                  {/* Credit Limit Checkbox - Only show for semester card types */}
                  <div className="md:col-span-2 mt-4 pt-4 border-t border-gray-200">
                    <label className="flex items-center space-x-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={newCourse.count_towards_limit}
                        onChange={(e) =>
                          setNewCourse({
                            ...newCourse,
                            count_towards_limit: e.target.checked,
                          })
                        }
                        className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                      />
                      <span className="text-sm font-medium text-gray-700">
                        Include this course's credits in the curriculum's max
                        credit limit calculation
                      </span>
                    </label>
                    <p className="text-xs text-gray-500 mt-2 ml-7">
                      Uncheck this if the course credits should not count
                      towards the total {curriculum?.max_credits || 0} credit
                      limit
                    </p>
                  </div>
                </>
              )}

              <div className="md:col-span-2">
                <button type="submit" className="w-full btn-primary-custom">
                  Add Course
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Courses Table */}
        {courses.length === 0 ? (
          <div className="card-custom p-12 text-center">
            <svg
              className="w-20 h-20 text-gray-300 mx-auto mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
              />
            </svg>
            <h3 className="text-xl font-semibold text-gray-900 mb-2">
              No Courses Yet
            </h3>
            <p className="text-gray-600 mb-6">
              Get started by adding your first course
            </p>
            <button
              onClick={() => setShowAddForm(true)}
              className="btn-primary-custom"
            >
              Add Course
            </button>
          </div>
        ) : (
          <div className="card-custom overflow-hidden">
            <div className="overflow-x-auto">
              <table className="table-custom">
                <thead>
                  <tr>
                    <th className="text-left w-32">Course Code</th>
                    <th className="text-left min-w-[200px]">Course Name</th>
                    <th className="text-center w-32">Type</th>
                    <th className="text-center w-28">Category</th>
                    <th className="text-center w-28">
                      {curriculumTemplate === "2022" ? "L-T-P" : "L-T-P-A"}
                    </th>
                    <th className="text-center w-20">Credits</th>
                    <th className="text-center w-32">Marks</th>
                    <th className="text-center w-48">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {courses.map((course) => (
                    <tr key={course.id}>
                      <td className="font-semibold tracking-tight text-gray-900">
                        {course.course_code}
                      </td>
                      <td className="tracking-tight text-gray-800">
                        {course.course_name}
                      </td>
                      <td className="text-center">
                        <span
                          className={`inline-block px-2.5 py-1 tracking-tight uppercase rounded-md text-xs font-semibold whitespace-nowrap ${
                            course.course_type === "theory"
                              ? "bg-blue-50 text-blue-700 border border-blue-200"
                              : course.course_type === "lab"
                                ? "bg-green-50 text-green-700 border border-green-200"
                                : course.course_type === "theory_with_lab"
                                  ? "bg-purple-50 text-purple-700 border border-purple-200"
                                  : "bg-gray-50 text-gray-700 border border-gray-200"
                          }`}
                        >
                          {course.course_type === "theory_with_lab"
                            ? "Theory+Lab"
                            : course.course_type}
                        </span>
                      </td>
                      <td className="text-center">
                        <span className="py-1 font-medium tracking-tight text-gray-700">
                          {course.category}
                        </span>
                      </td>
                      <td className="font-mono text-sm text-center text-gray-700">
                        {curriculumTemplate === "2022"
                          ? `${course.lecture_hrs || 0}-${course.tutorial_hrs || 0}-${course.practical_hrs || 0}`
                          : `${course.lecture_hrs || 0}-${course.tutorial_hrs || 0}-${course.practical_hrs || 0}-${course.activity_hrs || 0}`}
                      </td>
                      <td className="font-bold text-center text-gray-900">
                        {course.credit}
                      </td>
                      <td>
                        <div className="flex flex-col items-center gap-0.5 text-xs">
                          <div className="flex items-center gap-1">
                            <span className="text-gray-500 font-medium">
                              CIA:
                            </span>
                            <span className="font-semibold text-gray-900">
                              {course.cia_marks || 0}
                            </span>
                          </div>
                          <div className="flex items-center gap-1">
                            <span className="text-gray-500 font-medium">
                              SEE:
                            </span>
                            <span className="font-semibold text-gray-900">
                              {course.see_marks || 0}
                            </span>
                          </div>
                          <div className="flex items-center gap-1 pt-0.5 border-t border-gray-200 mt-0.5">
                            <span className="text-blue-600 font-semibold">
                              Total:
                            </span>
                            <span className="text-blue-600 font-bold">
                              {course.total_marks || 0}
                            </span>
                          </div>
                        </div>
                      </td>
                      <td className="text-center">
                        <div className="flex gap-1.5 justify-center items-center">
                          <button
                            onClick={() =>
                              navigate(`/course/${course.id}/syllabus`)
                            }
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs font-medium rounded-md transition-all tracking-tight shadow-sm hover:shadow"
                            title="View Syllabus"
                          >
                            Syllabus
                          </button>
                          <button
                            onClick={() =>
                              navigate(`/course/${course.id}/mapping`)
                            }
                            className="px-3 py-1.5 bg-purple-600 hover:bg-purple-700 text-white text-xs font-medium rounded-md transition-all tracking-tight shadow-sm hover:shadow"
                            title="View Mapping"
                          >
                            Mapping
                          </button>
                          <button
                            onClick={() => handleEditCourse(course)}
                            className="p-1.5 text-green-600 hover:text-white hover:bg-green-600 rounded-md transition-all shadow-sm hover:shadow"
                            title="Edit Course"
                          >
                            <svg
                              className="w-4 h-4"
                              fill="none"
                              stroke="currentColor"
                              viewBox="0 0 24 24"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                              />
                            </svg>
                          </button>
                          <button
                            onClick={() => handleRemoveCourse(course.id)}
                            className="p-1.5 rounded-md text-red-600 hover:bg-red-600 hover:text-white transition-all shadow-sm hover:shadow"
                            title="Delete Course"
                          >
                            <svg
                              className="w-4 h-4"
                              fill="none"
                              stroke="currentColor"
                              viewBox="0 0 24 24"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                              />
                            </svg>
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Edit Course Modal */}
        {showEditModal && editingCourse && (
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4"
            onClick={() => setShowEditModal(false)}
          >
            <div
              className="bg-white rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="bg-gradient-to-r from-green-600 to-green-700 text-white px-8 py-5 flex items-center justify-between sticky top-0 rounded-t-2xl">
                <div>
                  <h3 className="text-xl font-bold">Edit Course</h3>
                  <p className="text-sm text-green-100">
                    Update course details
                  </p>
                </div>
                <button
                  onClick={() => setShowEditModal(false)}
                  className="text-white hover:bg-white/20 rounded-lg p-2 transition-all"
                >
                  <svg
                    className="w-6 h-6"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>

              <form onSubmit={handleUpdateCourse} className="p-8 space-y-5">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Course Code
                    </label>
                    <input
                      type="text"
                      value={editCourseData.course_code}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          course_code: e.target.value,
                        })
                      }
                      placeholder="e.g., CS101"
                      required
                      className="input-custom"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Course Title
                    </label>
                    <input
                      type="text"
                      value={editCourseData.course_name}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          course_name: e.target.value,
                        })
                      }
                      placeholder="e.g., Programming Fundamentals"
                      required
                      className="input-custom"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Course Type
                    </label>
                    <select
                      value={editCourseData.course_type}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          course_type: parseInt(e.target.value) || 0,
                        })
                      }
                      required
                      className="input-custom"
                    >
                      <option value="0">Select Type</option>
                      <option value="1">Theory</option>
                      <option value="2">Lab</option>
                      <option value="3">Theory with Lab</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Category
                    </label>
                    <select
                      value={editCourseData.category}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          category: e.target.value,
                        })
                      }
                      required
                      className="input-custom"
                    >
                      <option value="">Select Category</option>
                      <option value="BS - Basic Sciences">
                        BS - Basic Sciences
                      </option>
                      <option value="ES - Engineering Sciences">
                        ES - Engineering Sciences
                      </option>
                      <option value="HSS - Humanities and Social Sciences">
                        HSS - Humanities and Social Sciences
                      </option>
                      <option value="PC - Professional Core">
                        PC - Professional Core
                      </option>
                      <option value="PE - Professional Elective">
                        PE - Professional Elective
                      </option>
                      <option value="OE - Open Elective">
                        OE - Open Elective
                      </option>
                      <option value="EEC - Employability Enhancement Course">
                        EEC - Employability Enhancement Course
                      </option>
                      <option value="NA">NA</option>
                    </select>
                  </div>
                </div>

                {/* Hours per week section */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Credits
                    </label>
                    <input
                      type="number"
                      value={editCourseData.credit}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          credit: e.target.value,
                        })
                      }
                      placeholder="4"
                      required
                      min="0"
                      className="input-custom"
                    />
                  </div>

                  {!(
                    curriculumTemplate === "2022" &&
                    editCourseData.course_type === 2
                  ) && (
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">
                        Lecture (hrs per week)
                      </label>
                      <input
                        type="number"
                        value={editCourseData.lecture_hrs}
                        onChange={(e) =>
                          setEditCourseData({
                            ...editCourseData,
                            lecture_hrs: e.target.value,
                          })
                        }
                        placeholder="3"
                        min="0"
                        className="input-custom"
                      />
                    </div>
                  )}

                  {!(
                    curriculumTemplate === "2022" &&
                    editCourseData.course_type === 2
                  ) && (
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">
                        Tutorial (hrs per week)
                      </label>
                      <input
                        type="number"
                        value={editCourseData.tutorial_hrs}
                        onChange={(e) =>
                          setEditCourseData({
                            ...editCourseData,
                            tutorial_hrs: e.target.value,
                          })
                        }
                        placeholder="0"
                        min="0"
                        className="input-custom"
                      />
                    </div>
                  )}

                  {editCourseData.course_type !== 1 && (
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">
                        Practical (hrs per week)
                      </label>
                      <input
                        type="number"
                        value={editCourseData.practical_hrs}
                        onChange={(e) =>
                          setEditCourseData({
                            ...editCourseData,
                            practical_hrs: e.target.value,
                          })
                        }
                        placeholder="2"
                        min="0"
                        className="input-custom"
                      />
                    </div>
                  )}

                  {curriculumTemplate !== "2022" && (
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-2">
                        Activity (hrs per week)
                      </label>
                      <input
                        type="number"
                        value={editCourseData.activity_hrs}
                        onChange={(e) =>
                          setEditCourseData({
                            ...editCourseData,
                            activity_hrs: e.target.value,
                          })
                        }
                        placeholder="0"
                        min="0"
                        className="input-custom"
                      />
                    </div>
                  )}
                </div>

                {/* Marks section */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      CIA Marks
                    </label>
                    <input
                      type="number"
                      value={editCourseData.cia_marks}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          cia_marks: e.target.value,
                        })
                      }
                      placeholder="40"
                      min="0"
                      max="100"
                      className="input-custom"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      SEE Marks
                    </label>
                    <input
                      type="number"
                      value={editCourseData.see_marks}
                      onChange={(e) =>
                        setEditCourseData({
                          ...editCourseData,
                          see_marks: e.target.value,
                        })
                      }
                      placeholder="60"
                      min="0"
                      max="100"
                      className="input-custom"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Total Score (Auto)
                    </label>
                    <input
                      type="number"
                      value={
                        (parseInt(editCourseData.cia_marks) || 0) +
                        (parseInt(editCourseData.see_marks) || 0)
                      }
                      readOnly
                      className={`input-custom bg-gray-100 cursor-not-allowed ${(parseInt(editCourseData.cia_marks) || 0) + (parseInt(editCourseData.see_marks) || 0) > 100 ? "border-red-500 border-2" : ""}`}
                    />
                    {(parseInt(editCourseData.cia_marks) || 0) +
                      (parseInt(editCourseData.see_marks) || 0) >
                      100 && (
                      <p className="text-red-600 text-xs mt-1 font-medium">
                        ⚠ Total marks cannot exceed 100
                      </p>
                    )}
                  </div>
                </div>

                {/* Total Hours for whole semester - Theory */}
                {editCourseData.course_type === 1 && (
                  <>
                    <div className="border-t pt-4 mt-4">
                      <h3 className="text-sm font-bold text-gray-900 mb-3">
                        Total Hours (for whole semester)
                      </h3>
                    </div>

                    {curriculumTemplate === "2026" ? (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            THEORY HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.lecture_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TUTORIAL HOURS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.tutorial_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            ACTIVITY HOURS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.activity_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TOTAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.lecture_hrs) || 0) * 15 +
                              (parseInt(editCourseData.tutorial_hrs) || 0) *
                                15 +
                              (parseInt(editCourseData.activity_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    ) : (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            THEORY HRS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.theory_hours}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                theory_hours: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TUTORIAL HOURS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.tutorial_hours}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                tutorial_hours: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TOTAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.theory_hours) || 0) +
                              (parseInt(editCourseData.tutorial_hours) || 0)
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    )}
                  </>
                )}

                {/* Total Hours for whole semester - Theory&Lab */}
                {editCourseData.course_type === 3 && (
                  <>
                    <div className="border-t pt-4 mt-4">
                      <h3 className="text-sm font-bold text-gray-900 mb-3">
                        Total Hours (for whole semester)
                      </h3>
                    </div>

                    {curriculumTemplate === "2026" ? (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            THEORY HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.lecture_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TUTORIAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.tutorial_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            PRACTICAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.practical_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TOTAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.lecture_hrs) || 0) * 15 +
                              (parseInt(editCourseData.tutorial_hrs) || 0) *
                                15 +
                              (parseInt(editCourseData.practical_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    ) : (
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            THEORY HRS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.theory_hours}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                theory_hours: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TUTORIAL HRS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.tutorial_hours}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                tutorial_hours: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            PRACTICAL HRS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.practical_hours}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                practical_hours: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TOTAL (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.theory_hours) || 0) +
                              (parseInt(editCourseData.tutorial_hours) || 0) +
                              (parseInt(editCourseData.practical_hours) || 0)
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    )}
                  </>
                )}

                {/* Total Hours for whole semester - Lab */}
                {editCourseData.course_type === 2 && (
                  <>
                    <div className="border-t pt-4 mt-4">
                      <h3 className="text-sm font-bold text-gray-900 mb-3">
                        Total Hours (for whole semester)
                      </h3>
                    </div>

                    {curriculumTemplate === "2026" ? (
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            PRACTICAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.practical_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TW/SL HRS
                          </label>
                          <input
                            type="number"
                            value={editCourseData.tw_sl_hrs}
                            onChange={(e) =>
                              setEditCourseData({
                                ...editCourseData,
                                tw_sl_hrs: e.target.value,
                              })
                            }
                            placeholder="0"
                            min="0"
                            className="input-custom"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            TOTAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.practical_hrs) || 0) *
                                15 +
                              (parseInt(editCourseData.tw_sl_hrs) || 0)
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div>
                          <label className="block text-sm font-semibold text-gray-700 mb-2">
                            PRACTICAL HRS (Auto)
                          </label>
                          <input
                            type="number"
                            value={
                              (parseInt(editCourseData.practical_hrs) || 0) * 15
                            }
                            readOnly
                            className="input-custom bg-gray-100 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    )}
                  </>
                )}

                {semester?.card_type === "semester" && (
                  <>
                    {/* Credit Limit Checkbox - Only show for semester card types */}
                    <div className="mt-4 pt-4 border-t border-gray-200">
                      <label className="flex items-center space-x-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={editCourseData.count_towards_limit}
                          onChange={(e) =>
                            setEditCourseData({
                              ...editCourseData,
                              count_towards_limit: e.target.checked,
                            })
                          }
                          className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                        />
                        <span className="text-sm font-medium text-gray-700">
                          Include this course's credits in the curriculum's max
                          credit limit calculation
                        </span>
                      </label>
                      <p className="text-xs text-gray-500 mt-2 ml-7">
                        Uncheck this if the course credits should not count
                        towards the total {curriculum?.max_credits || 0} credit
                        limit
                      </p>
                    </div>
                  </>
                )}

                <div className="flex gap-3 justify-end pt-2">
                  <button
                    type="button"
                    onClick={() => setShowEditModal(false)}
                    className="btn-secondary-custom"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="bg-green-600 hover:bg-green-700 text-white font-medium px-5 py-2.5 rounded-lg transition-all duration-200 shadow-sm hover:shadow-md active:scale-95"
                  >
                    Update Course
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
}

export default SemesterDetailPage;
