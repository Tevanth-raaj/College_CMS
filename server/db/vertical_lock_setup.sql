-- ============================================================
-- Vertical Lock System for Honour/Minor Slots
-- ============================================================
-- This ensures that once an HOD picks a vertical for honour/minor
-- in a semester (e.g., "Fullstack" for sem 5), the same vertical
-- is enforced for the same batch across all future semesters.
-- ============================================================

-- 1. Add batch column to academic_calendar so each semester maps to a student batch
ALTER TABLE academic_calendar
ADD COLUMN batch VARCHAR(20) DEFAULT NULL COMMENT 'Student batch e.g., "2024-2028"'
AFTER allocation_run_at;

-- Example: Update existing rows with batch info
-- Year level 1 (sem 1-2) → newest batch, Year level 4 (sem 7-8) → oldest batch
-- UPDATE academic_calendar SET batch = '2025-2029' WHERE year_level = 1 AND academic_year = '2025-2026';
-- UPDATE academic_calendar SET batch = '2024-2028' WHERE year_level = 2 AND academic_year = '2025-2026';
-- UPDATE academic_calendar SET batch = '2023-2027' WHERE year_level = 3 AND academic_year = '2025-2026';
-- UPDATE academic_calendar SET batch = '2022-2026' WHERE year_level = 4 AND academic_year = '2025-2026';

-- 2. Create vertical locks table
CREATE TABLE IF NOT EXISTS honour_minor_vertical_locks (
    id INT NOT NULL AUTO_INCREMENT,
    department_id INT NOT NULL,
    batch VARCHAR(20) NOT NULL COMMENT 'Student batch e.g., "2024-2028"',
    lock_type ENUM('honour', 'minor') NOT NULL COMMENT 'Whether this lock is for honour or minor slots',
    vertical_id INT NOT NULL COMMENT 'FK to normal_cards(id) — the locked vertical',
    vertical_name VARCHAR(255) NOT NULL COMMENT 'Cached vertical name for display',
    locked_by_user_id INT NOT NULL COMMENT 'HOD who first locked this vertical',
    first_semester INT NOT NULL COMMENT 'The semester where the lock was first established',
    first_academic_year VARCHAR(20) NOT NULL COMMENT 'Academic year when lock was created',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY unique_dept_batch_type (department_id, batch, lock_type),
    KEY idx_department (department_id),
    KEY idx_batch (batch),
    KEY idx_vertical (vertical_id),
    CONSTRAINT fk_vlock_dept FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE CASCADE,
    CONSTRAINT fk_vlock_vertical FOREIGN KEY (vertical_id) REFERENCES normal_cards(id) ON DELETE CASCADE,
    CONSTRAINT fk_vlock_user FOREIGN KEY (locked_by_user_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
