import React, { useEffect, useMemo, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

const RANGE_KEYS = ['0-10', '11-20', '21-24', '25-30', '31-40', '41-50']

function ResultAnalysisPage() {
  const username = localStorage.getItem('username')
  const userRole = (localStorage.getItem('userRole') || '').toLowerCase()
  const isAdminMode = userRole === 'admin' || userRole === 'coe'
  const canAccess = isAdminMode || userRole === 'hod'

  const [semester, setSemester] = useState('4')
  const [examType, setExamType] = useState('ALL')
  const [learningMode, setLearningMode] = useState('PBL')
  const [departmentId, setDepartmentId] = useState('')
  const [markTypeOptions, setMarkTypeOptions] = useState([])

  const [departments, setDepartments] = useState([])

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [departmentName, setDepartmentName] = useState('')
  const [rows, setRows] = useState([])

  useEffect(() => {
    if (!canAccess) return
    if (isAdminMode) fetchDepartments()
  }, [canAccess, isAdminMode])

  useEffect(() => {
    if (!canAccess) return
    fetchMarkTypes()
  }, [canAccess, learningMode])

  useEffect(() => {
    if (!canAccess) return
    fetchAnalysis()
  }, [semester, examType, learningMode, departmentId, canAccess])

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

  const fetchAnalysis = async () => {
    if (!semester) return
    setLoading(true)
    setError('')

    try {
      const params = new URLSearchParams()
      params.append('semester', semester)
      if (examType && examType !== 'ALL') params.append('exam_type', examType)
      params.append('learning_mode', learningMode)
      params.append('role', userRole)

      if (isAdminMode) {
        if (!departmentId) {
          setRows([])
          setDepartmentName('')
          setLoading(false)
          return
        }
        params.append('department_id', departmentId)
      } else {
        params.append('username', username || '')
      }

      const requestUrl = `${API_BASE_URL}/hod/result-analysis?${params.toString()}`
      const response = await fetch(requestUrl)
      const rawText = await response.text()
      let data = {}
      try {
        data = rawText ? JSON.parse(rawText) : {}
      } catch (parseError) {
        data = {}
      }
      if (!response.ok) throw new Error(data.error || data.message || 'Failed to load analysis')

      setRows(Array.isArray(data.rows) ? data.rows : [])
      setDepartmentName(data.department_name || '')
    } catch (err) {
      setRows([])
      setDepartmentName('')
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const fetchMarkTypes = async () => {
    try {
      const learningModeId = learningMode === 'UAL' ? '1' : '2'
      const response = await fetch(`${API_BASE_URL}/mark-categories/by-learning-mode?learning_mode_id=${learningModeId}`)
      const data = await response.json()
      const options = Array.isArray(data) ? data : []
      setMarkTypeOptions(options)

      setExamType((previous) => {
        if (previous === 'ALL') return 'ALL'
        const exists = options.some((item) => (item?.name || '').trim() === previous)
        return exists ? previous : 'ALL'
      })
    } catch (err) {
      setMarkTypeOptions([])
      setExamType('ALL')
    }
  }

  const columnHeaders = useMemo(() => rows.map((item) => item.course_code), [rows])

  const componentMetrics = useMemo(() => {
    const componentMap = new Map()
    rows.forEach((row) => {
      const components = Array.isArray(row.components) ? row.components : []
      components.forEach((component) => {
        const id = Number(component.component_id)
        if (!Number.isFinite(id)) return
        if (!componentMap.has(id)) {
          componentMap.set(id, component.component_name || `Component ${id}`)
        }
      })
    })

    return Array.from(componentMap.entries())
      .sort((a, b) => a[0] - b[0])
      .map(([id, name]) => ({ key: `component_avg_${id}`, label: `Component Avg: ${name}` }))
  }, [rows])

  if (!canAccess) {
    return (
      <MainLayout title="Result Analysis" subtitle="Course-wise result analytics">
        <div className="bg-white border rounded-lg p-6 text-red-600">Access denied.</div>
      </MainLayout>
    )
  }

  const valueByKey = (row, key) => {
    switch (key) {
      case 'registered': return row.registered || 0
      case 'appeared': return row.appeared || 0
      case 'absent': return row.absent || 0
      case 'passed': return row.passed || 0
      case 'failed': return row.failed || 0
      case 'pass_percent': return `${Number(row.pass_percent || 0).toFixed(2)}%`
      case 'maximum_mark': return Number(row.maximum_mark || 0).toFixed(2)
      case 'minimum_mark': return Number(row.minimum_mark || 0).toFixed(2)
      case 'average_mark': return Number(row.average_mark || 0).toFixed(2)
      default:
        if (key.startsWith('component_avg_')) {
          const componentID = Number(key.replace('component_avg_', ''))
          const components = Array.isArray(row.components) ? row.components : []
          const component = components.find((item) => Number(item.component_id) === componentID)
          if (!component) return '-'
          return Number(component.average_mark || 0).toFixed(2)
        }
        if (key.startsWith('range_')) {
          const bucket = key.replace('range_', '')
          return row.ranges?.[bucket] || 0
        }
        return '-'
    }
  }

  const metricRows = [
    { key: 'registered', label: 'No. of students registered' },
    { key: 'appeared', label: 'No. of students appeared' },
    { key: 'absent', label: 'No. of students absent' },
    { key: 'passed', label: 'No. of students passed' },
    { key: 'failed', label: 'No. of students failed' },
    { key: 'pass_percent', label: 'Pass %' },
    { key: 'maximum_mark', label: 'Maximum marks obtained' },
    { key: 'minimum_mark', label: 'Minimum marks obtained' },
    { key: 'average_mark', label: 'Average marks obtained' },
    ...componentMetrics,
    ...RANGE_KEYS.map((bucket) => ({ key: `range_${bucket}`, label: `Range ${bucket}` })),
  ]

  const examTypeLabel = examType === 'ALL' ? 'All' : examType

  return (
    <MainLayout
      title="Result Analysis"
      subtitle={`Department: ${departmentName || '-'} • Semester: ${semester} • Exam: ${examTypeLabel} • Mode: ${learningMode}`}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <select value={examType} onChange={(e) => setExamType(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm">
            <option value="ALL">All</option>
            {markTypeOptions.map((item) => (
              <option key={item.id} value={item.name}>{item.name}</option>
            ))}
          </select>

          <select value={semester} onChange={(e) => setSemester(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm">
            {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
              <option key={value} value={value}>Semester {value}</option>
            ))}
          </select>

          <select value={learningMode} onChange={(e) => setLearningMode(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm min-w-[140px]">
            <option value="PBL">PBL</option>
            <option value="UAL">UAL</option>
          </select>

          {isAdminMode && (
            <select value={departmentId} onChange={(e) => setDepartmentId(e.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm min-w-[180px]">
              <option value="">Select department</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>{d.label}</option>
              ))}
            </select>
          )}

          <button className="bg-primary text-white px-3 py-2 rounded-lg text-sm" onClick={fetchAnalysis}>Refresh</button>
        </div>
      }
    >
      {error && <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg p-3 text-sm mb-4">{error}</div>}

      <div className="bg-white border rounded-lg overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead className="bg-gray-50">
              <tr>
                <th className="text-left px-3 py-2 min-w-[250px]">Description / Course</th>
                {columnHeaders.map((header, index) => (
                  <th key={`${header}-${index}`} className="text-center px-3 py-2 min-w-[120px]">{header}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr><td colSpan={Math.max(columnHeaders.length + 1, 2)} className="px-3 py-6 text-center text-gray-500">Loading...</td></tr>
              ) : rows.length === 0 ? (
                <tr><td colSpan={2} className="px-3 py-6 text-center text-gray-500">No analysis data.</td></tr>
              ) : metricRows.map((metric) => (
                <tr key={metric.key} className="border-t">
                  <td className="px-3 py-2 font-medium text-gray-800">{metric.label}</td>
                  {rows.map((row) => (
                    <td key={`${metric.key}-${row.course_id}`} className="px-3 py-2 text-center text-gray-700">{valueByKey(row, metric.key)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </MainLayout>
  )
}

export default ResultAnalysisPage
