# Minor Slot System - Simplified Implementation

## Overview

The minor system now uses a slot-based approach similar to Honor Slots. HODs can assign courses to **Minor Slot 1** and **Minor Slot 2** in each semester (4-8).

## Changes Made

### 1. Database Setup

**File:** `server/db/minor_slots_setup.sql`

- Added "Minor Slot 1" and "Minor Slot 2" to semesters 4-8
- Similar structure to Honour Slots
- Each semester now has 2 minor slots available

### 2. Frontend (HODElectivePage.js)

- **Removed** the separate "Minor Program Management" section
- Minor slots now work automatically through the regular elective slot system
- HOD assigns courses to Minor Slot 1 and Minor Slot 2 just like any other slot
- **Validation:** Each minor slot can have only 1 course (like honor slots)
- **Validation:** Max 2 minor courses per semester
- Minor slots are displayed with indigo badge: "Minor"

### 3. Backend (electives.go)

- Added validation for Minor Slot 1 and Minor Slot 2
- Each individual minor slot can have max 1 course
- Max 2 minor courses allowed per semester
- Minor courses cannot overlap with Professional Electives
- Minor courses tracked separately from honor courses

## How It Works

### For HOD:

1. Navigate to HOD Elective Management page
2. Select a semester (e.g., Semester 5)
3. Find "Minor Slot 1" and "Minor Slot 2" in the slot list
4. Assign 1 course to each minor slot from available courses
5. Save assignments
6. The 2 courses assigned to minor slots become available for students from other departments

### Slot System:

- **Minor Slot 1**: Can hold 1 course only
- **Minor Slot 2**: Can hold 1 course only
- Total: 2 minor courses per semester
- Works across semesters 4-8

### Validation Rules:

1. Each specific minor slot (Minor Slot 1 or Minor Slot 2) can have only 1 course
2. Maximum 2 minor courses per semester total
3. Minor courses cannot be the same as professional electives
4. No overlap between minor and professional elective courses

## Database Migration

Run this SQL to add minor slots:

```sql
-- Located in: server/db/minor_slots_setup.sql
-- This adds Minor Slot 1 and Minor Slot 2 to semesters 4-8
```

## Testing

1. **Test Case 1: Assign 2 minor courses**
   - Select Semester 5
   - Assign Course A to "Minor Slot 1"
   - Assign Course B to "Minor Slot 2"
   - Click "Save Assignments"
   - ✓ Should save successfully

2. **Test Case 2: Try to assign 2 courses to same minor slot**
   - Assign Course A to "Minor Slot 1"
   - Try to assign Course B to "Minor Slot 1" also
   - ⚠️ Should show error: "Minor Slot 1 can have only 1 course"

3. **Test Case 3: Prevent overlap with professional electives**
   - Assign Course A to "Professional Elective 1"
   - Try to assign Course A to "Minor Slot 1"
   - ⚠️ Should prevent or warn about overlap

## Advantages of Slot-Based System

1. **Simpler UI**: No separate management section needed
2. **Consistent**: Works exactly like Honor Slots
3. **Flexible**: HOD can choose any courses from their curriculum
4. **Integrated**: Uses existing slot validation and save system
5. **No extra API calls**: Uses the same save endpoint as other electives

## Notes

- The old Minor Program Management section (with vertical selection and department selection) has been removed
- Minor courses are now just regular elective assignments using specialized slots
- Students from other departments can view and select from available minor courses
- The system automatically displays minor slots with indigo colored badges
