package db

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func InitDB() error {
	var err error
	// Use only Aiven database - build DSN from env vars
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	if host == "" || port == "" || user == "" || name == "" {
		return fmt.Errorf("missing required database environment variables (DB_HOST, DB_PORT, DB_USER, DB_NAME)")
	}

	// Setup TLS for Aiven
	rootCertPool := x509.NewCertPool()
	caPEM := os.Getenv("DB_CA_CERT")
	var dsn string

	if caPEM != "" {
		// Format the PEM certificate correctly by adding newlines
		// Replace escaped newlines if present
		caPEM = strings.ReplaceAll(caPEM, "\\n", "\n")

		// If certificate is on a single line, format it properly
		if !strings.Contains(caPEM, "\n") {
			// Add newline after BEGIN CERTIFICATE
			caPEM = strings.Replace(caPEM, "-----BEGIN CERTIFICATE-----", "-----BEGIN CERTIFICATE-----\n", 1)
			// Add newline before END CERTIFICATE
			caPEM = strings.Replace(caPEM, "-----END CERTIFICATE-----", "\n-----END CERTIFICATE-----", 1)
			// Split the certificate body into 64-character lines
			parts := strings.SplitN(caPEM, "\n", 2)
			if len(parts) == 2 {
				middle := strings.SplitN(parts[1], "\n", 2)
				if len(middle) == 2 {
					certBody := middle[0]
					var formattedBody strings.Builder
					for i := 0; i < len(certBody); i += 64 {
						end := i + 64
						if end > len(certBody) {
							end = len(certBody)
						}
						formattedBody.WriteString(certBody[i:end])
						formattedBody.WriteString("\n")
					}
					caPEM = parts[0] + "\n" + formattedBody.String() + middle[1]
				}
			}
		}

		pemBytes := []byte(caPEM)

		if rootCertPool.AppendCertsFromPEM(pemBytes) {
			cfg := &tls.Config{RootCAs: rootCertPool}
			tlsName := "aiven"
			if err := mysql.RegisterTLSConfig(tlsName, cfg); err != nil {
				log.Fatal("Failed to register TLS config:", err)
				return err
			}
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local&tls=%s", user, pass, host, port, name, tlsName)
		} else {
			return fmt.Errorf("failed to parse DB_CA_CERT")
		}
	} else {
		// Fallback without TLS (not recommended for production)
		log.Println("WARNING: Connecting without TLS certificate")
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", user, pass, host, port, name)
	}

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
		return err
	}

	// Test the connection
	err = DB.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
		return err
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		log.Printf("Warning: Migrations failed: %v", err)
	}

	return nil
}

