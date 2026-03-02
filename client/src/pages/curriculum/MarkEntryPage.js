import React, { useState, useEffect } from 'react'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'

function MarkEntryPage() {
  const [courses, setCourses] = useState([])
  const [selectedCourse, setSelectedCourse] = useState(null)
  const [markCategories, setMarkCategories] = useState([])
  const [students, setStudents] = useState([])
  const [allStudents, setAllStudents] = useState([])
  const [studentMarks, setStudentMarks] = useState({})
  const [loading, setLoading] = useState(false)
  const [savingMarks, setSavingMarks] = useState(false)
  const [message, setMessage] = useState({ type: '', text: '' })
  const [learningMode, setLearningMode] = useState(null) // Will be auto-detected: 'PBL' or 'UAL'
  const [hasActiveWindow, setHasActiveWindow] = useState(true)

  const teacherId = localStorage.getItem('teacher_id') || localStorage.getItem('teacherId')
  const username = localStorage.getItem('username') // Username is needed for users with mark entry permissions
  const userRole = localStorage.getItem('userRole') || localStorage.getItem('role')
  const isTeacher = userRole === 'teacher' && !!teacherId
  const facultyIdentifier = isTeacher ? teacherId : username

  // Fetch courses on component mount (teacher or user)
  useEffect(() => {
    if (isTeacher) {
      fetchTeacherCourses()
    } else if (username) {
      // For users with mark entry permissions, use username
      fetchUserCourses()
    } else {
      setMessage({ type: 'error', text: 'User credentials not found. Please login again.' })
    }
  }, [isTeacher, teacherId, username, userRole])

  const fetchTeacherCourses = async () => {
    try {
      setLoading(true)
      console.log('=== TEACHER ID DEBUG ===')
      console.log('localStorage.teacher_id:', localStorage.getItem('teacher_id'))
      console.log('localStorage.teacherId:', localStorage.getItem('teacherId'))
      console.log('Using teacherId value:', teacherId)
      console.log('API URL:', `${API_BASE_URL}/teachers/${teacherId}/courses`)
      console.log('======================')
      const response = await fetch(`${API_BASE_URL}/teachers/${teacherId}/courses`)
      if (!response.ok) throw new Error('Failed to fetch courses')
      const data = await response.json()
      console.log('Received courses data:', data)
      
      // Filter courses with enrollments
      const coursesWithStudents = data.filter((course) => course.enrollments && course.enrollments.length > 0)
      console.log('Filtered courses with students:', coursesWithStudents)
      console.log('Filtered out courses:', data.filter((course) => !course.enrollments || course.enrollments.length === 0))
      setCourses(coursesWithStudents)
      
      // Select first course if available
      if (coursesWithStudents.length > 0) {
        setSelectedCourse(coursesWithStudents[0])
      }
      setMessage({ type: '', text: '' })
    } catch (error) {
      console.error('Error fetching courses:', error)
      setMessage({ type: 'error', text: 'Failed to load courses. Please try again.' })
    } finally {
      setLoading(false)
    }
  }

  const fetchUserCourses = async () => {
    try {
      setLoading(true)
      const response = await fetch(`${API_BASE_URL}/users/${username}/courses`)
      if (!response.ok) throw new Error('Failed to fetch courses')
      const data = await response.json()
      
      // Transform user courses to match teacher courses format
      const formattedCourses = data.map(course => ({
        course_id: course.course_id,
        course_code: course.course_code,
        course_name: course.course_name,
        category: course.category,
        semester: course.semester,
        window_id: course.window_id,
        enrollments: [] // Will be populated when fetching students
      }))
      
      setCourses(formattedCourses)
      
      // Select first course if available
      if (formattedCourses.length > 0) {
        setSelectedCourse(formattedCourses[0])
      }
      setMessage({ type: '', text: '' })
    } catch (error) {
      console.error('Error fetching courses:', error)
      setMessage({ type: 'error', text: 'Failed to load courses. Please try again.' })
    } finally {
      setLoading(false)
    }
  }

  // Auto-detect learning mode when course is selected
  useEffect(() => {
    if (!selectedCourse || !selectedCourse.enrollments || selectedCourse.enrollments.length === 0) return
    
    // Detect which learning modes students have
    const learningModes = selectedCourse.enrollments
      .map(s => s.learning_mode_id)
      .filter(mode => mode === 1 || mode === 2)
    
    const hasUAL = learningModes.includes(1)
    const hasPBL = learningModes.includes(2)
    
    // Set default mode: prefer UAL if present, otherwise PBL
    if (hasUAL) {
      setLearningMode('UAL')
      console.log('Auto-detected learning mode: UAL')
    } else if (hasPBL) {
      setLearningMode('PBL')
      console.log('Auto-detected learning mode: PBL')
    } else {
      setLearningMode('UAL') // Default fallback
      console.log('No learning mode detected, defaulting to: UAL')
    }
  }, [selectedCourse])

  // Fetch mark categories when course is selected or learning mode changes
  useEffect(() => {
    if (!selectedCourse || !learningMode) return
    if (userRole !== 'teacher' && !hasActiveWindow) return
    fetchMarkCategories()
    loadExistingMarks()
  }, [selectedCourse, learningMode, hasActiveWindow, userRole])

  const fetchMarkCategories = async () => {
    try {
      if (!facultyIdentifier) {
        setMessage({ type: 'error', text: 'User identifier not found. Please login again.' })
        return
      }

      // Auto-detect learning modes from enrolled students
      const enrollments = selectedCourse.enrollments || []
      const uniqueLearningModes = [...new Set(
        enrollments
          .map(s => s.learning_mode_id)
          .filter(mode => mode === 1 || mode === 2)
      )]
      
      // If no learning modes detected, default to both UAL and PBL
      const learningModesParam = uniqueLearningModes.length > 0 
        ? uniqueLearningModes.join(',') 
        : '1,2'
      
      console.log('Requesting mark categories for learning modes:', learningModesParam)
      const response = await fetch(
        `${API_BASE_URL}/course/${selectedCourse.course_id}/mark-categories?teacher_id=${facultyIdentifier}&learning_modes=${learningModesParam}`
      )
      if (response.status === 403) {
        setMarkCategories([])
        setMessage({ type: 'warning', text: 'Mark entry window is closed for this course.' })
        return
      }
      if (!response.ok) throw new Error('Failed to fetch mark categories')
      const data = await response.json()
      setMarkCategories(data || [])
    } catch (error) {
      console.error('Error fetching mark categories:', error)
      setMessage({ type: 'error', text: 'Failed to load mark categories.' })
    }
  }

  const loadExistingMarks = async () => {
    try {
      if (!facultyIdentifier) {
        setMessage({ type: 'error', text: 'User identifier not found. Please login again.' })
        return
      }
      const response = await fetch(
        `${API_BASE_URL}/course/${selectedCourse.course_id}/student-marks?teacher_id=${facultyIdentifier}`
      )
      if (response.status === 403) {
        setStudentMarks({})
        setMessage({ type: 'warning', text: 'Mark entry window is closed for this course.' })
        return
      }
      if (!response.ok) throw new Error('Failed to fetch marks')
      const data = await response.json()
      
      // Convert array of marks to object structure
      const marksObj = {}
      if (data && Array.isArray(data)) {
        data.forEach((mark) => {
          if (!marksObj[mark.student_id]) {
            marksObj[mark.student_id] = {}
          }
          marksObj[mark.student_id][mark.assessment_component_id] = mark.obtained_marks
        })
      }
      setStudentMarks(marksObj)
    } catch (error) {
      console.error('Error loading existing marks:', error)
      // Initialize empty marks if fetch fails
      const emptyMarks = {}
      selectedCourse.enrollments.forEach((student) => {
        emptyMarks[student.student_id] = {}
      })
      setStudentMarks(emptyMarks)
    }
  }

  const enrichStudentsWithEnrollmentNumbers = async (enrollments) => {
    try {
      // Fetch all students to get enrollment numbers and learning mode
      const response = await fetch(`${API_BASE_URL}/students`)
      if (!response.ok) throw new Error('Failed to fetch students')
      const allStudents = await response.json()
      
      // Create maps for student data
      const enrollmentMap = {}
      const registerMap = {}
      const learningModeMap = {}
      if (Array.isArray(allStudents)) {
        allStudents.forEach((student) => {
          enrollmentMap[student.student_id] = student.enrollment_no || ''
          registerMap[student.student_id] = student.register_no || ''
          learningModeMap[student.student_id] = student.learning_mode_id
        })
      }
      
      // Enrich enrollments with enrollment numbers, register numbers, and learning mode
      return enrollments.map((student) => ({
        ...student,
        enrollment_no: enrollmentMap[student.student_id] || '',
        register_no: registerMap[student.student_id] || '',
        learning_mode_id: student.learning_mode_id || learningModeMap[student.student_id]
      }))
    } catch (error) {
      console.error('Error fetching enrollment numbers:', error)
      // Return original enrollments if fetch fails
      return enrollments.map((student) => ({
        ...student,
        enrollment_no: '',
      }))
    }
  }

  // Update students when course changes
  useEffect(() => {
    if (!selectedCourse) return

    // For users, fetch assigned students from window
    if (!isTeacher && username) {
      fetchUserAssignedStudents()
    }
    // For teachers, use enrollments from course
    else if (selectedCourse.enrollments) {
      enrichStudentsWithEnrollmentNumbers(selectedCourse.enrollments).then((enrichedStudents) => {
        setAllStudents(enrichedStudents)
        
        // Filter students by learning mode (UAL=1, PBL=2)
        const learningModeId = learningMode === 'UAL' ? 1 : 2
        const filteredStudents = enrichedStudents.filter(
          (student) => student.learning_mode_id === learningModeId
        )
        
        setStudents(filteredStudents)
      })
      
      // Initialize marks for new students
      const newMarks = {}
      selectedCourse.enrollments.forEach((student) => {
        newMarks[student.student_id] = studentMarks[student.student_id] || {}
      })
      setStudentMarks(newMarks)
    }
  }, [selectedCourse, learningMode, username, teacherId])

  const fetchUserAssignedStudents = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/mark-entry/user-assigned-students?user_id=${username}&course_id=${selectedCourse.course_id}`
      )
      if (!response.ok) throw new Error('Failed to fetch assigned students')
      const data = await response.json()
      
      // Transform to match student format with enrollment info
      const studentList = await enrichStudentsWithEnrollmentNumbers(
        data.map(s => ({
          student_id: s.student_id,
          student_name: s.student_name,
          enrollment_no: s.enrollment_no
        }))
      )
      
      setAllStudents(studentList)

      const hasWindow = studentList.length > 0
      setHasActiveWindow(hasWindow)
      if (!hasWindow) {
        setStudents([])
        setStudentMarks({})
        setMessage({ type: 'warning', text: 'No active mark entry window for this course.' })
        return
      }
      
      // Filter by learning mode (include students with missing learning_mode_id)
      const learningModeId = learningMode === 'UAL' ? 1 : 2
      const filteredStudents = studentList.filter(
        (student) => !student.learning_mode_id || student.learning_mode_id === learningModeId
      )
      
      setStudents(filteredStudents)
      
      // Initialize marks
      const newMarks = {}
      studentList.forEach((student) => {
        newMarks[student.student_id] = studentMarks[student.student_id] || {}
      })
      setStudentMarks(newMarks)
    } catch (error) {
      console.error('Error fetching assigned students:', error)
      setHasActiveWindow(false)
      setMessage({ type: 'error', text: 'Failed to load assigned students.' })
    }
  }

  const handleMarkChange = (studentId, categoryId, value) => {
    const category = markCategories.find((cat) => cat.id === categoryId)
    const maxMarks = category?.max_marks || 0
    
    // Allow empty value
    if (value === '' || value === null || value === undefined) {
      setStudentMarks((prev) => ({
        ...prev,
        [studentId]: {
          ...prev[studentId],
          [categoryId]: '',
        },
      }))
      return
    }

    const numValue = parseFloat(value) || 0
    // Limit to max marks
    const finalValue = Math.min(Math.max(numValue, 0), maxMarks)

    setStudentMarks((prev) => ({
      ...prev,
      [studentId]: {
        ...prev[studentId],
        [categoryId]: finalValue,
      },
    }))
  }

  const calculateConvertedMarks = (earnedMarks, maxMarks, conversionMarks) => {
    if (maxMarks === 0 || !earnedMarks) return '0.00'
    return ((earnedMarks / maxMarks) * conversionMarks).toFixed(2)
  }

  // Filter mark categories by selected learning mode (UAL=1, PBL=2)
  const filteredMarkCategories = markCategories.filter((category) => {
    const learningModeId = learningMode === 'UAL' ? 1 : 2
    return category.learning_mode_id === learningModeId
  })

  const calculateStudentTotal = (studentId) => {
    let total = 0
    filteredMarkCategories.forEach((category) => {
      const earned = studentMarks[studentId]?.[category.id]
      if (earned !== '' && earned !== null && earned !== undefined) {
        const converted = parseFloat(calculateConvertedMarks(earned, category.max_marks, category.conversion_marks))
        total += converted
      }
    })
    return total.toFixed(2)
  }

  const calculateTotalWeightage = () => {
    return filteredMarkCategories.reduce((sum, cat) => sum + cat.conversion_marks, 0).toFixed(2)
  }

  const handleSaveMarks = async () => {
    const facultyId = facultyIdentifier
    if (!selectedCourse || !facultyId) {
      setMessage({ type: 'error', text: 'Invalid course or user information' })
      return
    }

    // Collect all mark entries
    const markEntries = []
    students.forEach((student) => {
      filteredMarkCategories.forEach((category) => {
        const obtainedMarks = studentMarks[student.student_id]?.[category.id]
        if (obtainedMarks !== undefined && obtainedMarks !== null && obtainedMarks !== '') {
          markEntries.push({
            student_id: student.student_id,
            course_id: selectedCourse.course_id,
            assessment_component_id: category.id,
            obtained_marks: parseFloat(obtainedMarks),
          })
        }
      })
    })

    if (markEntries.length === 0) {
      setMessage({ type: 'warning', text: 'No marks to save. Please enter some marks first.' })
      return
    }

    try {
      setSavingMarks(true)
      const response = await fetch(`${API_BASE_URL}/student-marks/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          course_id: selectedCourse.course_id,
          faculty_id: facultyId,
          mark_entries: markEntries,
        }),
      })

      const result = await response.json()
      if (response.ok && result.success) {
        setMessage({ type: 'success', text: result.message })
        // Refresh marks after save
        setTimeout(() => loadExistingMarks(), 1000)
      } else {
        setMessage({ type: 'error', text: result.message || 'Failed to save marks' })
      }
    } catch (error) {
      console.error('Error saving marks:', error)
      setMessage({ type: 'error', text: 'Error saving marks. Please try again.' })
    } finally {
      setSavingMarks(false)
    }
  }

  if (loading) {
    return (
      <MainLayout title="Mark Entry" subtitle="Enter and manage student marks">
        <div className="flex justify-center items-center h-64">
          <div className="text-center">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
            <p className="mt-4 text-gray-600">Loading courses...</p>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout title="Mark Entry" subtitle="Enter and manage student marks">
      <div className="space-y-6">
        {/* Message Display */}
        {message.text && (
          <div
            className={`rounded-lg p-4 border-l-4 ${
              message.type === 'error'
                ? 'bg-red-50 text-red-700 border-red-400'
                : message.type === 'success'
                ? 'bg-green-50 text-green-700 border-green-400'
                : 'bg-yellow-50 text-primary border-primary'
            }`}
          >
            {message.text}
          </div>
        )}

        {/* Course Selection Card */}
        {courses.length > 0 ? (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100">
            <div className="border-b border-gray-200 px-6 py-3">
              <h3 className="text-sm font-semibold text-gray-700">Course Selection</h3>
            </div>
            <div className="p-6">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Select Course
              </label>
              <select
                value={selectedCourse?.course_id || ''}
                onChange={(e) => {
                  const course = courses.find((c) => c.course_id === parseInt(e.target.value))
                  setSelectedCourse(course)
                }}
                className="w-full max-w-md px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              >
                {courses.map((course) => (
                  <option key={course.course_id} value={course.course_id}>
                    {course.course_code} - {course.course_name}
                  </option>
                ))}
              </select>
              {selectedCourse && (
                <div className="mt-4 flex flex-wrap gap-4 text-sm text-gray-600">
                  <div>
                    <span className="font-medium text-gray-700">Category:</span> {selectedCourse.category}
                  </div>
                  <div>
                    <span className="font-medium text-gray-700">Credit:</span> {selectedCourse.credit}
                  </div>
                  <div>
                    <span className="font-medium text-gray-700">Students:</span> {selectedCourse.enrollments?.length || 0}
                  </div>
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-yellow-800">
            No courses found for you. Please contact administrator.
          </div>
        )}

        {/* Learning Mode Toggle */}
        {selectedCourse && learningMode && (
          <div className="bg-purple-50 rounded-lg p-4 border border-purple-100 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <h4 className="text-sm font-semibold text-gray-700 mb-1">Student Learning Mode</h4>
                <p className="text-xs text-gray-600">
                  Toggle to view and enter marks for {learningMode === 'PBL' ? 'PBL' : 'UAL'} students
                </p>
              </div>
              <div className="flex items-center gap-3">
                <span className={`text-sm font-semibold transition-colors ${learningMode === 'PBL' ? 'text-blue-700' : 'text-gray-400'}`}>
                  PBL
                </span>
                <button
                  onClick={() => setLearningMode(learningMode === 'PBL' ? 'UAL' : 'PBL')}
                  className={`relative inline-flex h-7 w-14 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 ${
                    learningMode === 'PBL' ? 'bg-blue-600' : 'bg-orange-600'
                  }`}
                >
                  <span
                    className={`inline-block h-5 w-5 transform rounded-full bg-white shadow-lg transition-transform ${
                      learningMode === 'PBL' ? 'translate-x-1' : 'translate-x-8'
                    }`}
                  />
                </button>
                <span className={`text-sm font-semibold transition-colors ${learningMode === 'UAL' ? 'text-orange-700' : 'text-gray-400'}`}>
                  UAL
                </span>
              </div>
            </div>
            <div className="mt-3 flex items-center justify-between text-xs">
              <span className="text-gray-600">
                Showing: <span className="font-semibold text-gray-800">{learningMode === 'PBL' ? 'Problem-Based Learning' : 'University Aided Learning'}</span> students
              </span>
              <span className="text-gray-600">
                Students: <span className="font-semibold text-gray-800">{students.length}</span> / <span className="text-gray-500">{allStudents.length} total</span>
              </span>
            </div>
          </div>
        )}

        {/* Mark Entry Table */}
        {selectedCourse && filteredMarkCategories.length > 0 && (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden flex flex-col" style={{ height: 'calc(100vh - 400px)', minHeight: '500px' }}>
            <div className="border-b border-gray-200 px-6 py-4 flex-shrink-0">
              <div className="flex justify-between items-center">
                <div>
                  <h3 className="text-sm font-semibold text-gray-700">
                    Mark Entry - {selectedCourse.course_code}
                  </h3>
                  <p className="text-xs text-gray-500 mt-1">Enter marks for each assessment component</p>
                </div>
                <button
                  onClick={handleSaveMarks}
                  disabled={savingMarks}
                  className="px-6 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                >
                  {savingMarks ? 'Saving...' : 'Save Marks'}
                </button>
              </div>
            </div>

            <div className="overflow-auto flex-1">
              {students.length === 0 ? (
                <div className="p-8 text-center text-gray-500">
                  <p className="text-sm font-medium">No {learningMode} students found for this course</p>
                  <p className="text-xs mt-1">Try switching to {learningMode === 'PBL' ? 'UAL' : 'PBL'} mode to see other students</p>
                </div>
              ) : (
                <table className="w-full divide-y divide-gray-200 relative">
                  <thead className="bg-gray-50 sticky top-0 z-20">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider border-r border-gray-200 sticky left-0 bg-gray-50 z-30 shadow-sm" style={{ minWidth: '220px', maxWidth: '220px' }}>
                      Student
                    </th>
                    {filteredMarkCategories.map((category) => (
                      <th
                        key={category.id}
                        className="px-3 py-3 text-center text-xs font-semibold text-gray-700 border-r border-gray-200"
                        style={{ minWidth: '150px', maxWidth: '200px' }}
                      >
                        <div className="break-words leading-tight">{category.name}</div>
                        <div className="text-gray-500 font-normal mt-1">Max: {category.max_marks}</div>
                      </th>
                    ))}
                    <th className="px-4 py-3 text-center text-xs font-semibold text-gray-700 uppercase tracking-wider bg-blue-50 sticky right-0 z-30 shadow-sm" style={{ minWidth: '120px', maxWidth: '120px' }}>
                      <div>Total</div>
                      <div className="text-gray-500 font-normal mt-0.5">/ {calculateTotalWeightage()}</div>
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {students.map((student, idx) => (
                    <tr
                      key={student.student_id}
                      className={`${
                        idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'
                      } hover:bg-blue-50 transition-colors`}
                    >
                      <td
                        className={`px-4 py-3 border-r border-gray-200 sticky left-0 z-10 shadow-sm ${
                          idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'
                        } hover:bg-blue-50 transition-colors`}
                        style={{ minWidth: '220px', maxWidth: '220px' }}
                      >
                        <div className="text-sm font-semibold text-gray-800 truncate" title={student.student_name}>{student.student_name}</div>
                        <div className="text-xs text-gray-500 mt-0.5 truncate" title={`${student.enrollment_no || 'N/A'} / ${student.register_no || 'N/A'}`}>
                          {student.enrollment_no || 'N/A'} / {student.register_no || 'N/A'}
                        </div>
                      </td>
                      {filteredMarkCategories.map((category) => {
                        const earned = studentMarks[student.student_id]?.[category.id]
                        const converted = calculateConvertedMarks(earned, category.max_marks, category.conversion_marks)
                        return (
                          <td key={category.id} className="px-3 py-3 text-center border-r border-gray-200" style={{ minWidth: '150px', maxWidth: '200px' }}>
                            <input
                              type="number"
                              min="0"
                              max={category.max_marks}
                              step="0.01"
                              value={earned ?? ''}
                              onChange={(e) => handleMarkChange(student.student_id, category.id, e.target.value)}
                              placeholder="0"
                              className="w-20 px-2 py-1 border border-gray-300 rounded text-center text-sm font-medium text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                            <div className="text-xs text-green-600 font-semibold mt-1">{converted}</div>
                          </td>
                        )
                      })}
                      <td 
                        className={`px-4 py-3 text-center text-base font-bold text-blue-700 bg-blue-50 sticky right-0 z-10 shadow-sm hover:bg-blue-100 transition-colors`}
                        style={{ minWidth: '120px', maxWidth: '120px' }}
                      >
                        {calculateStudentTotal(student.student_id)}
                      </td>
                    </tr>
                  ))}
                </tbody>
                </table>
              )}
            </div>

            {/* Legend */}
            <div className="border-t border-gray-200 px-6 py-4 bg-gray-50 flex-shrink-0">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                <div>
                  <p className="font-semibold text-gray-700 mb-1">Input Format</p>
                  <p className="text-gray-600">Enter marks (capped at maximum)</p>
                </div>
                <div>
                  <p className="font-semibold text-gray-700 mb-1">Calculation</p>
                  <p className="text-gray-600">Green value = (Earned ÷ Max) × Conversion</p>
                </div>
                <div>
                  <p className="font-semibold text-gray-700 mb-1">Total Score</p>
                  <p className="text-gray-600">Sum of all converted marks</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {selectedCourse && filteredMarkCategories.length === 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-blue-800">
            No mark categories found for {learningMode} mode. Try switching to {learningMode === 'PBL' ? 'UAL' : 'PBL'} mode.
          </div>
        )}
      </div>
    </MainLayout>
  )
}

export default MarkEntryPage
