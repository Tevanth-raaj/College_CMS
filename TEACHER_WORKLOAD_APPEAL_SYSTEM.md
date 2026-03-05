# Teacher Workload Appeal System - Implementation Summary

## Overview
Implemented a complete appeal system allowing teachers to appeal their assigned workload. When a teacher submits an appeal, they cannot select courses until HR reviews and resolves the appeal (approved or rejected).

## Database Changes

### 1. Modified `teacher_course_appeal` Table
**Migration File:** `server/db/migrations/20260226_modify_appeal_and_limits.sql`

**Changes:**
- ✅ Removed `course_category` column (HR can reduce workload in any category when appealed)
- ✅ Removed `new_count` column (not needed for simplified workflow)
- ✅ Kept essential columns:
  - `faculty_id` - Teacher who submitted appeal
  - `appeal_message` - Teacher's message explaining the appeal
  - `appeal_status` - 0 = pending, 1 = resolved
  - `hr_action` - APPROVED or REJECTED
  - `hr_message` - HR's response message
  - `created_at`, `updated_at`, `resolved_at` - Timestamps

### 2. Modified `teacher_course_limits` Table
**Changes:**
- ✅ Added `is_active` column (TINYINT, default 1) - Tracks if limits are active for current academic year
- ✅ Added `academic_year` column (VARCHAR(20)) - Tracks which year these limits are for
- ✅ Added `created_at` and `updated_at` timestamps
- ✅ Updated unique constraint to include `academic_year`: `(teacher_id, course_type_id, academic_year)`

**Purpose:** When HR assigns new counts for an academic year, the system can override previous limits. Limits are automatically set to inactive when the window period closes.

## Backend Implementation

### 1. Migration System
**File:** `server/db/db.go`

**Changes:**
- ✅ Added `runSQLMigrations()` function that automatically reads and executes all `.sql` files from `db/migrations/` directory when the backend starts
- ✅ Migrations are executed in alphabetical order
- ✅ Migration errors are logged but don't stop the application from starting

### 2. Appeal Handlers
**File:** `server/handlers/student-teacher_entry/appeals.go`

**Updated Handlers:**
- ✅ `CreateCourseAppeal` - Creates new appeal (simplified, no course_category needed)
- ✅ `GetTeacherPendingAppeal` - Checks if teacher has a pending appeal
- ✅ `GetAllAppeals` - Gets all appeals with status filter (for HR page)
- ✅ `UpdateAppealStatus` - HR resolves appeal (APPROVED or REJECTED)
- ✅ `GetAppealByID` - Gets specific appeal details
- ✅ `GetTeacherAppealHistory` - Gets all appeals for a teacher

**Key Changes:**
- Removed `course_category` parameter from all handlers
- Removed `new_count` logic
- Changed `appeal_status` from BOOLEAN to TINYINT (0/1)
- Simplified HR action to just APPROVED or REJECTED

### 3. Course Selection Blocking
**File:** `server/handlers/student-teacher_entry/teacher_course_preferences.go`

**Logic:**
- ✅ Before allowing course selection, checks if teacher has pending appeal
- ✅ If pending appeal exists, blocks course selection with error message
- ✅ Updated query to use `appeal_status = 0` instead of `appeal_status = FALSE`

### 4. API Routes
**File:** `server/routes/routes.go`

**New Routes:**
```go
// Teacher endpoints
POST   /api/teachers/appeals                    // Submit appeal
GET    /api/teachers/appeals/pending            // Check pending appeal (?faculty_id=)
GET    /api/teachers/appeals/history            // Get appeal history (?faculty_id=)

// HR endpoints
GET    /api/hr/appeals                          // Get all appeals (?status=pending|resolved)
GET    /api/hr/appeals/{appeal_id}              // Get specific appeal
PUT    /api/hr/appeals/{appeal_id}/resolve      // Resolve appeal
```

## Frontend Implementation

### 1. Teacher Course Selection Page
**File:** `client/src/pages/teacher/TeacherCourseSelectionPage.js`

**New Features:**
- ✅ **Appeal Button** - Appears in Allocation Summary card (orange button "Appeal Workload")
- ✅ **Pending Appeal Indicator** - Shows yellow badge when appeal is pending
- ✅ **Appeal Modal** - Form to submit appeal message
- ✅ **Course Selection Blocking** - Automatically checks for pending appeal and prevents course selection

**New State Variables:**
```javascript
const [showAppealModal, setShowAppealModal] = useState(false);
const [appealMessage, setAppealMessage] = useState('');
const [hasPendingAppeal, setHasPendingAppeal] = useState(false);
const [appealSubmitting, setAppealSubmitting] = useState(false);
```

