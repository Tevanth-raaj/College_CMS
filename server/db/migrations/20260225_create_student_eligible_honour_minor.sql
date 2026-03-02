-- Migration to create student_eligible_honour_minor table
-- This table tracks which students are eligible to see and select Honour/Minor courses
-- Created: 2026-02-25

CREATE TABLE IF NOT EXISTS student_eligible_honour_minor (
    id INT AUTO_INCREMENT PRIMARY KEY,
    student_email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_student_email (student_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert some sample data (optional - remove if not needed)
-- INSERT INTO student_eligible_honour_minor (student_email) VALUES 
-- ('student1@example.com'),
-- ('student2@example.com');
