-- MySQL Commands to clean up old "Minor" slot

-- Step 1: Find which "Minor" slots exist
SELECT id, semester, slot_name 
FROM elective_semester_slots 
WHERE slot_name LIKE 'Minor%'
ORDER BY semester, slot_order;

-- Step 2: Delete assignments to old "Minor" slot first
DELETE FROM hod_elective_selections 
WHERE slot_id IN (
  SELECT id FROM elective_semester_slots 
  WHERE slot_name = 'Minor' 
  AND slot_name NOT LIKE 'Minor Slot%'
);

-- Step 3: Delete the old "Minor" slot
DELETE FROM elective_semester_slots 
WHERE slot_name = 'Minor' 
AND slot_name NOT LIKE 'Minor Slot%';

-- Step 4: Verify only "Minor Slot 1" and "Minor Slot 2" remain
SELECT id, semester, slot_name 
FROM elective_semester_slots 
WHERE slot_name LIKE 'Minor%'
ORDER BY semester, slot_order;