func runMigrations() error {
	// Skip course_type table migrations - use existing table structure
	// The course_type table already exists with column 'course_type', not 'name'
	log.Println("Skipping course_type migrations - using existing table structure")
	
	// Just ensure the table exists with basic structure
	var tableExists int
	err := DB.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'course_type'").Scan(&tableExists)
	if err == nil && tableExists == 0 {
		// Create course_type table only if it doesn't exist
		_, err = DB.Exec(`
			CREATE TABLE IF NOT EXISTS course_type (
				id INT PRIMARY KEY AUTO_INCREMENT,
				course_type VARCHAR(50) NOT NULL UNIQUE
			)
		`)
		if err != nil {
			return err
		}
		
		// Insert initial values
		_, err = DB.Exec(`
			INSERT IGNORE INTO course_type (id, course_type) VALUES 
			(1, 'theory'), 
			(2, 'lab'), 
			(3, 'theory with lab')
		`)
	}
	if err != nil {
		log.Printf("Initial values insert warning: %v", err)
	}

	// Create teacher_course_limits table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS teacher_course_limits (
			id INT PRIMARY KEY AUTO_INCREMENT,
			teacher_id BIGINT UNSIGNED NOT NULL,
			course_type_id INT NOT NULL,
			max_count INT DEFAULT 0,
			UNIQUE KEY (teacher_id, course_type_id),
			FOREIGN KEY (course_type_id) REFERENCES course_type(id)
		)
	`)
	if err != nil {
		return err
	}

	// Create student_enrollments table for allocation calculations
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS student_enrollments (
			id INT NOT NULL AUTO_INCREMENT,
			student_id INT NOT NULL,
			course_id INT NOT NULL,
			academic_year VARCHAR(20) NOT NULL COMMENT 'e.g., "2025-2026"',
			semester INT NOT NULL COMMENT 'Semester number 1-8',
			enrollment_status VARCHAR(50) DEFAULT 'enrolled' COMMENT 'enrolled, dropped, completed',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_student_id (student_id),
			KEY idx_course_id (course_id),
			KEY idx_academic_year_semester (academic_year, semester),
			KEY idx_student_course_year_sem (student_id, course_id, academic_year, semester),
			CONSTRAINT fk_se_student FOREIGN KEY (student_id) REFERENCES students (id) ON DELETE CASCADE ON UPDATE CASCADE,
			CONSTRAINT fk_se_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE ON UPDATE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Populate student_enrollments based on existing curriculum structure
	_, err = DB.Exec(`
		INSERT IGNORE INTO student_enrollments (student_id, course_id, academic_year, semester, enrollment_status)
		SELECT DISTINCT
			ad.student_id,
			cc.course_id,
			'2025-2026' as academic_year,
			nc.semester_number as semester,
			'enrolled' as enrollment_status
		FROM academic_details ad
		JOIN curriculum c ON ad.curriculum_id = c.id
		JOIN normal_cards nc ON nc.curriculum_id = c.id
		JOIN curriculum_courses cc ON cc.curriculum_id = c.id AND cc.semester_id = nc.id
		WHERE ad.student_id IS NOT NULL
		  AND nc.semester_number IS NOT NULL
		  AND nc.card_type = 'semester'
		  AND cc.course_id IS NOT NULL
	`)
	if err != nil {
		log.Printf("Warning: Student enrollments population failed: %v", err)
	}

	// Create student_eligible_honour_minor table
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS student_eligible_honour_minor (
			id INT AUTO_INCREMENT PRIMARY KEY,
			student_email VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_student_email (student_email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		log.Printf("Warning: Failed to create student_eligible_honour_minor table: %v", err)
	} else {
		log.Println("student_eligible_honour_minor table created/verified successfully")
	}

	return nil
}

// CreateCurriculumTable creates the curriculum table if it doesn't exist
func CreateCurriculumTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS curriculum (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		academic_year VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create curriculum table:", err)
		return err
	}

	fmt.Println("Curriculum table created/verified successfully!")
	return nil
}

// CreateCoursesTable creates the core courses table (using existing schema with course_id)
func CreateCoursesTable() error {
	// Check if table exists
	var tableExists bool
	err := DB.QueryRow("SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'courses'").Scan(&tableExists)

	if err == nil && tableExists {
		// Table already exists, verify it has course_id column
		var courseIdExists bool
		err = DB.QueryRow("SELECT COUNT(*) > 0 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'courses' AND column_name = 'course_id'").Scan(&courseIdExists)

		if courseIdExists {
			fmt.Println("Courses table verified successfully!")
			// Ensure all required columns exist
			_ = ensureColumnExists("courses", "course_code", "VARCHAR(50) NOT NULL")
			_ = ensureColumnExists("courses", "course_name", "VARCHAR(255) NOT NULL")
			_ = ensureColumnExists("courses", "course_type", "VARCHAR(50)")
			_ = ensureColumnExists("courses", "category", "VARCHAR(50)")
			_ = ensureColumnExists("courses", "credit", "INT")
			_ = ensureColumnExists("courses", "theory_hours", "INT")
			_ = ensureColumnExists("courses", "activity_hours", "INT")
			_ = ensureColumnExists("courses", "lecture_hours", "INT")
			_ = ensureColumnExists("courses", "tutorial_hours", "INT")
			_ = ensureColumnExists("courses", "practical_hours", "INT")
			_ = ensureColumnExists("courses", "cia_marks", "INT")
			_ = ensureColumnExists("courses", "see_marks", "INT")
			_ = ensureColumnExists("courses", "total_marks", "INT")
			_ = ensureColumnExists("courses", "total_hours", "INT")
			_ = ensureColumnExists("courses", "status", "TINYINT(1) DEFAULT 1")
			// Set existing NULL status rows to 1 (active)
			DB.Exec("UPDATE courses SET status = 1 WHERE status IS NULL")
			return nil
		}
	}

	// Create table if it doesn't exist (using course_id to match existing schema)
	query := `
	CREATE TABLE IF NOT EXISTS courses (
		course_id INT AUTO_INCREMENT PRIMARY KEY,
		course_code VARCHAR(50) NOT NULL,
		course_name VARCHAR(255) NOT NULL,
		course_type VARCHAR(50),
		category VARCHAR(50),
		credit INT,
		theory_hours INT,
		activity_hours INT,
		lecture_hours INT,
		tutorial_hours INT,
		practical_hours INT,
		cia_marks INT,
		see_marks INT,
		total_marks INT,
		total_hours INT,
		status TINYINT(1) DEFAULT 1,
		UNIQUE KEY unique_course_code (course_code)
	) ENGINE=InnoDB
	`

	_, err = DB.Exec(query)
	if err != nil {
		log.Println("Error creating courses table:", err)
		return err
	}

	fmt.Println("Courses table created/verified successfully!")
	return nil
}

// CreateCurriculumCoursesTable creates the junction table linking courses to curriculum semesters
func CreateCurriculumCoursesTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS curriculum_courses (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		semester_id INT NOT NULL,
		course_id INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE,
		UNIQUE KEY unique_course_semester (curriculum_id, semester_id, course_id)
	) ENGINE=InnoDB
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create curriculum_courses table:", err)
		return err
	}

	fmt.Println("Curriculum courses table created/verified successfully!")
	return nil
}

// CreateCurriculumLogsTable creates the curriculum_logs table if it doesn't exist
func CreateCurriculumLogsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS curriculum_logs (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		action VARCHAR(255) NOT NULL,
		description TEXT,
		changed_by VARCHAR(255) DEFAULT 'System',
		diff JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curriculum_id) REFERENCES curriculum(id) ON DELETE CASCADE
	)
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Failed to create curriculum_logs table:", err)
		return err
	}

	// Add diff column if it doesn't exist (for existing tables)
	alterQuery := `
	ALTER TABLE curriculum_logs 
	ADD COLUMN IF NOT EXISTS diff JSON AFTER changed_by
	`
	DB.Exec(alterQuery) // Ignore errors as column may already exist

	fmt.Println("Curriculum logs table created/verified successfully!")
	return nil
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// CreateClusterTables creates cluster management tables
func CreateClusterTables() error {
	// Create clusters table
	clustersQuery := `
	CREATE TABLE IF NOT EXISTS clusters (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`
	_, err := DB.Exec(clustersQuery)
	if err != nil {
		log.Fatal("Failed to create clusters table:", err)
		return err
	}

	// Create cluster_departments mapping table
	clusterDepartmentsQuery := `
	CREATE TABLE IF NOT EXISTS cluster_departments (
		id INT AUTO_INCREMENT PRIMARY KEY,
		cluster_id INT NOT NULL,
		department_id INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE CASCADE,
		UNIQUE KEY unique_department (department_id)
	)
	`
	_, err = DB.Exec(clusterDepartmentsQuery)
	if err != nil {
		log.Fatal("Failed to create cluster_departments table:", err)
		return err
	}

	return nil
}

// CreateDepartmentOverviewTables creates department overview and related tables
func CreateDepartmentOverviewTables() error {
	// Main curriculum_vision table
	overviewQuery := `
	CREATE TABLE IF NOT EXISTS curriculum_vision (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		vision TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY unique_regulation (curriculum_id)
	)
	`
	_, err := DB.Exec(overviewQuery)
	if err != nil {
		log.Fatal("Failed to create curriculum_vision table:", err)
		return err
	}

	// Mission table
	missionQuery := `
	CREATE TABLE IF NOT EXISTS curriculum_mission (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		mission_text TEXT NOT NULL,
		visibility ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE',
		position INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curriculum_id) REFERENCES curriculum_vision(id) ON DELETE CASCADE
	)
	`
	_, err = DB.Exec(missionQuery)
	if err != nil {
		log.Fatal("Failed to create curriculum_mission table:", err)
		return err
	}

	// PEOs table
	peosQuery := `
	CREATE TABLE IF NOT EXISTS curriculum_peos (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		peo_text TEXT NOT NULL,
		visibility ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE',
		position INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curriculum_id) REFERENCES curriculum_vision(id) ON DELETE CASCADE
	)
	`
	_, err = DB.Exec(peosQuery)
	if err != nil {
		log.Fatal("Failed to create curriculum_peos table:", err)
		return err
	}

	// POs table
	posQuery := `
	CREATE TABLE IF NOT EXISTS curriculum_pos (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		po_text TEXT NOT NULL,
		visibility ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE',
		position INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curriculum_id) REFERENCES curriculum_vision(id) ON DELETE CASCADE
	)
	`
	_, err = DB.Exec(posQuery)
	if err != nil {
		log.Fatal("Failed to create curriculum_pos table:", err)
		return err
	}

	// PSOs table
	psosQuery := `
	CREATE TABLE IF NOT EXISTS curriculum_psos (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		pso_text TEXT NOT NULL,
		visibility ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE',
		position INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curriculum_id) REFERENCES curriculum_vision(id) ON DELETE CASCADE
	)
	`
	_, err = DB.Exec(psosQuery)
	if err != nil {
		log.Fatal("Failed to create curriculum_psos table:", err)
		return err
	}

	return nil
}

// AddVisibilityColumns adds visibility ENUM column to department data tables
func AddVisibilityColumns() error {
	tables := []string{"curriculum_mission", "curriculum_peos", "curriculum_pos", "curriculum_psos"}

	for _, table := range tables {
		err := ensureColumnExists(table, "visibility", "ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE'")
		if err != nil {
			log.Printf("Warning: Failed to add visibility column to %s: %v", table, err)
		}
	}

	return nil
}

// CreateCourseSyllabusTable - DEPRECATED: course_syllabus table removed in favor of course-centric design
// All syllabus data now references courses.id directly
func CreateCourseSyllabusTable() error {
	// Drop the legacy course_syllabus table if it exists
	_, _ = DB.Exec("DROP TABLE IF EXISTS course_syllabus")

	fmt.Println("Legacy course_syllabus table removed - using course-centric design")
	return nil
}

// CreateNormalizedSyllabusTables creates normalized tables for syllabus child data
func CreateNormalizedSyllabusTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS course_objectives (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			objective TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_outcomes (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			outcome TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_references (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			reference_text TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_prerequisites (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			prerequisite TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_teamwork (
			course_id INT NOT NULL PRIMARY KEY,
			total_hours INT NOT NULL DEFAULT 0,
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_teamwork_activities (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			activity TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_selflearning (
			course_id INT NOT NULL PRIMARY KEY,
			total_hours INT NOT NULL DEFAULT 0,
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_selflearning_topics (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			main_text TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_course_position (course_id, position),
			FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS course_selflearning_resources (
			id INT AUTO_INCREMENT PRIMARY KEY,
			main_id INT NOT NULL,
			internal_text TEXT NOT NULL,
			position INT NOT NULL,
			UNIQUE KEY unique_main_position (main_id, position),
			FOREIGN KEY (main_id) REFERENCES course_selflearning_topics(id) ON DELETE CASCADE
		)`,
	}

	for _, createSQL := range tables {
		if _, err := DB.Exec(createSQL); err != nil {
			log.Println("Error creating normalized table:", err)
			return err
		}
	}

	// Ensure status columns exist for tables where we soft-delete
	_ = ensureColumnExists("course_references", "status", "TINYINT(1) DEFAULT 1")
	_ = ensureColumnExists("course_outcomes", "status", "TINYINT(1) DEFAULT 1")
	_ = ensureColumnExists("course_objectives", "status", "TINYINT(1) DEFAULT 1")

	fmt.Println("Normalized syllabus tables created/verified successfully!")
	return nil
}

