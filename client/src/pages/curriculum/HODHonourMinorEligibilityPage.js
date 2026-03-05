import React, { useState, useEffect } from "react";
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

  // Window selection state
  const [windows, setWindows] = useState([]);
  const [selectedWindow, setSelectedWindow] = useState(null);
  const [loadingWindows, setLoadingWindows] = useState(false);

  // Fetch allocation windows on mount
  useEffect(() => {
    const fetchWindows = async () => {
      setLoadingWindows(true);
      try {
        const res = await fetch(`${API_BASE_URL}/hod/teacher-limits/windows`);
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
  }, []);

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
      let url = `${API_BASE_URL}/hod/teacher-limits/export`;
      if (selectedWindow) {
        const params = new URLSearchParams({
          window_start: selectedWindow.window_start,
          window_end: selectedWindow.window_end,
        });
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
      a.download = `teacher_limits_${windowLabel}.xlsx`;
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

  return (
    <MainLayout
      title="Honour/Minor Eligibility Management"
      subtitle="Download templates and import student eligibility data for honour and minor programs"
    >
      <div className="max-w-4xl space-y-6">
        {/* Teacher Export Section */}
        <div className="bg-white border border-gray-200 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-gray-900">📊 Export Teacher Data</h2>
          <p className="text-sm text-gray-600">
            Export teacher workload and course allocation data including assigned limits, subjects, and labs.
          </p>

          {/* Window selector */}
          <div className="space-y-1">
            <label className="block text-sm font-medium text-gray-700">
              Allocation Window
            </label>
            {loadingWindows ? (
              <p className="text-sm text-gray-500">Loading windows…</p>
            ) : windows.length === 0 ? (
              <p className="text-sm text-gray-400 italic">No allocation windows found — exporting all records.</p>
            ) : (
              <select
                value={selectedWindow ? `${selectedWindow.window_start}||${selectedWindow.window_end}` : ""}
                onChange={(e) => {
                  const val = e.target.value;
                  if (!val) {
                    setSelectedWindow(null);
                    return;
                  }
                  const found = windows.find(
                    (w) => `${w.window_start}||${w.window_end}` === val
                  );
                  setSelectedWindow(found || null);
                }}
                className="w-full max-w-md border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
              >
                {windows.map((w) => (
                  <option key={`${w.window_start}||${w.window_end}`} value={`${w.window_start}||${w.window_end}`}>
                    {w.label}
                  </option>
                ))}
              </select>
            )}
          </div>

          <button
            onClick={handleExportTeacherLimits}
            disabled={exportingTeachers}
            className={`px-4 py-2 rounded-lg text-white transition ${
              exportingTeachers ? "bg-gray-400 cursor-not-allowed" : "bg-primary hover:opacity-90"
            }`}
          >
            {exportingTeachers ? "Exporting..." : "Export Teacher Data"}
          </button>
        </div>

        {/* Honour Eligibility Section */}
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-6 space-y-4">
          <h2 className="text-xl font-semibold text-blue-900">🏆 Honour Eligibility Management</h2>
          
          <div className="bg-white rounded-lg p-4 space-y-4">
            <div>
              <h3 className="text-base font-semibold text-gray-900">1. Download Honour Template</h3>
              <p className="text-sm text-gray-600">
                Download the CSV template, fill the <span className="font-medium">student_email</span> column with students eligible for honour programs, and save as CSV.
              </p>
              <button
                onClick={handleDownloadHonourTemplate}
                className="mt-3 px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition"
              >
                Download Honour Template
              </button>
            </div>

            <div className="border-t pt-4">
              <h3 className="text-base font-semibold text-gray-900">2. Import Honour Data</h3>
              <p className="text-sm text-gray-600">
                Upload the completed CSV file to import honour eligible students into the system.
              </p>
              <input
                id="honour-import-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(e) => setHonourFile(e.target.files?.[0] || null)}
                className="mt-3 block w-full text-sm text-gray-700 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:bg-blue-100 file:text-blue-700 hover:file:bg-blue-200"
              />
              <button
                onClick={handleImportHonourData}
                disabled={honourUploading}
                className={`mt-3 px-4 py-2 rounded-lg text-white transition ${
                  honourUploading ? "bg-gray-400 cursor-not-allowed" : "bg-blue-600 hover:bg-blue-700"
                }`}
              >
                {honourUploading ? "Importing..." : "Import Honour Data"}
              </button>
            </div>
          </div>
        </div>

        {/* Minor Eligibility Section */}
        <div className="bg-purple-50 border border-purple-200 rounded-xl p-6 space-y-4">
          <h2 className="text-xl font-semibold text-purple-900">📚 Minor Eligibility Management</h2>
          
          <div className="bg-white rounded-lg p-4 space-y-4">
            <div>
              <h3 className="text-base font-semibold text-gray-900">1. Download Minor Template</h3>
              <p className="text-sm text-gray-600">
                Download the CSV template, fill the <span className="font-medium">student_email</span> column with students eligible for minor programs, and save as CSV.
              </p>
              <button
                onClick={handleDownloadMinorTemplate}
                className="mt-3 px-4 py-2 rounded-lg bg-purple-600 text-white hover:bg-purple-700 transition"
              >
                Download Minor Template
              </button>
            </div>

            <div className="border-t pt-4">
              <h3 className="text-base font-semibold text-gray-900">2. Import Minor Data</h3>
              <p className="text-sm text-gray-600">
                Upload the completed CSV file to import minor eligible students into the system.
              </p>
              <input
                id="minor-import-file"
                type="file"
                accept=".csv,text/csv"
                onChange={(e) => setMinorFile(e.target.files?.[0] || null)}
                className="mt-3 block w-full text-sm text-gray-700 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:bg-purple-100 file:text-purple-700 hover:file:bg-purple-200"
              />
              <button
                onClick={handleImportMinorData}
                disabled={minorUploading}
                className={`mt-3 px-4 py-2 rounded-lg text-white transition ${
                  minorUploading ? "bg-gray-400 cursor-not-allowed" : "bg-purple-600 hover:bg-purple-700"
                }`}
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
