import pandas as pd

# --- CONFIGURATION ---
FACULTY_FILE = "TEACHERSFILE.csv"
OUTPUT_SQL = "teacher_course_mapping.sql"

# System Constants
DEPT_CODE = "AD"
DEPT_NAME = "Artificial Intelligence and Data Science"

def run_mapping_setup():
    sql_lines = [
        "-- TEACHER AND COURSE MAPPING SETUP",
        "SET FOREIGN_KEY_CHECKS = 0;",
        "",
        "-- Ensure Department Exists",
        f"INSERT INTO `departments` (department_code, department_name, status) VALUES ('{DEPT_CODE}', '{DEPT_NAME}', 1) ON DUPLICATE KEY UPDATE status=1;",
        f"SET @dept_id = (SELECT id FROM `departments` WHERE department_code = '{DEPT_CODE}' LIMIT 1);",
        ""
    ]

    try:
        # Load Teacher Data from CSV or Excel
        df = pd.read_excel(FACULTY_FILE) if FACULTY_FILE.endswith('.xlsx') else pd.read_csv(FACULTY_FILE)
        
        # Define the Faculty ID column based on your previous schema
        fac_id_col = 'Faculty ID\n(ME1880)'
        
        # Clean data: drop rows missing essential mapping info
        df = df.dropna(subset=[fac_id_col, 'Course Code'])

        # 1. SECTION: TEACHER PROFILES & DEPT LINKS
        sql_lines.append("-- SECTION 1: TEACHER PROFILES")
        unique_teachers = df[[fac_id_col, 'Faculty Name', 'Mail ID']].drop_duplicates()
        
        for _, row in unique_teachers.iterrows():
            f_id = str(row[fac_id_col]).strip()
            name = str(row['Faculty Name']).replace("'", "''")
            email = str(row['Mail ID']).strip()
            
            # Insert teacher master record
            sql_lines.append(f"INSERT INTO `teachers` (faculty_id, name, email) VALUES ('{f_id}', '{name}', '{email}') ON DUPLICATE KEY UPDATE name=VALUES(name), email=VALUES(email);")
            
            # Link teacher to department
            sql_lines.append(f"INSERT INTO `department_teachers` (teacher_id, department_id) VALUES ('{f_id}', @dept_id) ON DUPLICATE KEY UPDATE status=1;")

        # 2. SECTION: TEACHER-COURSE ALLOCATION
        sql_lines.append("\n-- SECTION 2: TEACHER-COURSE ALLOCATION")
        for _, row in df.iterrows():
            f_id = str(row[fac_id_col]).strip()
            c_code = str(row['Course Code']).strip()
            
            # This query looks up the internal course ID based on the alphanumeric course code
            sql_lines.append(
                f"INSERT INTO `teacher_course_allocation` (course_id, teacher_id) "
                f"SELECT id, '{f_id}' FROM `courses` WHERE course_code = '{c_code}' LIMIT 1 "
                f"ON DUPLICATE KEY UPDATE teacher_id=VALUES(teacher_id);"
            )

        sql_lines.append("\nSET FOREIGN_KEY_CHECKS = 1;")

        # Write generated SQL to file
        with open(OUTPUT_SQL, "w", encoding="utf-8") as f:
            f.write("\n".join(sql_lines))
        
        print(f"✅ Success! Mapping SQL generated: {OUTPUT_SQL}")
        print("👉 Run this SQL file to link teachers to existing courses.")

    except Exception as e:
        print(f"❌ Error: {e}")

if __name__ == "__main__":
    run_mapping_setup()
