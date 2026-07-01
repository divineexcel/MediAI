package database

import (
	"errors"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/infrastructure/database/migrations"
	"github.com/medisave/app/pkg/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "postgres":
		if cfg.URL == "" {
			return nil, errors.New("database URL is required for postgres driver")
		}
		dialector = postgres.Open(cfg.URL)
	case "sqlite", "":
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.Path)
	default:
		return nil, errors.New("unsupported database driver: " + cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if cfg.Driver == "postgres" {
		logger.Info("database connected", zap.String("driver", "postgres"))
	} else {
		// WAL mode: readers don't block writers — critical for concurrent API requests
		db.Exec("PRAGMA journal_mode=WAL")
		// Enforce foreign key constraints at DB level, not just application level
		db.Exec("PRAGMA foreign_keys=ON")
		// Improve write performance: sync only when SQLite deems necessary
		db.Exec("PRAGMA synchronous=NORMAL")
		// 64MB page cache
		db.Exec("PRAGMA cache_size=-65536")
		logger.Info("database connected", zap.String("driver", "sqlite"), zap.String("path", cfg.Path))
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	return migrations.Run(db)
}
