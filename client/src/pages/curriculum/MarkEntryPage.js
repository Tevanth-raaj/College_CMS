import React, { useState, useEffect, useRef } from 'react'
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
  const [isSubmitted, setIsSubmitted] = useState(false)
  const [markErrors, setMarkErrors] = useState({}) // key: `${studentId}_${categoryId}`
  const [absentees, setAbsentees] = useState([]) // [{student_id, mark_category_id}]
  const [autoSaveStatus, setAutoSaveStatus] = useState('') // '', 'saving', 'saved', 'error'
  const autoSaveTimerRef = useRef(null)
  const pendingMarksRef = useRef({})

  // Missed-window appeal states (for teachers who never submitted before the window closed)
  const [missedWindowAppeals, setMissedWindowAppeals] = useState({}) // { courseId → appeal | null }
  const [appealReason, setAppealReason] = useState('')
  const [appealSubmitting, setAppealSubmitting] = useState(false)

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCourse, learningMode, hasActiveWindow, userRole])

  const fetchMarkCategories = async () => {
    try {
      if (!facultyIdentifier) {
        setMessage({ type: 'error', text: 'User identifier not found. Please login again.' })
        return
      }

      // Use the selected learning mode from the PBL/UAL switch
      const learningModesParam = learningMode === 'UAL' ? 1 : 2
      
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

  const getInnovativePracticeBaseName = (name = '') => {
    const normalized = String(name).replace(/\s+/g, ' ').trim()
    const match = normalized.match(/^(Innovative Practice\s+[12])\s*-\s*\(\s*[12]\s*\)$/i)
    return match ? match[1] : null
  }

  const buildDisplayCategories = (categories = []) => {
    const display = []
    const groupedMap = new Map()

    categories.forEach((category) => {
      const baseName = getInnovativePracticeBaseName(category.name)
      if (!baseName) {
        display.push({
          ...category,
          display_name: category.name,
          component_ids: [category.id],
        })
        return
      }

      const key = `${category.learning_mode_id || 0}|${baseName.toLowerCase()}`
      if (!groupedMap.has(key)) {
        const grouped = {
          ...category,
          name: baseName,
          display_name: baseName,
          component_ids: [category.id],
          max_marks: category.max_marks,
        }
        groupedMap.set(key, grouped)
        display.push(grouped)
      } else {
        const grouped = groupedMap.get(key)
        grouped.component_ids.push(category.id)
        grouped.max_marks = Math.max(grouped.max_marks || 0, category.max_marks || 0)
      }
    })

    return display
  }

  const getDisplayMarkValue = (studentId, category) => {
    const componentIds = category.component_ids || [category.id]
    for (const componentId of componentIds) {
      const value = studentMarks[studentId]?.[componentId]
      if (value !== '' && value !== null && value !== undefined) return value
    }
    return ''
  }

  const isDisplayCellAbsent = (studentId, category) => {
    const componentIds = category.component_ids || [category.id]
    return componentIds.every((componentId) => isCellAbsent(studentId, componentId))
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

      // Also refresh absentees whenever marks are loaded
      loadAbsentees()

      // If DB has no marks, re-check submission status from the DB
      if (!data || data.length === 0) {
        try {
          const subRes = await fetch(
            `${API_BASE_URL}/mark-submissions/check?teacher_id=${encodeURIComponent(facultyIdentifier)}&course_id=${selectedCourse.course_id}`
          )
          if (subRes.ok) {
            const subData = await subRes.json()
            setIsSubmitted(subData.submitted === true)
          }
        } catch (e) {
          console.error('Error re-checking submission status:', e)
        }
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

  // Update students when course changes
  useEffect(() => {
    if (!selectedCourse) return

    // For users, fetch assigned students from window
    if (!isTeacher && username) {
      fetchUserAssignedStudents()
    }
    // For teachers, use enrollments from course (already includes enrollment_no, register_no, learning_mode_id)
    else if (selectedCourse.enrollments) {
      const enrollments = selectedCourse.enrollments
      setAllStudents(enrollments)
      
      // Filter students by learning mode (UAL=1, PBL=2) - strict matching only
      const learningModeId = learningMode === 'UAL' ? 1 : 2
      const filteredStudents = enrollments.filter(
        (student) => student.learning_mode_id === learningModeId
      )
      
      setStudents(filteredStudents)
      
      // Initialize marks for new students
      const newMarks = {}
      enrollments.forEach((student) => {
        newMarks[student.student_id] = studentMarks[student.student_id] || {}
      })
      setStudentMarks(newMarks)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCourse, learningMode, username, teacherId])

  const fetchUserAssignedStudents = async () => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/mark-entry/user-assigned-students?user_id=${username}&course_id=${selectedCourse.course_id}`
      )
      if (!response.ok) throw new Error('Failed to fetch assigned students')
      const data = await response.json()
      
      // Data already includes enrollment_no, register_no, learning_mode_id from the backend
      const studentList = Array.isArray(data) ? data : []
      
      setAllStudents(studentList)

      const hasWindow = studentList.length > 0
      setHasActiveWindow(hasWindow)
      if (!hasWindow) {
        setStudents([])
        setStudentMarks({})
        setMessage({ type: 'warning', text: 'No active mark entry window for this course.' })
        return
      }
      
      // Filter by learning mode - strict matching only
      const learningModeId = learningMode === 'UAL' ? 1 : 2
      const filteredStudents = studentList.filter(
        (student) => student.learning_mode_id === learningModeId
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

  // Debounced auto-save handler
  const triggerAutoSave = () => {
    // Clear existing timer
    if (autoSaveTimerRef.current) {
      clearTimeout(autoSaveTimerRef.current)
    }

    setAutoSaveStatus('saving')

    // Set new timer for batch save (500ms debounce for fast response)
    autoSaveTimerRef.current = setTimeout(async () => {
      const pendingMarks = { ...pendingMarksRef.current }
      pendingMarksRef.current = {}

      if (Object.keys(pendingMarks).length === 0) {
        setAutoSaveStatus('')
        return
      }

      const facultyId = facultyIdentifier
      if (!selectedCourse || !facultyId) {
        setAutoSaveStatus('')
        return
      }

      try {
        const markEntries = []
        for (const key in pendingMarks) {
          const [studentId, categoryId] = key.split('_')
          const obtainedMarks = pendingMarks[key]
          if (obtainedMarks !== '' && obtainedMarks !== null && obtainedMarks !== undefined) {
            markEntries.push({
              student_id: parseInt(studentId),
              course_id: selectedCourse.course_id,
              assessment_component_id: parseInt(categoryId),
              obtained_marks: parseFloat(obtainedMarks),
            })
          }
        }

        if (markEntries.length > 0) {
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
            setAutoSaveStatus('saved')
            setTimeout(() => setAutoSaveStatus(''), 2000)
          } else {
            setAutoSaveStatus('error')
            setTimeout(() => setAutoSaveStatus(''), 3000)
          }
        } else {
          setAutoSaveStatus('')
        }
      } catch (error) {
        console.error('Auto-save error:', error)
        setAutoSaveStatus('error')
        setTimeout(() => setAutoSaveStatus(''), 3000)
      }
    }, 500) // 500ms debounce for fast auto-save
  }

  const handleMarkChange = (studentId, category, value) => {
    const componentIds = category.component_ids || [category.id]
    const maxMarks = category?.max_marks || 0
    const errorKey = `${studentId}_${category.id}`

    // Allow empty value
    if (value === '' || value === null || value === undefined) {
      setStudentMarks((prev) => {
        const updatedStudent = { ...(prev[studentId] || {}) }
        componentIds.forEach((componentId) => {
          updatedStudent[componentId] = ''
        })
        return {
          ...prev,
          [studentId]: updatedStudent,
        }
      })
      setMarkErrors((prev) => { const n = { ...prev }; delete n[errorKey]; return n })
      // Add to pending marks and trigger auto-save
      componentIds.forEach((componentId) => {
        pendingMarksRef.current[`${studentId}_${componentId}`] = ''
      })
      triggerAutoSave()
      return
    }

    const numValue = parseFloat(value)
    if (isNaN(numValue) || numValue < 0) return

    setStudentMarks((prev) => {
      const updatedStudent = { ...(prev[studentId] || {}) }
      componentIds.forEach((componentId) => {
        updatedStudent[componentId] = numValue
      })
      return {
        ...prev,
        [studentId]: updatedStudent,
      }
    })

    if (numValue > maxMarks) {
      setMarkErrors((prev) => ({ ...prev, [errorKey]: `Max is ${maxMarks}` }))
    } else {
      setMarkErrors((prev) => { const n = { ...prev }; delete n[errorKey]; return n })
      // Add to pending marks and trigger auto-save
      componentIds.forEach((componentId) => {
        pendingMarksRef.current[`${studentId}_${componentId}`] = numValue
      })
      triggerAutoSave()
    }
  }

  // Filter mark categories by selected learning mode (UAL=1, PBL=2)
  const filteredMarkCategories = markCategories.filter((category) => {
    const learningModeId = learningMode === 'UAL' ? 1 : 2
    return category.learning_mode_id === learningModeId
  })

  const displayMarkCategories = buildDisplayCategories(filteredMarkCategories)

  // All marks are considered complete when every student×category cell is either
  // absent (blocked) or has a value entered.
  const allMarksFilled =
    students.length > 0 &&
    displayMarkCategories.length > 0 &&
    students.every(student =>
      displayMarkCategories.every(cat => {
        if (isDisplayCellAbsent(student.student_id, cat)) return true
        const val = getDisplayMarkValue(student.student_id, cat)
        return val !== '' && val !== null && val !== undefined
      }
      )
    )

  // Group categories by the prefix before "->" (with or without surrounding spaces)
  const getCategoryGroup = (name) => {
    const match = name.match(/^(.+?)\s*->\s*.+$/)
    return match ? match[1].trim() : name.trim()
  }

  const categoryGroups = displayMarkCategories.reduce((groups, cat) => {
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
      const val = getDisplayMarkValue(studentId, cat)
      if (val !== '' && val !== null && val !== undefined) {
        total += parseFloat(val) || 0
      }
    })
    return total
  }

  // Check submission status from DB when course or faculty changes
  useEffect(() => {
    if (!selectedCourse || !facultyIdentifier) return
    const checkSubmission = async () => {
      try {
        const res = await fetch(
          `${API_BASE_URL}/mark-submissions/check?teacher_id=${encodeURIComponent(facultyIdentifier)}&course_id=${selectedCourse.course_id}`
        )
        if (res.ok) {
          const data = await res.json()
          setIsSubmitted(data.submitted === true)
        }
      } catch (err) {
        console.error('Error checking submission status:', err)
      }
    }
    checkSubmission()
  }, [selectedCourse, facultyIdentifier])

  // Build submission summary: all students across both modes with their marks status
  const buildSubmitSummary = () => {
    return allStudents.map((student) => {
      const studentCategories = markCategories.filter((cat) => {
        const modId = student.learning_mode_id
        if (!modId) return true
        return cat.learning_mode_id === modId
      })
      const studentDisplayCategories = buildDisplayCategories(studentCategories)
      let absentCount = 0
      const missing = studentDisplayCategories.filter((cat) => {
        // Absent cells are not "missing" — they're excused
        if (isDisplayCellAbsent(student.student_id, cat)) {
          absentCount++
          return false
        }
        const val = getDisplayMarkValue(student.student_id, cat)
        return val === '' || val === null || val === undefined
      })
      return { student, total: studentDisplayCategories.length, missing: missing.length, absent: absentCount }
    })
  }

  const handleConfirmSubmit = async () => {
    // Clear any pending auto-saves
    if (autoSaveTimerRef.current) {
      clearTimeout(autoSaveTimerRef.current)
    }

    setSavingMarks(true)
    try {
      // Do a final save of all marks first
      const saved = await doSaveMarks()
      if (saved === false) {
        // doSaveMarks already set an error message
        return
      }

      // Record submission in the DB
      const res = await fetch(`${API_BASE_URL}/mark-submissions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          teacher_id: facultyIdentifier,
          course_id: selectedCourse.course_id,
        }),
      })

      if (!res.ok) {
        const errText = await res.text()
        throw new Error(errText || 'Failed to record submission')
      }

      setIsSubmitted(true)
      setShowSubmitDialog(false)
      setMessage({ type: 'success', text: 'Marks submitted successfully. You can no longer edit marks for this course.' })
    } catch (error) {
      console.error('Error submitting marks:', error)
      setMessage({ type: 'error', text: error.message || 'Failed to submit marks. Please try again.' })
    } finally {
      setSavingMarks(false)
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
      displayMarkCategories.forEach((category) => {
        const obtainedMarks = getDisplayMarkValue(student.student_id, category)
        if (obtainedMarks !== undefined && obtainedMarks !== null && obtainedMarks !== '') {
          ;(category.component_ids || [category.id]).forEach((componentId) => {
            markEntries.push({
              student_id: student.student_id,
              course_id: selectedCourse.course_id,
              assessment_component_id: componentId,
              obtained_marks: parseFloat(obtainedMarks),
            })
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



  // Fetch any existing appeal for the selected course's missed window
  useEffect(() => {
    if (!selectedCourse || !selectedCourse.has_missed_submission || !selectedCourse.missed_window) return
    if (!teacherId) return
    fetchMissedWindowAppeal(selectedCourse)
    setAppealReason('') // reset form when course changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCourse?.course_id])

  const fetchMissedWindowAppeal = async (course) => {
    try {
      const res = await fetch(
        `${API_BASE_URL}/mark-appeals?teacher_id=${encodeURIComponent(teacherId)}&course_id=${course.course_id}&window_id=${course.missed_window.id}`
      )
      if (res.ok) {
        const data = await res.json()
        setMissedWindowAppeals(prev => ({
          ...prev,
          [course.course_id]: Array.isArray(data) && data.length > 0 ? data[0] : null,
        }))
      }
    } catch (err) {
      console.error('Error fetching missed window appeal:', err)
    }
  }

  const submitMissedWindowAppeal = async (course) => {
    if (!appealReason.trim()) {
      setMessage({ type: 'error', text: 'Please enter a reason for the appeal.' })
      return
    }
    setAppealSubmitting(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-appeals`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          teacher_id: teacherId,
          course_id: course.course_id,
          window_id: course.missed_window.id,
          reason: appealReason.trim(),
        }),
      })
      if (res.ok) {
        setAppealReason('')
        setMessage({ type: 'success', text: 'Appeal submitted. The COE will review your request.' })
        await fetchMissedWindowAppeal(course)
      } else {
        const errText = await res.text()
        setMessage({ type: 'error', text: errText || 'Failed to submit appeal.' })
      }
    } catch (err) {
      setMessage({ type: 'error', text: 'Error submitting appeal. Please try again.' })
    } finally {
      setAppealSubmitting(false)
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

        {/* Missed Window Appeal Card — shown when a teacher never submitted before the window closed
            Hidden if teacher now has an active window (COE already extended it). */}
        {selectedCourse && selectedCourse.has_missed_submission && selectedCourse.missed_window && isTeacher && !selectedCourse.has_window && (() => {
          const appeal = missedWindowAppeals[selectedCourse.course_id]
          const win = selectedCourse.missed_window
          return (
            <div className="bg-white border border-red-200 rounded-xl shadow-sm overflow-hidden">
              {/* Header */}
              <div className="px-6 py-4 bg-gradient-to-r from-red-500 to-rose-500 flex items-center gap-3">
                <div className="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center flex-shrink-0">
                  <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.04 0 1.911-.757 1.993-1.79L21 4.79A1.99 1.99 0 0019 3H5a1.99 1.99 0 00-2 2.21l.947 13.21c.082 1.033.954 1.79 1.993 1.79z" />
                  </svg>
                </div>
                <div>
                  <h3 className="text-white font-bold text-sm">Missed Mark Entry Window</h3>
                  <p className="text-red-100 text-xs">You did not submit marks during the window period for this course.</p>
                </div>
              </div>

              {/* Window Details */}
              <div className="px-6 py-4 border-b border-red-100 bg-red-50/40">
                <p className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Window Details</p>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-xs text-gray-500">Window</span>
                    <p className="font-medium text-gray-800">{win.name || `Window #${win.id}`}</p>
                  </div>
                  <div>
                    <span className="text-xs text-gray-500">Period</span>
                    <p className="font-medium text-gray-800">
                      {new Date(win.start_at).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })}
                      {' → '}
                      {new Date(win.end_at).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })}
                    </p>
                  </div>
                </div>
              </div>

              {/* Appeal Section */}
              <div className="px-6 py-5">
                {appeal === undefined ? (
                  // Still loading appeal status
                  <div className="flex items-center gap-2 text-gray-400 text-sm">
                    <div className="animate-spin w-4 h-4 border-2 border-gray-300 border-t-transparent rounded-full" />
                    Checking appeal status...
                  </div>
                ) : !appeal ? (
                  // No appeal yet — show submission form
                  <div>
                    <p className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">Submit an Appeal</p>
                    <p className="text-xs text-gray-500 mb-3">
                      If you missed the deadline due to a valid reason, submit an appeal. The COE will review and can extend the window for you.
                    </p>
                    <textarea
                      value={appealReason}
                      onChange={e => setAppealReason(e.target.value)}
                      placeholder="Explain why you missed the mark entry deadline..."
                      rows={3}
                      className="w-full px-4 py-2.5 text-sm border border-red-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-red-400 focus:border-red-400 resize-none placeholder-gray-400"
                    />
                    <div className="mt-3 flex justify-end">
                      <button
                        onClick={() => submitMissedWindowAppeal(selectedCourse)}
                        disabled={appealSubmitting || !appealReason.trim()}
                        className="px-5 py-2 text-sm font-semibold text-white bg-red-600 hover:bg-red-700 disabled:bg-gray-400 disabled:cursor-not-allowed rounded-lg transition-colors flex items-center gap-2"
                      >
                        {appealSubmitting ? (
                          <>
                            <div className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full" />
                            Submitting...
                          </>
                        ) : (
                          <>
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                            </svg>
                            Submit Appeal
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                ) : appeal.status === 'pending' ? (
                  // Appeal submitted, waiting for COE
                  <div className="flex items-start gap-3 bg-amber-50 border border-amber-200 rounded-lg p-4">
                    <div className="w-8 h-8 rounded-full bg-amber-100 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <svg className="w-4 h-4 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-amber-800">Appeal Submitted — Awaiting COE Review</p>
                      <p className="text-xs text-gray-700 mt-1 leading-relaxed">{appeal.reason}</p>
                      <p className="text-xs text-gray-400 mt-1.5">Submitted on {new Date(appeal.created_at).toLocaleString()}</p>
                    </div>
                  </div>
                ) : appeal.status === 'resolved' ? (
                  // COE approved — window extended, teacher should reload
                  <div className="flex items-start gap-3 bg-green-50 border border-green-200 rounded-lg p-4">
                    <div className="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-green-800">Appeal Approved — Window Has Been Extended</p>
                      <p className="text-xs text-gray-600 mt-1">The COE has extended the mark entry window for you. Reload the page to start entering marks.</p>
                      <button
                        onClick={() => window.location.reload()}
                        className="mt-2.5 px-3 py-1.5 text-xs font-semibold text-white bg-green-600 hover:bg-green-700 rounded-lg transition-colors"
                      >
                        Reload Page
                      </button>
                    </div>
                  </div>
                ) : (
                  // Rejected (edge case)
                  <div className="flex items-start gap-3 bg-gray-50 border border-gray-200 rounded-lg p-4">
                    <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-gray-700">Appeal Rejected</p>
                      <p className="text-xs text-gray-600 mt-1">The COE has reviewed your appeal and declined the extension. Please contact the COE directly for further assistance.</p>
                      {appeal.resolved_at && (
                        <p className="text-xs text-gray-400 mt-1">Reviewed on {new Date(appeal.resolved_at).toLocaleString()}</p>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )
        })()}

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
        {selectedCourse && displayMarkCategories.length > 0 && (
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
                  {autoSaveStatus && !isSubmitted && (
                    <span className={`px-2 py-1 rounded text-xs font-medium ${
                      autoSaveStatus === 'saving'
                        ? 'bg-blue-100 text-blue-700'
                        : autoSaveStatus === 'saved'
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {autoSaveStatus === 'saving' ? 'Auto-saving…' : autoSaveStatus === 'saved' ? 'Auto-saved' : 'Auto-save failed'}
                    </span>
                  )}
                  {isSubmitted ? (
                    <span className="flex items-center gap-2 px-4 py-2 bg-green-100 text-green-700 text-sm font-semibold rounded-lg border border-green-200">
                      <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                      Submitted
                    </span>
                  ) : (
                    <>
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
                              const earned = getDisplayMarkValue(student.student_id, category)
                              const errorKey = `${student.student_id}_${category.id}`
                              const hasError = !!markErrors[errorKey]
                              return (
                                <td key={category.id} className="px-3 py-3 text-center border-r border-gray-200" style={{ minWidth: '140px', maxWidth: '180px' }}>
                                  {isSubmitted ? (
                                    isDisplayCellAbsent(student.student_id, category) ? (
                                      <span className="inline-flex items-center gap-1 text-red-400 text-xs font-medium tracking-wide">
                                        <svg className="w-3 h-3 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                          <circle cx="12" cy="12" r="10" />
                                          <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
                                        </svg>
                                        Absent
                                      </span>
                                    ) : (
                                      <span className="inline-block w-20 px-2 py-1 text-center text-sm font-medium text-gray-700 bg-gray-100 rounded border border-gray-200">
                                        {earned ?? '—'}
                                      </span>
                                    )
                                  ) : isDisplayCellAbsent(student.student_id, category) ? (
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
                                        onChange={(e) => handleMarkChange(student.student_id, category, e.target.value)}
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
                      {summary.map(({ student, total, missing, absent }) => (
                        <tr key={student.student_id} className={missing > 0 ? 'bg-yellow-50' : absent > 0 ? 'bg-orange-50' : ''}>
                          <td className="py-2 pr-4">
                            <div className="font-medium text-gray-800 text-xs">{student.student_name}</div>
                            <div className="text-xs text-gray-400">{student.enrollment_no || student.register_no || ''}</div>
                          </td>
                          <td className="py-2 text-center text-xs text-gray-600">
                            {total - missing - absent} / {total} filled
                            {absent > 0 && (
                              <span className="ml-1 text-orange-600">({absent} absent)</span>
                            )}
                          </td>
                          <td className="py-2 text-center">
                            {missing === 0 && absent === 0 ? (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-green-700">
                                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd"/></svg>
                                Complete
                              </span>
                            ) : missing === 0 && absent > 0 ? (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-orange-600">
                                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" /></svg>
                                Absent
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

        {selectedCourse && displayMarkCategories.length === 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-blue-800">
            No mark categories found for {learningMode} mode. Try switching to {learningMode === 'PBL' ? 'UAL' : 'PBL'} mode.
          </div>
        )}
      </div>
    </MainLayout>
  )
}

export default MarkEntryPage
