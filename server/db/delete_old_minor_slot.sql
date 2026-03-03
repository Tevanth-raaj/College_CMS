-- Delete the old "Minor" slot (exact match, not like)
DELETE FROM elective_semester_slots 
WHERE slot_name = 'Minor';

-- Verify: Should show only Minor Slot 1 and Minor Slot 2
SELECT id, semester, slot_name 
FROM elective_semester_slots 
WHERE slot_name LIKE 'Minor%'
ORDER BY semester, slot_order;
