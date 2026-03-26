import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import * as XLSX from 'xlsx'
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

const getInnovativePracticeBaseName = (name = '') => {
  const normalized = String(name).replace(/\s+/g, ' ').trim()
  const match = normalized.match(/^(Innovative Practice\s+[12])\s*-\s*\(\s*[12]\s*\)$/i)
  return match ? match[1] : null
}

const normalizeInnovativePracticeSelections = (selectedIds = [], components = []) => {
  const selected = new Set(selectedIds)
  const groups = new Map()

  components.forEach((component) => {
    const baseName = getInnovativePracticeBaseName(component.name)
    if (!baseName) return

    const key = `${component.learning_mode_id || 0}|${baseName.toLowerCase()}`
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key).push(component.id)
  })

  groups.forEach((ids) => {
    if (ids.some((id) => selected.has(id))) {
      ids.forEach((id) => selected.add(id))
    }
  })

  return Array.from(selected)
}

const buildDisplayComponents = (components = []) => {
  const display = []
  const groupedIndex = new Map()

  components.forEach((component) => {
    const baseName = getInnovativePracticeBaseName(component.name)
    if (!baseName) {
      display.push({
        ...component,
        display_name: component.name,
        component_ids: [component.id],
      })
      return
    }

    const key = `${component.course_type_name || 'Other'}|${component.learning_mode_id || 0}|${baseName.toLowerCase()}`
    if (!groupedIndex.has(key)) {
      const item = {
        ...component,
        display_name: baseName,
        component_ids: [component.id],
      }
      groupedIndex.set(key, item)
      display.push(item)
      return
    }

    groupedIndex.get(key).component_ids.push(component.id)
  })

  return display
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
  const [windowName, setWindowName] = useState('')
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
  const [userRoleFilter, setUserRoleFilter] = useState('all')
  const [students, setStudents] = useState([])
  const [selectedStudents, setSelectedStudents] = useState([])
  const [studentFilters, setStudentFilters] = useState({
    department: '',
    year: '',
    semester: '',
    search: '',
    learning_mode: 'UAL&PBL'
  })
  const [assignmentLoading, setAssignmentLoading] = useState(false)
  const [userAssignedWindows, setUserAssignedWindows] = useState([])

  // Window Extension States
  const [pendingWindows, setPendingWindows] = useState([])
  const [loadingPendingWindows, setLoadingPendingWindows] = useState(false)
  const [expandedWindowId, setExpandedWindowId] = useState(null)
  const [closedPendingWindows, setClosedPendingWindows] = useState([])
  const [loadingClosedWindows, setLoadingClosedWindows] = useState(false)
  const [expandedClosedWindowId, setExpandedClosedWindowId] = useState(null)
  const [closedWindowView, setClosedWindowView] = useState({}) // { [windowId]: 'pending' | 'completed' }
  const [activeWindowView, setActiveWindowView] = useState({}) // { [windowId]: 'pending' | 'completed' }
  const [activeWindowLearningMode, setActiveWindowLearningMode] = useState({}) // { [windowId]: 'PBL' | 'UAL' | 'ALL' }
  const [closedWindowLearningMode, setClosedWindowLearningMode] = useState({}) // { [windowId]: 'PBL' | 'UAL' | 'ALL' }

  // Search/filter state for window lists
  const [existingWindowSearch, setExistingWindowSearch] = useState('')
  const [existingUserWindowSearch, setExistingUserWindowSearch] = useState('')
  const [activePendingSearch, setActivePendingSearch] = useState('')
  const [closedPendingSearch, setClosedPendingSearch] = useState('')
  const [selectedTeachers, setSelectedTeachers] = useState({}) // { [windowId]: Set of "teacherId|courseId" }
  const [extensionModal, setExtensionModal] = useState(null) // { windowId, teachers: [{teacher_id, course_id}] }
  const [extensionEndDate, setExtensionEndDate] = useState('')
  const [extensionLoading, setExtensionLoading] = useState(false)

  // Appeal states
  const [closedWindowAppeals, setClosedWindowAppeals] = useState({}) // { "windowId|teacherId|courseId" → appeal }
  const [activeWindowAppeals, setActiveWindowAppeals] = useState({}) // { "windowId|teacherId|courseId" → appeal } for active windows
  const [appealDetailModal, setAppealDetailModal] = useState(null) // appeal object

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
      fetchAvailableUsers(userSearchTerm)
      fetchStudentsForAssignment()
      fetchUserAssignedWindows()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab])

  const filteredAvailableUsers = useMemo(() => {
    const search = userSearchTerm.trim().toLowerCase()
    const roleFiltered = availableUsers.filter((user) => {
      if (userRoleFilter === 'all') return true
      return (user.role || '').toLowerCase() === userRoleFilter
    })

    const scored = roleFiltered.map((user) => {
      const username = (user.username || '').toLowerCase()
      const email = (user.email || '').toLowerCase()
      const role = (user.role || '').toLowerCase()

      let score = 0
      if (!search) score = 1
      else if (username === search) score = 100
      else if (username.startsWith(search)) score = 80
      else if (email.startsWith(search)) score = 60
      else if (username.includes(search)) score = 50
      else if (email.includes(search)) score = 40
      else if (role.includes(search)) score = 30

      return { user, score }
    })

    return scored
      .filter((item) => item.score > 0)
      .sort((a, b) => b.score - a.score || (a.user.username || '').localeCompare(b.user.username || ''))
      .map((item) => item.user)
  }, [availableUsers, userRoleFilter, userSearchTerm])

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

  // Fetch pending submissions when extension tab is opened
  useEffect(() => {
    if (mainTab === 'extension') {
      fetchPendingWindows()
      fetchClosedPendingWindows()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mainTab])

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
    // Skip loadWindowRule when editing - editWindow directly populates selections
    if (!editingWindow) {
      loadWindowRule()
    }
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
      setWindowEnabled(true)
      return
    }

    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-window?${query}`)
      if (!res.ok) throw new Error('Failed to fetch window rule')
      const data = await res.json()

      if (!data) {
        setWindowEnabled(true)
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

        const pblComponents = allComponents.filter(c => c.learning_mode_id === 2)
        const ualComponents = allComponents.filter(c => c.learning_mode_id === 1)
        setSelectedPBLComponents(normalizeInnovativePracticeSelections(pblIds, pblComponents))
        setSelectedUALComponents(normalizeInnovativePracticeSelections(ualIds, ualComponents))
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
    if (!windowName.trim()) {
      setMessage({ type: 'error', text: 'Window name is required.' })
      return
    }
    if (!windowStartAt || !windowEndAt) {
      setMessage({ type: 'error', text: 'Start and end dates are required.' })
      return
    }

    // Convert datetime-local to ISO 8601 format with timezone
    const startDate = new Date(windowStartAt)
    const endDate = new Date(windowEndAt)

    const selectedComponentIds = Array.from(new Set([...selectedPBLComponents, ...selectedUALComponents]))

    const payload = {
      start_at: startDate.toISOString(),
      end_at: endDate.toISOString(),
      enabled: windowEnabled,
      window_name: windowName.trim(),
      component_ids: selectedComponentIds.length > 0
        ? selectedComponentIds
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

  const fetchPendingWindows = async () => {
    setLoadingPendingWindows(true)
    try {
      const [windowsRes, appealsRes] = await Promise.all([
        fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?status=active`),
        fetch(`${API_BASE_URL}/mark-appeals?status=pending`),
      ])
      if (!windowsRes.ok) throw new Error('Failed to fetch pending submissions')
      const data = await windowsRes.json()
      setPendingWindows(data || [])
      // Build appeal lookup for active windows
      if (appealsRes.ok) {
        const appealsData = await appealsRes.json()
        const lookup = {}
        ;(appealsData || []).forEach(a => {
          const key = `${a.window_id}|${a.teacher_id}|${a.course_id}`
          lookup[key] = a
        })
        setActiveWindowAppeals(lookup)
      }
    } catch (error) {
      console.error('Error fetching pending submissions:', error)
      setPendingWindows([])
    } finally {
      setLoadingPendingWindows(false)
    }
  }

  const fetchClosedPendingWindows = async () => {
    setLoadingClosedWindows(true)
    try {
      const [windowsRes, appealsRes] = await Promise.all([
        fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?status=closed`),
        fetch(`${API_BASE_URL}/mark-appeals?status=pending`),
      ])
      if (!windowsRes.ok) throw new Error('Failed to fetch closed window submissions')
      const data = await windowsRes.json()
      setClosedPendingWindows(data || [])
      // Build appeal lookup
      if (appealsRes.ok) {
        const appealsData = await appealsRes.json()
        const lookup = {}
        ;(appealsData || []).forEach(a => {
          const key = `${a.window_id}|${a.teacher_id}|${a.course_id}`
          lookup[key] = a
        })
        setClosedWindowAppeals(lookup)
      }
    } catch (error) {
      console.error('Error fetching closed window submissions:', error)
      setClosedPendingWindows([])
    } finally {
      setLoadingClosedWindows(false)
    }
  }

  // Toggle teacher selection in a closed window's pending list
  const toggleTeacherSelection = (windowId, teacherId, courseId) => {
    const key = `${teacherId}|${courseId}`
    setSelectedTeachers(prev => {
      const currentSet = new Set(prev[windowId] || [])
      if (currentSet.has(key)) {
        currentSet.delete(key)
      } else {
        currentSet.add(key)
      }
      return { ...prev, [windowId]: currentSet }
    })
  }

  // Toggle all teachers in a window
  const toggleAllTeachers = (windowId, pendingList) => {
    setSelectedTeachers(prev => {
      const currentSet = new Set(prev[windowId] || [])
      const allKeys = pendingList.map(t => `${t.teacher_id}|${t.course_id}`)
      const allSelected = allKeys.length > 0 && allKeys.every(k => currentSet.has(k))
      if (allSelected) {
        // Deselect all
        return { ...prev, [windowId]: new Set() }
      } else {
        // Select all
        return { ...prev, [windowId]: new Set(allKeys) }
      }
    })
  }

  // Open extension modal for selected teachers
  const openExtensionModal = (windowId) => {
    const selected = selectedTeachers[windowId]
    if (!selected || selected.size === 0) return
    const teachers = Array.from(selected).map(key => {
      const [teacher_id, course_id] = key.split('|')
      return { teacher_id, course_id: parseInt(course_id) }
    })
    // Default to 24 hours from now
    const defaultEnd = new Date(Date.now() + 24 * 60 * 60 * 1000)
    const pad = (n) => String(n).padStart(2, '0')
    const defaultEndStr = `${defaultEnd.getFullYear()}-${pad(defaultEnd.getMonth() + 1)}-${pad(defaultEnd.getDate())}T${pad(defaultEnd.getHours())}:${pad(defaultEnd.getMinutes())}`
    setExtensionEndDate(defaultEndStr)
    setExtensionModal({ windowId, teachers })
  }

  // Submit the extension
  const submitExtension = async () => {
    if (!extensionModal || !extensionEndDate) return
    setExtensionLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows/extend-for-teachers`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          source_window_id: extensionModal.windowId,
          teachers: extensionModal.teachers,
          new_end_at: extensionEndDate,
        }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.message || 'Failed to extend window')
      setMessage({ type: 'success', text: data.message || `Extended window for ${data.created} teacher(s)` })
      setExtensionModal(null)
      setSelectedTeachers(prev => ({ ...prev, [extensionModal.windowId]: new Set() }))
      fetchClosedPendingWindows()
      fetchPendingWindows()
      fetchExistingWindows()
    } catch (error) {
      console.error('Error extending window:', error)
      setMessage({ type: 'error', text: error.message || 'Failed to extend window for selected teachers.' })
    } finally {
      setExtensionLoading(false)
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
    setWindowName(win.window_name || '')
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
    setWindowDepartmentId(win.department_id ? String(win.department_id) : '')
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

        const pblComponents = allComponents.filter(c => c.learning_mode_id === 2)
        const ualComponents = allComponents.filter(c => c.learning_mode_id === 1)
        setSelectedPBLComponents(normalizeInnovativePracticeSelections(allPBL, pblComponents))
        setSelectedUALComponents(normalizeInnovativePracticeSelections(allUAL, ualComponents))
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
    setWindowName(latestWindow.window_name || '')
    setWindowStartAt(formatDateTimeLocal(latestWindow.start_at))
    setWindowEndAt(formatDateTimeLocal(latestWindow.end_at))
    setWindowEnabled(latestWindow.enabled)

    // Directly separate window's component_ids into PBL and UAL
    if (win.component_ids && win.component_ids.length > 0) {
      try {
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
        win.component_ids.forEach(id => {
          const comp = allComponents.find(c => c.id === id)
          if (comp) {
            if (comp.learning_mode_id === 2) pblIds.push(id)
            else if (comp.learning_mode_id === 1) ualIds.push(id)
          }
        })
        const pblComponents = allComponents.filter(c => c.learning_mode_id === 2)
        const ualComponents = allComponents.filter(c => c.learning_mode_id === 1)
        setSelectedPBLComponents(normalizeInnovativePracticeSelections(pblIds, pblComponents))
        setSelectedUALComponents(normalizeInnovativePracticeSelections(ualIds, ualComponents))
      } catch (err) {
        console.error('Error loading component details for edit:', err)
        setSelectedPBLComponents([])
        setSelectedUALComponents([])
      }
    } else {
      setSelectedPBLComponents([])
      setSelectedUALComponents([])
    }

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

    const selectedComponentIds = Array.from(new Set([...selectedPBLComponents, ...selectedUALComponents]))

    const payload = {
      start_at: startDate.toISOString(),
      end_at: endDate.toISOString(),
      enabled: windowEnabled,
      window_name: windowName.trim(),
      component_ids: selectedComponentIds.length > 0
        ? selectedComponentIds
        : null,
      teacher_id: null,
      user_id: null,
      department_id: null,
      semester: null,
      course_id: null,
    }

    const originalTeacherID = editingWindow.teacher_id || null
    const originalUserID = editingWindow.user_username || null
    const originalDepartmentID = editingWindow.department_id ?? null
    const originalSemester = editingWindow.semester ?? null
    const originalCourseID = editingWindow.course_id ?? null
    const isLikelyDepartmentScope =
      windowScope === 'department_semester' ||
      windowScope === 'department_semester_course' ||
      (!originalTeacherID && !originalUserID && originalSemester !== null)

    if (windowScope === 'teacher_course') {
      payload.teacher_id = selectedTeacherId || originalTeacherID
      payload.course_id = selectedCourseId ? parseInt(selectedCourseId, 10) : originalCourseID
    }

    if (windowScope === 'department_semester') {
      if (windowDepartmentId === '0') payload.department_id = null
      else if (windowDepartmentId) payload.department_id = parseInt(windowDepartmentId, 10)
      else payload.department_id = originalDepartmentID

      payload.semester = windowSemester ? parseInt(windowSemester, 10) : originalSemester
    }

    if (windowScope === 'department_semester_course') {
      if (windowDepartmentId === '0') payload.department_id = null
      else if (windowDepartmentId) payload.department_id = parseInt(windowDepartmentId, 10)
      else payload.department_id = originalDepartmentID

      payload.semester = windowSemester ? parseInt(windowSemester, 10) : originalSemester
      payload.course_id = windowCourseId && windowCourseId !== 'all' ? parseInt(windowCourseId, 10) : originalCourseID
    }

    // Safety override for stale UI scope state during edit mode:
    // if the original window is dept-scoped, always carry dept/semester/course explicitly.
    if (isLikelyDepartmentScope) {
      if (windowDepartmentId === '0') payload.department_id = null
      else if (windowDepartmentId) payload.department_id = parseInt(windowDepartmentId, 10)
      else payload.department_id = originalDepartmentID

      payload.semester = windowSemester ? parseInt(windowSemester, 10) : originalSemester

      if (windowCourseId === 'all') payload.course_id = null
      else if (windowCourseId) payload.course_id = parseInt(windowCourseId, 10)
      else payload.course_id = originalCourseID
    }

    if (!payload.teacher_id && !payload.user_id && !payload.semester && !payload.course_id) {
      payload.teacher_id = originalTeacherID
      payload.user_id = originalUserID
      payload.department_id = payload.department_id ?? originalDepartmentID
      payload.semester = payload.semester ?? originalSemester
      payload.course_id = payload.course_id ?? originalCourseID
    }

    setWindowLoading(true)
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows/${editingWindow.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      try {
        await res.json()
      } catch {
        // no-op
      }

      if (!res.ok) {
        throw new Error('Failed to update window')
      }

      await fetchExistingWindows()

      setMessage({ type: 'success', text: 'Window updated successfully.' })
      setEditingWindow(null)
    } catch (error) {
      console.error('Error updating window:', error)
      setMessage({ type: 'error', text: 'Failed to update window.' })
    } finally {
      setWindowLoading(false)
    }
  }

  const cancelEdit = () => {
    setEditingWindow(null)
    setWindowName('')
    setWindowStartAt('')
    setWindowEndAt('')
    setWindowEnabled(true)
    setSelectedPBLComponents([])
    setSelectedUALComponents([])
  }

  const getScopeDescription = (window) => {
    if (window.teacher_id && window.course_id) {
      return `${window.teacher_name} - ${window.course_code} (Most Specific)`
    } else if (window.user_id && window.course_id) {
      return `${window.user_username || 'User'} - ${window.course_code}`
    } else if (window.department_id && window.semester && window.course_id) {
      return `${window.department_name} - Sem ${window.semester} - ${window.course_code}`
    } else if (window.department_id && window.semester) {
      return `${window.department_name} - Semester ${window.semester} (Least Specific)`
    } else if (!window.department_id && window.semester && window.course_id) {
      return `All Departments - Sem ${window.semester} - ${window.course_code}`
    } else if (!window.department_id && window.semester) {
      return `All Departments - Semester ${window.semester}`
    }

    const parts = []
    if (window.teacher_name) parts.push(window.teacher_name)
    if (window.user_username) parts.push(window.user_username)
    if (window.department_name) parts.push(window.department_name)
    if (window.semester) parts.push(`Sem ${window.semester}`)
    if (window.course_code) parts.push(window.course_code)
    return parts.length > 0 ? parts.join(' - ') : 'General Window'
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

  const downloadWindowTeacherDetails = async (win) => {
    try {
      const res = await fetch(`${API_BASE_URL}/mark-entry-windows/pending-submissions?window_id=${win.id}`)
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Failed to fetch window details for export')
      }

      const data = await res.json()
      const details = Array.isArray(data) && data.length > 0 ? data[0] : null
      if (!details) {
        setMessage({ type: 'error', text: 'No teacher data found for this window.' })
        return
      }

      const teacherRows = [
        ...(Array.isArray(details.pending_teachers) ? details.pending_teachers : []).map((teacher) => ({
          WindowID: details.window_id,
          WindowName: details.window_name || '',
          Department: details.department_name || 'All',
          Semester: details.semester || 'All',
          CourseCode: teacher.course_code || details.course_code || '',
          CourseName: teacher.course_name || details.course_name || '',
          TeacherID: teacher.teacher_id || '',
          TeacherName: teacher.teacher_name || '',
          LearningModes: Array.isArray(teacher.learning_modes) ? teacher.learning_modes.join(', ') : '',
          SubmissionStatus: 'Pending',
          SubmittedAt: '',
          StartAt: details.start_at || '',
          EndAt: details.end_at || '',
        })),
        ...(Array.isArray(details.completed_teachers) ? details.completed_teachers : []).map((teacher) => ({
          WindowID: details.window_id,
          WindowName: details.window_name || '',
          Department: details.department_name || 'All',
          Semester: details.semester || 'All',
          CourseCode: teacher.course_code || details.course_code || '',
          CourseName: teacher.course_name || details.course_name || '',
          TeacherID: teacher.teacher_id || '',
          TeacherName: teacher.teacher_name || '',
          LearningModes: Array.isArray(teacher.learning_modes) ? teacher.learning_modes.join(', ') : '',
          SubmissionStatus: 'Completed',
          SubmittedAt: teacher.submitted_at || '',
          StartAt: details.start_at || '',
          EndAt: details.end_at || '',
        })),
      ]

      if (teacherRows.length === 0) {
        setMessage({ type: 'error', text: 'No teacher data found for this window.' })
        return
      }

      const wb = XLSX.utils.book_new()
      const ws = XLSX.utils.json_to_sheet(teacherRows)
      XLSX.utils.book_append_sheet(wb, ws, 'Teacher Details')

      const safeName = String(details.window_name || `window_${details.window_id}`)
        .replace(/[^a-zA-Z0-9_-]+/g, '_')
        .replace(/^_+|_+$/g, '')
      XLSX.writeFile(wb, `mark_window_${safeName || details.window_id}_teachers.xlsx`)
    } catch (error) {
      console.error('Error downloading teacher details export:', error)
      setMessage({ type: 'error', text: error.message || 'Failed to download teacher details.' })
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

  const handleUserSearchFocus = () => {
    // On box click/focus, show full list for current role filter instead of narrowing to selected label text.
    if (userSearchTerm.trim() !== '') {
      setUserSearchTerm('')
    }
    fetchAvailableUsers('')
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
      if (studentFilters.learning_mode && studentFilters.learning_mode !== 'UAL&PBL') params.append('learning_mode', studentFilters.learning_mode)
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
    if (!windowName.trim()) {
      setMessage({ type: 'error', text: 'Window name is required.' })
      return
    }
    if (!selectedUserId || selectedStudents.length === 0 || !windowStartAt || !windowEndAt) {
      setMessage({ type: 'error', text: 'Please fill all required fields: user, time period, and at least one student.' })
      return
    }

    setAssignmentLoading(true)
    try {
      // Combine PBL and UAL components
      const allComponents = Array.from(new Set([...selectedPBLComponents, ...selectedUALComponents]))

      if (editingWindow) {
        // Update existing window
        const updateRes = await fetch(`${API_BASE_URL}/mark-entry-windows/${editingWindow.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            start_at: new Date(windowStartAt).toISOString(),
            end_at: new Date(windowEndAt).toISOString(),
            enabled: windowEnabled,
            window_name: windowName.trim(),
            component_ids: allComponents.length > 0 ? allComponents : null,
            teacher_id: null,
            user_id: selectedUserId || null,
            department_id: windowDepartmentId && windowDepartmentId !== '0' ? parseInt(windowDepartmentId, 10) : null,
            semester: windowSemester ? parseInt(windowSemester, 10) : null,
            course_id: windowCourseId && windowCourseId !== 'all' ? parseInt(windowCourseId, 10) : null,
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
            window_name: windowName.trim(),
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
      setWindowName('')
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
    <MainLayout
      title="Mark Permissions"
      subtitle="Manage mark entry windows and permissions"
      actions={
        <div className="bg-gray-50 rounded-xl shadow-md border border-gray-200 p-1 inline-flex gap-1 overflow-x-auto">
          {[
            { key: 'create', label: 'Mark Entry Window' },
            { key: 'existing', label: 'Existing Windows' },
            { key: 'extension', label: 'Window Extension' },
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => !tab.disabled && setMainTab(tab.key)}
              disabled={tab.disabled}
              className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all whitespace-nowrap ${
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
      }
    >
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

                {/* Window Name */}
                <div>
                  <label className="block text-sm font-semibold text-gray-700 mb-2">
                    Window Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={windowName}
                    onChange={(e) => setWindowName(e.target.value)}
                    placeholder="e.g. PT - 1, Mid Semester Exam, Final Lab..."
                    maxLength={100}
                    className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="text-xs text-gray-500 mt-1">{windowName.length}/100 characters</p>
                </div>

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
                              {department.department_name || department.name || `Department ${department.id}`}
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
                        buildDisplayComponents(windowComponents).reduce((groups, component) => {
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

                              const isChecked = component.component_ids.some((id) => selectedComponents.includes(id))

                              return (
                                <label
                                  key={`${component.id}-${component.display_name}`}
                                  className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer hover:text-blue-600"
                                >
                                  <input
                                    type="checkbox"
                                    checked={isChecked}
                                    onChange={(e) => {
                                      if (e.target.checked) {
                                        setSelectedComponents(Array.from(new Set([...selectedComponents, ...component.component_ids])))
                                      } else {
                                        setSelectedComponents(
                                          selectedComponents.filter((id) => !component.component_ids.includes(id))
                                        )
                                      }
                                    }}
                                    className="h-4 w-4 accent-blue-600 cursor-pointer"
                                  />
                                  <span className="font-medium">{component.display_name || component.name}</span>
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
                  <div className="mb-3 flex flex-wrap items-center gap-2">
                    {[
                      { key: 'all', label: 'All Roles' },
                      { key: 'teacher', label: 'Teacher' },
                      { key: 'hod', label: 'HOD' },
                      { key: 'coe', label: 'COE' },
                      { key: 'admin', label: 'Admin' },
                    ].map((roleItem) => (
                      <button
                        key={roleItem.key}
                        type="button"
                        onClick={() => setUserRoleFilter(roleItem.key)}
                        className={`px-3 py-1.5 text-xs font-semibold rounded-full border transition-colors ${
                          userRoleFilter === roleItem.key
                            ? 'bg-primary text-white border-primary'
                            : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'
                        }`}
                      >
                        {roleItem.label}
                      </button>
                    ))}
                  </div>
                  <div className="relative user-search-container">
                    <SearchBarWithDropdown
                      value={userSearchTerm}
                      onChange={(e) => {
                        setUserSearchTerm(e.target.value)
                        fetchAvailableUsers(e.target.value)
                      }}
                      onFocus={handleUserSearchFocus}
                      items={filteredAvailableUsers}
                      onSelect={(user) => {
                        setSelectedUserId(user.username)
                        setUserSearchTerm(user.username)
                      }}
                      filterFunction={() => true}
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
                      width="w-full md:w-2/3"
                    />
                  </div>
                  <p className="mt-2 text-xs text-gray-500">
                    Showing {filteredAvailableUsers.length} user{filteredAvailableUsers.length === 1 ? '' : 's'}
                  </p>
                  {selectedUserId && (
                    <div className="mt-2 p-2 bg-green-50 border border-green-200 rounded text-sm text-green-800">
                      ✓ Selected: <span className="font-semibold">{selectedUserId}</span>
                      <button
                        type="button"
                        onClick={() => {
                          setSelectedUserId('')
                          setUserSearchTerm('')
                        }}
                        className="ml-3 text-xs font-semibold text-green-700 underline hover:text-green-900"
                      >
                        Clear
                      </button>
                    </div>
                  )}
                </div>

                {/* Window Name */}
                <div className="p-6 border-b border-gray-100">
                  <h4 className="text-sm font-semibold text-gray-700 mb-3">Window Name <span className="text-red-500">*</span></h4>
                  <input
                    type="text"
                    value={windowName}
                    onChange={(e) => setWindowName(e.target.value)}
                    placeholder="e.g. PT - 1, Mid Semester Exam, Final Lab..."
                    maxLength={100}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500"
                  />
                  <p className="text-xs text-gray-500 mt-1">{windowName.length}/100 characters</p>
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
                          <option key={dept.id} value={dept.id}>{dept.department_name || dept.name || `Department ${dept.id}`}</option>
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
                        buildDisplayComponents(windowComponents).reduce((groups, component) => {
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
                              const isChecked = component.component_ids.some((id) => selectedComponents.includes(id))
                              return (
                                <label key={`${component.id}-${component.display_name}`} className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer hover:text-blue-600">
                                  <input
                                    type="checkbox"
                                    checked={isChecked}
                                    onChange={(e) => {
                                      if (e.target.checked) {
                                        setSelectedComponents(Array.from(new Set([...selectedComponents, ...component.component_ids])))
                                      } else {
                                        setSelectedComponents(selectedComponents.filter((id) => !component.component_ids.includes(id)))
                                      }
                                    }}
                                    className="h-4 w-4 accent-blue-600 cursor-pointer"
                                  />
                                  <span className="font-medium">{component.display_name || component.name}</span>
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
                  <div className="grid grid-cols-1 md:grid-cols-5 gap-3 mb-4">
                    <input type="text" placeholder="Search by name/enrollment..." value={studentFilters.search}
                      onChange={(e) => setStudentFilters({ ...studentFilters, search: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500" />
                    <select value={studentFilters.department}
                      onChange={(e) => setStudentFilters({ ...studentFilters, department: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 bg-white">
                      <option value="">All Departments</option>
                      {departments.map(dept => (
                        <option key={dept.id} value={dept.department_name || dept.name || ''}>
                          {dept.department_name || dept.name || `Department ${dept.id}`}
                        </option>
                      ))}
                    </select>
                    <select value={studentFilters.learning_mode}
                      onChange={(e) => setStudentFilters({ ...studentFilters, learning_mode: e.target.value })}
                      className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 bg-white">
                      <option value="UAL&PBL">UAL & PBL</option>
                      <option value="UAL">UAL</option>
                      <option value="PBL">PBL</option>
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
                          setWindowName(''); setWindowDepartmentId(''); setWindowSemester(''); setWindowCourseId('')
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
                <div className="flex items-center gap-3">
                  <input
                    type="text"
                    placeholder="Search by name, ID, dept, course…"
                    value={existingWindowSearch}
                    onChange={e => setExistingWindowSearch(e.target.value)}
                    className="w-56 border border-gray-300 rounded-lg px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <span className="px-3 py-1 bg-background text-primary rounded-full text-sm font-semibold">
                    {existingWindows.filter(w => {
                      const q = existingWindowSearch.toLowerCase()
                      return !q || String(w.id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q) || (w.course_name||'').toLowerCase().includes(q)
                    }).length} {existingWindows.length === 1 ? 'window' : 'windows'}
                  </span>
                </div>
              ) : (
                <div className="flex items-center gap-3">
                  <input
                    type="text"
                    placeholder="Search by name, ID, user, course…"
                    value={existingUserWindowSearch}
                    onChange={e => setExistingUserWindowSearch(e.target.value)}
                    className="w-56 border border-gray-300 rounded-lg px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <span className="px-3 py-1 bg-purple-100 text-purple-700 rounded-full text-xs font-semibold">
                    {userAssignedWindows.filter(w => {
                      const q = existingUserWindowSearch.toLowerCase()
                      return !q || String(w.id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.user_username||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q)
                    }).length} {userAssignedWindows.length === 1 ? 'window' : 'windows'}
                  </span>
                </div>
              )}
            </div>

            {/* Window Scope content */}
            {existingTab === 'window-scope' && (
              <div className="p-6">
                {existingWindows.length === 0 ? (
                  <div className="text-center py-12 text-gray-400">
                    <p className="text-sm">No windows configured yet</p>
                  </div>
                ) : (() => {
                  const filtered = existingWindows.filter(w => {
                    const q = existingWindowSearch.toLowerCase()
                    return !q || String(w.id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q) || (w.course_name||'').toLowerCase().includes(q)
                  })
                  return filtered.length === 0 ? (
                    <div className="text-center py-12 text-gray-400">
                      <p className="text-sm">No windows match &ldquo;{existingWindowSearch}&rdquo;</p>
                    </div>
                  ) : (
                  <div className="space-y-3">
                    {filtered.map((win) => {
                      const { status, color } = getWindowStatus(win)
                      return (
                        <div
                          key={win.id}
                          className="bg-gray-50 rounded-lg p-4 border border-gray-200 hover:border-blue-300 hover:shadow-sm transition-all cursor-pointer"
                          onClick={() => navigate(`/mark-entry-windows/${win.id}`)}
                        >
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-start gap-3 flex-1">
                              <div className="flex-shrink-0 w-12 h-12 rounded-lg bg-blue-50 border border-blue-200 flex flex-col items-center justify-center">
                                <span className="text-[9px] text-blue-400 font-semibold uppercase tracking-wider leading-none">ID</span>
                                <span className="text-base font-bold text-blue-700 leading-tight">{win.id}</span>
                              </div>
                              <div>
                                <p className="font-semibold text-gray-800 text-sm">{win.window_name || getScopeDescription(win)}</p>
                                {win.window_name && <p className="text-xs text-gray-500 mb-1">{getScopeDescription(win)}</p>}
                                <span className={`px-2 py-1 rounded text-xs font-medium ${color}`}>{status}</span>
                              </div>
                            </div>
                            <div className="flex gap-2 ml-4">
                              <button
                                onClick={(e) => { e.stopPropagation(); navigate(`/mark-entry-windows/${win.id}`) }}
                                className="px-3 py-1.5 text-xs text-primary hover:bg-blue-50 rounded-lg font-medium transition-colors"
                              >
                                View Details
                              </button>
                              <button
                                onClick={(e) => { e.stopPropagation(); downloadWindowTeacherDetails(win) }}
                                className="px-3 py-1.5 text-xs text-emerald-700 hover:bg-emerald-50 rounded-lg font-medium transition-colors"
                              >
                                Download Excel
                              </button>
                              <button onClick={(e) => { e.stopPropagation(); editWindow(win); setMainTab('create'); setActiveTab('windows') }}
                                className="px-3 py-1.5 text-xs text-blue-600 hover:bg-blue-50 rounded-lg font-medium transition-colors">Edit</button>
                              <button onClick={(e) => { e.stopPropagation(); deleteWindow(win.id) }}
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
                              <div className="font-medium">{win.window_name || (win.component_ids && win.component_ids.length > 0 ? `${win.component_ids.length} selected` : 'All allowed')}</div>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                  )
                })()}
              </div>
            )}

            {/* User-Student Scope content */}
            {existingTab === 'user-student' && (
              <div className="p-6">
                {userAssignedWindows.length === 0 ? (
                  <div className="text-center py-12 text-gray-400">
                    <p className="text-sm">No user-assigned windows configured yet</p>
                  </div>
                ) : (() => {
                  const filteredU = userAssignedWindows.filter(w => {
                    const q = existingUserWindowSearch.toLowerCase()
                    return !q || String(w.id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.user_username||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q)
                  })
                  return filteredU.length === 0 ? (
                    <div className="text-center py-12 text-gray-400">
                      <p className="text-sm">No windows match "{existingUserWindowSearch}"</p>
                    </div>
                  ) : (
                  <div className="space-y-3">
                    {filteredU.map((win) => {
                      const { status, color } = getWindowStatus(win)
                      return (
                        <div key={win.id} className="bg-background rounded-lg p-4 border border-purple-200 hover:border-purple-300 hover:shadow-sm transition-all">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex-1">
                              <p className="font-semibold text-gray-800 text-sm">
                                {win.window_name || (
                                  <>
                                    <span>User:</span> {win.user_username || `ID: ${win.user_id}`}
                                    {win.course_name && <span className="text-gray-600"> • {win.course_name}</span>}
                                    {win.department_name && <span className="text-gray-600"> • {win.department_name}</span>}
                                    {win.semester && <span className="text-gray-600"> • Sem {win.semester}</span>}
                                  </>
                                )}
                              </p>
                              {win.window_name && (
                                <p className="text-xs text-gray-500 mb-1">
                                  <span>User:</span> {win.user_username || `ID: ${win.user_id}`}
                                  {win.course_name && <span> • {win.course_name}</span>}
                                  {win.department_name && <span> • {win.department_name}</span>}
                                  {win.semester && <span> • Sem {win.semester}</span>}
                                </p>
                              )}
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
                              <div className="font-medium">{win.window_name || (win.component_ids && win.component_ids.length > 0 ? `${win.component_ids.length} selected` : 'All allowed')}</div>
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                  )
                })()}
              </div>
            )}
          </div>
        )}

        {/* ── Extension Tab ── */}
        {mainTab === 'extension' && (
          <div className="space-y-6">
            {/* ── Section 1: Active Windows - Pending Submissions ── */}
            <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h3 className="text-lg font-semibold text-gray-800">Active Windows - Pending Submissions</h3>
                  <p className="text-sm text-gray-500 mt-1">
                    Teachers who have not yet submitted marks for currently active windows
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <input
                    type="text"
                    placeholder="Search windows…"
                    value={activePendingSearch}
                    onChange={e => setActivePendingSearch(e.target.value)}
                    className="w-44 border border-gray-300 rounded-lg px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                  <button
                    onClick={fetchPendingWindows}
                    disabled={loadingPendingWindows}
                    className="px-4 py-2 bg-primary hover:bg-purple-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                  >
                    {loadingPendingWindows ? 'Refreshing...' : 'Refresh'}
                  </button>
                </div>
              </div>

              {loadingPendingWindows ? (
                <div className="py-16 text-center text-gray-400">
                  <div className="animate-spin w-8 h-8 border-4 border-gray-200 border-t-primary rounded-full mx-auto mb-4"></div>
                  Loading pending submissions...
                </div>
              ) : pendingWindows.length === 0 ? (
                <div className="py-12 text-center">
                  <svg className="w-12 h-12 mx-auto text-green-400 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <h3 className="text-base font-semibold text-gray-700 mb-1">All Caught Up!</h3>
                  <p className="text-sm text-gray-500">
                    All teachers have submitted their marks for active windows.
                  </p>
                </div>
              ) : (  
                <div className="space-y-4">
                  {pendingWindows.filter(w => {
                    const q = activePendingSearch.toLowerCase()
                    return !q || String(w.window_id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q)
                  }).map((window) => {
                    const isExpanded = expandedWindowId === window.window_id
                    const startDate = new Date(window.start_at)
                    const endDate = new Date(window.end_at)
                    const currentView = activeWindowView[window.window_id] || 'pending'
                    const learningModeFilter = activeWindowLearningMode[window.window_id] || 'ALL'
                    
                    // Total unfiltered counts for badges
                    const totalPending = (window.pending_teachers || []).length
                    const totalCompleted = (window.completed_teachers || []).length
                    
                    // Filter teachers based on learning mode selection
                    const filterByLearningMode = (teachers) => {
                      if (!window.has_pbl || !window.has_ual || learningModeFilter === 'ALL') {
                        return teachers
                      }
                      return teachers.filter(t => {
                        const modes = t.learning_modes || []
                        return modes.includes(learningModeFilter)
                      })
                    }
                    
                    const pendingList = filterByLearningMode(window.pending_teachers || [])
                    const completedList = filterByLearningMode(window.completed_teachers || [])
                    const showingList = currentView === 'completed' ? completedList : pendingList

                    return (
                      <div
                        key={window.window_id}
                        className="rounded-xl border border-gray-200 overflow-hidden hover:border-primary transition-colors"
                      >
                        <button
                          onClick={() => setExpandedWindowId(isExpanded ? null : window.window_id)}
                          className="w-full px-5 py-4 flex items-center gap-4 bg-gradient-to-r from-purple-50 to-white hover:from-purple-100 transition-colors"
                        >
                          <div className="flex-shrink-0 w-14 h-14 rounded-xl bg-primary/10 border border-primary/20 flex flex-col items-center justify-center">
                            <span className="text-[10px] font-semibold text-primary uppercase tracking-wider">Window</span>
                            <span className="text-lg font-bold text-primary">{window.window_id}</span>
                          </div>

                          <div className="flex-1 text-left min-w-0">
                            {window.window_name && (
                              <p className="font-semibold text-gray-800 text-sm mb-1">{window.window_name}</p>
                            )}
                            <div className="flex items-center gap-2 flex-wrap mb-1">
                              {window.department_name && (
                                <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded-full">
                                  {window.department_name}
                                </span>
                              )}
                              {!window.department_name && window.department_id && (
                                <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded-full">
                                  Dept: {window.department_id}
                                </span>
                              )}
                              {window.semester && (
                                <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-medium rounded-full">
                                  Sem {window.semester}
                                </span>
                              )}
                              {window.course_code && (
                                <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded-full">
                                  {window.course_code}
                                </span>
                              )}
                              {!window.course_code && window.course_id && (
                                <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded-full">
                                  Course ID: {window.course_id}
                                </span>
                              )}
                              <span className="px-2 py-0.5 bg-red-100 text-red-700 text-xs font-bold rounded-full">
                                {totalPending} Pending
                              </span>
                              <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-bold rounded-full">
                                {totalCompleted} Submitted
                              </span>
                              <span className="px-2 py-0.5 bg-green-100 text-green-800 text-xs font-semibold rounded-full">
                                Active
                              </span>
                            </div>
                            <div className="text-xs text-gray-500">
                              {startDate.toLocaleString()} &rarr; {endDate.toLocaleString()}
                            </div>
                          </div>

                          <svg
                            className={`w-5 h-5 text-gray-400 flex-shrink-0 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            strokeWidth={2}
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                          </svg>
                        </button>

                        {isExpanded && (
                          <div className="border-t border-gray-200">
                            {/* PBL/UAL Filter Toggle (only show if window has both) */}
                            {(() => {
                              console.log(`Active Window ${window.window_id}:`, { has_pbl: window.has_pbl, has_ual: window.has_ual })
                              return window.has_pbl && window.has_ual
                            })() && (
                              <div className="px-5 py-3 bg-purple-50 border-b border-purple-100 flex items-center justify-between">
                                <div className="text-xs text-gray-600 font-medium">Filter by Learning Mode:</div>
                                <div className="inline-flex rounded-lg bg-purple-200 p-0.5">
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setActiveWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'ALL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      (activeWindowLearningMode[window.window_id] || 'ALL') === 'ALL'
                                        ? 'bg-purple-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    All
                                  </button>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setActiveWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'PBL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      activeWindowLearningMode[window.window_id] === 'PBL'
                                        ? 'bg-blue-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    PBL
                                  </button>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setActiveWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'UAL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      activeWindowLearningMode[window.window_id] === 'UAL'
                                        ? 'bg-orange-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    UAL
                                  </button>
                                </div>
                              </div>
                            )}

                            {/* Toggle between Pending and Submitted */}
                            <div className="px-5 py-3 bg-gray-50 border-b border-gray-100 flex items-center">
                              <div className="inline-flex rounded-lg bg-gray-200 p-0.5">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    setActiveWindowView(prev => ({ ...prev, [window.window_id]: 'pending' }))
                                  }}
                                  className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                    currentView === 'pending'
                                      ? 'bg-red-500 text-white shadow-sm'
                                      : 'text-gray-600 hover:text-gray-800'
                                  }`}
                                >
                                  Not Submitted ({pendingList.length})
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    setActiveWindowView(prev => ({ ...prev, [window.window_id]: 'completed' }))
                                  }}
                                  className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                    currentView === 'completed'
                                      ? 'bg-green-500 text-white shadow-sm'
                                      : 'text-gray-600 hover:text-gray-800'
                                  }`}
                                >
                                  Submitted ({completedList.length})
                                </button>
                              </div>
                            </div>

                            {showingList.length === 0 ? (
                              <div className="py-8 text-center text-gray-400 text-sm">
                                {currentView === 'completed' ? 'No teachers have submitted marks for this window yet.' : 'All teachers have submitted marks for this window!'}
                              </div>
                            ) : (
                              <div className="overflow-x-auto">
                                <table className="w-full text-sm">
                                  <thead className={currentView === 'completed' ? 'bg-green-50' : 'bg-red-50'}>
                                    <tr>
                                      {['#', 'Teacher ID', 'Teacher Name', 'Course Code', 'Course Name', ...(currentView === 'completed' ? ['Submitted At', 'Appeal'] : [])].map(h => (
                                        <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                                          {h}
                                        </th>
                                      ))}
                                    </tr>
                                  </thead>
                                  <tbody className="divide-y divide-gray-100">
                                    {showingList.map((teacher, idx) => (
                                      <tr key={`${teacher.teacher_id}-${teacher.course_id}`} className={idx % 2 === 0 ? 'bg-white' : (currentView === 'completed' ? 'bg-green-50/30' : 'bg-red-50/30')}>
                                        <td className="px-4 py-3 text-gray-400 text-xs">{idx + 1}</td>
                                        <td className="px-4 py-3 font-mono text-gray-700 text-xs">{teacher.teacher_id}</td>
                                        <td className="px-4 py-3 font-medium text-gray-800">{teacher.teacher_name}</td>
                                        <td className="px-4 py-3">
                                          <span className="px-2 py-0.5 bg-violet-100 text-violet-700 text-xs font-medium rounded-full">
                                            {teacher.course_code}
                                          </span>
                                        </td>
                                        <td className="px-4 py-3 text-gray-600 text-xs">{teacher.course_name}</td>
                                        {currentView === 'completed' && (
                                          <td className="px-4 py-3 text-gray-500 text-xs">
                                            {teacher.submitted_at ? new Date(teacher.submitted_at).toLocaleString() : '—'}
                                          </td>
                                        )}
                                        {currentView === 'completed' && (() => {
                                          const appealKey = `${window.window_id}|${teacher.teacher_id}|${teacher.course_id}`
                                          const appeal = activeWindowAppeals[appealKey]
                                          return (
                                            <td className="px-4 py-3">
                                              {appeal ? (
                                                <button
                                                  onClick={() => setAppealDetailModal(appeal)}
                                                  className="inline-flex items-center gap-1 px-2 py-1 bg-amber-100 border border-amber-300 text-amber-700 text-xs font-medium rounded-full hover:bg-amber-200 transition-colors"
                                                >
                                                  <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                                                  </svg>
                                                  Appeal
                                                </button>
                                              ) : (
                                                <span className="text-gray-300 text-xs">—</span>
                                              )}
                                            </td>
                                          )
                                        })()}
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>

            {/* ── Section 2: Closed Windows - Incomplete Submissions ── */}
            <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h3 className="text-lg font-semibold text-gray-800">Closed Windows - Incomplete Submissions</h3>
                  <p className="text-sm text-gray-500 mt-1">
                    Teachers who did not complete mark entry before the window deadline passed
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <input
                    type="text"
                    placeholder="Search windows…"
                    value={closedPendingSearch}
                    onChange={e => setClosedPendingSearch(e.target.value)}
                    className="w-44 border border-gray-300 rounded-lg px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-amber-500"
                  />
                  <button
                    onClick={fetchClosedPendingWindows}
                    disabled={loadingClosedWindows}
                    className="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                  >
                    {loadingClosedWindows ? 'Refreshing...' : 'Refresh'}
                  </button>
                </div>
              </div>

              {loadingClosedWindows ? (
                <div className="py-16 text-center text-gray-400">
                  <div className="animate-spin w-8 h-8 border-4 border-gray-200 border-t-amber-500 rounded-full mx-auto mb-4"></div>
                  Loading closed window data...
                </div>
              ) : closedPendingWindows.length === 0 ? (
                <div className="py-12 text-center">
                  <svg className="w-12 h-12 mx-auto text-green-400 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <h3 className="text-base font-semibold text-gray-700 mb-1">No Overdue Submissions</h3>
                  <p className="text-sm text-gray-500">
                    All teachers completed their mark entries for past windows, or no windows have closed yet.
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {closedPendingWindows.filter(w => {
                    const q = closedPendingSearch.toLowerCase()
                    return !q || String(w.window_id).includes(q) || (w.window_name||'').toLowerCase().includes(q) || (w.department_name||'').toLowerCase().includes(q) || (w.course_code||'').toLowerCase().includes(q)
                  }).map((window) => {
                    const isExpanded = expandedClosedWindowId === window.window_id
                    const startDate = new Date(window.start_at)
                    const endDate = new Date(window.end_at)
                    const currentView = closedWindowView[window.window_id] || 'pending'
                    const learningModeFilter = closedWindowLearningMode[window.window_id] || 'ALL'
                    
                    // Total unfiltered counts for badges
                    const totalPending = (window.pending_teachers || []).length
                    const totalCompleted = (window.completed_teachers || []).length
                    
                    // Filter teachers based on learning mode selection
                    const filterByLearningMode = (teachers) => {
                      if (!window.has_pbl || !window.has_ual || learningModeFilter === 'ALL') {
                        return teachers
                      }
                      return teachers.filter(t => {
                        const modes = t.learning_modes || []
                        return modes.includes(learningModeFilter)
                      })
                    }
                    
                    const pendingList = filterByLearningMode(window.pending_teachers || [])
                    const completedList = filterByLearningMode(window.completed_teachers || [])
                    const showingList = currentView === 'completed' ? completedList : pendingList
                    const windowSelected = selectedTeachers[window.window_id] || new Set()
                    const allPendingKeys = pendingList.map(t => `${t.teacher_id}|${t.course_id}`)
                    const allPendingSelected = allPendingKeys.length > 0 && allPendingKeys.every(k => windowSelected.has(k))
                    const someSelected = windowSelected.size > 0

                    // Calculate how long ago the window closed
                    const closedAgo = Math.floor((Date.now() - endDate.getTime()) / (1000 * 60 * 60 * 24))
                    const closedAgoText = closedAgo === 0 ? 'today' : closedAgo === 1 ? '1 day ago' : `${closedAgo} days ago`
                    
                    return (
                      <div
                        key={window.window_id}
                        className="rounded-xl border border-amber-200 overflow-hidden hover:border-amber-400 transition-colors"
                      >
                        <button
                          onClick={() => setExpandedClosedWindowId(isExpanded ? null : window.window_id)}
                          className="w-full px-5 py-4 flex items-center gap-4 bg-gradient-to-r from-amber-50 to-white hover:from-amber-100 transition-colors"
                        >
                          <div className="flex-shrink-0 w-14 h-14 rounded-xl bg-amber-100 border border-amber-200 flex flex-col items-center justify-center">
                            <span className="text-[10px] font-semibold text-amber-700 uppercase tracking-wider">Window</span>
                            <span className="text-lg font-bold text-amber-700">{window.window_id}</span>
                          </div>

                          <div className="flex-1 text-left min-w-0">
                            {window.window_name && (
                              <p className="font-semibold text-gray-800 text-sm mb-1">{window.window_name}</p>
                            )}
                            <div className="flex items-center gap-2 flex-wrap mb-1">
                              {window.department_name && (
                                <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded-full">
                                  {window.department_name}
                                </span>
                              )}
                              {!window.department_name && window.department_id && (
                                <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded-full">
                                  Dept: {window.department_id}
                                </span>
                              )}
                              {window.semester && (
                                <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-medium rounded-full">
                                  Sem {window.semester}
                                </span>
                              )}
                              {window.course_code && (
                                <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded-full">
                                  {window.course_code}
                                </span>
                              )}
                              {!window.course_code && window.course_id && (
                                <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs font-medium rounded-full">
                                  Course ID: {window.course_id}
                                </span>
                              )}
                              <span className="px-2 py-0.5 bg-red-100 text-red-700 text-xs font-bold rounded-full">
                                {totalPending} Not Submitted
                              </span>
                              <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-bold rounded-full">
                                {totalCompleted} Completed
                              </span>
                              <span className="px-2 py-0.5 bg-amber-100 text-amber-800 text-xs font-semibold rounded-full">
                                Closed {closedAgoText}
                              </span>
                            </div>
                            <div className="text-xs text-gray-500">
                              {startDate.toLocaleString()} &rarr; {endDate.toLocaleString()}
                            </div>
                          </div>

                          <svg
                            className={`w-5 h-5 text-gray-400 flex-shrink-0 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            strokeWidth={2}
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                          </svg>
                        </button>

                        {isExpanded && (
                          <div className="border-t border-amber-200">
                            {/* PBL/UAL Filter Toggle (only show if window has both) */}
                            {(() => {
                              console.log(`Closed Window ${window.window_id}:`, { has_pbl: window.has_pbl, has_ual: window.has_ual })
                              return window.has_pbl && window.has_ual
                            })() && (
                              <div className="px-5 py-3 bg-purple-50 border-b border-purple-100 flex items-center justify-between">
                                <div className="text-xs text-gray-600 font-medium">Filter by Learning Mode:</div>
                                <div className="inline-flex rounded-lg bg-purple-200 p-0.5">
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setClosedWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'ALL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      (closedWindowLearningMode[window.window_id] || 'ALL') === 'ALL'
                                        ? 'bg-purple-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    All
                                  </button>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setClosedWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'PBL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      closedWindowLearningMode[window.window_id] === 'PBL'
                                        ? 'bg-blue-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    PBL
                                  </button>
                                  <button
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      setClosedWindowLearningMode(prev => ({ ...prev, [window.window_id]: 'UAL' }))
                                    }}
                                    className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                      closedWindowLearningMode[window.window_id] === 'UAL'
                                        ? 'bg-orange-600 text-white shadow-sm'
                                        : 'text-gray-700 hover:text-gray-900'
                                    }`}
                                  >
                                    UAL
                                  </button>
                                </div>
                              </div>
                            )}

                            {/* Toggle switch between Pending and Completed + Extend button */}
                            <div className="px-5 py-3 bg-gray-50 border-b border-amber-100 flex items-center justify-between">
                              <div className="inline-flex rounded-lg bg-gray-200 p-0.5">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    setClosedWindowView(prev => ({ ...prev, [window.window_id]: 'pending' }))
                                  }}
                                  className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                    currentView === 'pending'
                                      ? 'bg-red-500 text-white shadow-sm'
                                      : 'text-gray-600 hover:text-gray-800'
                                  }`}
                                >
                                  Not Submitted ({pendingList.length})
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    setClosedWindowView(prev => ({ ...prev, [window.window_id]: 'completed' }))
                                  }}
                                  className={`px-3 py-1.5 text-xs font-semibold rounded-md transition-all ${
                                    currentView === 'completed'
                                      ? 'bg-green-500 text-white shadow-sm'
                                      : 'text-gray-600 hover:text-gray-800'
                                  }`}
                                >
                                  Completed ({completedList.length})
                                </button>
                              </div>

                              {currentView === 'pending' && someSelected && (
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation()
                                    openExtensionModal(window.window_id)
                                  }}
                                  className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1.5 shadow-sm"
                                >
                                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                                  </svg>
                                  Extend for {windowSelected.size} Selected
                                </button>
                              )}
                            </div>

                            {showingList.length === 0 ? (
                              <div className="py-8 text-center text-gray-400 text-sm">
                                {currentView === 'completed' ? 'No teachers have submitted marks for this window yet.' : 'All teachers have submitted marks for this window!'}
                              </div>
                            ) : (
                              <div className="overflow-x-auto">
                                <table className="w-full text-sm">
                                  <thead className={currentView === 'completed' ? 'bg-green-50' : 'bg-amber-50'}>
                                    <tr>
                                      {currentView === 'pending' && (
                                        <th className="px-4 py-3 text-left w-10">
                                          <input
                                            type="checkbox"
                                            checked={allPendingSelected}
                                            onChange={(e) => {
                                              e.stopPropagation()
                                              toggleAllTeachers(window.window_id, pendingList)
                                            }}
                                            className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                          />
                                        </th>
                                      )}
                                      {['#', 'Teacher ID', 'Teacher Name', 'Course Code', 'Course Name', ...(currentView === 'completed' ? ['Submitted At', 'Appeal'] : ['Appeal'])].map(h => (
                                        <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                                          {h}
                                        </th>
                                      ))}
                                    </tr>
                                  </thead>
                                  <tbody className="divide-y divide-gray-100">
                                    {showingList.map((teacher, idx) => {
                                      const tKey = `${teacher.teacher_id}|${teacher.course_id}`
                                      const isSelected = windowSelected.has(tKey)
                                      return (
                                        <tr
                                          key={tKey}
                                          className={`${isSelected ? 'bg-indigo-50' : (idx % 2 === 0 ? 'bg-white' : (currentView === 'completed' ? 'bg-green-50/30' : 'bg-amber-50/30'))} ${currentView === 'pending' ? 'cursor-pointer hover:bg-indigo-50/50' : ''}`}
                                          onClick={currentView === 'pending' ? () => toggleTeacherSelection(window.window_id, teacher.teacher_id, teacher.course_id) : undefined}
                                        >
                                          {currentView === 'pending' && (
                                            <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                                              <input
                                                type="checkbox"
                                                checked={isSelected}
                                                onChange={() => toggleTeacherSelection(window.window_id, teacher.teacher_id, teacher.course_id)}
                                                className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
                                              />
                                            </td>
                                          )}
                                          <td className="px-4 py-3 text-gray-400 text-xs">{idx + 1}</td>
                                          <td className="px-4 py-3 font-mono text-gray-700 text-xs">{teacher.teacher_id}</td>
                                          <td className="px-4 py-3 font-medium text-gray-800">{teacher.teacher_name}</td>
                                          <td className="px-4 py-3">
                                            <span className="px-2 py-0.5 bg-violet-100 text-violet-700 text-xs font-medium rounded-full">
                                              {teacher.course_code}
                                            </span>
                                          </td>
                                          <td className="px-4 py-3 text-gray-600 text-xs">{teacher.course_name}</td>
                                          {currentView === 'completed' && (
                                            <td className="px-4 py-3 text-gray-500 text-xs">
                                              {teacher.submitted_at ? new Date(teacher.submitted_at).toLocaleString() : '—'}
                                            </td>
                                          )}
                                          {/* Appeal column — shown for both pending and completed teachers */}
                                          {(() => {
                                            const appealKey = `${window.window_id}|${teacher.teacher_id}|${teacher.course_id}`
                                            const appeal = closedWindowAppeals[appealKey]
                                            return (
                                              <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                                                {appeal ? (
                                                  <button
                                                    onClick={() => setAppealDetailModal(appeal)}
                                                    className="inline-flex items-center space-x-1 px-2 py-1 bg-amber-100 border border-amber-300 text-amber-700 text-xs font-medium rounded-full hover:bg-amber-200 transition-colors"
                                                  >
                                                    <svg className="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
                                                    </svg>
                                                    <span>Appeal</span>
                                                  </button>
                                                ) : (
                                                  <span className="text-gray-300 text-xs">—</span>
                                                )}
                                              </td>
                                            )
                                          })()}
                                        </tr>
                                      )
                                    })}
                                  </tbody>
                                </table>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>

            {/* Extension Modal */}
            {extensionModal && (
              <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setExtensionModal(null)}>
                <div className="bg-white rounded-2xl shadow-2xl w-full max-w-lg mx-4 overflow-hidden" onClick={(e) => e.stopPropagation()}>
                  <div className="px-6 py-5 bg-gradient-to-r from-indigo-600 to-indigo-700 text-white">
                    <h3 className="text-lg font-bold">Extend Window for Selected Teachers</h3>
                    <p className="text-indigo-200 text-sm mt-1">
                      Creating individual windows for {extensionModal.teachers.length} teacher(s) from Window #{extensionModal.windowId}
                    </p>
                  </div>
                  <div className="p-6 space-y-4">
                    <div>
                      <label className="block text-sm font-semibold text-gray-700 mb-1.5">New Window End Date & Time</label>
                      <input
                        type="datetime-local"
                        value={extensionEndDate}
                        onChange={(e) => setExtensionEndDate(e.target.value)}
                        className="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-sm"
                      />
                      <p className="text-xs text-gray-400 mt-1">Window will start immediately and end at the selected date/time.</p>
                    </div>

                    <div className="bg-gray-50 rounded-lg p-3">
                      <p className="text-xs font-semibold text-gray-600 mb-2">Selected Teachers:</p>
                      <div className="space-y-1 max-h-40 overflow-y-auto">
                        {extensionModal.teachers.map((t, i) => {
                          const win = closedPendingWindows.find(w => w.window_id === extensionModal.windowId)
                          const teacherData = win?.pending_teachers?.find(pt => pt.teacher_id === t.teacher_id && pt.course_id === t.course_id)
                            || win?.completed_teachers?.find(pt => pt.teacher_id === t.teacher_id && pt.course_id === t.course_id)
                          return (
                            <div key={i} className="flex items-center gap-2 text-xs text-gray-600">
                              <span className="w-4 h-4 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center text-[10px] font-bold flex-shrink-0">{i + 1}</span>
                              <span className="font-mono">{t.teacher_id}</span>
                              {teacherData && <span className="text-gray-400">— {teacherData.teacher_name} ({teacherData.course_code})</span>}
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  </div>
                  <div className="px-6 py-4 bg-gray-50 border-t flex items-center justify-end gap-3">
                    <button
                      onClick={() => setExtensionModal(null)}
                      className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={submitExtension}
                      disabled={extensionLoading || !extensionEndDate}
                      className="px-5 py-2 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition-colors disabled:opacity-50 flex items-center gap-2"
                    >
                      {extensionLoading ? (
                        <>
                          <div className="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
                          Extending...
                        </>
                      ) : (
                        <>
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          Extend Window
                        </>
                      )}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Appeal Detail Modal */}
            {appealDetailModal && (() => {
              // Three categories:
              // 1. Active window appeal — teacher submitted & window still open → delete submission, re-enter
              // 2. Submitted + expired appeal — teacher submitted & window now closed → extend window for them
              // 3. Missed submission appeal — teacher never submitted, window closed → extend window
              const isActiveWindowAppeal = pendingWindows.some(w => w.window_id === appealDetailModal.window_id)
              const isSubmittedExpiredAppeal = !isActiveWindowAppeal && closedPendingWindows.some(w =>
                w.window_id === appealDetailModal.window_id &&
                (w.completed_teachers || []).some(t => t.teacher_id === appealDetailModal.teacher_id && t.course_id === appealDetailModal.course_id)
              )
              const modalTitle = isActiveWindowAppeal
                ? 'Post-Submission Appeal'
                : isSubmittedExpiredAppeal
                  ? 'Post-Submission Amendment Appeal'
                  : 'Missed Submission Appeal'
              const modalSubtitle = isActiveWindowAppeal
                ? 'Teacher submitted marks but is requesting to amend them (window still active)'
                : isSubmittedExpiredAppeal
                  ? 'Teacher submitted before window closed and now wants to amend — extend window to allow re-entry'
                  : 'Teacher missed the deadline — review and extend the window if approved'
              return (
              <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setAppealDetailModal(null)}>
                <div className="bg-white rounded-2xl shadow-2xl w-full max-w-md mx-4 overflow-hidden" onClick={(e) => e.stopPropagation()}>
                  <div className="px-6 py-5 bg-gradient-to-r from-amber-500 to-amber-600 text-white">
                    <h3 className="text-lg font-bold">{modalTitle}</h3>
                    <p className="text-amber-100 text-sm mt-1">{modalSubtitle}</p>
                  </div>
                  <div className="px-6 py-5 space-y-4">
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div>
                        <p className="text-xs text-gray-500 uppercase font-semibold mb-1">Teacher</p>
                        <p className="font-medium text-gray-900">{appealDetailModal.teacher_name}</p>
                        <p className="text-xs text-gray-500 font-mono">{appealDetailModal.teacher_id}</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500 uppercase font-semibold mb-1">Course</p>
                        <p className="font-medium text-gray-900">{appealDetailModal.course_name}</p>
                        <p className="text-xs text-gray-500">{appealDetailModal.course_code}</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500 uppercase font-semibold mb-1">Window</p>
                        <p className="text-sm text-gray-700">{appealDetailModal.window_name || `#${appealDetailModal.window_id}`}</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500 uppercase font-semibold mb-1">Submitted On</p>
                        <p className="text-sm text-gray-700">{new Date(appealDetailModal.created_at).toLocaleString()}</p>
                      </div>
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 uppercase font-semibold mb-1">Reason</p>
                      <p className="text-sm text-gray-800 bg-amber-50 border border-amber-100 rounded-lg px-3 py-3 leading-relaxed">
                        {appealDetailModal.reason}
                      </p>
                    </div>
                  </div>
                  <div className="px-6 py-4 bg-gray-50 border-t flex items-center justify-end gap-3">
                    <button
                      onClick={() => setAppealDetailModal(null)}
                      className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
                    >
                      Close
                    </button>
                    <button
                      onClick={async () => {
                        if (isActiveWindowAppeal) {
                          // Active window: delete the submission record so teacher can re-enter marks
                          try {
                            await fetch(`${API_BASE_URL}/mark-appeals/${appealDetailModal.id}/resolve`, {
                              method: 'POST',
                              headers: { 'Content-Type': 'application/json' },
                              body: JSON.stringify({
                                resolved_by: localStorage.getItem('username') || 'coe',
                                status: 'resolved',
                                delete_submission: true,
                              }),
                            })
                          } catch (_) {}
                          setAppealDetailModal(null)
                          setMessage({ type: 'success', text: 'Appeal approved — mark submission cleared. Teacher can now re-enter marks.' })
                          fetchPendingWindows()
                        } else {
                          // Closed window (missed OR submitted+expired): pre-select teacher, resolve appeal, open extension modal
                          const wid = appealDetailModal.window_id
                          const tid = appealDetailModal.teacher_id
                          const cid = appealDetailModal.course_id
                          setSelectedTeachers(prev => {
                            const key = `${tid}|${cid}`
                            const currentSet = new Set(prev[wid] || [])
                            currentSet.add(key)
                            return { ...prev, [wid]: currentSet }
                          })
                          try {
                            await fetch(`${API_BASE_URL}/mark-appeals/${appealDetailModal.id}/resolve`, {
                              method: 'POST',
                              headers: { 'Content-Type': 'application/json' },
                              body: JSON.stringify({ resolved_by: localStorage.getItem('username') || 'coe', status: 'resolved' }),
                            })
                          } catch (_) {}
                          setAppealDetailModal(null)
                          setTimeout(() => openExtensionModal(wid), 50)
                        }
                      }}
                      className="px-5 py-2 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition-colors flex items-center gap-2"
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        {isActiveWindowAppeal ? (
                          <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        ) : (
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                        )}
                      </svg>
                      {isActiveWindowAppeal ? 'Approve & Allow Re-entry' : isSubmittedExpiredAppeal ? 'Approve & Extend Window for Amendment' : 'Extend Window for this Teacher'}
                    </button>
                  </div>
                </div>
              </div>
              )
            })()}
          </div>
        )}
      </div>
    </MainLayout>
  )
}

export default MarkEntryPermissionsPage
