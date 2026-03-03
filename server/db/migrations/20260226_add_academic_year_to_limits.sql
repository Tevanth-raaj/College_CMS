-- Add academic_year column to teacher_course_limits to track which window period the allocation belongs to

-- Check if column exists before adding
SET @col_exists := (
    SELECT COUNT(*)
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
        AND table_name = 'teacher_course_limits'
        AND column_name = 'academic_year'
);

SET @sql_add_column := IF(@col_exists = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN academic_year VARCHAR(20) NULL COMMENT "Links allocation to window period (e.g., 2025-2026)"',
    'SELECT "Column academic_year already exists in teacher_course_limits" AS Info'
);

PREPARE stmt FROM @sql_add_column;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add index for performance if missing
SET @idx_exists := (
    SELECT COUNT(*)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
        AND table_name = 'teacher_course_limits'
        AND index_name = 'idx_academic_year'
);

SET @sql_add_index := IF(@idx_exists = 0,
    'ALTER TABLE teacher_course_limits ADD KEY idx_academic_year (academic_year)',
    'SELECT "Index idx_academic_year already exists" AS Info'
);

PREPARE stmt FROM @sql_add_index;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
