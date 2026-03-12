# Student-Teacher-Course Allocation System

## Overview

This system automatically allocates students to teachers for each course based on:
- Student enrollment in courses (`student_courses` table)
- Teacher allocation to courses (`teacher_course_allocation` table)  
- Student learning modes (UAL=1, PBL=2)
- Dynamic teacher distribution based on student ratios

**Key Constraint:** No teacher can have both UAL and PBL students for the same course.

---

## API Endpoints

### 1. Full Allocation (All Courses)
```
POST /api/allocations/students/run
```

Allocates all unallocated students across all courses with teachers.

**Response:**
```json
{
  "success": true,
  "total_courses": 25,
  "successful_courses": 23,
  "failed_courses": 2,
  "total_students_allocated": 1250,
  "execution_time_seconds": 3.45,
  "timestamp": "2026-03-11T10:30:00Z",
  "course_results": [
    {
      "course_id": 101,
      "course_name": "Mathematics",
      "success": true,
      "allocated_count": 120,
      "ual_count": 20,
      "pbl_count": 100,
      "teachers_used": 4
    },
    {
      "course_id": 102,
      "course_name": "Physics",
      "success": false,
      "error": "teacher_course_allocation is empty for this course. Please allocate teachers first.",
      "error_code": "NO_TEACHERS_ALLOCATED"
    }
  ]
}
```

### 2. Single Course Allocation
```
POST /api/allocations/students/course/{course_id}
```

Allocates students for a specific course.

**Example:**
```bash
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

### 3. View Allocations by Course
```
GET /api/allocations/students/course/{course_id}/view
```

View current student-teacher allocations for a course.

**Response:**
```json
{
  "success": true,
  "course_id": 101,
  "course_name": "Mathematics",
  "allocations": [
    {
      "teacher_name": "Dr. Smith",
      "teacher_id": "CS1001",
      "mode": "UAL",
      "mode_id": 1,
      "student_count": 20
    },
    {
      "teacher_name": "Dr. Johnson",
      "teacher_id": "CS1002",
      "mode": "PBL",
      "mode_id": 2,
      "student_count": 40
    }
  ]
}
```

---

## Error Codes & Resolutions

### 1. `NO_TEACHERS_ALLOCATED`
**Error:** `"teacher_course_allocation is empty for this course"`

**Resolution:**
1. Go to teacher allocation page
2. Add teachers to the course
3. Run allocation again

### 2. `INSUFFICIENT_TEACHERS`
**Error:** `"Course needs both UAL and PBL teachers but only 1 teacher(s) available"`

**Resolution:**
1. The course has both UAL and PBL students but insufficient teachers
2. Add at least one more teacher to the course in `teacher_course_allocation`
3. Re-run allocation

**Example Response:**
```json
{
  "success": false,
  "error": "Insufficient teachers. Course needs both UAL and PBL teachers but only 1 teacher(s) available. Please add at least 1 more teacher to this course.",
  "error_code": "INSUFFICIENT_TEACHERS",
  "ual_count": 25,
  "pbl_count": 30
}
```

### 3. `NULL_LEARNING_MODE`
**Error:** `"X students have NULL learning_mode_id"`

**Resolution:**
1. Update student records to set `learning_mode_id` to 1 (UAL) or 2 (PBL)
2. Re-run allocation

**Example Response:**
```json
{
  "success": false,
  "error": "5 students have NULL learning_mode_id. Please update student records.",
  "error_code": "NULL_LEARNING_MODE",
  "students_with_null_mode": [
    {
      "student_id": 3050,
      "student_name": "John Doe",
      "enrollment_no": "22CS001",
      "register_no": "21001",
      "defaulted_to_ual": false
    }
  ]
}
```

**Fix SQL:**
```sql
-- Set learning mode for specific students
UPDATE students 
SET learning_mode_id = 1  -- 1=UAL, 2=PBL
WHERE id IN (3050, 3051, 3052);
```

---

## Automatic Execution

The system runs automatically:
1. ✅ Teacher preference window closes
2. ✅ Teacher-to-course allocation runs ([auto_allocate.go](server/handlers/allocation/auto_allocate.go))
3. ✅ **Student-teacher-course allocation runs automatically** (NEW!)
4. ✅ Window marked as inactive

**Scheduler:** [allocation_scheduler.go](server/scheduler/allocation_scheduler.go)
- Checks every 1 minute for closed windows
- Runs both teacher and student allocations sequentially

---

## Algorithm Details

### Distribution Logic

#### Step 1: Check Prerequisites
- ✅ Students have valid `learning_mode_id` (1 or 2)
- ✅ Teachers exist in `teacher_course_allocation`
- ✅ Sufficient teachers for both modes if needed

#### Step 2: Analyze Current State
```go
// For each course:
unallocatedUAL = students with mode=1 not in course_student_teacher_allocation
unallocatedPBL = students with mode=2 not in course_student_teacher_allocation