// CreateSyllabusRelationalTables creates models, titles, topics tables with cascades
// All tables now reference course_id directly (course-centric design)
func CreateSyllabusRelationalTables() error {
	// models - references course_id directly
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS syllabus (
			id INT AUTO_INCREMENT PRIMARY KEY,
			course_id INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			model_name VARCHAR(255) NOT NULL,
			position INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_models_course FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
		) ENGINE=InnoDB`); err != nil {
		return err
	}

	// Drop legacy syllabus_id column if exists
	_ = dropColumnIfExists("syllabus", "syllabus_id")

	// Ensure required columns exist for legacy schemas
	_ = ensureColumnExists("syllabus", "course_id", "INT")
	_ = ensureColumnExists("syllabus", "name", "VARCHAR(255) NOT NULL DEFAULT ''")
	_ = ensureColumnExists("syllabus", "model_name", "VARCHAR(255) NOT NULL DEFAULT ''")
	_ = ensureColumnExists("syllabus", "position", "INT DEFAULT 0")
	_ = ensureColumnExists("syllabus", "status", "TINYINT(1) DEFAULT 1")
	// Index for filtering by course_id
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_models_course ON syllabus(course_id)")
	// titles
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS syllabus_titles (
			id INT AUTO_INCREMENT PRIMARY KEY,
			model_id INT NOT NULL,
				title VARCHAR(512) NOT NULL,
				title_name VARCHAR(512) NOT NULL,
			hours INT DEFAULT 0,
			position INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_titles_model FOREIGN KEY (model_id) REFERENCES syllabus(id) ON DELETE CASCADE
		) ENGINE=InnoDB`); err != nil {
		return err
	}
	_ = ensureColumnExists("syllabus_titles", "model_id", "INT")
	_ = ensureColumnExists("syllabus_titles", "title", "VARCHAR(512) NOT NULL")
	_ = ensureColumnExists("syllabus_titles", "title_name", "VARCHAR(512) NOT NULL DEFAULT ''")
	_ = ensureColumnExists("syllabus_titles", "hours", "INT DEFAULT 0")
	_ = ensureColumnExists("syllabus_titles", "position", "INT DEFAULT 0")
	_ = ensureColumnExists("syllabus_titles", "status", "TINYINT(1) DEFAULT 1")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_titles_model ON syllabus_titles(model_id)")
	// topics
	if _, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS syllabus_topics (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title_id INT NOT NULL,
				topic VARCHAR(1024) NOT NULL,
				content TEXT NOT NULL,
			position INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_topics_title FOREIGN KEY (title_id) REFERENCES syllabus_titles(id) ON DELETE CASCADE
		) ENGINE=InnoDB`); err != nil {
		return err
	}
	_ = ensureColumnExists("syllabus_topics", "title_id", "INT")
	_ = ensureColumnExists("syllabus_topics", "topic", "VARCHAR(1024) NOT NULL DEFAULT ''")
	_ = ensureColumnExists("syllabus_topics", "content", "TEXT NOT NULL")
	_ = ensureColumnExists("syllabus_topics", "position", "INT DEFAULT 0")
	_ = ensureColumnExists("syllabus_topics", "status", "TINYINT(1) DEFAULT 1")
	_, _ = DB.Exec("CREATE INDEX IF NOT EXISTS idx_topics_title ON syllabus_topics(title_id)")
	fmt.Println("Syllabus relational tables created/verified successfully!")
	return nil
}
func ensureColumnExists(table, column, colType string) error {
	// First try IF NOT EXISTS (MySQL 8.0.29+)
	alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s", table, column, colType)
	if _, err := DB.Exec(alter); err == nil {
		fmt.Println("Ensured column", column, "on", table)
		return nil
	}
	// Check if column already exists
	exists, err := columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		// already present
		return nil
	}
	// Fallback: add without IF NOT EXISTS
	alter2 := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType)
	if _, err := DB.Exec(alter2); err != nil {
		return err
	}
	return nil
}

func columnExists(table, column string) (bool, error) {
	// Some MySQL servers/drivers don't allow placeholders in SHOW statements
	q := fmt.Sprintf("SHOW COLUMNS FROM %s LIKE '%s'", table, column)
	rows, err := DB.Query(q)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

// dropColumnIfExists drops a column only if it exists.
func dropColumnIfExists(table, column string) error {
	exists, err := columnExists(table, column)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	q := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column)
	if _, err := DB.Exec(q); err != nil {
		return err
	}
	fmt.Println("Dropped column", column, "from", table)
	return nil
}

// AddVisibilitySemestersCourses adds visibility columns to normal_cards and courses tables
func AddVisibilitySemestersCourses() error {
	// Add visibility to normal_cards table
	if err := ensureColumnExists("normal_cards", "visibility", "ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE'"); err != nil {
		return fmt.Errorf("failed to add visibility to normal_cards: %w", err)
	}

	// Add visibility to courses table
	if err := ensureColumnExists("courses", "visibility", "ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE'"); err != nil {
		return fmt.Errorf("failed to add visibility to courses: %w", err)
	}

	return nil
}

// AddSourceDepartmentColumns adds source_curriculum_id to track shared item origins
func AddSourceDepartmentColumns() error {
	// Add to department tables
	tables := []string{"curriculum_mission", "curriculum_peos", "curriculum_pos", "curriculum_psos"}
	for _, table := range tables {
		if err := ensureColumnExists(table, "source_curriculum_id", "INT DEFAULT NULL"); err != nil {
			return fmt.Errorf("failed to add source_curriculum_id to %s: %w", table, err)
		}
	}

	// Add to normal_cards
	if err := ensureColumnExists("normal_cards", "source_curriculum_id", "INT DEFAULT NULL"); err != nil {
		return fmt.Errorf("failed to add source_curriculum_id to normal_cards: %w", err)
	}

	// Add to courses (source_curriculum_id to track which regulation it came from)
	if err := ensureColumnExists("courses", "source_curriculum_id", "INT DEFAULT NULL"); err != nil {
		return fmt.Errorf("failed to add source_curriculum_id to courses: %w", err)
	}

	return nil
}

// CreateSharingTrackingTable creates a table to track shared items
func CreateSharingTrackingTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS sharing_tracking (
			id INT AUTO_INCREMENT PRIMARY KEY,
			source_curriculum_id INT NOT NULL,
			target_curriculum_id INT NOT NULL,
			item_type VARCHAR(50) NOT NULL,
			source_item_id INT NOT NULL,
			copied_item_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_source (source_curriculum_id, item_type, source_item_id),
			INDEX idx_target (target_curriculum_id, item_type),
			INDEX idx_copied (copied_item_id, item_type)
		)
	`

	if _, err := DB.Exec(query); err != nil {
		return fmt.Errorf("failed to create sharing_tracking table: %w", err)
	}

	return nil
}

