package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/infrastructure/database"
	"github.com/medisave/app/internal/infrastructure/database/seed"
	"github.com/medisave/app/internal/presentation/http/router"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/logger"
)

func main() {
	// Bootstrap
	cfg := config.Load()
	logger.Init("info")

	logger.Info("starting MediSave",
		zap.String("env", cfg.App.Env),
		zap.String("port", cfg.App.Port),
	)

	// Database
	db, err := database.Connect(cfg.Database.Path)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	if err := database.Migrate(db); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	if cfg.App.Env == "development" {
		if err := seed.Run(db); err != nil {
			logger.Error("seeder failed", zap.Error(err))
		}
	}

	// JWT
	jwtManager := pkgjwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiryHours,
		cfg.JWT.RefreshExpiryDays,
	)

	// Router
	r := router.New(cfg, jwtManager)

	// Register all module routes
	RegisterRoutes(r, db, cfg, jwtManager)

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r.Engine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start in goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}
