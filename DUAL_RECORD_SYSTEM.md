# Dual-Type Record System Explanation

## How Dual Records Work

### Database Structure
```sql
-- Same student can have TWO records in student_eligible_honour_minor
-- Record 1: (id=1, student_email='john@example.com', type='HONOUR')
-- Record 2: (id=2, student_email='john@example.com', type='MINOR')

-- Unique constraint allows this because constraint is on (student_email, type)
-- NOT just on student_email
UNIQUE KEY idx_student_email_type (student_email, type)
```

### Example Records for Same Student

| id | student_email | type | created_at | updated_at |
|----|---|---|---|---|
| 1 | john@example.com | HONOUR | 2026-03-02 10:00:00 | 2026-03-02 10:00:00 |
| 2 | john@example.com | MINOR | 2026-03-02 10:05:00 | 2026-03-02 10:05:00 |
| 3 | jane@example.com | HONOUR | 2026-03-02 10:10:00 | 2026-03-02 10:10:00 |

In this example:
- **John** has BOTH honour AND minor eligibility (2 records with same email)
- **Jane** has only HONOUR eligibility (1 record)

## Constraint Behavior

### What's ALLOWED ✅
```sql
-- Insert same email with different types
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'HONOUR');

INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'MINOR');
-- ✅ SUCCESS - Both records exist
```

### What's BLOCKED ❌
```sql
-- Try to insert same email + type combination twice
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'HONOUR');

INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'HONOUR');
-- ❌ DUPLICATE KEY ERROR - Violates unique constraint
```

## Eligibility Checking Logic

### Step 1: Query Both Types Separately
```go
// Check HONOUR eligibility
SELECT EXISTS(
    SELECT 1 FROM student_eligible_honour_minor 
    WHERE student_email = ? AND type = 'HONOUR'
)
// Returns: true/false

// Check MINOR eligibility
SELECT EXISTS(
    SELECT 1 FROM student_eligible_honour_minor 
    WHERE student_email = ? AND type = 'MINOR'
)
// Returns: true/false
```

### Step 2: Store in Map
```go
eligibilityMap["HONOUR"] = hasHonourEligibility  // true or false
eligibilityMap["MINOR"] = hasMinorEligibility      // true or false
```

### Step 3: Filter Slots Based on Type
```go
// For each slot returned from database
for _, slot := range slots {
    // If slot is HONOUR type, check HONOUR eligibility
    if slot.SlotType == "HONOR" && !eligibilityMap["HONOUR"] {
        continue // Skip this slot - not eligible
    }
    
    // If slot is MINOR type, check MINOR eligibility
    if slot.SlotType == "MINOR" && !eligibilityMap["MINOR"] {
        continue // Skip this slot - not eligible
    }
    
    // Otherwise include the slot
    filteredSlots = append(filteredSlots, slot)
}
```

## Import Handling

### Importing Honour for Student John
```bash
# Download honour template
# Upload CSV:
# student_email
# john@example.com

# Result: INSERT with type='HONOUR'
# DB State: (id=1, john@example.com, HONOUR, ...)
# Status: ✅ Record created
```

### Then Importing Minor for Student John
```bash
# Download minor template
# Upload CSV:
# student_email
# john@example.com

# Result: INSERT with type='MINOR'
# DB State: 
#   (id=1, john@example.com, HONOUR, ...)
#   (id=2, john@example.com, MINOR, ...)
# Status: ✅ Both records exist - student has dual eligibility
```

### Re-importing Same Type (Duplicate Handling)
```bash
# Import honour csv again with same student
# Upload CSV:
# student_email
# john@example.com

# Result: INSERT IGNORE with type='HONOUR'
# DB State: UNCHANGED (duplicate within same type is ignored)
# Status: ⚠️ Skipped (already exists)
```

## Eligibility Scenarios

### Scenario 1: Student with ONLY Honour
**DB Records:**
```
(email='alice@ex.com', type='HONOUR')
```

**When Selecting Electives:**
- `eligibilityMap["HONOUR"]` = true
- `eligibilityMap["MINOR"]` = false
- **Result:** Show HONOUR slots ✅, Hide MINOR slots ❌

### Scenario 2: Student with ONLY Minor
**DB Records:**
```
(email='bob@ex.com', type='MINOR')
```

**When Selecting Electives:**
- `eligibilityMap["HONOUR"]` = false
- `eligibilityMap["MINOR"]` = true
- **Result:** Hide HONOUR slots ❌, Show MINOR slots ✅

### Scenario 3: Student with BOTH Honour and Minor
**DB Records:**
```
(email='charlie@ex.com', type='HONOUR')
(email='charlie@ex.com', type='MINOR')
```