// RemoveUniquePositionConstraints removes UNIQUE constraints on (curriculum_id, position)
// to allow soft deletes - multiple records can have same position if only one has status=1
func RemoveUniquePositionConstraints() error {
	tables := []string{"curriculum_mission", "curriculum_peos", "curriculum_pos", "curriculum_psos"}

	for _, table := range tables {
		// Check if the constraint exists
		var constraintExists bool
		checkQuery := `
			SELECT COUNT(*) > 0
			FROM information_schema.TABLE_CONSTRAINTS 
			WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ? 
			AND CONSTRAINT_NAME = 'department_id'
		`
		err := DB.QueryRow(checkQuery, table).Scan(&constraintExists)
		if err != nil {
			log.Printf("Error checking constraint for %s: %v\n", table, err)
			continue
		}

		if constraintExists {
			dropQuery := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `department_id`", table)
			_, err := DB.Exec(dropQuery)
			if err != nil {
				log.Printf("Error dropping constraint for %s: %v\n", table, err)
				continue
			}
			fmt.Printf("Dropped UNIQUE constraint from %s\n", table)
		}
	}

	fmt.Println("UNIQUE position constraints removed successfully!")
	return nil
}

// CreateRegulationTables creates the regulation management tables
func CreateRegulationTables() error {
	// Create regulations table
	regulationsTable := `
		CREATE TABLE IF NOT EXISTS regulations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			code VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			status ENUM('DRAFT','PUBLISHED','LOCKED') DEFAULT 'DRAFT',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_code (code),
			INDEX idx_status (status)
		) ENGINE=InnoDB
	`
	if _, err := DB.Exec(regulationsTable); err != nil {
		return fmt.Errorf("failed to create regulations table: %w", err)
	}

	// Create regulation_sections table
	sectionsTable := `
		CREATE TABLE IF NOT EXISTS regulation_sections (
			id INT AUTO_INCREMENT PRIMARY KEY,
			regulation_id INT NOT NULL,
			section_no INT NOT NULL,
			title VARCHAR(255) NOT NULL,
			display_order INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (regulation_id) REFERENCES regulations(id) ON DELETE CASCADE,
			UNIQUE KEY unique_section (regulation_id, section_no),
			INDEX idx_regulation (regulation_id),
			INDEX idx_order (regulation_id, display_order)
		) ENGINE=InnoDB
	`
	if _, err := DB.Exec(sectionsTable); err != nil {
		return fmt.Errorf("failed to create regulation_sections table: %w", err)
	}

	// Create regulation_clauses table
	clausesTable := `
		CREATE TABLE IF NOT EXISTS regulation_clauses (
			id INT AUTO_INCREMENT PRIMARY KEY,
			regulation_id INT NOT NULL,
			section_id INT NOT NULL,
			section_no INT NOT NULL,
			clause_no VARCHAR(10) NOT NULL,
			title VARCHAR(255),
			content TEXT NOT NULL,
			display_order INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (regulation_id) REFERENCES regulations(id) ON DELETE CASCADE,
			FOREIGN KEY (section_id) REFERENCES regulation_sections(id) ON DELETE CASCADE,
			UNIQUE KEY unique_clause (regulation_id, section_no, clause_no),
			INDEX idx_regulation (regulation_id),
			INDEX idx_section (section_id),
			INDEX idx_order (section_id, display_order)
		) ENGINE=InnoDB
	`
	if _, err := DB.Exec(clausesTable); err != nil {
		return fmt.Errorf("failed to create regulation_clauses table: %w", err)
	}

	// Create regulation_clause_history table
	historyTable := `
		CREATE TABLE IF NOT EXISTS regulation_clause_history (
			id INT AUTO_INCREMENT PRIMARY KEY,
			clause_id INT NOT NULL,
			old_content TEXT NOT NULL,
			new_content TEXT NOT NULL,
			changed_by VARCHAR(255),
			changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			change_reason VARCHAR(500),
			FOREIGN KEY (clause_id) REFERENCES regulation_clauses(id) ON DELETE CASCADE,
			INDEX idx_clause (clause_id),
			INDEX idx_changed_at (changed_at)
		) ENGINE=InnoDB
	`
	if _, err := DB.Exec(historyTable); err != nil {
		return fmt.Errorf("failed to create regulation_clause_history table: %w", err)
	}

	return nil
}

