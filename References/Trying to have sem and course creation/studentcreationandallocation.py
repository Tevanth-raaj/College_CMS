import pandas as pd
import re

# --- CONFIGURATION ---
STUDENT_FILE = "STUDENTSFILE.csv"
OUTPUT_SQL = "student_allocation_setup.sql"

# System Constants
DEPT_CODE = "AD"
ACADEMIC_YEAR = "2025-26"
SEMESTER = 4

def generate_student_script():
    sql_lines = [
        "-- STUDENT PROFILE AND BALANCED ALLOCATION SETUP",
        "SET FOREIGN_KEY_CHECKS = 0;",
        f"SET @dept_id = (SELECT id FROM `departments` WHERE department_code = '{DEPT_CODE}' LIMIT 1);",
        ""
    ]

    try:
        # 1. LOAD DATA
        df_stud = pd.read_csv(STUDENT_FILE)
        
        # Define the columns where course codes are stored (matching your CSV headers)
        course_cols = [
            'PROFESSIONAL ELECTIVE 1', '22XX401', '22XX402', '22XX403', 
            '22XX404', '22XX405', '22XX007', '22HS008', '22HS010'
        ]

        # 2. SECTION: STUDENT PROFILES
        sql_lines.append("-- SECTION: STUDENT PROFILES")
        for _, row in df_stud.iterrows():
            reg_no = str(row['Reg. No.']).strip()
            name = str(row['Student Name']).replace("'", "''")
            
            # Insert Student
            sql_lines.append(
                f"INSERT INTO `students` (register_no, student_name, department_id) "
                f"VALUES ('{reg_no}', '{name}', @dept_id) "
                f"ON DUPLICATE KEY UPDATE student_name=VALUES(student_name);"
            )
            
            # 3. SECTION: STUDENT COURSE REGISTRATION
            # This parses entries like '22AG401-CROP PRODUCTION' to extract '22AG401'
            for col in course_cols:
                val = str(row.get(col, ''))
                if '-' in val:
                    c_code = val.split('-')[0].strip()
                    sql_lines.append(
                        f"INSERT IGNORE INTO `student_courses` (student_id, course_id) "
                        f"SELECT s.id, c.id FROM `students` s, `courses` c "
                        f"WHERE s.register_no = '{reg_no}' AND c.course_code = '{c_code}';"
                    )

        # 4. SECTION: BALANCED AUTOMATIC ALLOCATION
        # This logic divides students equally among the multiple teachers assigned to a course.
        sql_lines.append("\n-- SECTION: BALANCED TEACHER-STUDENT ALLOCATION")
        sql_lines.append("TRUNCATE TABLE `course_student_teacher_allocation`;")
        
        sql_lines.append(
            "INSERT INTO `course_student_teacher_allocation` (student_id, course_id, teacher_id) "
            "SELECT S_Ranked.student_id, S_Ranked.course_id, T_Ranked.teacher_faculty_id FROM ( "
            "    SELECT student_id, course_id, "
            "    ROW_NUMBER() OVER (PARTITION BY course_id ORDER BY student_id) as s_idx "
            "    FROM `student_courses` "
            ") AS S_Ranked "
            "JOIN ( "
            "    SELECT t.faculty_id AS teacher_faculty_id, tca.course_id, "
            "    ROW_NUMBER() OVER (PARTITION BY tca.course_id ORDER BY t.faculty_id) as t_idx, "
            "    COUNT(*) OVER (PARTITION BY tca.course_id) as t_total "
            "    FROM `teacher_course_allocation` tca "
            "    JOIN `teachers` t ON (tca.teacher_id = t.faculty_id OR CAST(tca.teacher_id AS CHAR) = CAST(t.id AS CHAR)) "
            ") AS T_Ranked ON S_Ranked.course_id = T_Ranked.course_id "
            "WHERE T_Ranked.t_idx = ((S_Ranked.s_idx - 1) % T_Ranked.t_total) + 1;"
        )

        sql_lines.append("\nSET FOREIGN_KEY_CHECKS = 1;")

        # Write to File
        with open(OUTPUT_SQL, "w", encoding="utf-8") as f:
            f.write("\n".join(sql_lines))
        
        print(f"✅ Success! SQL generated in {OUTPUT_SQL}")
        print(f"👉 Now run the SQL file in your DB to finish the allocation.")

    except Exception as e:
        print(f"❌ Error: {e}")

if __name__ == "__main__":
    generate_student_script()
