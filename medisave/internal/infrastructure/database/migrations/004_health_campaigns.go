package migrations

import (
	"gorm.io/gorm"
)

func migration004HealthCampaigns() Migration {
	return Migration{
		Version:     "004",
		Description: "create health campaigns table",
		Up: func(db *gorm.DB) error {
			return db.Exec(`
				CREATE TABLE IF NOT EXISTS health_campaigns (
					id           INTEGER PRIMARY KEY AUTOINCREMENT,
					title        TEXT NOT NULL,
					message      TEXT NOT NULL,
					category     TEXT NOT NULL,
					target_role  TEXT NOT NULL,
					location     TEXT NOT NULL,
					created_at   DATETIME NOT NULL,
					updated_at   DATETIME NOT NULL
				)
			`).Error
		},
		Down: func(db *gorm.DB) error {
			return db.Exec(`DROP TABLE IF EXISTS health_campaigns`).Error
		},
	}
}
