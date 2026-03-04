-- Complete setup for teacher 140

-- 1. Create the teacher record
INSERT INTO teachers (id, name, email, faculty_id, dept, designation, theory_subject_count, theory_with_lab_subject_count)
VALUES (
  140, 
  'Test Teacher', 
  'teacher140@example.com', 
  'AD140', 
  'AI&DS',  -- Your department
  'Assistant Professor',
  2,  -- theory subjects allowed
  1   -- theory with lab subjects allowed
)
ON DUPLICATE KEY UPDATE name='Test Teacher';

-- 2. Set up teacher course tracking (current semester type)
INSERT INTO teacher_course_tracking (teacher_id, current_semester_type, last_updated)
VALUES (140, 'even', NOW())
ON DUPLICATE KEY UPDATE current_semester_type='even', last_updated=NOW();

-- 3. Set up course limits per type (using faculty_id, not numeric id)
INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count)
VALUES 
  ('AD140', 1, 2),  -- theory: 2 courses
  ('AD140', 2, 1),  -- lab: 1 course
  ('AD140', 3, 1)   -- theory_with_lab: 1 course
ON DUPLICATE KEY UPDATE max_count=VALUES(max_count);

-- 4. Verify the setup
SELECT 'Teacher Record:' as info;
SELECT id, name, email, faculty_id, dept, designation FROM teachers WHERE id = 140;

SELECT 'Course Tracking:' as info;
SELECT * FROM teacher_course_tracking WHERE teacher_id = 140;

SELECT 'Course Limits:' as info;
SELECT tcl.*, ct.course_type 
FROM teacher_course_limits tcl
JOIN course_type ct ON tcl.course_type_id = ct.id
WHERE tcl.teacher_id = 140;
