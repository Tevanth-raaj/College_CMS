-- FINAL: Verify the current state
-- Run these queries to check your elective_semester_slots table:

-- 1. Show all honour-related slots
SELECT id, semester, slot_name, slot_order FROM elective_semester_slots 
WHERE slot_name LIKE 'Honour%' 
ORDER BY semester, slot_order;

-- 2. Show all minor-related slots (should only be Minor Slot 1 and Minor Slot 2)
SELECT id, semester, slot_name, slot_order FROM elective_semester_slots 
WHERE slot_name LIKE 'Minor%' 
ORDER BY semester, slot_order;

-- 3. Show ALL slots grouped by type
SELECT slot_name, COUNT(*) as count, GROUP_CONCAT(semester) as semesters
FROM elective_semester_slots 
WHERE is_active = 1
GROUP BY slot_name
ORDER BY slot_name;

-- 4. Count summary
SELECT 
  COUNT(*) as total_slots,
  SUM(CASE WHEN slot_name LIKE 'Honour%' THEN 1 ELSE 0 END) as honour_slots,
  SUM(CASE WHEN slot_name LIKE 'Minor%' THEN 1 ELSE 0 END) as minor_slots,
  SUM(CASE WHEN slot_name LIKE 'Add On' THEN 1 ELSE 0 END) as addon_slots,
  SUM(CASE WHEN slot_name LIKE 'Professional%' THEN 1 ELSE 0 END) as professional_slots
FROM elective_semester_slots 
WHERE is_active = 1;
