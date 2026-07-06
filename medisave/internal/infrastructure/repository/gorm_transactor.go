package repository

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	domainrepo "github.com/medisave/app/internal/domain/repository"
	"github.com/medisave/app/pkg/logger"
)

type GORMTransactor struct {
	db *gorm.DB
}

func NewGORMTransactor(db *gorm.DB) domainrepo.Transactor {
	return &GORMTransactor{db: db}
}

func (t *GORMTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	logger.Info("database transaction: starting transaction block")
	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := fn(domainrepo.WithTransaction(ctx, tx))
		if err != nil {
			logger.Error("database transaction failed, rolling back", zap.Error(err))
		}
		return err
	})
	if err != nil {
		logger.Warn("database transaction: rolled back due to error", zap.Error(err))
		return err
	}
	logger.Info("database transaction: committed successfully")
	return nil
}
