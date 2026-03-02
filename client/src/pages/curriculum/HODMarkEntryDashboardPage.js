import React, { useEffect, useMemo, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function HODMarkEntryDashboardPage() {
  const username = localStorage.getItem('username')
  const userRole = localStorage.getItem('userRole')

  const [semester, setSemester] = useState('4')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [overview, setOverview] = useState([])
  const [summary, setSummary] = useState(null)
  const [selectedRow, setSelectedRow] = useState(null)
  const [students, setStudents] = useState([])
  const [studentsLoading, setStudentsLoading] = useState(false)

  const canAccess = userRole === 'hod' || userRole === 'admin' || userRole === 'coe'

  const fetchOverview = async () => {
    if (!username || !semester) return
    setLoading(true)
    setError('')
    try {
      const response = await fetch(`${API_BASE_URL}/hod/mark-entry/overview?username=${encodeURIComponent(username)}&semester=${semester}`)
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || 'Failed to load overview')
      }
      setOverview(Array.isArray(data.rows) ? data.rows : [])
      setSummary(data.summary || null)
    } catch (err) {
      setError(err.message)
      setOverview([])
      setSummary(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (canAccess) {
      fetchOverview()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [semester, canAccess])

  const loadStudents = async (row) => {
    setSelectedRow(row)
    setStudents([])
    setStudentsLoading(true)
    try {
      const response = await fetch(`${API_BASE_URL}/hod/mark-entry/teacher-students?teacher_id=${encodeURIComponent(row.teacher_id)}&course_id=${row.course_id}`)
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || 'Failed to fetch students')
      }
      setStudents(Array.isArray(data.students) ? data.students : [])
    } catch (err) {
      setStudents([])
      setError(err.message)
    } finally {
      setStudentsLoading(false)
    }
  }

  const analytics = useMemo(() => {
    if (!summary) return { total: 0, completed: 0, late: 0, requested: 0, notRequested: 0 }
    return {
      total: summary.total_assignments || 0,
      completed: summary.completed || 0,
      late: summary.late || 0,
      requested: summary.requested || 0,
      notRequested: summary.not_requested || 0,
    }
  }, [summary])

  const downloadReport = async (reportType, format) => {
    try {
      const url = `${API_BASE_URL}/hod/mark-entry/download?username=${encodeURIComponent(username)}&semester=${semester}&report_type=${reportType}&format=${format}`
      const response = await fetch(url)
      if (!response.ok) {
        throw new Error('Failed to download report')
      }
      const blob = await response.blob()
      const link = document.createElement('a')
      const objectUrl = URL.createObjectURL(blob)
      link.href = objectUrl
      link.download = `mark_entry_${reportType}_sem${semester}.${format}`
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(objectUrl)
    } catch (err) {
      setError(err.message)
    }
  }

  if (!canAccess) {
    return (
      <MainLayout title="Mark Entry Monitor" subtitle="Department mark entry monitoring">
        <div className="bg-white border rounded-lg p-6 text-red-600">Access denied.</div>
      </MainLayout>
    )
  }

  return (
    <MainLayout
      title="Mark Entry Monitor"
      subtitle="Track course-wise completion, late entries, and extension demand"
      actions={
        <div className="flex items-center gap-2">
          <select
            value={semester}
            onChange={(event) => setSemester(event.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm"
          >
            {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
              <option key={value} value={value}>Semester {value}</option>
            ))}
          </select>
          <button onClick={fetchOverview} className="bg-primary text-white px-3 py-2 rounded-lg text-sm">
            Refresh
          </button>
        </div>
      }
    >
      <div className="space-y-4">
        {error && <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm">{error}</div>}

        <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
          <MetricCard title="Total" value={analytics.total} />
          <MetricCard title="Completed" value={analytics.completed} />
          <MetricCard title="Late" value={analytics.late} />
          <MetricCard title="Requested" value={analytics.requested} />
          <MetricCard title="Not Requested" value={analytics.notRequested} />
        </div>

        <div className="bg-white border rounded-lg p-4">
          <h3 className="text-sm font-semibold text-gray-800 mb-3">Downloads</h3>
          <div className="flex flex-wrap gap-2">
            <button onClick={() => downloadReport('student', 'csv')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Student CSV</button>
            <button onClick={() => downloadReport('student', 'xlsx')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Student XLSX</button>
            <button onClick={() => downloadReport('student', 'pdf')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Student PDF</button>
            <button onClick={() => downloadReport('course', 'csv')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Course CSV</button>
            <button onClick={() => downloadReport('course', 'xlsx')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Course XLSX</button>
            <button onClick={() => downloadReport('course', 'pdf')} className="px-3 py-2 bg-gray-100 rounded-lg text-sm">Course PDF</button>
          </div>
        </div>

        <div className="bg-white border rounded-lg overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="text-left px-3 py-2">Course</th>
                  <th className="text-left px-3 py-2">Teacher</th>
                  <th className="text-left px-3 py-2">Window</th>
                  <th className="text-right px-3 py-2">Assigned</th>
                  <th className="text-right px-3 py-2">Completed</th>
                  <th className="text-right px-3 py-2">Completion %</th>
                  <th className="text-left px-3 py-2">Extension</th>
                  <th className="text-left px-3 py-2">Students</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">Loading...</td></tr>
                ) : overview.length === 0 ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">No rows found.</td></tr>
                ) : overview.map((row) => (
                  <tr key={`${row.course_id}-${row.teacher_id}`} className="border-t">
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{row.course_code}</div>
                      <div className="text-gray-500">{row.course_name}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{row.teacher_name}</div>
                      <div className="text-gray-500">{row.teacher_id}</div>
                    </td>
                    <td className="px-3 py-2">
                      <span className={`px-2 py-1 rounded text-xs ${row.window_status === 'open' ? 'bg-green-100 text-green-700' : row.window_status === 'closed' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>
                        {row.window_status}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-right">{row.assigned_students}</td>
                    <td className="px-3 py-2 text-right">{row.completed_students}</td>
                    <td className="px-3 py-2 text-right">{(row.completion_percent || 0).toFixed(2)}</td>
                    <td className="px-3 py-2">{row.extension_status || 'none'}</td>
                    <td className="px-3 py-2">
                      <button onClick={() => loadStudents(row)} className="text-primary hover:underline">View</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {selectedRow && (
          <div className="bg-white border rounded-lg p-4">
            <h3 className="text-sm font-semibold text-gray-900 mb-3">
              Students entered by {selectedRow.teacher_name} - {selectedRow.course_code}
            </h3>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="text-left px-3 py-2">Enrollment No</th>
                    <th className="text-left px-3 py-2">Student Name</th>
                    <th className="text-right px-3 py-2">Total Marks</th>
                    <th className="text-left px-3 py-2">Entered</th>
                  </tr>
                </thead>
                <tbody>
                  {studentsLoading ? (
                    <tr><td colSpan="4" className="px-3 py-4 text-center text-gray-500">Loading...</td></tr>
                  ) : students.length === 0 ? (
                    <tr><td colSpan="4" className="px-3 py-4 text-center text-gray-500">No students found.</td></tr>
                  ) : students.map((item) => (
                    <tr key={item.student_id} className="border-t">
                      <td className="px-3 py-2">{item.enrollment_no}</td>
                      <td className="px-3 py-2">{item.student_name}</td>
                      <td className="px-3 py-2 text-right">{Number(item.total_marks || 0).toFixed(2)}</td>
                      <td className="px-3 py-2">{item.has_mark_entry ? 'Yes' : 'No'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
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