package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByPhone(ctx context.Context, phone string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	UpdateFCMToken(ctx context.Context, userID uint, token string) error
	Delete(ctx context.Context, id uint) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
}
