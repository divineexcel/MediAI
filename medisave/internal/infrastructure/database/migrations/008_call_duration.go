package migrations

import "gorm.io/gorm"

func migration008CallDuration() Migration {
	return Migration{
		Version:     "008",
		Description: "add call duration to appointments table",
		Up: func(db *gorm.DB) error {
			return db.Exec(`ALTER TABLE appointments ADD COLUMN call_duration INTEGER NOT NULL DEFAULT 0`).Error
		},
		Down: func(db *gorm.DB) error {
			return nil
		},
	}
}
