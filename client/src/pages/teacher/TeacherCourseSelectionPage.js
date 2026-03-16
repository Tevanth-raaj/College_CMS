import React, { useState, useEffect } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const TeacherCourseSelectionPage = () => {
  // State
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [teacherId, setTeacherId] = useState(null);
  const [allocationSummary, setAllocationSummary] = useState(null);
  const [availableCourses, setAvailableCourses] = useState([]);
  const [selectedCourses, setSelectedCourses] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeTab, setActiveTab] = useState('theory');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [preferencesLocked, setPreferencesLocked] = useState(false);
  const [lockCheckDone, setLockCheckDone] = useState(false);
  
  // Academic information
  const [academicYear, setAcademicYear] = useState('');
  const [nextSemester, setNextSemester] = useState(1);
  const [nextSemesterType, setNextSemesterType] = useState('odd');
  const [windowStatus, setWindowStatus] = useState({ isOpen: false, message: '' });

  // Appeal state
  const [showAppealModal, setShowAppealModal] = useState(false);
  const [showAppealHistoryModal, setShowAppealHistoryModal] = useState(false);
  const [appealMessage, setAppealMessage] = useState('');
  const [appealSubmitting, setAppealSubmitting] = useState(false);
  const [appealHistory, setAppealHistory] = useState([]);
  const [hasPendingAppeal, setHasPendingAppeal] = useState(false);
  const [pendingAppealData, setPendingAppealData] = useState(null);

  // Initialization
  useEffect(() => {
    const storedTeacherId = localStorage.getItem('teacher_id') || localStorage.getItem('userId');
    if (storedTeacherId) {
      setTeacherId(storedTeacherId);
      fetchAllData(storedTeacherId);
      checkPendingAppeal(storedTeacherId);
    } else {
      setError('Teacher ID not found. Please log in.');
      setLoading(false);
    }
  }, []);

  const fetchAllData = async (tid) => {
    try {
      setLoading(true);
      setLockCheckDone(false);
      let acadYear = '';
      let effectiveAcademicYear = '';
      let nextSemType = 'odd';
      
      // Fetch academic calendar
      const calRes = await fetch(`${API_BASE_URL}/academic-calendar/current`);
      if (calRes.ok) {
        const calData = await calRes.json();
        acadYear = calData.academic_year;
        effectiveAcademicYear = acadYear;
        setAcademicYear(effectiveAcademicYear);
      }

      // Fetch window status using academic year - this also returns semester type
      if (acadYear) {
        const winRes = await fetch(`${API_BASE_URL}/teachers/course-window/${acadYear}`);
        if (winRes.ok) {
          const winData = await winRes.json();
          
          // Use next_semester_type from the API (it's calculated from the DB's current_semester_type)
          nextSemType = winData.next_semester_type || 'odd';

          // Keep lock/preference checks aligned with backend tracking academic year
          effectiveAcademicYear = acadYear;
          setAcademicYear(effectiveAcademicYear);
          setNextSemesterType(nextSemType);
          
          // Check if window is currently open
          const parseLocalDate = (dateStr) => {
            const [year, month, day] = (dateStr || '').split('T')[0].split('-');
            return new Date(year, month - 1, day);
          };
          
          const now = new Date();
          const startDate = winData.window_start ? parseLocalDate(winData.window_start) : null;
          const endDate = winData.window_end ? parseLocalDate(winData.window_end) : null;
          
          // Extend endDate to end of day for inclusive comparison
          if (endDate) {
            endDate.setHours(23, 59, 59, 999);
          }
          
          let isOpen = false;
          let message = '';
          
          console.log('Window check - now:', now.toISOString().split('T')[0], 'start:', winData.window_start, 'end:', winData.window_end, 'isOpen calc:', !(now < startDate) && !(now > endDate));
          
          if (!winData.configured || !startDate || !endDate) {
            message = 'Selection window not configured';
          } else if (now < startDate) {
            message = `Opens on ${startDate.toLocaleDateString()}`;
          } else if (now > endDate) {
            message = `Closed on ${endDate.toLocaleDateString()}`;
          } else {
            isOpen = true;
            message = `Open until ${endDate.toLocaleDateString()}`;
          }
          
          setWindowStatus({ isOpen, message, startDate, endDate });
        } else {
          setWindowStatus({ isOpen: false, message: 'Could not fetch window status' });
        }
      }

      // Explicit lock check on page load using effective academic year
      if (effectiveAcademicYear) {
        const prefRes = await fetch(`${API_BASE_URL}/teachers/${tid}/course-preferences?academic_year=${effectiveAcademicYear}`);
        console.log('Lock-check response status:', prefRes.status, 'for year:', effectiveAcademicYear);
        if (prefRes.ok) {
          const prefData = await prefRes.json();
          console.log('Preference data returned:', prefData);
          const hasExistingPreferences = Array.isArray(prefData.preferences) && prefData.preferences.length > 0;
          console.log('hasExistingPreferences:', hasExistingPreferences, 'prefData.locked:', prefData.locked);
          setPreferencesLocked(!!prefData.locked || hasExistingPreferences);
          if (prefData.preferences && prefData.preferences.length > 0) {
            setSelectedCourses(prefData.preferences);
          }
        } else {
          // Fail-safe: prevent duplicate saves if lock-check endpoint is unavailable
          console.log('Lock-check failed with status:', prefRes.status, 'enabling fail-safe lock');
          setPreferencesLocked(true);
        }
      }
      console.log('Setting lockCheckDone to true');
      setLockCheckDone(true);

      // Fetch allocation summary for next semester type
      const allocRes = await fetch(`${API_BASE_URL}/teachers/${tid}/allocation-summary`);
      if (allocRes.ok) {
        const allocData = await allocRes.json();
        setAllocationSummary(allocData.summary);
      }

      // Fetch available courses for next semester type (SPLIT INTO CORE + EXTRA)
      const semesters = nextSemType === 'odd' ? [1, 3, 5, 7] : [2, 4, 6, 8];
      console.log('Fetching courses for semesters:', semesters, 'nextSemType:', nextSemType);
      let allCourses = [];
      
      const coursePromises = semesters.map(semester => 
        fetch(`${API_BASE_URL}/teachers/${tid}/semester/${semester}/courses?academic_year=${encodeURIComponent(effectiveAcademicYear || acadYear)}`)
          .then(res => {
            console.log(`Semester ${semester} response status:`, res.status);
            return res.ok ? res.json() : { coreCourses: [], extraCourses: [] };
          })
          .then(data => {
            console.log(`Semester ${semester} data:`, data);
            // Combine both core and extra courses with metadata
            const coreCourses = Array.isArray(data.coreCourses) 
              ? data.coreCourses.map(c => ({ ...c, semester, courseSource: 'core' })) 
              : [];
            const extraCourses = Array.isArray(data.extraCourses) 
              ? data.extraCourses.map(c => ({ ...c, semester, courseSource: 'extra' })) 
              : [];
            return {
              semester,
              courses: [...coreCourses, ...extraCourses]
            };
          })
          .catch(err => {
            console.error(`Error fetching semester ${semester}:`, err);
            return { semester, courses: [] };
          })
      );
      
      const results = await Promise.all(coursePromises);
      results.forEach(result => {
        console.log(`Adding ${result.courses.length} courses from semester ${result.semester}`);
        allCourses = [...allCourses, ...result.courses];
      });
      
      console.log('Total courses loaded:', allCourses.length);
      console.log('Sample courses with sources:', allCourses.slice(0, 3));
      
      // Separate core and extra for display hints
      const coreCount = allCourses.filter(c => c.courseSource === 'core').length;
      const extraCount = allCourses.filter(c => c.courseSource === 'extra').length;
      console.log(`Core courses: ${coreCount}, Extra courses (student-enrolled): ${extraCount}`);
      
      setAvailableCourses(allCourses);

      setLoading(false);
    } catch (err) {
      console.error('Error fetching data:', err);
      setError('Failed to load page. Please refresh.');
      setLockCheckDone(true);
      setLoading(false);
    }
  };

  const checkPendingAppeal = async (tid) => {
    try {
      const response = await fetch(`${API_BASE_URL}/teachers/appeals/pending?faculty_id=${tid}`);
      if (response.ok) {
        const data = await response.json();
        setHasPendingAppeal(data.has_pending_appeal || false);
        if (data.has_pending_appeal && data.appeal) {
          setPendingAppealData(data.appeal);
        } else {
          setPendingAppealData(null);
        }
      }
    } catch (err) {
      console.error('Error checking pending appeal:', err);
    }
  };

  const fetchAppealHistory = async () => {
    if (!teacherId) return;
    
    try {
      const response = await fetch(`${API_BASE_URL}/teachers/appeals/history?faculty_id=${teacherId}`);
      if (response.ok) {
        const data = await response.json();
        setAppealHistory(data || []);
        setShowAppealHistoryModal(true);
      } else {
        setError('Failed to fetch appeal history');
        setTimeout(() => setError(''), 3000);
      }
    } catch (err) {
      console.error('Error fetching appeal history:', err);
      setError('Network error. Please try again.');
      setTimeout(() => setError(''), 3000);
    }
  };

  const handleSubmitAppeal = async () => {
    if (!appealMessage.trim()) {
      setError('Please enter an appeal message');
      setTimeout(() => setError(''), 3000);
      return;
    }

    setAppealSubmitting(true);
    try {
      const response = await fetch(`${API_BASE_URL}/teachers/appeals`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          faculty_id: parseInt(teacherId),
          appeal_message: appealMessage
        })
      });

      if (response.ok) {
        const data = await response.json();
        setSuccess('Appeal submitted successfully! HR will review your request.');
        setShowAppealModal(false);
        setAppealMessage('');
        setHasPendingAppeal(true);
        setPendingAppealData(data);
        setTimeout(() => setSuccess(''), 5000);
      } else {
        const errorText = await response.text();
        setError(errorText || 'Failed to submit appeal');
        setTimeout(() => setError(''), 3000);
      }
    } catch (err) {
      console.error('Error submitting appeal:', err);
      setError('Network error. Please try again.');
      setTimeout(() => setError(''), 3000);
    } finally {
      setAppealSubmitting(false);
    }
  };

  // Helper functions
  const getSelectedCount = (type, source = null) => {
    if (source) {
      return selectedCourses.filter(c => c.course_type === type && c.course_source === source).length;
    }
    return selectedCourses.filter(c => c.course_type === type).length;
  };

  const getRequiredCount = (type, source = null) => {
    // Fixed requirements for each course type
    const typeName = (type || '').toLowerCase();
    if (typeName === 'theory' && source === 'core') return 2;
    if (typeName === 'theory' && source === 'extra') return 2;
    if (typeName === 'theory' && !source) return 4; // Total theory
    if (typeName === 'lab') return 1;
    if (typeName === 'theory_with_lab') return 1;
    // Fallback to allocation summary if type doesn't match fixed rules
    return allocationSummary?.type_summaries?.find(t => t.type_name === type)?.allocated || 0;
  };

  const toggleCourseSelection = (course) => {
    if (preferencesLocked) {
      setError('Your selections are locked. Contact HR to make changes.');
      setTimeout(() => setError(''), 3000);
      return;
    }

    const isSelected = selectedCourses.some(c => c.course_id === String(course.id));
    
    if (isSelected) {
      setSelectedCourses(selectedCourses.filter(c => c.course_id !== String(course.id)));
      setError('');
    } else {
      // Check limits based on course type and source
      const courseType = course.course_type;
      const courseSource = course.courseSource; // 'core' or 'extra'
      
      if (courseType === 'theory') {
        // For theory courses, check source-specific limits
        const required = getRequiredCount(courseType, courseSource);
        const selected = getSelectedCount(courseType, courseSource);
        const sourceName = courseSource === 'core' ? 'Core' : 'Student Elective';
        
        if (selected >= required) {
          setError(`You can only select ${required} ${sourceName} Theory courses`);
          setTimeout(() => setError(''), 4000);
          return;
        }
      } else {
        // For other course types, check overall limit
        const required = getRequiredCount(courseType);
        const selected = getSelectedCount(courseType);
        
        if (selected >= required) {
          setError(`You can only select ${required} ${courseType} courses`);
          setTimeout(() => setError(''), 4000);
          return;
        }
      }

      setSelectedCourses([...selectedCourses, {
        course_id: String(course.id),
        course_type: course.course_type,
        course_source: course.courseSource,
        semester: course.semester,
        course_name: course.course_name
      }]);
      setError('');
    }
  };

  const handleSave = async () => {
    if (!windowStatus.isOpen) {
      setError('The selection window is currently closed');
      return;
    }

    // Validate all types are satisfied using fixed requirements
    // Check core theory
    const coreTheorySelected = getSelectedCount('theory', 'core');
    if (coreTheorySelected !== 2) {
      setError(`Please select exactly 2 Core Theory courses (currently ${coreTheorySelected})`);
      return;
    }
    
    // Check student elective theory
    const electiveTheorySelected = getSelectedCount('theory', 'extra');
    if (electiveTheorySelected !== 2) {
      setError(`Please select exactly 2 Student Elective Theory courses (currently ${electiveTheorySelected})`);
      return;
    }
    
    // Check other types
    const theoryWithLabSelected = getSelectedCount('theory_with_lab');
    if (theoryWithLabSelected !== 1) {
      setError(`Please select exactly 1 Theory with Lab course (currently ${theoryWithLabSelected})`);
      return;
    }
    
    const labSelected = getSelectedCount('lab');
    if (labSelected !== 1) {
      setError(`Please select exactly 1 Lab course (currently ${labSelected})`);
      return;
    }

    setSaving(true);
    try {
      const response = await fetch(`${API_BASE_URL}/teachers/course-preferences`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          teacher_id: parseInt(teacherId),
          academic_year: academicYear,
          preferences: selectedCourses
        })
      });

      if (response.ok) {
        const data = await response.json().catch(() => ({}));
        setSuccess('✓ Your selections have been saved successfully!');
        setPreferencesLocked(!!data.locked || true);
        setTimeout(() => setSuccess(''), 4000);
      } else {
        const errorData = await response.json().catch(() => null);
        if (errorData?.locked) {
          setPreferencesLocked(true);
        }
        setError(errorData?.error || 'Failed to save selections');
      }
    } catch (err) {
      setError('Network error. Please try again.');
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  const handleClear = () => {
    if (window.confirm('Are you sure you want to clear all selections?')) {
      setSelectedCourses([]);
      setError('');
    }
  };

  const filteredCourses = availableCourses.filter(course => {
    // Normalize both course type and active tab for comparison
    const courseType = (course.course_type || '').toLowerCase().trim();
    const tabType = activeTab.toLowerCase().trim();
    const matchesType = courseType === tabType;
    const matchesSearch = searchQuery === '' || 
      course.course_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      course.course_code.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesType && matchesSearch;
  });

  // Loading state
  if (loading) {
    return (
      <MainLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center">
          <div className="text-center">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-4 border-indigo-500 border-t-transparent"></div>
            <p className="mt-4 text-lg text-gray-700">Loading courses...</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  // Calculate progress using fixed requirements
  const totalRequired = 2 + 2 + 1 + 1; // Core Theory + Elective Theory + Theory with Lab + Lab = 6
  const totalSelected = selectedCourses.length;
  const progressPercent = totalRequired > 0 ? (totalSelected / totalRequired) * 100 : 0;

  const adminCardClass = 'bg-white border rounded-lg';
  const adminInputClass = 'w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary';
  const adminPrimaryBtnClass = 'bg-primary text-white px-3 py-2 rounded-lg text-sm disabled:opacity-60 disabled:cursor-not-allowed';
  const adminSecondaryBtnClass = 'border border-gray-300 px-3 py-2 rounded-lg text-sm text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-60 disabled:cursor-not-allowed';
  const tabBaseClass = 'px-3 py-2 rounded-lg text-sm font-medium transition-all';

  const formatTypeLabel = (typeName) => {
    const normalized = String(typeName || '').toLowerCase().trim();
    if (normalized === 'theory_with_lab') return 'Theory + Lab';
    if (normalized === 'theory') return 'Theory';
    if (normalized === 'lab') return 'Lab';
    return String(typeName || 'Type');
  };

  const workloadTypeSummaries = Array.isArray(allocationSummary?.type_summaries)
    ? allocationSummary.type_summaries
    : [];

  // Main render
  return (
    <MainLayout
      title="Teacher Course Selection"
      subtitle="Select and submit your next-semester course preferences"
    >
      <div>
        {/* Sticky Header */}
        <div className={`${adminCardClass} sticky top-0 z-10`}>
          <div className="w-full px-6 py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                {/* <button
                  onClick={() => navigate(-1)}
                  className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                  title="Go back"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                  </svg>
                </button> */}
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">Course Selection for Next Semester</h1>
                  <p className="text-sm text-gray-600">
                    Academic Year {academicYear} • {nextSemesterType === 'odd' ? 'Odd' : 'Even'} Semesters 
                    ({nextSemesterType === 'odd' ? '1, 3, 5, 7' : '2, 4, 6, 8'})
                  </p>
                  <p className="text-xs text-gray-500 mt-1">
                    You can submit course preferences twice per academic year: once for odd semesters, once for even semesters
                  </p>
                </div>
              </div>
              
              {/* Window Status Badge and Appeal Buttons */}
              <div className="flex items-center gap-3">
                {/* Window Status Badge */}
                <div className={`px-4 py-2 rounded-full flex items-center gap-2 ${
                  windowStatus.isOpen ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                }`}>
                  <div className={`w-2 h-2 rounded-full ${
                    windowStatus.isOpen ? 'bg-green-500' : 'bg-red-500'
                  }`}></div>
                  <span className="text-sm font-semibold">{windowStatus.isOpen ? 'Window Open' : 'Window Closed'}</span>
                </div>

                {/* Appeal Buttons */}
                <button
                  onClick={fetchAppealHistory}
                  className="px-3 py-2 bg-gray-100 text-gray-700 text-sm font-medium rounded-lg hover:bg-gray-200 transition-colors flex items-center gap-2 border border-gray-300"
                  title="View Appeal History"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span className="hidden sm:inline">Appeal History</span>
                </button>
                
                {!hasPendingAppeal && windowStatus.isOpen && (
                  <button
                    onClick={() => setShowAppealModal(true)}
                    className={`${adminPrimaryBtnClass} flex items-center gap-2`}
                    title="Appeal Workload"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <span className="hidden sm:inline">Appeal Workload</span>
                  </button>
                )}
                
                {hasPendingAppeal && (
                  <div className="px-3 py-2 bg-yellow-50 text-yellow-800 text-sm font-medium rounded-lg border-2 border-yellow-300 flex items-center gap-2" title="Appeal under review">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <span className="hidden sm:inline">Pending Review</span>
                  </div>
                )}
              </div>
            </div>

            {/* Progress Bar */}
            <div className="mt-4">
              {workloadTypeSummaries.length > 0 && (
                <div className="mb-3 flex flex-wrap gap-2">
                  {workloadTypeSummaries.map((summary) => {
                    const typeName = formatTypeLabel(summary.type_name);
                    const allocated = Number(summary.allocated || 0);
                    const selected = Number(summary.selected || 0);
                    return (
                      <div
                        key={`${summary.course_type_id || summary.type_name}`}
                        className="px-3 py-1.5 rounded-lg border border-gray-200 bg-gray-50 text-xs"
                        title={`Allocation limit for ${typeName}`}
                      >
                        <span className="font-semibold text-gray-800">{typeName}</span>
                        <span className="mx-1 text-gray-400">•</span>
                        <span className="text-gray-600">Limit:</span>{' '}
                        <span className="font-semibold text-gray-900">{allocated}</span>
                        <span className="mx-1 text-gray-400">|</span>
                        <span className="text-gray-600">Selected:</span>{' '}
                        <span className="font-semibold text-primary">{selected}</span>
                      </div>
                    );
                  })}
                </div>
              )}

              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-gray-700">Progress: {totalSelected} / {totalRequired} courses selected</span>
                <span className="text-sm font-medium text-primary">{Math.round(progressPercent)}%</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-primary h-2 rounded-full transition-all duration-500"
                  style={{ width: `${progressPercent}%` }}
                ></div>
              </div>
            </div>
          </div>
        </div>

        {/* Main Content */}
        <div className="w-full px-6 py-8">
          {/* Alert Messages */}
          {error && (
            <div className="bg-red-50 border border-red-200 p-4 mb-6 rounded-lg">
              <div className="flex items-start">
                <svg className="w-5 h-5 text-red-500 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                <p className="ml-3 text-sm text-red-800">{error}</p>
              </div>
            </div>
          )}
          {success && (
            <div className="bg-green-50 border border-green-200 p-4 mb-6 rounded-lg">
              <div className="flex items-start">
                <svg className="w-5 h-5 text-green-500 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                </svg>
                <p className="ml-3 text-sm text-green-800">{success}</p>
              </div>
            </div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Left Sidebar - Requirements */}
            <div className="lg:col-span-1 space-y-4">
              <div className={`${adminCardClass} p-6 sticky top-32`}>
                <h2 className="text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                  <svg className="w-5 h-5 text-primary" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M9 2a1 1 0 000 2h2a1 1 0 100-2H9z" />
                    <path fillRule="evenodd" d="M4 5a2 2 0 012-2 3 3 0 003 3h2a3 3 0 003-3 2 2 0 012 2v11a2 2 0 01-2 2H6a2 2 0 01-2-2V5zm3 4a1 1 0 000 2h.01a1 1 0 100-2H7zm3 0a1 1 0 000 2h3a1 1 0 100-2h-3zm-3 4a1 1 0 100 2h.01a1 1 0 100-2H7zm3 0a1 1 0 100 2h3a1 1 0 100-2h-3z" clipRule="evenodd" />
                  </svg>
                  Requirements
                </h2>
                
                {[
                  { name: 'theory', source: 'core', displayName: 'Theory (Core)', required: 2 },
                  { name: 'theory', source: 'extra', displayName: 'Theory (Student Elective)', required: 2 },
                  { name: 'theory_with_lab', displayName: 'Theory with Lab', required: 1 },
                  { name: 'lab', displayName: 'Lab', required: 1 }
                ].map((type, idx) => {
                  const selected = getSelectedCount(type.name, type.source);
                  const isComplete = selected === type.required;
                  return (
                    <div key={`${type.name}-${type.source || 'all'}-${idx}`} className="mb-4 pb-4 border-b border-gray-100 last:border-0">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium text-gray-700">{type.displayName}</span>
                        {isComplete && (
                          <svg className="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                          </svg>
                        )}
                      </div>
                      <div className="flex items-baseline gap-2">
                        <span className={`text-2xl font-bold ${isComplete ? 'text-green-600' : 'text-primary'}`}>
                          {selected}
                        </span>
                        <span className="text-gray-400">/</span>
                        <span className="text-xl font-semibold text-gray-900">{type.required}</span>
                      </div>
                      <div className="mt-2 w-full bg-gray-200 rounded-full h-1.5">
                        <div 
                          className={`h-1.5 rounded-full transition-all ${
                            isComplete ? 'bg-green-500' : 'bg-primary'
                          }`}
                          style={{ width: `${type.required > 0 ? (selected / type.required) * 100 : 0}%` }}
                        ></div>
                      </div>
                    </div>
                  );
                })}

                {/* Action Buttons */}
                <div className="mt-6 space-y-3">
                  <button
                    onClick={handleSave}
                    disabled={saving || !lockCheckDone || preferencesLocked || totalSelected !== totalRequired}
                    className={`w-full py-3 rounded-lg font-semibold text-sm transition-all flex items-center justify-center gap-2 ${
                      !lockCheckDone
                        ? 'bg-gray-300 text-gray-600 cursor-not-allowed'
                        : preferencesLocked
                        ? 'bg-gray-300 text-gray-600 cursor-not-allowed'
                        : totalSelected !== totalRequired
                        ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                        : 'bg-primary text-white'
                    }`}
                    title={!lockCheckDone ? 'Checking lock status...' : preferencesLocked ? 'You have already submitted your preferences for this semester window' : totalSelected !== totalRequired ? `Select ${totalRequired - totalSelected} more course(s)` : ''}
                  >
                    {saving ? (
                      <>
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                        Saving...
                      </>
                    ) : !lockCheckDone ? (
                      <>
                        <div className="w-4 h-4 border-2 border-gray-500 border-t-transparent rounded-full animate-spin"></div>
                        Checking...
                      </>
                    ) : preferencesLocked ? (
                      <>
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd" />
                        </svg>
                        Locked
                      </>
                    ) : (
                      <>
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M7.707 10.293a1 1 0 10-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L11 11.586V6h5a2 2 0 012 2v7a2 2 0 01-2 2H4a2 2 0 01-2-2V8a2 2 0 012-2h5v5.586l-1.293-1.293zM9 4a1 1 0 012 0v2H9V4z" />
                        </svg>
                        Save Selection
                      </>
                    )}
                  </button>
                  <button
                    onClick={handleClear}
                    disabled={preferencesLocked}
                    className={`w-full py-3 rounded-lg font-semibold text-sm transition-all ${
                      preferencesLocked
                        ? 'bg-gray-200 text-gray-500 cursor-not-allowed'
                        : 'border border-gray-300 text-gray-700 bg-white hover:bg-gray-50'
                    }`}
                  >
                    Clear All
                  </button>
                </div>
              </div>
            </div>

            {/* Right Side - Course Selection */}
            <div className="lg:col-span-2">              {/* Course Categories Info */}
              <div className="grid grid-cols-2 gap-4 mb-6">
                <div className={`${adminCardClass} p-4`}>
                  <h3 className="text-sm font-semibold text-gray-900">Core Theory Courses (Select 2)</h3>
                  <p className="text-xs text-gray-600 mt-1">Required core courses for all students</p>
                </div>
                <div className={`${adminCardClass} p-4`}>
                  <h3 className="text-sm font-semibold text-gray-900">Student Elective Theory (Select 2)</h3>
                  <p className="text-xs text-gray-600 mt-1">Theory courses chosen by students</p>
                </div>
              </div>
              {/* Search and Filter */}
              <div className={`${adminCardClass} p-4 mb-6`}>
                <div className="flex flex-col sm:flex-row gap-4">
                  <div className="flex-1 relative">
                    <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                    </svg>
                    <input
                      type="text"
                      placeholder="Search courses..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className={`${adminInputClass} pl-10`}
                    />
                  </div>
                  <div className="flex gap-2">
                    {[
                      { value: 'theory', label: 'Theory' },
                      { value: 'theory_with_lab', label: 'Theory with Lab' },
                      { value: 'lab', label: 'Lab' }
                    ].map(type => (
                      <button
                        key={type.value}
                        onClick={() => setActiveTab(type.value)}
                        className={`${tabBaseClass} ${
                          activeTab === type.value
                            ? 'bg-primary text-white'
                            : 'border border-gray-300 bg-white text-gray-700 hover:bg-gray-50'
                        }`}
                      >
                        {type.label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* Course Grid */}
              {filteredCourses.length === 0 ? (
                <div className={`${adminCardClass} p-12 text-center`}>
                  <svg className="mx-auto w-16 h-16 text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <p className="text-gray-500 text-lg">No courses found</p>
                  <p className="text-gray-400 text-sm mt-1">Try adjusting your search or filter</p>
                </div>
              ) : (
                <div className="grid grid-cols-1 gap-4">
                  {filteredCourses.map(course => {
                    const isSelected = selectedCourses.some(c => c.course_id === String(course.id));
                    return (
                      <div
                        key={course.id}
                        onClick={() => toggleCourseSelection(course)}
                        className={`bg-white rounded-lg border p-5 cursor-pointer transition-all ${
                          isSelected
                            ? 'border-primary bg-blue-50'
                            : 'border-gray-200 hover:border-primary/40'
                        }`}
                      >
                        <div className="flex items-start gap-4">
                          <div className={`flex-shrink-0 w-10 h-10 rounded-md flex items-center justify-center ${
                            isSelected ? 'bg-primary text-white' : 'bg-gray-100 text-gray-600'
                          }`}>
                            {isSelected ? (
                              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                              </svg>
                            ) : (
                              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                              </svg>
                            )}
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2 mb-1">
                              <h3 className="text-base font-semibold text-gray-900">{course.course_name}</h3>
                              {course.courseSource === 'extra' && (
                                <span className="inline-block px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-semibold rounded">
                                  Student Choice
                                </span>
                              )}
                              {course.courseSource === 'core' && (
                                <span className="inline-block px-2 py-0.5 bg-gray-100 text-gray-700 text-xs font-semibold rounded">
                                  Core
                                </span>
                              )}
                            </div>
                            <div className="flex flex-wrap gap-3 text-sm text-gray-600">
                              <span className="flex items-center gap-1">
                                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clipRule="evenodd" />
                                </svg>
                                {course.course_code}
                              </span>
                              <span className="flex items-center gap-1">
                                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                  <path d="M10.394 2.08a1 1 0 00-.788 0l-7 3a1 1 0 000 1.84L5.25 8.051a.999.999 0 01.356-.257l4-1.714a1 1 0 11.788 1.838L7.667 9.088l1.94.831a1 1 0 00.787 0l7-3a1 1 0 000-1.838l-7-3zM3.31 9.397L5 10.12v4.102a8.969 8.969 0 00-1.05-.174 1 1 0 01-.89-.89 11.115 11.115 0 01.25-3.762zM9.3 16.573A9.026 9.026 0 007 14.935v-3.957l1.818.78a3 3 0 002.364 0l5.508-2.361a11.026 11.026 0 01.25 3.762 1 1 0 01-.89.89 8.968 8.968 0 00-5.35 2.524 1 1 0 01-1.4 0zM6 18a1 1 0 001-1v-2.065a8.935 8.935 0 00-2-.712V17a1 1 0 001 1z" />
                                </svg>
                                Sem {course.semester}
                              </span>
                            </div>
                          </div>
                          <div className={`flex-shrink-0 px-4 py-2 rounded-lg text-xs font-bold ${
                            isSelected
                              ? 'bg-primary text-white'
                              : 'border border-gray-300 bg-white text-gray-700'
                          }`}>
                            {isSelected ? 'SELECTED' : 'SELECT'}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Appeal Modal */}
      {showAppealModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full p-6">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Appeal Workload Assignment</h2>
            
            <div className="mb-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
              <p className="text-sm text-blue-800">
                <strong>Note:</strong> By submitting this appeal, you are requesting HR to review and potentially reduce your assigned workload. 
                You will not be able to select courses until HR reviews your appeal.
              </p>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Appeal Message *
              </label>
              <textarea
                value={appealMessage}
                onChange={(e) => setAppealMessage(e.target.value)}
                placeholder="Please explain why you need a workload reduction..."
                rows="5"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary resize-none"
              />
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowAppealModal(false);
                  setAppealMessage('');
                }}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg text-gray-700 text-sm hover:bg-gray-50"
                disabled={appealSubmitting}
              >
                Cancel
              </button>
              <button
                onClick={handleSubmitAppeal}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg text-sm disabled:opacity-50"
                disabled={appealSubmitting || !appealMessage.trim()}
              >
                {appealSubmitting ? 'Submitting...' : 'Submit Appeal'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Appeal History Modal */}
      {showAppealHistoryModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
            <div className="p-6 border-b border-gray-200">
              <h2 className="text-2xl font-bold text-gray-900">Appeal History</h2>
              <p className="text-sm text-gray-600 mt-1">View all your past workload appeals and their outcomes</p>
            </div>
            
            <div className="flex-1 overflow-y-auto p-6">
              {appealHistory.length === 0 ? (
                <div className="text-center py-12">
                  <svg className="w-16 h-16 text-gray-400 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  <p className="text-gray-600 font-medium">No appeal history found</p>
                  <p className="text-gray-500 text-sm mt-1">Your submitted appeals will appear here</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {appealHistory.map((appeal) => (
                    <div 
                      key={appeal.id} 
                      className={`border rounded-lg p-5 ${
                        appeal.appeal_status === 0 
                          ? 'border-yellow-200 bg-yellow-50' 
                          : appeal.hr_action?.String === 'APPROVED' 
                          ? 'border-primary bg-primary-50' 
                          : 'border-red-200 bg-red-50'
                      }`}
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-2">
                          <span className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${
                            appeal.appeal_status === 0 
                              ? 'bg-yellow-200 text-yellow-800' 
                              : appeal.hr_action?.String === 'APPROVED' 
                              ? 'bg-primary-100 text-primary-800' 
                              : 'bg-red-200 text-red-800'
                          }`}>
                            {appeal.appeal_status === 0 ? 'Pending' : appeal.hr_action?.String || 'Resolved'}
                          </span>
                          <span className="text-xs text-gray-500">
                            Appeal #{appeal.id}
                          </span>
                        </div>
                        <span className="text-xs text-gray-500">
                          {new Date(appeal.created_at).toLocaleDateString('en-US', { 
                            year: 'numeric', 
                            month: 'short', 
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit'
                          })}
                        </span>
                      </div>

                      <div className="space-y-3">
                        <div>
                          <label className="block text-xs font-semibold text-gray-700 mb-1">Your Message:</label>
                          <p className="text-sm text-gray-800 bg-white bg-opacity-50 p-3 rounded border border-gray-200 whitespace-pre-wrap">
                            {appeal.appeal_message}
                          </p>
                        </div>

                        {appeal.hr_message?.Valid && appeal.hr_message?.String && (
                          <div>
                            <label className="block text-xs font-semibold text-gray-700 mb-1">HR Response:</label>
                            <p className="text-sm text-gray-800 bg-white bg-opacity-50 p-3 rounded border border-gray-200 whitespace-pre-wrap">
                              {appeal.hr_message.String}
                            </p>
                          </div>
                        )}

                        {appeal.resolved_at?.Valid && (
                          <div className="text-xs text-gray-600 pt-2 border-t border-gray-300">
                            Resolved on: {new Date(appeal.resolved_at.Time).toLocaleDateString('en-US', { 
                              year: 'numeric', 
                              month: 'short', 
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit'
                            })}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="p-6 border-t border-gray-200">
              <button
                onClick={() => setShowAppealHistoryModal(false)}
                className="w-full px-4 py-2 btn-primary-custom"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </MainLayout>
  );
};

export default TeacherCourseSelectionPage;
