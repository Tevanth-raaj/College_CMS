import React, { useState, useEffect } from 'react'
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
        {course?.has_window && course?.window && (
          <div className={`${isWindowActive(course.window) ? 'bg-purple-50 border-[#7D53F6]' : 'bg-gray-50 border-gray-400'} border-l-4 p-4 rounded-lg`}>
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className={`h-6 w-6 ${isWindowActive(course.window) ? 'text-[#7D53F6]' : 'text-gray-500'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  {isWindowActive(course.window) ? (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  ) : (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  )}
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <h3 className={`text-lg font-semibold mb-3 ${isWindowActive(course.window) ? 'text-purple-900' : 'text-gray-700'}`}>
                  {isWindowActive(course.window) ? 'Active Mark Entry Window' : 'Expired Mark Entry Window'}
                </h3>
                <div className={`bg-white p-4 rounded-lg shadow-sm border ${isWindowActive(course.window) ? 'border-purple-200' : 'border-gray-200'}`}>
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
                  {isWindowActive(course.window) && (
                    <button
                      onClick={() => navigate('/mark-entry')}
                      className="mt-3 w-full px-4 py-2 bg-[#7D53F6] text-white rounded-lg hover:bg-purple-700 transition-colors"
                    >
                      Enter Marks
                    </button>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* No Window Message */}
        {!course?.has_window && (
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
