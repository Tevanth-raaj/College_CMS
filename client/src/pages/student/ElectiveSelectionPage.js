import React, { useState, useEffect } from 'react';
import MainLayout from '../../components/MainLayout';
import { API_BASE_URL } from '../../config';

const ElectiveSelectionPage = () => {
  const [selections, setSelections] = useState({});
  const [totalCreditUsed, setTotalCreditUsed] = useState(0);
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [message, setMessage] = useState({ type: '', text: '' });
  const [loading, setLoading] = useState(true);
  const [electiveData, setElectiveData] = useState(null);
  const [groupedElectives, setGroupedElectives] = useState({});
  // null = no optional track chosen, or one of 'HONOR' | 'MINOR'
  const [optionalType, setOptionalType] = useState(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const userEmail = localStorage.getItem('userEmail'); // Get email from login
  const userName = localStorage.getItem('userName');
  const userId = localStorage.getItem('userId');

  useEffect(() => {
    fetchElectives();
  }, []);

  const fetchElectives = async () => {
    try {
      setLoading(true);
      const url = `${API_BASE_URL}/students/electives/available?email=${userEmail}`;
      const response = await fetch(url);

      if (response.ok) {
        const data = await response.json();
        console.log('Elective Data:', data);
        setElectiveData(data);

        const grouped = {};
        if (data.slots && data.slots.length > 0) {
          data.slots.forEach(slot => {
            grouped[slot.slot_name || 'Unassigned'] = {
              slot_id: slot.slot_id,
              slot_name: slot.slot_name,
              slot_type: slot.slot_type,
              is_active: slot.is_active,
              courses: slot.courses || []
            };
          });
        }
        setGroupedElectives(grouped);

        let finalSelections = {};

        if (data.existing_selections && Object.keys(data.existing_selections).length > 0) {
          console.log('Loading existing selections from backend:', data.existing_selections);
          // Only keep selections whose slot_name still exists in the current slot list
          // — filters out ghost/stale entries from old UI iterations
          const validSlotNames = new Set((data.slots || []).map(s => s.slot_name));
          finalSelections = Object.fromEntries(
            Object.entries(data.existing_selections).filter(([k]) => validSlotNames.has(k))
          );
        }
        
        // Detect which optional track (HONOR/MINOR only) the student previously selected
        if (Object.keys(finalSelections).length > 0 && data.slots) {
          const hasHonor = data.slots.some(s => s.slot_type === 'HONOR' && finalSelections[s.slot_name]);
          const hasMinor = data.slots.some(s => s.slot_type === 'MINOR' && finalSelections[s.slot_name]);
          if (hasHonor) setOptionalType('HONOR');
          else if (hasMinor) setOptionalType('MINOR');
        }
        
        if (Object.keys(finalSelections).length > 0) {
          setSelections(finalSelections);

          // Calculate credits using data directly (avoid stale closure on electiveData)
          let total = 0;
          Object.entries(finalSelections).forEach(([slotName, courseId]) => {
            if (courseId === 'NOT_OPTED') return; // keep for any legacy data safety
            const slot = data.slots.find(s => s.slot_name === slotName);
            if (slot && ['HONOR', 'MINOR', 'ADDON'].includes(slot.slot_type)) {
              const course = slot.courses.find(c => c.hod_selection_id === courseId);
              if (course) total += course.credits;
            }
          });
          setTotalCreditUsed(total);

          // Check submitted: all REQUIRED slots are filled
          const requiredSlots = data.slots.filter(slot =>
            ['PROFESSIONAL', 'OPEN', 'MIXED'].includes(slot.slot_type)
          );
          const selectedRequiredSlots = Object.keys(finalSelections).filter(slotName => {
            const slot = data.slots.find(s => s.slot_name === slotName);
            return slot && ['PROFESSIONAL', 'OPEN', 'MIXED'].includes(slot.slot_type);
          });

          setIsSubmitted(
            requiredSlots.length > 0 &&
            selectedRequiredSlots.length === requiredSlots.length
          );
        } else {
          setSelections({});
          setIsSubmitted(false);
          setTotalCreditUsed(0);
          localStorage.removeItem(`elective_submitted_${userEmail}_sem${data.next_semester}`);
          localStorage.removeItem(`elective_submission_${userEmail}_sem${data.next_semester}`);
        }
      } else {
        const errorText = await response.text();
        setMessage({ type: 'error', text: errorText || 'Failed to fetch electives' });
      }
    } catch (error) {
      console.error('Error fetching electives:', error);
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  const loadSavedSelections = (semester) => {
    const savedSelections = localStorage.getItem(`elective_selections_${userEmail}_sem${semester}`);
    const savedSubmitted = localStorage.getItem(`elective_submitted_${userEmail}_sem${semester}`);

    if (savedSelections) {
      const parsed = JSON.parse(savedSelections);
      setSelections(parsed);
      calculateTotalCredits(parsed);
    }

    if (savedSubmitted === 'true') {
      setIsSubmitted(true);
    }
  };

  // Calculate TOTAL credit usage for honour/minor/addon (common pool)
  const calculateTotalCredits = (currentSelections) => {
    if (!electiveData || !electiveData.slots) return;

    let total = 0;
    console.log('Calculating credits for selections:', currentSelections);
    Object.entries(currentSelections).forEach(([slotName, courseId]) => {
      // Check if this slot is HONOR/MINOR/ADDON by slot type
      const slot = electiveData.slots.find(s => s.slot_name === slotName);
      if (slot && ['HONOR', 'MINOR', 'ADDON'].includes(slot.slot_type)) {
        const course = slot.courses.find(c => c.hod_selection_id === courseId);
        if (course) total += course.credits;
      }
    });
    setTotalCreditUsed(total);
  };

  const handleSelection = (slotName, courseId, credits) => {
    if (isSubmitted) return;

    // null courseId = unenroll (remove from selections)
    if (courseId === null) {
      const newSelections = { ...selections };
      delete newSelections[slotName];
      setSelections(newSelections);
      calculateTotalCredits(newSelections);
      localStorage.setItem(
        `elective_selections_${userEmail}_sem${electiveData.next_semester}`,
        JSON.stringify(newSelections)
      );
      return;
    }

    // Find the slot to check its type
    const slot = electiveData.slots.find(s => s.slot_name === slotName);

    // Check COMMON credit limit for honour/minor/addon (total 8 credits shared)
    if (slot && ['HONOR', 'MINOR', 'ADDON'].includes(slot.slot_type)) {
      let adjustedCredit = totalCreditUsed;

      // If changing selection, subtract old credits first
      if (selections[slotName]) {
        const oldCourse = slot.courses.find(c => c.hod_selection_id === selections[slotName]);
        if (oldCourse) {
          adjustedCredit -= oldCourse.credits;
        }
      }

      // Check if new selection exceeds total 8 credit limit
      if (adjustedCredit + credits > 8) {
        setMessage({
          type: 'error',
          text: `Cannot select. Total credit limit exceeded. Maximum 8 credits allowed for Honour/Minor/Add-On combined.`
        });
        setTimeout(() => setMessage({ type: '', text: '' }), 3000);
        return;
      }
    }

    const newSelections = {
      ...selections,
      [slotName]: courseId
    };

    setSelections(newSelections);
    calculateTotalCredits(newSelections);

    // Save to localStorage with semester-specific key
    localStorage.setItem(
      `elective_selections_${userEmail}_sem${electiveData.next_semester}`,
      JSON.stringify(newSelections)
    );
  };

  const handleSubmit = async () => {
    if (isSubmitting) return; // Prevent double-submit
    const requiredSlots = Object.keys(groupedElectives).filter(slotName => {
      const slotType = groupedElectives[slotName].slot_type;
      return slotType === 'PROFESSIONAL' || slotType === 'MIXED' || slotType === 'OPEN';
    });

    const requiredSelections = Object.keys(selections).filter(slotName => {
      const slotType = groupedElectives[slotName]?.slot_type;
      return slotType === 'PROFESSIONAL' || slotType === 'MIXED' || slotType === 'OPEN';
    });

    if (requiredSelections.length < requiredSlots.length) {
      setMessage({
        type: 'error',
        text: `Please select one course from each required slot. Selected: ${requiredSelections.length}/${requiredSlots.length} (Honors/Minor/Add-ons are optional)`
      });
      setTimeout(() => setMessage({ type: '', text: '' }), 3000);
      return;
    }

    // If student chose HONOUR track, all HONOR slots must be filled
    if (optionalType === 'HONOR') {
      const allHonorSlots = Object.keys(groupedElectives).filter(s => groupedElectives[s].slot_type === 'HONOR');
      const selectedHonorSlots = allHonorSlots.filter(s => selections[s]);
      if (selectedHonorSlots.length < allHonorSlots.length) {
        setMessage({ type: 'error', text: `Please select a course for all ${allHonorSlots.length} Honour slots (${selectedHonorSlots.length}/${allHonorSlots.length} filled).` });
        setTimeout(() => setMessage({ type: '', text: '' }), 4000);
        return;
      }
    }
    // If student chose MINOR track, all MINOR slots must be filled
    if (optionalType === 'MINOR') {
      const allMinorSlots = Object.keys(groupedElectives).filter(s => groupedElectives[s].slot_type === 'MINOR');
      const selectedMinorSlots = allMinorSlots.filter(s => selections[s]);
      if (selectedMinorSlots.length < allMinorSlots.length) {
        setMessage({ type: 'error', text: `Please select a course for all ${allMinorSlots.length} Minor slots (${selectedMinorSlots.length}/${allMinorSlots.length} filled).` });
        setTimeout(() => setMessage({ type: '', text: '' }), 4000);
        return;
      }
    }

    console.log('Submitting selections:', selections);
    console.log('Email:', userEmail);
    console.log('Semester:', electiveData.next_semester);

    setIsSubmitting(true);
    try {
      const response = await fetch(
        `${API_BASE_URL}/students/electives/selections?email=${userEmail}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            selections: selections,
            semester: electiveData.next_semester
          })
        }
      );

      console.log('Submit response status:', response.status);
      const responseText = await response.text();
      console.log('Submit response body:', responseText);

      if (response.ok) {
        // Mark as submitted
        setIsSubmitted(true);
        localStorage.setItem(`elective_submitted_${userEmail}_sem${electiveData.next_semester}`, 'true');

        // Save submission data
        const submissionData = {
          selections: selections,
          semester: electiveData.next_semester,
          submittedAt: new Date().toISOString(),
          userEmail: userEmail,
          userName: userName,
          totalCredits: totalCreditUsed
        };
        localStorage.setItem(
          `elective_submission_${userEmail}_sem${electiveData.next_semester}`,
          JSON.stringify(submissionData)
        );

        console.log('Submission Data:', submissionData);
        setMessage({ type: 'success', text: 'Selections submitted successfully!' });
        setTimeout(() => setMessage({ type: '', text: '' }), 3000);
      } else {
        let errorMsg = 'Failed to save selections';
        try {
          const error = JSON.parse(responseText);
          errorMsg = error.error || error.message || errorMsg;
        } catch (e) {
          errorMsg = responseText || errorMsg;
        }
        setMessage({ type: 'error', text: errorMsg });
        setTimeout(() => setMessage({ type: '', text: '' }), 5000);
      }
    } catch (error) {
      console.error('Error saving selections:', error);
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
      setTimeout(() => setMessage({ type: '', text: '' }), 5000);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Clear all optional slots and switch track
  // For HONOR/MINOR: auto-enroll the single course in each slot
  const handleOptionalTypeChange = (type) => {
    if (isSubmitted) return;

    // Clear only HONOR/MINOR optional selections (ADDON stays — always auto-enrolled)
    const newSelections = { ...selections };
    Object.keys(groupedElectives).forEach(slotName => {
      if (['HONOR', 'MINOR'].includes(groupedElectives[slotName].slot_type)) {
        delete newSelections[slotName];
      }
    });

    // For HONOR/MINOR: auto-select the single course in each slot
    if (type === 'HONOR' || type === 'MINOR') {
      let creditTotal = 0;
      Object.entries(groupedElectives).forEach(([slotName, slotData]) => {
        if (slotData.slot_type === type && slotData.courses.length > 0) {
          const course = slotData.courses[0]; // only 1 course per slot
          newSelections[slotName] = course.hod_selection_id;
          creditTotal += course.credits;
        }
      });
      setTotalCreditUsed(creditTotal);
    } else {
      setTotalCreditUsed(0);
    }

    setOptionalType(type);
    setSelections(newSelections);
    if (type !== 'HONOR' && type !== 'MINOR') calculateTotalCredits(newSelections);
    localStorage.setItem(`elective_selections_${userEmail}_sem${electiveData?.next_semester}`, JSON.stringify(newSelections));
  };

  const getCategoryTitle = (category) => {
    const titles = {
      'professional': 'Professional Elective',
      'open': 'Open Elective',
      'honour': 'Honour',
      'minor': 'Minor',
      'addon': 'Add-On'
    };
    return titles[category] || category.toUpperCase();
  };

  // Slot type config — label, icon, colour accent
  const slotTypeConfig = {
    PROFESSIONAL: { label: 'Professional Elective', icon: '📚', border: 'border-blue-300', required: true },
    OPEN:         { label: 'Open Elective',          icon: '🌐', border: 'border-green-300', required: true },
    MIXED:        { label: 'Professional + Open',    icon: '📚', border: 'border-purple-300', required: true },
    HONOR:        { label: 'Honour Course',          icon: '🏆', border: 'border-yellow-300', required: false },
    MINOR:        { label: 'Minor Course',           icon: '📖', border: 'border-orange-300', required: false },
    ADDON:        { label: 'Add-On Course',          icon: '➕', border: 'border-gray-300',   required: false },
  };

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center min-h-screen">
          <div className="flex flex-col items-center gap-3">
            <div className="w-10 h-10 border-4 border-gray-300 border-t-gray-800 rounded-full animate-spin" />
            <p className="text-gray-600 text-lg font-medium">Loading your electives…</p>
          </div>
        </div>
      </MainLayout>
    );
  }

  if (!electiveData || !electiveData.slots || electiveData.slots.length === 0) {
    return (
      <MainLayout>
        <div className="max-w-2xl mx-auto py-20 text-center">
          <div className="text-5xl mb-4 text-gray-300">
            <svg xmlns="http://www.w3.org/2000/svg" className="w-12 h-12 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8m0 8a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h14a2 2 0 012 2v8z" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-gray-800 mb-2">No Electives Available</h1>
          <p className="text-gray-500">There are no electives available for the next semester at this time.</p>
          <p className="text-gray-400 text-sm mt-1">Please contact your HOD if you believe this is an error.</p>
        </div>
      </MainLayout>
    );
  }

  const requiredSlots = Object.keys(groupedElectives).filter(s =>
    ['PROFESSIONAL', 'MIXED', 'OPEN'].includes(groupedElectives[s].slot_type)
  );
  const filledRequired = requiredSlots.filter(s => selections[s]);
  const progressPct = requiredSlots.length > 0 ? Math.round((filledRequired.length / requiredSlots.length) * 100) : 100;

  // Derive which optional track types are available (HONOR/MINOR only — ADDON is always auto-enrolled)
  const availableOptionalTypes = [...new Set(
    Object.values(groupedElectives)
      .filter(s => ['HONOR','MINOR'].includes(s.slot_type))
      .map(s => s.slot_type)
  )];

  const addonSlotEntries = Object.entries(groupedElectives).filter(([, s]) => s.slot_type === 'ADDON');

  const requiredSlotEntries = Object.entries(groupedElectives).filter(([, s]) =>
    ['PROFESSIONAL', 'OPEN', 'MIXED'].includes(s.slot_type)
  );
  const optionalSlotEntries = optionalType
    ? Object.entries(groupedElectives).filter(([, s]) => s.slot_type === optionalType)
    : [];

  // Track-specific fill counts (HONOR/MINOR only)
  const optionalSlotsForTrack = optionalType
    ? Object.keys(groupedElectives).filter(s => groupedElectives[s].slot_type === optionalType)
    : [];
  const filledOptional = optionalSlotsForTrack.filter(s => selections[s]);

  // HONOR/MINOR are auto-filled on track pick, so always complete
  const optionalTrackComplete =
    !optionalType || optionalType === 'HONOR' || optionalType === 'MINOR' ||
    optionalType === 'ADDON'; // ADDON is always optional per-slot, never blocks submit

  const canSubmit = filledRequired.length === requiredSlots.length && (electiveData?.window_open ?? true);

  // Shared required slot card renderer
  const renderRequiredCard = (slotName, slotData) => {
    const cfg = slotTypeConfig[slotData.slot_type];
    const selectedValue = selections[slotName] || '';
    const selectedCourse = slotData.courses.find(c => c.hod_selection_id === selectedValue);

    const handleDropdownChange = (e) => {
      const val = e.target.value;
      if (!val) {
        const newSel = { ...selections };
        delete newSel[slotName];
        setSelections(newSel);
        calculateTotalCredits(newSel);
        localStorage.setItem(`elective_selections_${userEmail}_sem${electiveData.next_semester}`, JSON.stringify(newSel));
        return;
      }
      const course = slotData.courses.find(c => String(c.hod_selection_id) === String(val));
      handleSelection(slotName, course.hod_selection_id, course.credits);
    };

    return (
      <div
        key={slotName}
        className={`bg-white rounded-md border shadow-sm overflow-hidden transition-all ${
          isSubmitted ? 'opacity-60 pointer-events-none' : ''
        } ${selectedValue ? 'border-gray-300' : 'border-gray-200 hover:border-gray-300'}`}
      >
        <div className="px-4 pt-4 pb-3 bg-gray-50 border-b border-gray-100">
          <div className="flex items-center gap-2 mb-1">
            {/* <span className="text-sm">{cfg.icon}</span> */}
            <span className={"inline-flex items-center px-2 py-0.5 mb-1 rounded-full text-xs font-bold bg-background text-primary"}>{cfg.label}</span>
            <span className="text-xs text-red-500 font-semibold">Required</span>
          </div>
          <h2 className="text-sm font-bold text-gray-900">{slotName}</h2>
        </div>
        <div className="px-4 py-4">
          <div className="relative">
            <select
              value={selectedValue}
              onChange={handleDropdownChange}
              disabled={isSubmitted || electiveData?.window_open === false}
              className={`w-full appearance-none bg-white border-2 rounded-md px-4 py-3 pr-10 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-offset-1 transition cursor-pointer ${
                selectedValue ? 'border-gray-900 text-gray-800 focus:ring-gray-400' : 'border-gray-200 text-gray-400 focus:ring-blue-300'
              } disabled:opacity-60 disabled:cursor-not-allowed`}
            >
              <option value="" style={{ color: '#111827' }}>— Choose a course —</option>
              {slotData.courses.map(course => (
                <option key={course.hod_selection_id} value={course.hod_selection_id} style={{ color: '#111827' }}>
                  {course.course_code} — {course.course_name} ({course.credits} cr)
                </option>
              ))}
            </select>
            <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center text-gray-400">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>
          {selectedValue && selectedCourse && (
            <div className="mt-3 flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-md">
              <svg className="w-4 h-4 text-green-600 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
              </svg>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold text-gray-900 truncate">{selectedCourse.course_name}</p>
                <p className="text-xs text-gray-500">{selectedCourse.course_code} · {selectedCourse.credits} cr</p>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <MainLayout
    title="Elective Selection"
    >
      <div className="py-8 px-6">

        {/* ── Page header ── */}
        <div className="mb-8">
          <p className="text-sm font-semibold uppercase tracking-wider text-gray-400 mb-1">Academic Registration</p>
          <h1 className="text-3xl font-bold text-gray-900">Elective Selection</h1>
          <p className="text-gray-500 mt-1">Semester {electiveData.next_semester}</p>
        </div>

        {/* ── Toast message ── */}
        {message.text && (
          <div className={`mb-5 flex items-start gap-3 p-4 rounded-xl text-sm font-medium shadow-sm ${
            message.type === 'success'
              ? 'bg-green-50 text-green-800 border border-green-200'
              : 'bg-red-50 text-red-800 border border-red-200'
          }`}>
            <span className="mt-0.5">
              {message.type === 'success' ? (
                <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
              ) : (
                <svg className="w-5 h-5 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
              )}
            </span>
            <span>{message.text}</span>
          </div>
        )}

        {/* —— Window closed banner —— */}
        {!isSubmitted && electiveData?.window_open === false && (
          <div className="mb-5 flex items-center gap-3 p-4 bg-amber-50 border border-amber-200 rounded-xl text-amber-800 text-sm font-semibold shadow-sm">
            <svg className="w-5 h-5 text-amber-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 11V7a4 4 0 10-8 0v4" /><rect x="3" y="11" width="14" height="10" rx="2" ry="2"/></svg>
            <span>
              Elective selection window is closed.
              {electiveData.window_start && electiveData.window_end
                ? ` Open from ${electiveData.window_start} to ${electiveData.window_end}.`
                : ' Contact your HOD for the selection schedule.'}
            </span>
          </div>
        )}

        {/* ── Submitted banner ── */}
        {isSubmitted && (
          <div className="mb-5 flex items-center gap-3 p-4 bg-emerald-50 border border-emerald-200 rounded-xl text-emerald-800 text-sm font-semibold shadow-sm">
            <svg className="w-5 h-5 text-emerald-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 11V7a4 4 0 10-8 0v4" /><rect x="3" y="11" width="14" height="10" rx="2" ry="2"/></svg>
            Your selections have been submitted and are now locked.
          </div>
        )}

        {/* ════════════════════════════════════════ */}
        {/* SECTION 1 — Required Electives                      */}
        {/* ════════════════════════════════════════ */}
        <div className="mb-8 card-custom p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-bold text-gray-900">Required Electives</h2>
              <p className="text-xs text-gray-400 mt-0.5">Select one course for each slot below</p>
            </div>
            <div className="flex items-center gap-2">
              <span className={`text-sm font-bold ${
                filledRequired.length === requiredSlots.length ? 'text-green-600' : 'text-gray-500'
              }`}>{filledRequired.length}/{requiredSlots.length}</span>
              {filledRequired.length === requiredSlots.length && (
                <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              )}
            </div>
          </div>

          {/* Required progress bar */}
          <div className="w-full bg-gray-100 rounded-full h-1.5 mb-5">
            <div
              className={`h-1.5 rounded-full transition-all duration-500 ${
                progressPct === 100 ? 'bg-green-500' : 'bg-primary'
              }`}
              style={{ width: `${progressPct}%` }}
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {requiredSlotEntries.map(([slotName, slotData]) => renderRequiredCard(slotName, slotData))}
          </div>
        </div>

        {/* ════════════════════════════════════════ */}
        {/* SECTION 2 — Optional Track                           */}
        {/* ════════════════════════════════════════ */}
        {availableOptionalTypes.length > 0 && (
          <div className="mb-8 card-custom p-6">
            <div className="border-t border-gray-200 pt-8 mb-5">
              <h2 className="text-lg font-bold text-gray-900">Optional Track</h2>
              <p className="text-xs text-gray-400 mt-0.5">Choose one additional track to enroll in, or skip</p>
            </div>

            {/* Track picker pills */}
            <div className={`grid gap-3 mb-6 ${
              availableOptionalTypes.length === 1 ? 'grid-cols-2' :
              availableOptionalTypes.length === 2 ? 'grid-cols-3' : 'grid-cols-4'
            }`}>
              {[
                { type: null,    label: 'None',   sub: 'Skip optional track', icon: '—',  activeBg: 'bg-gray-800 text-white', activeBorder: 'border-gray-800' },
                availableOptionalTypes.includes('HONOR') && {
                  type: 'HONOR', label: 'Honour', sub: 'Auto-enrolled in all', icon: '🏆', activeBg: 'bg-amber-500 text-white',  activeBorder: 'border-amber-500' },
                availableOptionalTypes.includes('MINOR') && {
                  type: 'MINOR', label: 'Minor',  sub: 'Auto-enrolled in all', icon: '📖', activeBg: 'bg-orange-500 text-white', activeBorder: 'border-orange-500' },
              ].filter(Boolean).map(({ type, label, sub, icon, activeBg, activeBorder }) => {
                const isActive = optionalType === type;
                return (
                    <button
                    key={String(type)}
                    disabled={isSubmitted}
                    onClick={() => handleOptionalTypeChange(type)}
                    className={`flex flex-col items-center gap-1.5 px-3 py-5 rounded-md border-2 transition-all font-medium disabled:opacity-50 disabled:cursor-not-allowed select-none ${
                      isActive
                        ? 'bg-primary text-white border-primary shadow-lg scale-[1.03]'
                        : 'bg-white border-gray-200 text-gray-600 hover:border-gray-400 hover:bg-gray-50'
                    }`}
                  >
                    <span className="text-2xl leading-none">{icon}</span>
                    <span className="text-sm font-bold mt-0.5">{label}</span>
                    <span className={`text-xs leading-tight text-center ${
                      isActive ? 'opacity-80' : 'text-gray-400'
                    }`}>{sub}</span>
                  </button>
                );
              })}
            </div>

            {/* HONOR / MINOR / ADDON: auto-enrolled cards (read-only, no interaction) */}
            {optionalType && optionalType !== null && (
              <div>
                <div className="flex items-center gap-2 mb-3 px-1">
                  <svg className="w-4 h-4 text-emerald-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <p className="text-sm font-semibold text-emerald-800">
                    You are enrolled in all {optionalType === 'MINOR' ? 'Minor' : 'Honour'} courses below
                    {totalCreditUsed > 0 && <span className="ml-2 text-emerald-700 font-bold">({totalCreditUsed} credits)</span>}
                  </p>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                  {optionalSlotEntries.map(([slotName, slotData]) => {
                    const course = slotData.courses[0];
                    const cfg = slotTypeConfig[slotData.slot_type];
                    const colors = {
                      HONOR: { border: 'border-primary', bg: 'bg-background', divider: '#7D53F6', badge: 'bg-background text-primary' },
                      MINOR: { border: 'border-primary', bg: 'bg-background', divider: '#7D53F6', badge: 'bg-background text-primary' },
                      ADDON: { border: 'border-primary', bg: 'bg-background', divider: '#7D53F6', badge: 'bg-background text-primary' },
                    };
                    const c = colors[optionalType] || colors.ADDON;
                    return (
                      <div key={slotName} className={`rounded-2xl border-2 overflow-hidden ${c.border} ${c.bg}`}>
                        <div className="px-4 py-3 border-b border-opacity-40 flex items-center justify-between"
                          style={{ borderColor: c.divider }}>
                          <div className="flex items-center gap-2">
                            <span className="text-sm">{cfg.icon}</span>
                            <span className="text-xs font-bold text-gray-700">{slotName}</span>
                          </div>
                          <span className={`text-xs font-bold px-2 py-0.5 rounded-full bg-gray-100 text-green-700`}>Enrolled</span>
                        </div>
                        {course ? (
                          <div className="px-4 py-4">
                            <p className="text-sm font-bold text-gray-900">{course.course_name}</p>
                            <p className="text-xs text-gray-500 mt-1">{course.course_code} · {course.credits} {course.credits === 1 ? 'credit' : 'credits'}</p>
                            {course.category && <p className="text-xs text-gray-400 mt-0.5">{course.category}</p>}
                          </div>
                        ) : (
                          <div className="px-4 py-4">
                            <p className="text-sm text-gray-400 italic">No course assigned to this slot yet.</p>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}


          </div>
        )}

        {/* ════════════════════════════════════════ */}
        {/* SECTION 3 — Add-On Courses (optional, student opts in) */}
        {/* ════════════════════════════════════════ */}
        {addonSlotEntries.length > 0 && (
          <div className="mb-8 card-custom p-6">
            <div className="border-t border-gray-200 pt-8 mb-4">
              <div className="flex items-center gap-2 mb-0.5">
                <h2 className="text-lg font-bold text-gray-900">Add-On Courses</h2>
                <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-gray-100 text-gray-500">Optional</span>
              </div>
              <p className="text-xs text-gray-400">Add-On courses are optional — enroll in any you'd like to take</p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {addonSlotEntries.map(([slotName, slotData]) => {
                const course = slotData.courses[0];
                const isEnrolled = !!selections[slotName];
                return (
                  <div key={slotName} className={`rounded-md border-2 overflow-hidden transition-all ${
                    isEnrolled ? 'border-teal-300 bg-teal-50' : 'border-gray-200 bg-white'
                  }`}>
                    <div className={`px-4 py-3 border-b flex items-center justify-between ${
                      isEnrolled ? 'border-teal-200' : 'border-gray-100'
                    }`}>
                      <div className="flex items-center gap-2">
                        <span className="text-sm">➕</span>
                        <span className="text-xs font-bold text-gray-700">{slotName}</span>
                      </div>
                      {isEnrolled
                        ? <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-background text-green-600">Enrolled</span>
                        : <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-background text-red-600">Not enrolled</span>
                      }
                    </div>
                    {course ? (
                      <div className="px-4 py-4">
                        <p className="text-sm font-bold text-gray-900">{course.course_name}</p>
                        <p className="text-xs text-gray-500 mt-1">{course.course_code} · {course.credits} {course.credits === 1 ? 'credit' : 'credits'}</p>
                        {course.category && <p className="text-xs text-gray-400 mt-0.5">{course.category}</p>}
                        {!isSubmitted && (
                          <button
                            onClick={() => handleSelection(slotName, isEnrolled ? null : course.hod_selection_id, course.credits)}
                            className={`mt-3 w-full py-2 rounded-xl text-xs font-bold transition-all ${
                              isEnrolled
                                ? 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                                : 'btn-primary-custom'
                            }`}
                          >
                            {isEnrolled ? 'Remove' : 'Enroll'}
                          </button>
                        )}
                      </div>
                    ) : (
                      <div className="px-4 py-4">
                        <p className="text-sm text-gray-400 italic">No course assigned yet.</p>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* ── Submit / Submitted footer ── */}
        <div className="border-t border-gray-200 pt-6 mt-4 pb-6">
          {isSubmitted ? (
            <div className="flex items-center justify-center gap-2 py-4 px-6 bg-emerald-50 border border-emerald-200 rounded-2xl text-emerald-700 font-semibold">
              <svg className="w-5 h-5 text-emerald-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 11V7a4 4 0 10-8 0v4" /><rect x="3" y="11" width="14" height="10" rx="2" ry="2"/></svg>
              Selections submitted and locked
            </div>
          ) : (
            <div className="flex flex-col items-stretch gap-3">
              <button
                onClick={handleSubmit}
                disabled={!canSubmit || isSubmitting}
                className={`w-full py-4 rounded-2xl font-bold text-base tracking-wide transition-all shadow-sm ${
                  !canSubmit || isSubmitting
                    ? 'bg-gray-200 cursor-not-allowed text-gray-400 shadow-none'
                    : 'bg-primary text-white active:scale-[0.98] shadow-md'
                }`}
              >
                {isSubmitting
                  ? 'Submitting…'
                  : filledRequired.length < requiredSlots.length
                    ? `Select ${requiredSlots.length - filledRequired.length} more required course${requiredSlots.length - filledRequired.length !== 1 ? 's' : ''} to continue`
                    : 'Submit Selections'}
              </button>
              <div className="flex items-center justify-center gap-4 text-xs text-gray-400">
                <span className={filledRequired.length === requiredSlots.length ? 'text-green-600 font-semibold' : ''}>
                  {filledRequired.length}/{requiredSlots.length} required
                </span>
                {optionalType && (
                  <><span>·</span><span className="text-green-600 font-semibold">All {optionalType === 'HONOR' ? 'Honour' : 'Minor'} slots enrolled</span></>
                )}
              </div>
            </div>
          )}
        </div>

      </div>
    </MainLayout>
  );
};

export default ElectiveSelectionPage;