// Identify teacher modes:
teacherModes = map each teacher to their current assigned mode (if any)
```

#### Step 3: Assign Free Teachers
```go
freeTeachers = teachers with no current allocations
totalUnallocated = unallocatedUAL + unallocatedPBL

ualRatio = unallocatedUAL / totalUnallocated
ualTeachersNeeded = ceil(ualRatio * len(freeTeachers))

// Ensure minimum 1 teacher per mode if students exist
if unallocatedUAL > 0 && ualTeachersNeeded == 0 {
    ualTeachersNeeded = 1
}

// Assign first N teachers to UAL, rest to PBL
```

#### Step 4: Round-Robin Distribution
```go
// Sort teachers by current load (ascending)
// Distribute students evenly using round-robin
for i, student := range students {
    teacherIndex = i % len(teachers)
    assignStudentToTeacher(student, teachers[teacherIndex])
}
```

### Example Scenarios

#### Scenario A: Fresh Allocation
```
Input:
- 4 teachers (T1, T2, T3, T4) - all free
- 100 PBL students
- 20 UAL students

Algorithm:
- Ratio: 20/120 = 16.7%
- UAL teachers: ceil(0.167 * 4) = 1
- PBL teachers: 3

Result:
T1 → 33 PBL students
T2 → 33 PBL students
T3 → 34 PBL students
T4 → 20 UAL students
```

#### Scenario B: Incremental PBL Addition
```
Before:
- T1 → 30 UAL students (locked to UAL)
- T2 → 25 UAL students (locked to UAL)

New:
- T3, T4 added
- 80 new PBL students

Algorithm:
- T1, T2 stay with UAL
- T3, T4 assigned to PBL
- Distribute 80 PBL students

Result:
T1 → 30 UAL (unchanged)
T2 → 25 UAL (unchanged)
T3 → 40 PBL (new)
T4 → 40 PBL (new)
```

#### Scenario C: Inactive Teacher Reallocation
```
Before:
- T1 → 40 PBL students (now inactive)
- T2, T3, T4 active

Process:
1. Detect T1 is inactive (status=0)
2. Mark T1's allocations as status=0
3. Reallocate 40 students to T2, T3, T4

Result:
T2 → 13 PBL students (reallocated from T1)
T3 → 13 PBL students (reallocated from T1)
T4 → 14 PBL students (reallocated from T1)
```

---

## Validation Queries

### 1. Verify No Mixed Modes per Teacher
```sql
-- Should return 0 rows
SELECT 
    csta.course_id,
    c.course_name,
    csta.teacher_id,
    t.name,
    GROUP_CONCAT(DISTINCT s.learning_mode_id ORDER BY s.learning_mode_id) AS modes,
    COUNT(*) as total_students
FROM course_student_teacher_allocation csta
INNER JOIN students s ON csta.student_id = s.id
INNER JOIN teachers t ON csta.teacher_id = t.faculty_id
INNER JOIN courses c ON csta.course_id = c.id
WHERE csta.status = 1
GROUP BY csta.course_id, csta.teacher_id
HAVING COUNT(DISTINCT s.learning_mode_id) > 1;
```

### 2. View Distribution Summary
```sql
SELECT 
    c.course_name,
    t.name AS teacher_name,
    CASE s.learning_mode_id 
        WHEN 1 THEN 'UAL'
        WHEN 2 THEN 'PBL'
        ELSE 'Unknown'
    END AS mode,
    COUNT(*) AS student_count
FROM course_student_teacher_allocation csta
INNER JOIN students s ON csta.student_id = s.id
INNER JOIN teachers t ON csta.teacher_id = t.faculty_id
INNER JOIN courses c ON csta.course_id = c.id
WHERE csta.status = 1
GROUP BY c.id, t.id, s.learning_mode_id
ORDER BY c.course_name, s.learning_mode_id, student_count;
```

### 3. Find Unallocated Students
```sql
SELECT 
    c.course_name,
    s.student_name,
    s.enrollment_no,
    CASE s.learning_mode_id 
        WHEN 1 THEN 'UAL'
        WHEN 2 THEN 'PBL'
        ELSE 'Unknown'
    END AS mode
