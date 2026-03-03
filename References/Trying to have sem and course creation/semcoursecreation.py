import pandas as pd
import requests
import re

# --- CONFIGURATION ---
FACULTY_FILE = "TEACHERSFILE.csv"
OUTPUT_SQL = "faculty_course_linkage.sql"

# API Configuration
CURRICULUM_ID = 296
BASE_URL = "http://localhost:5000/api/curriculum/{}".format(CURRICULUM_ID)

# System Constants
DEPT_CODE = "AD"
DEPT_NAME = "Artificial Intelligence and Data Science"

# Cache for Semester IDs
semester_cache = {}

def get_or_create_semester(sem_str):
    """API: Creates semester (e.g., 'S4' -> 4) and returns the ID."""
    try:
        sem_num = int(re.search(r'\d+', str(sem_str)).group())
    except:
        return None

    if sem_num in semester_cache:
        return semester_cache[sem_num]

    url = f"{BASE_URL}/semester"
    payload = {"semester_number": sem_num, "card_type": "semester"}
    
    try:
        response = requests.post(url, json=payload, timeout=5)
        if response.status_code in [200, 201]:
            sem_id = response.json().get("id")
            semester_cache[sem_num] = sem_id
            return sem_id
        elif response.status_code == 400:
            print(f"⚠️ Semester {sem_num} exists. Ensure ID is mapped.")
            return None
    except Exception as e:
        print(f"❌ Semester API Error: {e}")
    return None

def sync_course_api(sem_id, row):
    """API: Links course to the created semester."""
    if not sem_id: return
    
    url = f"{BASE_URL}/semester/{sem_id}/course"
    payload = {
        "course_code": str(row['Course Code']).strip(),
        "course_name": str(row['Course Name']).replace("'", "''"),
        "course_type": str(row.get('Course Type', 'CORE')),
        "category": str(row.get('Course Nature', 'THEORY')),
        "credit": int(row.get('C', 0)),
        "lecture_hrs": int(row.get('L', 0)),
        "tutorial_hrs": int(row.get('T', 0)),
        "practical_hrs": int(row.get('P', 0)),
        "cia_marks": 40,
        "see_marks": 60
    }
    
    try:
        response = requests.post(url, json=payload, timeout=5)
        if response.status_code in [200, 201]:
            print(f"✅ Course Sync: {payload['course_code']}")
        else:
            print(f"❌ Course Sync Failed: {payload['course_code']} - {response.text}")
    except Exception as e:
        print(f"⚠️ API Error: {e}")

def run_setup():
    sql_lines = [
        "-- FACULTY AND COURSE LINKAGE SETUP",
        "SET FOREIGN_KEY_CHECKS = 0;",
        f"INSERT INTO `departments` (department_code, department_name, status) VALUES ('{DEPT_CODE}', '{DEPT_NAME}', 1) ON DUPLICATE KEY UPDATE status=1;",
        f"SET @dept_id = (SELECT id FROM `departments` WHERE department_code = '{DEPT_CODE}' LIMIT 1);",
        ""
    ]

    try:
        # Load Teacher Data
        df = pd.read_excel(FACULTY_FILE) if FACULTY_FILE.endswith('.xlsx') else pd.read_csv(FACULTY_FILE)
        fac_id_col = 'Faculty ID\n(ME1880)'
        df = df.dropna(subset=[fac_id_col, 'Course Code'])

        # 1. API: SYNC SEMESTERS AND COURSES
        print("🚀 Syncing Semesters and Courses via API...")
        unique_courses = df.groupby('Course Code').first().reset_index()
        for _, row in unique_courses.iterrows():
            sem_id = get_or_create_semester(row.get('Semester', 'S1'))
            sync_course_api(sem_id, row)

        # 2. SQL: TEACHERS
        sql_lines.append("-- SECTION: TEACHERS")
        unique_teachers = df[[fac_id_col, 'Faculty Name', 'Mail ID']].drop_duplicates()
        for _, row in unique_teachers.iterrows():
            f_id = str(row[fac_id_col]).strip()
            name = str(row['Faculty Name']).replace("'", "''")
            sql_lines.append(f"INSERT INTO `teachers` (faculty_id, name, email) VALUES ('{f_id}', '{name}', '{row['Mail ID']}') ON DUPLICATE KEY UPDATE name=VALUES(name);")
            sql_lines.append(f"INSERT INTO `department_teachers` (teacher_id, department_id) VALUES ('{f_id}', @dept_id) ON DUPLICATE KEY UPDATE status=1;")

        # 3. SQL: TEACHER-COURSE ALLOCATION
        # FIXED: Removed academic_year and semester from this specific insertion
        sql_lines.append("\n-- SECTION: TEACHER-COURSE ALLOCATION")
        for _, row in df.iterrows():
            f_id = str(row[fac_id_col]).strip()
            c_code = str(row['Course Code']).strip()
            
            sql_lines.append(
                f"INSERT INTO `teacher_course_allocation` (course_id, teacher_id) "
                f"SELECT id, '{f_id}' FROM `courses` WHERE course_code = '{c_code}' LIMIT 1 "
                f"ON DUPLICATE KEY UPDATE teacher_id=VALUES(teacher_id);"
            )

        sql_lines.append("\nSET FOREIGN_KEY_CHECKS = 1;")

        with open(OUTPUT_SQL, "w", encoding="utf-8") as f:
            f.write("\n".join(sql_lines))
        
        print(f"\n💾 Success! SQL generated in {OUTPUT_SQL}")

    except Exception as e:
        print(f"❌ Processing Error: {e}")

if __name__ == "__main__":
    run_setup()
