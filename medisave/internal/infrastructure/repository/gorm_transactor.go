package repository

import (
	"context"

	"gorm.io/gorm"

	domainrepo "github.com/medisave/app/internal/domain/repository"
)

type GORMTransactor struct {
	db *gorm.DB
}

func NewGORMTransactor(db *gorm.DB) domainrepo.Transactor {
	return &GORMTransactor{db: db}
}

func (t *GORMTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(domainrepo.WithTransaction(ctx, tx))
	})
}