FROM students s
INNER JOIN student_courses sc ON s.id = sc.student_id
INNER JOIN courses c ON sc.course_id = c.id
LEFT JOIN course_student_teacher_allocation csta 
    ON csta.student_id = s.id AND csta.course_id = sc.course_id AND csta.status = 1
WHERE csta.id IS NULL 
    AND s.status = 1
ORDER BY c.course_name, s.learning_mode_id;
```

### 4. Check for NULL Learning Modes
```sql
SELECT 
    s.id,
    s.student_name,
    s.enrollment_no,
    s.register_no,
    COUNT(sc.course_id) as enrolled_courses
FROM students s
LEFT JOIN student_courses sc ON s.id = sc.student_id
WHERE s.status = 1
    AND (s.learning_mode_id IS NULL OR s.learning_mode_id NOT IN (1, 2))
GROUP BY s.id
ORDER BY s.student_name;
```

### 5. Teacher Workload Report
```sql
SELECT 
    t.name AS teacher_name,
    t.faculty_id,
    COUNT(DISTINCT csta.course_id) as total_courses,
    SUM(CASE WHEN s.learning_mode_id = 1 THEN 1 ELSE 0 END) as ual_students,
    SUM(CASE WHEN s.learning_mode_id = 2 THEN 1 ELSE 0 END) as pbl_students,
    COUNT(*) as total_students
FROM teachers t
LEFT JOIN course_student_teacher_allocation csta ON t.faculty_id = csta.teacher_id AND csta.status = 1
LEFT JOIN students s ON csta.student_id = s.id
WHERE t.status = 1
GROUP BY t.id
ORDER BY total_students DESC;
```

---

## Database Schema

### course_student_teacher_allocation
```sql
CREATE TABLE `course_student_teacher_allocation` (
  `id` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `student_id` int NOT NULL,
  `course_id` int NOT NULL,
  `teacher_id` varchar(45) NOT NULL,
  `status` tinyint(1) DEFAULT 1,
  
  UNIQUE KEY `uq_student_course_assign` (`student_id`, `course_id`),
  FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
  FOREIGN KEY (`student_id`) REFERENCES `students` (`id`)
);
```

**Key Points:**
- `teacher_id` stores `faculty_id` (VARCHAR), not teacher table's `id`
- `status = 1` means active allocation
- `status = 0` means inactive (used for reallocation tracking)
- Unique constraint ensures one teacher per student per course

---

## Manual Operations

The allocation system normally runs automatically when teacher preference windows close. However, you can manually trigger allocations in several scenarios:

### When to Trigger Manually

✅ **After adding new students** to `students` and `student_courses` tables  
✅ **After adding new teachers** to `teacher_course_allocation` table  
✅ **When allocation window is closed** but you need to allocate outside schedule  
✅ **After fixing student learning modes** that were previously NULL  
✅ **When redistributing load** after teacher status changes  
✅ **For testing purposes** before production rollout

---

### 1. Allocate All Courses (Recommended)

**Use Case:** You've added multiple students/teachers across different courses and want to allocate everything at once.

**Command:**
```bash
curl -X POST \
  http://localhost:8080/api/allocations/students/run \
  -H "Content-Type: application/json"
