package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type GORMReviewRepository struct {
	db *gorm.DB
}

func NewGORMReviewRepository(db *gorm.DB) domainrepo.ReviewRepository {
	return &GORMReviewRepository{db: db}
}

func (r *GORMReviewRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMReviewRepository) Create(ctx context.Context, rev *entity.Review) error {
	return r.dbc(ctx).Create(rev).Error
}

func (r *GORMReviewRepository) FindByID(ctx context.Context, id uint) (*entity.Review, error) {
	var rev entity.Review
	err := r.dbc(ctx).Preload("Patient.User").First(&rev, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrReviewNotFound
	}
	return &rev, err
}

func (r *GORMReviewRepository) FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.Review, error) {
	var rev entity.Review
	err := r.dbc(ctx).Where("appointment_id = ?", appointmentID).First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrReviewNotFound
	}
	return &rev, err
}

func (r *GORMReviewRepository) ListByDoctor(ctx context.Context, doctorID uint, p pagination.Params) ([]*entity.Review, int64, error) {
	var list []*entity.Review
	var total int64

	q := r.dbc(ctx).Model(&entity.Review{}).
		Where("doctor_id = ? AND is_visible = true", doctorID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Patient.User").
		Order("created_at DESC").
		Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMReviewRepository) AverageRatingByDoctor(ctx context.Context, doctorID uint) (float64, int, error) {
	var result struct {
		Avg   float64
		Count int
	}
	err := r.dbc(ctx).Model(&entity.Review{}).
		Select("COALESCE(AVG(rating), 0) as avg, COUNT(*) as count").
		Where("doctor_id = ? AND is_visible = true", doctorID).
		Scan(&result).Error
	return result.Avg, result.Count, err
}
