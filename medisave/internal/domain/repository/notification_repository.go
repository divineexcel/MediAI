package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	FindByID(ctx context.Context, id uint) (*entity.Notification, error)
	ListByUser(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Notification, int64, error)
	MarkRead(ctx context.Context, id uint) error
	MarkAllRead(ctx context.Context, userID uint) error
	CountUnread(ctx context.Context, userID uint) (int64, error)
	MarkSent(ctx context.Context, id uint) error
}
