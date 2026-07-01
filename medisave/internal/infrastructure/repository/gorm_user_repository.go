package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
)

type GORMUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) domainrepo.UserRepository {
	return &GORMUserRepository{db: db}
}

func (r *GORMUserRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GORMUserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) FindByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrUserNotFound
	}
	return &user, err
}

func (r *GORMUserRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *GORMUserRepository) UpdateFCMToken(ctx context.Context, userID uint, token string) error {
	return r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ?", userID).
		Update("fcm_token", token).Error
}

func (r *GORMUserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.User{}, id).Error
}

func (r *GORMUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *GORMUserRepository) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}
