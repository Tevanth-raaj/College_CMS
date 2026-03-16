import React, { useState, useEffect, useCallback } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

const AcademicCalendarPage = () => {
  const [calendars, setCalendars] = useState([]);
  const [loading, setLoading] = useState(true);
  const [editingId, setEditingId] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [showAdvanceModal, setShowAdvanceModal] = useState(false);
  const [saveMessage, setSaveMessage] = useState(null);
  const [teacherWindows, setTeacherWindows] = useState([]);
  const [teacherWindowLoading, setTeacherWindowLoading] = useState(false);
  const [editingTeacherWindowId, setEditingTeacherWindowId] = useState(null);
  const [teacherWindowForm, setTeacherWindowForm] = useState({
    academic_year: "",
    current_semester_type: "even",
    window_start: "",
    window_end: "",
  });

  const emptyForm = {
    academic_year: "",
    year_level: 1,
    current_semester: 1,
    batch: "",
    semester_start_date: "",
    semester_end_date: "",
    elective_selection_start: "",
    elective_selection_end: "",
    is_current: true,
  };

  const [formData, setFormData] = useState(emptyForm);
  const [advanceData, setAdvanceData] = useState({
    new_academic_year: "",
    semester_start_date: "",
    semester_end_date: "",
  });

  const fetchCalendars = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE_URL}/admin/academic-calendars`);
      const data = await res.json();
      if (data.calendars) {
        setCalendars(data.calendars);
      }
    } catch (err) {
      console.error("Error fetching calendars:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCalendars();
  }, [fetchCalendars]);

  useEffect(() => {
    fetchTeacherWindows();
  }, []);

  const showMessage = (msg, type = "success") => {
    setSaveMessage({ text: msg, type });
    setTimeout(() => setSaveMessage(null), 4000);
  };

  const formatDateForInput = (dateStr) => {
    if (!dateStr) return "";
    try {
      const d = new Date(dateStr);
      return d.toISOString().split("T")[0];
    } catch {
      return "";
    }
  };

  const fetchTeacherWindows = async () => {
    setTeacherWindowLoading(true);
    try {
      const res = await fetch(`${API_BASE_URL}/admin/teacher-course-windows`);
      const data = await res.json();
      setTeacherWindows(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("Error fetching teacher windows:", err);
      setTeacherWindows([]);
    } finally {
      setTeacherWindowLoading(false);
    }
  };

  const resetTeacherWindowForm = () => {
    setEditingTeacherWindowId(null);
    setTeacherWindowForm({
      academic_year: "",
      current_semester_type: "even",
      window_start: "",
      window_end: "",
    });
  };

  const startTeacherWindowEdit = (windowRow) => {
    setEditingTeacherWindowId(windowRow.id);
    setTeacherWindowForm({
      academic_year: windowRow.academic_year || "",
      current_semester_type: (windowRow.current_semester_type || "even").toLowerCase(),
      window_start: windowRow.window_start || "",
      window_end: windowRow.window_end || "",
    });
  };

  const saveTeacherWindow = async (e) => {
    e.preventDefault();
    try {
      const endpoint = editingTeacherWindowId
        ? `${API_BASE_URL}/admin/teacher-course-windows/${editingTeacherWindowId}`
        : `${API_BASE_URL}/admin/teacher-course-windows`;

      const method = editingTeacherWindowId ? "PUT" : "POST";

      const res = await fetch(endpoint, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(teacherWindowForm),
      });

      if (!res.ok) {
        throw new Error("Failed to save teacher selection window");
      }

      showMessage(
        editingTeacherWindowId
          ? "Teacher elective selection window updated (reactivated)."
          : "Teacher elective selection window created."
      );
      resetTeacherWindowForm();
      fetchTeacherWindows();
    } catch (err) {
      showMessage(`Error saving teacher selection window: ${err.message}`, "error");
    }
  };

  const rerunTeacherAllocation = async (windowRow) => {
    const confirmText = `Rerun full allocation for ${windowRow.academic_year} (${windowRow.current_semester_type}) window?`;
    if (!window.confirm(confirmText)) return;

    try {
      const res = await fetch(
        `${API_BASE_URL}/admin/teacher-course-windows/${windowRow.id}/rerun-allocation`,
        { method: "POST" }
      );

      const data = await res.json();
      if (!res.ok || data.success === false) {
        throw new Error(data.error || data.message || "Rerun failed");
      }

      showMessage("Teacher allocation rerun completed for selected window.");
      fetchTeacherWindows();
    } catch (err) {
      showMessage(`Rerun failed: ${err.message}`, "error");
    }
  };

  const deleteTeacherWindow = async (windowRow) => {
    const confirmText = `Delete teacher selection window for ${windowRow.academic_year} (${windowRow.current_semester_type})?`;
    if (!window.confirm(confirmText)) return;

    try {
      const res = await fetch(`${API_BASE_URL}/admin/teacher-course-windows/${windowRow.id}`, {
        method: "DELETE",
      });

      const payload = await res.json().catch(() => ({}));
      if (!res.ok || payload.success === false) {
        throw new Error(payload.error || payload.message || "Delete failed");
      }

      showMessage("Teacher elective selection window deleted.");
      if (editingTeacherWindowId === windowRow.id) {
        resetTeacherWindowForm();
      }
      fetchTeacherWindows();
    } catch (err) {
      showMessage(`Delete failed: ${err.message}`, "error");
    }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    try {
      const body = {
        ...formData,
        batch: formData.batch || null,
        elective_selection_start: formData.elective_selection_start || null,
        elective_selection_end: formData.elective_selection_end || null,
      };
      const res = await fetch(`${API_BASE_URL}/admin/academic-calendars`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (data.success) {
        showMessage("Calendar entry created");
        setShowCreateForm(false);
        setFormData(emptyForm);
        fetchCalendars();
      } else {
        showMessage(data.message || "Failed to create", "error");
      }
    } catch (err) {
      showMessage("Error creating calendar: " + err.message, "error");
    }
  };

  const handleUpdate = async (e) => {
    e.preventDefault();
    try {
      const body = {
        ...formData,
        batch: formData.batch || null,
        elective_selection_start: formData.elective_selection_start || null,
        elective_selection_end: formData.elective_selection_end || null,
      };
      const res = await fetch(
        `${API_BASE_URL}/admin/academic-calendars/${editingId}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      );
      const data = await res.json();
      if (data.success) {
        showMessage("Calendar entry updated");
        setEditingId(null);
        setFormData(emptyForm);
        fetchCalendars();
      } else {
        showMessage(data.message || "Failed to update", "error");
      }
    } catch (err) {
      showMessage("Error updating calendar: " + err.message, "error");
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm("Are you sure you want to delete this calendar entry?"))
      return;
    try {
      const res = await fetch(
        `${API_BASE_URL}/admin/academic-calendars/${id}`,
        { method: "DELETE" },
      );
      const data = await res.json();
      if (data.success) {
        showMessage("Calendar entry deleted");
        fetchCalendars();
      } else {
        showMessage(data.message || "Failed to delete", "error");
      }
    } catch (err) {
      showMessage("Error deleting: " + err.message, "error");
    }
  };

  const handleAdvance = async (e) => {
    e.preventDefault();
    if (
      !window.confirm(
        `This will mark all current entries as inactive and create new entries for ${advanceData.new_academic_year}. Continue?`,
      )
    )
      return;
    try {
      const res = await fetch(
        `${API_BASE_URL}/admin/academic-calendars/advance`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(advanceData),
        },
      );
      const data = await res.json();
      if (data.success) {
        showMessage(
          `Advanced to ${advanceData.new_academic_year} — ${data.created} entries created`,
        );
        setShowAdvanceModal(false);
        setAdvanceData({
          new_academic_year: "",
          semester_start_date: "",
          semester_end_date: "",
        });
        fetchCalendars();
      } else {
        showMessage(data.message || "Failed to advance", "error");
      }
    } catch (err) {
      showMessage("Error advancing: " + err.message, "error");
    }
  };

  const startEdit = (cal) => {
    setEditingId(cal.id);
    setShowCreateForm(false);
    setFormData({
      academic_year: cal.academic_year,
      year_level: cal.year_level,
      current_semester: cal.current_semester,
      batch: cal.batch || "",
      semester_start_date: formatDateForInput(cal.semester_start_date),
      semester_end_date: formatDateForInput(cal.semester_end_date),
      elective_selection_start: formatDateForInput(
        cal.elective_selection_start,
      ),
      elective_selection_end: formatDateForInput(cal.elective_selection_end),
      is_current: cal.is_current,
    });
  };

  const cancelEdit = () => {
    setEditingId(null);
    setShowCreateForm(false);
    setFormData(emptyForm);
  };

  const currentCalendars = calendars.filter((c) => c.is_current);
  const pastCalendars = calendars.filter((c) => !c.is_current);

  return (
    <MainLayout
      title="Academic Calendar"
      subtitle="Manage academic calendar entries, batches, and semesters"
    >
      <div className="space-y-6">
        {/* Status Message */}
        {saveMessage && (
          <div
            className={`px-4 py-3 rounded-lg text-sm font-medium ${
              saveMessage.type === "error"
                ? "bg-red-100 text-red-800 border border-red-200"
                : "bg-green-100 text-green-800 border border-green-200"
            }`}
          >
            {saveMessage.text}
          </div>
        )}

        {/* Teacher Elective Selection Window */}
        <div className="bg-white rounded-lg shadow-md p-6 border border-blue-200 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-bold text-gray-900">Teacher Elective Selection Window</h3>
            <span className="text-xs px-2 py-1 rounded bg-blue-50 text-blue-700 font-medium">
              Auto allocation uses this window
            </span>
          </div>

          <form onSubmit={saveTeacherWindow} className="grid grid-cols-1 md:grid-cols-5 gap-3">
            <input
              type="text"
              value={teacherWindowForm.academic_year}
              onChange={(e) => setTeacherWindowForm({ ...teacherWindowForm, academic_year: e.target.value })}
              placeholder="Academic Year (e.g., 2026-2027)"
              className="w-full px-3 py-2 border rounded-lg text-sm"
              required
            />
            <select
              value={teacherWindowForm.current_semester_type}
              onChange={(e) => setTeacherWindowForm({ ...teacherWindowForm, current_semester_type: e.target.value })}
              className="w-full px-3 py-2 border rounded-lg text-sm"
            >
              <option value="even">Even</option>
              <option value="odd">Odd</option>
            </select>
            <input
              type="date"
              value={teacherWindowForm.window_start}
              onChange={(e) => setTeacherWindowForm({ ...teacherWindowForm, window_start: e.target.value })}
              className="w-full px-3 py-2 border rounded-lg text-sm"
              required
            />
            <input
              type="date"
              value={teacherWindowForm.window_end}
              onChange={(e) => setTeacherWindowForm({ ...teacherWindowForm, window_end: e.target.value })}
              className="w-full px-3 py-2 border rounded-lg text-sm"
              required
            />
            <div className="flex gap-2">
              <button
                type="submit"
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium"
              >
                {editingTeacherWindowId ? "Update Window" : "Create Window"}
              </button>
              {editingTeacherWindowId && (
                <button
                  type="button"
                  onClick={resetTeacherWindowForm}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 text-sm font-medium"
                >
                  Cancel
                </button>
              )}
            </div>
          </form>

          <div className="overflow-x-auto border rounded-lg">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-xs uppercase tracking-wider text-gray-600">
                <tr>
                  <th className="px-3 py-2 text-left">Academic Year</th>
                  <th className="px-3 py-2 text-left">Semester Type</th>
                  <th className="px-3 py-2 text-left">Window</th>
                  <th className="px-3 py-2 text-left">is_active</th>
                  <th className="px-3 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {teacherWindowLoading ? (
                  <tr>
                    <td className="px-3 py-3 text-gray-500" colSpan={5}>Loading teacher windows...</td>
                  </tr>
                ) : teacherWindows.length === 0 ? (
                  <tr>
                    <td className="px-3 py-3 text-gray-500" colSpan={5}>No teacher selection windows configured.</td>
                  </tr>
                ) : (
                  teacherWindows.map((win) => (
                    <tr key={win.id}>
                      <td className="px-3 py-2 font-medium text-gray-900">{win.academic_year}</td>
                      <td className="px-3 py-2 uppercase">{win.current_semester_type}</td>
                      <td className="px-3 py-2 text-gray-700">{win.window_start} → {win.window_end}</td>
                      <td className="px-3 py-2">
                        <span className={`px-2 py-0.5 rounded text-xs font-medium ${win.is_active === 1 ? "bg-blue-100 text-blue-800" : "bg-red-100 text-red-700"}`}>
                          {win.is_active}
                        </span>
                      </td>
                      <td className="px-3 py-2 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => startTeacherWindowEdit(win)}
                            className="px-3 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded hover:bg-blue-100"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => rerunTeacherAllocation(win)}
                            className="px-3 py-1 text-xs font-medium text-amber-700 bg-amber-50 rounded hover:bg-amber-100"
                          >
                            Rerun Allocation
                          </button>
                          <button
                            onClick={() => deleteTeacherWindow(win)}
                            className="px-3 py-1 text-xs font-medium text-red-700 bg-red-50 rounded hover:bg-red-100"
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-md p-6 border border-blue-200 space-y-4">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <h3 className="text-lg font-bold text-gray-900">Academic Calendar Entries</h3>
            <span className="text-xs px-2 py-1 rounded bg-blue-50 text-blue-700 font-medium">
              Manage academic year timeline
            </span>
            <div className="flex gap-2 flex-wrap">
              <button
                onClick={() => {
                  setShowCreateForm(true);
                  setEditingId(null);
                  setFormData(emptyForm);
                }}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium flex items-center gap-2"
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
                    d="M12 4v16m8-8H4"
                  />
                </svg>
                Add Entry
              </button>
              <button
                onClick={() => setShowAdvanceModal(true)}
                className="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 text-sm font-medium flex items-center gap-2"
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
                    d="M13 7l5 5m0 0l-5 5m5-5H6"
                  />
                </svg>
                Advance Academic Year
              </button>
            </div>
          </div>

        {/* Create / Edit Form */}
        {(showCreateForm || editingId !== null) && (
          <div className="bg-white rounded-lg shadow-md p-6 border border-blue-200">
            <h3 className="text-lg font-bold text-gray-900 mb-4">
              {editingId !== null
                ? "Edit Calendar Entry"
                : "Create New Calendar Entry"}
            </h3>
            <form
              onSubmit={editingId !== null ? handleUpdate : handleCreate}
              className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
            >
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Academic Year *
                </label>
                <input
                  type="text"
                  value={formData.academic_year}
                  onChange={(e) =>
                    setFormData({ ...formData, academic_year: e.target.value })
                  }
                  placeholder="e.g., 2025-2026"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Year Level *
                </label>
                <select
                  value={formData.year_level}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      year_level: parseInt(e.target.value),
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value={1}>Year 1 (Freshmen)</option>
                  <option value={2}>Year 2 (Sophomore)</option>
                  <option value={3}>Year 3 (Junior)</option>
                  <option value={4}>Year 4 (Senior)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Current Semester *
                </label>
                <select
                  value={formData.current_semester}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      current_semester: parseInt(e.target.value),
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                >
                  {[1, 2, 3, 4, 5, 6, 7, 8].map((s) => (
                    <option key={s} value={s}>
                      Semester {s}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Batch
                </label>
                <input
                  type="text"
                  value={formData.batch}
                  onChange={(e) =>
                    setFormData({ ...formData, batch: e.target.value })
                  }
                  placeholder="e.g., 2024-2028"
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Semester Start Date *
                </label>
                <input
                  type="date"
                  value={formData.semester_start_date}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      semester_start_date: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Semester End Date *
                </label>
                <input
                  type="date"
                  value={formData.semester_end_date}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      semester_end_date: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Elective Selection Start
                </label>
                <input
                  type="date"
                  value={formData.elective_selection_start}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      elective_selection_start: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Elective Selection End
                </label>
                <input
                  type="date"
                  value={formData.elective_selection_end}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      elective_selection_end: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <div className="flex items-end">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.is_current}
                    onChange={(e) =>
                      setFormData({ ...formData, is_current: e.target.checked })
                    }
                    className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <span className="text-sm font-medium text-gray-700">
                    Is Current
                  </span>
                </label>
              </div>
              <div className="col-span-full flex gap-3 pt-2">
                <button
                  type="submit"
                  className="px-5 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium"
                >
                  {editingId !== null ? "Update" : "Create"}
                </button>
                <button
                  type="button"
                  onClick={cancelEdit}
                  className="px-5 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 text-sm font-medium"
                >
                  Cancel
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Advance Academic Year Modal */}
        {showAdvanceModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md mx-4">
              <h3 className="text-lg font-bold text-gray-900 mb-2">
                Advance Academic Year
              </h3>
              <p className="text-sm text-gray-600 mb-4">
                This will mark all current entries as inactive and create new
                entries. Year 4 students will graduate out, years 1–3 advance,
                and a new Year 1 batch is created automatically.
              </p>
              <form onSubmit={handleAdvance} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    New Academic Year *
                  </label>
                  <input
                    type="text"
                    value={advanceData.new_academic_year}
                    onChange={(e) =>
                      setAdvanceData({
                        ...advanceData,
                        new_academic_year: e.target.value,
                      })
                    }
                    placeholder="e.g., 2026-2027"
                    className="w-full px-3 py-2 border rounded-lg text-sm"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Semester Start Date *
                  </label>
                  <input
                    type="date"
                    value={advanceData.semester_start_date}
                    onChange={(e) =>
                      setAdvanceData({
                        ...advanceData,
                        semester_start_date: e.target.value,
                      })
                    }
                    className="w-full px-3 py-2 border rounded-lg text-sm"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Semester End Date *
                  </label>
                  <input
                    type="date"
                    value={advanceData.semester_end_date}
                    onChange={(e) =>
                      setAdvanceData({
                        ...advanceData,
                        semester_end_date: e.target.value,
                      })
                    }
                    className="w-full px-3 py-2 border rounded-lg text-sm"
                    required
                  />
                </div>
                <div className="flex gap-3 pt-2">
                  <button
                    type="submit"
                    className="px-5 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 text-sm font-medium"
                  >
                    Advance
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowAdvanceModal(false)}
                    className="px-5 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 text-sm font-medium"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <p className="mt-2 text-gray-500 text-sm">Loading...</p>
          </div>
        )}

        {/* Current Calendars Table */}
        {!loading && currentCalendars.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-base font-bold text-blue-800 flex items-center gap-2">
              <span className="w-2.5 h-2.5 bg-blue-500 rounded-full inline-block"></span>
              Current Academic Year
            </h3>
            <div className="overflow-x-auto border rounded-lg">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-gray-600 text-xs uppercase tracking-wider">
                  <tr>
                    <th className="px-4 py-3 text-left">Academic Year</th>
                    <th className="px-4 py-3 text-left">Year Level</th>
                    <th className="px-4 py-3 text-left">Current Semester</th>
                    <th className="px-4 py-3 text-left">Batch</th>
                    <th className="px-4 py-3 text-left">Semester Dates</th>
                    <th className="px-4 py-3 text-left">
                      Elective Selection Window
                    </th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {currentCalendars.map((cal) => (
                    <tr key={cal.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 font-semibold text-gray-900">
                        {cal.academic_year}
                      </td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          Year {cal.year_level}
                        </span>
                      </td>
                      <td className="px-4 py-3 font-medium">
                        Sem {cal.current_semester}
                      </td>
                      <td className="px-4 py-3">
                        {cal.batch ? (
                          <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                            {cal.batch}
                          </span>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {formatDateForInput(cal.semester_start_date)} →{" "}
                        {formatDateForInput(cal.semester_end_date)}
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {cal.elective_selection_start
                          ? `${formatDateForInput(cal.elective_selection_start)} → ${formatDateForInput(cal.elective_selection_end)}`
                          : "—"}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => startEdit(cal)}
                            className="px-3 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded hover:bg-blue-100"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => handleDelete(cal.id)}
                            className="px-3 py-1 text-xs font-medium text-red-700 bg-red-50 rounded hover:bg-red-100"
                          >
                            Delete
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

        {/* Past Calendars Table */}
        {!loading && pastCalendars.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-base font-bold text-gray-600">
              Past Academic Years
            </h3>
            <div className="overflow-x-auto border rounded-lg">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-gray-500 text-xs uppercase tracking-wider">
                  <tr>
                    <th className="px-4 py-3 text-left">Academic Year</th>
                    <th className="px-4 py-3 text-left">Year Level</th>
                    <th className="px-4 py-3 text-left">Semester</th>
                    <th className="px-4 py-3 text-left">Batch</th>
                    <th className="px-4 py-3 text-left">Semester Dates</th>
                    <th className="px-4 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {pastCalendars.map((cal) => (
                    <tr key={cal.id} className="hover:bg-gray-50 text-gray-500">
                      <td className="px-4 py-3">{cal.academic_year}</td>
                      <td className="px-4 py-3">Year {cal.year_level}</td>
                      <td className="px-4 py-3">Sem {cal.current_semester}</td>
                      <td className="px-4 py-3">{cal.batch || "—"}</td>
                      <td className="px-4 py-3">
                        {formatDateForInput(cal.semester_start_date)} →{" "}
                        {formatDateForInput(cal.semester_end_date)}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => startEdit(cal)}
                            className="px-3 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded hover:bg-blue-100"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => handleDelete(cal.id)}
                            className="px-3 py-1 text-xs font-medium text-red-700 bg-red-50 rounded hover:bg-red-100"
                          >
                            Delete
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

        {/* Empty State */}
        {!loading && calendars.length === 0 && (
          <div className="bg-white rounded-lg border p-12 text-center">
            <svg
              className="w-12 h-12 text-gray-300 mx-auto mb-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
            <p className="text-gray-500 text-sm">
              No academic calendar entries found.
            </p>
            <p className="text-gray-400 text-xs mt-1">
              Use "Add Entry" in Academic Calendar Entries to create one.
            </p>
          </div>
        )}

        </div>
      </div>
    </MainLayout>
  );
};

export default AcademicCalendarPage;
