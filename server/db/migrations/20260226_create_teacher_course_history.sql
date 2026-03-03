-- Create teacher_course_history table to track allocation history
-- Maps teacher using alphanumeric teacher_id, stores snapshots of allocations

CREATE TABLE IF NOT EXISTS teacher_course_history (
    id INT PRIMARY KEY AUTO_INCREMENT,
    teacher_id VARCHAR(50) NOT NULL,
    course_type_id INT NOT NULL,
    max_count INT DEFAULT 0,
    allocated_count INT DEFAULT 0,
    window_start DATE NOT NULL,
    window_end DATE NOT NULL,
    semester_type VARCHAR(10) COMMENT 'ODD or EVEN',
    academic_year VARCHAR(20) COMMENT 'e.g., 2025-2026',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    archived_at TIMESTAMP NULL COMMENT 'When this allocation was archived (window closed)',
    
    FOREIGN KEY (course_type_id) REFERENCES course_type(id),
    KEY idx_teacher_id (teacher_id),
    KEY idx_window_dates (window_start, window_end),
    KEY idx_archived_at (archived_at),
    INDEX idx_teacher_course_window (teacher_id, course_type_id, window_start)
);
