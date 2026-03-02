-- Migration: Create student_enrollments table
-- Purpose: Track student enrollment history for allocation calculations
-- Date: 2026-02-24

-- Create the student_enrollments table
CREATE TABLE IF NOT EXISTS `student_enrollments` (
  `id` int NOT NULL AUTO_INCREMENT,
  `student_id` int NOT NULL,
  `course_id` int NOT NULL,
  `academic_year` varchar(20) NOT NULL COMMENT 'e.g., "2025-2026"',
  `semester` int NOT NULL COMMENT 'Semester number 1-8',
  `enrollment_status` varchar(50) DEFAULT 'enrolled' COMMENT 'enrolled, dropped, completed',
  `created_at` timestamp DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_student_id` (`student_id`),
  KEY `idx_course_id` (`course_id`),
  KEY `idx_academic_year_semester` (`academic_year`, `semester`),
  KEY `idx_student_course_year_sem` (`student_id`, `course_id`, `academic_year`, `semester`),
  CONSTRAINT `fk_se_student` FOREIGN KEY (`student_id`) REFERENCES `students` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_se_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Populate student_enrollments based on existing curriculum structure
-- This generates enrollments for all students in their respective curriculums and semesters
INSERT INTO `student_enrollments` (student_id, course_id, academic_year, semester, enrollment_status)
SELECT DISTINCT
    ad.student_id,
    cc.course_id,
    '2025-2026' as academic_year,
    nc.semester_number as semester,
    'enrolled' as enrollment_status
FROM academic_details ad
JOIN curriculum c ON ad.curriculum_id = c.id
JOIN normal_cards nc ON nc.curriculum_id = c.id
JOIN curriculum_courses cc ON cc.curriculum_id = c.id AND cc.semester_id = nc.id
WHERE ad.student_id IS NOT NULL
  AND nc.semester_number IS NOT NULL
  AND nc.card_type = 'semester'
  AND cc.course_id IS NOT NULL
ON DUPLICATE KEY UPDATE
    updated_at = CURRENT_TIMESTAMP;

-- Alternative query if the above doesn't work (simpler version):
-- This creates enrollments based on all courses in normal_cards
-- INSERT INTO `student_enrollments` (student_id, course_id, academic_year, semester, enrollment_status)
-- SELECT DISTINCT
--     ad.student_id,
--     cc.course_id,
--     '2025-2026' as academic_year,
--     nc.semester_number as semester,
--     'enrolled' as enrollment_status
-- FROM academic_details ad
-- JOIN curriculum c ON ad.curriculum_id = c.id
-- JOIN curriculum_courses cc ON cc.curriculum_id = c.id
-- JOIN normal_cards nc ON cc.semester_id = nc.id
-- WHERE ad.student_id IS NOT NULL
--   AND nc.semester_number IS NOT NULL
--   AND ad.batch != ''
-- ON DUPLICATE KEY UPDATE
--     updated_at = CURRENT_TIMESTAMP;
