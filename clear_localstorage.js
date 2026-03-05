// Open browser console (F12) and run this to clear stale teacher data:

// Clear all teacher-related localStorage
localStorage.removeItem('teacher_id');
localStorage.removeItem('faculty_id');
localStorage.removeItem('teacher_name');
localStorage.removeItem('teacher_email');
localStorage.removeItem('teacher_dept');
localStorage.removeItem('teacher_designation');
localStorage.removeItem('theory_subject_count');
localStorage.removeItem('theory_with_lab_subject_count');

// Or clear everything and re-login:
localStorage.clear();

// Then reload the page
location.reload();
