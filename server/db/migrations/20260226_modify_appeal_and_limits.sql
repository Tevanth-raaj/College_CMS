-- Migration to modify teacher_course_appeal and teacher_course_limits tables
-- Date: 2026-02-26
-- Purpose: Simplify appeal system and add is_active to limits table

-- Step 1: Modify teacher_course_appeal table
-- Drop course_category, new_count columns
-- Keep only: faculty_id, appeal_message, hr_message, appeal_status, hr_action

-- Check if course_category exists before dropping
SET @exist_course_category := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_appeal' 
    AND column_name = 'course_category'
);

SET @sql_drop_category := IF(@exist_course_category > 0, 
    'ALTER TABLE teacher_course_appeal DROP COLUMN course_category', 
    'SELECT "Column course_category does not exist"'
);

PREPARE stmt FROM @sql_drop_category;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Check if new_count exists before dropping
SET @exist_new_count := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_appeal' 
    AND column_name = 'new_count'
);

SET @sql_drop_count := IF(@exist_new_count > 0, 
    'ALTER TABLE teacher_course_appeal DROP COLUMN new_count', 
    'SELECT "Column new_count does not exist"'
);

PREPARE stmt FROM @sql_drop_count;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Ensure the essential columns exist with correct types
ALTER TABLE teacher_course_appeal 
  MODIFY COLUMN faculty_id BIGINT UNSIGNED NOT NULL,
  MODIFY COLUMN appeal_message TEXT NOT NULL COMMENT 'Message from faculty explaining appeal',
  MODIFY COLUMN appeal_status TINYINT(1) DEFAULT 0 COMMENT '0 = pending, 1 = resolved',
  MODIFY COLUMN hr_action VARCHAR(50) DEFAULT NULL COMMENT 'APPROVED, REJECTED',
  MODIFY COLUMN hr_message TEXT DEFAULT NULL COMMENT 'HR response message';

-- Add index if it doesn't exist
SET @exist_idx_faculty_status := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_appeal' 
    AND index_name = 'idx_faculty_status'
);

SET @sql_add_idx_appeal := IF(@exist_idx_faculty_status = 0,
    'CREATE INDEX idx_faculty_status ON teacher_course_appeal(faculty_id, appeal_status)',
    'SELECT "Index idx_faculty_status already exists"'
);

PREPARE stmt FROM @sql_add_idx_appeal;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Modify teacher_course_limits table
-- Add is_active column to track if limits are active for current academic year

-- Check if is_active already exists
SET @exist_is_active := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'is_active'
);

SET @sql_add_active := IF(@exist_is_active = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN is_active TINYINT(1) DEFAULT 1 COMMENT "1 = active for current year, 0 = inactive"',
    'SELECT "Column is_active already exists"'
);

PREPARE stmt FROM @sql_add_active;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add academic_year column to track which year these limits are for
SET @exist_academic_year := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'academic_year'
);

SET @sql_add_year := IF(@exist_academic_year = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN academic_year VARCHAR(20) DEFAULT "2025-2026" COMMENT "Academic year for these limits"',
    'SELECT "Column academic_year already exists"'
);

PREPARE stmt FROM @sql_add_year;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add timestamp columns for tracking
SET @exist_created_at := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'created_at'
);

SET @sql_add_created := IF(@exist_created_at = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP',
    'SELECT "Column created_at already exists"'
);

PREPARE stmt FROM @sql_add_created;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @exist_updated_at := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'updated_at'
);

SET @sql_add_updated := IF(@exist_updated_at = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP',
    'SELECT "Column updated_at already exists"'
);

PREPARE stmt FROM @sql_add_updated;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Set all existing limits to active
UPDATE teacher_course_limits SET is_active = 1 WHERE is_active IS NULL;

-- Add index for is_active queries (only if column exists)
SET @exist_idx_teacher_active := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'idx_teacher_active'
);

SET @sql_drop_idx1 := IF(@exist_idx_teacher_active > 0,
    'ALTER TABLE teacher_course_limits DROP INDEX idx_teacher_active',
    'SELECT "Index idx_teacher_active does not exist"'
);

PREPARE stmt FROM @sql_drop_idx1;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Only create index if is_active column exists
SET @col_is_active_exists := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'is_active'
);

SET @sql_create_idx1 := IF(@col_is_active_exists > 0,
    'CREATE INDEX idx_teacher_active ON teacher_course_limits(teacher_id, is_active)',
    'SELECT "Column is_active does not exist, skipping index creation"'
);

PREPARE stmt FROM @sql_create_idx1;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add index for academic_year (only if columns exist)
SET @exist_idx_academic_year := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'idx_academic_year'
);

SET @sql_drop_idx2 := IF(@exist_idx_academic_year > 0,
    'ALTER TABLE teacher_course_limits DROP INDEX idx_academic_year',
    'SELECT "Index idx_academic_year does not exist"'
);

PREPARE stmt FROM @sql_drop_idx2;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Only create index if both columns exist
SET @col_academic_year_exists := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'academic_year'
);

SET @sql_create_idx2 := IF(@col_is_active_exists > 0 AND @col_academic_year_exists > 0,
    'CREATE INDEX idx_academic_year ON teacher_course_limits(academic_year, is_active)',
    'SELECT "Required columns do not exist, skipping index creation"'
);

PREPARE stmt FROM @sql_create_idx2;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Update unique constraint to include academic_year
-- First drop the old unique key if it exists
SET @exist_uk_teacher_id := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'teacher_id'
);

SET @sql_drop_uk1 := IF(@exist_uk_teacher_id > 0,
    'ALTER TABLE teacher_course_limits DROP INDEX teacher_id',
    'SELECT "Index teacher_id does not exist"'
);

PREPARE stmt FROM @sql_drop_uk1;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @exist_uk_teacher_id2 := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'teacher_id_2'
);

SET @sql_drop_uk2 := IF(@exist_uk_teacher_id2 > 0,
    'ALTER TABLE teacher_course_limits DROP INDEX teacher_id_2',
    'SELECT "Index teacher_id_2 does not exist"'
);

PREPARE stmt FROM @sql_drop_uk2;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add new unique constraint with academic_year (only if column exists)
SET @exist_uk_new := (
    SELECT COUNT(*) 
    FROM information_schema.statistics 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'unique_teacher_course_year'
);

-- Re-check if academic_year column exists before adding unique key
SET @col_academic_year_exists2 := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'academic_year'
);

SET @sql_add_uk := IF(@exist_uk_new = 0 AND @col_academic_year_exists2 > 0,
    'ALTER TABLE teacher_course_limits ADD UNIQUE KEY unique_teacher_course_year (teacher_id, course_type_id, academic_year)',
    'SELECT "Unique key already exists or academic_year column missing"'
);

PREPARE stmt FROM @sql_add_uk;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
