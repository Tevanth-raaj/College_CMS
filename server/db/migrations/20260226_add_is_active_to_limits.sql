-- Add is_active column to teacher_course_limits for tracking academic year allocation history
-- When is_active = 1, the allocation is current and can be overridden
-- When is_active = 0, the allocation is historical (from a previous academic year)

SET @col_exists := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND column_name = 'is_active'
);

-- Only add is_active column if it doesn't exist
SET @sql_add_column := IF(@col_exists = 0,
    'ALTER TABLE teacher_course_limits ADD COLUMN is_active TINYINT DEFAULT 1 AFTER max_count',
    'SELECT "Column is_active already exists"'
);

PREPARE stmt FROM @sql_add_column;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Add index on is_active if it doesn't exist
SET @idx_exists := (
    SELECT COUNT(*) 
    FROM information_schema.STATISTICS 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_limits' 
    AND index_name = 'idx_is_active'
);

SET @sql_add_index := IF(@idx_exists = 0,
    'ALTER TABLE teacher_course_limits ADD INDEX idx_is_active (is_active)',
    'SELECT "Index idx_is_active already exists"'
);

PREPARE stmt FROM @sql_add_index;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
