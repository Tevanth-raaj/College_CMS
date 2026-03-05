import React, { useState, useEffect } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

function TeacherStudentMappingPage() {
  const [departments, setDepartments] = useState([]);
  const [years, setYears] = useState([]);
  const [selectedDepartment, setSelectedDepartment] = useState("");
  const [selectedYear, setSelectedYear] = useState("");
  const [academicYear, setAcademicYear] = useState("2025-2026");
  const [teachers, setTeachers] = useState([]);
  const [students, setStudents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [message, setMessage] = useState("");

  // Fetch filter options on component mount
  useEffect(() => {
    fetchFilters();
  }, []);

  // Fetch data when filters change
  useEffect(() => {
    if (selectedDepartment && selectedYear) {
      fetchMappingData();
    }
  }, [selectedDepartment, selectedYear, academicYear]);

  const fetchFilters = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/student-teacher-mapping/filters`,
      );
      const data = await response.json();
      setDepartments(data.departments || []);
      setYears(data.years || []);
    } catch (error) {
      console.error("Error fetching filters:", error);
      setMessage("Failed to load filters");
    }
  };

  const fetchMappingData = async () => {
    setLoading(true);
    try {
      const url = `${API_BASE_URL}/student-teacher-mapping/data?department_id=${selectedDepartment}&year=${selectedYear}&academic_year=${academicYear}`;
      
      const response = await fetch(url);
      const data = await response.json();
      
      setTeachers(data.teachers || []);
      setStudents(data.students || []);
      setMessage("");
    } catch (error) {
      console.error("Error fetching mapping data:", error);
      setMessage("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleAssign = async () => {
    if (!selectedDepartment || !selectedYear) {
      setMessage("Please select department and year");
      return;
    }

    if (teachers.length === 0) {
      setMessage("No teachers available in this department");
      return;
    }

    if (students.length === 0) {
      setMessage("No students available for this selection");
      return;
    }

    setAssigning(true);
    try {
      const response = await fetch(
        `${API_BASE_URL}/student-teacher-mapping/assign`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            department_id: parseInt(selectedDepartment),
            year: parseInt(selectedYear),
            academic_year: academicYear,
          }),
        },
      );

      const result = await response.json();

      if (result.success) {
        setMessage(result.message);
        // Refresh data to show new mappings
        fetchMappingData();
      } else {
        setMessage("Failed to assign students");
      }
    } catch (error) {
      console.error("Error assigning students:", error);
      setMessage("Error assigning students to teachers");
    } finally {
      setAssigning(false);
    }
  };

  const handleClear = async () => {
    if (!selectedDepartment || !selectedYear || !academicYear) {
      setMessage("Please select all filters");
      return;
    }

    if (
      !window.confirm(
        "Are you sure you want to clear all mappings for this selection?",
      )
    ) {
      return;
    }

    try {
      const url = `${API_BASE_URL}/student-teacher-mapping/clear?department_id=${selectedDepartment}&year=${selectedYear}&academic_year=${academicYear}`;
      const response = await fetch(url, { method: "DELETE" });
      const result = await response.json();

      if (result.success) {
        setMessage(result.message);
        fetchMappingData();
      } else {
        setMessage("Failed to clear mappings");
      }
    } catch (error) {
      console.error("Error clearing mappings:", error);
      setMessage("Error clearing mappings");
    }
  };

  const getMappedCount = () => {
    return students.filter((s) => s.teacher_id).length;
  };

  return (
    <MainLayout
      title="Teacher Student Mapping"
      subtitle="Assign students to teachers"
    >
      <div className="p-6 bg-white rounded-lg shadow-sm">
        {/* Filters Section */}
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <h3 className="text-lg font-semibold mb-4">Select Filters</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Department
              </label>
              <select
                value={selectedDepartment}
                onChange={(e) => setSelectedDepartment(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select Department</option>
                {departments.map((dept) => (
                  <option key={dept.id} value={dept.id}>
                    {dept.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Year
              </label>
              <select
                value={selectedYear}
                onChange={(e) => setSelectedYear(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">Select Year</option>
                {years.map((year) => (
                  <option key={year} value={year}>
                    {year}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Academic Year
              </label>
              <input
                type="text"
                value={academicYear}
                onChange={(e) => setAcademicYear(e.target.value)}
                placeholder="2025-2026"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
        </div>

        {/* Message Display */}
        {message && (
          <div
            className={`mb-4 p-3 rounded-md ${
              message.includes("Failed") || message.includes("Error")
                ? "bg-red-100 text-red-700 border border-red-300"
                : "bg-green-100 text-green-700 border border-green-300"
            }`}
          >
            {message}
          </div>
        )}

        {/* Action Buttons */}
        {selectedDepartment && selectedYear && (
          <div className="mb-6 flex gap-4">
            <button
              onClick={handleAssign}
              disabled={assigning || loading}
              className="px-6 py-2 bg-primary text-white rounded-md  disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
            >
              {assigning ? "Assigning..." : "Assign Students to Teachers"}
            </button>
            <button
              onClick={handleClear}
              disabled={assigning || loading}
              className="px-6 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
            >
              Clear All Mappings
            </button>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8">
            <p className="text-gray-600">Loading data...</p>
          </div>
        ) : selectedDepartment && selectedYear ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Teachers Section */}
            <div>
              <div className="bg-blue-50 p-4 rounded-lg mb-4">
                <h3 className="text-lg font-semibold text-blue-900">
                  Teachers ({teachers.length})
                </h3>
              </div>
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {teachers.length === 0 ? (
                  <p className="text-gray-500 text-center py-4">
                    No teachers found for this department
                  </p>
                ) : (
                  teachers.map((teacher) => (
                    <div
                      key={teacher.teacher_id}
                      className="p-4 border border-gray-200 rounded-lg hover:bg-gray-50"
                    >
                      <div className="flex items-center gap-3">
                        {teacher.profile_img && (
                          <img
                            src={`${API_BASE_URL.replace('/api', '')}${teacher.profile_img}`}
                            alt={teacher.teacher_name}
                            className="w-12 h-12 rounded-full object-cover"
                          />
                        )}
                        <div className="flex-1">
                          <h4 className="font-semibold text-gray-900">
                            {teacher.teacher_name}
                          </h4>
                          <p className="text-sm text-gray-600">
                            {teacher.email}
                          </p>
                          <p className="text-sm text-gray-500">
                            {teacher.designation}
                          </p>
                        </div>
                        <div className="text-right">
                          <span className="inline-block px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm font-medium">
                            {teacher.student_count} students
                          </span>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Students Section */}
            <div>
              <div className="bg-green-50 p-4 rounded-lg mb-4">
                <h3 className="text-lg font-semibold text-green-900">
                  Students ({students.length})
                  <span className="ml-2 text-sm font-normal text-green-700">
                    {getMappedCount()} mapped
                  </span>
                </h3>
              </div>
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {students.length === 0 ? (
                  <p className="text-gray-500 text-center py-4">
                    No students found for this selection
                  </p>
                ) : (
                  students.map((student) => (
                    <div
                      key={student.student_id}
                      className={`p-3 border rounded-lg ${
                        student.teacher_id
                          ? "border-green-300 bg-green-50"
                          : "border-gray-200 bg-white"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="font-medium text-gray-900">
                            {student.student_name}
                          </p>
                          <p className="text-sm text-gray-600">
                            {student.enrollment_no}
                          </p>
                        </div>
                        {student.teacher_name && (
                          <div className="text-right">
                            <p className="text-sm text-green-700 font-medium">
                              {student.teacher_name}
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center py-12 text-gray-500">
            <p>Please select department and year to view mapping data</p>
          </div>
        )}

        {/* Statistics Summary */}
        {selectedDepartment && selectedYear && students.length > 0 && (
          <div className="mt-6 mb-4 p-4 bg-gray-50 rounded-lg">
            <h4 className="font-semibold mb-2">Distribution Summary</h4>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
              <div>
                <p className="text-2xl font-bold text-primary">
                  {teachers.length}
                </p>
                <p className="text-sm text-gray-600">Teachers</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-primary">
                  {students.length}
                </p>
                <p className="text-sm text-gray-600">Students</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-primary">
                  {teachers.length > 0
                    ? Math.floor(students.length / teachers.length)
                    : 0}
                </p>
                <p className="text-sm text-gray-600">Per Teacher</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-primary">
                  {teachers.length > 0 ? students.length % teachers.length : 0}
                </p>
                <p className="text-sm text-gray-600">Extra Students</p>
              </div>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
}

export default TeacherStudentMappingPage;
