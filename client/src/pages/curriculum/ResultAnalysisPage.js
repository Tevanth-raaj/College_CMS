import React, { useEffect, useMemo, useState } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

const RANGE_KEYS = ['0-10', '11-20', '21-24', '25-30', '31-40', '41-50']

function ResultAnalysisPage() {
  const username = localStorage.getItem('username')
  const userRole = localStorage.getItem('userRole')

  const [semester, setSemester] = useState('4')
  const [examType, setExamType] = useState('PT1')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [departmentName, setDepartmentName] = useState('')
  const [rows, setRows] = useState([])

  const canAccess = userRole === 'hod' || userRole === 'admin' || userRole === 'coe'

  const fetchAnalysis = async () => {
    if (!username || !semester) return
    setLoading(true)
    setError('')
    try {
      const response = await fetch(`${API_BASE_URL}/hod/result-analysis?username=${encodeURIComponent(username)}&semester=${semester}&exam_type=${encodeURIComponent(examType)}`)
      const data = await response.json()
      if (!response.ok) {
        throw new Error(data.error || data.message || 'Failed to load analysis')
      }
      setRows(Array.isArray(data.rows) ? data.rows : [])
      setDepartmentName(data.department_name || '')
    } catch (err) {
      setRows([])
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (canAccess) fetchAnalysis()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [semester, examType, canAccess])

  const columnHeaders = useMemo(() => rows.map((item) => item.course_code), [rows])

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
    ...RANGE_KEYS.map((bucket) => ({ key: `range_${bucket}`, label: `Range ${bucket}` })),
  ]

  return (
    <MainLayout
      title="Result Analysis"
      subtitle={`Department: ${departmentName || '-'} • Semester: ${semester} • Exam: ${examType}`}
      actions={
        <div className="flex items-center gap-2">
          <select value={examType} onChange={(event) => setExamType(event.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm">
            <option value="PT1">PT1</option>
            <option value="PT2">PT2</option>
            <option value="Model">Model</option>
            <option value="EndSem">EndSem</option>
          </select>
          <select value={semester} onChange={(event) => setSemester(event.target.value)} className="border border-gray-300 rounded-lg px-3 py-2 text-sm">
            {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
              <option key={value} value={value}>Semester {value}</option>
            ))}
          </select>
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