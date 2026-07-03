package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"go.uber.org/zap"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/logger"
)

type GORMUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) domainrepo.UserRepository {
	return &GORMUserRepository{db: db}
}

func (r *GORMUserRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMUserRepository) Create(ctx context.Context, user *entity.User) error {
	err := r.dbc(ctx).Create(user).Error
	if err != nil {
		logger.Error("GORMUserRepository.Create failed", zap.Error(err), zap.String("email", user.Email))
	}
	return err
}

func (r *GORMUserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	err := r.dbc(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.dbc(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) FindByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var user entity.User
	err := r.dbc(ctx).Where("phone = ?", phone).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) Update(ctx context.Context, user *entity.User) error {
	err := r.dbc(ctx).Save(user).Error
	if err != nil {
		logger.Error("GORMUserRepository.Update failed", zap.Error(err), zap.Uint("user_id", user.ID))
	}
	return err
}

func (r *GORMUserRepository) UpdateFCMToken(ctx context.Context, userID uint, token string) error {
	err := r.dbc(ctx).
		Model(&entity.User{}).
		Where("id = ?", userID).
		Update("fcm_token", token).Error
	if err != nil {
		logger.Error("GORMUserRepository.UpdateFCMToken failed", zap.Error(err), zap.Uint("user_id", userID))
	}
	return err
}

func (r *GORMUserRepository) Delete(ctx context.Context, id uint) error {
	err := r.dbc(ctx).Delete(&entity.User{}, id).Error
	if err != nil {
		logger.Error("GORMUserRepository.Delete failed", zap.Error(err), zap.Uint("user_id", id))
	}
	return err
}

func (r *GORMUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *GORMUserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.User{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}
