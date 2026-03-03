-- Diagnostic queries to debug "Found 0 departments" issue for Semester 4, Year Level 2

-- Step 1: Check if normal_cards entry exists for Semester 4
SELECT 'Step 1: Check normal_cards' as diagnostic;
SELECT id, curriculum_id, semester_number, card_type, is_current
FROM normal_cards
WHERE semester_number = 4 AND card_type = 'semester'
LIMIT 5;

-- Step 2: Check academic_details for students in year level 2
SELECT 'Step 2: Check academic_details for year level 2' as diagnostic;
SELECT DISTINCT ad.student_id, ad.curriculum_id, ad.department, ad.year_level
FROM academic_details ad
WHERE ad.year_level = 2
LIMIT 10;

-- Step 3: Check student enrollments count for semester 4
SELECT 'Step 3: Check student enrollments for semester 4' as diagnostic;
SELECT 
    se.semester_id,
    nc.semester_number,
    COUNT(DISTINCT se.student_id) as student_count,
    COUNT(DISTINCT se.course_id) as course_count
FROM student_enrollments se
JOIN normal_cards nc ON se.semester_id = nc.id
WHERE nc.semester_number = 4 AND nc.card_type = 'semester'
GROUP BY se.semester_id, nc.semester_number;

-- Step 4: Check curriculum_courses for semester 4
SELECT 'Step 4: Check curriculum_courses for semester 4' as diagnostic;
SELECT 
    cc.id,
    cc.curriculum_id,
    cc.course_id,
    cc.semester_id,
    nc.semester_number,
    COUNT(*) as count
FROM curriculum_courses cc
JOIN normal_cards nc ON cc.semester_id = nc.id
WHERE nc.semester_number = 4 AND nc.card_type = 'semester'
GROUP BY cc.id, cc.curriculum_id, cc.course_id, cc.semester_id, nc.semester_number
LIMIT 10;

-- Step 5: Check the actual join chain for departments
SELECT 'Step 5: Full join chain - departments for semester 4' as diagnostic;
SELECT DISTINCT 
    ad.department,
    ad.year_level,
    COUNT(DISTINCT ad.student_id) as student_count
FROM academic_details ad
JOIN student_enrollments se ON ad.student_id = se.student_id
JOIN normal_cards nc ON se.semester_id = nc.id
WHERE nc.semester_number = 4 
    AND nc.card_type = 'semester'
    AND ad.year_level = 2
    AND ad.department IS NOT NULL 
    AND ad.department != ''
GROUP BY ad.department, ad.year_level;

-- Step 6: Check if there's a mismatch in curriculum_id references
SELECT 'Step 6: Check curriculum_id alignment' as diagnostic;
SELECT 
    'academic_details' as source,
    COUNT(DISTINCT curriculum_id) as distinct_curriculum_ids
FROM academic_details
WHERE year_level = 2
UNION ALL
SELECT 
    'normal_cards' as source,
    COUNT(DISTINCT curriculum_id)
FROM normal_cards
WHERE semester_number = 4;

-- Step 7: Check academic_calendar verification
SELECT 'Step 7: Verify academic_calendar entry' as diagnostic;
SELECT 
    id, academic_year, year_level, current_semester,
    teacher_course_selection_start, teacher_course_selection_end,
    allocation_run_at
FROM academic_calendar
WHERE academic_year = '2025-2026' AND current_semester = 4 AND year_level = 2;
