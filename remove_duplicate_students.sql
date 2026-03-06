START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS tmp_student_id_merge;
CREATE TEMPORARY TABLE tmp_student_id_merge (
  duplicate_id INT PRIMARY KEY,
  keep_id INT NOT NULL,
  enrollment_no VARCHAR(50) NOT NULL
);

INSERT INTO tmp_student_id_merge (duplicate_id, keep_id, enrollment_no)
SELECT
  s.id AS duplicate_id,
  d.keep_id,
  s.enrollment_no
FROM students s
JOIN (
  SELECT enrollment_no, MIN(id) AS keep_id
  FROM students
  WHERE enrollment_no IS NOT NULL AND TRIM(enrollment_no) <> ''
  GROUP BY enrollment_no
  HAVING COUNT(*) > 1
) d ON d.enrollment_no = s.enrollment_no
WHERE s.id <> d.keep_id;

SELECT
  enrollment_no,
  keep_id,
  GROUP_CONCAT(duplicate_id ORDER BY duplicate_id) AS duplicate_ids,
  COUNT(*) AS duplicate_count
FROM tmp_student_id_merge
GROUP BY enrollment_no, keep_id
ORDER BY enrollment_no;

DELETE csta_dup
FROM course_student_teacher_allocation csta_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = csta_dup.student_id
JOIN course_student_teacher_allocation csta_keep
  ON csta_keep.student_id = m.keep_id
 AND csta_keep.course_id = csta_dup.course_id;

DELETE sc_dup
FROM student_courses sc_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = sc_dup.student_id
JOIN student_courses sc_keep
  ON sc_keep.student_id = m.keep_id
 AND sc_keep.course_id = sc_dup.course_id;

DELETE sm_dup
FROM student_marks sm_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = sm_dup.student_id
JOIN student_marks sm_keep
  ON sm_keep.student_id = m.keep_id
 AND sm_keep.course_id = sm_dup.course_id
 AND sm_keep.assessment_component_id = sm_dup.assessment_component_id;

DELETE stm_dup
FROM student_teacher_mapping stm_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = stm_dup.student_id
JOIN student_teacher_mapping stm_keep
  ON stm_keep.student_id = m.keep_id
 AND stm_keep.year = stm_dup.year
 AND stm_keep.academic_year = stm_dup.academic_year;

DELETE sec_dup
FROM student_elective_choices sec_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = sec_dup.student_id
JOIN student_elective_choices sec_keep
  ON sec_keep.student_id = m.keep_id
 AND sec_keep.hod_selection_id = sec_dup.hod_selection_id;

DELETE ea_dup
FROM exam_absentees ea_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = ea_dup.student_id
JOIN exam_absentees ea_keep
  ON ea_keep.student_id = m.keep_id
 AND ea_keep.window_id = ea_dup.window_id
 AND ea_keep.course_id = ea_dup.course_id
 AND ea_keep.mark_category_id = ea_dup.mark_category_id;

DELETE msp_dup
FROM mark_entry_student_permissions msp_dup
JOIN tmp_student_id_merge m ON m.duplicate_id = msp_dup.student_id
JOIN mark_entry_student_permissions msp_keep
  ON msp_keep.student_id = m.keep_id
 AND msp_keep.window_id = msp_dup.window_id
 AND msp_keep.user_id = msp_dup.user_id;

DROP PROCEDURE IF EXISTS remap_student_id_references;
DELIMITER $$
CREATE PROCEDURE remap_student_id_references()
BEGIN
  DECLARE done INT DEFAULT 0;
  DECLARE v_table VARCHAR(128);

  DECLARE cur CURSOR FOR
    SELECT DISTINCT table_name
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND column_name = 'student_id'
      AND table_name NOT IN ('students', 'tmp_student_id_merge');

  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = 1;

  OPEN cur;

  read_loop: LOOP
    FETCH cur INTO v_table;
    IF done = 1 THEN
      LEAVE read_loop;
    END IF;

    SET @sql_stmt = CONCAT(
      'UPDATE `', v_table, '` t ',
      'JOIN tmp_student_id_merge m ON t.student_id = m.duplicate_id ',
      'SET t.student_id = m.keep_id'
    );

    PREPARE dyn_stmt FROM @sql_stmt;
    EXECUTE dyn_stmt;
    DEALLOCATE PREPARE dyn_stmt;
  END LOOP;

  CLOSE cur;
END$$
DELIMITER ;

CALL remap_student_id_references();

DELETE s
FROM students s
JOIN tmp_student_id_merge m ON m.duplicate_id = s.id;

SELECT enrollment_no, COUNT(*) AS cnt
FROM students
WHERE enrollment_no IS NOT NULL AND TRIM(enrollment_no) <> ''
GROUP BY enrollment_no
HAVING COUNT(*) > 1
ORDER BY enrollment_no;

DROP PROCEDURE IF EXISTS remap_student_id_references;
DROP TEMPORARY TABLE IF EXISTS tmp_student_id_merge;

COMMIT;
