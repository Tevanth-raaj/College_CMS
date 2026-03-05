# Course Preferences - Two Window Submission System

## Overview
Teachers can now submit course preferences **twice per academic year** - once for each semester window (odd and even). Core courses are always shown, while extra courses (electives, honors, minors) are filtered to display only what students in the teacher's department have enrolled in.

## Implementation Summary

### 1. Backend Changes

#### File: `server/handlers/curriculum/department_semester_courses.go`
**Change**: Complete rewrite of `GetCoursesForTeacherSemester()` function

**New Logic**:
- Splits courses into two categories:
  1. **Core Courses**: From `curriculum_courses` table
     - Required courses all students must take
     - Query filters by curriculum and semester
     - Excludes "Elective", "Open", "Honour" categories
  
  2. **Extra Courses**: From `student_elective_choices` table
     - Student-chosen electives, honours, minors, open electives
     - Only shows courses students in teacher's department actually chose
     - Joins: courses → hod_elective_selections → student_elective_choices → students

**Response Format**: 
```json
{
  "coreCourses": [...],
  "extraCourses": [...],
  "message": "Core courses required for all; Extra courses are only those students in your department have enrolled in"
}
```

### 2. Backend Submission Logic

#### File: `server/handlers/student_teacher_entry/teacher_course_preferences.go`
**Existing Logic - Already Implemented**:
- Submission lock checks: `(teacher_id, academic_year, current_semester_type)`
- Allows ONE submission per semester type per academic year
- Returns HTTP 403 Forbidden on duplicate submission

**Two Window System**:
- `current_semester_type` = 'odd' → WINDOW 1 (Semesters 1, 3, 5, 7)
- `current_semester_type` = 'even' → WINDOW 2 (Semesters 2, 4, 6, 8)
- Total: 2 submissions per academic year allowed

### 3. Frontend Changes

#### File: `client/src/pages/teacher/TeacherCourseSelectionPage.js`

**Updated Course Fetching**:
- Handles new `coreCourses` and `extraCourses` response
- Combines both arrays with `courseSource` metadata
- Stores source info ('core' or 'extra') on each course object

**Visual Indicators**:
- **Course Info Cards**: Show blue "Core" badge or green "Student Choice" badge
- **Info Section**: Two info boxes explaining:
  - Core courses (blue): Required for all students
  - Student-chosen electives (green): Only courses department students enrolled in
- **Header Note**: "You can submit course preferences twice per academic year: once for odd semesters, once for even semesters"

**Course Display Flow**:
1. Fetch courses from both categories
2. Combine with semester metadata
3. Filter by course_type (Theory, Theory with Lab, Lab) via tabs
4. Search functionality works across both categories
5. Display count shows total available courses

## Data Flow

### For Window 1 (Odd Semesters):
```
GET /teachers/{teacherId}/semester/{1,3,5,7}/courses
├─ Semester 1 Core Courses (category NOT LIKE '%Elective%')
├─ Semester 1 Extra Courses (from student_elective_choices)
├─ Semester 3 Core Courses
├─ Semester 3 Extra Courses
├─ Semester 5 Core Courses
├─ Semester 5 Extra Courses
├─ Semester 7 Core Courses
└─ Semester 7 Extra Courses

Response: {coreCourses: [...], extraCourses: [...]}
```

### For Window 2 (Even Semesters):
```
GET /teachers/{teacherId}/semester/{2,4,6,8}/courses
├─ Same pattern for even semesters
```

### Submission:
```
POST /teachers/course-preferences
Request: {teacher_id, academic_year, current_semester_type, preferences}
├─ Check: Existing records for (teacher_id, academic_year, current_semester_type)
├─ If found: Return HTTP 403 "Already submitted"
└─ If NOT found: INSERT preferences
```

## Key Features

✅ **Two Submissions Per Year**: One for odd window, one for even window
✅ **Core Course Display**: Always shows core curriculum courses
✅ **Smart Elective Filtering**: Shows only electives/honours/minors students chose
✅ **Clear Visual Distinction**: Badges indicate course source
✅ **Submission Lock**: Database prevents duplicate submissions per window
✅ **Course Type Filtering**: Theory, Theory with Lab, Lab tabs work across both categories
✅ **Backward Compatible**: Falls back gracefully if courseSource metadata missing

## Testing Checklist

- [ ] Login as teacher
- [ ] View course selection page
- [ ] Verify core courses display (with blue badge)
- [ ] Verify extra courses display (with green badge)
- [ ] Filter by course type (theory, theory with lab, lab)
- [ ] Search courses
- [ ] Select courses from both categories
- [ ] Submit preferences (Window 1)
- [ ] Verify success message
- [ ] Check database for stored preferences
- [ ] **Attempt second submission for same window** → Should get 403 "Already submitted"
- [ ] **Wait for Window 2 (different semester_type)** → Should allow NEW submission
- [ ] Verify appeal system still works
- [ ] Verify allocation summary displays correctly

## Database Considerations

**Required Tables**:
- `courses`: course data with category field
- `curriculum_courses`: core course mappings
- `hod_elective_selections`: elective slot definitions
- `student_elective_choices`: student course selections
- `teacher_course_preferences`: submission storage
- `course_type`: course type definitions (theory, lab, theory_with_lab)

**Foreign Keys**:
- student_elective_choices → hod_elective_selections → courses
- hod_elective_selections → students (via joins)

## Error Handling

**Submission Lock Errors**:
```
HTTP 403 Forbidden
{
  "success": false,
  "error": "You have already submitted your course preferences for [academic_year] ([semester_type]). You can submit again when the next window opens."
}
```

**Missing Teacher Data**:
```
HTTP 404 Not Found
{
  "error": "Could not determine teacher's department"
}
```

**Course Fetch Errors**:
- Gracefully returns empty arrays if queries fail
- Logs errors for debugging
- No data loss, user can retry

## Performance Notes

- Core courses query: Simple curriculum filter (fast)
- Extra courses query: Multiple joins through student_elective_choices (may be slower with many students)
- Recommend indexing on: student_id, hod_selection_id, department_id, semester
- Response combines 4 semesters worth of data (parallel fetches in frontend)

## Future Enhancements

- Add window names (e.g., "Odd Window Q1-Q2", "Even Window Q3-Q4")
- Display submission history/timeline per window
- Add course capacity indicators
- Show number of students who enrolled in each course
- Add export/print functionality for course preferences
