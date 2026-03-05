import React, { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function TeacherCourseStudentsPage() {
  const { courseId } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const [course, setCourse] = useState(location.state?.course || null)
  const [students, setStudents] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Appeal / submission state for expired windows
  const [submissionInfo, setSubmissionInfo] = useState(null)   // {submitted, submitted_at?}
  const [appealInfo, setAppealInfo] = useState(null)           // appeal object or null
  const [showAppealForm, setShowAppealForm] = useState(false)
  const [appealReason, setAppealReason] = useState('')
  const [submittingAppeal, setSubmittingAppeal] = useState(false)
  const [appealError, setAppealError] = useState('')

  useEffect(() => {
    if (courseId) {
      fetchCourseStudents()
    }
  }, [courseId])

  const fetchCourseStudents = async () => {
    setLoading(true)
    setError('')

    try {
      const teacherID = localStorage.getItem('teacherId') || localStorage.getItem('teacher_id')
      if (!teacherID) {
        setError('Teacher ID not found. Please login again.')
        setLoading(false)
        return
      }
      
      const url = `${API_BASE_URL}/teachers/${teacherID}/courses`
      const response = await fetch(url)

      if (!response.ok) {
        throw new Error('Failed to fetch course details')
      }

      const data = await response.json()
      const foundCourse = data.find(c => c.id === parseInt(courseId))

      if (foundCourse) {
        setCourse(foundCourse)
        setStudents(foundCourse.enrollments || [])
        if (foundCourse.has_window && foundCourse.window) {
          // Seed submission info directly from the course payload (already computed by the server)
          setSubmissionInfo({
            submitted: foundCourse.is_submitted === true,
            submitted_at: foundCourse.submitted_at || null,
          })
          // Always fetch appeal status using the correct course_id (courses.id, not allocation row id)
          fetchAppealStatus(teacherID, foundCourse.course_id, foundCourse.window.id)
        } else if (!foundCourse.has_window && foundCourse.submitted_expired_window) {
          // Window expired but teacher DID submit — seed submission info + fetch appeal status
          setSubmissionInfo({
            submitted: true,
            submitted_at: foundCourse.submitted_expired_window.submitted_at || null,
          })
          fetchAppealStatus(teacherID, foundCourse.course_id, foundCourse.submitted_expired_window.id)
        } else if (!foundCourse.has_window && foundCourse.missed_window) {
          // Expired window — fetch both submission + appeal status
          fetchExpiredWindowStatus(teacherID, foundCourse.course_id, foundCourse.missed_window.id)
        }
      } else {
        setError('Course not found')
      }
    } catch (err) {
      console.error('Error fetching course students:', err)
      setError(err.message || 'Failed to fetch students')
    } finally {
      setLoading(false)
    }
  }

  // Fetch only the appeal status for an active window (submission already known from course payload)
  const fetchAppealStatus = async (teacherID, courseId, windowId) => {
    try {
      const appealRes = await fetch(
        `${API_BASE_URL}/mark-appeals?teacher_id=${encodeURIComponent(teacherID)}&course_id=${courseId}&window_id=${windowId}`
      )
      if (appealRes.ok) {
        const appealData = await appealRes.json()
        if (appealData && appealData.length > 0) {
          setAppealInfo(appealData[0])
        }
      }
    } catch (err) {
      console.error('Error fetching appeal status:', err)
    }
  }

  const fetchExpiredWindowStatus = async (teacherID, cId, windowId) => {
    try {
      // Check submission
      const subRes = await fetch(
        `${API_BASE_URL}/mark-submissions/check?teacher_id=${encodeURIComponent(teacherID)}&course_id=${cId}&window_id=${windowId}`
      )
      if (subRes.ok) {
        const subData = await subRes.json()
        setSubmissionInfo(subData)
      }
      // Check existing appeal
      const appealRes = await fetch(
        `${API_BASE_URL}/mark-appeals?teacher_id=${encodeURIComponent(teacherID)}&course_id=${cId}&window_id=${windowId}`
      )
      if (appealRes.ok) {
        const appealData = await appealRes.json()
        if (appealData && appealData.length > 0) {
          setAppealInfo(appealData[0])
        }
      }
    } catch (err) {
      console.error('Error fetching expired window status:', err)
    }
  }

  // Returns the window ID to use when submitting/refreshing an appeal
  const getAppealWindowId = () => {
    if (course?.has_window && course?.window) return course.window.id
    if (course?.submitted_expired_window) return course.submitted_expired_window.id
    if (course?.missed_window) return course.missed_window.id
    return null
  }

  const handleSubmitAppeal = async () => {
    if (!appealReason.trim()) {
      setAppealError('Please enter a reason for the appeal.')
      return
    }
    setAppealError('')
    setSubmittingAppeal(true)
    try {
      const teacherID = localStorage.getItem('teacherId') || localStorage.getItem('teacher_id')
      const windowId = getAppealWindowId()
      const res = await fetch(`${API_BASE_URL}/mark-appeals`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          teacher_id: teacherID,
          course_id: course.course_id,
          window_id: windowId,
          reason: appealReason.trim(),
        }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Failed to submit appeal')
      }
      // Refresh appeal info only (submission status unchanged)
      await fetchAppealStatus(teacherID, course.course_id, windowId)
      setShowAppealForm(false)
      setAppealReason('')
    } catch (err) {
      setAppealError(err.message || 'Failed to submit appeal')
    } finally {
      setSubmittingAppeal(false)
    }
  }

  const formatDateTime = (dateString) => {
    const date = new Date(dateString)
    return date.toLocaleString('en-IN', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true
    })
  }

  const isWindowActive = (window) => {
    const now = new Date()
    const startDate = new Date(window.start_at)
    const endDate = new Date(window.end_at)
    return now >= startDate && now <= endDate
  }

  if (loading) {
    return (
      <MainLayout title="Course Students" subtitle="Loading...">
        <div className="flex justify-center items-center py-20">
          <div className="text-center">
            <svg className="animate-spin h-12 w-12 text-blue-600 mx-auto mb-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-gray-600">Loading students...</p>
          </div>
        </div>
      </MainLayout>
    )
  }

  if (error) {
    return (
      <MainLayout title="Course Students" subtitle="Error">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <p className="text-red-600">{error}</p>
          <button
            onClick={() => navigate('/teacher-dashboard')}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Back to Dashboard
          </button>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout
      title={course?.course_name || 'Course Students'}
      subtitle={`${course?.course_code || ''} • ${students.length} student${students.length !== 1 ? 's' : ''}`}
      actions={
        <button
          onClick={() => navigate('/teacher-dashboard')}
          className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors flex items-center space-x-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          <span>Back to Dashboard</span>
        </button>
      }
    >
      <div className="space-y-6">
        {/* Mark Entry Window Section */}
        {course?.has_window && course?.window && (() => {
          const active = isWindowActive(course.window)
          const submittedActive = active && submissionInfo?.submitted === true
          // Outer card colour
          const cardClass = submittedActive
            ? 'bg-green-50 border-green-500'
            : active
              ? 'bg-purple-50 border-[#7D53F6]'
              : 'bg-gray-50 border-gray-400'
          // Icon colour
          const iconClass = submittedActive ? 'text-green-600' : active ? 'text-[#7D53F6]' : 'text-gray-500'
          // Heading colour
          const headingClass = submittedActive ? 'text-green-900' : active ? 'text-purple-900' : 'text-gray-700'
          // Heading text
          const headingText = submittedActive
            ? 'Marks Submitted'
            : active
              ? 'Active Mark Entry Window'
              : 'Expired Mark Entry Window'
          // Inner card border
          const innerBorder = submittedActive ? 'border-green-200' : active ? 'border-purple-200' : 'border-gray-200'
          return (
          <div className={`${cardClass} border-l-4 p-4 rounded-lg`}>
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className={`h-6 w-6 ${iconClass}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  {submittedActive || active ? (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  ) : (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  )}
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <h3 className={`text-lg font-semibold mb-3 ${headingClass}`}>
                  {headingText}
                </h3>
                <div className={`bg-white p-4 rounded-lg shadow-sm border ${innerBorder}`}>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-gray-600">Window ID</p>
                      <p className="text-sm font-semibold text-gray-900">#{course.window.id}</p>
                    </div>
                    {course.window.department_name && (
                      <div>
                        <p className="text-sm text-gray-600">Department</p>
                        <p className="text-sm font-semibold text-gray-900">{course.window.department_name}</p>
                      </div>
                    )}
                    <div>
                      <p className="text-sm text-gray-600">Start Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(course.window.start_at)}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">End Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(course.window.end_at)}</p>
                    </div>
                    {course.window.semester && (
                      <div>
                        <p className="text-sm text-gray-600">Semester</p>
                        <p className="text-sm font-semibold text-gray-900">{course.window.semester}</p>
                      </div>
                    )}
                    {course.window.component_names && course.window.component_names.length > 0 && (
                      <div className="col-span-2">
                        <p className="text-sm text-gray-600">Assessment Components</p>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {/* Remove duplicates and render unique component names */}
                          {[...new Set(course.window.component_names)].map((compName, idx) => (
                            <span key={idx} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                              {compName}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                  {submittedActive ? (
                    /* ✅ Active window — marks already submitted → Appeal mode */
                    <div className="mt-3 space-y-3">
                      {/* Submission confirmation banner */}
                      <div className="flex items-center space-x-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg">
                        <svg className="h-5 w-5 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <div>
                          <p className="text-sm font-semibold text-green-800">Marks Submitted Successfully</p>
                          {submissionInfo.submitted_at && (
                            <p className="text-xs text-green-600">Submitted on {formatDateTime(submissionInfo.submitted_at)}</p>
                          )}
                        </div>
                      </div>

                      {/* Appeal section */}
                      {appealInfo?.status === 'pending' ? (
                          <div className="flex items-center space-x-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg">
                            <svg className="h-5 w-5 text-amber-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <div>
                              <p className="text-sm font-semibold text-amber-800">Appeal Submitted — Awaiting Review</p>
                              <p className="text-xs text-amber-600">Submitted on {formatDateTime(appealInfo.created_at)}</p>
                            </div>
                          </div>
                      ) : showAppealForm ? (
                        /* Appeal form */
                        <div className="space-y-2">
                          <p className="text-xs text-gray-500">Describe why you need to amend your submitted marks:</p>
                          <textarea
                            rows={3}
                            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 resize-none"
                            placeholder="Briefly explain why you need to amend the submission…"
                            value={appealReason}
                            onChange={e => setAppealReason(e.target.value)}
                          />
                          {appealError && <p className="text-xs text-red-600">{appealError}</p>}
                          <div className="flex space-x-2">
                            <button
                              onClick={handleSubmitAppeal}
                              disabled={submittingAppeal}
                              className="flex-1 px-3 py-2 bg-amber-500 text-white text-sm rounded-lg hover:bg-amber-600 disabled:opacity-60 transition-colors"
                            >
                              {submittingAppeal ? 'Submitting…' : 'Submit Appeal'}
                            </button>
                            <button
                              onClick={() => { setShowAppealForm(false); setAppealReason(''); setAppealError('') }}
                              className="px-3 py-2 bg-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-300 transition-colors"
                            >
                              Cancel
                            </button>
                          </div>
                        </div>
                      ) : (
                        <button
                          onClick={() => setShowAppealForm(true)}
                          className="w-full px-3 py-2 bg-amber-100 border border-amber-300 text-amber-800 text-sm font-medium rounded-lg hover:bg-amber-200 transition-colors flex items-center justify-center space-x-2"
                        >
                          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                          </svg>
                          <span>Appeal Submission</span>
                        </button>
                      )}
                    </div>
                  ) : active ? (
                    <button
                      onClick={() => navigate('/mark-entry')}
                      className="mt-3 w-full px-4 py-2 bg-[#7D53F6] text-white rounded-lg hover:bg-purple-700 transition-colors"
                    >
                      Enter Marks
                    </button>
                  ) : (
                    /* Expired window — show submission / appeal status */
                    <div className="mt-3 space-y-3">
                      {submissionInfo === null ? (
                        <div className="flex items-center justify-center py-2">
                          <svg className="animate-spin h-5 w-5 text-gray-400 mr-2" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
                          </svg>
                          <span className="text-xs text-gray-400">Checking submission status…</span>
                        </div>
                      ) : submissionInfo.submitted ? (
                        /* ✅ Submitted */
                        <div className="flex items-center space-x-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg">
                          <svg className="h-5 w-5 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          <div>
                            <p className="text-sm font-semibold text-green-800">Marks Submitted</p>
                            {submissionInfo.submitted_at && (
                              <p className="text-xs text-green-600">
                                Submitted on {formatDateTime(submissionInfo.submitted_at)}
                              </p>
                            )}
                          </div>
                        </div>
                      ) : (
                        /* ❌ Missed submission */
                        <>
                          <div className="flex items-center space-x-2 px-3 py-2 bg-red-50 border border-red-200 rounded-lg">
                            <svg className="h-5 w-5 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                            </svg>
                            <p className="text-sm font-semibold text-red-700">Missed Submission — Window Expired</p>
                          </div>

                          {/* Appeal status */}
                          {appealInfo ? (
                            appealInfo.status === 'pending' ? (
                              <div className="flex items-center space-x-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg">
                                <svg className="h-5 w-5 text-amber-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                <div>
                                  <p className="text-sm font-semibold text-amber-800">Appeal Submitted — Awaiting Review</p>
                                  <p className="text-xs text-amber-600">Submitted on {formatDateTime(appealInfo.created_at)}</p>
                                </div>
                              </div>
                            ) : appealInfo.status === 'resolved' ? (
                              <div className="flex items-center space-x-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg">
                                <svg className="h-5 w-5 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                <div>
                                  <p className="text-sm font-semibold text-green-800">Appeal Approved — You can now enter marks</p>
                                  <button
                                    onClick={() => navigate('/mark-entry')}
                                    className="mt-1 text-xs text-green-700 underline hover:text-green-900"
                                  >
                                    Go to Mark Entry →
                                  </button>
                                </div>
                              </div>
                            ) : (
                              <div className="flex items-center space-x-2 px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg">
                                <svg className="h-5 w-5 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                </svg>
                                <p className="text-sm font-semibold text-gray-600">Appeal Rejected</p>
                              </div>
                            )
                          ) : showAppealForm ? (
                            /* Appeal form */
                            <div className="space-y-2">
                              <textarea
                                rows={3}
                                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 resize-none"
                                placeholder="Briefly explain why you missed the submission…"
                                value={appealReason}
                                onChange={e => setAppealReason(e.target.value)}
                              />
                              {appealError && <p className="text-xs text-red-600">{appealError}</p>}
                              <div className="flex space-x-2">
                                <button
                                  onClick={handleSubmitAppeal}
                                  disabled={submittingAppeal}
                                  className="flex-1 px-3 py-2 bg-amber-500 text-white text-sm rounded-lg hover:bg-amber-600 disabled:opacity-60 transition-colors"
                                >
                                  {submittingAppeal ? 'Submitting…' : 'Submit Appeal'}
                                </button>
                                <button
                                  onClick={() => { setShowAppealForm(false); setAppealReason(''); setAppealError('') }}
                                  className="px-3 py-2 bg-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-300 transition-colors"
                                >
                                  Cancel
                                </button>
                              </div>
                            </div>
                          ) : (
                            <button
                              onClick={() => setShowAppealForm(true)}
                              className="w-full px-3 py-2 bg-amber-100 border border-amber-300 text-amber-800 text-sm font-medium rounded-lg hover:bg-amber-200 transition-colors flex items-center justify-center space-x-2"
                            >
                              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                              </svg>
                              <span>Appeal Missed Submission</span>
                            </button>
                          )}
                        </>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
          )
        })()}

        {/* Expired + Submitted Window — post-expiry appeal path */}
        {!course?.has_window && course?.submitted_expired_window && (() => {          const win = course.submitted_expired_window
          return (
          <div className="bg-green-50 border-green-500 border-l-4 p-4 rounded-lg">
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <h3 className="text-lg font-semibold mb-3 text-green-900">
                  Marks Submitted (Window Expired)
                </h3>
                <div className="bg-white p-4 rounded-lg shadow-sm border border-green-200">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-gray-600">Window ID</p>
                      <p className="text-sm font-semibold text-gray-900">#{win.id}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">Status</p>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700 border border-gray-200">Expired</span>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">Start Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(win.start_at)}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">End Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(win.end_at)}</p>
                    </div>
                  </div>
                  <div className="mt-3 space-y-3">
                    {/* Submission confirmation banner */}
                    <div className="flex items-center space-x-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg">
                      <svg className="h-5 w-5 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <div>
                        <p className="text-sm font-semibold text-green-800">Marks Submitted Successfully</p>
                        {win.submitted_at && (
                          <p className="text-xs text-green-600">Submitted on {formatDateTime(win.submitted_at)}</p>
                        )}
                      </div>
                    </div>

                    {/* Appeal section */}
                    {appealInfo?.status === 'pending' ? (
                      <div className="flex items-center space-x-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg">
                        <svg className="h-5 w-5 text-amber-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <div>
                          <p className="text-sm font-semibold text-amber-800">Appeal Submitted — Awaiting Review</p>
                          <p className="text-xs text-amber-600">Submitted on {formatDateTime(appealInfo.created_at)}</p>
                        </div>
                      </div>
                    ) : showAppealForm ? (
                      <div className="space-y-2">
                        <p className="text-xs text-gray-500">Describe why you need to amend your submitted marks:</p>
                        <textarea
                          rows={3}
                          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 resize-none"
                          placeholder="Briefly explain why you need to amend the submission…"
                          value={appealReason}
                          onChange={e => setAppealReason(e.target.value)}
                        />
                        {appealError && <p className="text-xs text-red-600">{appealError}</p>}
                        <div className="flex space-x-2">
                          <button
                            onClick={handleSubmitAppeal}
                            disabled={submittingAppeal}
                            className="flex-1 px-3 py-2 bg-amber-500 text-white text-sm rounded-lg hover:bg-amber-600 disabled:opacity-60 transition-colors"
                          >
                            {submittingAppeal ? 'Submitting…' : 'Submit Appeal'}
                          </button>
                          <button
                            onClick={() => { setShowAppealForm(false); setAppealReason(''); setAppealError('') }}
                            className="px-3 py-2 bg-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-300 transition-colors"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : (
                      <button
                        onClick={() => setShowAppealForm(true)}
                        className="w-full px-3 py-2 bg-amber-100 border border-amber-300 text-amber-800 text-sm font-medium rounded-lg hover:bg-amber-200 transition-colors flex items-center justify-center space-x-2"
                      >
                        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                        </svg>
                        <span>Appeal Submission</span>
                      </button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
          )
        })()}

        {/* Missed Submission — window expired and teacher did not submit */}
        {!course?.has_window && course?.missed_window && !course?.submitted_expired_window && (() => {
          const win = course.missed_window
          return (
          <div className="bg-red-50 border-red-500 border-l-4 p-4 rounded-lg">
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className="h-6 w-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <h3 className="text-lg font-semibold mb-3 text-red-900">
                  Missed Submission — Window Expired
                </h3>
                <div className="bg-white p-4 rounded-lg shadow-sm border border-red-200">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-gray-600">Window ID</p>
                      <p className="text-sm font-semibold text-gray-900">#{win.id}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">Status</p>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-700 border border-red-200">Expired</span>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">Start Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(win.start_at)}</p>
                    </div>
                    <div>
                      <p className="text-sm text-gray-600">End Date</p>
                      <p className="text-sm font-semibold text-gray-900">{formatDateTime(win.end_at)}</p>
                    </div>
                  </div>
                  <div className="mt-3 space-y-3">
                    <div className="flex items-center space-x-2 px-3 py-2 bg-red-50 border border-red-200 rounded-lg">
                      <svg className="h-5 w-5 text-red-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      <p className="text-sm font-semibold text-red-700">Marks Not Submitted Before Deadline</p>
                    </div>
                    {/* Appeal section */}
                    {appealInfo?.status === 'pending' ? (
                      <div className="flex items-center space-x-2 px-3 py-2 bg-amber-50 border border-amber-200 rounded-lg">
                        <svg className="h-5 w-5 text-amber-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <div>
                          <p className="text-sm font-semibold text-amber-800">Appeal Submitted — Awaiting Review</p>
                          <p className="text-xs text-amber-600">Submitted on {formatDateTime(appealInfo.created_at)}</p>
                        </div>
                      </div>
                    ) : appealInfo?.status === 'resolved' ? (
                      <div className="flex items-center space-x-2 px-3 py-2 bg-green-50 border border-green-200 rounded-lg">
                        <svg className="h-5 w-5 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <div>
                          <p className="text-sm font-semibold text-green-800">Appeal Approved — You can now enter marks</p>
                          <button
                            onClick={() => navigate('/mark-entry')}
                            className="mt-1 text-xs text-green-700 underline hover:text-green-900"
                          >
                            Go to Mark Entry →
                          </button>
                        </div>
                      </div>
                    ) : appealInfo?.status === 'rejected' ? (
                      <div className="flex items-center space-x-2 px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg">
                        <svg className="h-5 w-5 text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                        <p className="text-sm font-semibold text-gray-600">Appeal Rejected</p>
                      </div>
                    ) : showAppealForm ? (
                      <div className="space-y-2">
                        <p className="text-xs text-gray-500">Explain why you missed the submission deadline:</p>
                        <textarea
                          rows={3}
                          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 resize-none"
                          placeholder="Briefly explain why you missed the submission…"
                          value={appealReason}
                          onChange={e => setAppealReason(e.target.value)}
                        />
                        {appealError && <p className="text-xs text-red-600">{appealError}</p>}
                        <div className="flex space-x-2">
                          <button
                            onClick={handleSubmitAppeal}
                            disabled={submittingAppeal}
                            className="flex-1 px-3 py-2 bg-amber-500 text-white text-sm rounded-lg hover:bg-amber-600 disabled:opacity-60 transition-colors"
                          >
                            {submittingAppeal ? 'Submitting…' : 'Submit Appeal'}
                          </button>
                          <button
                            onClick={() => { setShowAppealForm(false); setAppealReason(''); setAppealError('') }}
                            className="px-3 py-2 bg-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-300 transition-colors"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : (
                      <button
                        onClick={() => setShowAppealForm(true)}
                        className="w-full px-3 py-2 bg-amber-100 border border-amber-300 text-amber-800 text-sm font-medium rounded-lg hover:bg-amber-200 transition-colors flex items-center justify-center space-x-2"
                      >
                        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                        </svg>
                        <span>Appeal Missed Submission</span>
                      </button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
          )
        })()}

        {/* No Window Message */}
        {!course?.has_window && !course?.submitted_expired_window && !course?.missed_window && (
          <div className="bg-blue-50 border-l-4 border-blue-400 p-4 rounded-lg">
            <div className="flex items-center">
              <svg className="h-6 w-6 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="ml-3 text-blue-700 font-medium">No mark entry window available for this course</p>
            </div>
          </div>
        )}

        {/* Students List */}
        {/* Course Info Card */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <p className="text-sm text-gray-600 mb-1">Course Code</p>
              <p className="text-lg font-semibold">{course?.course_code}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Course Type</p>
              <span className={`px-3 py-1 tracking-tighter uppercase rounded-full text-xs font-semibold ${course.course_type === 'theory'
                ? 'bg-blue-100 text-blue-700 border border-blue-200'
                : course.course_type === 'lab'
                  ? 'bg-green-100 text-green-700 border border-green-200'
                  : course.course_type === 'theory_with_lab'
                    ? 'bg-purple-100 text-purple-700 border border-purple-200'
                    : 'bg-gray-100 text-gray-700 border border-gray-200'
                }`}>
                {course.course_type === 'theory_with_lab' ? 'Theory+Lab' : course.course_type}
              </span>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Credits</p>
              <p className="text-lg font-semibold text-gray-900">{course?.credit}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Category</p>
              <p className="text-lg font-semibold text-gray-900">{course?.category}</p>
            </div>
          </div>
        </div>

        {/* Students List */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900">
              Allocated Students ({students.length})
            </h3>
          </div>

          {students.length > 0 ? (
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {students.map((student) => (
                  <div
                    key={student.student_id}
                    className="flex items-center space-x-4 p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors border border-gray-200"
                  >
                    <div
                      className="w-16 h-16 rounded-full bg-background flex items-center justify-center text-primary font-semibold text-2xl shadow-md flex-shrink-0 border-2 border-primary"
                    >
                      {student.student_name.charAt(0).toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-semibold text-gray-900 truncate">
                        {student.student_name}
                      </p>
                      <p className="text-xs text-gray-600 mt-1">
                        ID: {student.student_id}
                      </p>
                      {student.enrollment_no && (
                        <p className="text-xs text-gray-500 mt-1">
                          Enrollment: {student.enrollment_no}
                        </p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="p-12 text-center">
              <svg className="w-16 h-16 text-gray-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
              <p className="text-gray-500 text-lg font-medium">No students allocated yet</p>
              <p className="text-gray-400 text-sm mt-2">Students will appear here once they are allocated to this course</p>
            </div>
          )}
        </div>
      </div>
    </MainLayout>
  )
}

export default TeacherCourseStudentsPage
