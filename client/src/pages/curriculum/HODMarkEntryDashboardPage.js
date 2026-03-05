import React, { useEffect, useMemo, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

const EXPORT_FIELDS = {
  teacher: [
    { key: 'student_name', label: 'Student Name' },
    { key: 'enrollment_no', label: 'Enrollment No' },
    { key: 'course_code', label: 'Course Code' },
    { key: 'course_name', label: 'Course Name' },
    { key: 'components', label: 'Components' },
    { key: 'faculty_code', label: 'Faculty Code' },
    { key: 'teacher_name', label: 'Faculty' },
    { key: 'window_name', label: 'Window Name' },
    { key: 'semester', label: 'Semester' },
    { key: 'department', label: 'Department' },
  ],
  course: [
    { key: 'student_name', label: 'Student Name' },
    { key: 'enrollment_no', label: 'Enrollment No' },
    { key: 'course_code', label: 'Course Code' },
    { key: 'course_name', label: 'Course Name' },
    { key: 'components', label: 'Components' },
    { key: 'faculty_code', label: 'Faculty Code' },
    { key: 'teacher_name', label: 'Faculty' },
    { key: 'window_name', label: 'Window Name' },
    { key: 'semester', label: 'Semester' },
    { key: 'department', label: 'Department' },
  ],
}

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
  const [tableSearch, setTableSearch] = useState('')
  const [showTableFilters, setShowTableFilters] = useState(false)
  const [tableFilters, setTableFilters] = useState({
    department: '',
    course: '',
    teacher: '',
    window: '',
    assigned: '',
    completed: '',
    completion: '',
  })
  const [downloadingType, setDownloadingType] = useState('')
  const [exportModal, setExportModal] = useState({ open: false, reportType: 'teacher' })
  const [fieldOrder, setFieldOrder] = useState([])
  const [selectedFields, setSelectedFields] = useState({})

  const [selectedRow, setSelectedRow] = useState(null)
  const [students, setStudents] = useState([])
  const [studentsLoading, setStudentsLoading] = useState(false)
  const [componentModal, setComponentModal] = useState({ open: false, student: null })

  const splitComponentLabel = (name) => {
    const raw = String(name || '').trim()
    if (!raw) return { main: 'Component', sub: '' }
    const [mainPart, subPart] = raw.split('->').map((part) => (part || '').trim())
    return { main: mainPart || 'Component', sub: subPart || '' }
  }

  const groupStudentComponents = (components) => {
    const groupedMap = new Map()
    ;(Array.isArray(components) ? components : []).forEach((component) => {
      const { main, sub } = splitComponentLabel(component.assessment_component_name)
      if (!groupedMap.has(main)) {
        groupedMap.set(main, { main, total: 0, subMarks: {}, raw: [] })
      }
      const item = groupedMap.get(main)
      const marks = Number(component.obtained_marks || 0)
      item.total += marks
      item.raw.push(component)
      if (sub) {
        item.subMarks[sub] = marks
      }
    })

    const grouped = Array.from(groupedMap.values())
    grouped.sort((a, b) => a.main.localeCompare(b.main))
    return grouped
  }

  const sortSubComponents = (subKeys = []) => {
    return [...subKeys].sort((a, b) => {
      const aText = String(a || '').trim().toUpperCase()
      const bText = String(b || '').trim().toUpperCase()
      const aMatch = aText.match(/^CO\s*(\d+)$/)
      const bMatch = bText.match(/^CO\s*(\d+)$/)
      if (aMatch && bMatch) return Number(aMatch[1]) - Number(bMatch[1])
      return aText.localeCompare(bText)
    })
  }

  const openExportModal = (reportType) => {
    const preset = EXPORT_FIELDS[reportType] || []
    const order = preset.map((field) => field.key)
    const selected = {}
    order.forEach((key) => {
      selected[key] = true
    })
    setFieldOrder(order)
    setSelectedFields(selected)
    setExportModal({ open: true, reportType })
  }

  const moveField = (key, direction) => {
    setFieldOrder((prev) => {
      const index = prev.indexOf(key)
      if (index < 0) return prev
      const targetIndex = direction === 'up' ? index - 1 : index + 1
      if (targetIndex < 0 || targetIndex >= prev.length) return prev
      const next = [...prev]
      const tmp = next[index]
      next[index] = next[targetIndex]
      next[targetIndex] = tmp
      return next
    })
  }

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
      const list = Array.isArray(data) ? data : []
      setWindows(list.map((windowItem) => ({
        ...windowItem,
        window_name: windowItem.window_name || windowItem.windowName || windowItem.name || '',
      })))
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
      const normalizedRows = (Array.isArray(data.rows) ? data.rows : []).map((row) => ({
        ...row,
        window_name: row.window_name || row.windowName || '',
      }))
      setRows(normalizedRows)
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

  const downloadReport = async (reportType, fields = [], rowScope = null) => {
    if (!isAdminMode && !username) {
      setError('Username not found. Please login again.')
      return
    }
    if (isAdminMode && !departmentId) {
      setError('Please select a department before downloading report.')
      return
    }

    if (!Array.isArray(fields) || fields.length === 0) {
      setError('Please select at least one field for export.')
      return
    }

    setDownloadingType(reportType)
    setError('')
    try {
      const params = new URLSearchParams()
      if (isAdminMode) {
        params.append('department_id', departmentId)
      } else {
        params.append('username', username)
      }
      params.append('semester', semester)
      params.append('report_type', reportType)
      params.append('format', 'xlsx')
      params.append('fields', fields.join(','))

      if (rowScope && typeof rowScope === 'object') {
        if (rowScope.teacher_id) params.append('teacher_id', rowScope.teacher_id)
        if (rowScope.course_code) params.append('course_code', rowScope.course_code)
        if (rowScope.window_name) params.append('window_name', rowScope.window_name)
      }

      const endpoint = isAdminMode
        ? `${API_BASE_URL}/admin/mark-entry/download?${params.toString()}`
        : `${API_BASE_URL}/hod/mark-entry/download?${params.toString()}`

      const response = await fetch(endpoint)
      if (!response.ok) {
        const message = await response.text()
        throw new Error(message || 'Failed to download report')
      }

      const blob = await response.blob()
      const objectURL = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectURL
      const scopeSuffix = rowScope?.teacher_id
        ? `_${String(rowScope.teacher_id).replace(/\s+/g, '_')}_${String(rowScope.course_code || '').replace(/\s+/g, '_')}`
        : ''
      anchor.download = `mark_entry_${reportType}_sem${semester}${scopeSuffix}.xlsx`
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

  const filteredRows = useMemo(() => {
    const q = tableSearch.trim().toLowerCase()
    return (Array.isArray(rows) ? rows : []).filter((row) => {
      const departmentText = String(row.department_name || '').toLowerCase()
      const courseText = `${row.course_code || ''} ${row.course_name || ''}`.toLowerCase()
      const teacherText = `${row.teacher_name || ''} ${row.teacher_id || ''}`.toLowerCase()
      const windowText = `${row.window_name || ''} ${row.window_status || ''}`.toLowerCase()
      const assignedText = String(row.assigned_students ?? '').toLowerCase()
      const completedText = String(row.completed_students ?? '').toLowerCase()
      const completionText = Number(row.completion_percent || 0).toFixed(2).toLowerCase()

      const matchesSearch = !q || (
        departmentText.includes(q) ||
        courseText.includes(q) ||
        teacherText.includes(q) ||
        windowText.includes(q)
      )

      if (!matchesSearch) return false
      if (tableFilters.department && !departmentText.includes(tableFilters.department.trim().toLowerCase())) return false
      if (tableFilters.course && !courseText.includes(tableFilters.course.trim().toLowerCase())) return false
      if (tableFilters.teacher && !teacherText.includes(tableFilters.teacher.trim().toLowerCase())) return false
      if (tableFilters.window && !windowText.includes(tableFilters.window.trim().toLowerCase())) return false
      if (tableFilters.assigned && !assignedText.includes(tableFilters.assigned.trim().toLowerCase())) return false
      if (tableFilters.completed && !completedText.includes(tableFilters.completed.trim().toLowerCase())) return false
      if (tableFilters.completion && !completionText.includes(tableFilters.completion.trim().toLowerCase())) return false
      return true
    })
  }, [rows, tableSearch, tableFilters])

  const defaultTeacherFields = useMemo(() => (EXPORT_FIELDS.teacher || []).map((field) => field.key), [])

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
            {windowOptions.map((w) => {
              const rawName = (w.window_name ?? w.windowName ?? w.name ?? '')
              const windowLabel = String(rawName).trim() || `Window #${w.id}`
              return (
                <option key={w.id} value={w.id}>{windowLabel} ({w.start_at?.slice(0, 16)} - {w.end_at?.slice(0, 16)})</option>
              )
            })}
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
        </div>
      }
    >
      <div className="space-y-4">
        {error && <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm">{error}</div>}

        <div className="bg-white border rounded-lg p-3 flex flex-wrap items-center justify-between gap-2">
          <div className="text-sm text-gray-600">Export</div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => openExportModal('teacher')}
              disabled={downloadingType !== ''}
              className="border border-gray-300 px-3 py-2 rounded-lg text-sm disabled:opacity-60"
            >
              {downloadingType === 'teacher' ? 'Downloading...' : 'Teacher-wise XLSX'}
            </button>
            <button
              onClick={() => openExportModal('course')}
              disabled={downloadingType !== ''}
              className="border border-gray-300 px-3 py-2 rounded-lg text-sm disabled:opacity-60"
            >
              {downloadingType === 'course' ? 'Downloading...' : 'Course-wise XLSX'}
            </button>
          </div>
        </div>

        {exportModal.open && (
          <div className="fixed inset-0 bg-black bg-opacity-30 flex items-center justify-center z-50 p-4" onClick={() => setExportModal({ open: false, reportType: 'teacher' })}>
            <div className="bg-white rounded-lg w-full max-w-2xl p-4" onClick={(e) => e.stopPropagation()}>
              <div className="flex justify-between items-center mb-3">
                <h3 className="text-sm font-semibold text-gray-900">
                  Configure {exportModal.reportType === 'teacher' ? 'Teacher-wise' : 'Course-wise'} XLSX
                </h3>
                <button onClick={() => setExportModal({ open: false, reportType: 'teacher' })} className="text-gray-500 hover:text-gray-700">Close</button>
              </div>

              <div className="text-xs text-gray-500 mb-3">Choose fields and arrange order (top to bottom) for Excel columns.</div>

              <div className="max-h-[360px] overflow-auto border rounded-lg">
                <table className="min-w-full text-sm">
                  <thead className="bg-gray-50 sticky top-0">
                    <tr>
                      <th className="text-left px-3 py-2">Use</th>
                      <th className="text-left px-3 py-2">Field</th>
                      <th className="text-left px-3 py-2">Order</th>
                    </tr>
                  </thead>
                  <tbody>
                    {fieldOrder.map((key, index) => {
                      const label = (EXPORT_FIELDS[exportModal.reportType] || []).find((f) => f.key === key)?.label || key
                      return (
                        <tr key={key} className="border-t">
                          <td className="px-3 py-2">
                            <input
                              type="checkbox"
                              checked={!!selectedFields[key]}
                              onChange={(e) => setSelectedFields((prev) => ({ ...prev, [key]: e.target.checked }))}
                            />
                          </td>
                          <td className="px-3 py-2">{label}</td>
                          <td className="px-3 py-2">
                            <div className="flex gap-2">
                              <button disabled={index === 0} onClick={() => moveField(key, 'up')} className="px-2 py-1 border rounded disabled:opacity-50">↑</button>
                              <button disabled={index === fieldOrder.length - 1} onClick={() => moveField(key, 'down')} className="px-2 py-1 border rounded disabled:opacity-50">↓</button>
                            </div>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              <div className="flex justify-end gap-2 mt-4">
                <button onClick={() => setExportModal({ open: false, reportType: 'teacher' })} className="px-3 py-2 border rounded-lg text-sm">Cancel</button>
                <button
                  onClick={() => {
                    const orderedFields = fieldOrder.filter((key) => selectedFields[key])
                    downloadReport(exportModal.reportType, orderedFields)
                    setExportModal({ open: false, reportType: 'teacher' })
                  }}
                  className="px-3 py-2 bg-primary text-white rounded-lg text-sm"
                >
                  Download XLSX
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          <MetricCard title="Assignments" value={summary?.total_assignments || 0} />
          <MetricCard title="Completed" value={summary?.completed_assignments || 0} />
          <MetricCard title="Pending" value={summary?.pending_assignments || 0} />
          <MetricCard title="Students" value={summary?.total_students || 0} />
          <MetricCard title="Done Students" value={summary?.completed_students || 0} />
        </div>

        <div className="bg-white border rounded-lg overflow-hidden">
          <div className="px-3 py-2 border-b bg-gray-50 flex flex-wrap gap-2 items-center">
            <button
              onClick={() => setShowTableFilters((prev) => !prev)}
              className="px-3 py-1.5 text-sm border rounded-lg"
            >
              {showTableFilters ? 'Hide Filters' : 'Show Filters'}
            </button>
            {showTableFilters && (
              <input
                value={tableSearch}
                onChange={(e) => setTableSearch(e.target.value)}
                placeholder="Search..."
                className="border border-gray-300 rounded-lg px-3 py-1.5 text-sm min-w-[220px]"
              />
            )}
            <button
              onClick={() => {
                setTableSearch('')
                setTableFilters({ department: '', course: '', teacher: '', window: '', assigned: '', completed: '', completion: '' })
              }}
              className="px-3 py-1.5 text-sm border rounded-lg"
            >
              Clear Filters
            </button>
          </div>
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
                  <th className="text-left px-3 py-2">Actions</th>
                </tr>
                {showTableFilters && (
                  <tr>
                    <th className="px-3 py-2">
                      <input value={tableFilters.department} onChange={(e) => setTableFilters((prev) => ({ ...prev, department: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.course} onChange={(e) => setTableFilters((prev) => ({ ...prev, course: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.teacher} onChange={(e) => setTableFilters((prev) => ({ ...prev, teacher: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.window} onChange={(e) => setTableFilters((prev) => ({ ...prev, window: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.assigned} onChange={(e) => setTableFilters((prev) => ({ ...prev, assigned: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs text-right" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.completed} onChange={(e) => setTableFilters((prev) => ({ ...prev, completed: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs text-right" />
                    </th>
                    <th className="px-3 py-2">
                      <input value={tableFilters.completion} onChange={(e) => setTableFilters((prev) => ({ ...prev, completion: e.target.value }))} placeholder="Filter" className="w-full border border-gray-300 rounded px-2 py-1 text-xs text-right" />
                    </th>
                    <th className="px-3 py-2"></th>
                  </tr>
                )}
              </thead>
              <tbody>
                {loading ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">Loading...</td></tr>
                ) : filteredRows.length === 0 ? (
                  <tr><td colSpan="8" className="px-3 py-6 text-center text-gray-500">No monitor rows found.</td></tr>
                ) : filteredRows.map((row) => (
                  <React.Fragment key={`${row.course_id}-${row.teacher_id}-${row.department_id}`}>
                    <tr className="border-t hover:bg-gray-50">
                      <td className="px-3 py-2">{row.department_name || '-'}</td>
                      <td className="px-3 py-2">
                        <div className="font-medium text-gray-900">{row.course_code}</div>
                        <div className="text-gray-500">{row.course_name}</div>
                      </td>
                      <td className="px-3 py-2">
                        <div className="font-medium text-gray-900">{row.teacher_name}</div>
                        <div className="text-gray-500">{row.teacher_id}</div>
                      </td>
                      <td className="px-3 py-2">
                        {row.window_name
                          ? `${row.window_name}${row.window_status ? ` (${row.window_status})` : ''}`
                          : (row.window_status || '-')}
                      </td>
                      <td className="px-3 py-2 text-right">{row.assigned_students || 0}</td>
                      <td className="px-3 py-2 text-right">{row.completed_students || 0}</td>
                      <td className="px-3 py-2 text-right">{Number(row.completion_percent || 0).toFixed(2)}</td>
                      <td className="px-3 py-2">
                        <button onClick={() => loadStudents(row)} className="text-primary hover:underline text-left">View students</button>
                      </td>
                    </tr>
                    <tr className="border-t bg-gray-50/40">
                      <td colSpan="8" className="px-3 py-2">
                        <button
                          onClick={() => downloadReport('teacher', defaultTeacherFields, row)}
                          disabled={downloadingType !== ''}
                          className="text-primary hover:underline text-sm disabled:opacity-60"
                        >
                          {downloadingType === 'teacher' ? 'Downloading...' : 'Download XLSX for this faculty row'}
                        </button>
                      </td>
                    </tr>
                  </React.Fragment>
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
                      const groupedComponents = groupStudentComponents(components)
                      const showInline = groupedComponents.length > 0 && groupedComponents.length <= 3
                      return (
                        <tr key={student.student_id} className="border-t">
                          <td className="px-3 py-2">{student.enrollment_no || '-'}</td>
                          <td className="px-3 py-2">{student.student_name}</td>
                          <td className="px-3 py-2 text-right">{Number(student.total_marks || 0).toFixed(2)}</td>
                          <td className="px-3 py-2">{student.has_mark_entry ? 'Yes' : 'No'}</td>
                          <td className="px-3 py-2">
                            {showInline ? (
                              <div className="flex flex-wrap gap-1">
                                {groupedComponents.map((group) => (
                                  <span key={`${student.student_id}-${group.main}`} className="px-2 py-1 text-xs rounded bg-gray-100 text-gray-700">
                                    {group.main}: {Number(group.total || 0).toFixed(2)}
                                  </span>
                                ))}
                              </div>
                            ) : groupedComponents.length > 0 ? (
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
                <h4 className="font-semibold text-gray-900">{componentModal.student.student_name} - Component Summary</h4>
                <button onClick={() => setComponentModal({ open: false, student: null })} className="text-gray-500 hover:text-gray-700">Close</button>
              </div>
              <div className="max-h-[420px] overflow-auto border rounded-lg">
                {(() => {
                  const grouped = groupStudentComponents(componentModal.student.components || [])
                  if (grouped.length === 0) {
                    return <div className="px-3 py-4 text-center text-gray-500 text-sm">No components found.</div>
                  }

                  const groupsWithSubs = grouped.map((group) => {
                    const subKeys = sortSubComponents(Object.keys(group.subMarks || {}).filter(Boolean))
                    return { ...group, subKeys }
                  })

                  return (
                    <table className="min-w-full text-sm">
                      <thead className="bg-gray-50">
                        <tr>
                          <th rowSpan={2} className="text-left px-3 py-2">Metric</th>
                          {groupsWithSubs.map((group) => {
                            const span = group.subKeys.length > 0 ? group.subKeys.length + 1 : 1
                            return (
                              <th key={`main-head-${group.main}`} colSpan={span} className="text-center px-3 py-2">{group.main}</th>
                            )
                          })}
                        </tr>
                        <tr>
                          {groupsWithSubs.map((group) => (
                            <React.Fragment key={`sub-head-${group.main}`}>
                              {group.subKeys.map((sub) => (
                                <th key={`sub-col-${group.main}-${sub}`} className="text-right px-3 py-2">{sub}</th>
                              ))}
                              <th className="text-right px-3 py-2">Total</th>
                            </React.Fragment>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        <tr className="border-t">
                          <td className="px-3 py-2 font-medium">Marks</td>
                          {groupsWithSubs.map((group) => (
                            <React.Fragment key={`sub-data-${group.main}`}>
                              {group.subKeys.map((sub) => (
                                <td key={`sub-mark-${group.main}-${sub}`} className="px-3 py-2 text-right">
                                  {group.subMarks[sub] !== undefined ? Number(group.subMarks[sub]).toFixed(2) : '-'}
                                </td>
                              ))}
                              <td className="px-3 py-2 text-right font-semibold">{Number(group.total || 0).toFixed(2)}</td>
                            </React.Fragment>
                          ))}
                        </tr>
                      </tbody>
                    </table>
                  )
                })()}
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