```

**What Happens:**
1. System fetches all courses with teacher allocations
2. For each course, identifies unallocated students
3. Distributes students to teachers based on UAL/PBL ratios
4. Returns comprehensive summary report

**Expected Response (Success):**
```json
{
  "success": true,
  "total_courses": 25,
  "successful_courses": 23,
  "failed_courses": 2,
  "total_students_allocated": 1250,
  "execution_time_seconds": 3.45,
  "timestamp": "2026-03-11T10:30:00Z",
  "course_results": [
    {
      "course_id": 101,
      "course_name": "Data Structures",
      "success": true,
      "allocated_count": 120,
      "ual_count": 20,
      "pbl_count": 100,
      "teachers_used": 4,
      "execution_time_seconds": 0.15
    },
    {
      "course_id": 102,
      "course_name": "Operating Systems",
      "success": true,
      "allocated_count": 80,
      "ual_count": 10,
      "pbl_count": 70,
      "teachers_used": 3,
      "execution_time_seconds": 0.12
    }
  ]
}
```

**Expected Response (Partial Failure):**
```json
{
  "success": false,
  "total_courses": 25,
  "successful_courses": 23,
  "failed_courses": 2,
  "total_students_allocated": 1150,
  "execution_time_seconds": 3.21,
  "timestamp": "2026-03-11T10:30:00Z",
  "course_results": [
    {
      "course_id": 103,
      "course_name": "Database Management",
      "success": false,
      "error": "teacher_course_allocation is empty for this course. Please allocate teachers first.",
      "error_code": "NO_TEACHERS_ALLOCATED"
    },
    {
      "course_id": 104,
      "course_name": "Computer Networks",
      "success": false,
      "error": "5 students have NULL learning_mode_id. Please update student records.",
      "error_code": "NULL_LEARNING_MODE",
      "students_with_null_mode": [
        {
          "student_id": 3050,
          "student_name": "John Doe",
          "enrollment_no": "22CS001",
          "register_no": "21001"
        }
      ]
    }
  ]
}
```

**Performance Expectations:**
- **Small scale:** 10 courses, 500 students = ~1-2 seconds
- **Medium scale:** 50 courses, 2000 students = ~5-10 seconds
- **Large scale:** 100 courses, 5000 students = ~30-60 seconds

---

### 2. Allocate Single Course (Targeted)

**Use Case:** You need to allocate/re-allocate students for one specific course without affecting others.

**Command:**
```bash
# Replace {course_id} with actual course ID (e.g., 101)
curl -X POST \
  http://localhost:8080/api/allocations/students/course/101 \
  -H "Content-Type: application/json"
```

**Real Example:**
```bash
# Allocate students for course ID 101 (e.g., Data Structures)
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

**What Happens:**
1. System fetches course details (name, id)
2. Checks if teachers are allocated in `teacher_course_allocation`
3. Validates student learning modes
4. Distributes unallocated students to teachers
5. Returns course-specific report

**Expected Response (Success):**
```json
{
  "success": true,
  "course_id": 101,
  "course_name": "Data Structures",
  "allocated_count": 120,
  "ual_count": 20,
  "pbl_count": 100,
  "teachers_used": 4,
  "teacher_breakdown": [
    {
      "teacher_id": "CS1001",
      "teacher_name": "Dr. Alice Smith",
      "mode": "UAL",
      "student_count": 20
    },
    {
      "teacher_id": "CS1002",
      "teacher_name": "Dr. Bob Johnson",
      "mode": "PBL",
      "student_count": 34
    },
    {
      "teacher_id": "CS1003",
      "teacher_name": "Dr. Carol Williams",
      "mode": "PBL",
      "student_count": 33
    },
    {
      "teacher_id": "CS1004",
      "teacher_name": "Dr. David Brown",
      "mode": "PBL",
      "student_count": 33
    }
  ],
  "execution_time_seconds": 0.15,
  "timestamp": "2026-03-11T10:30:00Z"
}
```

**Expected Response (Error - No Teachers):**
```json
{
  "success": false,
  "error": "teacher_course_allocation is empty for this course. Please allocate teachers first.",
  "error_code": "NO_TEACHERS_ALLOCATED",
  "course_id": 101,
  "course_name": "Data Structures"
}
```

**When to Use Single Course:**
- ✅ Quick fix for one specific course
- ✅ Testing allocation logic on smaller dataset
- ✅ After adding teachers to only one course
- ✅ Faster execution (0.1-0.3 seconds per course)

---

### 3. View Current Allocations (Read-Only)

**Use Case:** Check how students are currently distributed among teachers for a course.

**Command:**
```bash
curl -X GET \
  http://localhost:8080/api/allocations/students/course/101/view \
  -H "Content-Type: application/json"
```

**Expected Response:**
```json
{
  "success": true,
  "course_id": 101,
  "course_name": "Data Structures",
  "total_students": 120,
  "allocations": [
    {
      "teacher_id": "CS1001",
      "teacher_name": "Dr. Alice Smith",
      "mode": "UAL",
      "mode_id": 1,
      "student_count": 20,
      "students": [
        {
          "student_id": 3050,
          "student_name": "John Doe",
          "enrollment_no": "22CS001",
          "register_no": "21001"
        }
      ]
    },
    {
      "teacher_id": "CS1002",
      "teacher_name": "Dr. Bob Johnson",
      "mode": "PBL",
      "mode_id": 2,
      "student_count": 34,
      "students": [...]
    }
  ]
}
```

