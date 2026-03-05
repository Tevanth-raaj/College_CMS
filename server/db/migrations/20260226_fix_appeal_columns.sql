-- Fix migration to ensure course_category and new_count are removed from teacher_course_appeal
-- This must run to fix existing tables

-- Drop course_category if it exists
ALTER TABLE teacher_course_appeal DROP COLUMN IF EXISTS course_category;

-- Drop new_count if it exists  
ALTER TABLE teacher_course_appeal DROP COLUMN IF EXISTS new_count;
