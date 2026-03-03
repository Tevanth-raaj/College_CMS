-- Reset allocation_run_at to NULL to allow re-testing
UPDATE academic_calendar 
SET allocation_run_at = NULL
WHERE teacher_course_selection_end IS NOT NULL 
AND NOW() > teacher_course_selection_end
AND is_current = 1;

-- Verify the update
SELECT id, academic_year, current_semester, teacher_course_selection_end, allocation_run_at
FROM academic_calendar
WHERE is_current = 1 
ORDER BY academic_year, current_semester;
