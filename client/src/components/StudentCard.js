import React, { useState } from "react";
import { API_BASE_URL } from "../config";

function StudentCard({ student, onEdit, onDelete }) {
  const [showDetails, setShowDetails] = useState(false);
  
  // Get base URL without /api for static files
  const baseUrl = API_BASE_URL.replace("/api", "");
  const imageUrl = student.profile_img
    ? `${baseUrl}${student.profile_img}`
    : null;

  return (
    <div className="relative p-6 border border-primary-200 rounded-lg bg-white card-custom hover:border-primary-300 transition-colors">
      {/* Avatar and Name */}
      <div className="flex items-center space-x-3 mb-4">
        {imageUrl ? (
          <img
            src={imageUrl}
            alt={student.student_name}
            className="w-16 h-16 rounded-full object-cover shadow-md flex-shrink-0 border-2 border-primary-200"
            onError={(e) => {
              e.target.style.display = "none";
              e.target.nextElementSibling.style.display = "flex";
            }}
          />
        ) : null}
        <div
          className="w-16 h-16 rounded-full bg-background flex items-center justify-center text-primary font-semibold text-2xl shadow-md flex-shrink-0 border-2 border-primary"
          style={{ display: imageUrl ? "none" : "flex" }}
        >
          {student.student_name ? student.student_name.charAt(0).toUpperCase() : "S"}
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="text-xl font-bold text-gray-900 truncate">
            {student.student_name || "—"}
          </h3>
          <p className="text-sm text-primary-700 font-medium mt-1" title={student.register_no || ""}>
            {student.register_no || ""}
          </p>
        </div>
      </div>

      {/* Detailed Information */}
      <div className="space-y-3 mb-4">
        <div className="flex items-start">
          <svg className="w-5 h-5 text-primary-400 mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l4.414 4.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
          </svg>
          <div className="flex-1 min-w-0">
            <p className="text-xs text-gray-500 uppercase font-bold">Register Number</p>
            <p className="text-sm text-gray-900 truncate" title={student.register_no}>
              {student.register_no || "-"}
            </p>
          </div>
        </div>

        <div className="flex items-start">
          <svg className="w-5 h-5 text-primary-400 mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5V4H2v16h5m10 0v-8H7v8m10 0H7"/>
          </svg>
          <div className="flex-1">
            <p className="text-xs text-gray-500 uppercase font-bold">Learning Mode</p>
            <p className="text-sm text-gray-900">{student.learning_mode || "-"}</p>
          </div>
        </div>

        <div className="flex items-start">
          <svg className="w-5 h-5 text-primary-400 mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
          </svg>
          <div className="flex-1">
            <p className="text-xs text-gray-500 uppercase font-bold">Year</p>
            <p className="text-sm text-gray-900">{student.year || "-"}</p>
          </div>
        </div>

        <div className="flex items-start">
          <svg className="w-5 h-5 text-primary-400 mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7l9-4 9 4-9 4-9-4zm0 10l9 4 9-4M3 12l9 4 9-4"/>
          </svg>
          <div className="flex-1">
            <p className="text-xs text-gray-500 uppercase font-bold">Department Code</p>
            <p className="text-sm text-gray-900">{student.department_code || "-"}</p>
          </div>
        </div>

        <div className="flex items-start">
          <svg className="w-5 h-5 text-primary-400 mr-2 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 12H8m8 0a4 4 0 11-8 0m8 0a4 4 0 10-8 0m8 0v1a2 2 0 01-2 2H10a2 2 0 01-2-2v-1"/>
          </svg>
          <div className="flex-1 min-w-0">
            <p className="text-xs text-gray-500 uppercase font-bold">Mail ID</p>
            <p className="text-sm text-gray-900 truncate" title={student.mail_id}>
              {student.mail_id || "-"}
            </p>
          </div>
        </div>
      </div>

      {/* Status Badge */}
      <div className="mb-4">
        {student.status === 1 && (
          <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold bg-background text-primary">
            <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"/>
            </svg>
            Active
          </span>
        )}
      </div>

      {/* Action Buttons */}
      <div className="flex gap-2 pt-4 border-t border-primary-100">
        <button type="button" onClick={() => onEdit(student.student_id || student.id)}
          className="flex-1 px-4 py-2.5 text-sm font-medium bg-primary-50 text-primary-700 border border-primary-200 rounded-lg hover:bg-primary-100 hover:border-primary-300 transition-colors flex items-center justify-center"
        >
          <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
          </svg>
          Edit
        </button>

        {onDelete && (
          <button type="button" onClick={() => onDelete(student.student_id || student.id)}
            className="flex-1 px-4 py-2.5 text-sm font-bold bg-primary text-white border border-primary rounded-lg
                    hover:bg-primary-600 hover:border-primary-600 transition-all duration-200
                    flex items-center justify-center"
          >
            <svg className="w-4 h-4 mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
            Delete
          </button>
        )}
      </div>
    </div>
  );
}

export default StudentCard;