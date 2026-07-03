package migrations

import "gorm.io/gorm"

func migration007DoctorRejectionStatus() Migration {
	return Migration{
		Version:     "007",
		Description: "add rejected status to check constraint and remarks column to doctors",
		Up: func(db *gorm.DB) error {
			// Disable foreign key constraints temporarily to allow recreating the table
			if err := db.Exec(`PRAGMA foreign_keys = OFF`).Error; err != nil {
				return err
			}

			// Create the new table structure with updated constraints and remarks column
			err := db.Exec(`
				CREATE TABLE doctors_new (
					id                   INTEGER PRIMARY KEY AUTOINCREMENT,
					user_id              INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					license_number       TEXT    UNIQUE NOT NULL,
					specialty            TEXT    NOT NULL,
					sub_specialty        TEXT,
					years_of_experience  INTEGER NOT NULL DEFAULT 0,
					hospital             TEXT,
					consultation_fee     REAL    NOT NULL DEFAULT 0,
					is_available         INTEGER NOT NULL DEFAULT 0,
					status               TEXT    NOT NULL DEFAULT 'pending'
					                     CHECK(status IN ('pending','verified','suspended','rejected')),
					remarks              TEXT    NOT NULL DEFAULT '',
					bio                  TEXT,
					education            TEXT,
					certifications       TEXT,
					languages            TEXT,
					rating               REAL    NOT NULL DEFAULT 0,
					total_reviews        INTEGER NOT NULL DEFAULT 0,
					total_consultations  INTEGER NOT NULL DEFAULT 0,
					created_at           DATETIME NOT NULL,
					updated_at           DATETIME NOT NULL,
					work_id_url          TEXT NOT NULL DEFAULT '',
					medical_license_url  TEXT NOT NULL DEFAULT ''
				);
			`).Error
			if err != nil {
				return err
			}

			// Copy existing doctor data into the new table, defaulting remarks to an empty string
			err = db.Exec(`
				INSERT INTO doctors_new (
					id, user_id, license_number, specialty, sub_specialty, years_of_experience,
					hospital, consultation_fee, is_available, status, remarks, bio, education,
					certifications, languages, rating, total_reviews, total_consultations,
					created_at, updated_at, work_id_url, medical_license_url
				) SELECT 
					id, user_id, license_number, specialty, sub_specialty, years_of_experience,
					hospital, consultation_fee, is_available, status, '', bio, education,
					certifications, languages, rating, total_reviews, total_consultations,
					created_at, updated_at, work_id_url, medical_license_url
				FROM doctors;
			`).Error
			if err != nil {
				return err
			}

			// Drop the old doctors table
			if err := db.Exec(`DROP TABLE doctors;`).Error; err != nil {
				return err
			}

			// Rename the new table to doctors
			if err := db.Exec(`ALTER TABLE doctors_new RENAME TO doctors;`).Error; err != nil {
				return err
			}

			// Re-enable foreign key constraints
			return db.Exec(`PRAGMA foreign_keys = ON;`).Error
		},
		Down: func(db *gorm.DB) error {
			return nil
		},
	}
}
