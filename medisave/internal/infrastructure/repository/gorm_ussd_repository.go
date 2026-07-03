package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
)

type GORMUSSDRepository struct {
	db *gorm.DB
}

func NewGORMUSSDRepository(db *gorm.DB) domainrepo.USSDRepository {
	return &GORMUSSDRepository{db: db}
}

func (r *GORMUSSDRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMUSSDRepository) FindBySessionID(ctx context.Context, sessionID string) (*entity.USSDSession, error) {
	var s entity.USSDSession
	err := r.dbc(ctx).Where("session_id = ?", sessionID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

func (r *GORMUSSDRepository) Upsert(ctx context.Context, session *entity.USSDSession) error {
	if session.ID == 0 {
		return r.dbc(ctx).Create(session).Error
	}
	return r.dbc(ctx).Save(session).Error
}

func (r *GORMUSSDRepository) DeleteExpired(ctx context.Context) error {
	return r.dbc(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entity.USSDSession{}).Error
}