// AddRegulationRefColumns is deprecated - curriculum_ref_id has been removed
func AddRegulationRefColumns() error {
	// No-op: curriculum_ref_id columns have been removed from the schema
	return nil
}

// CreateHonourCardTables creates tables for honour cards and verticals
func CreateHonourCardTables() error {
	// Create honour_cards table
	honourCardsQuery := `
	CREATE TABLE IF NOT EXISTS honour_cards (
		id INT AUTO_INCREMENT PRIMARY KEY,
		curriculum_id INT NOT NULL,
		title VARCHAR(255) NOT NULL,
		number INT NOT NULL,
		visibility ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE',
		source_curriculum_id INT DEFAULT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_regulation (curriculum_id)
	) ENGINE=InnoDB
	`
	_, err := DB.Exec(honourCardsQuery)
	if err != nil {
		return fmt.Errorf("failed to create honour_cards table: %w", err)
	}

	// Ensure new columns exist for existing databases
	if err := ensureColumnExists("honour_cards", "visibility", "ENUM('UNIQUE', 'CLUSTER') DEFAULT 'UNIQUE'"); err != nil {
		return fmt.Errorf("failed to add visibility to honour_cards: %w", err)
	}
	if err := ensureColumnExists("honour_cards", "source_curriculum_id", "INT DEFAULT NULL"); err != nil {
		return fmt.Errorf("failed to add source_curriculum_id to honour_cards: %w", err)
	}

	// Create honour_verticals table
	honourVerticalsQuery := `
	CREATE TABLE IF NOT EXISTS honour_verticals (
		id INT AUTO_INCREMENT PRIMARY KEY,
		honour_card_id INT NOT NULL,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (honour_card_id) REFERENCES honour_cards(id) ON DELETE CASCADE,
		INDEX idx_honour_card (honour_card_id)
	) ENGINE=InnoDB
	`
	_, err = DB.Exec(honourVerticalsQuery)
	if err != nil {
		return fmt.Errorf("failed to create honour_verticals table: %w", err)
	}

	// Create honour_vertical_courses table (junction table for courses in verticals)
	honourVerticalCoursesQuery := `
	CREATE TABLE IF NOT EXISTS honour_vertical_courses (
		id INT AUTO_INCREMENT PRIMARY KEY,
		honour_vertical_id INT NOT NULL,
		course_id INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (honour_vertical_id) REFERENCES honour_verticals(id) ON DELETE CASCADE,
		FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE,
		UNIQUE KEY unique_course_vertical (honour_vertical_id, course_id),
		INDEX idx_vertical (honour_vertical_id)
	) ENGINE=InnoDB
	`
	_, err = DB.Exec(honourVerticalCoursesQuery)
	if err != nil {
		return fmt.Errorf("failed to create honour_vertical_courses table: %w", err)
	}

	return nil
}