**Use this to:**
- ✅ Verify allocations after running manual trigger
- ✅ Check teacher workload distribution
- ✅ Confirm UAL/PBL separation
- ✅ Debug allocation issues

---

### 4. Complete Re-allocation (Clean Slate)

**Use Case:** You want to completely reset and redistribute all students (e.g., after major teacher changes).

**⚠️ WARNING: This deletes all existing allocations!**

**Command:**
```bash
# Step 1: Clear existing allocations (DESTRUCTIVE)
# For specific course:
mysql -u root -p -e "DELETE FROM course_student_teacher_allocation WHERE course_id = 101;" academics

# For all courses:
mysql -u root -p -e "TRUNCATE TABLE course_student_teacher_allocation;" academics

# Step 2: Run allocation
curl -X POST http://localhost:8080/api/allocations/students/run
```

**Alternative (Safer - Mark as Inactive):**
```bash
# Mark existing allocations as inactive instead of deleting
mysql -u root -p -e "UPDATE course_student_teacher_allocation SET status = 0 WHERE course_id = 101;" academics

# Then re-run allocation (will create new active allocations)
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

---

### 5. Fix Prerequisites Before Allocation

**Scenario A: Students with NULL Learning Mode**

```sql
-- Step 1: Find affected students
SELECT 
    s.id,
    s.student_name,
    s.enrollment_no,
    s.register_no,
    COUNT(sc.course_id) as enrolled_courses
FROM students s
LEFT JOIN student_courses sc ON s.id = sc.student_id
WHERE s.status = 1
    AND (s.learning_mode_id IS NULL OR s.learning_mode_id NOT IN (1, 2))
GROUP BY s.id
ORDER BY s.student_name;

-- Step 2: Fix learning modes
-- Option 1: Set all to UAL
UPDATE students 
SET learning_mode_id = 1  -- 1 = UAL
WHERE learning_mode_id IS NULL 
AND status = 1;

-- Option 2: Set specific students to PBL
UPDATE students 
SET learning_mode_id = 2  -- 2 = PBL
WHERE id IN (3050, 3051, 3052)
AND status = 1;

-- Step 3: Run allocation
curl -X POST http://localhost:8080/api/allocations/students/run
```

**Scenario B: No Teachers Assigned to Course**

```sql
-- Step 1: Check which courses lack teachers
SELECT 
    c.id,
    c.course_code,
    c.course_name,
    COUNT(sc.student_id) as student_count,
    (SELECT COUNT(*) FROM teacher_course_allocation WHERE course_id = c.id) as teacher_count
FROM courses c
LEFT JOIN student_courses sc ON c.id = sc.course_id
WHERE (SELECT COUNT(*) FROM teacher_course_allocation WHERE course_id = c.id) = 0
GROUP BY c.id
HAVING student_count > 0;

-- Step 2: Add teachers to course (example)
INSERT INTO teacher_course_allocation (teacher_id, course_id, status)
VALUES 
    ('CS1001', 101, 1),
    ('CS1002', 101, 1),
    ('CS1003', 101, 1);

-- Step 3: Run allocation
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

**Scenario C: Insufficient Teachers for Mixed Mode Course**

```sql
-- Problem: Course has both UAL and PBL students but only 1 teacher
-- Solution: Add at least one more teacher

-- Step 1: Check current situation
SELECT 
    c.course_name,
    COUNT(DISTINCT CASE WHEN s.learning_mode_id = 1 THEN s.id END) as ual_students,
    COUNT(DISTINCT CASE WHEN s.learning_mode_id = 2 THEN s.id END) as pbl_students,
    (SELECT COUNT(*) FROM teacher_course_allocation WHERE course_id = c.id) as teacher_count
FROM courses c
INNER JOIN student_courses sc ON c.id = sc.course_id
INNER JOIN students s ON sc.student_id = s.id
WHERE c.id = 101
GROUP BY c.id;

-- Step 2: Add more teachers
INSERT INTO teacher_course_allocation (teacher_id, course_id, status)
VALUES ('CS1004', 101, 1);  -- Add second teacher

-- Step 3: Run allocation
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

---

### 6. Advanced: Using with Scripts

**Bash Script for Bulk Allocation:**

```bash
#!/bin/bash
# allocate_students.sh

API_URL="http://localhost:8080"
LOG_FILE="allocation_$(date +%Y%m%d_%H%M%S).log"

