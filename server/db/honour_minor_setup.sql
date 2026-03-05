-- Honour and Minor System Setup
-- Run this SQL to add honour slots and create minor selections table

-- ========================================
-- 1. Add Honour Slots to elective_semester_slots
-- ========================================
-- Add 2 honour slots for each semester (4-8)
-- These will be appended after existing slots with higher slot_order values

-- Get the max slot_order for each semester first, then insert
INSERT INTO elective_semester_slots (semester, slot_name, slot_order, is_active)
SELECT 4, 'Honour Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 4, 'Honour Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 5, 'Honour Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 5, 'Honour Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 6, 'Honour Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 6, 'Honour Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 7, 'Honour Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 7, 'Honour Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 8, 'Honour Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 8
UNION ALL
SELECT 8, 'Honour Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 8;

-- ========================================
-- 2. Create hod_minor_selections table
-- ========================================
CREATE TABLE IF NOT EXISTS hod_minor_selections (
    id INT AUTO_INCREMENT PRIMARY KEY,
    department_id INT NOT NULL,
    curriculum_id INT NOT NULL,
    vertical_id INT NOT NULL,
    semester INT NOT NULL,
    course_id INT NOT NULL,
    allowed_dept_ids JSON NOT NULL COMMENT 'Array of department IDs allowed to take this minor',
    academic_year VARCHAR(20) NOT NULL,
    batch VARCHAR(20) DEFAULT NULL,
    approved_by_user_id INT NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_department (department_id),
    INDEX idx_curriculum (curriculum_id),
    INDEX idx_vertical (vertical_id),
    INDEX idx_semester (semester),
    INDEX idx_academic_year (academic_year),
    INDEX idx_status (status),
    
    UNIQUE KEY unique_minor_assignment (department_id, curriculum_id, semester, course_id, academic_year, batch),
    
    FOREIGN KEY (department_id) REFERENCES departments(id),
    FOREIGN KEY (curriculum_id) REFERENCES curriculum(id),
    FOREIGN KEY (vertical_id) REFERENCES honour_verticals(id),
    FOREIGN KEY (course_id) REFERENCES courses(id),
    FOREIGN KEY (approved_by_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ========================================
-- Verification Queries
-- ========================================
-- Check honour slots were added
-- SELECT semester, slot_name, slot_order FROM elective_semester_slots WHERE slot_name LIKE 'Honour Slot%' ORDER BY semester, slot_order;

-- Check minor table exists
-- SHOW TABLES LIKE 'hod_minor_selections';
-- DESCRIBE hod_minor_selections;
