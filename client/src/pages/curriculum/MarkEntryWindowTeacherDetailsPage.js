import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import * as XLSX from 'xlsx'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryWindowTeacherDetailsPage() {
  const { windowId } = useParams()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const scope = (searchParams.get('scope') || 'teacher').trim().toLowerCase()
  const teacherId = (searchParams.get('teacher_id') || '').trim()
  const userId = (searchParams.get('user_id') || '').trim()
  const userName = (searchParams.get('user_name') || '').trim()
  const courseId = (searchParams.get('course_id') || '').trim()
  const teacherName = (searchParams.get('teacher_name') || '').trim()
  const courseCode = (searchParams.get('course_code') || '').trim()
  const courseName = (searchParams.get('course_name') || '').trim()
  const selectedDepartmentId = (searchParams.get('department_id') || '').trim()

  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const [error, setError] = useState('')
  const [students, setStudents] = useState([])
  const [lastUpdatedAt, setLastUpdatedAt] = useState(null)
  const [assessmentComponents, setAssessmentComponents] = useState([])

  const isUserScope = scope === 'user'
  const canFetch = Boolean(windowId && courseId && (isUserScope ? userId : teacherId))

  const loadStudents = useCallback(async (isRefresh = false) => {
    if (!canFetch) return
    if (isRefresh) setRefreshing(true)
    else setLoading(true)
    setError('')

    try {
      const params = new URLSearchParams()
      params.append('course_id', courseId)
      if (selectedDepartmentId) {
        params.append('department_id', selectedDepartmentId)
      }

      if (isUserScope) {
        params.append('user_id', userId)
      } else {
        params.append('teacher_id', teacherId)
        params.append('window_id', windowId)
      }

      const endpoint = isUserScope
        ? `${API_BASE_URL}/mark-entry-windows/${windowId}/user-students?${params.toString()}`
        : `${API_BASE_URL}/hod/mark-entry/teacher-students?${params.toString()}`

      const res = await fetch(endpoint)
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Failed to fetch entry details')
      }
      const data = await res.json()
      setStudents(Array.isArray(data.students) ? data.students : [])
      setLastUpdatedAt(new Date())
    } catch (err) {
      setStudents([])
      setError(err.message || 'Failed to load student details')
    } finally {
      if (isRefresh) setRefreshing(false)
      else setLoading(false)
    }
  }, [canFetch, teacherId, userId, courseId, windowId, isUserScope, selectedDepartmentId])

  useEffect(() => {
    if (!canFetch) {
      setLoading(false)
      setError(isUserScope ? 'Missing user_id or course_id in URL' : 'Missing teacher_id or course_id in URL')
      return
    }

    loadStudents(false)
    const interval = setInterval(() => {
      loadStudents(true)
    }, 15000)

    return () => clearInterval(interval)
  }, [canFetch, loadStudents, isUserScope])

  useEffect(() => {
    const loadWindowComponents = async () => {
      if (!windowId) {
        setAssessmentComponents([])
        return
      }

      try {
        const res = await fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?window_id=${windowId}`)
        if (!res.ok) {
          throw new Error('Failed to fetch window components')
        }

        const data = await res.json()
        const details = Array.isArray(data) && data.length > 0 ? data[0] : null
        const components = Array.isArray(details?.assessment_components) ? details.assessment_components : []
        setAssessmentComponents(Array.from(new Set(
          components
            .map((component) => String(component || '').trim())
            .filter(Boolean)
        )))
      } catch (err) {
        console.error('Error loading window assessment components:', err)
        setAssessmentComponents([])
      }
    }

    loadWindowComponents()
  }, [windowId])

  const summary = useMemo(() => {
    const totalStudents = students.length
    const updatedStudents = students.filter((student) => student.has_mark_entry).length
    return {
      totalStudents,
      updatedStudents,
      notUpdatedStudents: totalStudents - updatedStudents,
    }
  }, [students])

  const getInnovativePracticeBaseName = (name = '') => {
    const normalized = String(name).replace(/\s+/g, ' ').trim()
    const match = normalized.match(/^(Innovative Practice\s+[12])\s*-\s*\(\s*[12]\s*\)$/i)
    return match ? match[1] : null
  }

  const buildDisplayComponents = (components = []) => {
    const display = []
    const groupedMap = new Map()

    components.forEach((component) => {
      const baseName = getInnovativePracticeBaseName(component.assessment_component_name)
      if (!baseName) {
        display.push(component)
        return
      }

      const learningModePart = component.learning_mode_id || component.learning_mode || 'na'
      const key = `${learningModePart}|${baseName.toLowerCase()}`

      if (!groupedMap.has(key)) {
        groupedMap.set(key, {
          ...component,
          assessment_component_name: baseName,
          assessment_component_ids: [component.assessment_component_id].filter(Boolean),
        })
        display.push(groupedMap.get(key))
        return
      }

      const grouped = groupedMap.get(key)
      if (component.assessment_component_id) {
        grouped.assessment_component_ids.push(component.assessment_component_id)
      }

      const groupedMarkMissing = grouped.obtained_marks === '' || grouped.obtained_marks === null || grouped.obtained_marks === undefined
      const componentMarkPresent = component.obtained_marks !== '' && component.obtained_marks !== null && component.obtained_marks !== undefined
      if (groupedMarkMissing && componentMarkPresent) {
        grouped.obtained_marks = component.obtained_marks
      }
    })

    return display
  }

  const getDisplayTotalMarks = (displayComponents = []) => {
    return displayComponents.reduce((sum, component) => {
      const mark = Number(component?.obtained_marks)
      return Number.isFinite(mark) ? sum + mark : sum
    }, 0)
  }

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
    if (isUserScope) {
      if (userName) return userName
      return userId || 'User'
    }
    if (teacherName) return teacherName
    return teacherId || 'Teacher'
  }

  const getCourseDisplay = () => {
    if (courseCode && courseName) return `${courseCode} - ${courseName}`
    if (courseCode) return courseCode
    if (courseName) return courseName
    return `Course ${courseId}`
  }

  const downloadStudentDetailsExcel = () => {
    try {
      setDownloading(true)
      const rows = students.map((student) => {
        const components = Array.isArray(student.components) ? student.components : []
        const displayComponents = buildDisplayComponents(components)
        const enteredMarks = displayComponents.length > 0
          ? displayComponents
            .map((component) => `${component.assessment_component_name}: ${Number(component.obtained_marks || 0).toFixed(2)}`)
            .join('; ')
          : 'Not Updated'

        return {
          RegisterNo: student.register_id || '-',
          StudentName: student.student_name || '-',
          TotalMarks: getDisplayTotalMarks(displayComponents).toFixed(2),
          Status: student.has_mark_entry ? 'Updated' : 'Not Updated',
          EnteredMarks: enteredMarks,
        }
      })

      const wb = XLSX.utils.book_new()
      const ws = XLSX.utils.json_to_sheet(rows)
      XLSX.utils.book_append_sheet(wb, ws, 'Teacher Student Marks')

      const safeTeacher = String(getDisplayName())
        .replace(/[^a-zA-Z0-9_-]+/g, '_')
        .replace(/^_+|_+$/g, '')
      const safeCourse = String(courseCode || courseName || courseId || 'course')
        .replace(/[^a-zA-Z0-9_-]+/g, '_')
        .replace(/^_+|_+$/g, '')

      XLSX.writeFile(wb, `window_${windowId}_${safeTeacher || 'teacher'}_${safeCourse || 'course'}_student_marks.xlsx`)
    } finally {
      setDownloading(false)
    }
  }

  return (
    <MainLayout title={isUserScope ? 'User Student Mark Details' : 'Teacher Student Mark Details'} subtitle="Live view of entered marks in the selected window">
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <button
            onClick={() => navigate(`/mark-entry-windows/${windowId}`)}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            ← Back to Window Details
          </button>
          <div className="flex items-center gap-2">
            <button
              onClick={downloadStudentDetailsExcel}
              disabled={loading || students.length === 0 || downloading}
              className="px-4 py-2 text-sm font-medium text-emerald-700 bg-emerald-50 border border-emerald-200 rounded-lg hover:bg-emerald-100 disabled:opacity-60"
            >
              {downloading ? 'Downloading...' : 'Download Excel'}
            </button>
            <button
              onClick={() => loadStudents(true)}
              disabled={!canFetch || refreshing}
              className="px-4 py-2 text-sm font-medium text-white bg-primary border border-primary rounded-lg hover:opacity-90 disabled:opacity-60"
            >
              {refreshing ? 'Refreshing...' : 'Refresh Now'}
            </button>
          </div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-800">{getDisplayName()}</h3>
              <p className="text-xs text-gray-500 mt-1">{isUserScope ? 'User ID' : 'Teacher ID'}: {isUserScope ? (userId || '-') : (teacherId || '-')}</p>
              <p className="text-xs text-gray-500 mt-1">Window ID: #{windowId}</p>
              {selectedDepartmentId && (
                <p className="text-xs text-gray-500 mt-1">Department ID: {selectedDepartmentId}</p>
              )}
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

          <div className="mt-5">
            <div className="text-xs font-semibold uppercase tracking-wide text-gray-500">Assessment Components</div>
            {assessmentComponents.length > 0 ? (
              <div className="mt-3 flex flex-wrap gap-2">
                {assessmentComponents.map((component) => (
                  <span
                    key={`${windowId}-${component}`}
                    className="inline-flex items-center rounded-full border border-purple-500 bg-purple-50 px-3 py-1 text-xs font-medium text-purple-700"
                  >
                    {component}
                  </span>
                ))}
              </div>
            ) : (
              <div className="mt-2 text-sm text-gray-500">No assessment components available for this window.</div>
            )}
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
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Register No</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Student</th>
                    <th className="px-4 py-3 text-right text-xs font-semibold text-gray-600 uppercase">Total Marks</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Entered Marks</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {students.map((student) => {
                    const components = Array.isArray(student.components) ? student.components : []
                    const displayComponents = buildDisplayComponents(components)
                    const displayTotalMarks = getDisplayTotalMarks(displayComponents)
                    const hasMarks = Boolean(student.has_mark_entry)
                    return (
                      <tr key={student.student_id} className="hover:bg-gray-50">
                        <td className="px-4 py-3 text-sm text-gray-700">{student.register_id || '-'}</td>
                        <td className="px-4 py-3 text-sm text-gray-800 font-medium">{student.student_name || '-'}</td>
                        <td className="px-4 py-3 text-sm text-gray-700 text-right">{displayTotalMarks.toFixed(2)}</td>
                        <td className="px-4 py-3 text-sm">
                          {hasMarks ? (
                            <span className="inline-flex items-center px-2 py-1 rounded border text-xs font-medium bg-green-100 text-green-700 border-green-200">Updated</span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-1 rounded border text-xs font-medium bg-amber-100 text-amber-700 border-amber-200">Not Updated</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-700">
                          {hasMarks && displayComponents.length > 0 ? (
                            <div className="flex flex-wrap gap-1">
                              {displayComponents.map((component, index) => (
                                <span key={`${student.student_id}-${(component.assessment_component_ids || [component.assessment_component_id || index]).join('-')}`} className="px-2 py-1 text-xs rounded bg-gray-100 text-gray-700 border border-gray-200">
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
