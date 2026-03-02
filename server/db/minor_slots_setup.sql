-- Minor Slots Setup
-- Adds "Minor" slot to each semester that has Honour slots
-- Minor slots allow HODs to assign courses from other departments' PE selections

-- ========================================
-- 1. Add "Minor" Slots to elective_semester_slots
-- ========================================
-- Add 1 "Minor" slot for each semester that has Honour slots (matching honour_minor_setup.sql semesters)

INSERT INTO elective_semester_slots (semester, slot_name, slot_order, is_active)
SELECT 4, 'Minor', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 4
UNION ALL
SELECT 5, 'Minor', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 5
UNION ALL
SELECT 6, 'Minor', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 6
UNION ALL
SELECT 7, 'Minor', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 7
UNION ALL
SELECT 8, 'Minor', COALESCE(MAX(slot_order), 0) + 1, 1 FROM elective_semester_slots WHERE semester = 8;

-- ========================================
-- Verification Queries (uncomment to test)
-- ========================================
-- SELECT semester, slot_name, slot_order FROM elective_semester_slots WHERE slot_name = 'Minor' ORDER BY semester;
-- SELECT COUNT(*) as total_minor_slots FROM elective_semester_slots WHERE slot_name = 'Minor';
