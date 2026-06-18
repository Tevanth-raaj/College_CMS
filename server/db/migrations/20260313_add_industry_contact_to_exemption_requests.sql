ALTER TABLE student_elective_exemption_requests
ADD COLUMN industry_contact VARCHAR(255) DEFAULT NULL AFTER industry_name;