echo "=== Student Allocation Script ===" | tee -a $LOG_FILE
echo "Started at: $(date)" | tee -a $LOG_FILE
echo "" | tee -a $LOG_FILE

# Run full allocation
echo "Triggering full allocation..." | tee -a $LOG_FILE
response=$(curl -s -X POST "$API_URL/api/allocations/students/run")

# Parse response
success=$(echo $response | jq -r '.success')
total_allocated=$(echo $response | jq -r '.total_students_allocated')
failed_courses=$(echo $response | jq -r '.failed_courses')

if [ "$success" = "true" ]; then
    echo "✅ SUCCESS: Allocated $total_allocated students" | tee -a $LOG_FILE
    echo "⚠️  Failed courses: $failed_courses" | tee -a $LOG_FILE
else
    echo "❌ FAILED: Check response below" | tee -a $LOG_FILE
fi

# Save full response
echo "" | tee -a $LOG_FILE
echo "Full Response:" | tee -a $LOG_FILE
echo $response | jq '.' | tee -a $LOG_FILE

echo "" | tee -a $LOG_FILE
echo "Completed at: $(date)" | tee -a $LOG_FILE
```

**Usage:**
```bash
chmod +x allocate_students.sh
./allocate_students.sh
```

**Python Script for Course-by-Course Allocation:**

```python
#!/usr/bin/env python3
import requests
import json
from datetime import datetime

API_URL = "http://localhost:8080"
COURSE_IDS = [101, 102, 103, 104, 105]  # List of course IDs

def allocate_course(course_id):
    """Allocate students for a specific course"""
    url = f"{API_URL}/api/allocations/students/course/{course_id}"
    
    try:
        response = requests.post(url, headers={"Content-Type": "application/json"})
        result = response.json()
        
        if result.get('success'):
            print(f"✅ Course {course_id}: Allocated {result.get('allocated_count', 0)} students")
            return True, result
        else:
            error = result.get('error', 'Unknown error')
            print(f"❌ Course {course_id}: {error}")
            return False, result
    except Exception as e:
        print(f"⚠️  Course {course_id}: Exception - {str(e)}")
        return False, str(e)

def main():
    print("=== Student Allocation Script ===")
    print(f"Started at: {datetime.now()}")
    print(f"Allocating {len(COURSE_IDS)} courses...\n")
    
    results = {
        'success': [],
        'failed': []
    }
    
    for course_id in COURSE_IDS:
        success, result = allocate_course(course_id)
        
        if success:
            results['success'].append(course_id)
        else:
            results['failed'].append({'course_id': course_id, 'result': result})
    
    print("\n=== Summary ===")
    print(f"Successful: {len(results['success'])} courses")
    print(f"Failed: {len(results['failed'])} courses")
    
    if results['failed']:
        print("\nFailed Courses:")
        for item in results['failed']:
            print(f"  - Course {item['course_id']}: {item['result'].get('error', 'Unknown')}")
    
    print(f"\nCompleted at: {datetime.now()}")

if __name__ == "__main__":
    main()
```

**Usage:**
```bash
chmod +x allocate_students.py
python3 allocate_students.py
```

---

### 7. Common Manual Trigger Scenarios

**Scenario 1: Added 50 New Students to Existing Course**

```bash
# Students added to student_courses table
# Now allocate them

curl -X POST http://localhost:8080/api/allocations/students/course/101

# Verify allocation
curl -X GET http://localhost:8080/api/allocations/students/course/101/view
```

**Scenario 2: Added 3 New Teachers to Course (Need Redistribution)**

```bash
# Option A: Keep existing allocations, only allocate new students
curl -X POST http://localhost:8080/api/allocations/students/course/101

# Option B: Redistribute all students evenly across all teachers
mysql -u root -p -e "DELETE FROM course_student_teacher_allocation WHERE course_id = 101;" academics
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

**Scenario 3: Fixed Learning Modes for 20 Students**

```bash
# After updating learning_mode_id in students table
curl -X POST http://localhost:8080/api/allocations/students/run
```

**Scenario 4: Teacher Became Inactive Mid-Semester**

```bash
# Mark teacher as inactive in teachers table
mysql -u root -p -e "UPDATE teachers SET status = 0 WHERE faculty_id = 'CS1001';" academics

# System will automatically reallocate students from inactive teacher
curl -X POST http://localhost:8080/api/allocations/students/course/101

# Or run full allocation to handle all courses
curl -X POST http://localhost:8080/api/allocations/students/run
```

