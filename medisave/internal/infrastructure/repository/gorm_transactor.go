package repository

import (
	"context"

	"gorm.io/gorm"
	"go.uber.org/zap"

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
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := fn(domainrepo.WithTransaction(ctx, tx))
		if err != nil {
			logger.Error("database transaction failed, rolling back", zap.Error(err))
		}
		return err
	})
}