**New Functions:**
- `checkPendingAppeal()` - Checks if teacher has pending appeal on page load
- `handleSubmitAppeal()` - Submits appeal to backend

**UI Components:**
- Appeal button in allocation summary (only shown if no pending appeal)
- Pending appeal indicator (yellow badge)
- Appeal modal with textarea and submit button

### 2. HR Appeals Review Page
**File:** `client/src/pages/hr/HRAppealsReviewPage.js`

**Updated Features:**
- ✅ Updated API endpoints to new routes
- ✅ Removed `course_category` column from table
- ✅ Removed `new_count` input field
- ✅ Simplified decision options to APPROVED or REJECTED
- ✅ Updated modal display to show simplified appeal information

**API Changes:**
```javascript
// Old endpoints
GET  /teachers/course-appeal/all?status=...
PUT  /teachers/course-appeal/update?appeal_id=...

// New endpoints  
GET  /hr/appeals?status=...
PUT  /hr/appeals/{appeal_id}/resolve
```

## Workflow

### Teacher Side
1. **View Allocation Summary** - Teacher sees their assigned workload counts
2. **Click "Appeal Workload"** - Button appears if no pending appeal
3. **Write Appeal Message** - Explain why workload reduction is needed
4. **Submit Appeal** - Appeal sent to HR for review
5. **Course Selection Blocked** - Cannot select courses until appeal is resolved
6. **Wait for HR Decision** - Pending indicator shows yellow badge

### HR Side
1. **View All Appeals** - Switch between Pending/Resolved/All
2. **Review Appeal** - Click "Review" button to see details
3. **Make Decision** - Choose APPROVED (reduce workload) or REJECTED
4. **Add HR Message** - Optional message explaining the decision
5. **Save Decision** - Appeal status updated, teacher can now select courses (if approved)

### After Appeal Resolution
- If **APPROVED**: HR must manually adjust workload counts in the HR Faculty page
- If **REJECTED**: Teacher's original workload remains unchanged
- In both cases: Teacher can now proceed with course selection

## Testing Checklist

### Backend
- [ ] Start server and check migration logs
- [ ] Verify `teacher_course_appeal` table structure
- [ ] Verify `teacher_course_limits` table has new columns
- [ ] Test appeal creation API
- [ ] Test pending appeal check API
- [ ] Test HR resolve appeal API

### Frontend - Teacher
- [ ] Open Teacher Course Selection page
- [ ] Verify appeal button appears in allocation summary
- [ ] Click appeal button and submit appeal
- [ ] Verify pending indicator appears
- [ ] Try to select courses (should be blocked)
- [ ] Check for error message about pending appeal

### Frontend - HR
- [ ] Open HR Appeals Review page
- [ ] Verify pending appeals appear
- [ ] Click Review on an appeal
- [ ] Make decision (APPROVED or REJECTED)
- [ ] Add optional HR message
- [ ] Save decision
- [ ] Verify appeal moves to Resolved tab

### Integration
- [ ] Submit appeal as teacher
- [ ] Approve appeal as HR
- [ ] Verify teacher can now select courses
- [ ] Verify pending indicator is removed

## Files Modified

### Backend
1. `server/db/db.go` - Added migration system
2. `server/db/migrations/20260226_modify_appeal_and_limits.sql` - New migration file
3. `server/handlers/student-teacher_entry/appeals.go` - Updated all handlers
4. `server/handlers/student-teacher_entry/teacher_course_preferences.go` - Updated appeal check
5. `server/routes/routes.go` - Added appeal routes

### Frontend
1. `client/src/pages/teacher/TeacherCourseSelectionPage.js` - Added appeal system
2. `client/src/pages/hr/HRAppealsReviewPage.js` - Updated to new API

## Next Steps

1. **Test the implementation** - Run through the testing checklist
2. **HR Manual Workload Update** - When appeal is approved, HR needs to manually update counts in HR Faculty page
3. **Add email notifications** (optional) - Notify teacher when appeal is resolved
4. **Add appeal history view** (optional) - Show teacher their past appeals
5. **Automatic workload reset** - When window period closes, set `is_active = 0` for all limits

## Notes

- Appeals are checked on page load, so teacher must refresh page to see updated status after HR resolves
- HR can only reduce workload manually after approving appeal (system doesn't auto-adjust)
- One pending appeal per teacher at a time
- Appeal message is required, HR message is optional
