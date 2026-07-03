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

		// In Vercel, the only writable directory is /tmp.
		// If DB_DRIVER is sqlite, map the path to /tmp/medisave.db to avoid read-only filesystem errors.
		if (cfg.Database.Driver == "sqlite" || cfg.Database.Driver == "") && os.Getenv("VERCEL") == "1" {
			cfg.Database.Path = "/tmp/medisave.db"
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
