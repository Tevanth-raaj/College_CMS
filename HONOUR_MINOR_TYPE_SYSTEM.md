# Honour/Minor Eligibility Management - Type-Based System

## Overview
The system now supports separate tracking of **Honour** and **Minor** eligibility using a `type` column in the `student_eligible_honour_minor` table. This allows students to be eligible for either honour, minor, or both programs independently.

## Database Schema

### Updated Table: `student_eligible_honour_minor`
```sql
CREATE TABLE student_eligible_honour_minor (
    id INT AUTO_INCREMENT PRIMARY KEY,
    student_email VARCHAR(255) NOT NULL,
    type VARCHAR(20) DEFAULT 'HONOUR' COMMENT 'Type of eligibility: HONOUR or MINOR',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_student_email_type (student_email, type),
    INDEX idx_type (type),
    INDEX idx_type_email (type, student_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### Key Changes:
- **`type` Column**: Can be 'HONOUR' or 'MINOR'
- **Unique Constraint**: `(student_email, type)` - a student can have at most one record per type
- **Indexes**: Optimized for filtering by type and fetching by email/type combination

## Frontend Management

### New Endpoints

#### Honour Management
- **Download Template**: `GET /api/hod/honour-eligibility/template`
- **Import Data**: `POST /api/hod/honour-eligibility/import` (form field: `file`)

#### Minor Management
- **Download Template**: `GET /api/hod/minor-eligibility/template`
- **Import Data**: `POST /api/hod/minor-eligibility/import` (form field: `file`)

#### Legacy Endpoint (still works)
- **Download Template**: `GET /api/hod/honour-minor-eligibility/template?type=HONOUR|MINOR`
- **Import Data**: `POST /api/hod/honour-minor-eligibility/import` (form field: `type=HONOUR|MINOR` and `file`)

### Frontend UI
The `HODHonourMinorEligibilityPage` now displays:
1. **Honour Eligibility Section** (Blue themed)
   - Download honour template
   - Import honour data

2. **Minor Eligibility Section** (Purple themed)
   - Download minor template
   - Import minor data

## CSV Import Format

Both honour and minor use the same CSV format:

```csv
student_email
student1@example.com
student2@example.com
student3@example.com
```

**Column Requirements:**
- Exactly one column: `student_email`
- Email addresses should be in lowercase
- One student per row

## Backend Eligibility Checking

### Updated Query Logic
The system now checks eligibility by type:

```go
// Check for HONOUR eligibility
SELECT EXISTS(
    SELECT 1 FROM student_eligible_honour_minor 
    WHERE student_email = ? AND type = 'HONOUR'
)

// Check for MINOR eligibility
SELECT EXISTS(
    SELECT 1 FROM student_eligible_honour_minor 
    WHERE student_email = ? AND type = 'MINOR'
)
```

### Student Elective Selection Flow
1. Student logs in and accesses elective selection
2. System fetches available slots via `GetAvailableElectives`
3. System checks both honour and minor eligibility separately
4. Honour slots shown only if type = 'HONOUR' is in table
5. Minor slots shown only if type = 'MINOR' is in table
6. Addon slots shown to all students

## Database Migration

### Running Migrations
Execute the migration file to add the type column:

```bash
mysql -u username -p database_name < server/db/migrations/20260302_add_type_to_student_eligible_honour_minor.sql
```

Or manually run:
```sql
ALTER TABLE student_eligible_honour_minor 
ADD COLUMN IF NOT EXISTS type VARCHAR(20) DEFAULT 'HONOUR' AFTER student_email;

ALTER TABLE student_eligible_honour_minor 
DROP INDEX IF EXISTS student_email;

ALTER TABLE student_eligible_honour_minor 
ADD UNIQUE KEY `idx_student_email_type` (`student_email`, `type`);

ALTER TABLE student_eligible_honour_minor 
ADD INDEX IF NOT EXISTS `idx_type` (`type`);

