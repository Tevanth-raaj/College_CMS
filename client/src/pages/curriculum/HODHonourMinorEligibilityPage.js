import React, { useState, useEffect, useRef } from "react";
import MainLayout from "../../components/MainLayout";
import { API_BASE_URL } from "../../config";

const HODHonourMinorEligibilityPage = () => {
  // Honour state
  const [honourFile, setHonourFile] = useState(null);
  const [honourUploading, setHonourUploading] = useState(false);

  // Minor state
  const [minorFile, setMinorFile] = useState(null);
  const [minorUploading, setMinorUploading] = useState(false);

  // Common state
  const [message, setMessage] = useState("");
  const [messageType, setMessageType] = useState("info");
  const [exportingTeachers, setExportingTeachers] = useState(false);
  const [exportingTeacherPreferences, setExportingTeacherPreferences] = useState(false);

  // Window selection state
  const [windows, setWindows] = useState([]);
  const [selectedWindow, setSelectedWindow] = useState(null);
  const [loadingWindows, setLoadingWindows] = useState(false);

  // Department dropdown state
  const [departments, setDepartments] = useState([]);
  const [selectedDept, setSelectedDept] = useState(""); // "" = All
  const [deptSearch, setDeptSearch] = useState("");
  const [deptOpen, setDeptOpen] = useState(false);
  const deptRef = useRef(null);
  const windowRef = useRef(null);
  const courseRef = useRef(null);

  // Close dept dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (deptRef.current && !deptRef.current.contains(e.target)) {
        setDeptOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Close window dropdown on outside click
  const [windowSearch, setWindowSearch] = useState("");
  const [windowOpen, setWindowOpen] = useState(false);
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (windowRef.current && !windowRef.current.contains(e.target)) {
        setWindowOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Course filter state
  const [courses, setCourses] = useState([]);
  const [selectedCourse, setSelectedCourse] = useState(null); // { course_code, course_name, label }
  const [courseSearch, setCourseSearch] = useState("");
  const [courseOpen, setCourseOpen] = useState(false);
  const [preferencePeriods, setPreferencePeriods] = useState([]);
  const [loadingPreferencePeriods, setLoadingPreferencePeriods] = useState(false);
  const [selectedPreferencePeriod, setSelectedPreferencePeriod] = useState("");
  const [preferenceDept, setPreferenceDept] = useState("");
  const [preferenceCourses, setPreferenceCourses] = useState([]);
  const [selectedPreferenceCourse, setSelectedPreferenceCourse] = useState("");
  const [preferenceTeachers, setPreferenceTeachers] = useState([]);
  const [selectedPreferenceTeacher, setSelectedPreferenceTeacher] = useState("");
  const [loadingPreferenceTeachers, setLoadingPreferenceTeachers] = useState(false);
  const [preferencePeriodSearch, setPreferencePeriodSearch] = useState("");
  const [preferencePeriodOpen, setPreferencePeriodOpen] = useState(false);
  const [preferenceDeptSearch, setPreferenceDeptSearch] = useState("");
  const [preferenceDeptOpen, setPreferenceDeptOpen] = useState(false);
  const [preferenceCourseSearch, setPreferenceCourseSearch] = useState("");
  const [preferenceCourseOpen, setPreferenceCourseOpen] = useState(false);
  const [preferenceTeacherSearch, setPreferenceTeacherSearch] = useState("");
  const [preferenceTeacherOpen, setPreferenceTeacherOpen] = useState(false);
  const preferencePeriodRef = useRef(null);
  const preferenceDeptRef = useRef(null);
  const preferenceCourseRef = useRef(null);
  const preferenceTeacherRef = useRef(null);
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (courseRef.current && !courseRef.current.contains(e.target)) {
        setCourseOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    const handleClickOutside = (e) => {
      if (preferencePeriodRef.current && !preferencePeriodRef.current.contains(e.target)) {
        setPreferencePeriodOpen(false);
      }
      if (preferenceDeptRef.current && !preferenceDeptRef.current.contains(e.target)) {
        setPreferenceDeptOpen(false);
      }
      if (preferenceCourseRef.current && !preferenceCourseRef.current.contains(e.target)) {
        setPreferenceCourseOpen(false);
      }
      if (preferenceTeacherRef.current && !preferenceTeacherRef.current.contains(e.target)) {
        setPreferenceTeacherOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const hodDepartmentId = (localStorage.getItem("teacher_dept") || "").trim();

  // Fetch departments on mount
  useEffect(() => {
    const fetchDepartments = async () => {
      try {
        const res = await fetch(`${API_BASE_URL}/departments`);
        if (res.ok) {
          const data = await res.json();
          if (Array.isArray(data)) setDepartments(data);
          else if (data.success && Array.isArray(data.departments)) setDepartments(data.departments);
        }
      } catch (e) {
        // Non-fatal
      }
    };
    fetchDepartments();
  }, []);

  // Fetch courses whenever selectedDept changes
  useEffect(() => {
    const fetchCourses = async () => {
      try {
        let url = `${API_BASE_URL}/hod/teacher-limits/courses`;
        const deptFilter = selectedDept || hodDepartmentId;
        if (deptFilter) url += `?department_id=${encodeURIComponent(deptFilter)}`;
        const res = await fetch(url);
        if (res.ok) {
          const data = await res.json();
          setCourses(Array.isArray(data) ? data : []);
        }
      } catch (e) {
        // Non-fatal
      }
    };
    // Reset course selection when dept changes
    setSelectedCourse(null);
    setCourseSearch("");
    fetchCourses();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDept]);

  // Fetch allocation windows on mount
  useEffect(() => {
    const fetchWindows = async () => {
      setLoadingWindows(true);
      try {
        let url = `${API_BASE_URL}/hod/teacher-limits/windows`;
        if (hodDepartmentId) {
          const params = new URLSearchParams({ department_id: hodDepartmentId });
          url += `?${params.toString()}`;
        }
        const res = await fetch(url);
        if (res.ok) {
          const data = await res.json();
          setWindows(data);
          if (data.length > 0) {
            // Default to latest window (first in the list – ordered DESC)
            setSelectedWindow(data[0]);
          }
        }
      } catch (e) {
        // Non-fatal: export will still work without filter
      } finally {
        setLoadingWindows(false);
      }
    };
    fetchWindows();
  }, [hodDepartmentId]);

  useEffect(() => {
    const fetchPreferencePeriods = async () => {
      setLoadingPreferencePeriods(true);
      try {
        let url = `${API_BASE_URL}/hod/teacher-preferences/periods`;
        const deptFilter = preferenceDept || hodDepartmentId;
        if (deptFilter) url += `?department_id=${encodeURIComponent(deptFilter)}`;
        const res = await fetch(url);
        if (res.ok) {
          const data = await res.json();
          setPreferencePeriods(Array.isArray(data) ? data : []);
        } else {
          setPreferencePeriods([]);
        }
      } catch {
        setPreferencePeriods([]);
      } finally {
        setLoadingPreferencePeriods(false);
      }
    };
    fetchPreferencePeriods();
  }, [preferenceDept, hodDepartmentId]);

  useEffect(() => {
    const fetchPreferenceCourses = async () => {
      try {
        let url = `${API_BASE_URL}/hod/teacher-limits/courses`;
        const deptFilter = preferenceDept || hodDepartmentId;
        if (deptFilter) url += `?department_id=${encodeURIComponent(deptFilter)}`;
        const res = await fetch(url);
        if (res.ok) {
          const data = await res.json();
          setPreferenceCourses(Array.isArray(data) ? data : []);
        } else {
          setPreferenceCourses([]);
        }
      } catch {
        setPreferenceCourses([]);
      }
    };
    setSelectedPreferenceCourse("");
    setPreferenceCourseSearch("");
    fetchPreferenceCourses();
  }, [preferenceDept, hodDepartmentId]);

  useEffect(() => {
    const fetchPreferenceTeachers = async () => {
      setLoadingPreferenceTeachers(true);
      try {
        let url = `${API_BASE_URL}/hod/teacher-preferences/teachers`;
        const deptFilter = preferenceDept || hodDepartmentId;
        if (deptFilter) url += `?department_id=${encodeURIComponent(deptFilter)}`;
        const res = await fetch(url);
        if (res.ok) {
          const data = await res.json();
          setPreferenceTeachers(Array.isArray(data) ? data : []);
        } else {
          setPreferenceTeachers([]);
        }
      } catch {
        setPreferenceTeachers([]);
      } finally {
        setLoadingPreferenceTeachers(false);
      }
    };
    setSelectedPreferenceTeacher("");
    setPreferenceTeacherSearch("");
    fetchPreferenceTeachers();
  }, [preferenceDept, hodDepartmentId]);

  // Honour templates and imports
  const handleDownloadHonourTemplate = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/hod/honour-eligibility/template`);
      if (!response.ok) {
        throw new Error("Failed to download honour template");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "student_eligible_honour_template.csv";
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);

      setMessageType("success");
      setMessage("Honour template downloaded successfully.");
    } catch (error) {
      setMessageType("error");
      setMessage("Failed to download honour template.");
    }
  };

  const handleImportHonourData = async () => {
    if (!honourFile) {
      setMessageType("error");
      setMessage("Please choose a CSV file to import for honour eligibility.");
      return;
    }

    setHonourUploading(true);
    setMessage("");

    try {
      const formData = new FormData();
      formData.append("file", honourFile);

      const response = await fetch(`${API_BASE_URL}/hod/honour-eligibility/import`, {
        method: "POST",
        body: formData,
      });

      const data = await response.json();

      if (!response.ok || !data.success) {
        const firstError = Array.isArray(data.errors) && data.errors.length > 0 ? ` ${data.errors[0]}` : "";
        throw new Error(data.message || `Honour import failed.${firstError}`);
      }

      setMessageType("success");
      setMessage(`Honour import successful. Inserted: ${data.inserted || 0}, Skipped: ${data.skipped || 0}`);
      setHonourFile(null);
      const fileInput = document.getElementById("honour-import-file");
      if (fileInput) fileInput.value = "";
    } catch (error) {
      setMessageType("error");
      setMessage(error.message || "Honour import failed.");
    } finally {
      setHonourUploading(false);
    }
  };

  // Minor templates and imports
  const handleDownloadMinorTemplate = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/hod/minor-eligibility/template`);
      if (!response.ok) {
        throw new Error("Failed to download minor template");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "student_eligible_minor_template.csv";
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);

      setMessageType("success");
      setMessage("Minor template downloaded successfully.");
    } catch (error) {
      setMessageType("error");
      setMessage("Failed to download minor template.");
    }
  };

  const handleImportMinorData = async () => {
    if (!minorFile) {
      setMessageType("error");
      setMessage("Please choose a CSV file to import for minor eligibility.");
      return;
    }

    setMinorUploading(true);
    setMessage("");

    try {
      const formData = new FormData();
      formData.append("file", minorFile);

      const response = await fetch(`${API_BASE_URL}/hod/minor-eligibility/import`, {
        method: "POST",
        body: formData,
      });

      const data = await response.json();

      if (!response.ok || !data.success) {
        const firstError = Array.isArray(data.errors) && data.errors.length > 0 ? ` ${data.errors[0]}` : "";
        throw new Error(data.message || `Minor import failed.${firstError}`);
      }

      setMessageType("success");
      setMessage(`Minor import successful. Inserted: ${data.inserted || 0}, Skipped: ${data.skipped || 0}`);
      setMinorFile(null);
      const fileInput = document.getElementById("minor-import-file");
      if (fileInput) fileInput.value = "";
    } catch (error) {
      setMessageType("error");
      setMessage(error.message || "Minor import failed.");
    } finally {
      setMinorUploading(false);
    }
  };

  const handleExportTeacherLimits = async () => {
    try {
      setExportingTeachers(true);
      // selectedDept overrides hodDepartmentId; "" = All (no dept filter)
      const deptFilter = selectedDept || hodDepartmentId;
      let url = `${API_BASE_URL}/hod/teacher-limits/export`;
      const params = new URLSearchParams();
      if (selectedWindow) {
        params.set("window_start", selectedWindow.window_start);
        params.set("window_end", selectedWindow.window_end);
      }
      if (deptFilter) {
        params.set("department_id", deptFilter);
      }
      if (selectedCourse) {
        params.set("course_code", selectedCourse.course_code);
      }
      if ([...params].length > 0) {
        url += `?${params.toString()}`;
      }
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error("Failed to export teacher data");
      }

      const blob = await response.blob();
      const urlObj = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = urlObj;
      const windowLabel = selectedWindow
        ? `${selectedWindow.academic_year}_${selectedWindow.semester_type}`
        : new Date().toISOString().slice(0, 10);
      const courseLabel = selectedCourse
        ? selectedCourse.course_code === "__ALL__"
          ? "all_courses_expanded"
          : `course_${selectedCourse.course_code}`
        : "";
      a.download = `teacher_limits_${windowLabel}${courseLabel ? "_" + courseLabel : ""}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(urlObj);

      setMessageType("success");
      setMessage("Teacher limits data exported successfully.");
    } catch (error) {
      setMessageType("error");
      setMessage("Failed to export teacher limits data.");
    } finally {
      setExportingTeachers(false);
    }
  };

  const handleExportTeacherPreferences = async () => {
    try {
      setExportingTeacherPreferences(true);
      const params = new URLSearchParams();
      const deptFilter = preferenceDept || hodDepartmentId;
      if (selectedPreferencePeriod) {
        const [academicYear, semesterType] = selectedPreferencePeriod.split("||");
        if (academicYear) params.set("academic_year", academicYear);
        if (semesterType) params.set("semester_type", semesterType);
      }
      if (deptFilter) params.set("department_id", deptFilter);
      if (selectedPreferenceCourse) params.set("course_code", selectedPreferenceCourse);
      if (selectedPreferenceTeacher) params.set("teacher_id", selectedPreferenceTeacher);

      let url = `${API_BASE_URL}/hod/teacher-preferences/export`;
      if ([...params].length > 0) url += `?${params.toString()}`;

      const response = await fetch(url);
      if (!response.ok) throw new Error("Failed to export teacher preference data");

      const blob = await response.blob();
      const urlObj = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = urlObj;
      a.download = `teacher_preferences_${new Date().toISOString().slice(0, 10)}.xlsx`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(urlObj);

      setMessageType("success");
      setMessage("Teacher preference data exported successfully.");
    } catch (error) {
      setMessageType("error");
      setMessage("Failed to export teacher preference data.");
    } finally {
      setExportingTeacherPreferences(false);
    }
  };

  const adminInputClass =
    "w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary pr-8";
  const adminDropdownClass =
    "absolute z-50 mt-1 w-full bg-white border border-gray-300 rounded-lg shadow-lg max-h-60 overflow-y-auto text-sm";
  const adminOptionClass = "px-3 py-2 cursor-pointer hover:bg-blue-50 border-b border-gray-100 last:border-0";
  const adminPrimaryBtnClass =
    "bg-primary text-white px-3 py-2 rounded-lg text-sm disabled:opacity-60 disabled:cursor-not-allowed";

  return (
    <MainLayout
      title="Honour/Minor Eligibility Management"
      subtitle="Download templates and import student eligibility data for honour and minor programs"
    >
      <div className="max-w-4xl space-y-6">
        {/* Teacher Export Section */}
        <div className="bg-white border rounded-lg p-4 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Export Teacher Data</h2>
          <p className="text-sm text-gray-600">
            Export teacher workload and course allocation data including assigned limits, subjects, and labs.
          </p>

          {/* Department selector — searchable combobox */}
          <div className="space-y-1" ref={deptRef}>
            <label className="block text-sm font-medium text-gray-700">
              Department
            </label>
            <div className="relative w-full max-w-md">
              <input
                type="text"
                value={deptSearch}
                placeholder={
                  selectedDept
                    ? (departments.find((d) => String(d.id) === selectedDept)?.name ?? "All Departments")
                    : "All Departments"
                }
                onChange={(e) => {
                  setDeptSearch(e.target.value);
                  setDeptOpen(true);
                }}
                onFocus={() => setDeptOpen(true)}
                className={adminInputClass}
              />
              {/* Clear / chevron button */}
              <button
                type="button"
                onClick={() => {
                  setSelectedDept("");
                  setDeptSearch("");
                  setDeptOpen(false);
                }}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                title="Clear"
              >
                {selectedDept ? "✕" : "▾"}
              </button>
              {deptOpen && (
                <ul className={adminDropdownClass}>
                  {/* All option */}
                  {("all departments").includes(deptSearch.toLowerCase()) || deptSearch === "" ? (
                    <li
                      className={`${adminOptionClass} ${
                        selectedDept === "" ? "bg-blue-50 font-medium text-primary" : ""
                      }`}
                      onMouseDown={() => {
                        setSelectedDept("");
                        setDeptSearch("");
                        setDeptOpen(false);
                      }}
                    >
                      All Departments
                    </li>
                  ) : null}
                  {departments
                    .filter((d) =>
                      d.name.toLowerCase().includes(deptSearch.toLowerCase())
                    )
                    .map((d) => (
                      <li
                        key={d.id}
                        className={`${adminOptionClass} ${
                          selectedDept === String(d.id) ? "bg-blue-50 font-medium text-primary" : ""
                        }`}
                        onMouseDown={() => {
                          setSelectedDept(String(d.id));
                          setDeptSearch("");
                          setDeptOpen(false);
                        }}
                      >
                        {d.name}
                      </li>
                    ))}
                  {departments.filter((d) =>
                    d.name.toLowerCase().includes(deptSearch.toLowerCase())
                  ).length === 0 && deptSearch !== "" && (
                    <li className="px-3 py-2 text-gray-400 italic">No departments match</li>
                  )}
                </ul>
              )}
            </div>
          </div>

          {/* Window selector — searchable combobox */}
          <div className="space-y-1" ref={windowRef}>
            <label className="block text-sm font-medium text-gray-700">
              Allocation Window
            </label>
            {loadingWindows ? (
              <p className="text-sm text-gray-500">Loading windows…</p>
            ) : windows.length === 0 ? (
              <p className="text-sm text-gray-400 italic">No allocation windows found — exporting all records.</p>
            ) : (
              <div className="relative w-full max-w-md">
                <input
                  type="text"
                  value={windowSearch}
                  placeholder={selectedWindow ? selectedWindow.label : "All Windows"}
                  onChange={(e) => {
                    setWindowSearch(e.target.value);
                    setWindowOpen(true);
                  }}
                  onFocus={() => setWindowOpen(true)}
                    className={adminInputClass}
                />
                <button
                  type="button"
                  onClick={() => {
                    setSelectedWindow(null);
                    setWindowSearch("");
                    setWindowOpen(false);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                  title="Clear"
                >
                  {selectedWindow ? "✕" : "▾"}
                </button>
                {windowOpen && (
                  <ul className={adminDropdownClass}>
                    {/* All windows option */}
                    {("all windows").includes(windowSearch.toLowerCase()) || windowSearch === "" ? (
                      <li
                        className={`${adminOptionClass} ${
                          !selectedWindow ? "bg-blue-50 font-medium text-primary" : ""
                        }`}
                        onMouseDown={() => {
                          setSelectedWindow(null);
                          setWindowSearch("");
                          setWindowOpen(false);
                        }}
                      >
                        All Windows
                      </li>
                    ) : null}
                    {windows
                      .filter((w) =>
                        w.label.toLowerCase().includes(windowSearch.toLowerCase())
                      )
                      .map((w) => {
                        const key = `${w.window_start}||${w.window_end}`;
                        const isSelected = selectedWindow && `${selectedWindow.window_start}||${selectedWindow.window_end}` === key;
                        return (
                          <li
                            key={key}
                            className={`${adminOptionClass} ${
                              isSelected ? "bg-blue-50 font-medium text-primary" : ""
                            }`}
                            onMouseDown={() => {
                              setSelectedWindow(w);
                              setWindowSearch("");
                              setWindowOpen(false);
                            }}
                          >
                            {w.label}
                          </li>
                        );
                      })}
                    {windows.filter((w) =>
                      w.label.toLowerCase().includes(windowSearch.toLowerCase())
                    ).length === 0 && windowSearch !== "" && (
                      <li className="px-3 py-2 text-gray-400 italic">No windows match</li>
                    )}
                  </ul>
                )}
              </div>
            )}
          </div>

          {/* Course filter — searchable combobox */}
          <div className="space-y-1" ref={courseRef}>
            <label className="block text-sm font-medium text-gray-700">
              Course
              {!selectedCourse && (
                <span className="ml-2 text-xs font-normal text-gray-400">(leave blank for normal export)</span>
              )}
              {selectedCourse && (
                <span className="ml-2 text-xs text-amber-600 font-medium">
                  {selectedCourse.course_code === "__ALL__"
                    ? "↪ expanded: one row per teacher × course"
                    : `↪ expanded: teachers for ${selectedCourse.course_code}`}
                </span>
              )}
            </label>
            <div className="relative w-full max-w-md">
              <input
                type="text"
                value={courseSearch}
                placeholder={
                  selectedCourse
                    ? selectedCourse.label
                    : "Select a course to enable expanded export..."
                }
                onChange={(e) => {
                  setCourseSearch(e.target.value);
                  setCourseOpen(true);
                }}
                onFocus={() => setCourseOpen(true)}
                className={adminInputClass}
              />
              <button
                type="button"
                onClick={() => {
                  setSelectedCourse(null);
                  setCourseSearch("");
                  setCourseOpen(false);
                }}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                title={selectedCourse ? "Clear (revert to normal export)" : "Open"}
              >
                {selectedCourse ? "✕" : "▾"}
              </button>
              {courseOpen && (
                <ul className={adminDropdownClass}>
                  {/* __ALL__ sentinel — expanded view for every course */}
                  {"all courses (expanded)".includes(courseSearch.toLowerCase()) || courseSearch === "" ? (
                    <li
                      className={`${adminOptionClass} ${
                        selectedCourse?.course_code === "__ALL__" ? "bg-blue-50 font-medium text-primary" : ""
                      }`}
                      onMouseDown={() => {
                        setSelectedCourse({ course_code: "__ALL__", course_name: "", label: "— All Courses (expanded) —" });
                        setCourseSearch("");
                        setCourseOpen(false);
                      }}
                    >
                      — All Courses (expanded) —
                    </li>
                  ) : null}
                  {courses
                    .filter((c) =>
                      c.label.toLowerCase().includes(courseSearch.toLowerCase())
                    )
                    .map((c) => (
                      <li
                        key={c.course_code}
                        className={`${adminOptionClass} ${
                          selectedCourse?.course_code === c.course_code ? "bg-blue-50 font-medium text-primary" : ""
                        }`}
                        onMouseDown={() => {
                          setSelectedCourse(c);
                          setCourseSearch("");
                          setCourseOpen(false);
                        }}
                      >
                        {c.label}
                      </li>
                    ))}
                  {courses.filter((c) =>
                    c.label.toLowerCase().includes(courseSearch.toLowerCase())
                  ).length === 0 && courseSearch !== "" && (
                    <li className="px-3 py-2 text-gray-400 italic">No courses match</li>
                  )}
                </ul>
              )}
            </div>
          </div>

          <button
            onClick={handleExportTeacherLimits}
            disabled={exportingTeachers}
            className={adminPrimaryBtnClass}
          >
            {exportingTeachers ? "Exporting..." : "Export Teacher Data"}
          </button>
        </div>

        <div className="bg-white border rounded-lg p-4 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">Export Teacher Preference Data</h2>
          <p className="text-sm text-gray-600">
            Export teacher submitted course preference data filtered by academic year/semester type, department, and course.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1" ref={preferencePeriodRef}>
              <label className="block text-sm font-medium text-gray-700">Academic Year / Semester Type</label>
              <div className="relative w-full">
                <input
                  type="text"
                  value={preferencePeriodSearch}
                  placeholder={
                    selectedPreferencePeriod
                      ? (preferencePeriods.find((p) => `${p.academic_year}||${p.semester_type}` === selectedPreferencePeriod)?.label || "Selected Period")
                      : "All Periods"
                  }
                  onChange={(e) => {
                    setPreferencePeriodSearch(e.target.value);
                    setPreferencePeriodOpen(true);
                  }}
                  onFocus={() => setPreferencePeriodOpen(true)}
                  className={adminInputClass}
                />
                <button
                  type="button"
                  onClick={() => {
                    setSelectedPreferencePeriod("");
                    setPreferencePeriodSearch("");
                    setPreferencePeriodOpen(false);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                  title="Clear"
                >
                  {selectedPreferencePeriod ? "✕" : "▾"}
                </button>
                {preferencePeriodOpen && (
                  <ul className={adminDropdownClass}>
                    {("all periods").includes(preferencePeriodSearch.toLowerCase()) || preferencePeriodSearch === "" ? (
                      <li
                        className={`${adminOptionClass} ${!selectedPreferencePeriod ? "bg-blue-50 font-medium text-primary" : ""}`}
                        onMouseDown={() => {
                          setSelectedPreferencePeriod("");
                          setPreferencePeriodSearch("");
                          setPreferencePeriodOpen(false);
                        }}
                      >
                        All Periods
                      </li>
                    ) : null}
                    {preferencePeriods
                      .filter((period) => (period.label || `${period.academic_year} - ${period.semester_type}`).toLowerCase().includes(preferencePeriodSearch.toLowerCase()))
                      .map((period) => {
                        const key = `${period.academic_year}||${period.semester_type}`;
                        return (
                          <li
                            key={key}
                            className={`${adminOptionClass} ${selectedPreferencePeriod === key ? "bg-blue-50 font-medium text-primary" : ""}`}
                            onMouseDown={() => {
                              setSelectedPreferencePeriod(key);
                              setPreferencePeriodSearch("");
                              setPreferencePeriodOpen(false);
                            }}
                          >
                            {period.label || `${period.academic_year} - ${period.semester_type}`}
                          </li>
                        );
                      })}
                    {preferencePeriods.filter((period) => (period.label || `${period.academic_year} - ${period.semester_type}`).toLowerCase().includes(preferencePeriodSearch.toLowerCase())).length === 0 && preferencePeriodSearch !== "" && (
                      <li className="px-3 py-2 text-gray-400 italic">No periods match</li>
                    )}
                  </ul>
                )}
              </div>
              {loadingPreferencePeriods && <p className="text-xs text-gray-500">Loading periods...</p>}
            </div>

            <div className="space-y-1" ref={preferenceDeptRef}>
              <label className="block text-sm font-medium text-gray-700">Department</label>
              <div className="relative w-full">
                <input
                  type="text"
                  value={preferenceDeptSearch}
                  placeholder={
                    preferenceDept
                      ? (departments.find((d) => String(d.id) === preferenceDept)?.name || departments.find((d) => String(d.id) === preferenceDept)?.department_name || "Selected Department")
                      : (hodDepartmentId ? "Default HOD Department" : "All Departments")
                  }
                  onChange={(e) => {
                    setPreferenceDeptSearch(e.target.value);
                    setPreferenceDeptOpen(true);
                  }}
                  onFocus={() => setPreferenceDeptOpen(true)}
                  className={adminInputClass}
                />
                <button
                  type="button"
                  onClick={() => {
                    setPreferenceDept("");
                    setPreferenceDeptSearch("");
                    setPreferenceDeptOpen(false);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                  title="Clear"
                >
                  {preferenceDept ? "✕" : "▾"}
                </button>
                {preferenceDeptOpen && (
                  <ul className={adminDropdownClass}>
                    {(hodDepartmentId ? "default hod department" : "all departments").includes(preferenceDeptSearch.toLowerCase()) || preferenceDeptSearch === "" ? (
                      <li
                        className={`${adminOptionClass} ${preferenceDept === "" ? "bg-blue-50 font-medium text-primary" : ""}`}
                        onMouseDown={() => {
                          setPreferenceDept("");
                          setPreferenceDeptSearch("");
                          setPreferenceDeptOpen(false);
                        }}
                      >
                        {hodDepartmentId ? "Default HOD Department" : "All Departments"}
                      </li>
                    ) : null}
                    {departments
                      .filter((dept) => (dept.name || dept.department_name || "").toLowerCase().includes(preferenceDeptSearch.toLowerCase()))
                      .map((dept) => (
                        <li
                          key={dept.id}
                          className={`${adminOptionClass} ${preferenceDept === String(dept.id) ? "bg-blue-50 font-medium text-primary" : ""}`}
                          onMouseDown={() => {
                            setPreferenceDept(String(dept.id));
                            setPreferenceDeptSearch("");
                            setPreferenceDeptOpen(false);
                          }}
                        >
                          {dept.name || dept.department_name}
                        </li>
                      ))}
                    {departments.filter((dept) => (dept.name || dept.department_name || "").toLowerCase().includes(preferenceDeptSearch.toLowerCase())).length === 0 && preferenceDeptSearch !== "" && (
                      <li className="px-3 py-2 text-gray-400 italic">No departments match</li>
                    )}
                  </ul>
                )}
              </div>
            </div>

            <div className="space-y-1 md:col-span-2" ref={preferenceCourseRef}>
              <label className="block text-sm font-medium text-gray-700">Course</label>
              <div className="relative w-full max-w-md">
                <input
                  type="text"
                  value={preferenceCourseSearch}
                  placeholder={
                    selectedPreferenceCourse
                      ? (preferenceCourses.find((c) => c.course_code === selectedPreferenceCourse)?.label || "Selected Course")
                      : "All Courses"
                  }
                  onChange={(e) => {
                    setPreferenceCourseSearch(e.target.value);
                    setPreferenceCourseOpen(true);
                  }}
                  onFocus={() => setPreferenceCourseOpen(true)}
                  className={adminInputClass}
                />
                <button
                  type="button"
                  onClick={() => {
                    setSelectedPreferenceCourse("");
                    setPreferenceCourseSearch("");
                    setPreferenceCourseOpen(false);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                  title="Clear"
                >
                  {selectedPreferenceCourse ? "✕" : "▾"}
                </button>
                {preferenceCourseOpen && (
                  <ul className={adminDropdownClass}>
                    {("all courses").includes(preferenceCourseSearch.toLowerCase()) || preferenceCourseSearch === "" ? (
                      <li
                        className={`${adminOptionClass} ${selectedPreferenceCourse === "" ? "bg-blue-50 font-medium text-primary" : ""}`}
                        onMouseDown={() => {
                          setSelectedPreferenceCourse("");
                          setPreferenceCourseSearch("");
                          setPreferenceCourseOpen(false);
                        }}
                      >
                        All Courses
                      </li>
                    ) : null}
                    {preferenceCourses
                      .filter((course) => (course.label || `${course.course_code} - ${course.course_name}`).toLowerCase().includes(preferenceCourseSearch.toLowerCase()))
                      .map((course) => (
                        <li
                          key={course.course_code}
                          className={`${adminOptionClass} ${selectedPreferenceCourse === course.course_code ? "bg-blue-50 font-medium text-primary" : ""}`}
                          onMouseDown={() => {
                            setSelectedPreferenceCourse(course.course_code);
                            setPreferenceCourseSearch("");
                            setPreferenceCourseOpen(false);
                          }}
                        >
                          {course.label || `${course.course_code} - ${course.course_name}`}
                        </li>
                      ))}
                    {preferenceCourses.filter((course) => (course.label || `${course.course_code} - ${course.course_name}`).toLowerCase().includes(preferenceCourseSearch.toLowerCase())).length === 0 && preferenceCourseSearch !== "" && (
                      <li className="px-3 py-2 text-gray-400 italic">No courses match</li>
                    )}
                  </ul>
                )}
              </div>
            </div>

            <div className="space-y-1 md:col-span-2" ref={preferenceTeacherRef}>
              <label className="block text-sm font-medium text-gray-700">Teacher Name</label>
              <div className="relative w-full max-w-md">
                <input
                  type="text"
                  value={preferenceTeacherSearch}
                  placeholder={
                    selectedPreferenceTeacher
                      ? (preferenceTeachers.find((t) => String(t.teacher_id) === selectedPreferenceTeacher)?.label || "Selected Teacher")
                      : "All Teachers"
                  }
                  onChange={(e) => {
                    setPreferenceTeacherSearch(e.target.value);
                    setPreferenceTeacherOpen(true);
                  }}
                  onFocus={() => setPreferenceTeacherOpen(true)}
                  className={adminInputClass}
                />
                <button
                  type="button"
                  onClick={() => {
                    setSelectedPreferenceTeacher("");
                    setPreferenceTeacherSearch("");
                    setPreferenceTeacherOpen(false);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 text-xs"
                  title="Clear"
                >
                  {selectedPreferenceTeacher ? "✕" : "▾"}
                </button>
                {preferenceTeacherOpen && (
                  <ul className={adminDropdownClass}>
                    {("all teachers").includes(preferenceTeacherSearch.toLowerCase()) || preferenceTeacherSearch === "" ? (
                      <li
                        className={`${adminOptionClass} ${selectedPreferenceTeacher === "" ? "bg-blue-50 font-medium text-primary" : ""}`}
                        onMouseDown={() => {
                          setSelectedPreferenceTeacher("");
                          setPreferenceTeacherSearch("");
                          setPreferenceTeacherOpen(false);
                        }}
                      >
                        All Teachers
                      </li>
                    ) : null}
                    {preferenceTeachers
                      .filter((teacher) => (teacher.label || `${teacher.faculty_id} - ${teacher.teacher_name}`).toLowerCase().includes(preferenceTeacherSearch.toLowerCase()))
                      .map((teacher) => (
                        <li
                          key={teacher.teacher_id}
                          className={`${adminOptionClass} ${selectedPreferenceTeacher === String(teacher.teacher_id) ? "bg-blue-50 font-medium text-primary" : ""}`}
                          onMouseDown={() => {
                            setSelectedPreferenceTeacher(String(teacher.teacher_id));
                            setPreferenceTeacherSearch("");
                            setPreferenceTeacherOpen(false);
                          }}
                        >
                          {teacher.label || `${teacher.faculty_id} - ${teacher.teacher_name}`}
                        </li>
                      ))}
                    {preferenceTeachers.filter((teacher) => (teacher.label || `${teacher.faculty_id} - ${teacher.teacher_name}`).toLowerCase().includes(preferenceTeacherSearch.toLowerCase())).length === 0 && preferenceTeacherSearch !== "" && (
                      <li className="px-3 py-2 text-gray-400 italic">No teachers match</li>
                    )}
                  </ul>
                )}
              </div>
              {loadingPreferenceTeachers && <p className="text-xs text-gray-500">Loading teachers...</p>}
            </div>
          </div>

          <button
            onClick={handleExportTeacherPreferences}
            disabled={exportingTeacherPreferences}
            className={adminPrimaryBtnClass}
          >
            {exportingTeacherPreferences ? "Exporting..." : "Export Teacher Preference Data"}
          </button>
        </div>

        {/* Honour Eligibility Section */}
        <div className="bg-white border rounded-lg p-5 space-y-5">
          <h2 className="text-lg font-semibold text-gray-900">Honour Eligibility Management</h2>
          
          <div className="bg-white rounded-lg p-5 space-y-5">
            <div className="space-y-3">
              <h3 className="text-base font-semibold text-gray-900">1. Download Honour Template</h3>
              <p className="text-sm text-gray-600">
                Download the CSV template, fill the <span className="font-medium">student_email</span> column with students eligible for honour programs, and save as CSV.
              </p>
              <button
                onClick={handleDownloadHonourTemplate}
                className={`mt-1 ${adminPrimaryBtnClass}`}
              >
                Download Honour Template
              </button>
            </div>

            <div className="border-t pt-5 space-y-3">
              <h3 className="text-base font-semibold text-gray-900">2. Import Honour Data</h3>
              <p className="text-sm text-gray-600">
                Upload the completed CSV file to import honour eligible students into the system.
              </p>
              <input
                id="honour-import-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(e) => setHonourFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              <div className="mt-3 flex items-center gap-3 w-full min-w-0">
                <label
                  htmlFor="honour-import-file"
                  className={`${adminPrimaryBtnClass} inline-flex items-center whitespace-nowrap cursor-pointer`}
                >
                  Choose File
                </label>
                <span className="text-sm text-gray-600 min-w-0 flex-1 truncate">
                  {honourFile ? honourFile.name : "No file chosen"}
                </span>
              </div>
              <button
                onClick={handleImportHonourData}
                disabled={honourUploading}
                className={`mt-4 ${adminPrimaryBtnClass}`}
              >
                {honourUploading ? "Importing..." : "Import Honour Data"}
              </button>
            </div>
          </div>
        </div>

        {/* Minor Eligibility Section */}
        <div className="bg-white border rounded-lg p-5 space-y-5">
          <h2 className="text-lg font-semibold text-gray-900">Minor Eligibility Management</h2>
          
          <div className="bg-white rounded-lg p-5 space-y-5">
            <div className="space-y-3">
              <h3 className="text-base font-semibold text-gray-900">1. Download Minor Template</h3>
              <p className="text-sm text-gray-600">
                Download the CSV template, fill the <span className="font-medium">student_email</span> column with students eligible for minor programs, and save as CSV.
              </p>
              <button
                onClick={handleDownloadMinorTemplate}
                className={`mt-1 ${adminPrimaryBtnClass}`}
              >
                Download Minor Template
              </button>
            </div>

            <div className="border-t pt-5 space-y-3">
              <h3 className="text-base font-semibold text-gray-900">2. Import Minor Data</h3>
              <p className="text-sm text-gray-600">
                Upload the completed CSV file to import minor eligible students into the system.
              </p>
              <input
                id="minor-import-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(e) => setMinorFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              <div className="mt-3 flex items-center gap-3 w-full min-w-0">
                <label
                  htmlFor="minor-import-file"
                  className={`${adminPrimaryBtnClass} inline-flex items-center whitespace-nowrap cursor-pointer`}
                >
                  Choose File
                </label>
                <span className="text-sm text-gray-600 min-w-0 flex-1 truncate">
                  {minorFile ? minorFile.name : "No file chosen"}
                </span>
              </div>
              <button
                onClick={handleImportMinorData}
                disabled={minorUploading}
                className={`mt-4 ${adminPrimaryBtnClass}`}
              >
                {minorUploading ? "Importing..." : "Import Minor Data"}
              </button>
            </div>
          </div>
        </div>

        {/* Message Display */}
        {message && (
          <div
            className={`border rounded-lg px-4 py-3 text-sm ${
              messageType === "success"
                ? "bg-green-50 border-green-200 text-green-700"
                : "bg-red-50 border-red-200 text-red-700"
            }`}
          >
            {message}
          </div>
        )}
      </div>
    </MainLayout>
  );
};

export default HODHonourMinorEligibilityPage;
