import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import MainLayout from "../../components/MainLayout";
import TeacherCard from "../../components/TeacherCard";
import { API_BASE_URL } from "../../config";

function TeacherDetailsPage() {
  const navigate = useNavigate();
  const [formData, setFormData] = useState({
    name: "",
    email: "",
    phone: "",
    profile_img: "",
    department: "",
    designation: "",
  });

  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);
  const [teachers, setTeachers] = useState([]);
  const [departments, setDepartments] = useState([]);
  const [searchTerm, setSearchTerm] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editingTeacher, setEditingTeacher] = useState(null);
  const [profileFile, setProfileFile] = useState(null);
  const [profilePreview, setProfilePreview] = useState("");

  // Fetch teachers from backend
  const fetchTeachers = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/teachers`);
      if (!response.ok) {
        throw new Error("Failed to fetch teachers");
      }
      const data = await response.json();
      setTeachers(data || []);
    } catch (err) {
      console.error("Error fetching teachers:", err);
      setError("Failed to load teachers. Please try again.");
    }
  };

  // Fetch departments from backend
  const fetchDepartments = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/departments`);
      if (!response.ok) {
        throw new Error("Failed to fetch departments");
      }
      const data = await response.json();
      setDepartments(data.departments || []);
    } catch (err) {
      console.error("Error fetching departments:", err);
      setError("Failed to load departments. Please try again.");
    }
  };

  // Fetch teachers and departments on component mount
  useEffect(() => {
    fetchTeachers();
    fetchDepartments();
  }, []);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      // Validate file type
      if (!file.type.startsWith("image/")) {
        setError("Please select a valid image file");
        return;
      }
      // Validate file size (max 5MB)
      if (file.size > 5 * 1024 * 1024) {
        setError("Image size should be less than 5MB");
        return;
      }
      setProfileFile(file);
      // Create preview
      const reader = new FileReader();
      reader.onloadend = () => {
        setProfilePreview(reader.result);
      };
      reader.readAsDataURL(file);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);

    try {
      // Create FormData for file upload
      const formDataToSend = new FormData();
      formDataToSend.append("name", formData.name);
      formDataToSend.append("email", formData.email);
      formDataToSend.append("phone", formData.phone);
      formDataToSend.append("department", formData.department);
      formDataToSend.append("designation", formData.designation);

      if (profileFile) {
        formDataToSend.append("profile_img", profileFile);
      }

      if (editingTeacher) {
        // Check if anything was changed
        const isChanged =
          formData.name !== editingTeacher.name ||
          formData.email !== editingTeacher.email ||
          formData.phone !== (editingTeacher.phone || "") ||
          profileFile !== null ||
          formData.department !== (editingTeacher.department || "") ||
          formData.designation !==
            (editingTeacher.desg || editingTeacher.designation || "");

        if (!isChanged) {
          setError(
            "No changes detected. Please update at least one field or click Cancel.",
          );
          setLoading(false);
          return;
        }

        // Update existing teacher
        const response = await fetch(
          `${API_BASE_URL}/teachers/${editingTeacher.id}`,
          {
            method: "PUT",
            body: formDataToSend,
          },
        );

        if (!response.ok) {
          const errorData = await response.text();
          throw new Error(errorData || "Failed to update teacher");
        }

        const updatedTeacher = await response.json();
        setTeachers(
          teachers.map((t) =>
            t.id === editingTeacher.id ? updatedTeacher : t,
          ),
        );
        setSuccess("Teacher updated successfully!");
      } else {
        // Create new teacher
        const response = await fetch(`${API_BASE_URL}/teachers`, {
          method: "POST",
          body: formDataToSend,
        });

        if (!response.ok) {
          const errorData = await response.text();
          throw new Error(errorData || "Failed to create teacher");
        }

        const newTeacher = await response.json();
        setTeachers([newTeacher, ...teachers]);
        setSuccess("Teacher created successfully!");
      }

      // Reset form and state
      resetForm();
      setEditingTeacher(null);
      setShowForm(false);
    } catch (err) {
      console.error("Error saving teacher:", err);
      setError(err.message || "Failed to save teacher. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setFormData({
      name: "",
      email: "",
      phone: "",
      profile_img: "",
      department: "",
      designation: "",
    });
    setProfileFile(null);
    setProfilePreview("");
  };

  const handleEdit = (teacher) => {
    setFormData({
      name: teacher.name || "",
      email: teacher.email || "",
      phone: teacher.phone || "",
      profile_img: teacher.profile_img || "",
      department: teacher.department || "",
      designation: teacher.desg || teacher.designation || "",
    });
    setEditingTeacher(teacher);
    setShowForm(true);
    setError("");
    setProfileFile(null);
    if (teacher.profile_img) {
      // Remove /api from URL for static files
      const baseUrl = API_BASE_URL.replace("/api", "");
      setProfilePreview(`${baseUrl}${teacher.profile_img}`);
    } else {
      setProfilePreview("");
    }
  };

  const handleDelete = async (teacher) => {
    if (window.confirm(`Are you sure you want to delete ${teacher.name}?`)) {
      try {
        const response = await fetch(`${API_BASE_URL}/teachers/${teacher.id}`, {
          method: "DELETE",
        });

        if (!response.ok) {
          const errorData = await response.text();
          throw new Error(errorData || "Failed to delete teacher");
        }

        // Remove teacher from local state
        setTeachers(teachers.filter((t) => t.id !== teacher.id));
        setSuccess("Teacher deleted successfully!");
      } catch (err) {
        console.error("Error deleting teacher:", err);
        setError(err.message || "Failed to delete teacher. Please try again.");
      }
    }
  };

  return (
    <MainLayout
      title="Teacher Details"
      subtitle="Add and manage teacher information"
      actions={
        !showForm && (
          <div className="flex items-center space-x-3">
            <input
              type="search"
              placeholder="Search by name, teacher id or department..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="input-custom w-64"
            />
            <button
              type="button"
              onClick={() => setShowForm(true)}
              className="btn-primary-custom"
            >
              Create Teacher
            </button>
          </div>
        )
      }
    >
      <div className="min-w-screen mx-auto">
        {/* Messages */}
        {error && (
          <div className="mb-6 flex items-start space-x-3 p-4 bg-red-50 border border-red-200 rounded-lg">
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
          <div className="mb-6 flex items-start space-x-3 p-4 bg-green-50 border border-green-200 rounded-lg">
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

        {/* Teacher List (hidden when form shown) */}
        {!showForm && (
          <div className="mb-8">
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
              {teachers
                .filter((t) => {
                  if (!searchTerm) return true;
                  const q = searchTerm.toLowerCase();
                  const name = (t.name || "").toLowerCase();
                  const id = (t.id || "").toString().toLowerCase();
                  const dept = (t.department || "").toLowerCase();
                  return name.includes(q) || id.includes(q) || dept.includes(q);
                })
                .map((teacher) => (
                  <TeacherCard
                    key={teacher.id}
                    teacher={teacher}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                  />
                ))}
            </div>
          </div>
        )}

        {/* Teacher Entry Form (toggleable) */}
        {showForm && (
          <div className="card-custom p-8">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-2xl font-bold text-gray-900">
                {editingTeacher
                  ? "Edit Teacher - " + (editingTeacher.name || "Teacher")
                  : "Add New Teacher"}
              </h2>
              <button
                type="button"
                onClick={() => {
                  setShowForm(false);
                  setEditingTeacher(null);
                  resetForm();
                }}
                className="text-gray-500 hover:text-gray-700 text-2xl"
              >
                ✕
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-8">
              {/* Teacher Information */}
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4 pb-2 border-b border-gray-200">
                  Teacher Information
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="md:col-span-2">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Teacher Name <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      name="name"
                      value={formData.name}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="Full name of teacher"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Email <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="email"
                      name="email"
                      value={formData.email}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="email@example.com"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Phone <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="tel"
                      name="phone"
                      value={formData.phone}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                      placeholder="+91 XXXXX XXXXX"
                    />
                  </div>

                  <div className="md:col-span-2">
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Profile Image
                    </label>
                    <div className="flex items-start space-x-4">
                      <div className="flex-1">
                        <input
                          type="file"
                          accept="image/*"
                          onChange={handleFileChange}
                          className="input-custom"
                        />
                        <p className="text-xs text-gray-500 mt-1">
                          Maximum file size: 5MB. Accepted formats: JPG, PNG,
                          GIF
                        </p>
                      </div>
                      {profilePreview && (
                        <div className="flex-shrink-0">
                          <img
                            src={profilePreview}
                            alt="Preview"
                            className="w-24 h-24 object-cover rounded-lg border border-gray-300"
                          />
                        </div>
                      )}
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Department <span className="text-red-500">*</span>
                    </label>
                    <select
                      name="department"
                      value={formData.department}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
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
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Designation <span className="text-red-500">*</span>
                    </label>
                    <select
                      name="designation"
                      value={formData.designation}
                      onChange={handleInputChange}
                      required
                      className="input-custom"
                    >
                      <option value="">Select Designation</option>
                      <option value="Professor">Professor</option>
                      <option value="Associate Professor">
                        Associate Professor
                      </option>
                      <option value="Assistant Professor">
                        Assistant Professor
                      </option>
                      <option value="Lecturer">Lecturer</option>
                      <option value="Lab Assistant">Lab Assistant</option>
                    </select>
                  </div>
                </div>
              </div>

              {/* Submit Button */}
              <div className="flex justify-end space-x-4 pt-4">
                <button
                  type="button"
                  onClick={resetForm}
                  className="px-6 py-2.5 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 font-medium transition-colors"
                >
                  Reset
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setShowForm(false);
                    setEditingTeacher(null);
                    resetForm();
                  }}
                  className="px-6 py-2.5 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 font-medium transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="btn-primary-custom disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {editingTeacher
                    ? loading
                      ? "Updating Teacher..."
                      : "Update Teacher"
                    : loading
                      ? "Adding Teacher..."
                      : "Add Teacher"}
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </MainLayout>
  );
}

export default TeacherDetailsPage;
