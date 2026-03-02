import pandas as pd

# --- CONFIGURATION ---
STUDENT_FILE = "STUDENTSFILE.csv"
OUTPUT_SQL = "student_course_registration.sql"

def generate_student_registration():
    sql_lines = [
        "-- STUDENT COURSE REGISTRATION SETUP",
        "SET FOREIGN_KEY_CHECKS = 0;",
        ""
    ]

    try:
        # 1. LOAD DATA
        # Ensure your CSV has headers like 'Reg. No.' and the specific course columns
        df_stud = pd.read_csv(STUDENT_FILE)
        
        # Defined based on your previous input for course headers
        course_cols = [
            'PROFESSIONAL ELECTIVE 1', '22XX401', '22XX402', '22XX403', 
            '22XX404', '22XX405', '22XX007', '22HS008', '22HS010'
        ]

        sql_lines.append("-- SECTION: INSERTING INTO student_courses")
        
        for _, row in df_stud.iterrows():
            reg_no = str(row['Reg. No.']).strip()
            
            for col in course_cols:
                val = str(row.get(col, ''))
                
                # Logic to handle format like '22AD401-COURSE NAME' or just '22AD401'
                if val and val.lower() != 'nan' and val.strip() != '':
                    # Extract the course code (the part before the hyphen)
                    c_code = val.split('-')[0].strip()
                    
                    # Generate the SQL to link the student ID to the course ID using lookups
                    sql_lines.append(
                        f"INSERT IGNORE INTO `student_courses` (student_id, course_id) "
                        f"SELECT s.id, c.id FROM `students` s, `courses` c "
                        f"WHERE s.register_no = '{reg_no}' AND c.course_code = '{c_code}';"
                    )

        sql_lines.append("\nSET FOREIGN_KEY_CHECKS = 1;")

        # 2. WRITE TO FILE
        with open(OUTPUT_SQL, "w", encoding="utf-8") as f:
            f.write("\n".join(sql_lines))
        
        print(f"✅ Success! SQL generated in {OUTPUT_SQL}")
        print(f"👉 Run this file to populate the student_courses table.")

    except Exception as e:
        print(f"❌ Error: {e}")

if __name__ == "__main__":
    generate_student_registration()