// RenameSemestersToNormalCards renames the semesters table to normal_cards
func RenameSemestersToNormalCards() error {
	// Check if semesters table exists
	var semestersExists bool
	err := DB.QueryRow("SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'semesters'").Scan(&semestersExists)
	if err != nil {
		return fmt.Errorf("failed to check if semesters table exists: %w", err)
	}

	// Check if normal_cards table exists
	var normalCardsExists bool
	err = DB.QueryRow("SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'normal_cards'").Scan(&normalCardsExists)
	if err != nil {
		return fmt.Errorf("failed to check if normal_cards table exists: %w", err)
	}

	// Only rename if semesters exists and normal_cards doesn't
	if semestersExists && !normalCardsExists {
		_, err = DB.Exec("RENAME TABLE semesters TO normal_cards")
		if err != nil {
			return fmt.Errorf("failed to rename semesters to normal_cards: %w", err)
		}
		fmt.Println("Successfully renamed semesters table to normal_cards!")
	} else if normalCardsExists {
		fmt.Println("normal_cards table already exists, skipping rename")
	} else {
		// semesters table doesn't exist, nothing to rename
		return nil
	}

	return nil
}

// AddSemesterNameColumn adds a name column to normal_cards table
func AddSemesterNameColumn() error {
	// Add name column to normal_cards	// Add name column to normal_cards
	if err := ensureColumnExists("normal_cards", "name", "VARCHAR(255) DEFAULT NULL"); err != nil {
		return fmt.Errorf("failed to add name to normal_cards: %w", err)
	}

	// Add card_type column to normal_cards
	if err := ensureColumnExists("normal_cards", "card_type", "VARCHAR(50) DEFAULT 'semester'"); err != nil {
		return fmt.Errorf("failed to add card_type to normal_cards: %w", err)
	}

	// Make semester_number nullable
	modifyQuery := "ALTER TABLE normal_cards MODIFY COLUMN semester_number INT NULL"
	_, err := DB.Exec(modifyQuery)
	if err != nil {
		log.Println("Note: semester_number column modification might have failed (may already be nullable):", err)
	}

	return nil
}

