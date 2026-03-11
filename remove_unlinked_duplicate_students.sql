-- Preview only: find duplicate students and show whether they are linked.
-- Duplicate key used: enrollment_no (non-empty).
-- Keep row: smallest id in each duplicate group.

START TRANSACTION;

DROP TEMPORARY TABLE IF EXISTS tmp_duplicate_students;
CREATE TEMPORARY TABLE tmp_duplicate_students (
  student_id INT PRIMARY KEY,
  keep_id INT NOT NULL,
  enrollment_no VARCHAR(50) NOT NULL,
  student_name VARCHAR(100) NULL,
  has_student_courses TINYINT(1) NOT NULL,
  has_csta TINYINT(1) NOT NULL
);

INSERT INTO tmp_duplicate_students (
  student_id,
  keep_id,
  enrollment_no,
  student_name,
  has_student_courses,
  has_csta
)
SELECT
  s.id AS student_id,
  d.keep_id,
  s.enrollment_no,
  s.student_name,
  EXISTS(
    SELECT 1
    FROM student_courses sc
    WHERE sc.student_id = s.id
  ) AS has_student_courses,
  EXISTS(
    SELECT 1
    FROM course_student_teacher_allocation csta
    WHERE csta.student_id = s.id
  ) AS has_csta
FROM students s
JOIN (
  SELECT enrollment_no, MIN(id) AS keep_id
  FROM students
  WHERE enrollment_no IS NOT NULL
    AND TRIM(enrollment_no) <> ''
  GROUP BY enrollment_no
  HAVING COUNT(*) > 1
) d ON d.enrollment_no = s.enrollment_no
WHERE s.id <> d.keep_id;

-- 1) See all duplicates and whether they are linked in either table.
SELECT
  enrollment_no,
  student_id,
  keep_id,
  student_name,
  has_student_courses,
  has_csta,
  CASE
    WHEN has_student_courses = 1 OR has_csta = 1 THEN 'KEEP (linked)'
    ELSE 'DELETE (unlinked)'
  END AS action
FROM tmp_duplicate_students
ORDER BY enrollment_no, student_id;

-- 2) Summary counts.
SELECT
  COUNT(*) AS total_duplicate_rows,
  SUM(CASE WHEN has_student_courses = 1 OR has_csta = 1 THEN 1 ELSE 0 END) AS linked_rows_to_keep,
  SUM(CASE WHEN has_student_courses = 0 AND has_csta = 0 THEN 1 ELSE 0 END) AS unlinked_rows_to_delete
FROM tmp_duplicate_students;

-- 3) Summary by enrollment number (people grouped).
SELECT
  enrollment_no,
  MIN(keep_id) AS keep_id,
  GROUP_CONCAT(student_id ORDER BY student_id) AS duplicate_student_ids,
  COUNT(*) AS duplicate_rows,
  SUM(has_student_courses) AS linked_in_student_courses,
  SUM(has_csta) AS linked_in_csta
FROM tmp_duplicate_students
GROUP BY enrollment_no
ORDER BY enrollment_no;

-- 4) Current duplicate groups in students table (no changes made).
SELECT
  enrollment_no,
  COUNT(*) AS rows_currently_present
FROM students
WHERE enrollment_no IS NOT NULL
  AND TRIM(enrollment_no) <> ''
GROUP BY enrollment_no
HAVING COUNT(*) > 1
ORDER BY enrollment_no;

DROP TEMPORARY TABLE IF EXISTS tmp_duplicate_students;

ROLLBACK;

-- This script is preview-only and does not modify data.
