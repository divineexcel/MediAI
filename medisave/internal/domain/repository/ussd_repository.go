package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
)

type USSDRepository interface {
	FindBySessionID(ctx context.Context, sessionID string) (*entity.USSDSession, error)
	Upsert(ctx context.Context, session *entity.USSDSession) error
	DeleteExpired(ctx context.Context) error
}
