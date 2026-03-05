-- Add course allocation fields to teacher_course_history if missing

SET @col_course_id := (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'teacher_course_history'
    AND column_name = 'course_id'
);
SET @sql_course_id := IF(@col_course_id = 0,
  'ALTER TABLE teacher_course_history ADD COLUMN course_id INT DEFAULT NULL AFTER teacher_id',
  'SELECT "Column course_id already exists"'
);
PREPARE stmt FROM @sql_course_id;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_course_code := (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'teacher_course_history'
    AND column_name = 'course_code'
);
SET @sql_course_code := IF(@col_course_code = 0,
  'ALTER TABLE teacher_course_history ADD COLUMN course_code VARCHAR(20) DEFAULT NULL AFTER course_id',
  'SELECT "Column course_code already exists"'
);
PREPARE stmt FROM @sql_course_code;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_course_name := (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'teacher_course_history'
    AND column_name = 'course_name'
);
SET @sql_course_name := IF(@col_course_name = 0,
  'ALTER TABLE teacher_course_history ADD COLUMN course_name VARCHAR(255) DEFAULT NULL AFTER course_code',
  'SELECT "Column course_name already exists"'
);
PREPARE stmt FROM @sql_course_name;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_record_type := (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'teacher_course_history'
    AND column_name = 'record_type'
);
SET @sql_record_type := IF(@col_record_type = 0,
  'ALTER TABLE teacher_course_history ADD COLUMN record_type VARCHAR(20) DEFAULT NULL COMMENT "limit or course" AFTER academic_year',
  'SELECT "Column record_type already exists"'
);
PREPARE stmt FROM @sql_record_type;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_allocated_date := (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'teacher_course_history'
    AND column_name = 'allocated_date'
);
SET @sql_allocated_date := IF(@col_allocated_date = 0,
  'ALTER TABLE teacher_course_history ADD COLUMN allocated_date TIMESTAMP NULL COMMENT "When the cron assigned this course" AFTER record_type',
  'SELECT "Column allocated_date already exists"'
);
PREPARE stmt FROM @sql_allocated_date;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
