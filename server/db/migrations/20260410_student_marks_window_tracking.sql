ALTER TABLE student_marks
	ADD COLUMN IF NOT EXISTS window_id INT NOT NULL DEFAULT 0 AFTER assessment_component_id;

UPDATE student_marks
SET window_id = 0
WHERE window_id IS NULL;

DELETE sm_old
FROM student_marks sm_old
INNER JOIN student_marks sm_new
	ON sm_old.student_id = sm_new.student_id
	AND sm_old.course_id = sm_new.course_id
	AND sm_old.assessment_component_id = sm_new.assessment_component_id
	AND sm_old.id < sm_new.id;

ALTER TABLE student_marks DROP INDEX uq_student_course_component;

ALTER TABLE student_marks
	ADD UNIQUE INDEX uq_student_course_component (student_id, course_id, assessment_component_id);

ALTER TABLE student_marks
	ADD INDEX idx_student_marks_window (window_id);
