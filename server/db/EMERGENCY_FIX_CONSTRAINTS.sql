-- Emergency Fix: Reset student_eligible_honour_minor constraints
-- Use this if minors are not importing correctly

-- Show current indexes/constraints
SHOW INDEXES FROM student_eligible_honour_minor;

-- Drop ALL old unique constraints on student_email (the problematic one)
ALTER TABLE student_eligible_honour_minor 
DROP INDEX IF EXISTS student_email;

-- Create the correct unique constraint on (student_email, type)
ALTER TABLE student_eligible_honour_minor 
ADD UNIQUE KEY IF NOT EXISTS `idx_student_email_type` (`student_email`, `type`);

-- Verify the fix
SHOW INDEXES FROM student_eligible_honour_minor;

-- Test: Try inserting same email with different types
-- DELETE FROM student_eligible_honour_minor WHERE student_email = 'test@example.com';

-- Insert HONOUR
-- INSERT INTO student_eligible_honour_minor (student_email, type) VALUES ('test@example.com', 'HONOUR');
-- Result: Should succeed ✅

-- Insert MINOR for same email
-- INSERT INTO student_eligible_honour_minor (student_email, type) VALUES ('test@example.com', 'MINOR');
-- Result: Should succeed ✅ (different type, so allowed)

-- Check result
-- SELECT * FROM student_eligible_honour_minor WHERE student_email = 'test@example.com';
-- Should show 2 rows now
