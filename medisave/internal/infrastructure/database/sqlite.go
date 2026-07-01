package database

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/medisave/app/internal/infrastructure/database/migrations"
	"github.com/medisave/app/pkg/logger"
)

func Connect(dbPath string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// WAL mode: readers don't block writers — critical for concurrent API requests
	db.Exec("PRAGMA journal_mode=WAL")
	// Enforce foreign key constraints at DB level, not just application level
	db.Exec("PRAGMA foreign_keys=ON")
	// Improve write performance: sync only when SQLite deems necessary
	db.Exec("PRAGMA synchronous=NORMAL")
	// 64MB page cache
	db.Exec("PRAGMA cache_size=-65536")

	logger.Info("database connected", zap.String("path", dbPath))
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return migrations.Run(db)
}
