-- Add On Courses Setup
-- Adds "Add On" slot to each semester and enables HOD to offer department-specific courses

-- ========================================
-- 1. Add "Add On" Slots to elective_semester_slots
-- ========================================
-- Add 1 "Add On" slot for each semester (4-8)
-- These will be appended after existing slots with highest slot_order

INSERT INTO elective_semester_slots (semester, slot_name, slot_order, is_active)
SELECT 4, 'Add On', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 5, 'Add On', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 6, 'Add On', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 7, 'Add On', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 8, 'Add On', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 8;

-- ========================================
-- Verification Queries (uncomment to test)
-- ========================================
-- SELECT semester, slot_name, slot_order FROM elective_semester_slots WHERE slot_name = 'Add On' ORDER BY semester;
-- SELECT COUNT(*) as total_add_on_slots FROM elective_semester_slots WHERE slot_name = 'Add On';
