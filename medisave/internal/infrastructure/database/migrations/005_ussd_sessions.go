package migrations

import "gorm.io/gorm"

func migration005USSDSessions() Migration {
	return Migration{
		Version:     "005",
		Description: "create ussd_sessions table",
		Up: func(db *gorm.DB) error {
			return db.Exec(`
				CREATE TABLE IF NOT EXISTS ussd_sessions (
					id         INTEGER PRIMARY KEY AUTOINCREMENT,
					session_id TEXT    NOT NULL UNIQUE,
					phone      TEXT    NOT NULL,
					user_id    INTEGER NOT NULL DEFAULT 0,
					menu_state TEXT    NOT NULL DEFAULT 'home',
					prev_menu  TEXT    NOT NULL DEFAULT '',
					data       TEXT    NOT NULL DEFAULT '{}',
					expires_at DATETIME NOT NULL,
					created_at DATETIME NOT NULL,
					updated_at DATETIME NOT NULL
				);
				CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone ON ussd_sessions(phone);
				CREATE INDEX IF NOT EXISTS idx_ussd_sessions_expires ON ussd_sessions(expires_at);
			`).Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec(`DROP TABLE IF EXISTS ussd_sessions`).Error
		},
	}
}