**Scenario 5: Testing Before Production**

```bash
# 1. Create test database backup
mysqldump -u root -p academics course_student_teacher_allocation > backup.sql

# 2. Run test allocation
curl -X POST http://localhost:8080/api/allocations/students/run

# 3. Verify results
curl -X GET http://localhost:8080/api/allocations/students/course/101/view

# 4. If not satisfied, restore backup
mysql -u root -p academics < backup.sql

# 5. Make adjustments and re-run
curl -X POST http://localhost:8080/api/allocations/students/run
```

---

### 8. Monitoring Manual Triggers

**Check Server Logs (Real-time):**
```bash
# In terminal where server is running, you'll see:
🎓 STARTING STUDENT-TEACHER-COURSE ALLOCATION
📚 Found 25 courses with teacher allocations

📖 Processing Course: Data Structures (ID: 101)
👨‍🏫 Found 4 active teachers
📊 Unallocated students: 20 UAL, 100 PBL
🎯 Assigning 4 free teachers: 1 to UAL, 3 to PBL
✅ Distributed 20 UAL students to 1 teachers
✅ Distributed 100 PBL students to 3 teachers

✅ ALLOCATION COMPLETE
```

**Save API Response for Analysis:**
```bash
# Save to file
curl -X POST http://localhost:8080/api/allocations/students/run \
  -o allocation_result.json

# Pretty print
cat allocation_result.json | jq '.'

# Check success status
cat allocation_result.json | jq '.success'

# Count allocated students
cat allocation_result.json | jq '.total_students_allocated'

# List failed courses
cat allocation_result.json | jq '.course_results[] | select(.success == false)'
```

---

### 9. Troubleshooting Manual Triggers

**Issue: API returns 404 Not Found**

```bash
# Check if server is running
curl http://localhost:8080/ping

# Check route registration
# Look for these routes in server logs:
# POST /api/allocations/students/run
# POST /api/allocations/students/course/{course_id}
# GET /api/allocations/students/course/{course_id}/view
```

**Issue: Allocation runs but no students allocated**

```sql
-- Check if students exist in student_courses
SELECT COUNT(*) FROM student_courses WHERE course_id = 101;

-- Check if students have valid learning_mode_id
SELECT COUNT(*) FROM students s
INNER JOIN student_courses sc ON s.id = sc.student_id
WHERE sc.course_id = 101
AND (s.learning_mode_id IS NULL OR s.learning_mode_id NOT IN (1,2));

-- Check if students are already allocated
SELECT COUNT(*) FROM course_student_teacher_allocation 
WHERE course_id = 101 AND status = 1;
```

**Issue: "INSUFFICIENT_TEACHERS" error**

```sql
-- Check teacher count vs student modes
SELECT 
    c.course_name,
    COUNT(DISTINCT CASE WHEN s.learning_mode_id = 1 THEN s.id END) as ual_students,
    COUNT(DISTINCT CASE WHEN s.learning_mode_id = 2 THEN s.id END) as pbl_students,
    (SELECT COUNT(*) FROM teacher_course_allocation WHERE course_id = c.id AND status = 1) as teacher_count
FROM courses c
INNER JOIN student_courses sc ON c.id = sc.course_id
INNER JOIN students s ON sc.student_id = s.id
WHERE c.id = 101
GROUP BY c.id;

-- If you have both UAL and PBL students, you need at least 2 teachers
-- Add more teachers:
INSERT INTO teacher_course_allocation (teacher_id, course_id, status)
VALUES ('CS1005', 101, 1);
```

---

### 10. Best Practices

✅ **Always verify before clearing allocations**
```bash
# Check current allocation count
echo "Current allocations: "
mysql -u root -p -e "SELECT COUNT(*) as count FROM course_student_teacher_allocation WHERE course_id = 101;" academics
```

✅ **Run single course first for testing**
```bash
# Test on one course before running full allocation
curl -X POST http://localhost:8080/api/allocations/students/course/101
```

✅ **Monitor execution time**
```bash
# Use time command
time curl -X POST http://localhost:8080/api/allocations/students/run
```

✅ **Save responses for audit trail**
```bash
# Create timestamped log
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
curl -X POST http://localhost:8080/api/allocations/students/run \
  -o "allocation_${TIMESTAMP}.json"
```

