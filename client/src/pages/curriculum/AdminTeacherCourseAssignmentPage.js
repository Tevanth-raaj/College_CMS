import React, { useEffect, useMemo, useState } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const AdminTeacherCourseAssignmentPage = () => {
  const [departments, setDepartments] = useState([]);
  const [deptFilter, setDeptFilter] = useState('');
  const [teacherSearch, setTeacherSearch] = useState('');

  const [teachers, setTeachers] = useState([]);
  const [loadingTeachers, setLoadingTeachers] = useState(false);
  const [selectedTeacher, setSelectedTeacher] = useState('');

  const [teacherInfo, setTeacherInfo] = useState(null);
  const [courses, setCourses] = useState([]);
  const [selectedCourseIds, setSelectedCourseIds] = useState(new Set());
  const [loadingContext, setLoadingContext] = useState(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchInit = async () => {
      setLoadingTeachers(true);
      try {
        const [teacherRes, deptRes] = await Promise.all([
          fetch(`${API_BASE_URL}/admin/teacher-course-assignment/teachers`),
          fetch(`${API_BASE_URL}/departments`),
        ]);
        if (!teacherRes.ok) throw new Error('Failed to load teachers');
        const teacherData = await teacherRes.json();
        setTeachers(Array.isArray(teacherData) ? teacherData : []);
        if (deptRes.ok) {
          const deptData = await deptRes.json();
          setDepartments(Array.isArray(deptData) ? deptData : []);
        }
      } catch (err) {
        setError(err.message || 'Failed to load teachers');
      } finally {
        setLoadingTeachers(false);
      }
    };

    fetchInit();
  }, []);

  useEffect(() => {
    const fetchContext = async () => {
      if (!selectedTeacher) {
        setTeacherInfo(null);
        setCourses([]);
        setSelectedCourseIds(new Set());
        return;
      }

      setLoadingContext(true);
      setError('');
      setMessage('');
      try {
        const res = await fetch(`${API_BASE_URL}/admin/teacher-course-assignment/${encodeURIComponent(selectedTeacher)}`);
        if (!res.ok) {
          const txt = await res.text();
          throw new Error(txt || 'Failed to load teacher course context');
        }

        const data = await res.json();
        const list = Array.isArray(data?.courses) ? data.courses : [];
        setTeacherInfo(data?.teacher || null);
        setCourses(list);
        setSelectedCourseIds(new Set(list.filter((c) => c.is_assigned).map((c) => c.course_id)));
      } catch (err) {
        setError(err.message || 'Failed to load context');
        setTeacherInfo(null);
        setCourses([]);
        setSelectedCourseIds(new Set());
      } finally {
        setLoadingContext(false);
      }
    };

    fetchContext();
  }, [selectedTeacher]);

  const filteredTeachers = useMemo(() => {
    const q = teacherSearch.trim().toLowerCase();
    return teachers.filter((t) => {
      const deptMatch = !deptFilter || String(t.department_id) === String(deptFilter);
      const nameMatch = !q || (t.teacher_name || '').toLowerCase().includes(q) || (t.faculty_id || '').toLowerCase().includes(q);
      return deptMatch && nameMatch;
    });
  }, [teachers, deptFilter, teacherSearch]);

  const grouped = useMemo(() => {
    return {
      core: courses.filter((course) => course.source === 'core'),
      extra: courses.filter((course) => course.source === 'extra'),
    };
  }, [courses]);

  const toggleCourse = (courseId) => {
    setSelectedCourseIds((prev) => {
      const next = new Set(prev);
      if (next.has(courseId)) {
        next.delete(courseId);
      } else {
        next.add(courseId);
      }
      return next;
    });
  };

  const handleSave = async () => {
    if (!selectedTeacher) return;
    setSaving(true);
    setError('');
    setMessage('');

    try {
      const res = await fetch(`${API_BASE_URL}/admin/teacher-course-assignment/${encodeURIComponent(selectedTeacher)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ course_ids: Array.from(selectedCourseIds) }),
      });

      const payload = await res.json().catch(() => ({}));
      if (!res.ok || payload.success === false) {
        throw new Error(payload.error || payload.message || 'Failed to update assignments');
      }

      setMessage(`Updated successfully. Added: ${payload.added_count || 0}, Removed: ${payload.removed_count || 0}`);

      const updatedRes = await fetch(`${API_BASE_URL}/admin/teacher-course-assignment/${encodeURIComponent(selectedTeacher)}`);
      if (updatedRes.ok) {
        const updatedData = await updatedRes.json();
        const updatedCourses = Array.isArray(updatedData?.courses) ? updatedData.courses : [];
        setTeacherInfo(updatedData?.teacher || null);
        setCourses(updatedCourses);
        setSelectedCourseIds(new Set(updatedCourses.filter((c) => c.is_assigned).map((c) => c.course_id)));
      }
    } catch (err) {
      setError(err.message || 'Failed to update assignments');
    } finally {
      setSaving(false);
    }
  };

  const renderCourseTable = (title, list, emptyText) => (
    <div className="bg-white border rounded-lg">
      <div className="px-4 py-3 border-b bg-gray-50">
        <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
      </div>
      {list.length === 0 ? (
        <div className="px-4 py-4 text-sm text-gray-500">{emptyText}</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-xs uppercase text-gray-600">
              <tr>
                <th className="px-3 py-2 text-left">Assign</th>
                <th className="px-3 py-2 text-left">Course Code</th>
                <th className="px-3 py-2 text-left">Course Name</th>
                <th className="px-3 py-2 text-left">Category</th>
                <th className="px-3 py-2 text-left">Type</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {list.map((course) => (
                <tr key={course.course_id}>
                  <td className="px-3 py-2">
                    <input
                      type="checkbox"
                      checked={selectedCourseIds.has(course.course_id)}
                      onChange={() => toggleCourse(course.course_id)}
                    />
                  </td>
                  <td className="px-3 py-2 font-medium text-gray-900">{course.course_code}</td>
                  <td className="px-3 py-2 text-gray-700">{course.course_name}</td>
                  <td className="px-3 py-2 text-gray-700">{course.category || '-'}</td>
                  <td className="px-3 py-2 text-gray-700">{course.course_type || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );

  return (
    <MainLayout
      title="Admin Teacher Course Assignment"
      subtitle="Select a teacher and manually add / drop courses within department student scope"
    >
      <div className="space-y-4">
        {error && <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm">{error}</div>}
        {message && <div className="bg-green-50 border border-green-200 text-green-700 rounded-lg p-3 text-sm">{message}</div>}

        <div className="bg-white border rounded-lg p-4 space-y-3">
          <div className="flex flex-wrap gap-3 items-end">
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-gray-600">Department</label>
              <select
                value={deptFilter}
                onChange={(e) => { setDeptFilter(e.target.value); setSelectedTeacher(''); }}
                className="border border-gray-300 rounded-lg px-3 py-2 text-sm w-56"
              >
                <option value="">All Departments</option>
                {departments.map((dept) => (
                  <option key={dept.id} value={dept.id}>
                    {dept.name || dept.department_name}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-gray-600">Search Teacher</label>
              <input
                type="text"
                placeholder="Name or Faculty ID"
                value={teacherSearch}
                onChange={(e) => setTeacherSearch(e.target.value)}
                className="border border-gray-300 rounded-lg px-3 py-2 text-sm w-52"
              />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs font-medium text-gray-600">Teacher ({filteredTeachers.length})</label>
              <select
                value={selectedTeacher}
                onChange={(e) => setSelectedTeacher(e.target.value)}
                className="border border-gray-300 rounded-lg px-3 py-2 text-sm w-72"
                disabled={loadingTeachers}
              >
                <option value="">Select Teacher</option>
                {filteredTeachers.map((teacher) => (
                  <option key={teacher.teacher_id} value={teacher.teacher_id}>
                    {teacher.label || `${teacher.faculty_id} - ${teacher.teacher_name}`}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {teacherInfo && (
            <div className="text-sm text-gray-700">
              <span className="font-medium">Department:</span> {teacherInfo.department_name || '-'}
              <span className="mx-2 text-gray-400">•</span>
              <span className="font-medium">Faculty ID:</span> {teacherInfo.faculty_id}
              <span className="mx-2 text-gray-400">•</span>
              <span className="font-medium">Assigned in scope:</span> {Array.from(selectedCourseIds).length}
            </div>
          )}

          <p className="text-xs text-gray-500">
            Core courses are shown by default from department curriculum. Extra courses are shown only when students in this department have selected them (Open/Professional Elective/Honour/Minor/Add-on).
          </p>
        </div>

        {loadingContext && (
          <div className="bg-white border rounded-lg p-4 text-sm text-gray-500">Loading teacher courses...</div>
        )}

        {!loadingContext && teacherInfo && (
          <>
            {renderCourseTable('Core Courses (Default)', grouped.core, 'No core courses found for this department.')}
            {renderCourseTable('Extra Courses (From Department Student Selections)', grouped.extra, 'No extra courses currently selected by department students.')}

            <div className="flex justify-end">
              <button
                onClick={handleSave}
                disabled={saving}
                className="bg-primary text-white px-4 py-2 rounded-lg text-sm disabled:opacity-60 disabled:cursor-not-allowed"
              >
                {saving ? 'Saving...' : 'Save Assignments'}
              </button>
            </div>
          </>
        )}
      </div>
    </MainLayout>
  );
};

export default AdminTeacherCourseAssignmentPage;