ALTER TABLE student_eligible_honour_minor 
ADD INDEX IF NOT EXISTS `idx_type_email` (`type`, `student_email`);
```

## Usage Examples

### Adding Students for Honour Only
```sql
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student1@example.com', 'HONOUR');
```

### Adding Students for Minor Only
```sql
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student1@example.com', 'MINOR');
```

### Adding Students for Both Honour and Minor
```sql
INSERT INTO student_eligible_honour_minor (student_email, type) VALUES 
('student1@example.com', 'HONOUR'),
('student1@example.com', 'MINOR');
```

### Checking Student Eligibility
```sql
-- Check if eligible for honour
SELECT * FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'HONOUR';

-- Check if eligible for minor
SELECT * FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'MINOR';

-- Get all eligible students for honour
SELECT COUNT(*) FROM student_eligible_honour_minor WHERE type = 'HONOUR';

-- Get all eligible students for minor
SELECT COUNT(*) FROM student_eligible_honour_minor WHERE type = 'MINOR';
```

### Removing Eligibility
```sql
-- Remove honour eligibility
DELETE FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'HONOUR';

-- Remove minor eligibility
DELETE FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'MINOR';

-- Remove all eligibility for a student
DELETE FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com';
```

## Related Tables

### `hod_minor_selections` Table
Used for HOD to configure minor program courses:
- `id`: Primary key
- `department_id`: Which department can take this minor
- `curriculum_id`: Curriculum offering the minor
- `vertical_id`: Which vertical/specialization
- `semester`: Which semester (5, 6, or 7)
- `course_id`: Course to be offered
- `allowed_dept_ids`: JSON array of departments allowed to take this course
- `academic_year`: Academic year
- `batch`: Student batch
- `status`: ACTIVE/INACTIVE

### `student_eligible_honour_minor` Table
Tracks student eligibility:
- Students with type='HONOUR' will see HONOUR slots
- Students with type='MINOR' will see MINOR slots  
- Students can have both records for both eligibilities
- Each (email, type) combination must be unique

## Testing Checklist

- [ ] Migration runs successfully
- [ ] New type column exists with default 'HONOUR'
- [ ] Honour import works and creates type='HONOUR' records
- [ ] Minor import works and creates type='MINOR' records
- [ ] Honour-eligible student can see HONOUR slots
- [ ] Non-honour student cannot see HONOUR slots
- [ ] Minor-eligible student can see MINOR slots
- [ ] Non-minor student cannot see MINOR slots
- [ ] Student with both eligibilities sees both slot types
- [ ] Old data is imported as type='HONOUR' (due to default)

## Troubleshooting

### Students can't see Honour slots
```sql
-- Check if they have honour eligibility
SELECT * FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'HONOUR';

-- If missing, add them:
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'HONOUR');
```

### Students can't see Minor slots
```sql
-- Check if they have minor eligibility
SELECT * FROM student_eligible_honour_minor 
WHERE student_email = 'student@example.com' AND type = 'MINOR';

-- If missing, add them:
INSERT INTO student_eligible_honour_minor (student_email, type) 
VALUES ('student@example.com', 'MINOR');
```

### Duplicate Email Error
This can happen if you try to add the same (email, type) combination twice:
```sql
-- Check for duplicates
SELECT student_email, type, COUNT(*) 
FROM student_eligible_honour_minor 
GROUP BY student_email, type 
HAVING COUNT(*) > 1;

-- Remove duplicates
DELETE FROM student_eligible_honour_minor 
WHERE id NOT IN (
    SELECT MIN(id) FROM student_eligible_honour_minor 
    GROUP BY student_email, type
);
```

## Files Modified

### Backend
- `server/db/db.go` - Updated table creation schema
- `server/handlers/curriculum/honour_minor_eligibility_import.go` - Split import/export functions
- `server/handlers/student-teacher_entry/electives.go` - Updated eligibility check logic
- `server/routes/routes.go` - Added new endpoint routes

### Frontend
- `client/src/pages/curriculum/HODHonourMinorEligibilityPage.js` - Added separate honour/minor UI

### Database
- `server/db/migrations/20260302_add_type_to_student_eligible_honour_minor.sql` - Migration file

## Future Enhancements

1. **Bulk Operations**: Export currently eligible students for honour/minor
2. **Automatic Assignment**: Assign minor based on criteria (GPA, department, etc.)
3. **Statistics Dashboard**: Show counts of honour/minor eligible students
4. **Audit Trail**: Track who added/removed eligibility and when
