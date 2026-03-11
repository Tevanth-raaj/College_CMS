-- Merge/delete duplicate students for ONE enrollment number safely.
-- Set this before running:
SET @target_enrollment_no = '2024UAG1012';

START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS tmp_target_dupes;
CREATE TEMPORARY TABLE tmp_target_dupes (
  duplicate_id INT PRIMARY KEY,
  keep_id INT NOT NULL
);

INSERT INTO tmp_target_dupes (duplicate_id, keep_id)
SELECT
  s.id AS duplicate_id,
  k.keep_id
FROM students s
JOIN (
  SELECT enrollment_no, MIN(id) AS keep_id
  FROM students
  WHERE enrollment_no = @target_enrollment_no
  GROUP BY enrollment_no
  HAVING COUNT(*) > 1
) k ON k.enrollment_no = s.enrollment_no
WHERE s.id <> k.keep_id;

-- Preview target rows.
SELECT
  s.id,
  s.enrollment_no,
  s.student_name,
  s.email,
  s.register_no,
  CASE
    WHEN s.id = t.keep_id THEN 'KEEP'
    WHEN s.id = t.duplicate_id THEN 'DELETE_AFTER_MERGE'
    ELSE 'UNCHANGED'
  END AS action
FROM students s
LEFT JOIN tmp_target_dupes t
  ON s.id IN (t.duplicate_id, t.keep_id)
WHERE s.enrollment_no = @target_enrollment_no
ORDER BY s.id;

-- If there are no duplicates for the enrollment number, all operations below are no-op.

-- Resolve unique-key collisions before remap.
DELETE csta_dup
FROM course_student_teacher_allocation csta_dup
JOIN tmp_target_dupes t ON t.duplicate_id = csta_dup.student_id
JOIN course_student_teacher_allocation csta_keep
  ON csta_keep.student_id = t.keep_id
 AND csta_keep.course_id = csta_dup.course_id;

DELETE sc_dup
FROM student_courses sc_dup
JOIN tmp_target_dupes t ON t.duplicate_id = sc_dup.student_id
JOIN student_courses sc_keep
  ON sc_keep.student_id = t.keep_id
 AND sc_keep.course_id = sc_dup.course_id;

DELETE sm_dup
FROM student_marks sm_dup
JOIN tmp_target_dupes t ON t.duplicate_id = sm_dup.student_id
JOIN student_marks sm_keep
  ON sm_keep.student_id = t.keep_id
 AND sm_keep.course_id = sm_dup.course_id
 AND sm_keep.assessment_component_id = sm_dup.assessment_component_id;

DELETE stm_dup
FROM student_teacher_mapping stm_dup
JOIN tmp_target_dupes t ON t.duplicate_id = stm_dup.student_id
JOIN student_teacher_mapping stm_keep
  ON stm_keep.student_id = t.keep_id
 AND stm_keep.year = stm_dup.year
 AND stm_keep.academic_year = stm_dup.academic_year;

DELETE sec_dup
FROM student_elective_choices sec_dup
JOIN tmp_target_dupes t ON t.duplicate_id = sec_dup.student_id
JOIN student_elective_choices sec_keep
  ON sec_keep.student_id = t.keep_id
 AND sec_keep.hod_selection_id = sec_dup.hod_selection_id;

DELETE ea_dup
FROM exam_absentees ea_dup
JOIN tmp_target_dupes t ON t.duplicate_id = ea_dup.student_id
JOIN exam_absentees ea_keep
  ON ea_keep.student_id = t.keep_id
 AND ea_keep.window_id = ea_dup.window_id
 AND ea_keep.course_id = ea_dup.course_id
 AND ea_keep.mark_category_id = ea_dup.mark_category_id;

DELETE msp_dup
FROM mark_entry_student_permissions msp_dup
JOIN tmp_target_dupes t ON t.duplicate_id = msp_dup.student_id
JOIN mark_entry_student_permissions msp_keep
  ON msp_keep.student_id = t.keep_id
 AND msp_keep.window_id = msp_dup.window_id
 AND msp_keep.user_id = msp_dup.user_id;

-- Remap all student_id references in current schema (except students table itself).
DROP PROCEDURE IF EXISTS remap_target_student_refs;
DELIMITER $$
CREATE PROCEDURE remap_target_student_refs()
BEGIN
  DECLARE done INT DEFAULT 0;
  DECLARE v_table VARCHAR(128);

  DECLARE cur CURSOR FOR
    SELECT DISTINCT table_name
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND column_name = 'student_id'
      AND table_name NOT IN ('students', 'tmp_target_dupes');

  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

  OPEN cur;

  read_loop: LOOP
    FETCH cur INTO v_table;
    IF done = 1 THEN
      LEAVE read_loop;
    END IF;

    SET @sql_stmt = CONCAT(
      'UPDATE `', v_table, '` x ',
      'JOIN tmp_target_dupes t ON x.student_id = t.duplicate_id ',
      'SET x.student_id = t.keep_id'
    );

    PREPARE dyn_stmt FROM @sql_stmt;
    EXECUTE dyn_stmt;
    DEALLOCATE PREPARE dyn_stmt;
  END LOOP;

  CLOSE cur;
END$$
DELIMITER ;

CALL remap_target_student_refs();

-- Now remove only duplicate rows for that enrollment number.
DELETE s
FROM students s
JOIN tmp_target_dupes t ON t.duplicate_id = s.id;

-- Verification: should return exactly 1 row for target enrollment_no.
SELECT id, enrollment_no, student_name, email, register_no
FROM students
WHERE enrollment_no = @target_enrollment_no
ORDER BY id;

DROP PROCEDURE IF EXISTS remap_target_student_refs;
DROP TEMPORARY TABLE IF EXISTS tmp_target_dupes;

COMMIT;

-- For a dry-run, replace COMMIT with ROLLBACK.
