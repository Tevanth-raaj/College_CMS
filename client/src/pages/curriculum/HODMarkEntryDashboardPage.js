import React, { useEffect, useMemo, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function HODMarkEntryDashboardPage() {
  const username = localStorage.getItem('username')
  const userRole = (localStorage.getItem('userRole') || '').toLowerCase()
  const isAdminMode = userRole === 'admin' || userRole === 'coe'
  const canAccess = isAdminMode || userRole === 'hod'

  const [semester, setSemester] = useState('4')
  const [windowId, setWindowId] = useState('')
  const [departmentId, setDepartmentId] = useState('')
  const [departments, setDepartments] = useState([])
  const [windows, setWindows] = useState([])

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [summary, setSummary] = useState(null)
  const [rows, setRows] = useState([])
  const [downloadingType, setDownloadingType] = useState('')

  const [selectedRow, setSelectedRow] = useState(null)
  const [students, setStudents] = useState([])
  const [studentsLoading, setStudentsLoading] = useState(false)
  const [componentModal, setComponentModal] = useState({ open: false, student: null })

  const parseResponseBody = async (response) => {
    const rawText = await response.text()
    if (!rawText) return {}
    try {
      return JSON.parse(rawText)
    } catch {
      return { message: rawText }
    }
  }

  useEffect(() => {
    if (!canAccess) return
    fetchWindows()
    if (isAdminMode) fetchDepartments()
  }, [canAccess, isAdminMode])

  useEffect(() => {
    if (!canAccess) return
    fetchMonitor()
  }, [semester, windowId, departmentId, canAccess])

  const fetchDepartments = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/departments`)
      const data = await response.json()
      const rawList = Array.isArray(data) ? data : (data.departments || [])
      const list = rawList.map((department) => ({
        id: department.id || department.department_id,
        label: department.name || department.department_name || department.department || 'Unknown Department',
      }))
      setDepartments(list)
    } catch (err) {
      setDepartments([])
    }
  }

  const fetchWindows = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/mark-entry-windows`)
      if (!response.ok) throw new Error('Failed to fetch windows')
      const data = await parseResponseBody(response)
      setWindows(Array.isArray(data) ? data : [])
    } catch (err) {
      setWindows([])
    }
  }

  const fetchMonitor = async () => {
    setLoading(true)
    setError('')
    try {
      const params = new URLSearchParams()
      params.append('semester', semester)
      if (windowId) params.append('window_id', windowId)

      let endpoint = `${API_BASE_URL}/hod/mark-entry/window-monitor`
      if (isAdminMode) {
        endpoint = `${API_BASE_URL}/admin/mark-entry/window-monitor`
        if (departmentId) params.append('department_id', departmentId)
      } else {
        params.append('username', username || '')
      }

      const response = await fetch(`${endpoint}?${params.toString()}`)
      const data = await parseResponseBody(response)
      if (!response.ok) throw new Error(data.error || data.message || 'Failed to load monitor')
      setSummary(data.summary || null)
      setRows(Array.isArray(data.rows) ? data.rows : [])
      setSelectedRow(null)
      setStudents([])
    } catch (err) {
      setSummary(null)
      setRows([])
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const loadStudents = async (row) => {
    setSelectedRow(row)
    setStudents([])
    setStudentsLoading(true)
    try {
      const params = new URLSearchParams()
      params.append('teacher_id', row.teacher_id)
      params.append('course_id', row.course_id)
      if (row.window_id) params.append('window_id', row.window_id)
      const response = await fetch(`${API_BASE_URL}/hod/mark-entry/teacher-students?${params.toString()}`)
      const data = await parseResponseBody(response)
      if (!response.ok) throw new Error(data.error || data.message || 'Failed to fetch students')
      setStudents(Array.isArray(data.students) ? data.students : [])
    } catch (err) {
      setStudents([])
      setError(err.message)
    } finally {
      setStudentsLoading(false)
    }
  }

  const downloadReport = async (reportType) => {
    if (!username) {
      setError('Username not found. Please login again.')
      return
    }

    setDownloadingType(reportType)
    setError('')
    try {
      const params = new URLSearchParams()
      params.append('username', username)
      params.append('semester', semester)
      params.append('report_type', reportType)
      params.append('format', 'xlsx')

      const response = await fetch(`${API_BASE_URL}/hod/mark-entry/download?${params.toString()}`)
      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'Failed to download report')
      }

      const blob = await response.blob()
      const objectURL = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectURL
      anchor.download = `mark_entry_${reportType}_sem${semester}.xlsx`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(objectURL)
    } catch (err) {
      setError(err.message || 'Failed to download report')
    } finally {
      setDownloadingType('')
    }
  }

  const windowOptions = useMemo(() => {
    return windows
      .filter((w) => !w.semester || String(w.semester) === String(semester))
      .sort((a, b) => (b.id || 0) - (a.id || 0))
  }, [windows, semester])

  if (!canAccess) {
    return (
      <MainLayout title="Mark Entry Monitor" subtitle="Window-based mark entry tracking">
        <div className="bg-white border rounded-lg p-6 text-red-600">Access denied.</div>
      </MainLayout>
    )
  }

  return (
    <MainLayout
      title="Mark Entry Monitor"
      subtitle={isAdminMode ? 'All departments / all windows progress' : 'Department window progress'}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <select value={semester} onChange={(e) => setSemester(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm">
            {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
              <option key={value} value={value}>Semester {value}</option>
            ))}
          </select>

          <select value={windowId} onChange={(e) => setWindowId(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm min-w-[220px]">
            <option value="">All matching windows</option>
            {windowOptions.map((w) => (
              <option key={w.id} value={w.id}>Window #{w.id} ({w.start_at?.slice(0, 16)} - {w.end_at?.slice(0, 16)})</option>
            ))}
          </select>

          {isAdminMode && (
            <select value={departmentId} onChange={(e) => setDepartmentId(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm min-w-[180px]">
              <option value="">All departments</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>{d.label}</option>
              ))}
            </select>
          )}

          <button onClick={fetchMonitor} className="bg-primary text-white px-3 py-2 rounded-lg text-sm">Refresh</button>
          {!isAdminMode && (
            <>
              <button
                onClick={() => downloadReport('teacher')}
                disabled={downloadingType !== ''}
                className="border border-gray-300 px-3 py-2 rounded-lg text-sm disabled:opacity-60"
              >
                {downloadingType === 'teacher' ? 'Downloading...' : 'Teacher-wise XLSX'}
              </button>
              <button
                onClick={() => downloadReport('course')}
                disabled={downloadingType !== ''}
                className="border border-gray-300 px-3 py-2 rounded-lg text-sm disabled:opacity-60"
              >
                {downloadingType === 'course' ? 'Downloading...' : 'Course-wise XLSX'}
              </button>
            </>
          )}
        </div>
      }
    >
      <div className="space-y-4">
        {error && <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm">{error}</div>}

        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          <MetricCard title="Assignments" value={summary?.total_assignments || 0} />
          <MetricCard title="Completed" value={summary?.completed_assignments || 0} />
          <MetricCard title="Pending" value={summary?.pending_assignments || 0} />
          <MetricCard title="Students" value={summary?.total_students || 0} />
          <MetricCard title="Done Students" value={summary?.completed_students || 0} />
        </div>

        <div className="bg-white border rounded-lg overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="text-left px-3 py-2">Department</th>
                  <th className="text-left px-3 py-2">Course</th>
                  <th className="text-left px-3 py-2">Teacher</th>
                  <th className="text-left px-3 py-2">Window</th>
                  <th className="text-right px-3 py-2">Assigned</th>
                  <th className="text-right px-3 py-2">Completed</th>
                  <th className="text-right px-3 py-2">Completion %</th>
                  <th className="text-left px-3 py-2">Drill-down</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">Loading...</td></tr>
                ) : rows.length === 0 ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">No monitor rows found.</td></tr>
                ) : rows.map((row) => (
                  <tr key={`${row.course_id}-${row.teacher_id}-${row.department_id}`} className="border-t hover:bg-gray-50">
                    <td className="px-3 py-2">{row.department_name || '-'}</td>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{row.course_code}</div>
                      <div className="text-gray-500">{row.course_name}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{row.teacher_name}</div>
                      <div className="text-gray-500">{row.teacher_id}</div>
                    </td>
                    <td className="px-3 py-2">{row.window_status || '-'}</td>
                    <td className="px-3 py-2 text-right">{row.assigned_students || 0}</td>
                    <td className="px-3 py-2 text-right">{row.completed_students || 0}</td>
                    <td className="px-3 py-2 text-right">{Number(row.completion_percent || 0).toFixed(2)}</td>
                    <td className="px-3 py-2">
                      <button onClick={() => loadStudents(row)} className="text-primary hover:underline">View students</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {selectedRow && (
          <div className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-40 p-4" onClick={() => setSelectedRow(null)}>
            <div className="bg-white rounded-lg w-full max-w-6xl p-4" onClick={(e) => e.stopPropagation()}>
              <div className="flex justify-between items-center mb-3">
                <h3 className="text-sm font-semibold text-gray-900">
                  {selectedRow.teacher_name} • {selectedRow.course_code}
                </h3>
                <button onClick={() => setSelectedRow(null)} className="text-gray-500 hover:text-gray-700">Close</button>
              </div>

              <div className="max-h-[70vh] overflow-auto border rounded-lg">
                <table className="min-w-full text-sm">
                  <thead className="bg-gray-50 sticky top-0">
                    <tr>
                      <th className="text-left px-3 py-2">Enrollment</th>
                      <th className="text-left px-3 py-2">Student</th>
                      <th className="text-right px-3 py-2">Total Marks</th>
                      <th className="text-left px-3 py-2">Entered</th>
                      <th className="text-left px-3 py-2">Components</th>
                    </tr>
                  </thead>
                  <tbody>
                    {studentsLoading ? (
                      <tr><td colSpan="5" className="px-3 py-4 text-center text-gray-500">Loading...</td></tr>
                    ) : students.length === 0 ? (
                      <tr><td colSpan="5" className="px-3 py-4 text-center text-gray-500">No students found.</td></tr>
                    ) : students.map((student) => {
                      const components = Array.isArray(student.components) ? student.components : []
                      const showInline = components.length > 0 && components.length <= 3
                      return (
                        <tr key={student.student_id} className="border-t">
                          <td className="px-3 py-2">{student.enrollment_no || '-'}</td>
                          <td className="px-3 py-2">{student.student_name}</td>
                          <td className="px-3 py-2 text-right">{Number(student.total_marks || 0).toFixed(2)}</td>
                          <td className="px-3 py-2">{student.has_mark_entry ? 'Yes' : 'No'}</td>
                          <td className="px-3 py-2">
                            {showInline ? (
                              <div className="flex flex-wrap gap-1">
                                {components.map((c) => (
                                  <span key={`${student.student_id}-${c.assessment_component_id}`} className="px-2 py-1 text-xs rounded bg-gray-100 text-gray-700">
                                    {c.assessment_component_name}: {Number(c.obtained_marks || 0).toFixed(2)}
                                  </span>
                                ))}
                              </div>
                            ) : components.length > 0 ? (
                              <button onClick={() => setComponentModal({ open: true, student })} className="text-primary hover:underline">View all ({components.length})</button>
                            ) : (
                              '-'
                            )}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {componentModal.open && componentModal.student && (
          <div className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-50 p-4" onClick={() => setComponentModal({ open: false, student: null })}>
            <div className="bg-white rounded-lg w-full max-w-2xl p-4" onClick={(e) => e.stopPropagation()}>
              <div className="flex justify-between items-center mb-3">
                <h4 className="font-semibold text-gray-900">{componentModal.student.student_name} - Components</h4>
                <button onClick={() => setComponentModal({ open: false, student: null })} className="text-gray-500 hover:text-gray-700">Close</button>
              </div>
              <div className="max-h-[420px] overflow-auto border rounded-lg">
                <table className="min-w-full text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="text-left px-3 py-2">Component</th>
                      <th className="text-right px-3 py-2">Marks</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(componentModal.student.components || []).map((c) => (
                      <tr key={`modal-${componentModal.student.student_id}-${c.assessment_component_id}`} className="border-t">
                        <td className="px-3 py-2">{c.assessment_component_name}</td>
                        <td className="px-3 py-2 text-right">{Number(c.obtained_marks || 0).toFixed(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  )
}

function MetricCard({ title, value }) {
  return (
    <div className="bg-white border rounded-lg p-4">
      <div className="text-xs text-gray-500 uppercase tracking-wide">{title}</div>
      <div className="text-2xl font-bold text-gray-900 mt-1">{value}</div>
    </div>
  )
}

export default HODMarkEntryDashboardPage
