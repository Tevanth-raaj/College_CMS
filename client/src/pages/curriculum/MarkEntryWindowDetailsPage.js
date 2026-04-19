import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryWindowDetailsPage() {
  const { windowId } = useParams()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [windowData, setWindowData] = useState(null)
  const [userSummary, setUserSummary] = useState(null)
  const [selectedDepartmentId, setSelectedDepartmentId] = useState('')

  const requestedScope = (searchParams.get('scope') || '').trim().toLowerCase()
  const scopeType = (windowData?.scope_type || '').trim().toLowerCase()
  const isUserScope = scopeType === 'user_single' || scopeType === 'user_multi' || requestedScope === 'user'

  const departmentOptions = useMemo(() => {
    const sourceIds = Array.isArray(windowData?.department_ids) ? windowData.department_ids : []
    const sourceNames = Array.isArray(windowData?.department_names) ? windowData.department_names : []

    if (sourceIds.length > 0) {
      return sourceIds.map((id, index) => ({
        department_id: id,
        department_name: sourceNames[index] || `Department ${id}`,
      }))
    }

    if (Array.isArray(userSummary?.department_options) && userSummary.department_options.length > 0) {
      return userSummary.department_options
    }

    if (windowData?.department_id && windowData?.department_name) {
      return [{ department_id: windowData.department_id, department_name: windowData.department_name }]
    }

    return []
  }, [windowData, userSummary])

  const requiresDepartmentSelection = departmentOptions.length > 1

  useEffect(() => {
    const fetchWindowDetails = async () => {
      setLoading(true)
      setError('')
      try {
        const params = new URLSearchParams()
        params.set('window_id', windowId)
        if (selectedDepartmentId) {
          params.set('department_id', selectedDepartmentId)
        }

        const res = await fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?${params.toString()}`)
        if (!res.ok) throw new Error('Failed to fetch window details')
        const data = await res.json()
        const details = Array.isArray(data) && data.length > 0 ? data[0] : null
        if (!details) {
          setError('Window details not found')
          setWindowData(null)
        } else {
          setWindowData(details)
        }

        const detailsScopeType = (details?.scope_type || '').trim().toLowerCase()
        const detailsIsUserScope = detailsScopeType === 'user_single' || detailsScopeType === 'user_multi' || requestedScope === 'user'
        if (detailsIsUserScope && windowId) {
          const userParams = new URLSearchParams()
          if (selectedDepartmentId) {
            userParams.set('department_id', selectedDepartmentId)
          }
          const userRes = await fetch(`${API_BASE_URL}/mark-entry-windows/${windowId}/user-submissions?${userParams.toString()}`)
          if (!userRes.ok) throw new Error('Failed to fetch user-scope details')
          const userData = await userRes.json()
          setUserSummary(userData)
        } else {
          setUserSummary(null)
        }
      } catch (err) {
        setError(err.message || 'Failed to load window details')
        setWindowData(null)
        setUserSummary(null)
      } finally {
        setLoading(false)
      }
    }

    if (windowId) fetchWindowDetails()
  }, [windowId, requestedScope, selectedDepartmentId])

  useEffect(() => {
    if (!requiresDepartmentSelection) {
      if (departmentOptions.length === 1 && String(departmentOptions[0].department_id) !== selectedDepartmentId) {
        setSelectedDepartmentId(String(departmentOptions[0].department_id))
      }
      return
    }

    if (selectedDepartmentId) {
      return
    }
  }, [requiresDepartmentSelection, departmentOptions, selectedDepartmentId])

  const allTeachers = useMemo(() => {
    if (!windowData || isUserScope) return []
    const pending = Array.isArray(windowData.pending_teachers) ? windowData.pending_teachers : []
    const completed = Array.isArray(windowData.completed_teachers) ? windowData.completed_teachers : []
    return [
      ...pending.map(t => ({ ...t, status: 'Pending' })),
      ...completed.map(t => ({ ...t, status: 'Completed' })),
    ]
  }, [windowData, isUserScope])

  const allUserEntries = useMemo(() => {
    if (!isUserScope) return []
    const pending = Array.isArray(userSummary?.pending_users) ? userSummary.pending_users : []
    const completed = Array.isArray(userSummary?.completed_users) ? userSummary.completed_users : []
    return [
      ...pending.map((entry) => ({ ...entry, status: 'Pending' })),
      ...completed.map((entry) => ({ ...entry, status: 'Completed' })),
    ]
  }, [isUserScope, userSummary])

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
    <MainLayout title="Window Details" subtitle={isUserScope ? 'User scope: track submissions and entered marks' : 'Teachers involved in this mark entry window'}>
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
                  {isUserScope && (
                    <p className="text-xs text-gray-500 mt-1">User: {windowData.user_name || userSummary?.user_name || windowData.user_id}</p>
                  )}
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
                  <div className="text-gray-500">{isUserScope ? 'User Course Entries' : 'Teachers Involved'}</div>
                  <div className="font-semibold text-gray-800 mt-0.5">{isUserScope ? allUserEntries.length : allTeachers.length}</div>
                </div>
              </div>

              {requiresDepartmentSelection && (
                <div className="mt-4 pt-4 border-t border-gray-100">
                  <label className="block text-xs font-semibold text-gray-600 mb-2">Select Department</label>
                  <select
                    value={selectedDepartmentId}
                    onChange={(e) => setSelectedDepartmentId(e.target.value)}
                    className="w-full md:w-80 border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="">Choose department</option>
                    {departmentOptions.map((department) => (
                      <option key={department.department_id} value={String(department.department_id)}>
                        {department.department_name || `Department ${department.department_id}`}
                      </option>
                    ))}
                  </select>
                </div>
              )}
            </div>

            <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
              <div className="border-b border-gray-200 px-6 py-4">
                <h4 className="text-sm font-semibold text-gray-700">
                  {isUserScope ? 'Courses for User Mark Entry' : 'Teachers for Mark Entry'}
                </h4>
              </div>

              {requiresDepartmentSelection && !selectedDepartmentId ? (
                <div className="p-6 text-sm text-gray-500">Select a department to view submissions.</div>
              ) : (isUserScope ? allUserEntries.length === 0 : allTeachers.length === 0) ? (
                <div className="p-6 text-sm text-gray-500">
                  {isUserScope ? 'No user-course entries found for this window.' : 'No teachers found for this window.'}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">{isUserScope ? 'User' : 'Teacher'}</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Course</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Modes</th>
                        {isUserScope && <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Progress</th>}
                        <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {(isUserScope ? allUserEntries : allTeachers).map((teacher, idx) => (
                        <tr
                          key={`${isUserScope ? (teacher.user_id || windowData.user_id || 'user') : (teacher.teacher_id || 'teacher')}|${teacher.course_id}|${idx}`}
                          className="hover:bg-gray-50 cursor-pointer"
                          onClick={() => {
                            const params = new URLSearchParams()
                            if (isUserScope) {
                              params.set('scope', 'user')
                              params.set('user_id', String(teacher.user_id || windowData.user_id || ''))
                              params.set('user_name', teacher.user_name || windowData.user_name || '')
                            } else {
                              params.set('teacher_id', teacher.teacher_id || '')
                              params.set('teacher_name', teacher.teacher_name || '')
                            }
                            params.set('course_id', teacher.course_id || '')
                            params.set('course_code', teacher.course_code || '')
                            params.set('course_name', teacher.course_name || '')
                            if (selectedDepartmentId) {
                              params.set('department_id', selectedDepartmentId)
                            }
                            navigate(`/mark-entry-windows/${windowData.window_id}/teacher-details?${params.toString()}`)
                          }}
                        >
                          <td className="px-4 py-3 text-sm">
                            <div className="font-medium text-gray-800">{isUserScope ? (teacher.user_name || windowData.user_name || '-') : teacher.teacher_name}</div>
                            <div className="text-xs text-gray-500">{isUserScope ? (teacher.user_id || windowData.user_id || '-') : teacher.teacher_id}</div>
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-700">
                            {teacher.course_code ? `${teacher.course_code} - ${teacher.course_name || ''}` : (teacher.course_name || '-')}
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-700">
                            {Array.isArray(teacher.learning_modes) && teacher.learning_modes.length > 0
                              ? teacher.learning_modes.join(', ')
                              : '-'}
                          </td>
                          {isUserScope && (
                            <td className="px-4 py-3 text-sm text-gray-700">
                              {Number(teacher.updated_students || 0)} / {Number(teacher.assigned_students || 0)} updated
                            </td>
                          )}
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
