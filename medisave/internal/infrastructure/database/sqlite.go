package database

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/infrastructure/database/migrations"
	"github.com/medisave/app/pkg/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	if cfg.Driver != "sqlite" && cfg.Driver != "" {
		return nil, fmt.Errorf("unsupported database driver %q — only sqlite is supported", cfg.Driver)
	}

	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, err
	}
	cfg.Path = absPath

	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
		return nil, err
	}

	// Show SQL errors in development; suppress verbose query logs in production.
	logLevel := gormlogger.Error
	if cfg.Env == "development" {
		logLevel = gormlogger.Warn // errors + slow queries
	}

	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		logger.Error("failed to open database connection", zap.String("path", cfg.Path), zap.Error(err))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to retrieve database handle", zap.Error(err))
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)
	logger.Info("configured SQLite connection pool",
		zap.Int("max_open_conns", 1),
		zap.Int("max_idle_conns", 1),
	)

	// Each PRAGMA is critical for correctness and performance.
	// Log a warning on failure rather than silently continuing.
	type pragma struct {
		sql  string
		desc string
	}
	pragmas := []pragma{
		// WAL mode: readers don't block writers — essential for concurrent API requests
		{"PRAGMA journal_mode=WAL", "set WAL journal mode"},
		// Enforce FK constraints at DB level, not just application level
		{"PRAGMA foreign_keys=ON", "enable foreign key enforcement"},
		// Improve write throughput: sync only when SQLite deems necessary
		{"PRAGMA synchronous=NORMAL", "set synchronous=NORMAL"},
		// 64 MB page cache
		{"PRAGMA cache_size=-65536", "set cache_size to 64 MB"},
		// Flush any outstanding WAL frames into the main DB file on startup
		{"PRAGMA wal_checkpoint(TRUNCATE)", "WAL checkpoint on startup"},
	}

	for _, p := range pragmas {
		if result := db.Exec(p.sql); result.Error != nil {
			logger.Warn("database pragma failed — check DB file permissions",
				zap.String("pragma", p.desc),
				zap.Error(result.Error),
			)
		}
	}

	logger.Info("database connected successfully",
		zap.String("driver", "sqlite"),
		zap.String("path", cfg.Path),
	)

	return db, nil
}

func Migrate(db *gorm.DB) error {
	logger.Info("starting database migrations")
	if err := migrations.Run(db); err != nil {
		logger.Error("database migrations failed", zap.Error(err))
		return err
	}
	logger.Info("database migrations completed successfully")
	return nil
}
