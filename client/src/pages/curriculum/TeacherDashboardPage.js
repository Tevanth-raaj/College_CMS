import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'
import StatCard from '../../components/StatCard'

function TeacherDashboardPage() {
  const navigate = useNavigate()
  const teacherID = localStorage.getItem('teacherId') || localStorage.getItem('teacher_id')
  const teacherName = localStorage.getItem('teacher_name') || localStorage.getItem('userName')
  const username = localStorage.getItem('username')
  const storedUserId = Number(localStorage.getItem('userId') || localStorage.getItem('id') || 0)
  const userRole = localStorage.getItem('userRole')
  const userEmail = localStorage.getItem('userEmail')
  const [coursesByCategory, setCoursesByCategory] = useState({})
  const [userWindowCards, setUserWindowCards] = useState([])
  const [userWindowLoading, setUserWindowLoading] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')


  useEffect(() => {
    // If we have teacherID, fetch courses directly
    if (teacherID) {
      fetchTeacherCourses()
    } 
    // If teacher role but no teacherID, try to fetch from backend using email
    else if ((userRole === 'teacher' || userRole === 'hod') && userEmail) {
      fetchTeacherIdByEmail()
    } 
    // Otherwise show error
    else {
      setError('Unable to load teacher information. Please login again.')
      setLoading(false)
    }
  }, [teacherID, userRole, userEmail])

  useEffect(() => {
    fetchUserScopedWindowCards()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [username, teacherID, storedUserId])

  const isWindowActiveNow = (windowItem) => {
    if (!windowItem?.enabled) return false
    const now = new Date()
    const startAt = windowItem?.start_at ? new Date(windowItem.start_at) : null
    const endAt = windowItem?.end_at ? new Date(windowItem.end_at) : null
    if (!startAt || !endAt || Number.isNaN(startAt.getTime()) || Number.isNaN(endAt.getTime())) {
      return false
    }
    return now >= startAt && now <= endAt
  }

  const isUserMappedToWindow = (windowItem) => {
    const mappedUsernames = String(windowItem?.user_username || '')
      .split(',')
      .map((value) => value.trim().toLowerCase())
      .filter(Boolean)

    if (username && mappedUsernames.includes(String(username).toLowerCase())) {
      return true
    }

    const numericUserId = Number(windowItem?.user_id || 0)
    return Boolean(storedUserId > 0 && numericUserId > 0 && numericUserId === storedUserId)
  }

  const fetchWindowCourseCategories = async (courseId, teacherIdentifier) => {
    if (!courseId || !teacherIdentifier) return []
    try {
      const response = await fetch(
        `${API_BASE_URL}/course/${courseId}/mark-categories?teacher_id=${encodeURIComponent(teacherIdentifier)}&learning_modes=1,2`
      )
      if (!response.ok) return []
      const data = await response.json()
      return Array.isArray(data) ? data : []
    } catch {
      return []
    }
  }

  const fetchUserScopedWindowCards = async () => {
    const userIdentifier = username || teacherID || (storedUserId > 0 ? String(storedUserId) : '')
    if (!userIdentifier && storedUserId <= 0) {
      setUserWindowCards([])
      return
    }

    setUserWindowLoading(true)
    try {
      const safeReadJson = async (response) => {
        const text = await response.text()
        if (!text) return []
        try {
          return JSON.parse(text)
        } catch {
          return []
        }
      }

      const response = await fetch(`${API_BASE_URL}/mark-entry-windows?user_only=true`)
      if (!response.ok) {
        setUserWindowCards([])
        return
      }

      const windows = await safeReadJson(response)
      const userWindows = (Array.isArray(windows) ? windows : [])
        .filter((windowItem) => isWindowActiveNow(windowItem) && isUserMappedToWindow(windowItem))

      if (userWindows.length === 0 || !userIdentifier) {
        setUserWindowCards([])
        return
      }

      const windowCards = await Promise.all(
        userWindows.map(async (windowItem) => {
          try {
            const assignedResponse = await fetch(
              `${API_BASE_URL}/mark-entry/user-assigned-students?user_id=${encodeURIComponent(userIdentifier)}&window_id=${windowItem.id}`
            )

            const assignedStudents = assignedResponse.ok
              ? await safeReadJson(assignedResponse)
              : []

            const studentRows = Array.isArray(assignedStudents) ? assignedStudents : []
            const byDepartment = new Map()
            const uniqueCourseIds = new Set()

            studentRows.forEach((student) => {
              const departmentName = String(student?.department || 'Unassigned Department').trim()
              const courseId = Number(student?.course_id || 0)
              if (!courseId) return

              uniqueCourseIds.add(courseId)

              if (!byDepartment.has(departmentName)) {
                byDepartment.set(departmentName, new Map())
              }

              const deptCourses = byDepartment.get(departmentName)
              if (!deptCourses.has(courseId)) {
                deptCourses.set(courseId, {
                  course_id: courseId,
                  course_code: student?.course_code || `COURSE-${courseId}`,
                  course_name: student?.course_name || 'Unnamed Course',
                  components: [],
                  _student_ids: new Set(),
                })
              }

              const courseItem = deptCourses.get(courseId)
              courseItem._student_ids.add(Number(student?.student_id || 0))
            })

            const teacherIdentifier = username || teacherID
            const categoryResults = await Promise.all(
              Array.from(uniqueCourseIds).map(async (courseId) => {
                const categories = await fetchWindowCourseCategories(courseId, teacherIdentifier)
                return { courseId, categories }
              })
            )

            const categoryMap = new Map()
            categoryResults.forEach(({ courseId, categories }) => {
              categoryMap.set(courseId, Array.isArray(categories) ? categories : [])
            })

            const configuredComponentIds = Array.isArray(windowItem.component_ids)
              ? windowItem.component_ids.map((id) => Number(id)).filter((id) => Number.isFinite(id) && id > 0)
              : []

            let departments = Array.from(byDepartment.entries())
              .map(([department, coursesMap]) => {
                const courses = Array.from(coursesMap.values())
                  .map((course) => {
                    const categories = categoryMap.get(course.course_id) || []
                    let componentNames = []

                    if (configuredComponentIds.length === 0) {
                      componentNames = ['All Components']
                    } else {
                      const nameSet = new Set(
                        categories
                          .filter((category) => configuredComponentIds.includes(Number(category?.id)))
                          .map((category) => String(category?.name || '').trim())
                          .filter(Boolean)
                      )
                      componentNames = Array.from(nameSet)
                      if (componentNames.length === 0) {
                        componentNames = configuredComponentIds.map((id) => `Component #${id}`)
                      }
                    }

                    return {
                      course_id: course.course_id,
                      course_code: course.course_code,
                      course_name: course.course_name,
                      student_count: Array.from(course._student_ids).filter((id) => id > 0).length,
                      components: componentNames,
                    }
                  })
                  .sort((left, right) => String(left.course_code).localeCompare(String(right.course_code)))

                return { department, courses }
              })
              .sort((left, right) => String(left.department).localeCompare(String(right.department)))

            if (departments.length === 0) {
              const fallbackDepartmentNames = Array.isArray(windowItem.department_names) && windowItem.department_names.length > 0
                ? windowItem.department_names
                : [windowItem.department_name || 'All Departments']

              departments = fallbackDepartmentNames
                .filter(Boolean)
                .map((departmentName) => ({
                  department: String(departmentName),
                  courses: [],
                }))
            }

            return {
              id: Number(windowItem.id),
              window_name: windowItem.window_name || `User Window #${windowItem.id}`,
              start_at: windowItem.start_at,
              end_at: windowItem.end_at,
              departments,
              total_courses: departments.reduce((count, dept) => count + dept.courses.length, 0),
            }
          } catch (windowError) {
            console.warn('Skipping broken user window card payload:', windowError)
            return {
              id: Number(windowItem.id),
              window_name: windowItem.window_name || `User Window #${windowItem.id}`,
              start_at: windowItem.start_at,
              end_at: windowItem.end_at,
              departments: [{ department: windowItem.department_name || 'All Departments', courses: [] }],
              total_courses: 0,
            }
          }
        })
      )

      setUserWindowCards(windowCards)
    } catch (err) {
      console.error('Error loading user scoped window cards:', err)
      setUserWindowCards([])
    } finally {
      setUserWindowLoading(false)
    }
  }

  const fetchTeacherIdByEmail = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/teachers?email=${encodeURIComponent(userEmail)}`)
      if (response.ok) {
        const teachers = await response.json()
        if (teachers && teachers.length > 0) {
          const teacher = teachers[0]
          // Store the teacher ID in localStorage
          localStorage.setItem('teacherId', teacher.id)
          localStorage.setItem('teacher_id', teacher.id)
          localStorage.setItem('teacher_name', teacher.name || '')
          localStorage.setItem('faculty_id', teacher.faculty_id || '')
          console.log('Retrieved teacher data from email:', teacher)
          // Now fetch courses
          fetchTeacherCourses()
        } else {
          setError('Teacher profile not found. Please contact administrator.')
          setLoading(false)
        }
      } else {
        setError('Unable to load teacher profile. Please try again.')
        setLoading(false)
      }
    } catch (err) {
      console.error('Error fetching teacher ID:', err)
      setError('Failed to load teacher information.')
      setLoading(false)
    }
  }

  const fetchTeacherCourses = async () => {
    setLoading(true)
    setError('')

    try {
      const currentTeacherId = localStorage.getItem('teacherId') || localStorage.getItem('teacher_id')
      if (!currentTeacherId) {
        setError('Teacher ID not available')
        setLoading(false)
        return
      }

      const url = `${API_BASE_URL}/teachers/${currentTeacherId}/courses`
      const response = await fetch(url)

      if (!response.ok) {
        throw new Error('Failed to fetch your courses')
      }

      const data = await response.json()

      if (!data || data.length === 0) {
        setCoursesByCategory({})
        setLoading(false)
        return
      }

      // Group courses by category
      const grouped = {}
      data.forEach((course) => {
        const category = course.category || 'General'
        if (!grouped[category]) {
          grouped[category] = []
        }
        grouped[category].push(course)
      })

      // Sort categories
      const sorted = Object.keys(grouped)
        .sort()
        .reduce((obj, key) => {
          obj[key] = grouped[key].sort((a, b) =>
            a.course_code.localeCompare(b.course_code)
          )
          return obj
        }, {})

      setCoursesByCategory(sorted)
    } catch (err) {
      console.error('[TEACHER DASHBOARD] Error:', err)
      const message = String(err?.message || '')
      const isNoCourseMessage = message.toLowerCase().includes('no courses assigned')
      if (isNoCourseMessage) {
        setCoursesByCategory({})
        setError('')
      } else {
        setError(err.message || 'Failed to fetch your courses')
      }
    } finally {
      setLoading(false)
    }
  }



  const getCategoryColor = (category) => {
    const colors = {
      'Core': '#3b82f6',
      'Language Elective': '#8b5cf6',
      'Open': '#ec4899',
      'Foundation': '#10b981',
      'Lab': '#f59e0b',
      'Project': '#ef4444',
      'Seminar': '#06b6d4',
      'General': '#6b7280'
    }
    return colors[category] || '#6b7280'
  }

  const getActiveWindowCount = () => {
    let count = 0
    Object.values(coursesByCategory).forEach((courses) => {
      courses.forEach((course) => {
        if (course.has_window) {
          count++
        }
      })
    })
    return count
  }

  const getTotalCourses = () => {
    let total = 0
    Object.values(coursesByCategory).forEach((courses) => {
      total += courses.length
    })
    return total
  }

  const getTotalStudents = () => {
    let total = 0
    Object.values(coursesByCategory).forEach((courses) => {
      courses.forEach((course) => {
        total += course.enrollments?.length || 0
      })
    })
    return total
  }




  const handleCourseClick = (course) => {
    navigate(`/teacher-course/${course.id}/students`, { 
      state: { 
        course
      } 
    })
  }

  const getSemesterDisplay = (course) => {
    if (Array.isArray(course.mapped_semester_labels) && course.mapped_semester_labels.length > 0) {
      return course.mapped_semester_labels.join(', ')
    }

    if (Array.isArray(course.mapped_semesters) && course.mapped_semesters.length > 0) {
      return `Sem ${course.mapped_semesters.join(', ')}`
    }

    return 'Sem -'
  }

  return (
    <MainLayout title={`Welcome, ${teacherName || 'Teacher'}`} subtitle="Your Teaching Dashboard">
      <div className="space-y-6">
        {/* Loading State */}
        {loading && (
          <div className="flex items-center justify-center py-12">
            <div className="flex flex-col items-center space-y-4">
              <svg className="animate-spin h-12 w-12 text-blue-600" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <p className="text-gray-600 font-medium">Loading your dashboard...</p>
            </div>
          </div>
        )}

        {/* Error Message */}
        {error && !loading && (
          <div className="bg-red-50 border-l-4 border-red-500 p-4 rounded-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-red-500 mr-3" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <p className="text-red-700 font-medium">{error}</p>
            </div>
          </div>
        )}

        {/* User Window Cards */}
        {!loading && (userWindowLoading || userWindowCards.length > 0) && (
          <div className="rounded-2xl border border-primary_dim bg-white shadow-soft overflow-hidden">
            <div className="px-6 py-4 border-b border-primary_dim" style={{ background: 'linear-gradient(120deg, rgba(125, 83, 246, 0.12), rgba(125, 83, 246, 0.04))' }}>
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="text-base font-bold text-gray-900">My User-Scope Mark Entry Windows</h3>
                  <p className="text-xs text-gray-600 mt-0.5">Windows assigned to your account for mark entry access.</p>
                </div>
                <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold bg-primary_dim text-primary">
                  {userWindowCards.length} Active
                </span>
              </div>
            </div>
            <div className="p-4 space-y-4">
              {userWindowLoading && (
                <p className="text-sm text-gray-500">Loading user mark-entry windows...</p>
              )}

              {!userWindowLoading && userWindowCards.length === 0 && (
                <p className="text-sm text-gray-500">No active user-scoped mark entry windows assigned.</p>
              )}

              {!userWindowLoading && userWindowCards.map((windowCard) => (
                <div key={windowCard.id} className="rounded-xl border border-primary_dim bg-white shadow-card overflow-hidden">
                  <div className="px-4 py-3 border-b border-primary_dim flex flex-wrap items-center justify-between gap-3" style={{ background: 'linear-gradient(100deg, rgba(125, 83, 246, 0.10), rgba(125, 83, 246, 0.03))' }}>
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <span className="w-2 h-2 rounded-full bg-primary" />
                        <p className="text-sm font-semibold text-gray-900">{windowCard.window_name}</p>
                      </div>
                      <p className="text-xs text-gray-600">
                        {windowCard.start_at ? new Date(windowCard.start_at).toLocaleString() : '—'} to {windowCard.end_at ? new Date(windowCard.end_at).toLocaleString() : '—'}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => navigate('/mark-entry', {
                        state: {
                          selectedWindowId: windowCard.id,
                          selectedWindowName: windowCard.window_name,
                          source: 'teacher-dashboard-user-window-card',
                        },
                      })}
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg text-white transition-colors"
                      style={{ backgroundColor: '#7D53F6' }}
                    >
                      Open In Mark Entry
                    </button>
                  </div>

                  <div className="px-4 py-3 space-y-3">
                    {windowCard.departments.map((departmentItem) => (
                      <div key={`${windowCard.id}-${departmentItem.department}`}>
                        <div className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold uppercase tracking-wide mb-2 bg-primary_light text-primary border border-primary_dim">
                          {departmentItem.department}
                        </div>
                        <div className="space-y-2">
                          {departmentItem.courses.length === 0 ? (
                            <p className="text-xs text-gray-500">No specific course mapping in this window.</p>
                          ) : (
                            departmentItem.courses.map((courseItem) => (
                              <div key={`${windowCard.id}-${departmentItem.department}-${courseItem.course_id}`} className="text-sm text-gray-700 bg-sec-dim border border-primary_light rounded-lg p-3">
                                <p className="font-medium text-gray-900">
                                  {courseItem.course_code} - {courseItem.course_name}
                                </p>
                                <p className="text-xs text-gray-500 mt-1">
                                  {courseItem.student_count} students
                                </p>
                                <p className="text-xs text-gray-600 mt-1">
                                  <span className="font-semibold text-primary">Components:</span> {courseItem.components.join(', ')}
                                </p>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Dashboard Content */}
        {!loading && !error && Object.keys(coursesByCategory).length > 0 && (
          <>
            {/* Summary Stats */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {/* Total Courses Card */}
              <StatCard
                stat={{
                  title: "My Courses",
                  value: getTotalCourses(),
                  icon: (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
                    </svg>
                  )
                }}
              />

              {/* Total Students Card */}
              <StatCard
                stat={{
                  title: "Total Students",
                  value: getTotalStudents(),
                  icon: (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                    </svg>
                  )
                }}
              />

              {/* Active Mark Entry Card */}
              <StatCard
                stat={{
                  title: "Active Mark Entry",
                  value: getActiveWindowCount(),
                  icon: (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  )
                }}
              />

            </div>

            {/* Courses by Category */}
            <div className="space-y-4">
              {Object.entries(coursesByCategory).map(([category, courses]) => (
                <div key={category} className="card-custom overflow-hidden">
                  <div className="border-b border-gray-200 px-6 py-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-gray-700">{category}</h3>
                    <span className="px-3 py-1 bg-gray-100 text-gray-700 rounded-full text-xs font-medium">
                      {courses.length} {courses.length === 1 ? 'course' : 'courses'}
                    </span>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="w-[7.28%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            S.No
                          </th>
                          <th className="w-[12%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Code
                          </th>
                          <th className="w-[20%] px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Course Name
                          </th>
                          <th className="w-[12%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Type
                          </th>
                          <th className="w-[10%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Credit
                          </th>
                          <th className="w-[12%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Students
                          </th>
                          <th className="w-[15%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Windows
                          </th>
                          <th className="w-[14%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Academic / Sem
                          </th>
                          <th className="w-[10%] px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Action
                          </th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-200">
                        {courses.map((course, idx) => (
                          <tr
                            key={`${course.id}-${idx}`}
                            className="hover:bg-blue-50 transition-colors cursor-pointer"
                            onClick={() => handleCourseClick(course)}
                          >
                            <td className="px-4 py-4 text-center text-sm font-medium text-gray-900">
                              {idx + 1}
                            </td>
                            <td className="px-4 py-4 text-center text-sm font-semibold text-gray-900">
                              {course.course_code}
                            </td>
                            <td className="px-4 py-4 text-left">
                              <div className="max-w-full">
                                <p className="text-sm font-medium text-gray-900 truncate" title={course.course_name}>
                                  {course.course_name}
                                </p>
                              </div>
                            </td>
                            <td className="px-4 py-4 text-center">
                              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                                course.course_type === 'theory'
                                  ? 'bg-blue-100 text-blue-800'
                                  : course.course_type === 'lab'
                                    ? 'bg-green-100 text-green-800'
                                    : course.course_type === 'theory_with_lab'
                                      ? 'bg-purple-100 text-purple-800'
                                      : 'bg-gray-100 text-gray-800'
                              }`}>
                                {course.course_type === 'theory_with_lab' ? 'Theory+Lab' : course.course_type}
                              </span>
                            </td>
                            <td className="px-4 py-4 text-center text-sm font-medium text-gray-900">
                              {course.credit}
                            </td>
                            <td className="px-4 py-4 text-center">
                              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                                {course.enrollments?.length || 0} students
                              </span>
                            </td>
                            <td className="px-4 py-4 text-center">
                              {course.has_window ? (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                                  Active
                                </span>
                              ) : course.has_missed_submission ? (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-700 border border-red-200">
                                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                  </svg>
                                  Incomplete
                                </span>
                              ) : (
                                <span className="text-xs text-gray-400">No Window</span>
                              )}
                            </td>
                            <td className="px-4 py-4 text-center">
                              <div className="text-xs text-gray-700 font-medium">
                                {course.academic_year || '—'}
                              </div>
                              <div className="text-xs text-gray-500 mt-0.5">
                                {getSemesterDisplay(course)}
                                {' · '}
                                {course.semester_type ? String(course.semester_type).toUpperCase() : '—'}
                              </div>
                            </td>
                            <td className="px-4 py-4 text-center">
                              <svg className="w-5 h-5 text-blue-500 hover:text-blue-700 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                              </svg>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        {/* Empty State */}
        {!loading && !error && Object.keys(coursesByCategory).length === 0 && (
          <div className="card-custom p-12 text-center">
            <div className="w-16 h-16 rounded-full bg-blue-50 mx-auto mb-4 flex items-center justify-center">
              <svg className="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-gray-800 mb-2">No Teacher Courses Assigned</h3>
            <p className="text-gray-600">No teacher-assigned courses are available right now. User-scope mark entry windows (if any) are listed above.</p>
          </div>
        )}
      </div>

    </MainLayout>
  )
}

export default TeacherDashboardPage