**When Selecting Electives:**
- `eligibilityMap["HONOUR"]` = true
- `eligibilityMap["MINOR"]` = true
- **Result:** Show HONOUR slots ✅, Show MINOR slots ✅

### Scenario 4: Student with NO Eligibility
**DB Records:**
```
(No records for this student)
```

**When Selecting Electives:**
- `eligibilityMap["HONOUR"]` = false
- `eligibilityMap["MINOR"]` = false
- **Result:** Hide HONOUR slots ❌, Hide MINOR slots ❌, Show ADDON/PROFESSIONAL slots ✅

## Insert Queries

### Import Handler - Using INSERT IGNORE
```sql
-- This is called from ImportHonourEligibility
INSERT IGNORE INTO student_eligible_honour_minor 
(student_email, type, created_at, updated_at)
VALUES ('john@example.com', 'HONOUR', NOW(), NOW())

-- If (john@example.com, 'HONOUR') already exists: SKIPPED
-- If (john@example.com, 'HONOUR') doesn't exist: INSERTED
-- If (john@example.com, 'MINOR') exists: NO CONFLICT (different type)
```

### Adding Both Manually
```sql
-- Step 1: Add to honour
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('john@example.com', 'HONOUR');

-- Step 2: Add to minor
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('john@example.com', 'MINOR');

-- Both succeed - no constraint violation
```

## Query Examples

### Get All Honour-Eligible Students
```sql
SELECT DISTINCT student_email 
FROM student_eligible_honour_minor 
WHERE type = 'HONOUR'
ORDER BY student_email;
```

### Get All Minor-Eligible Students
```sql
SELECT DISTINCT student_email 
FROM student_eligible_honour_minor 
WHERE type = 'MINOR'
ORDER BY student_email;
```

### Get Students with BOTH Honour AND Minor
```sql
SELECT student_email, GROUP_CONCAT(type) as eligibilities
FROM student_eligible_honour_minor 
GROUP BY student_email 
HAVING COUNT(*) = 2;
-- Returns students with exactly 2 records (one HONOUR, one MINOR)
```

### Get Students with ONLY Honour
```sql
SELECT DISTINCT student_email 
FROM student_eligible_honour_minor 
WHERE type = 'HONOUR'
AND student_email NOT IN (
    SELECT DISTINCT student_email 
    FROM student_eligible_honour_minor 
    WHERE type = 'MINOR'
);
```

### Get Students with ONLY Minor
```sql
SELECT DISTINCT student_email 
FROM student_eligible_honour_minor 
WHERE type = 'MINOR'
AND student_email NOT IN (
    SELECT DISTINCT student_email 
    FROM student_eligible_honour_minor 
    WHERE type = 'HONOUR'
);
```

## CSV Import Workflow Examples

### Scenario A: Add Honour to All CS Students
```csv
student_email
student1@cs.ac.in
student2@cs.ac.in
student3@cs.ac.in
```

**Upload to:** `/api/hod/honour-eligibility/import`
**Result:** 3 records created with type='HONOUR'

### Scenario B: Later, Add Minor to Some of Those Students
```csv
student_email
student1@cs.ac.in
student3@cs.ac.in
```

**Upload to:** `/api/hod/minor-eligibility/import`
**Result:** 
- student1: Now has BOTH honour + minor (2 records)
- student2: Still has only honour (1 record)
- student3: Now has BOTH honour + minor (2 records)

## Verification Queries

### Verify Constraint Works
```sql
-- This should succeed
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('test@example.com', 'HONOUR');

-- This should succeed (different type)
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('test@example.com', 'MINOR');

-- This should FAIL with duplicate key error
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('test@example.com', 'HONOUR');
```

### Check Student Eligibility Status
```sql
SELECT 
    student_email,
    MAX(CASE WHEN type = 'HONOUR' THEN 1 ELSE 0 END) as has_honour,
    MAX(CASE WHEN type = 'MINOR' THEN 1 ELSE 0 END) as has_minor,
    GROUP_CONCAT(type) as all_types,
    COUNT(*) as total_records
FROM student_eligible_honour_minor
GROUP BY student_email
ORDER BY student_email;
```

**Output Example:**
| student_email | has_honour | has_minor | all_types | total_records |
|---|---|---|---|---|
| alice@ex.com | 0 | 1 | MINOR | 1 |
| bob@ex.com | 1 | 0 | HONOUR | 1 |
| charlie@ex.com | 1 | 1 | HONOUR,MINOR | 2 |

## Summary

✅ **System Correctly Handles:**
- Same student with HONOUR only
- Same student with MINOR only
- Same student with BOTH HONOUR and MINOR
- Prevents duplicate (email, type) combinations
- Allows multiple types for same email
- Filters elective slots based on eligibility type
- Independently imports honour and minor data
