import React, { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import MainLayout from '../../components/MainLayout'
import { API_BASE_URL } from '../../config'
import SearchBarWithDropdown from '../../components/SearchBarWithDropdown'

// Helper function to format date for datetime-local input (preserves local timezone)
const formatDateTimeLocal = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function MarkEntryPermissionsPage() {
  const navigate = useNavigate()
  const userRole = localStorage.getItem('userRole')

  const [teachers, setTeachers] = useState([])
  const [selectedTeacherId, setSelectedTeacherId] = useState('')
  const [courses, setCourses] = useState([])
  const [selectedCourseId, setSelectedCourseId] = useState('')
  const [message, setMessage] = useState({ type: '', text: '' })
  const [departments, setDepartments] = useState([])
  const [windowScope, setWindowScope] = useState('teacher_course')
  const [windowDepartmentId, setWindowDepartmentId] = useState('')
  const [windowSemester, setWindowSemester] = useState('')
  const [windowCourseId, setWindowCourseId] = useState('')
  const [windowCourses, setWindowCourses] = useState([])
  const [windowStartAt, setWindowStartAt] = useState('')
  const [windowEndAt, setWindowEndAt] = useState('')
  const [windowEnabled, setWindowEnabled] = useState(true)
  const [windowLoading, setWindowLoading] = useState(false)
  const [windowComponents, setWindowComponents] = useState([])
  const [selectedPBLComponents, setSelectedPBLComponents] = useState([])
  const [selectedUALComponents, setSelectedUALComponents] = useState([])
  const [existingWindows, setExistingWindows] = useState([])
  const [editingWindow, setEditingWindow] = useState(null)
  const [teacherSearch, setTeacherSearch] = useState('')
  const [courseSearch, setCourseSearch] = useState('')
  const [showTeacherDropdown, setShowTeacherDropdown] = useState(false)
  const [learningMode, setLearningMode] = useState('PBL') // 'PBL' or 'UAL'

  // Student Assignment States
  const [activeTab, setActiveTab] = useState('windows') // 'windows' or 'student-assignment'
  const [mainTab, setMainTab] = useState('create') // 'create' | 'existing' | 'extension'
  const [existingTab, setExistingTab] = useState('window-scope') // 'window-scope' | 'user-student'
  const [availableUsers, setAvailableUsers] = useState([])
  const [selectedUserId, setSelectedUserId] = useState('')
  const [userSearchTerm, setUserSearchTerm] = useState('')
  const [students, setStudents] = useState([])
  const [selectedStudents, setSelectedStudents] = useState([])
  const [studentFilters, setStudentFilters] = useState({
    department: '',
    year: '',
    semester: '',
    search: ''
  })
  const [assignmentLoading, setAssignmentLoading] = useState(false)
  const [userAssignedWindows, setUserAssignedWindows] = useState([])

  // Check if user has COE or admin role
  useEffect(() => {
    if (userRole !== 'coe' && userRole !== 'admin') {
      navigate('/dashboard')
    }
  }, [userRole, navigate])

  useEffect(() => {
    fetchTeachers()
    fetchDepartments()
    fetchExistingWindows()
  }, [])

  // Load all students when student-assignment tab is opened
  useEffect(() => {
    if (activeTab === 'student-assignment') {
      fetchStudentsForAssignment()
      fetchUserAssignedWindows()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab])

  // Reload students when Window Scope department changes
  useEffect(() => {
    if (activeTab === 'student-assignment') {
      fetchStudentsForAssignment()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [windowDepartmentId])

  // Fetch courses when department and semester change in student-assignment tab
  useEffect(() => {
    if (activeTab === 'student-assignment' && windowSemester) {
      // Fetch courses (all departments if windowDepartmentId is empty)
      fetchDepartmentCurriculumCourses(windowDepartmentId, windowSemester)
    } else if (activeTab === 'student-assignment') {
      setWindowCourses([])
      setWindowCourseId('')
    }
  }, [activeTab, windowDepartmentId, windowSemester])

  // Load components when course changes in student-assignment tab
  useEffect(() => {
    if (activeTab === 'student-assignment' && windowCourseId && windowCourseId !== 'all') {
      fetchMarkCategoriesForCourseType(windowCourseId)
    } else if (activeTab === 'student-assignment') {
      // Show all categories if no specific course selected or if "All Courses" is selected
      fetchAllMarkCategories()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, windowCourseId, learningMode])

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (showTeacherDropdown && !event.target.closest('.teacher-search-container')) {
        setShowTeacherDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showTeacherDropdown])

  useEffect(() => {
    // Load window components when course context changes
    if (windowScope === 'teacher_course' && selectedCourseId) {
      fetchMarkCategoriesForCourseType(selectedCourseId)
    } else if (windowScope === 'department_semester_course' && windowCourseId && windowCourseId !== 'all') {
      fetchMarkCategoriesForCourseType(windowCourseId)
    } else if (windowScope === 'department_semester') {
      // For dept+semester, show all categories since it applies to all courses
      fetchAllMarkCategories()
    } else {
      setWindowComponents([])
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [windowScope, selectedCourseId, windowCourseId, learningMode])

  useEffect(() => {
    if (selectedTeacherId) {
      fetchTeacherCourses(selectedTeacherId)
    } else {
      setCourses([])
      setSelectedCourseId('')
    }
  }, [selectedTeacherId])



  useEffect(() => {
    if (windowScope === 'department_semester_course' && windowDepartmentId && windowDepartmentId !== '0' && windowSemester) {
      fetchDepartmentCourses(windowDepartmentId, windowSemester)
    } else {
      setWindowCourses([])
      if (windowScope === 'department_semester_course') {
        setWindowCourseId('')
      }
    }
  }, [windowScope, windowDepartmentId, windowSemester])

  useEffect(() => {
    loadWindowRule()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [windowScope, selectedTeacherId, selectedCourseId, windowDepartmentId, windowSemester, windowCourseId])

  const fetchTeachers = async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/teachers`)
      const data = await res.json()
      setTeachers(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Error fetching teachers:', error)
      setMessage({ type: 'error', text: 'Failed to load teachers.' })
    }
  }

  const fetchAllMarkCategories = async () => {
    try {
      // Convert learning mode to ID (UAL=1, PBL=2)
      const learningModeId = learningMode === 'UAL' ? 1 : 2

      // Fetch all mark category types (Theory=1, Lab=2, Theory+Lab=3)
      const results = await Promise.all([
        fetch(`${API_BASE_URL}/mark-categories-by-type/1?learning_modes=${learningModeId}`).then(r => r.json()),
        fetch(`${API_BASE_URL}/mark-categories-by-type/2?learning_modes=${learningModeId}`).then(r => r.json()),
        fetch(`${API_BASE_URL}/mark-categories-by-type/3?learning_modes=${learningModeId}`).then(r => r.json())
      ])

      // Combine and deduplicate
      const allCategories = []
      const seen = new Set()

      results.forEach(categories => {
        if (Array.isArray(categories)) {
          categories.forEach(cat => {
            if (!seen.has(cat.id)) {
              seen.add(cat.id)
              allCategories.push(cat)
            }
          })
        }
      })

      // Sort by position
      allCategories.sort((a, b) => (a.position || 0) - (b.position || 0))
      setWindowComponents(allCategories)
    } catch (error) {
      console.error('Error fetching mark categories:', error)
    }
  }

  const mapCourseCategoryToTypeID = (category) => {
    const categoryLower = (category || '').toLowerCase().trim()
    if (!categoryLower) return 0
    if (categoryLower.includes('theory') && categoryLower.includes('lab')) return 3
    if (categoryLower.includes('lab')) return 2
    return 1
  }

  const fetchMarkCategoriesForCourseType = async (courseId) => {
    try {
      // Convert learning mode to ID (UAL=1, PBL=2)
      const learningModeId = learningMode === 'UAL' ? 1 : 2

      // First fetch the course to get its category
      const courseRes = await fetch(`${API_BASE_URL}/course/${courseId}`)
      if (!courseRes.ok) throw new Error('Failed to fetch course')
      const course = await courseRes.json()

      const courseTypeID = mapCourseCategoryToTypeID(course.category)
      if (courseTypeID === 0) {
        setWindowComponents([])
        return
      }

      // Fetch mark categories for this course type with selected learning mode
      const res = await fetch(`${API_BASE_URL}/mark-categories-by-type/${courseTypeID}?learning_modes=${learningModeId}`)
      if (!res.ok) throw new Error('Failed to fetch mark categories')
      const categories = await res.json()

      // Sort by position
      const sorted = Array.isArray(categories) ? categories : []
      sorted.sort((a, b) => (a.position || 0) - (b.position || 0))
      setWindowComponents(sorted)
    } catch (error) {
      console.error('Error fetching mark categories for course:', error)
      setWindowComponents([])
    }
  }

  const fetchDepartments = async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/departments`)
      const data = await res.json()
      // Handle both response formats: array or object with departments property
      const depts = Array.isArray(data) ? data : (data.departments || [])
      setDepartments(depts)
    } catch (error) {
      console.error('Error fetching departments:', error)
      setMessage({ type: 'error', text: 'Failed to load departments.' })
    }
  }

  const fetchDepartmentCurriculumCourses = async (departmentId, semester) => {
    if (!semester) {
      setWindowCourses([])
      return
    }
    try {
      // If departmentId is empty, fetch courses from all departments
      const url = departmentId
        ? `${API_BASE_URL}/departments/${departmentId}/curriculum/semester/${semester}/courses`
        : `${API_BASE_URL}/all-departments/semester/${semester}/courses`
      const res = await fetch(url)
      const data = await res.json()
      setWindowCourses(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Error fetching department curriculum courses:', error)
      setWindowCourses([])
      setMessage({ type: 'error', text: 'Failed to load courses for department and semester.' })
    }
  }

  const fetchTeacherCourses = async (teacherId) => {
    setMessage({ type: '', text: '' })
    try {
      const res = await fetch(`${API_BASE_URL}/teachers/${teacherId}/courses`)
      if (!res.ok) throw new Error('Failed to fetch teacher courses')
      const data = await res.json()
      setCourses(Array.isArray(data) ? data : [])
      if (data && data.length > 0) {
        setSelectedCourseId(String(data[0].course_id))
      } else {
        setSelectedCourseId('')
      }
    } catch (error) {
      console.error('Error fetching courses:', error)
      setCourses([])
      setSelectedCourseId('')
      setMessage({ type: 'error', text: 'Failed to load courses for teacher.' })
    }
  }



  const fetchDepartmentCourses = async (departmentId, semester) => {
    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/department/${departmentId}/semester/${semester}/courses`)
      if (!res.ok) throw new Error('Failed to fetch department courses')
      const data = await res.json()
      setWindowCourses(Array.isArray(data) ? data : [])
      if (data && data.length > 0) {
        setWindowCourseId(String(data[0].course_id))
      } else {
        setWindowCourseId('')
      }
    } catch (error) {
      console.error('Error fetching department courses:', error)
      setWindowCourses([])
      setWindowCourseId('')
      setMessage({ type: 'error', text: 'Failed to load department courses.' })
    } finally {
      setWindowLoading(false)
    }
  }

  const buildWindowQuery = () => {
    const params = new URLSearchParams()

    if (windowScope === 'teacher_course') {
      if (!selectedTeacherId || !selectedCourseId) return ''
      params.append('teacher_id', selectedTeacherId)
      params.append('course_id', selectedCourseId)
    }

    if (windowScope === 'department_semester') {
      if (windowDepartmentId === '' || !windowSemester) return ''
      if (windowDepartmentId !== '0') {
        params.append('department_id', windowDepartmentId)
      }
      params.append('semester', windowSemester)
    }

    if (windowScope === 'department_semester_course') {
      if (windowDepartmentId === '' || !windowSemester || !windowCourseId || windowCourseId === 'all') return ''
      if (windowDepartmentId !== '0') {
        params.append('department_id', windowDepartmentId)
      }
      params.append('semester', windowSemester)
      params.append('course_id', windowCourseId)
    }

    return params.toString()
  }

  const loadWindowRule = async () => {
    const query = buildWindowQuery()
    if (!query) {
      // setWindowStartAt('')
      // setWindowEndAt('')
      setWindowEnabled(true)
      return
    }

    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-window?${query}`)
      if (!res.ok) throw new Error('Failed to fetch window rule')
      const data = await res.json()

      if (!data) {
        setWindowStartAt('')
        setWindowEndAt('')
        setWindowEnabled(true)
        setSelectedPBLComponents([])
        setSelectedUALComponents([])
        return
      }

      setWindowStartAt(formatDateTimeLocal(data.start_at))
      setWindowEndAt(formatDateTimeLocal(data.end_at))
      setWindowEnabled(data.enabled !== false)

      // Separate components by learning mode
      if (data.component_ids && data.component_ids.length > 0) {
        // Fetch all components to check their learning modes
        const allComponentsRes = await Promise.all([
          fetch(`${API_BASE_URL}/mark-categories-by-type/1?learning_modes=1,2`).then(r => r.json()),
          fetch(`${API_BASE_URL}/mark-categories-by-type/2?learning_modes=1,2`).then(r => r.json()),
          fetch(`${API_BASE_URL}/mark-categories-by-type/3?learning_modes=1,2`).then(r => r.json())
        ])

        const allComponents = []
        allComponentsRes.forEach(cats => {
          if (Array.isArray(cats)) allComponents.push(...cats)
        })

        const pblIds = []
        const ualIds = []
        data.component_ids.forEach(id => {
          const comp = allComponents.find(c => c.id === id)
          if (comp) {
            if (comp.learning_mode_id === 2) pblIds.push(id)
            else if (comp.learning_mode_id === 1) ualIds.push(id)
          }
        })

        setSelectedPBLComponents(pblIds)
        setSelectedUALComponents(ualIds)
      } else {
        setSelectedPBLComponents([])
        setSelectedUALComponents([])
      }
    } catch (error) {
      console.error('Error loading window rule:', error)
      setMessage({ type: 'error', text: 'Failed to load window rule.' })
    } finally {
      setWindowLoading(false)
    }
  }



  const saveWindowRule = async () => {
    if (!windowStartAt || !windowEndAt) {
      setMessage({ type: 'error', text: 'Start and end dates are required.' })
      return
    }

    // Convert datetime-local to ISO 8601 format with timezone
    const startDate = new Date(windowStartAt)
    const endDate = new Date(windowEndAt)

    const payload = {
      start_at: startDate.toISOString(),
      end_at: endDate.toISOString(),
      enabled: windowEnabled,
      component_ids: [...selectedPBLComponents, ...selectedUALComponents].length > 0
        ? [...selectedPBLComponents, ...selectedUALComponents]
        : null,
    }

    if (windowScope === 'teacher_course') {
      if (!selectedTeacherId || !selectedCourseId) {
        setMessage({ type: 'error', text: 'Select a teacher and course first.' })
        return
      }
      payload.teacher_id = selectedTeacherId
      payload.course_id = parseInt(selectedCourseId, 10)
    }

    if (windowScope === 'department_semester') {
      if (windowDepartmentId === '' || !windowSemester) {
        setMessage({ type: 'error', text: 'Select a department and semester.' })
        return
      }
      if (windowDepartmentId !== '0') {
        payload.department_id = parseInt(windowDepartmentId, 10)
      }
      payload.semester = parseInt(windowSemester, 10)
    }

    if (windowScope === 'department_semester_course') {
      if (windowDepartmentId === '' || !windowSemester || !windowCourseId) {
        setMessage({ type: 'error', text: 'Select department, semester, and course.' })
        return
      }
      if (windowDepartmentId !== '0') {
        payload.department_id = parseInt(windowDepartmentId, 10)
      }
      payload.semester = parseInt(windowSemester, 10)
      payload.course_id = parseInt(windowCourseId, 10)
    }

    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-window`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) throw new Error('Failed to save window')
      setMessage({ type: 'success', text: 'Mark entry window saved.' })
      fetchExistingWindows() // Refresh the list
    } catch (error) {
      console.error('Error saving window:', error)
      setMessage({ type: 'error', text: 'Failed to save mark entry window.' })
    } finally {
      setWindowLoading(false)
    }
  }

  const fetchExistingWindows = async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows`)
      if (!res.ok) throw new Error('Failed to fetch windows')
      const data = await res.json()
      setExistingWindows(data || [])
      return data || []
    } catch (error) {
      console.error('Error fetching windows:', error)
      return []
    }
  }

  const fetchWindowById = async (windowId) => {
    const allWindows = await fetchExistingWindows()
    return (allWindows || []).find((window) => String(window.id) === String(windowId)) || null
  }

  const fetchUserAssignedWindows = async () => {
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows?user_only=true`)
      if (!res.ok) throw new Error('Failed to fetch user-assigned windows')
      const data = await res.json()
      setUserAssignedWindows(data || [])
    } catch (error) {
      console.error('Error fetching user-assigned windows:', error)
    }
  }

  const deleteWindow = async (windowId) => {
    if (!window.confirm('Are you sure you want to delete this window?')) return

    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows/${windowId}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Failed to delete window')
      setMessage({ type: 'success', text: 'Window deleted successfully.' })
      fetchExistingWindows()
      if (activeTab === 'student-assignment') {
        fetchUserAssignedWindows()
      }
    } catch (error) {
      console.error('Error deleting window:', error)
      setMessage({ type: 'error', text: 'Failed to delete window.' })
    }
  }

  const editUserWindow = async (win) => {
    // Switch to student-assignment tab
    setActiveTab('student-assignment')

    // Set editing mode
    setEditingWindow(win)
    setWindowStartAt(formatDateTimeLocal(win.start_at))
    setWindowEndAt(formatDateTimeLocal(win.end_at))
    setWindowEnabled(win.enabled)

    // Set user
    if (win.user_id) {
      const userRes = await fetch(`${API_BASE_URL}/users/${win.user_id}`)
      if (userRes.ok) {
        const userData = await userRes.json()
        setSelectedUserId(userData.username)
      }
    }

    // Set scope fields
    setWindowDepartmentId(win.department_id || '')
    setWindowSemester(win.semester || '')
    setWindowCourseId(win.course_id || '')

    // Load components if course is set
    if (win.course_id && win.component_ids) {
      try {
        // Fetch all components to check their learning modes
        const allComponentsRes = await Promise.all([
          fetch(`${API_BASE_URL}/mark-categories-by-type/1?learning_modes=1,2`).then(r => r.json()),
          fetch(`${API_BASE_URL}/mark-categories-by-type/2?learning_modes=1,2`).then(r => r.json()),
          fetch(`${API_BASE_URL}/mark-categories-by-type/3?learning_modes=1,2`).then(r => r.json())
        ])

        const allComponents = []
        allComponentsRes.forEach(cats => {
          if (Array.isArray(cats)) allComponents.push(...cats)
        })

        const allPBL = []
        const allUAL = []

        for (const compId of win.component_ids) {
          const comp = allComponents.find(c => c.id === compId)
          if (comp) {
            if (comp.learning_mode_id === 2) {
              allPBL.push(compId)
            } else if (comp.learning_mode_id === 1) {
              allUAL.push(compId)
            }
          }
        }

        setSelectedPBLComponents(allPBL)
        setSelectedUALComponents(allUAL)
      } catch (error) {
        console.error('Error loading components:', error)
      }
    } else {
      setSelectedPBLComponents([])
      setSelectedUALComponents([])
    }

    // Load assigned students for this window
    try {
      const res = await fetch(
        `${API_BASE_URL}/mark-entry/user-assigned-students?user_id=${win.user_id}&window_id=${win.id}`
      )
      if (res.ok) {
        const assignedStudents = await res.json()
        setSelectedStudents(assignedStudents.map(s => s.student_id))
      }
    } catch (error) {
      console.error('Error loading assigned students:', error)
    }
    // Scroll to top after loading
    setTimeout(() => window.scrollTo({ top: 0, behavior: 'smooth' }), 100)
  }

  const editWindow = async (win) => {
    const latestWindow = (await fetchWindowById(win.id)) || win

    setEditingWindow(latestWindow)
    setWindowStartAt(formatDateTimeLocal(latestWindow.start_at))
    setWindowEndAt(formatDateTimeLocal(latestWindow.end_at))
    setWindowEnabled(latestWindow.enabled)

    // Will be populated by loadWindowRule which separates by learning mode
    setSelectedPBLComponents([])
    setSelectedUALComponents([])

    // Determine scope and set appropriate fields
    if (latestWindow.teacher_id && latestWindow.course_id) {
      setWindowScope('teacher_course')
      setSelectedTeacherId(latestWindow.teacher_id)
      setSelectedCourseId(latestWindow.course_id.toString())
    } else if (latestWindow.semester && latestWindow.course_id) {
      setWindowScope('department_semester_course')
      setWindowDepartmentId(latestWindow.department_id ? latestWindow.department_id.toString() : '0')
      setWindowSemester(latestWindow.semester.toString())
      setWindowCourseId(latestWindow.course_id.toString())
    } else if (latestWindow.semester) {
      setWindowScope('department_semester')
      setWindowDepartmentId(latestWindow.department_id ? latestWindow.department_id.toString() : '0')
      setWindowSemester(latestWindow.semester.toString())
    }

    setTimeout(() => window.scrollTo({ top: 0, behavior: 'smooth' }), 100)
  }

  const updateWindow = async () => {
    if (!editingWindow) return

    // Convert datetime-local to ISO 8601 format with timezone
    const startDate = new Date(windowStartAt)
    const endDate = new Date(windowEndAt)

    const payload = {
      start_at: startDate.toISOString(),
      end_at: endDate.toISOString(),
      enabled: windowEnabled,
      component_ids: [...selectedPBLComponents, ...selectedUALComponents].length > 0
        ? [...selectedPBLComponents, ...selectedUALComponents]
        : null,
    }

    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows/${editingWindow.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) throw new Error('Failed to update window')
      setMessage({ type: 'success', text: 'Window updated successfully.' })
      setEditingWindow(null)
      await fetchExistingWindows()
    } catch (error) {
      console.error('Error updating window:', error)
      setMessage({ type: 'error', text: 'Failed to update window.' })
    } finally {
      setWindowLoading(false)
    }
  }

  const cancelEdit = () => {
    setEditingWindow(null)
    setWindowStartAt('')
    setWindowEndAt('')
    setWindowEnabled(true)
    setSelectedPBLComponents([])
    setSelectedUALComponents([])
  }

  const getScopeDescription = (window) => {
    if (window.teacher_id && window.course_id) {
      return `${window.teacher_name} - ${window.course_code} (Most Specific)`
    } else if (window.department_id && window.semester && window.course_id) {
      return `${window.department_name} - Sem ${window.semester} - ${window.course_code}`
    } else if (window.department_id && window.semester) {
      return `${window.department_name} - Semester ${window.semester} (Least Specific)`
    } else if (!window.department_id && window.semester && window.course_id) {
      return `All Departments - Sem ${window.semester} - ${window.course_code}`
    } else if (!window.department_id && window.semester) {
      return `All Departments - Semester ${window.semester}`
    }
    return 'Unknown scope'
  }

  const getWindowStatus = (win) => {
    const now = new Date()
    const startAt = new Date(win.start_at)
    const endAt = new Date(win.end_at)

    if (!win.enabled) {
      return { status: 'Disabled', color: 'bg-red-100 text-red-800' }
    } else if (now < startAt) {
      return { status: 'Scheduled', color: 'bg-blue-100 text-blue-800' }
    } else if (now > endAt) {
      return { status: 'Expired', color: 'bg-yellow-100 text-yellow-800' }
    } else {
      return { status: 'Active', color: 'bg-green-100 text-green-800' }
    }
  }

  // Student Assignment Functions
  const fetchAvailableUsers = async (search = '') => {
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry/available-users?search=${search}`)
      const data = await res.json()
      setAvailableUsers(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Error fetching users:', error)
      setMessage({ type: 'error', text: 'Failed to load users.' })
    }
  }

  const fetchStudentsForAssignment = async () => {
    setAssignmentLoading(true)
    try {
      const params = new URLSearchParams()
      // Filter by Window Scope department if selected
      if (windowDepartmentId) {
        params.append('department_id', windowDepartmentId)
      }
      // Additional filters from student filter section
      if (studentFilters.department) params.append('department', studentFilters.department)
      if (studentFilters.year) params.append('year', studentFilters.year)
      if (studentFilters.semester) params.append('semester', studentFilters.semester)
      if (studentFilters.search) params.append('search', studentFilters.search)

      const res = await fetch(`${API_BASE_URL}/mark-entry/available-students?${params}`)
      const data = await res.json()
      setStudents(Array.isArray(data) ? data : [])
    } catch (error) {
      console.error('Error fetching students:', error)
      setMessage({ type: 'error', text: 'Failed to load students.' })
    } finally {
      setAssignmentLoading(false)
    }
  }

  const assignStudentsToUser = async () => {
    if (!selectedUserId || selectedStudents.length === 0 || !windowStartAt || !windowEndAt) {
      setMessage({ type: 'error', text: 'Please fill all required fields: user, time period, and at least one student.' })
      return
    }

    setAssignmentLoading(true)
    try {
      // Combine PBL and UAL components
      const allComponents = [...selectedPBLComponents, ...selectedUALComponents]

      if (editingWindow) {
        // Update existing window
        const updateRes = await fetch(`${API_BASE_URL}/mark-entry-windows/${editingWindow.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            start_at: new Date(windowStartAt).toISOString(),
            end_at: new Date(windowEndAt).toISOString(),
            enabled: windowEnabled,
            component_ids: allComponents.length > 0 ? allComponents : null,
          })
        })

        if (!updateRes.ok) {
          const error = await updateRes.text()
          throw new Error(error || 'Failed to update window')
        }

        // Update student assignments by re-assigning
        const assignRes = await fetch(`${API_BASE_URL}/mark-entry/assign-students`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            window_id: editingWindow.id,
            user_id: selectedUserId,
            student_ids: selectedStudents
          })
        })

        if (!assignRes.ok) {
          const error = await assignRes.text()
          throw new Error(error || 'Failed to update student assignments')
        }

        setMessage({
          type: 'success',
          text: `Successfully updated window and student assignments!`
        })
      } else {
        // Create new window
        const res = await fetch(`${API_BASE_URL}/mark-entry/create-user-window`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            user_id: selectedUserId,
            department_id: windowDepartmentId ? parseInt(windowDepartmentId) : null,
            semester: windowSemester ? parseInt(windowSemester) : null,
            course_id: windowCourseId && windowCourseId !== 'all' ? parseInt(windowCourseId) : null,
            student_ids: selectedStudents,
            start_at: windowStartAt,
            end_at: windowEndAt,
            component_ids: allComponents.length > 0 ? allComponents : [],
            created_by: localStorage.getItem('username') || 'coe_admin'
          })
        })

        if (!res.ok) {
          const error = await res.text()
          throw new Error(error || 'Failed to create window')
        }

        const result = await res.json()
        setMessage({
          type: 'success',
          text: `Successfully created window #${result.window_id} and assigned ${result.assignments_created} students! User can now enter marks (will overwrite existing marks).`
        })
      }

      // Refresh windows lists to show the updated/created window
      fetchExistingWindows()
      fetchUserAssignedWindows()

      // Reset form
      setSelectedStudents([])
      setSelectedUserId('')
      setStudents([])
      setWindowStartAt('')
      setWindowEndAt('')
      setWindowDepartmentId('')
      setWindowSemester('')
      setWindowCourseId('')
      setSelectedPBLComponents([])
      setSelectedUALComponents([])
      setEditingWindow(null)
    } catch (error) {
      console.error('Error creating/updating window:', error)
      setMessage({ type: 'error', text: error.message || 'Failed to create/update window.' })
    } finally {
      setAssignmentLoading(false)
    }
  }

  const toggleStudentSelection = (studentId) => {
    if (selectedStudents.includes(studentId)) {
      setSelectedStudents(selectedStudents.filter(id => id !== studentId))
    } else {
      setSelectedStudents([...selectedStudents, studentId])
    }
  }

  const selectAllStudents = () => {
    setSelectedStudents(students.map(s => s.student_id))
  }

  const clearStudentSelection = () => {
    setSelectedStudents([])
  }

  return (
    <MainLayout title="Mark Permissions" subtitle="Manage mark entry windows and permissions">
      <div className="space-y-6">

        {/* Message Alert */}
        {message.text && (
          <div
            className={`rounded-lg p-4 border-l-4 ${message.type === 'error'
                ? 'bg-red-50 text-red-700 border-red-400'
                : message.type === 'success'
                  ? 'bg-green-50 text-green-700 border-green-400'
                  : 'bg-blue-50 text-blue-700 border-blue-400'
              }`}
          >
            {message.text}
          </div>
        )}

        {/* ── Main Tab Toggle ── */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-1 inline-flex gap-1">
          {[
            { key: 'create', label: 'Mark Entry Window' },
            { key: 'existing', label: 'Existing Windows' },
            { key: 'extension', label: 'Window Extension', disabled: true },
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => !tab.disabled && setMainTab(tab.key)}
              disabled={tab.disabled}
              className={`px-5 py-2.5 text-sm font-semibold rounded-lg transition-all whitespace-nowrap ${
                mainTab === tab.key
                  ? 'bg-primary text-white shadow-sm'
                  : tab.disabled
                  ? 'text-gray-300 cursor-not-allowed'
                  : 'text-gray-500 hover:text-gray-800'
              }`}
            >
              {tab.label}
              {tab.disabled && <span className="ml-1.5 text-[10px] font-normal opacity-70">Soon</span>}
            </button>
          ))}
        </div>

        {/* ── Create Tab ── */}
        {mainTab === 'create' && (
          <>
            {/* Create Card */}
            <div className="bg-white rounded-xl shadow-sm border border-gray-100">
              <div className="border-b border-gray-200 px-6 py-4 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-semibold text-gray-700">
                    {editingWindow ? 'Edit Mark Entry Window' : 'Create Mark Entry Window'}
                  </h3>
                  <p className="text-xs text-gray-500 mt-1">
                    {editingWindow ? 'Update window configuration' : 'Define time-based access control'}
                  </p>
                  {/* Inner sub-toggle */}
                  <div className="mt-3 inline-flex rounded-lg bg-gray-100 p-0.5 gap-0.5">
                    <button
                      onClick={() => setActiveTab('windows')}
                      className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-all ${
                        activeTab === 'windows' ? 'bg-primary text-white shadow-sm' : 'text-gray-500 hover:text-gray-800'
                      }`}
                    >
                      Mark Entry Windows
                    </button>
                    <button
                      onClick={() => setActiveTab('student-assignment')}
                      className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-all ${
                        activeTab === 'student-assignment' ? 'bg-primary text-white shadow-sm' : 'text-gray-500 hover:text-gray-800'
                      }`}
                    >
                      Student-User Assignment
                    </button>
                  </div>
                </div>
                {activeTab === 'windows' && (
                  <div className="flex gap-2">
                    {editingWindow ? (
                      <>
                        <button
                          onClick={cancelEdit}
                          className="px-4 py-2 bg-gray-500 text-white text-sm font-medium rounded-lg hover:bg-gray-600 transition-colors"
                        >
                          Cancel
                        </button>
                        <button
                          onClick={updateWindow}
                          className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                          disabled={windowLoading}
                        >
                          {windowLoading ? 'Updating...' : 'Update Window'}
                        </button>
                      </>
                    ) : (
                      <button
                        onClick={saveWindowRule}
                        className="px-4 py-2 bg-primary text-white text-sm font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        disabled={windowLoading}
                      >
                        {windowLoading ? 'Saving...' : 'Save Window'}
                      </button>
                    )}
                  </div>
                )}
              </div>

              {/* ── Mark Entry Windows form ── */}
              {activeTab === 'windows' && (
              <div className="p-6 space-y-5">
                {editingWindow && (
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                    <p className="text-sm text-blue-900">
                      <span className="font-semibold">Editing:</span> {getScopeDescription(editingWindow)}
                    </p>
                  </div>
                )}

                {/* Scope Selection */}
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Window Scope
                  </label>
                  <select
                    value={windowScope}
                    onChange={(e) => setWindowScope(e.target.value)}
                    disabled={editingWindow !== null}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                  >
                    <option value="teacher_course">Teacher + Course (Most Specific)</option>
                    <option value="department_semester">Department + Semester</option>
                    <option value="department_semester_course">Department + Semester + Course</option>
                  </select>
                  {editingWindow && (
                    <p className="text-xs text-gray-500 mt-1">Scope cannot be changed for existing windows</p>
                  )}
                </div>

                {/* Teacher & Course Selection */}
                {windowScope === 'teacher_course' && (
                  <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                    <h4 className="text-sm font-semibold text-gray-700 mb-3">Teacher & Course Selection</h4>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                      <div className="relative teacher-search-container">
                        <SearchBarWithDropdown
                          value={teacherSearch}
                          onChange={(e) => setTeacherSearch(e.target.value)}
                          items={teachers}
                          onSelect={(teacher) => {
                            setSelectedTeacherId(teacher.faculty_id)
                            setTeacherSearch(`${teacher.name} (${teacher.faculty_id})`)
                          }}
                          filterFunction={(teacher, term) => {
                            const search = term.toLowerCase()
                            return (
                              teacher.name.toLowerCase().includes(search) ||
                              teacher.faculty_id.toLowerCase().includes(search)
                            )
                          }}
                          renderItem={(teacher) => (
                            <div className={selectedTeacherId === teacher.faculty_id ? 'bg-blue-50' : ''}>
                              <div className="font-medium text-gray-900">{teacher.name}</div>
                              <div className="text-sm text-gray-500">{teacher.faculty_id}</div>
                            </div>
                          )}
                          getItemKey={(teacher) => teacher.faculty_id}
                          label="Teacher"
                          placeholder="Search by name or ID..."
                          width="w-full"
                        />
                      </div>
                      <div className="relative course-search-container">
                        <SearchBarWithDropdown
                          value={courseSearch}
                          onChange={(e) => setCourseSearch(e.target.value)}
                          items={courses}
                          onSelect={(course) => {
                            setSelectedCourseId(course.course_id)
                            setCourseSearch(`${course.course_code} - ${course.course_name}`)
                          }}
                          filterFunction={(course, term) => {
                            const search = term.toLowerCase()
                            return (
                              course.course_code.toLowerCase().includes(search) ||
                              course.course_name.toLowerCase().includes(search)
                            )
                          }}
                          renderItem={(course) => (
                            <div className={selectedCourseId === course.course_id ? 'bg-blue-50' : ''}>
                              <div className="font-medium text-gray-900">{course.course_code}</div>
                              <div className="text-sm text-gray-500">{course.course_name}</div>
                            </div>
                          )}
                          getItemKey={(course) => course.course_id}
                          label="Course"
                          placeholder="Search by code or name..."
                          width="w-full"
                          disabled={!selectedTeacherId}
                        />
                      </div>
                    </div>
                  </div>
                )}

                {/* Department/Semester Selection */}
                {(windowScope === 'department_semester' || windowScope === 'department_semester_course') && (
                  <div className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                    <h4 className="text-sm font-semibold text-gray-700 mb-3">Department & Semester</h4>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-xs font-medium text-gray-600 mb-1">Department</label>
                        <select
                          value={windowDepartmentId}
                          onChange={(e) => setWindowDepartmentId(e.target.value)}
                          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm"
                        >
                          <option value="">Select Department</option>
                          <option value="0">All Departments</option>
                          {departments.map((department) => (
                            <option key={department.id} value={department.id}>
                              {department.name}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gray-600 mb-1">Semester</label>
                        <select
                          value={windowSemester}
                          onChange={(e) => setWindowSemester(e.target.value)}
                          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm"
                        >
                          <option value="">Select Semester</option>
                          {[1, 2, 3, 4, 5, 6, 7, 8].map((sem) => (
                            <option key={sem} value={sem}>
                              Semester {sem}
                            </option>
                          ))}
                        </select>
                      </div>
                      {windowScope === 'department_semester_course' && (
                        <div>
                          <label className="block text-xs font-medium text-gray-600 mb-1">Course</label>
                          {windowDepartmentId === '0' ? (
                            <input
                              type="number"
                              value={windowCourseId}
                              onChange={(e) => setWindowCourseId(e.target.value)}
                              placeholder="Enter Course ID"
                              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm"
                            />
                          ) : (
                            <select
                              value={windowCourseId}
                              onChange={(e) => setWindowCourseId(e.target.value)}
                              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 text-sm"
                            >
                              <option value="">Select Course</option>
                              {windowCourses.map((course) => (
                                <option key={course.course_id} value={course.course_id}>
                                  {course.course_code} - {course.course_name}
                                </option>
                              ))}
                            </select>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Time Window */}
                <div className="bg-background rounded-lg p-4 border border-blue-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Time Window Configuration</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-gray-700 mb-2">Start Date & Time</label>
                      <input
                        type="datetime-local"
                        value={windowStartAt}
                        onChange={(e) => setWindowStartAt(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700 mb-2">End Date & Time</label>
                      <input
                        type="datetime-local"
                        value={windowEndAt}
                        onChange={(e) => setWindowEndAt(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                  </div>
                  <div className="mt-3">
                    <label className="inline-flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={windowEnabled}
                        onChange={(e) => setWindowEnabled(e.target.checked)}
                        className="h-4 w-4 accent-blue-600 cursor-pointer"
                      />
                      <span className="text-sm text-gray-700 font-medium">Enable this window</span>
                    </label>
                  </div>
                </div>

                {/* Learning Mode Toggle */}
                <div className="bg-background rounded-lg p-4 border border-purple-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Learning Mode Selection</h4>
                  <p className="text-xs text-gray-600 mb-3">
                    Toggle to select mark components for each learning mode. Students of both UAL and PBL can be in the same course.
                    <strong className="block mt-1">Both PBL and UAL component selections will be saved together in this window.</strong>
                  </p>
                  <div className="flex items-center justify-center gap-4">
                    <span className={`text-sm font-semibold transition-colors ${learningMode === 'PBL' ? 'text-primary' : 'text-gray-400'}`}>
                      PBL
                    </span>
                    <button
                      onClick={() => setLearningMode(learningMode === 'PBL' ? 'UAL' : 'PBL')}
                      className={`relative inline-flex h-8 w-16 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 ${learningMode === 'PBL' ? 'bg-primary' : 'bg-orange-600'
                        }`}
                    >
                      <span
                        className={`inline-block h-6 w-6 transform rounded-full bg-white shadow-lg transition-transform ${learningMode === 'PBL' ? 'translate-x-1' : 'translate-x-9'
                          }`}
                      />
                    </button>
                    <span className={`text-sm font-semibold transition-colors ${learningMode === 'UAL' ? 'text-orange-700' : 'text-gray-400'}`}>
                      UAL
                    </span>
                  </div>
                  <div className="mt-3 flex items-center justify-between px-4">
                    <p className="text-xs text-gray-600">
                      Currently viewing: <span className="font-semibold text-gray-800">{learningMode === 'PBL' ? 'Problem-Based Learning' : 'University Aided Learning'}</span>
                    </p>
                    <div className="flex gap-4 text-xs">
                      <span className="text-primary font-medium">PBL: {selectedPBLComponents.length}</span>
                      <span className="text-orange-600 font-medium">UAL: {selectedUALComponents.length}</span>
                      <span className="text-gray-600 font-semibold">Total: {selectedPBLComponents.length + selectedUALComponents.length}</span>
                    </div>
                  </div>
                </div>

                {/* Component Selection */}
                {windowComponents.length > 0 && (
                  <div>
                    <label className="block text-sm font-semibold text-gray-700 mb-2">
                      Allowed Mark Components
                    </label>
                    <div className="space-y-3">
                      {Object.entries(
                        windowComponents.reduce((groups, component) => {
                          const courseTypeName = component.course_type_name || 'Other'
                          if (!groups[courseTypeName]) groups[courseTypeName] = []
                          groups[courseTypeName].push(component)
                          return groups
                        }, {})
                      ).map(([courseTypeName, components]) => (
                        <div key={courseTypeName} className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                          <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider mb-3">
                            {courseTypeName}
                          </h4>
                          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                            {components.map((component) => {
                              const selectedComponents = learningMode === 'PBL' ? selectedPBLComponents : selectedUALComponents
                              const setSelectedComponents = learningMode === 'PBL' ? setSelectedPBLComponents : setSelectedUALComponents

                              return (
                                <label
                                  key={component.id}
                                  className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer hover:text-blue-600"
                                >
                                  <input
                                    type="checkbox"
                                    checked={selectedComponents.includes(component.id)}
                                    onChange={(e) => {
                                      if (e.target.checked) {
                                        setSelectedComponents([...selectedComponents, component.id])
                                      } else {
                                        setSelectedComponents(
                                          selectedComponents.filter((id) => id !== component.id)
                                        )
                                      }
                                    }}
                                    className="h-4 w-4 accent-blue-600 cursor-pointer"
                                  />
                                  <span className="font-medium">{component.name}</span>
                                </label>
                              )
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                    <p className="text-xs text-gray-500 mt-2">
                      Leave all unchecked to allow all components, or select specific ones to restrict access.
                    </p>
                  </div>
                )}
              </div>
              )}

              {/* ── Student-User Assignment form ── */}
              {activeTab === 'student-assignment' && (
              <div className="divide-y divide-gray-100">
                <div className="px-6 py-3 bg-amber-50 border-b border-amber-100">
                  <p className="text-xs text-amber-700">Create a dedicated mark entry window for a user to enter marks for specific students. User marks will overwrite any existing marks.</p>
                </div>

                {/* User Selection */}
                <div className="p-6 border-b border-gray-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Select User</h4>
                  <div className="relative user-search-container">
                    <SearchBarWithDropdown
                      value={userSearchTerm}
                      onChange={(e) => {
                        setUserSearchTerm(e.target.value)
                        fetchAvailableUsers(e.target.value)
                      }}
                      onFocus={() => fetchAvailableUsers(userSearchTerm)}
                      items={availableUsers}
                      onSelect={(user) => {
                        setSelectedUserId(user.username)
                        setUserSearchTerm(`${user.username} (${user.email}) - ${user.role}`)
                      }}
                      filterFunction={(user, term) => {
                        const search = term.toLowerCase()
                        return (
                          user.username.toLowerCase().includes(search) ||
                          user.email.toLowerCase().includes(search) ||
                          user.role.toLowerCase().includes(search)
                        )
                      }}
                      renderItem={(user) => (
                        <div className={selectedUserId === user.username ? 'bg-blue-50' : ''}>
                          <div className="font-medium text-gray-900">{user.username}</div>
                          <div className="text-sm text-gray-600">{user.email}</div>
                          <div className="text-xs text-gray-500 mt-1">
                            Role: <span className="px-2 py-0.5 bg-gray-100 rounded text-gray-700">{user.role}</span>
                          </div>
                        </div>
                      )}
                      getItemKey={(user) => user.id}
                      label="User"
                      placeholder="Search users by name, email, or role..."
                      width="w-1/3"
                    />
                  </div>
                  {selectedUserId && (
                    <div className="mt-2 p-2 bg-green-50 border border-green-200 rounded text-sm text-green-800">
                      ✓ Selected: <span className="font-semibold">{selectedUserId}</span>
                    </div>
                  )}
                </div>

                {/* Window Time Period */}
                <div className="p-6 border-b border-gray-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Time Period</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Start Date &amp; Time</label>
                      <input
                        type="datetime-local"
                        value={windowStartAt}
                        onChange={(e) => setWindowStartAt(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">End Date &amp; Time</label>
                      <input
                        type="datetime-local"
                        value={windowEndAt}
                        onChange={(e) => setWindowEndAt(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                      />
                    </div>
                  </div>
                </div>

                {/* Scope Selection */}
                <div className="p-6 border-b border-gray-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Window Scope (Optional)</h4>
                  <p className="text-xs text-gray-600 mb-3">Optionally specify department/semester or course. Leave blank for student-specific only.</p>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Department (Optional)</label>
                      <select
                        value={windowDepartmentId}
                        onChange={(e) => setWindowDepartmentId(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                      >
                        <option value="">All Departments</option>
                        {departments.map((dept) => (
                          <option key={dept.id} value={dept.id}>{dept.name}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Semester (Optional)</label>
                      <select
                        value={windowSemester}
                        onChange={(e) => setWindowSemester(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                      >
                        <option value="">-- None --</option>
                        {[1, 2, 3, 4, 5, 6, 7, 8].map(sem => (
                          <option key={sem} value={sem}>{sem}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-600 mb-1">Course (Optional)</label>
                      <select
                        value={windowCourseId}
                        onChange={(e) => setWindowCourseId(e.target.value)}
                        className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                        disabled={!windowSemester}
                      >
                        <option value="">-- None --</option>
                        <option value="all">All Courses</option>
                        {windowCourses.map((course) => (
                          <option key={course.course_id} value={course.course_id}>
                            {course.course_code} - {course.course_name}
                          </option>
                        ))}
                      </select>
                      {!windowSemester && <p className="text-xs text-gray-500 mt-1">Select semester first</p>}
                    </div>
                  </div>
                </div>

                {/* Component Selection */}
                <div className="p-6 border-b border-gray-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Allowed Mark Components</h4>
                  <div className="flex items-center justify-center gap-6 mb-3 bg-gray-50 rounded-lg p-3">
                    <span className={`text-sm font-semibold transition-colors ${learningMode === 'PBL' ? 'text-primary' : 'text-gray-400'}`}>PBL</span>
                    <button
                      onClick={() => setLearningMode(learningMode === 'PBL' ? 'UAL' : 'PBL')}
                      className={`relative inline-flex h-8 w-16 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 ${learningMode === 'PBL' ? 'bg-primary' : 'bg-orange-600'}`}
                    >
                      <span className={`inline-block h-6 w-6 transform rounded-full bg-white shadow-lg transition-transform ${learningMode === 'PBL' ? 'translate-x-1' : 'translate-x-9'}`} />
                    </button>
                    <span className={`text-sm font-semibold transition-colors ${learningMode === 'UAL' ? 'text-orange-700' : 'text-gray-400'}`}>UAL</span>
                  </div>
                  {windowComponents.length > 0 && (
                    <div className="space-y-3">
                      {Object.entries(
                        windowComponents.reduce((groups, component) => {
                          const courseTypeName = component.course_type_name || 'Other'
                          if (!groups[courseTypeName]) groups[courseTypeName] = []
                          groups[courseTypeName].push(component)
                          return groups
                        }, {})
                      ).map(([courseTypeName, components]) => (
                        <div key={courseTypeName} className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                          <h4 className="text-xs font-semibold text-gray-600 uppercase tracking-wider mb-3">{courseTypeName}</h4>
                          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                            {components.map((component) => {
                              const selectedComponents = learningMode === 'PBL' ? selectedPBLComponents : selectedUALComponents
                              const setSelectedComponents = learningMode === 'PBL' ? setSelectedPBLComponents : setSelectedUALComponents
                              return (
                                <label key={component.id} className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer hover:text-blue-600">
                                  <input
                                    type="checkbox"
                                    checked={selectedComponents.includes(component.id)}
                                    onChange={(e) => {
                                      if (e.target.checked) {
                                        setSelectedComponents([...selectedComponents, component.id])
                                      } else {
                                        setSelectedComponents(selectedComponents.filter((id) => id !== component.id))
                                      }
                                    }}
                                    className="h-4 w-4 accent-blue-600 cursor-pointer"
                                  />
                                  <span className="font-medium">{component.name}</span>
                                </label>
                              )
                            })}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                  <p className="text-xs text-gray-500 mt-2">Leave all unchecked to allow all components.</p>
                </div>

                {/* Student Selection */}
                <div className="p-6">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Select Students</h4>
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-3 mb-4">
                    <input type="text" placeholder="Search by name/enrollment..." value={studentFilters.search}
                      onChange={(e) => setStudentFilters({ ...studentFilters, search: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500" />
                    <select value={studentFilters.department}
                      onChange={(e) => setStudentFilters({ ...studentFilters, department: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 bg-white">
                      <option value="">All Departments</option>
                      {departments.map(dept => (<option key={dept.id} value={dept.name}>{dept.name}</option>))}
                    </select>
                    <input type="number" placeholder="Year (optional)" value={studentFilters.year}
                      onChange={(e) => setStudentFilters({ ...studentFilters, year: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500" />
                    <input type="number" placeholder="Semester (optional)" value={studentFilters.semester}
                      onChange={(e) => setStudentFilters({ ...studentFilters, semester: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500" />
                  </div>
                  <div className="flex gap-2 mb-4">
                    <button onClick={fetchStudentsForAssignment} disabled={assignmentLoading}
                      className="px-4 py-2 bg-primary text-white rounded-lg disabled:bg-gray-400 transition-colors">
                      {assignmentLoading ? 'Loading...' : 'Apply Filters'}
                    </button>
                    {students.length > 0 && (
                      <>
                        <button onClick={selectAllStudents} className="px-4 py-2 border border-primary text-primary rounded-lg hover:bg-gray-100 transition-colors">
                          Select All ({students.length})
                        </button>
                        <button onClick={clearStudentSelection} className="px-4 py-2 border border-red-600 text-red-600 rounded-lg hover:bg-red-50 transition-colors">
                          Clear Selection
                        </button>
                      </>
                    )}
                  </div>
                  {students.length > 0 && (
                    <div className="border border-gray-300 rounded-lg overflow-hidden mb-4">
                      <div className="max-h-96 overflow-y-auto">
                        <table className="w-full">
                          <thead className="bg-gray-100 sticky top-0">
                            <tr>
                              {['Select', 'Enrollment No', 'Name', 'Department', 'Year', 'Semester'].map(h => (
                                <th key={h} className="p-3 text-left text-xs font-semibold text-gray-700">{h}</th>
                              ))}
                            </tr>
                          </thead>
                          <tbody>
                            {students.map(student => (
                              <tr key={student.student_id}
                                className={`border-t hover:bg-blue-50 cursor-pointer transition-colors ${selectedStudents.includes(student.student_id) ? 'bg-blue-50' : ''}`}
                                onClick={() => toggleStudentSelection(student.student_id)}>
                                <td className="p-3">
                                  <input type="checkbox" checked={selectedStudents.includes(student.student_id)}
                                    onChange={() => toggleStudentSelection(student.student_id)}
                                    className="h-4 w-4 accent-blue-600 cursor-pointer" />
                                </td>
                                <td className="p-3 text-sm">{student.enrollment_no}</td>
                                <td className="p-3 text-sm font-medium">{student.student_name}</td>
                                <td className="p-3 text-sm">{student.department}</td>
                                <td className="p-3 text-sm">{student.year}</td>
                                <td className="p-3 text-sm">{student.semester}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      <div className="bg-gray-50 px-4 py-3 border-t border-gray-200">
                        <p className="text-sm text-gray-600">{selectedStudents.length} of {students.length} student(s) selected</p>
                      </div>
                    </div>
                  )}
                  {students.length === 0 && !assignmentLoading && (
                    <div className="text-center py-12 text-gray-400 border border-gray-200 rounded-lg mb-4">
                      <p className="text-sm">Use filters above to search for students</p>
                    </div>
                  )}
                  <div className="flex justify-end gap-3">
                    {editingWindow && (
                      <button
                        onClick={() => {
                          setEditingWindow(null); setSelectedUserId(''); setSelectedStudents([])
                          setWindowDepartmentId(''); setWindowSemester(''); setWindowCourseId('')
                          setWindowStartAt(''); setWindowEndAt(''); setWindowEnabled(true)
                          setSelectedPBLComponents([]); setSelectedUALComponents([])
                        }}
                        className="px-4 py-2 bg-gray-500 text-white text-sm font-medium rounded-lg hover:bg-gray-600 transition-colors">
                        Cancel
                      </button>
                    )}
                    <button
                      onClick={assignStudentsToUser}
                      disabled={!selectedUserId || selectedStudents.length === 0 || !windowStartAt || !windowEndAt || assignmentLoading}
                      className="px-6 py-3 bg-primary text-white font-semibold rounded-lg disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors">
                      {assignmentLoading ? 'Creating Window...' : editingWindow ? 'Update Window & Assignments' : 'Create Window & Assign Students'}
                    </button>
                  </div>
                </div>
              </div>
              )}
            </div>
          </>
        )}

        {/* ── Existing Windows Tab ── */}
        {mainTab === 'existing' && (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100">
            {/* Card header with inline sub-toggle */}
            <div className="border-b border-gray-200 px-6 py-4 flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-gray-700">Existing Windows</h3>
                <div className="mt-3 inline-flex rounded-lg bg-gray-100 p-0.5 gap-0.5">
                  <button
                    onClick={() => setExistingTab('window-scope')}
                    className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-all ${existingTab === 'window-scope' ? 'bg-primary text-white shadow-sm' : 'text-gray-500 hover:text-gray-800'}`}
                  >
                    Window Scope
                  </button>
                  <button
                    onClick={() => setExistingTab('user-student')}
                    className={`px-4 py-1.5 text-xs font-semibold rounded-md transition-all ${existingTab === 'user-student' ? 'bg-primary text-white shadow-sm' : 'text-gray-500 hover:text-gray-800'}`}
                  >
                    User-Student Scope
                  </button>
                </div>
              </div>
              {existingTab === 'window-scope' ? (
                <span className="px-3 py-1 bg-background text-primary rounded-full text-sm font-semibold">
                  {existingWindows.length} {existingWindows.length === 1 ? 'window' : 'windows'}
                </span>
              ) : (
                <span className="px-3 py-1 bg-purple-100 text-purple-700 rounded-full text-xs font-semibold">
                  {userAssignedWindows.length} {userAssignedWindows.length === 1 ? 'window' : 'windows'}
                </span>
              )}
            </div>

            {/* Window Scope content */}
            {existingTab === 'window-scope' && (
              <div className="p-6">
                {existingWindows.length === 0 ? (
                  <div className="text-center py-12 text-gray-400">
                    <p className="text-sm">No windows configured yet</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {existingWindows.map((win) => {
                      const { status, color } = getWindowStatus(win)
                      return (
                        <div key={win.id} className="bg-gray-50 rounded-lg p-4 border border-gray-200 hover:border-blue-300 hover:shadow-sm transition-all">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-start gap-3 flex-1">
                              <div className="flex-shrink-0 w-12 h-12 rounded-lg bg-blue-50 border border-blue-200 flex flex-col items-center justify-center">
                                <span className="text-[9px] text-blue-400 font-semibold uppercase tracking-wider leading-none">ID</span>
                                <span className="text-base font-bold text-blue-700 leading-tight">{win.id}</span>
                              </div>
                              <div>
                                <p className="font-semibold text-gray-800 text-sm mb-2">{getScopeDescription(win)}</p>
                                <span className={`px-2 py-1 rounded text-xs font-medium ${color}`}>{status}</span>
                              </div>
                            </div>
                            <div className="flex gap-2 ml-4">
                              <button onClick={() => { editWindow(win); setMainTab('create'); setActiveTab('windows') }}
                                className="px-3 py-1.5 text-xs text-blue-600 hover:bg-blue-50 rounded-lg font-medium transition-colors">Edit</button>
                              <button onClick={() => deleteWindow(win.id)}
                                className="px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 rounded-lg font-medium transition-colors">Delete</button>
                            </div>
                          </div>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs text-gray-600">
                            <div className="bg-white rounded p-2 border border-gray-100">
                              <div className="text-gray-500 mb-1">Start</div>
                              <div className="font-medium">{new Date(win.start_at).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                            </div>
                            <div className="bg-white rounded p-2 border border-gray-100">
                              <div className="text-gray-500 mb-1">End</div>
                              <div className="font-medium">{new Date(win.end_at).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                            </div>
                            <div className="bg-white rounded p-2 border border-gray-100">
                              <div className="text-gray-500 mb-1">Components</div>
                              <div className="font-medium">{win.component_ids && win.component_ids.length > 0 ? `${win.component_ids.length} selected` : 'All allowed'}</div>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )}

            {/* User-Student Scope content */}
            {existingTab === 'user-student' && (
              <div className="p-6">
                {userAssignedWindows.length === 0 ? (
                  <div className="text-center py-12 text-gray-400">
                    <p className="text-sm">No user-assigned windows configured yet</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {userAssignedWindows.map((win) => {
                      const { status, color } = getWindowStatus(win)
                      return (
                        <div key={win.id} className="bg-background rounded-lg p-4 border border-purple-200 hover:border-purple-300 hover:shadow-sm transition-all">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex-1">
                              <p className="font-semibold text-gray-800 text-sm mb-2">
                                <span>User:</span> {win.user_username || `ID: ${win.user_id}`}
                                {win.course_name && <span className="text-gray-600"> • {win.course_name}</span>}
                                {win.department_name && <span className="text-gray-600"> • {win.department_name}</span>}
                                {win.semester && <span className="text-gray-600"> • Sem {win.semester}</span>}
                              </p>
                              <span className={`px-2 py-1 rounded text-xs font-medium ${color}`}>{status}</span>
                              {win.student_count > 0 && (
                                <span className="ml-2 px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs font-medium">
                                  {win.student_count} student{win.student_count !== 1 ? 's' : ''}
                                </span>
                              )}
                            </div>
                            <div className="flex gap-2 ml-4">
                              <button onClick={() => { editUserWindow(win); setMainTab('create'); setActiveTab('student-assignment') }}
                                className="px-3 py-1.5 text-xs text-purple-600 hover:bg-purple-100 rounded-lg font-medium transition-colors">Edit</button>
                              <button onClick={() => deleteWindow(win.id)}
                                className="px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 rounded-lg font-medium transition-colors">Delete</button>
                            </div>
                          </div>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 text-xs text-gray-600">
                            <div className="bg-white rounded p-2 border border-purple-100">
                              <div className="text-gray-500 mb-1">Start</div>
                              <div className="font-medium">{new Date(win.start_at).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                            </div>
                            <div className="bg-white rounded p-2 border border-purple-100">
                              <div className="text-gray-500 mb-1">End</div>
                              <div className="font-medium">{new Date(win.end_at).toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' })}</div>
                            </div>
                            <div className="bg-white rounded p-2 border border-purple-100">
                              <div className="text-gray-500 mb-1">Components</div>
                              <div className="font-medium">{win.component_ids && win.component_ids.length > 0 ? `${win.component_ids.length} selected` : 'All allowed'}</div>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* ── Extension Tab ── */}
        {mainTab === 'extension' && (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-16 text-center">
            <svg className="w-12 h-12 mx-auto text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
            </svg>
            <h3 className="text-base font-semibold text-gray-500">Window Extension</h3>
            <p className="text-sm text-gray-400 mt-1">This feature will be configured separately.</p>
          </div>
        )}
      </div>
    </MainLayout>
  )
}

export default MarkEntryPermissionsPage
