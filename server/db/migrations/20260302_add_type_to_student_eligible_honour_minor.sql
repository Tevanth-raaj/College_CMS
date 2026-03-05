-- Migration to add type column to student_eligible_honour_minor table
-- This allows distinguishing between honour and minor eligibility
-- Created: 2026-03-02

-- Add type column if it doesn't exist
ALTER TABLE student_eligible_honour_minor 
ADD COLUMN IF NOT EXISTS type VARCHAR(20) DEFAULT 'HONOUR' 
COMMENT 'Type of eligibility: HONOUR, MINOR, or both' AFTER student_email;

-- Create a unique constraint on (student_email, type) to prevent duplicates
-- First drop existing unique constraint if it exists
ALTER TABLE student_eligible_honour_minor 
DROP INDEX IF EXISTS student_email;

-- Add new unique constraint
ALTER TABLE student_eligible_honour_minor 
ADD UNIQUE KEY `idx_student_email_type` (`student_email`, `type`);

-- Create an index on type for faster filtering
ALTER TABLE student_eligible_honour_minor 
ADD INDEX IF NOT EXISTS `idx_type` (`type`);

-- Optional: Create an index on (type, student_email) for common queries
ALTER TABLE student_eligible_honour_minor 
ADD INDEX IF NOT EXISTS `idx_type_email` (`type`, `student_email`);
