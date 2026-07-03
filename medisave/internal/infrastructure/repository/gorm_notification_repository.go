package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type GORMNotificationRepository struct {
	db *gorm.DB
}

func NewGORMNotificationRepository(db *gorm.DB) domainrepo.NotificationRepository {
	return &GORMNotificationRepository{db: db}
}

func (r *GORMNotificationRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMNotificationRepository) Create(ctx context.Context, n *entity.Notification) error {
	return r.dbc(ctx).Create(n).Error
}

func (r *GORMNotificationRepository) FindByID(ctx context.Context, id uint) (*entity.Notification, error) {
	var n entity.Notification
	err := r.dbc(ctx).First(&n, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &n, err
}

func (r *GORMNotificationRepository) ListByUser(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Notification, int64, error) {
	var items []*entity.Notification
	var total int64

	q := r.dbc(ctx).Model(&entity.Notification{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&items).Error
	return items, total, err
}

func (r *GORMNotificationRepository) MarkRead(ctx context.Context, id uint) error {
	return r.dbc(ctx).Model(&entity.Notification{}).
		Where("id = ?", id).Update("is_read", true).Error
}

func (r *GORMNotificationRepository) MarkAllRead(ctx context.Context, userID uint) error {
	return r.dbc(ctx).Model(&entity.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *GORMNotificationRepository) CountUnread(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

func (r *GORMNotificationRepository) MarkSent(ctx context.Context, id uint) error {
	now := time.Now()
	return r.dbc(ctx).Model(&entity.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"is_sent": true, "sent_at": now}).Error
}
