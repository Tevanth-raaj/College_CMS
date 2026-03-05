-- Create course_type table
CREATE TABLE IF NOT EXISTS course_type (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE
);

-- Insert initial values
INSERT INTO course_type (id, name) VALUES 
(1, 'theory'), 
(2, 'lab'), 
(3, 'theory with lab') 
ON DUPLICATE KEY UPDATE name=VALUES(name);

-- Create teacher_course_limits table
CREATE TABLE IF NOT EXISTS teacher_course_limits (
    id INT PRIMARY KEY AUTO_INCREMENT,
    teacher_id INT NOT NULL,
    course_type_id INT NOT NULL,
    max_count INT DEFAULT 0,
    UNIQUE KEY (teacher_id, course_type_id),
    FOREIGN KEY (course_type_id) REFERENCES course_type(id)
);

-- Migrate existing data if possible (best effort) - only if old columns exist
-- Check if theory_subject_count column exists
SET @col_theory_exists := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teachers' 
    AND column_name = 'theory_subject_count'
);

SET @sql_migrate_theory := IF(@col_theory_exists > 0,
    'INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count) SELECT faculty_id, 1, theory_subject_count FROM teachers WHERE theory_subject_count > 0 ON DUPLICATE KEY UPDATE max_count = VALUES(max_count)',
    'SELECT "Column theory_subject_count does not exist, skipping data migration"'
);

PREPARE stmt FROM @sql_migrate_theory;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Check if theory_with_lab_subject_count column exists
SET @col_lab_exists := (
    SELECT COUNT(*) 
    FROM information_schema.columns 
    WHERE table_schema = DATABASE() 
    AND table_name = 'teachers' 
    AND column_name = 'theory_with_lab_subject_count'
);

SET @sql_migrate_lab := IF(@col_lab_exists > 0,
    'INSERT INTO teacher_course_limits (teacher_id, course_type_id, max_count) SELECT faculty_id, 3, theory_with_lab_subject_count FROM teachers WHERE theory_with_lab_subject_count > 0 ON DUPLICATE KEY UPDATE max_count = VALUES(max_count)',
    'SELECT "Column theory_with_lab_subject_count does not exist, skipping data migration"'
);

PREPARE stmt FROM @sql_migrate_lab;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
