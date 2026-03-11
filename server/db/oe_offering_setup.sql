-- Open Elective Offering System Setup
-- Run this SQL to create the hod_oe_offerings table
-- This allows HODs to offer their Open Elective courses to other departments

CREATE TABLE IF NOT EXISTS hod_oe_offerings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    department_id INT NOT NULL,
    curriculum_id INT NOT NULL,
    oe_card_id INT NOT NULL COMMENT 'References normal_cards.id where card_type = open_elective',
    semester INT NOT NULL,
    course_id INT NOT NULL,
    allowed_dept_ids JSON NOT NULL COMMENT 'Array of department IDs allowed to take this OE course',
    academic_year VARCHAR(20) NOT NULL,
    batch VARCHAR(20) DEFAULT NULL,
    approved_by_user_id INT NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_department (department_id),
    INDEX idx_curriculum (curriculum_id),
    INDEX idx_oe_card (oe_card_id),
    INDEX idx_semester (semester),
    INDEX idx_academic_year (academic_year),
    INDEX idx_status (status),
    
    UNIQUE KEY unique_oe_offering (department_id, curriculum_id, semester, course_id, academic_year, batch),
    
    FOREIGN KEY (department_id) REFERENCES departments(id),
    FOREIGN KEY (curriculum_id) REFERENCES curriculum(id),
    FOREIGN KEY (oe_card_id) REFERENCES normal_cards(id),
    FOREIGN KEY (course_id) REFERENCES courses(id),
    FOREIGN KEY (approved_by_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Verification
-- SHOW TABLES LIKE 'hod_oe_offerings';
-- DESCRIBE hod_oe_offerings;
