import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MyAssignedStudentsPage() {
  const navigate = useNavigate()
  const userId = localStorage.getItem('username') || localStorage.getItem('teacherId')
  const [assignedStudents, setAssignedStudents] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [groupedByCourse, setGroupedByCourse] = useState({})

  useEffect(() => {
    if (userId) {
      loadAssignedStudents()
    } else {
      setError('User ID not found. Please login again.')
      setLoading(false)
    }
  }, [userId])

  const loadAssignedStudents = async () => {
    setLoading(true)
    setError('')

    try {
      const response = await fetch(
        `${API_BASE_URL}/mark-entry/user-assigned-students?user_id=${userId}`
      )

      if (!response.ok) {
        throw new Error('Failed to load assigned students')
      }

      const data = await response.json()
      
      setAssignedStudents(Array.isArray(data) ? data : [])

      // Group by course
      const grouped = {}
      if (Array.isArray(data)) {
        data.forEach((student) => {
          const key = student.course_id || 'no_course'
          if (!grouped[key]) {
            grouped[key] = {
              course_id: student.course_id,
              course_name: student.course_name || 'General Assignment',
              window_start: student.window_start,
              window_end: student.window_end,
              window_id: student.window_id,
              students: []
            }
          }
          grouped[key].students.push(student)
        })
      }

      setGroupedByCourse(grouped)
    } catch (err) {
      console.error('[ASSIGNED STUDENTS] Error:', err)
      setError(err.message || 'Failed to load assigned students')
    } finally {
      setLoading(false)
    }
  }

  const goToMarkEntry = (courseId) => {
    navigate(`/teacher/courses/${courseId}/students`)
  }

  const getTimeRemaining = (endTime) => {
    const now = new Date()
    const end = new Date(endTime)
    const diff = end - now
    
    if (diff <= 0) return 'Expired'
    
    const days = Math.floor(diff / (1000 * 60 * 60 * 24))
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
    
    if (days > 0) {
      return `${days} day${days > 1 ? 's' : ''} ${hours}h remaining`
    }
    return `${hours} hour${hours > 1 ? 's' : ''} remaining`
  }

  if (loading) {
    return (
      <MainLayout title="My Assigned Students" subtitle="Students assigned for mark entry">
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading assigned students...</p>
          </div>
        </div>
      </MainLayout>
    )
  }

  if (error) {
    return (
      <MainLayout title="My Assigned Students" subtitle="Students assigned for mark entry">
        <div className="bg-red-50 border border-red-200 rounded-lg p-6">
          <p className="text-red-700">{error}</p>
        </div>
      </MainLayout>
    )
  }

  if (assignedStudents.length === 0) {
    return (
      <MainLayout title="My Assigned Students" subtitle="Students assigned for mark entry">
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-12">
          <div className="text-center">
            <svg
              className="mx-auto h-16 w-16 text-gray-400 mb-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
              />
            </svg>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              No Students Assigned
            </h3>
            <p className="text-gray-600 mb-4">
              You currently have no students assigned for mark entry.
            </p>
            <p className="text-sm text-gray-500">
              Please contact the Controller of Examinations if you believe this is an error.
            </p>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout title="My Assigned Students" subtitle="Students assigned for mark entry">
      <div className="space-y-6">
        {/* Summary Card */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="bg-blue-50 border border-blue-200 rounded-xl p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-blue-600">Total Students</p>
                <p className="text-2xl font-bold text-blue-900">{assignedStudents.length}</p>
              </div>
            </div>
          </div>

          <div className="bg-green-50 border border-green-200 rounded-xl p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                </svg>
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-green-600">Courses</p>
                <p className="text-2xl font-bold text-green-900">{Object.keys(groupedByCourse).length}</p>
              </div>
            </div>
          </div>

          <div className="bg-purple-50 border border-purple-200 rounded-xl p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <svg className="h-8 w-8 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-purple-600">Active Windows</p>
                <p className="text-2xl font-bold text-purple-900">
                  {Object.values(groupedByCourse).filter(g => new Date(g.window_end) > new Date()).length}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Course-wise Student List */}
        {Object.values(groupedByCourse).map((courseGroup) => {
          const isActive = new Date(courseGroup.window_end) > new Date()
          const isExpired = new Date(courseGroup.window_end) <= new Date()
          
          return (
            <div key={courseGroup.course_id} className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
              {/* Course Header */}
              <div className={`px-6 py-4 border-b ${isExpired ? 'bg-gray-50' : 'bg-gradient-to-r from-blue-50 to-indigo-50'}`}>
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-gray-900 mb-1">
                      {courseGroup.course_name}
                    </h3>
                    <div className="flex items-center gap-4 text-sm text-gray-600">
                      <div className="flex items-center gap-1">
                        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                        <span>
                          {new Date(courseGroup.window_start).toLocaleDateString()} - {new Date(courseGroup.window_end).toLocaleDateString()}
                        </span>
                      </div>
                      <div className="flex items-center gap-1">
                        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <span className={isExpired ? 'text-red-600 font-medium' : 'text-green-600 font-medium'}>
                          {getTimeRemaining(courseGroup.window_end)}
                        </span>
                      </div>
                    </div>
                  </div>
                  {isActive && (
                    <button
                      onClick={() => goToMarkEntry(courseGroup.course_id)}
                      className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                    >
                      Enter Marks
                    </button>
                  )}
                  {isExpired && (
                    <span className="px-4 py-2 bg-red-100 text-red-700 rounded-lg font-medium text-sm">
                      Window Expired
                    </span>
                  )}
                </div>
              </div>

              {/* Students Table */}
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                        Enrollment No
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                        Student Name
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                        Department
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                        Year
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {courseGroup.students.map((student, index) => (
                      <tr key={student.student_id} className={index % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                          {student.enrollment_no}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {student.student_name}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                          {student.department}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                          {student.year}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Footer */}
              <div className="bg-gray-50 px-6 py-3 border-t border-gray-200">
                <p className="text-sm text-gray-600">
                  Total: <span className="font-semibold text-gray-900">{courseGroup.students.length}</span> student(s) assigned
                </p>
              </div>
            </div>
          )
        })}
      </div>
    </MainLayout>
  )
}

export default MyAssignedStudentsPage
