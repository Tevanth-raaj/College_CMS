-- Global student dedupe (all enrollment numbers)
-- Rule: keep oldest row (MIN(id)) per enrollment_no, remove newer duplicate rows and their connections.
-- IMPORTANT: This removes connections of newer IDs (data-loss by design).

START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS tmp_student_dupes;
CREATE TEMPORARY TABLE tmp_student_dupes (
  dup_id INT PRIMARY KEY,
  keep_id INT NOT NULL,
  enrollment_no VARCHAR(50) NOT NULL
);

INSERT INTO tmp_student_dupes (dup_id, keep_id, enrollment_no)
SELECT
  s.id AS dup_id,
  g.keep_id,
  s.enrollment_no
FROM students s
JOIN (
  SELECT enrollment_no, MIN(id) AS keep_id
  FROM students
  WHERE enrollment_no IS NOT NULL
    AND TRIM(enrollment_no) <> ''
  GROUP BY enrollment_no
  HAVING COUNT(*) > 1
) g ON g.enrollment_no = s.enrollment_no
WHERE s.id <> g.keep_id;

-- Preview duplicates that will be removed.
SELECT
  s.id,
  s.enrollment_no,
  s.student_name,
  s.email,
  s.register_no,
  d.keep_id,
  'DELETE_NEWER' AS action
FROM students s
JOIN tmp_student_dupes d ON d.dup_id = s.id
ORDER BY s.enrollment_no, s.id;

-- Summary.
SELECT
  COUNT(*) AS duplicate_student_rows_to_delete,
  COUNT(DISTINCT enrollment_no) AS duplicate_enrollment_groups
FROM tmp_student_dupes;

-- Delete all links for duplicate IDs without using stored procedures
-- (works for users without ALTER ROUTINE/CREATE ROUTINE privileges).

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'academic_details'),
  'DELETE t FROM academic_details t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'address'),
  'DELETE t FROM address t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'admission_payment'),
  'DELETE t FROM admission_payment t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'contact_details'),
  'DELETE t FROM contact_details t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'course_student_teacher_allocation'),
  'DELETE t FROM course_student_teacher_allocation t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'exam_absentees'),
  'DELETE t FROM exam_absentees t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'hostel_details'),
  'DELETE t FROM hostel_details t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'insurance_details'),
  'DELETE t FROM insurance_details t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'mark_entry_student_permissions'),
  'DELETE t FROM mark_entry_student_permissions t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'research_profiles'),
  'DELETE t FROM research_profiles t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'school_details'),
  'DELETE t FROM school_details t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'student_courses'),
  'DELETE t FROM student_courses t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'student_elective_choices'),
  'DELETE t FROM student_elective_choices t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'student_enrollments'),
  'DELETE t FROM student_enrollments t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'student_marks'),
  'DELETE t FROM student_marks t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'student_teacher_mapping'),
  'DELETE t FROM student_teacher_mapping t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

SET @sql_stmt = IF(
  EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'temp_import_table'),
  'DELETE t FROM temp_import_table t JOIN tmp_student_dupes d ON t.student_id = d.dup_id',
  'SELECT 1'
);
PREPARE dyn_stmt FROM @sql_stmt; EXECUTE dyn_stmt; DEALLOCATE PREPARE dyn_stmt;

-- Delete duplicate student rows themselves.
DELETE s
FROM students s
JOIN tmp_student_dupes d ON d.dup_id = s.id;

-- Verification: duplicates by enrollment_no should now be gone.
SELECT
  enrollment_no,
  COUNT(*) AS row_count
FROM students
WHERE enrollment_no IS NOT NULL
  AND TRIM(enrollment_no) <> ''
GROUP BY enrollment_no
HAVING COUNT(*) > 1
ORDER BY enrollment_no;

DROP TEMPORARY TABLE IF EXISTS tmp_student_dupes;

COMMIT;

-- For dry run: replace COMMIT with ROLLBACK.