// RemoveNameColumnFromNormalCards drops the name column from normal_cards table
func RemoveNameColumnFromNormalCards() error {
	// Check if column exists first
	var columnExists bool
	err := DB.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM information_schema.columns 
		WHERE table_schema = DATABASE() 
		AND table_name = 'normal_cards' 
		AND column_name = 'name'
	`).Scan(&columnExists)

	if err != nil {
		return fmt.Errorf("failed to check if name column exists: %w", err)
	}

	if !columnExists {
		fmt.Println("Name column doesn't exist in normal_cards, skipping removal")
		return nil
	}

	// Drop the name column
	_, err = DB.Exec("ALTER TABLE normal_cards DROP COLUMN name")
	if err != nil {
		return fmt.Errorf("failed to drop name column from normal_cards: %w", err)
	}

	fmt.Println("Successfully removed name column from normal_cards!")
	return nil
}

// CreateMarkEntryStudentPermissionsTables creates tables for student-specific mark entry permissions
func CreateMarkEntryStudentPermissionsTables() error {
	// Create mark_entry_student_permissions table for student-specific access control
	// Student assignments always have a window - either existing or newly created
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS mark_entry_student_permissions (
			id INT AUTO_INCREMENT PRIMARY KEY,
			window_id INT NOT NULL,
			user_id VARCHAR(100) NOT NULL,
			student_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_by VARCHAR(100),
			UNIQUE KEY unique_permission (window_id, user_id, student_id),
			INDEX idx_window_user (window_id, user_id),
			INDEX idx_student (student_id),
			INDEX idx_user (user_id),
			INDEX idx_user_student (user_id, student_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`

	_, err := DB.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create mark_entry_student_permissions table: %w", err)
	}

	fmt.Println("Successfully created mark_entry_student_permissions table!")
	return nil
}

// AddUserIdToMarkEntryWindows adds user_id column to mark_entry_windows to support user-based windows
func AddUserIdToMarkEntryWindows() error {
	// Check if user_id column already exists
	var columnExists bool
	err := DB.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM information_schema.columns 
		WHERE table_schema = DATABASE() 
		AND table_name = 'mark_entry_windows' 
		AND column_name = 'user_id'
	`).Scan(&columnExists)

	if err != nil {
		return fmt.Errorf("failed to check if user_id column exists: %w", err)
	}

	if columnExists {
		fmt.Println("user_id column already exists in mark_entry_windows")
		return nil
	}

	// Add user_id column - NULL for teacher windows, populated for user windows
	_, err = DB.Exec(`
		ALTER TABLE mark_entry_windows 
		ADD COLUMN user_id VARCHAR(100) NULL AFTER teacher_id,
		ADD INDEX idx_user_lookup (user_id, department_id, semester, course_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to add user_id column to mark_entry_windows: %w", err)
	}

	fmt.Println("Successfully added user_id column to mark_entry_windows!")
	return nil
}
