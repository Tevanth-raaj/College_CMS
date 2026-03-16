ALTER TABLE student_elective_exemption_requests
ADD COLUMN IF NOT EXISTS professional_elective_no INT DEFAULT NULL AFTER request_type;

ALTER TABLE student_elective_exemption_requests
DROP COLUMN IF EXISTS elective_semester_no;
