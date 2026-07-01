package service

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type NotificationService interface {
	ListByUser(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Notification, int64, error)
	MarkRead(ctx context.Context, notifID uint, userID uint) error
	MarkAllRead(ctx context.Context, userID uint) error
	CountUnread(ctx context.Context, userID uint) (int64, error)
	// CreateAndSend is called by other services (wallet, appointment, etc.)
	CreateAndSend(ctx context.Context, n *entity.Notification) error
}

type notificationService struct {
	notifRepo repository.NotificationRepository
}

func NewNotificationService(notifRepo repository.NotificationRepository) NotificationService {
	return &notificationService{notifRepo: notifRepo}
}

func (s *notificationService) ListByUser(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Notification, int64, error) {
	return s.notifRepo.ListByUser(ctx, userID, p)
}

func (s *notificationService) MarkRead(ctx context.Context, notifID uint, userID uint) error {
	n, err := s.notifRepo.FindByID(ctx, notifID)
	if err != nil {
		return err
	}
	if n.UserID != userID {
		return pkgerrors.ErrForbidden
	}
	return s.notifRepo.MarkRead(ctx, notifID)
}

func (s *notificationService) MarkAllRead(ctx context.Context, userID uint) error {
	return s.notifRepo.MarkAllRead(ctx, userID)
}

func (s *notificationService) CountUnread(ctx context.Context, userID uint) (int64, error) {
	return s.notifRepo.CountUnread(ctx, userID)
}

func (s *notificationService) CreateAndSend(ctx context.Context, n *entity.Notification) error {
	// Store the notification; async push/SMS dispatch is wired in Steps 12–15
	return s.notifRepo.Create(ctx, n)
}
