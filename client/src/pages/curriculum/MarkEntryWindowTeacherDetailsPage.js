import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryWindowTeacherDetailsPage() {
  const { windowId } = useParams()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const teacherId = (searchParams.get('teacher_id') || '').trim()
  const courseId = (searchParams.get('course_id') || '').trim()
  const teacherName = (searchParams.get('teacher_name') || '').trim()
  const courseCode = (searchParams.get('course_code') || '').trim()
  const courseName = (searchParams.get('course_name') || '').trim()

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [students, setStudents] = useState([])
  const [lastUpdatedAt, setLastUpdatedAt] = useState(null)

  const canFetch = Boolean(windowId && teacherId && courseId)

  const loadStudents = useCallback(async (isRefresh = false) => {
    if (!canFetch) return
    if (isRefresh) setRefreshing(true)
    else setLoading(true)
    setError('')

    try {
      const params = new URLSearchParams()
      params.append('teacher_id', teacherId)
      params.append('course_id', courseId)
      params.append('window_id', windowId)

      const res = await fetch(`${API_BASE_URL}/hod/mark-entry/teacher-students?${params.toString()}`)
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Failed to fetch teacher students')
      }
      const data = await res.json()
      setStudents(Array.isArray(data.students) ? data.students : [])
      setLastUpdatedAt(new Date())
    } catch (err) {
      setStudents([])
      setError(err.message || 'Failed to load teacher student details')
    } finally {
      if (isRefresh) setRefreshing(false)
      else setLoading(false)
    }
  }, [canFetch, teacherId, courseId, windowId])

  useEffect(() => {
    if (!canFetch) {
      setLoading(false)
      setError('Missing teacher_id or course_id in URL')
      return
    }

    loadStudents(false)
    const interval = setInterval(() => {
      loadStudents(true)
    }, 15000)

    return () => clearInterval(interval)
  }, [canFetch, loadStudents])

  const summary = useMemo(() => {
    const totalStudents = students.length
    const updatedStudents = students.filter((student) => student.has_mark_entry).length
    return {
      totalStudents,
      updatedStudents,
      notUpdatedStudents: totalStudents - updatedStudents,
    }
  }, [students])

  const formatDateTime = (value) => {
    if (!value) return '-'
    return new Date(value).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const getDisplayName = () => {
    if (teacherName) return teacherName
    return teacherId || 'Teacher'
  }

  const getCourseDisplay = () => {
    if (courseCode && courseName) return `${courseCode} - ${courseName}`
    if (courseCode) return courseCode
    if (courseName) return courseName
    return `Course ${courseId}`
  }

  return (
    <MainLayout title="Teacher Student Mark Details" subtitle="Live view of entered marks for this teacher in the selected window">
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <button
            onClick={() => navigate(`/mark-entry-windows/${windowId}`)}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            ← Back to Window Details
          </button>
          <button
            onClick={() => loadStudents(true)}
            disabled={!canFetch || refreshing}
            className="px-4 py-2 text-sm font-medium text-white bg-primary border border-primary rounded-lg hover:opacity-90 disabled:opacity-60"
          >
            {refreshing ? 'Refreshing...' : 'Refresh Now'}
          </button>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-800">{getDisplayName()}</h3>
              <p className="text-xs text-gray-500 mt-1">Teacher ID: {teacherId || '-'}</p>
              <p className="text-xs text-gray-500 mt-1">Window ID: #{windowId}</p>
            </div>
            <div className="text-xs text-gray-600">
              <div>Course: <span className="font-medium text-gray-800">{getCourseDisplay()}</span></div>
              <div className="mt-1">Last Updated: <span className="font-medium text-gray-800">{formatDateTime(lastUpdatedAt)}</span></div>
              <div className="mt-1">Auto Refresh: <span className="font-medium text-gray-800">Every 15 seconds</span></div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mt-4 text-xs">
            <div className="bg-gray-50 border border-gray-200 rounded p-3">
              <div className="text-gray-500">Total Students</div>
              <div className="font-semibold text-gray-800 mt-0.5">{summary.totalStudents}</div>
            </div>
            <div className="bg-green-50 border border-green-200 rounded p-3">
              <div className="text-green-700">Updated</div>
              <div className="font-semibold text-green-800 mt-0.5">{summary.updatedStudents}</div>
            </div>
            <div className="bg-amber-50 border border-amber-200 rounded p-3">
              <div className="text-amber-700">Not Updated</div>
              <div className="font-semibold text-amber-800 mt-0.5">{summary.notUpdatedStudents}</div>
            </div>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="border-b border-gray-200 px-6 py-4">
            <h4 className="text-sm font-semibold text-gray-700">Students and Entered Marks</h4>
          </div>

          {loading ? (
            <div className="p-6 text-sm text-gray-500">Loading students...</div>
          ) : error ? (
            <div className="p-6 text-sm text-red-700 bg-red-50 border-t border-red-200">{error}</div>
          ) : students.length === 0 ? (
            <div className="p-6 text-sm text-gray-500">No students found for this teacher and course.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Enrollment</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Student</th>
                    <th className="px-4 py-3 text-right text-xs font-semibold text-gray-600 uppercase">Total Marks</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Entered Marks</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {students.map((student) => {
                    const components = Array.isArray(student.components) ? student.components : []
                    const hasMarks = Boolean(student.has_mark_entry)
                    return (
                      <tr key={student.student_id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 text-sm text-gray-700">{student.enrollment_no || '-'}</td>
                        <td className="px-4 py-3 text-sm text-gray-800 font-medium">{student.student_name || '-'}</td>
                        <td className="px-4 py-3 text-sm text-gray-700 text-right">{Number(student.total_marks || 0).toFixed(2)}</td>
                        <td className="px-4 py-3 text-sm">
                          {hasMarks ? (
                            <span className="inline-flex items-center px-2 py-1 rounded border text-xs font-medium bg-green-100 text-green-700 border-green-200">Updated</span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-1 rounded border text-xs font-medium bg-amber-100 text-amber-700 border-amber-200">Not Updated</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-700">
                          {hasMarks && components.length > 0 ? (
                            <div className="flex flex-wrap gap-1">
                              {components.map((component, index) => (
                                <span key={`${student.student_id}-${component.assessment_component_id || index}`} className="px-2 py-1 text-xs rounded bg-gray-100 text-gray-700 border border-gray-200">
                                  {component.assessment_component_name}: {Number(component.obtained_marks || 0).toFixed(2)}
                                </span>
                              ))}
                            </div>
                          ) : (
                            <span className="text-amber-700 font-medium">Not Updated</span>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  )
}

export default MarkEntryWindowTeacherDetailsPage