package migrations

import "gorm.io/gorm"

func migration003DoctorDocuments() Migration {
	return Migration{
		Version:     "003",
		Description: "add document URL columns to doctors table",
		Up: func(db *gorm.DB) error {
			if db.Dialector.Name() == "postgres" {
				return nil
			}
			return db.Exec(`
				ALTER TABLE doctors ADD COLUMN work_id_url TEXT NOT NULL DEFAULT '';
				ALTER TABLE doctors ADD COLUMN medical_license_url TEXT NOT NULL DEFAULT '';
			`).Error
		},
		Down: func(db *gorm.DB) error {
			return nil // SQLite does not support DROP COLUMN easily
		},
	}
}
