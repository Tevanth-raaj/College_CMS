import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryWindowDetailsPage() {
  const { windowId } = useParams()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [windowData, setWindowData] = useState(null)

  useEffect(() => {
    const fetchWindowDetails = async () => {
      setLoading(true)
      setError('')
      try {
        const res = await fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?window_id=${windowId}`)
        if (!res.ok) throw new Error('Failed to fetch window details')
        const data = await res.json()
        const details = Array.isArray(data) && data.length > 0 ? data[0] : null
        if (!details) {
          setError('Window details not found')
          setWindowData(null)
        } else {
          setWindowData(details)
        }
      } catch (err) {
        setError(err.message || 'Failed to load window details')
        setWindowData(null)
      } finally {
        setLoading(false)
      }
    }

    if (windowId) fetchWindowDetails()
  }, [windowId])

  const allTeachers = useMemo(() => {
    if (!windowData) return []
    const pending = Array.isArray(windowData.pending_teachers) ? windowData.pending_teachers : []
    const completed = Array.isArray(windowData.completed_teachers) ? windowData.completed_teachers : []
    return [
      ...pending.map(t => ({ ...t, status: 'Pending' })),
      ...completed.map(t => ({ ...t, status: 'Completed' })),
    ]
  }, [windowData])

  const formatDate = (dateStr) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getStatusBadge = (status) => {
    if (status === 'Completed') {
      return 'bg-green-100 text-green-700 border-green-200'
    }
    return 'bg-amber-100 text-amber-700 border-amber-200'
  }

  return (
    <MainLayout title="Window Details" subtitle="Teachers involved in this mark entry window">
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <button
            onClick={() => navigate('/mark-entry-permissions')}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            ← Back to Existing Windows
          </button>
        </div>

        {loading ? (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-sm text-gray-500">
            Loading window details...
          </div>
        ) : error ? (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
            {error}
          </div>
        ) : windowData ? (
          <>
            <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="text-lg font-semibold text-gray-800">{windowData.window_name || `Window #${windowData.window_id}`}</h3>
                  <p className="text-xs text-gray-500 mt-1">Window ID: #{windowData.window_id}</p>
                </div>
                <div className="text-xs text-gray-600 text-right">
                  <div>Start: <span className="font-medium text-gray-800">{formatDate(windowData.start_at)}</span></div>
                  <div className="mt-1">End: <span className="font-medium text-gray-800">{formatDate(windowData.end_at)}</span></div>
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mt-4 text-xs">
                <div className="bg-gray-50 border border-gray-200 rounded p-3">
                  <div className="text-gray-500">Department</div>
                  <div className="font-semibold text-gray-800 mt-0.5">{windowData.department_name || 'All'}</div>
                </div>
                <div className="bg-gray-50 border border-gray-200 rounded p-3">
                  <div className="text-gray-500">Semester</div>
                  <div className="font-semibold text-gray-800 mt-0.5">{windowData.semester || 'All'}</div>
                </div>
                <div className="bg-gray-50 border border-gray-200 rounded p-3">
                  <div className="text-gray-500">Course</div>
                  <div className="font-semibold text-gray-800 mt-0.5">{windowData.course_code ? `${windowData.course_code} - ${windowData.course_name || ''}` : (windowData.course_name || 'All')}</div>
                </div>
                <div className="bg-gray-50 border border-gray-200 rounded p-3">
                  <div className="text-gray-500">Teachers Involved</div>
                  <div className="font-semibold text-gray-800 mt-0.5">{allTeachers.length}</div>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
              <div className="border-b border-gray-200 px-6 py-4">
                <h4 className="text-sm font-semibold text-gray-700">Teachers for Mark Entry</h4>
              </div>

              {allTeachers.length === 0 ? (
                <div className="p-6 text-sm text-gray-500">No teachers found for this window.</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Teacher</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Course</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Modes</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {allTeachers.map((teacher, idx) => (
                        <tr
                          key={`${teacher.teacher_id}|${teacher.course_id}|${idx}`}
                          className="hover:bg-gray-50 cursor-pointer"
                          onClick={() => {
                            const params = new URLSearchParams()
                            params.set('teacher_id', teacher.teacher_id || '')
                            params.set('course_id', teacher.course_id || '')
                            params.set('teacher_name', teacher.teacher_name || '')
                            params.set('course_code', teacher.course_code || '')
                            params.set('course_name', teacher.course_name || '')
                            navigate(`/mark-entry-windows/${windowData.window_id}/teacher-details?${params.toString()}`)
                          }}
                        >
                          <td className="px-4 py-3 text-sm">
                            <div className="font-medium text-gray-800">{teacher.teacher_name}</div>
                            <div className="text-xs text-gray-500">{teacher.teacher_id}</div>
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-700">
                            {teacher.course_code ? `${teacher.course_code} - ${teacher.course_name || ''}` : (teacher.course_name || '-')}
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-700">
                            {Array.isArray(teacher.learning_modes) && teacher.learning_modes.length > 0
                              ? teacher.learning_modes.join(', ')
                              : '-'}
                          </td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`inline-flex items-center px-2 py-1 rounded border text-xs font-medium ${getStatusBadge(teacher.status)}`}>
                              {teacher.status}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </>
        ) : null}
      </div>
    </MainLayout>
  )
}

export default MarkEntryWindowDetailsPage
