import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const TeacherCourseSelectionPage = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [teacherId, setTeacherId] = useState(null);
  const [allocationSummary, setAllocationSummary] = useState(null);
  const [availableCourses, setAvailableCourses] = useState([]);
  const [selectedCourses, setSelectedCourses] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSelectedPanel, setShowSelectedPanel] = useState(true); // Show by default
  const [activeTab, setActiveTab] = useState('theory');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [currentPreferences, setCurrentPreferences] = useState([]);
  const [preferencesLocked, setPreferencesLocked] = useState(false); // Lock after save

  // Initialize from LocalStorage
  useEffect(() => {
    // Only load if we have a teacherId
    const storedTeacherId = localStorage.getItem('teacher_id') || localStorage.getItem('userId');
    if (storedTeacherId) {
      const draft = localStorage.getItem(`course_draft_${storedTeacherId}`);
      if (draft) {
        try {
          const parsed = JSON.parse(draft);
          if (Array.isArray(parsed) && parsed.length > 0) {
            console.log("Loaded draft from localStorage:", parsed);
            setSelectedCourses(parsed);
          }
        } catch (e) {
          console.error("Failed to parse draft from localStorage", e);
        }
      }
    }
  }, [teacherId]); // Run once when teacherId is set (or changes, technically)

  // Save to LocalStorage whenever selection changes
  useEffect(() => {
    if (teacherId && selectedCourses.length > 0) {
      localStorage.setItem(`course_draft_${teacherId}`, JSON.stringify(selectedCourses));
    } else if (teacherId && selectedCourses.length === 0) {
      // Don't clear if it's just initial empty state, handled by logic above
      // But if user manually clears everything, we should probably clear storage?
      // Let's rely on manual clear actions if needed, but for now updates sync.
      // If empty array, we can update it to empty array.
       localStorage.setItem(`course_draft_${teacherId}`, JSON.stringify(selectedCourses));
    }
  }, [selectedCourses, teacherId]);
  
  // Auto-fetched from academic calendar
  const [academicYear, setAcademicYear] = useState('');
  const [currentSemester, setCurrentSemester] = useState(1);
  const [nextSemester, setNextSemester] = useState(2); // Next semester for preference entry
  const [nextSemesterType, setNextSemesterType] = useState('even'); // Type of next semester
  const [availableBatches, setAvailableBatches] = useState([]);
  
  // Window status
  const [windowStatus, setWindowStatus] = useState({
    isOpen: false,
    startDate: null,
    endDate: null,
    message: ''
  });
  
  // User selections
  const [selectedBatch, setSelectedBatch] = useState('');
  const [selectedSemester, setSelectedSemester] = useState(1);
  const [currentSemesterType, setCurrentSemesterType] = useState('odd'); // 'odd' or 'even'

  // Fetch academic calendar configuration on load
  useEffect(() => {
    fetchAcademicCalendar();
  }, []);

  const fetchAcademicCalendar = async () => {
    try {
      // Fetch current academic calendar for semester info
      const response = await fetch(`${API_BASE_URL}/academic-calendar/current`);
      if (response.ok) {
        const data = await response.json();
        setAcademicYear(data.academic_year);
        setCurrentSemester(data.current_semester);
        
        // Fetch window dates from teacher_course_tracking table
        const windowResponse = await fetch(`${API_BASE_URL}/teachers/course-window/${data.academic_year}`);
        if (windowResponse.ok) {
          const windowData = await windowResponse.json();
          
          // Check window status
          const now = new Date();
          const startDate = windowData.window_start ? new Date(windowData.window_start + 'T00:00:00') : null;
          const endDate = windowData.window_end ? new Date(windowData.window_end + 'T23:59:59') : null;
          
          let windowOpen = false;
          let message = '';
          
          if (!windowData.configured || !startDate || !endDate) {
            message = '⚠️ Teacher course selection window is not configured';
          } else if (now < startDate) {
            message = `⏳ Selection window opens on ${startDate.toLocaleDateString()}`;
          } else if (now > endDate) {
            message = `🔒 Selection window closed on ${endDate.toLocaleDateString()}`;
          } else {
            windowOpen = true;
            message = `✅ Selection window is open until ${endDate.toLocaleDateString()}`;
          }
          
          setWindowStatus({
            isOpen: windowOpen,
            startDate: startDate,
            endDate: endDate,
            message: message
          });
        } else {
          setWindowStatus({
            isOpen: false,
            startDate: null,
            endDate: null,
            message: '⚠️ Teacher course selection window is not configured'
          });
        }
        
        // Calculate next semester (wrap around: 8 -> 1)
        const next = data.current_semester === 8 ? 1 : data.current_semester + 1;
        setNextSemester(next);
        
        // Determine current and next semester types
        const semType = data.current_semester % 2 === 0 ? 'even' : 'odd';
        const nextSemType = next % 2 === 0 ? 'even' : 'odd';
        setCurrentSemesterType(semType);
        setNextSemesterType(nextSemType);
        setSelectedSemester(data.current_semester);
        
        // Calculate available batches based on current semester
        // For semester 1: batch is current year
        // For semester 2: batch is current year
        // For semester 3: batch is current year - 1, etc.
        const currentYear = new Date().getFullYear();
        const yearOffset = Math.floor((data.current_semester - 1) / 2);
        const batches = [];
        
        // Generate batches for all 8 semesters
        for (let i = 0; i < 4; i++) {
          batches.push((currentYear - i).toString());
        }
        
        setAvailableBatches(batches);
        setSelectedBatch(batches[yearOffset] || batches[0]);
      }
    } catch (err) {
      console.error('Failed to fetch academic calendar:', err);
      // Fallback values
      setAcademicYear('2024-2025');
      setCurrentSemester(2);
      setNextSemester(3);
      setCurrentSemesterType('even');
      setNextSemesterType('odd');
      setSelectedSemester(2);
      setAvailableBatches(['2024', '2023', '2022', '2021']);
      setSelectedBatch('2024');
      setWindowStatus({
        isOpen: false,
        startDate: null,
        endDate: null,
        message: '⚠️ Unable to check selection window status'
      });
    }
  };

  useEffect(() => {
    // Get teacher data by email (preferred) or fallback to stored teacher_id
    const fetchTeacherByEmail = async () => {
      const userEmail = localStorage.getItem('userEmail') || localStorage.getItem('teacher_email');
      
      if (!userEmail) {
        // Fallback to stored teacher_id
        const storedTeacherId = localStorage.getItem('teacher_id');
        if (storedTeacherId) {
          console.log('Using stored teacher ID:', storedTeacherId);
          setTeacherId(storedTeacherId);
        } else {
          setError('No teacher email or ID found. Please log in as a teacher.');
          setLoading(false);
        }
        return;
      }
      
      try {
        // Fetch teacher data from backend using email
        const response = await fetch(`${API_BASE_URL}/teachers/by-email?email=${encodeURIComponent(userEmail)}`);
        
        if (response.ok) {
          const data = await response.json();
          if (data.teacher && data.teacher.id) {
            console.log('Teacher found by email:', data.teacher);
            setTeacherId(data.teacher.id.toString());
            
            // Update localStorage with fresh teacher data
            localStorage.setItem('teacher_id', data.teacher.id);
            localStorage.setItem('faculty_id', data.teacher.faculty_id || '');
            localStorage.setItem('teacher_name', data.teacher.name || '');
            localStorage.setItem('teacher_dept', data.teacher.dept || '');
          } else {
            setError(`No teacher record found for email: ${userEmail}\nPlease contact your administrator to add you to the teachers database.`);
            setLoading(false);
          }
        } else if (response.status === 404) {
          setError(`No teacher record found for email: ${userEmail}\nPlease contact your administrator to add you to the teachers database.`);
          setLoading(false);
        } else {
          // Fallback to stored teacher_id if API fails
          const storedTeacherId = localStorage.getItem('teacher_id');
          if (storedTeacherId) {
            console.log('API failed, using stored teacher ID:', storedTeacherId);
            setTeacherId(storedTeacherId);
          } else {
            setError('Unable to fetch teacher data. Please try logging in again.');
            setLoading(false);
          }
        }
      } catch (err) {
        console.error('Error fetching teacher by email:', err);
        // Fallback to stored teacher_id
        const storedTeacherId = localStorage.getItem('teacher_id');
        if (storedTeacherId) {
          console.log('Error occurred, using stored teacher ID:', storedTeacherId);
          setTeacherId(storedTeacherId);
        } else {
          setError('Unable to fetch teacher data. Please try logging in again.');
          setLoading(false);
        }
      }
    };
    
    fetchTeacherByEmail();
  }, []);

  useEffect(() => {
    if (teacherId && academicYear) {
      fetchAllocationSummary(teacherId);
    }
  }, [teacherId, academicYear]);

  // Separate effect for fetching preferences - depends on next semester and window status
  useEffect(() => {
    if (teacherId && nextSemester && windowStatus.endDate !== null) {
      console.log('Fetching preferences for next semester:', { teacherId, nextSemester, windowOpen: windowStatus.isOpen });
      fetchCurrentPreferences(teacherId);
    }
  }, [teacherId, nextSemester, windowStatus.isOpen]);

  useEffect(() => {
    if (teacherId && nextSemesterType) {
      // Fetch all courses for all semesters of next semester type (odd or even)
      fetchAvailableCourses();
    }
  }, [teacherId, nextSemesterType]);

  const fetchAllocationSummary = async (tid) => {
    try {
      const response = await fetch(`${API_BASE_URL}/teachers/${tid}/allocation-summary`);
      if (response.ok) {
        const data = await response.json();
        console.log('Allocation Summary received:', data.summary);
        console.log('Type summaries:', data.summary?.type_summaries);
        if (data.summary) {
          setAllocationSummary(data.summary);
          // Set active tab to first available type if not set
          if (data.summary.type_summaries?.length > 0 && activeTab === 'theory') {
            setActiveTab(data.summary.type_summaries[0].type_name);
          }
        }
      } else if (response.status === 404) {
        setError(`Teacher ID ${tid} not found in the system. Please contact admin.`);
        setLoading(false);
      } else {
        console.warn('Allocation summary not available, continuing without it');
      }
    } catch (err) {
      console.error('Allocation summary error:', err);
    }
  };

  const fetchCurrentPreferences = async (tid) => {
    try {
      // Use the calculated nextSemester and its batch
      const nextBatch = getBatchForSemester(nextSemester);
      
      // Check for preferences with academic_year (backend expects this parameter)
      const url = `${API_BASE_URL}/teachers/${tid}/course-preferences?academic_year=${academicYear}`;
      console.log('Checking preferences for academic year:', academicYear, 'next semester:', nextSemester, 'batch:', nextBatch);
      console.log('Fetching from URL:', url);
      
      const response = await fetch(url);
      const data = await response.json();
      console.log('Fetched preferences:', data);
      console.log('Preferences array:', data.preferences);
      console.log('Preferences length:', data.preferences?.length);
      
      setCurrentPreferences(data.preferences || []);
      
      // Load preferences into the form if they exist
      if (data.preferences && data.preferences.length > 0) {
        console.log('Loading saved preferences for next semester:', nextSemester, 'batch:', nextBatch);
        
        // Pre-populate courses from saved preferences
        const preSelected = data.preferences.map(p => ({
          course_id: p.course_id,
          course_type: p.course_type,
          course_type_id: p.course_type_id,
          semester: p.semester,
          batch: p.batch,
          priority: p.priority,
          course_name: p.course_name || availableCourses.find(av => av.course_code === p.course_id)?.course_name
        }));
        setSelectedCourses(preSelected);
        
        // Lock form ONLY if window is closed or preferences are already locked
        // During window period, allow editing only if not locked
        if (!windowStatus.isOpen) {
          setPreferencesLocked(true);
          setSuccess(`Your course preferences were submitted. Window is now closed.`);
        } else if (data.locked) {
          setPreferencesLocked(true);
          setSuccess(`Your course preferences have already been submitted for this window.`);
        } else {
          setPreferencesLocked(false);
          setSuccess(`Your previously saved preferences are loaded. You can update them until ${windowStatus.endDate?.toLocaleDateString()}.`);
        }
      } else {
        if (data.locked) {
          console.log('Preferences locked but empty?');
          setPreferencesLocked(true);
          setSuccess(`Your course preferences have already been submitted for this window.`);
        } else {
          console.log('NO preferences found for next semester - form remains UNLOCKED');
          setPreferencesLocked(false);
        }
      }
      
      setLoading(false);
    } catch (err) {
      setError('Failed to load current preferences');
      setLoading(false);
      console.error(err);
    }
  };

  const fetchAvailableCourses = async () => {
    if (!teacherId || !nextSemesterType) return;
    
    try {
      setLoading(true);
      // Determine which semesters to fetch based on NEXT semester type
      const semesters = nextSemesterType === 'odd' ? [1, 3, 5, 7] : [2, 4, 6, 8];
      
      let allCourses = [];
      
      // Fetch courses for each semester in parallel
      const promises = semesters.map(semester => 
        fetch(`${API_BASE_URL}/teachers/${teacherId}/semester/${semester}/courses`)
          .then(res => {
            if (res.status === 404) {
              console.warn(`Teacher ${teacherId} not found or no curriculum mapping`);
              return { courses: [] };
            }
            if (res.status === 500) {
              console.error(`Server error fetching courses for semester ${semester}`);
              return { courses: [] };
            }
            return res.ok ? res.json() : { courses: [] };
          })
          .then(data => ({
            semester,
            courses: Array.isArray(data.courses) ? data.courses.map(c => ({...c, semester})) : []
          }))
          .catch(err => {
            console.error(`Error fetching semester ${semester}:`, err);
            return { semester, courses: [] };
          })
      );
      
      const results = await Promise.all(promises);
      
      // Combine all courses and filter out electives
      results.forEach(result => {
        allCourses = [...allCourses, ...result.courses];
      });
      
      // Check if all results are empty
      if (allCourses.length === 0) {
        setError(`No courses found for teacher ID ${teacherId}. This may be because:\n• Teacher record doesn't exist in the database\n• Teacher's department is not mapped to a curriculum\n• No courses are configured for the upcoming ${nextSemesterType} semesters\n\nPlease contact your department administrator.`);
      }
      
      // Filter out elective courses:
      // - Open Elective
      // - Professional Elective I, II, III, IV, V, VI, VII, VIII (roman numerals)
      // - Professional Elective 1, 2, 3, 4, 5, 6, 7, 8 (numbers)
      const electivePattern = /\b(open\s+elective|professional\s+elective\s+([ivxIVX]+|[1-8]))\b/i;
      
      allCourses = allCourses.filter(course => {
        const courseName = course.course_name || '';
        const isElective = electivePattern.test(courseName);
        if (isElective) {
          console.log('Filtering out elective course:', courseName);
        }
        return !isElective;
      });
      
      console.log(`All ${nextSemesterType} semester courses loaded (excluding electives):`, allCourses);
      setAvailableCourses(allCourses);
      setLoading(false);
    } catch (err) {
      setError('Failed to load courses');
      setAvailableCourses([]);
      setLoading(false);
      console.error(err);
    }
  };

  // Calculate which batch a course belongs to based on semester number
  // e.g., AY 2025-2026: Sem 1-2 = Batch 2025-2029, Sem 3-4 = Batch 2024-2028, Sem 5-6 = Batch 2023-2027, Sem 7-8 = Batch 2022-2026
  const getBatchForSemester = (semester) => {
    if (!academicYear || academicYear === '') {
      console.warn('Academic year not available for batch calculation');
      return '';
    }
    const startYear = parseInt(academicYear.split('-')[0]); // e.g., 2025 from "2025-2026"
    if (isNaN(startYear)) {
      console.error('Invalid academic year format:', academicYear);
      return '';
    }
    const yearOffset = Math.floor((semester - 1) / 2); // sem 1,2 = 0, sem 3,4 = 1, sem 5,6 = 2, sem 7,8 = 3
    const batchStartYear = startYear - yearOffset;
    const batchEndYear = batchStartYear + 4; // 4-year degree program
    const batch = `${batchStartYear}-${batchEndYear}`;
    console.log(`Batch calculation: semester=${semester}, academicYear=${academicYear}, startYear=${startYear}, yearOffset=${yearOffset}, batch=${batch}`);
    return batch;
  };

  const handleCourseToggle = (course) => {
    // Don't allow changes if preferences are locked (window closed or already submitted)
    if (preferencesLocked) {
      setError('Preferences cannot be changed. Either the window has closed or you have already submitted your preferences.');
      setTimeout(() => setError(''), 3000);
      return;
    }
    
    // USE course.course_code as unique identifier for selection logic
    const isSelected = selectedCourses.some(c => c.course_id === course.course_code);
    
    if (isSelected) {
      setSelectedCourses(selectedCourses.filter(c => c.course_id !== course.course_code));
    } else {
      // Find the limit for this course type
      const typeSummary = allocationSummary?.type_summaries?.find(s => s.type_name === course.course_type);
      console.log('Course being added:', course);
      console.log('Looking for type_name:', course.course_type);
      console.log('Available type_summaries:', allocationSummary?.type_summaries);
      console.log('Found typeSummary:', typeSummary);
      
      const maxAllowed = typeSummary ? typeSummary.allocated : 0;
      const currentCount = selectedCourses.filter(c => c.course_type === course.course_type).length;
      
      if (currentCount >= maxAllowed) {
        setError(`You can only select ${maxAllowed} ${course.course_type} courses. Remove a course first.`);
        setTimeout(() => setError(''), 3000);
        return;
      }
      
      setSelectedCourses([...selectedCourses, {
        course_id: course.course_code, // Store code in course_id for backend
        course_type: course.course_type, // String name for display
        course_type_id: typeSummary?.course_type_id, // Integer ID for backend
        semester: course.semester,
        batch: getBatchForSemester(course.semester), // Calculate batch from semester
        priority: selectedCourses.length + 1,
        course_name: course.course_name // Store name for UI display
      }]);
    }
  };

  const handleSavePreferences = async () => {
    // Check if window is open
    if (!windowStatus.isOpen) {
      setError(windowStatus.message || 'Teacher course selection window is not currently open');
      return;
    }
    
    // preferencesLocked will be true if window is closed or already submitted
    if (preferencesLocked) {
      setError('Preferences cannot be changed. Either the window has closed or you have already submitted your preferences.');
      return;
    }
    
    if (selectedCourses.length === 0) {
      setError('Please select at least one course');
      return;
    }

    // Validate that all allocated course counts are met
    const validationErrors = [];
    allocationSummary?.type_summaries?.forEach((ts) => {
      const selectedCount = getSelectedCount(ts.type_name);
      if (selectedCount !== ts.allocated) {
        validationErrors.push(
          `${ts.type_name}: selected ${selectedCount} but required ${ts.allocated}`
        );
      }
    });

    if (validationErrors.length > 0) {
      setError(`Please select exactly the allocated courses:\n${validationErrors.join('\n')}`);
      return;
    }

    setSaving(true);
    setError('');
    setSuccess('');

    try {
      // Map selectedCourses to backend format with course_type as integer ID
      const preferencesForBackend = selectedCourses.map(course => {
        const batch = course.batch || getBatchForSemester(course.semester);
        console.log(`Course ${course.course_id}: semester=${course.semester}, batch=${batch}`);
        return {
          course_id: course.course_id,
          course_type: course.course_type_id, // Send integer ID, not string name
          semester: course.semester,
          batch: batch, // Batch from semester
          priority: course.priority
        };
      });
      
      console.log('Preferences being sent:', preferencesForBackend);
      console.log('Academic Year:', academicYear);
      
      const payload = {
        teacher_id: parseInt(teacherId),
        academic_year: academicYear,
        current_semester_type: currentSemesterType,
        preferences: preferencesForBackend
      };
      
      console.log('Full payload:', payload);

      const response = await fetch(`${API_BASE_URL}/teachers/course-preferences`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      });

      let data;
      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        // Backend returned plain text error
        const text = await response.text();
        console.error('Backend error (plain text):', text);
        data = { error: text };
      }

      if (response.ok) {
        setSuccess('Course preferences saved successfully! ✓');
        setPreferencesLocked(true);
        // Clear localStorage draft after successful save
        localStorage.removeItem(`course_draft_${teacherId}`);
        // Don't fetch preferences again - we just saved them and locked the form
        // fetchCurrentPreferences(teacherId);
        setTimeout(() => setSuccess(''), 3000);
      } else {
        setError(data.error || 'Failed to save preferences');
      }
    } catch (err) {
      setError('Network error. Please try again.');
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  // Filter courses by search query and active tab, then group by semester
  const getFilteredCourses = () => {
    if (!Array.isArray(availableCourses)) return [];
    
    // Filter by Active Tab first
    let courses = availableCourses.filter(course => course.course_type === activeTab);

    // Then filter by search query if present
    if (searchQuery) {
      const lowerQuery = searchQuery.toLowerCase();
      courses = courses.filter(course => 
        course.course_name.toLowerCase().includes(lowerQuery) || 
        course.course_code.toLowerCase().includes(lowerQuery)
      );
    }
    
    return courses;
  };

  // Group filtered courses by semester
  const getCoursesGroupedBySemester = () => {
    const filtered = getFilteredCourses();
    const grouped = {};
    
    filtered.forEach(course => {
      const sem = course.semester || selectedSemester;
      if (!grouped[sem]) {
        grouped[sem] = [];
      }
      grouped[sem].push(course);
    });
    
    return grouped;
  };

  const getSelectedCount = (type) => {
    if (!Array.isArray(selectedCourses)) return 0;
    return selectedCourses.filter(c => c.course_type === type).length;
  };

  // Check if all allocation counts are filled
  const isAllocationComplete = () => {
    if (!allocationSummary?.type_summaries || allocationSummary.type_summaries.length === 0) {
      return false;
    }
    
    return allocationSummary.type_summaries.every(ts => {
      const selectedCount = getSelectedCount(ts.type_name);
      return selectedCount === ts.allocated;
    });
  };

  // Get all available semesters (both odd and even)
  const getAvailableSemesters = () => {
    return [1, 2, 3, 4, 5, 6, 7, 8]; // All semesters
  };

  if (loading) {
    return (
      <MainLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout title="Teacher Course Selection">
      {error && (
        <div className="fixed top-4 left-1/2 transform -translate-x-1/2 z-50 w-[calc(100%-2rem)] max-w-4xl">
          <div className="bg-red-50 border border-red-200 p-4 rounded-lg shadow-md flex items-start justify-between">
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-red-600" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-red-800 whitespace-pre-line">{error}</p>
              </div>
            </div>
            <div className="ml-4 flex items-start">
              <button
                onClick={() => setError('')}
                className="text-sm font-medium text-red-600 hover:text-red-800 ml-4"
                aria-label="Dismiss error"
              >
                Dismiss
              </button>
            </div>
          </div>
        </div>
      )}
      <div className="min-h-screen card-custom bg-gray-50 py-8 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          {/* Header */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <div className="flex justify-between items-start">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-3xl font-bold text-gray-900">
                  Course Selection
                </h1>
                <span className={`px-3 py-1 text-sm font-semibold rounded-full ${
                  nextSemesterType === 'odd' 
                    ? 'bg-purple-100 text-primary' 
                    : 'bg-green-100 text-green-800'
                }`}>
                  {nextSemesterType === 'odd' ? '🟣 Odd Semester' : '🟢 Even Semester'}
                  <span className="ml-2 text-xs opacity-90">(Semester {nextSemester})</span>
                </span>
              </div>
              <p className="text-gray-600">
                Select your preferred courses for the next semester ({nextSemesterType} - Semester {nextSemester})
              </p>
            </div>
            <button
              onClick={() => navigate(-1)}
              className="px-4 py-2 text-gray-600 hover:text-gray-900 font-medium"
            >
              ← Back
            </button>
          </div>
        </div>

        {/* Locked State Warning */}
        {preferencesLocked && (
          <div className="bg-gray-50 border border-gray-200 p-4 mb-6 rounded-lg">
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-gray-900">Preferences Submitted</h3>
                <p className="mt-1 text-sm text-gray-600">
                  Your course preferences for Semester {nextSemester} (Batch {getBatchForSemester(nextSemester)}) have been submitted.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Alerts */}
        {error && (
          <div className="bg-red-50 border border-red-200 p-4 mb-6 rounded-lg">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-red-600" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <p className="text-sm text-red-800 whitespace-pre-line">{error}</p>
                {error.includes('Teacher ID') && teacherId && (
                  <div className="mt-3 text-xs text-gray-600 bg-white p-3 rounded border border-gray-200">
                    <p className="font-semibold mb-1">Debug Info:</p>
                    <p>Teacher ID from localStorage: {teacherId}</p>
                    <p>User ID: {localStorage.getItem('userId') || 'Not set'}</p>
                    <p>User Role: {localStorage.getItem('userRole') || 'Not set'}</p>
                    <p>Faculty ID: {localStorage.getItem('faculty_id') || 'Not set'}</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {success && (
          <div className="bg-gray-50 border border-gray-200 p-4 mb-6 rounded-lg">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-gray-600" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-gray-800">{success}</p>
              </div>
            </div>
          </div>
        )}

        {/* Allocation Summary Card */}
        {allocationSummary && (
          <div className="bg-white border border-gray-200 rounded-lg shadow-sm p-6 mb-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Allocation Summary</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4">
              {allocationSummary.type_summaries?.map((ts) => {
                const selectedCount = getSelectedCount(ts.type_name);
                const isComplete = selectedCount === ts.allocated;
                const isOver = selectedCount > ts.allocated;
                
                return (
                  <div key={ts.course_type_id} className={`border rounded-lg p-4 ${
                    isComplete ? 'border-gray-900 bg-gray-50' :
                    isOver ? 'border-red-300 bg-red-50' :
                    'border-gray-200 bg-white'
                  }`}>
                    <div className="flex items-center justify-between mb-2">
                      <div className="text-xs font-medium text-primary uppercase tracking-wide">{ts.type_name.replace('_', ' ')}</div>
                      {isComplete && <span className="text-primary">✓</span>}
                      {isOver && <span className="text-red-600">⚠</span>}
                    </div>
                    <div className="text-2xl font-bold text-primary">
                      {selectedCount} <span className="text-primary">/ {ts.allocated}</span>
                    </div>
                    <div className="text-xs mt-1 text-gray-500">
                      {isComplete ? 'Complete' : 
                       isOver ? `${selectedCount - ts.allocated} over` :
                       `${ts.allocated - selectedCount} remaining`}
                    </div>
                  </div>
                );
              })}
              
              <div className={`border rounded-lg p-4 ${
                isAllocationComplete() ? 'border-gray-900 bg-gray-50' : 'border-gray-200 bg-white'
              }`}>
                <div className="text-xs font-medium text-primary uppercase tracking-wide mb-2">Status</div>
                <div className="text-lg font-semibold text-primary mt-2">
                  {isAllocationComplete() ? '✓ Complete' : 'Incomplete'}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Academic Year Display */}
        <div className="bg-white border border-gray-200 rounded-lg shadow-sm p-4 mb-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-gray-500 mb-1">Academic Year</p>
              <p className="text-base font-semibold text-gray-900">{academicYear || 'Loading...'}</p>
            </div>
            <div>
              <p className="text-xs text-gray-500 mb-1">Semester Type</p>
              <span className="px-3 py-1 text-sm font-medium rounded-md bg-gray-100 text-gray-900">
                {nextSemesterType === 'odd' ? 'Odd (1,3,5,7)' : 'Even (2,4,6,8)'}
              </span>
            </div>
          </div>
          
          {/* Selection Window Status */}
          <div className="mt-3 pt-3 border-t border-gray-200">
            <div className={`p-3 rounded-md ${
              windowStatus.isOpen 
                ? 'bg-green-50 border border-green-200' 
                : 'bg-yellow-50 border border-yellow-200'
            }`}>
              <p className={`text-sm font-medium ${
                windowStatus.isOpen ? 'text-green-800' : 'text-yellow-800'
              }`}>
                {windowStatus.message}
              </p>
            </div>
          </div>
          
          <div className="mt-3 pt-3 border-t border-gray-200">
            <p className="text-xs text-gray-600">
              Showing all {nextSemesterType} semester courses grouped by semester number
            </p>
          </div>
        </div>

        {/* Search Filter */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <h3 className="text-lg font-semibold text-gray-900 whitespace-nowrap">
              Available Courses
            </h3>
            <div className="w-full md:w-1/2 relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
              <input
                type="text"
                placeholder="Search by Course Code or Name..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg leading-5 bg-white placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary sm:text-sm"
              />
            </div>
            <button
              onClick={() => setShowSelectedPanel(!showSelectedPanel)}
              className="flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg transition-colors"
            >
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
              </svg>
              <span>View Selected ({selectedCourses.length})</span>
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Course List (Left Side - 2/3 width) */}
          <div className="lg:col-span-2 transition-all duration-300 flex flex-col h-[calc(100vh-200px)] sticky top-6 bg-white border border-gray-200 rounded-lg shadow-sm">
            
            {/* Tab Headers */}
            <div className="bg-white rounded-t-lg shadow-sm overflow-hidden border border-gray-200 mb-0">
              <nav className="flex flex-wrap sticky top-0 bg-white z-10">
                {allocationSummary?.type_summaries?.map((ts) => (
                  <button
                    key={ts.course_type_id}
                    onClick={() => setActiveTab(ts.type_name)}
                    className={`flex-1 min-w-[120px] py-3 px-4 text-center font-medium text-sm capitalize transition-all ${
                      activeTab === ts.type_name
                        ? 'border-b-2 border-primary text-primary bg-gray-50'
                        : 'border-b-2 border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    {ts.type_name.replace('_', ' ')}
                    <span className={`ml-2 px-2 py-0.5 text-xs rounded-md ${
                      activeTab === ts.type_name ? 'bg-primary text-white' : 'bg-gray-200 text-gray-600'
                    }`}>
                      {getSelectedCount(ts.type_name)} / {ts.allocated}
                    </span>
                  </button>
                ))}
              </nav>
            </div>

            <div className="bg-white rounded-b-lg shadow-sm overflow-auto border-l border-r border-b border-gray-200 flex-1">
              <div className="px-4 py-3 border-b border-gray-200 flex justify-between items-center">
                <span className="font-medium text-primary capitalize text-sm">{activeTab.replace('_', ' ')} Courses</span>
                <span className="text-xs text-gray-500 px-2 py-1 bg-gray-100 rounded-md">{getFilteredCourses().length} courses</span>
              </div>
              
              <div className="p-4">
                {getFilteredCourses().length === 0 ? (
                  <div className="text-center py-12 text-gray-500">
                    <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    <p className="mt-4 text-sm">No {activeTab.replace('_', ' ')} courses found</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {/* Group courses by semester */}
                    {Object.entries(getCoursesGroupedBySemester()).map(([semester, courses]) => (
                      <div key={semester} className="border-l-2 border-background pl-4">
                        <h3 className="text-base font-semibold text-primary mb-3 flex items-center gap-2">
                          Semester {semester}
                          <span className="text-xs font-normal text-gray-500 px-2 py-0.5 bg-gray-100 rounded-md">
                            {courses.length}
                          </span>
                        </h3>
                        
                        <div className="space-y-2">
                          {courses.map((course) => {
                            const isSelected = selectedCourses.some(c => c.course_id === course.course_code);
                  
                  return (
                    <div
                      key={course.id}
                      onClick={() => !preferencesLocked && handleCourseToggle(course)}
                      className={`border rounded-lg p-3 transition-all ${
                        preferencesLocked 
                          ? 'cursor-not-allowed opacity-50 bg-gray-50' 
                          : 'cursor-pointer hover:border-gray-400'
                      } ${
                        isSelected
                          ? 'border-primary bg-gray-50'
                          : 'border-gray-200 bg-white'
                      }`}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex items-start space-x-3 flex-1">
                          <div className="flex-shrink-0 mt-0.5">
                            <div className={`w-4 h-4 rounded border flex items-center justify-center ${
                              isSelected ? 'bg-primary border-primary' : 'border-gray-300'
                            }`}>
                              {isSelected && (
                                <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                                </svg>
                              )}
                            </div>
                          </div>
                          
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center space-x-2 mb-1">
                              <h4 className="font-medium text-gray-900 text-sm">{course.course_code}</h4>
                              <span className="px-2 py-0.5 text-[10px] rounded bg-gray-100 text-gray-600 font-medium">
                                {course.credits} Credits
                              </span>
                            </div>
                            <p className="text-gray-600 text-sm leading-snug">{course.course_name}</p>
                            <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
                              <span>Sem {course.semester}</span>
                              <span>•</span>
                              <span>Batch {getBatchForSemester(course.semester)}</span>
                            </div>
                          </div>
                        </div>
                        
                        <div className="flex-shrink-0 ml-2">
                          <span className={`px-2 py-1 text-xs font-medium rounded ${
                            isSelected ? 'bg-primary text-white' : 'bg-gray-100 text-gray-600'
                          }`}>
                            {isSelected ? '✓' : ''}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>

      {/* Right Side Panel - Selected Courses */}
      <div className="lg:col-span-1 bg-white border border-gray-200 rounded-lg shadow-sm flex flex-col justify-center h-[calc(100vh-200px)] sticky top-6">
        <div className="p-4 border-b border-gray-200 flex justify-between items-center rounded-t-lg">
           <h3 className="font-semibold text-gray-900 text-sm">Selected ({selectedCourses.length})</h3>
           {/* Show allocation summary in panel header */}
           <div className="text-[10px] text-gray-600 font-medium">
             {allocationSummary?.type_summaries?.map((ts) => (
               <span key={ts.type_name} className="ml-2">
                 {ts.type_name.replace('_', ' ')}: {getSelectedCount(ts.type_name)}/{ts.allocated}
               </span>
             ))}
           </div>
        </div>
        
        <div className="flex-1 overflow-y-auto p-3 space-y-2 bg-gray-50">
           {selectedCourses.length === 0 ? (
             <div className="text-center text-gray-400 py-12">
               <svg className="mx-auto h-10 w-10 text-gray-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                 <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
               </svg>
               <p className="text-sm">No courses selected</p>
               <p className="text-xs mt-1">Click courses to add</p>
             </div>
           ) : (
             selectedCourses.map((course, idx) => (
                <div key={idx} className="bg-white border border-gray-200 rounded-lg p-3 relative group hover:border-gray-300 transition-colors">
                  <div className="pr-6">
                    <div className="font-medium text-sm text-gray-900">{course.course_id}</div>
                    <div className="text-xs text-gray-600 mt-1 line-clamp-2">{course.course_name || 'Loading...'}</div>
                    <div className="mt-2">
                       <span className="text-[10px] bg-gray-100 text-gray-700 px-2 py-0.5 rounded font-medium">{course.course_type?.replace('_', ' ')}</span>
                    </div>
                  </div>
                  {!preferencesLocked && (
                    <button 
                      onClick={() => handleCourseToggle({course_code: course.course_id, course_type: course.course_type})}
                      className="absolute top-2 right-2 text-gray-400 hover:text-gray-900 p-1 opacity-0 group-hover:opacity-100 transition-all"
                      title="Remove"
                    >
                      ✕
                    </button>
                  )}
                </div>
             ))
           )}
        </div>
        
        <div className="p-4 border-t border-gray-200 bg-white rounded-b-lg">
          <button
              onClick={handleSavePreferences}
              disabled={saving || !isAllocationComplete() || preferencesLocked || !windowStatus.isOpen}
              className={`w-full py-2.5 rounded-lg font-medium transition-colors text-sm ${
                saving || !isAllocationComplete() || preferencesLocked || !windowStatus.isOpen
                  ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                  : 'bg-gray-900 text-white hover:bg-gray-800'
              }`}
            >
              {!windowStatus.isOpen 
                ? '🔒 Window Closed' 
                : (preferencesLocked 
                    ? '✓ Submitted' 
                    : (saving 
                        ? 'Saving...' 
                        : (isAllocationComplete() 
                            ? (currentPreferences.length > 0 ? '🔄 Update Preferences' : '💾 Save Preferences')
                            : 'Complete Allocations'
                          )
                      )
                  )
              }
            </button>
        </div>
      </div>
    </div>

        {!showSelectedPanel && (
        <div className="mt-6 flex justify-end space-x-4">
          <button
            onClick={() => setSelectedCourses([])}
            disabled={preferencesLocked || !windowStatus.isOpen}
            className={`px-6 py-3 border border-gray-200 text-gray-700 font-medium rounded-lg transition-colors ${
              preferencesLocked || !windowStatus.isOpen
                ? 'opacity-50 cursor-not-allowed'
                : 'hover:bg-gray-50'
            }`}
          >
            Clear All
          </button>
          
          <button
            onClick={handleSavePreferences}
            disabled={saving || !isAllocationComplete() || preferencesLocked || !windowStatus.isOpen}
            className={`px-8 py-3 rounded-lg font-medium transition-colors ${
              saving || !isAllocationComplete() || preferencesLocked || !windowStatus.isOpen
                ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
                : 'bg-blue-600 text-white hover:bg-blue-700'
            }`}
          >
            {saving ? (
              <span className="flex items-center">
                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Saving...
              </span>
            ) : (
              !windowStatus.isOpen 
                ? '🔒 Window Closed' 
                : (preferencesLocked 
                    ? '✓ Submitted' 
                    : (isAllocationComplete() 
                        ? `Save Preferences (${selectedCourses.length})`
                        : 'Complete Allocations'
                      )
                  )
            )}
          </button>
        </div>
        )}

        {/* Previously Selected Courses */}
        {currentPreferences.length > 0 && (
          <div className="mt-8 bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold mb-4 text-gray-900">
              Previous Selections History
            </h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Course ID
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Type
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Semester
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Batch
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Selected On
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {currentPreferences.map((pref, index) => (
                    <tr key={index}>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {pref.course_id}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        <span className="px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-800">
                          {pref.course_type.replace('_', ' ')}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {pref.semester}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {pref.batch}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span className={`px-2 py-1 text-xs rounded-full ${
                          pref.status === 'approved' ? 'bg-green-100 text-green-800' :
                          pref.status === 'pending' ? 'bg-yellow-100 text-yellow-800' :
                          pref.status === 'active' ? 'bg-blue-100 text-blue-800' :
                          'bg-gray-100 text-gray-800'
                        }`}>
                          {pref.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {new Date(pref.created_at).toLocaleDateString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
        </div>
      </div>
    </MainLayout>
  );
};

export default TeacherCourseSelectionPage;
