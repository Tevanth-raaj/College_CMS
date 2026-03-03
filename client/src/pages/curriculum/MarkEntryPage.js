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
  const [showSubmitDialog, setShowSubmitDialog] = useState(false)
  const [showSaveDialog, setShowSaveDialog] = useState(false)
  const [isSubmitted, setIsSubmitted] = useState(false)
  const [markErrors, setMarkErrors] = useState({}) // key: `${studentId}_${categoryId}`
  const [absentees, setAbsentees] = useState([]) // [{student_id, mark_category_id}]
  const [applicableWindows, setApplicableWindows] = useState([])
  const [selectedWindowId, setSelectedWindowId] = useState(null)

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
  }, [selectedCourse, learningMode, hasActiveWindow, userRole, selectedWindowId])

  useEffect(() => {
    if (!isTeacher || !selectedCourse || !facultyIdentifier) {
      setApplicableWindows([])
      setSelectedWindowId(null)
      return
    }
    fetchApplicableWindows()
  }, [isTeacher, selectedCourse?.course_id, facultyIdentifier])

  const fetchApplicableWindows = async () => {
    try {
      const params = new URLSearchParams({
        teacher_id: facultyIdentifier,
        course_id: String(selectedCourse.course_id),
      })
      const response = await fetch(`${API_BASE_URL}/mark-entry/applicable-windows?${params.toString()}`)
      if (!response.ok) {
        setApplicableWindows([])
        setSelectedWindowId(null)
        return
      }

      const data = await response.json()
      const windows = Array.isArray(data) ? data : []
      setApplicableWindows(windows)

      if (windows.length === 0) {
        setSelectedWindowId(null)
      } else if (!windows.some((w) => w.id === selectedWindowId)) {
        setSelectedWindowId(windows[0].id)
      }
    } catch (error) {
      console.error('Error fetching applicable windows:', error)
      setApplicableWindows([])
      setSelectedWindowId(null)
    }
  }

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
      const params = new URLSearchParams({
        teacher_id: facultyIdentifier,
        learning_modes: learningModesParam,
      })
      if (selectedWindowId) {
        params.set('window_id', String(selectedWindowId))
      }
      const response = await fetch(
        `${API_BASE_URL}/course/${selectedCourse.course_id}/mark-categories?${params.toString()}`
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

  const loadAbsentees = async () => {
    if (!selectedCourse) return
    try {
      const url = facultyIdentifier 
        ? `${API_BASE_URL}/course/${selectedCourse.course_id}/exam-absentees?teacher_id=${facultyIdentifier}`
        : `${API_BASE_URL}/course/${selectedCourse.course_id}/exam-absentees`
      const res = await fetch(url)
      if (res.ok) {
        const data = await res.json()
        console.log('[ABSENTEES] Loaded absentees:', data)
        setAbsentees(Array.isArray(data) ? data : [])
      }
    } catch (err) {
      console.error('Error fetching exam absentees:', err)
    }
  }

  // helper: is a given (studentId, categoryId) cell absent?
  const isCellAbsent = (studentId, categoryId) =>
    absentees.some(a => a.student_id === studentId && a.mark_category_id === categoryId)

  const loadExistingMarks = async () => {
    try {
      if (!facultyIdentifier) {
        setMessage({ type: 'error', text: 'User identifier not found. Please login again.' })
        return
      }
      const params = new URLSearchParams({
        teacher_id: facultyIdentifier,
      })
      if (selectedWindowId) {
        params.set('window_id', String(selectedWindowId))
      }
      const response = await fetch(
        `${API_BASE_URL}/course/${selectedCourse.course_id}/student-marks?${params.toString()}`
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

      // Also refresh absentees whenever marks are loaded
      loadAbsentees()

      // If DB has no marks, reset the submitted flag — handles manual DB clears / testing
      const key = `mark_submitted_${selectedCourse.course_id}_${facultyIdentifier}`
      if (!data || data.length === 0) {
        localStorage.removeItem(key)
        setIsSubmitted(false)
      }
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
    const errorKey = `${studentId}_${categoryId}`

    // Allow empty value
    if (value === '' || value === null || value === undefined) {
      setStudentMarks((prev) => ({
        ...prev,
        [studentId]: { ...prev[studentId], [categoryId]: '' },
      }))
      setMarkErrors((prev) => { const n = { ...prev }; delete n[errorKey]; return n })
      return
    }

    const numValue = parseFloat(value)
    if (isNaN(numValue) || numValue < 0) return

    setStudentMarks((prev) => ({
      ...prev,
      [studentId]: { ...prev[studentId], [categoryId]: numValue },
    }))

    if (numValue > maxMarks) {
      setMarkErrors((prev) => ({ ...prev, [errorKey]: `Max is ${maxMarks}` }))
    } else {
      setMarkErrors((prev) => { const n = { ...prev }; delete n[errorKey]; return n })
    }
  }

  // Filter mark categories by selected learning mode (UAL=1, PBL=2)
  const filteredMarkCategories = markCategories.filter((category) => {
    const learningModeId = learningMode === 'UAL' ? 1 : 2
    return category.learning_mode_id === learningModeId
  })

  // All marks are considered complete when every student×category cell is either
  // absent (blocked) or has a value entered.
  const allMarksFilled =
    students.length > 0 &&
    filteredMarkCategories.length > 0 &&
    students.every(student =>
      filteredMarkCategories.every(cat =>
        isCellAbsent(student.student_id, cat.id) ||
        (studentMarks[student.student_id]?.[cat.id] !== '' &&
         studentMarks[student.student_id]?.[cat.id] !== null &&
         studentMarks[student.student_id]?.[cat.id] !== undefined)
      )
    )

  // Group categories by the prefix before "->" (with or without surrounding spaces)
  const getCategoryGroup = (name) => {
    const match = name.match(/^(.+?)\s*->\s*.+$/)
    return match ? match[1].trim() : name.trim()
  }

  const categoryGroups = filteredMarkCategories.reduce((groups, cat) => {
    const groupName = getCategoryGroup(cat.name)
    const existing = groups.find(g => g.groupName === groupName)
    if (existing) {
      existing.categories.push(cat)
    } else {
      groups.push({ groupName, categories: [cat] })
    }
    return groups
  }, [])

  const calculateGroupTotal = (studentId, categories) => {
    let total = 0
    categories.forEach(cat => {
      const val = studentMarks[studentId]?.[cat.id]
      if (val !== '' && val !== null && val !== undefined) {
        total += parseFloat(val) || 0
      }
    })
    return total
  }

  const calculateStudentTotal = (studentId) => {
    let total = 0
    filteredMarkCategories.forEach((category) => {
      const earned = studentMarks[studentId]?.[category.id]
      if (earned !== '' && earned !== null && earned !== undefined) {
        total += parseFloat(earned) || 0
      }
    })
    return total.toFixed(2)
  }

  const calculateTotalWeightage = () => {
    return filteredMarkCategories.reduce((sum, cat) => sum + (cat.max_marks || 0), 0).toFixed(2)
  }

  // Check submission status when course or faculty changes
  useEffect(() => {
    if (!selectedCourse || !facultyIdentifier) return
    const key = `mark_submitted_${selectedCourse.course_id}_${facultyIdentifier}`
    setIsSubmitted(localStorage.getItem(key) === 'true')
  }, [selectedCourse, facultyIdentifier])

  // Build submission summary: all students across both modes with their marks status
  const buildSubmitSummary = () => {
    return allStudents.map((student) => {
      const studentCategories = markCategories.filter((cat) => {
        const modId = student.learning_mode_id
        if (!modId) return true
        return cat.learning_mode_id === modId
      })
      const missing = studentCategories.filter((cat) => {
        const val = studentMarks[student.student_id]?.[cat.id]
        return val === '' || val === null || val === undefined
      })
      return { student, total: studentCategories.length, missing: missing.length }
    })
  }

  const handleConfirmSubmit = async () => {
    // Save marks to DB first, then lock
    const saved = await doSaveMarks()
    if (saved) {
      const key = `mark_submitted_${selectedCourse.course_id}_${facultyIdentifier}`
      localStorage.setItem(key, 'true')
      setIsSubmitted(true)
      setShowSubmitDialog(false)
      setMessage({ type: 'success', text: 'Marks saved and submitted successfully. You can no longer edit marks for this course.' })
    } else {
      setShowSubmitDialog(false)
    }
  }

  const doSaveMarks = async () => {
    const facultyId = facultyIdentifier
    if (!selectedCourse || !facultyId) {
      setMessage({ type: 'error', text: 'Invalid course or user information' })
      return false
    }

    if (Object.keys(markErrors).length > 0) {
      setMessage({ type: 'error', text: 'Fix the errors above before saving — some marks exceed the maximum.' })
      return false
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
      return false
    }

    try {
      setSavingMarks(true)
      const response = await fetch(`${API_BASE_URL}/student-marks/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          course_id: selectedCourse.course_id,
          faculty_id: facultyId,
          window_id: selectedWindowId || undefined,
          mark_entries: markEntries,
        }),
      })

      const result = await response.json()
      if (response.ok && result.success) {
        setMessage({ type: 'success', text: result.message })
        setTimeout(() => loadExistingMarks(), 1000)
        return true
      } else {
        setMessage({ type: 'error', text: result.message || 'Failed to save marks' })
        return false
      }
    } catch (error) {
      console.error('Error saving marks:', error)
      setMessage({ type: 'error', text: 'Error saving marks. Please try again.' })
      return false
    } finally {
      setSavingMarks(false)
    }
  }

  const handleSaveMarks = () => {
    setShowSaveDialog(true)
  }

  const handleFillOnes = () => {
    const updated = { ...studentMarks }
    students.forEach((student) => {
      if (!updated[student.student_id]) {
        updated[student.student_id] = {}
      }
      filteredMarkCategories.forEach((category) => {
        if (isCellAbsent(student.student_id, category.id)) return
        updated[student.student_id][category.id] = 1
      })
    })
    setStudentMarks(updated)
    setMessage({ type: 'success', text: 'Filled mark cells with 1 for current students/components.' })
  }

  const handleConfirmSave = async () => {
    setShowSaveDialog(false)
    await doSaveMarks()
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

        {/* Debug: Applicable Window Selector (faculty only, shown only if multiple windows apply) */}
        {isTeacher && selectedCourse && applicableWindows.length > 1 && (
          <div className="bg-amber-50 rounded-lg p-4 border border-amber-200 shadow-sm">
            <div className="flex items-center justify-between gap-4 flex-wrap">
              <div>
                <h4 className="text-sm font-semibold text-gray-800">Debug Window Selector</h4>
                <p className="text-xs text-gray-600 mt-1">Multiple active windows apply for this course/faculty. Select one for mark load/save.</p>
              </div>
              <div className="min-w-[320px]">
                <select
                  value={selectedWindowId || ''}
                  onChange={(e) => setSelectedWindowId(e.target.value ? parseInt(e.target.value, 10) : null)}
                  className="w-full px-3 py-2 border border-amber-300 rounded-lg focus:outline-none focus:border-amber-500 focus:ring-1 focus:ring-amber-500 bg-white"
                >
                  {applicableWindows.map((window) => (
                    <option key={window.id} value={window.id}>
                      Window #{window.id} | {window.start_at} to {window.end_at}
                    </option>
                  ))}
                </select>
              </div>
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
                <div className="flex items-center gap-3">
                  {isSubmitted ? (
                    <span className="flex items-center gap-2 px-4 py-2 bg-green-100 text-green-700 text-sm font-semibold rounded-lg border border-green-200">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                      Submitted
                    </span>
                  ) : (
                    <>
                      <button
                        onClick={handleFillOnes}
                        disabled={savingMarks}
                        className="px-5 py-2 bg-gray-600 text-white text-sm font-medium rounded-lg hover:bg-gray-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                      >
                        Fill 1s
                      </button>
                      <button
                        onClick={handleSaveMarks}
                        disabled={savingMarks}
                        className="px-5 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                      >
                        {savingMarks ? 'Saving...' : 'Save'}
                      </button>
                      <button
                        onClick={() => setShowSubmitDialog(true)}
                        disabled={!allMarksFilled || Object.keys(markErrors).length > 0}
                        title={!allMarksFilled ? 'Fill all mark fields before submitting' : ''}
                        className="px-5 py-2 bg-green-600 text-white text-sm font-medium rounded-lg hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                      >
                        Submit
                      </button>
                    </>
                  )}
                </div>
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
                    {/* Row 1 — group names */}
                    <tr className="border-b border-gray-300">
                      <th
                        rowSpan={2}
                        className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider border-r border-gray-200 sticky left-0 bg-gray-50 z-30 shadow-sm"
                        style={{ minWidth: '220px', maxWidth: '220px' }}
                      >
                        Student
                      </th>
                      {categoryGroups.map((group) => (
                        <React.Fragment key={group.groupName}>
                          <th
                            colSpan={group.categories.length}
                            className="px-3 py-2 text-center text-xs font-bold text-violet-700 bg-violet-50 border-r border-violet-200 border-l border-violet-200"
                          >
                            {group.groupName}
                          </th>
                          <th
                            rowSpan={2}
                            className="px-3 py-2 text-center text-xs font-bold text-violet-800 bg-violet-100 border-r border-violet-300"
                            style={{ minWidth: '90px' }}
                          >
                            <div>Total</div>
                            <div className="text-violet-500 font-normal text-xs mt-0.5">
                              / {group.categories.reduce((s, c) => s + (c.max_marks || 0), 0)}
                            </div>
                          </th>
                        </React.Fragment>
                      ))}
                    </tr>
                    {/* Row 2 — individual components */}
                    <tr>
                      {categoryGroups.map((group) =>
                        group.categories.map((category) => (
                          <th
                            key={category.id}
                            className="px-3 py-2 text-center text-xs font-semibold text-gray-700 border-r border-gray-200"
                            style={{ minWidth: '140px', maxWidth: '180px' }}
                          >
                            <div className="break-words leading-tight">
                              {/\s*->\s*/.test(category.name)
                                ? category.name.split(/\s*->\s*/)[1]
                                : category.name}
                            </div>
                            <div className="text-gray-500 font-normal mt-0.5">Max: {category.max_marks}</div>
                          </th>
                        ))
                      )}
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {students.map((student, idx) => (
                      <tr
                        key={student.student_id}
                        className={`${idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'} hover:bg-blue-50 transition-colors`}
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
                        {categoryGroups.map((group) => (
                          <React.Fragment key={group.groupName}>
                            {group.categories.map((category) => {
                              const earned = studentMarks[student.student_id]?.[category.id]
                              const errorKey = `${student.student_id}_${category.id}`
                              const hasError = !!markErrors[errorKey]
                              return (
                                <td key={category.id} className="px-3 py-3 text-center border-r border-gray-200" style={{ minWidth: '140px', maxWidth: '180px' }}>
                                  {isSubmitted ? (
                                    <span className="inline-block w-20 px-2 py-1 text-center text-sm font-medium text-gray-700 bg-gray-100 rounded border border-gray-200">
                                      {earned ?? '—'}
                                    </span>
                                  ) : isCellAbsent(student.student_id, category.id) ? (
                                    <span className="inline-flex items-center gap-1 text-red-300 text-xs font-medium tracking-wide">
                                      <svg className="w-3 h-3 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                        <circle cx="12" cy="12" r="10" />
                                        <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                                      </svg>
                                      Absent
                                    </span>
                                  ) : (
                                    <div className="flex flex-col items-center gap-0.5">
                                      <input
                                        type="number"
                                        min="0"
                                        max={category.max_marks}
                                        step="0.01"
                                        value={earned ?? ''}
                                        onChange={(e) => handleMarkChange(student.student_id, category.id, e.target.value)}
                                        placeholder="0"
                                        className={`w-20 px-2 py-1 border rounded text-center text-sm font-medium focus:outline-none focus:ring-2 ${
                                          hasError
                                            ? 'border-red-500 text-red-600 bg-red-50 focus:ring-red-400'
                                            : 'border-gray-300 text-gray-700 focus:ring-blue-500 focus:border-blue-500'
                                        }`}
                                      />
                                      {hasError && (
                                        <span className="text-xs text-red-600 font-semibold">{markErrors[errorKey]}</span>
                                      )}
                                    </div>
                                  )}
                                </td>
                              )
                            })}
                            {/* Group total */}
                            <td className="px-3 py-3 text-center font-bold text-violet-800 bg-violet-50 border-r border-violet-200" style={{ minWidth: '90px' }}>
                              {calculateGroupTotal(student.student_id, group.categories).toFixed(2)}
                            </td>
                          </React.Fragment>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>


          </div>
        )}

        {/* Save Preview Dialog */}
        {showSaveDialog && (() => {
          const summary = buildSubmitSummary()
          const filledCount = summary.filter(s => s.missing === 0).length
          const totalStudents = summary.length
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
              <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl mx-4 overflow-hidden">
                {/* Dialog Header */}
                <div className="px-6 pt-6 pb-4 border-b border-blue-200 bg-blue-50">
                  <div className="flex items-start gap-4">
                    <div className="flex-shrink-0 w-12 h-12 rounded-full bg-blue-100 border-2 border-blue-300 flex items-center justify-center">
                      <svg className="w-6 h-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" /></svg>
                    </div>
                    <div className="flex-1">
                      <h3 className="text-lg font-bold text-gray-900">Save Marks — {selectedCourse.course_code}</h3>
                      <p className="text-sm text-gray-600 mt-1">{filledCount} / {totalStudents} students fully filled</p>
                    </div>
                  </div>
                </div>

                {/* Summary */}
                <div className="px-6 py-4 max-h-80 overflow-y-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-200">
                        <th className="text-left py-2 text-xs font-semibold text-gray-600 uppercase">Student</th>
                        <th className="text-center py-2 text-xs font-semibold text-gray-600 uppercase">Components</th>
                        <th className="text-center py-2 text-xs font-semibold text-gray-600 uppercase">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {summary.map(({ student, total, missing }) => (
                        <tr key={student.student_id} className={missing > 0 ? 'bg-yellow-50' : ''}>
                          <td className="py-2 pr-4">
                            <div className="font-medium text-gray-800 text-xs">{student.student_name}</div>
                            <div className="text-xs text-gray-400">{student.enrollment_no || student.register_no || ''}</div>
                          </td>
                          <td className="py-2 text-center text-xs text-gray-600">{total - missing} / {total} filled</td>
                          <td className="py-2 text-center">
                            {missing === 0 ? (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-green-700">
                                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                                Complete
                              </span>
                            ) : (
                              <span className="text-xs font-semibold text-yellow-600">{missing} missing</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Dialog Footer */}
                <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3">
                  <button
                    onClick={() => setShowSaveDialog(false)}
                    className="px-5 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleConfirmSave}
                    disabled={savingMarks}
                    className="px-5 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                  >
                    {savingMarks ? 'Saving...' : 'Confirm Save'}
                  </button>
                </div>
              </div>
            </div>
          )
        })()}

        {/* Submit Confirmation Dialog */}
        {showSubmitDialog && (() => {
          const summary = buildSubmitSummary()
          const missingCount = summary.filter(s => s.missing > 0).length
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
              <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl mx-4 overflow-hidden">
                {/* Dialog Header */}
                <div className="px-6 pt-6 pb-4 border-b border-red-200 bg-red-50">
                  <div className="flex items-start gap-4">
                    <div className="flex-shrink-0 w-12 h-12 rounded-full bg-red-100 border-2 border-red-300 flex items-center justify-center">
                      <svg className="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" /></svg>
                    </div>
                    <div className="flex-1">
                      <h3 className="text-lg font-bold text-gray-900">Submit Marks — {selectedCourse.course_code}</h3>
                      <div className="mt-2 px-4 py-2.5 bg-red-600 rounded-lg">
                        <p className="text-sm font-bold text-white tracking-wide">
                          ⚠ Once submitted, marks cannot be edited until the window reopens.
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Summary */}
                <div className="px-6 py-4 max-h-80 overflow-y-auto">
                  {missingCount > 0 && (
                    <div className="mb-3 px-3 py-2 bg-yellow-50 border border-yellow-200 rounded-lg text-xs text-yellow-800 font-medium">
                      ⚠ {missingCount} student{missingCount > 1 ? 's have' : ' has'} incomplete marks. All marks must be filled before submitting.
                    </div>
                  )}
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-200">
                        <th className="text-left py-2 text-xs font-semibold text-gray-600 uppercase">Student</th>
                        <th className="text-center py-2 text-xs font-semibold text-gray-600 uppercase">Components</th>
                        <th className="text-center py-2 text-xs font-semibold text-gray-600 uppercase">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {summary.map(({ student, total, missing }) => (
                        <tr key={student.student_id} className={missing > 0 ? 'bg-yellow-50' : ''}>
                          <td className="py-2 pr-4">
                            <div className="font-medium text-gray-800 text-xs">{student.student_name}</div>
                            <div className="text-xs text-gray-400">{student.enrollment_no || student.register_no || ''}</div>
                          </td>
                          <td className="py-2 text-center text-xs text-gray-600">{total - missing} / {total} filled</td>
                          <td className="py-2 text-center">
                            {missing === 0 ? (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-green-700">
                                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                                Complete
                              </span>
                            ) : (
                              <span className="text-xs font-semibold text-yellow-600">{missing} missing</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Dialog Footer */}
                <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end gap-3">
                  <button
                    onClick={() => setShowSubmitDialog(false)}
                    className="px-5 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleConfirmSubmit}
                    disabled={!allMarksFilled || savingMarks}
                    className="px-5 py-2 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
                  >
                    {savingMarks ? 'Submitting...' : 'Confirm Submit'}
                  </button>
                </div>
              </div>
            </div>
          )
        })()}

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
