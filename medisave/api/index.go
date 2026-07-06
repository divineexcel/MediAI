package handler

import (
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/bootstrap"
	"github.com/medisave/app/internal/infrastructure/database"
	"github.com/medisave/app/internal/infrastructure/database/seed"
	"github.com/medisave/app/internal/presentation/http/router"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/logger"
)

var (
	ginEngine *gin.Engine
	once      sync.Once
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		// Initialize configurations and logger
		cfg := config.Load()
		logger.Init("info")

		logger.Info("starting MediSave on Vercel")

		// In actual Vercel cloud hosting, the only writable directory is /tmp.
		// If DB_DRIVER is sqlite and running on Vercel Cloud (where VERCEL=1 and NOW_REGION is set),
		// map the path to /tmp/medisave.db. For local vercel dev, keep using the persistent local path.
		if (cfg.Database.Driver == "sqlite" || cfg.Database.Driver == "") && os.Getenv("VERCEL") == "1" {
			if os.Getenv("NOW_REGION") != "" {
				cfg.Database.Path = "/tmp/medisave.db"
				logger.Warn("detected Vercel Cloud deployment — overriding SQLite path to ephemeral /tmp/medisave.db due to read-only filesystem restrictions", zap.String("path", cfg.Database.Path))
			} else {
				logger.Info("detected Vercel local dev environment — preserving persistent SQLite path", zap.String("path", cfg.Database.Path))
			}
		}

		// Connect database
		db, err := database.Connect(&cfg.Database)
		if err != nil {
			logger.Fatal("failed to connect to database", zap.Error(err))
		}

		// Run migrations
		if err := database.Migrate(db); err != nil {
			logger.Fatal("failed to run migrations", zap.Error(err))
		}

		// Seed database in development
		if cfg.App.Env == "development" {
			if err := seed.Run(db); err != nil {
				logger.Error("seeder failed", zap.Error(err))
			}
		}

		// Initialize JWT
		jwtManager := pkgjwt.NewManager(
			cfg.JWT.AccessSecret,
			cfg.JWT.RefreshSecret,
			cfg.JWT.AccessExpiryHours,
			cfg.JWT.RefreshExpiryDays,
		)

		// Create Router and Register Routes
		appRouter := router.New(cfg, jwtManager)
		bootstrap.RegisterRoutes(appRouter, db, cfg, jwtManager)
		ginEngine = appRouter.Engine()
	})

	ginEngine.ServeHTTP(w, r)
}
