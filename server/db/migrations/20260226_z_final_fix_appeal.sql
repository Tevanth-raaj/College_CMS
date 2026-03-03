-- Final fix to remove course_category and new_count columns
-- These columns must not exist for the simplified appeal system

-- Drop course_category if it exists
SET @col_course_category := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_appeal' 
    AND column_name = 'course_category'
);

SET @sql_drop_cat := IF(@col_course_category > 0, 
    'ALTER TABLE teacher_course_appeal DROP COLUMN course_category', 
    'SELECT "Column course_category does not exist"'
);

PREPARE stmt FROM @sql_drop_cat;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Drop new_count if it exists
SET @col_new_count := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teacher_course_appeal' 
    AND column_name = 'new_count'
);

SET @sql_drop_nc := IF(@col_new_count > 0, 
    'ALTER TABLE teacher_course_appeal DROP COLUMN new_count', 
    'SELECT "Column new_count does not exist"'
);

PREPARE stmt FROM @sql_drop_nc;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
