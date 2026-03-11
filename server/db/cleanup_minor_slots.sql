-- Cleanup: Remove old "Minor" slots (singular) that are not "Minor Slot 1" or "Minor Slot 2"
-- Keep only: Minor Slot 1 and Minor Slot 2

DELETE FROM elective_semester_slots 
WHERE slot_name = 'Minor' 
AND slot_name NOT LIKE 'Minor Slot%';

-- Verify: Check what minor-related slots remain
-- SELECT id, semester, slot_name FROM elective_semester_slots 
-- WHERE slot_name LIKE 'Minor%' 
-- ORDER BY semester, slot_order;
