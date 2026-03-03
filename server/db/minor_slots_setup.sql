-- Minor Slots Setup
-- Adds "Minor Slot 1" and "Minor Slot 2" to each semester (similar to Honour Slots)
-- Minor slots allow HODs to assign 2 courses for minor programs

-- ========================================
-- 1. Add "Minor Slot 1" and "Minor Slot 2" to elective_semester_slots
-- ========================================
-- Add 2 minor slots for each semester (4-8)

INSERT INTO elective_semester_slots (semester, slot_name, slot_order, is_active)
SELECT 4, 'Minor Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 4, 'Minor Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 5, 'Minor Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 5, 'Minor Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 6, 'Minor Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 6, 'Minor Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 7, 'Minor Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 7, 'Minor Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 8, 'Minor Slot 1', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 8
UNION ALL
SELECT 8, 'Minor Slot 2', COALESCE(MAX(slot_order), 0) + 2, 1 FROM elective_semester_slots WHERE semester = 8;

-- ========================================
-- Verification Queries (uncomment to test)
-- ========================================
-- SELECT semester, slot_name, slot_order FROM elective_semester_slots WHERE slot_name LIKE 'Minor Slot%' ORDER BY semester, slot_order;
-- SELECT COUNT(*) as total_minor_slots FROM elective_semester_slots WHERE slot_name LIKE 'Minor Slot%';