✅ **Validate after allocation**
```sql
-- Run validation query
SELECT 
    csta.course_id,
    c.course_name,
    csta.teacher_id,
    t.name,
    GROUP_CONCAT(DISTINCT s.learning_mode_id ORDER BY s.learning_mode_id) AS modes,
    COUNT(*) as total_students
FROM course_student_teacher_allocation csta
INNER JOIN students s ON csta.student_id = s.id
INNER JOIN teachers t ON csta.teacher_id = t.faculty_id
INNER JOIN courses c ON csta.course_id = c.id
WHERE csta.status = 1
GROUP BY csta.course_id, csta.teacher_id
HAVING COUNT(DISTINCT s.learning_mode_id) > 1;  -- Should return 0 rows
```

---

## Integration with Mark Entry

### Before This System
- Teachers saw ALL students in a course
- No mode-based separation in mark entry

### After This System
- Teachers see only THEIR allocated students
- Mark entry queries filter by `teacher_id`

**Updated Mark Entry Query:**
```sql
SELECT s.* 
FROM students s
INNER JOIN course_student_teacher_allocation csta 
    ON s.id = csta.student_id
WHERE csta.course_id = ? 
    AND csta.teacher_id = ?  -- Current teacher's faculty_id
    AND csta.status = 1
    AND s.learning_mode_id = ?  -- Optional: filter by mode
ORDER BY s.student_name;
```

---

## Troubleshooting

### Problem: Students not allocated
**Check:**
1. Is student in `student_courses`?
2. Does course have teachers in `teacher_course_allocation`?
3. Does student have valid `learning_mode_id`?

**Debug Query:**
```sql
SELECT 
    s.id,
    s.student_name,
    s.learning_mode_id,
    sc.course_id,
    c.course_name,
    (SELECT COUNT(*) FROM teacher_course_allocation WHERE course_id = sc.course_id) as teacher_count
FROM students s
INNER JOIN student_courses sc ON s.id = sc.student_id
INNER JOIN courses c ON sc.course_id = c.id
LEFT JOIN course_student_teacher_allocation csta 
    ON csta.student_id = s.id AND csta.course_id = sc.course_id
WHERE csta.id IS NULL 
    AND s.id = 3050;  -- specific student
```

### Problem: Uneven distribution
**Cause:** Teachers added at different times

**Solution:** Clear and re-allocate
```sql
DELETE FROM course_student_teacher_allocation WHERE course_id = 101;
-- Then run API: POST /api/allocations/students/course/101
```

### Problem: Teacher marked inactive but still has students
**Solution:** System automatically reallocates on next run

**Force immediate reallocation:**
```bash
curl -X POST http://localhost:8080/api/allocations/students/course/{course_id}
```

---

## Performance Considerations

- **Batch Size:** Inserts in batches of 500 for optimal performance
- **Large Courses:** 1000 students = ~2-3 seconds
- **Full Allocation:** 100 courses = ~30-60 seconds
- **Database Indexes:** Ensure indexes on `student_id`, `course_id`, `teacher_id`

### Recommended Indexes
```sql
CREATE INDEX idx_csta_course ON course_student_teacher_allocation(course_id);
CREATE INDEX idx_csta_teacher ON course_student_teacher_allocation(teacher_id);
CREATE INDEX idx_csta_status ON course_student_teacher_allocation(status);
CREATE INDEX idx_student_mode ON students(learning_mode_id);
```

---

## Logs & Monitoring

### Success Log Example
```
🎓 STARTING STUDENT-TEACHER-COURSE ALLOCATION
📚 Found 25 courses with teacher allocations

📖 Processing Course: Mathematics (ID: 101)
👨‍🏫 Found 4 active teachers
📊 Unallocated students: 20 UAL, 100 PBL
👥 Teacher availability: 0 free, 0 UAL-locked, 0 PBL-locked
🎯 Assigning 4 free teachers: 1 to UAL, 3 to PBL
✅ Distributed 20 UAL students to 1 teachers
✅ Distributed 100 PBL students to 3 teachers

✅ ALLOCATION COMPLETE
   Total Courses: 25
   Successful: 25
   Failed: 0
   Total Students Allocated: 1250
   Execution Time: 3.45 seconds
```

### Error Log Example
```
📖 Processing Course: Physics (ID: 102)
❌ No teachers allocated for course 102

📖 Processing Course: Chemistry (ID: 103)
⚠️  Found 5 students with NULL learning_mode_id
```

---

## Contact & Support

For issues or questions:
1. Check logs in console
2. Run validation queries
3. Review error codes in this document
4. Check API responses for detailed error messages
