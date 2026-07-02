package migrations

import "gorm.io/gorm"

func migration006ConsultationRooms() Migration {
	return Migration{
		Version:     "006",
		Description: "create consultation_rooms table",
		Up: func(db *gorm.DB) error {
			return db.Exec(`
				CREATE TABLE IF NOT EXISTS consultation_rooms (
					id             INTEGER PRIMARY KEY AUTOINCREMENT,
					appointment_id INTEGER NOT NULL UNIQUE,
					room_name      TEXT    NOT NULL,
					status         TEXT    NOT NULL DEFAULT 'active',
					created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
					ended_at       DATETIME
				);
			`).Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec(`DROP TABLE IF EXISTS consultation_rooms`).Error
		},
	}
}
