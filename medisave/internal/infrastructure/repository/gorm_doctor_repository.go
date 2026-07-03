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

type GORMDoctorRepository struct {
	db *gorm.DB
}

func NewGORMDoctorRepository(db *gorm.DB) domainrepo.DoctorRepository {
	return &GORMDoctorRepository{db: db}
}

func (r *GORMDoctorRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMDoctorRepository) Create(ctx context.Context, doctor *entity.Doctor) error {
	return r.dbc(ctx).Create(doctor).Error
}

func (r *GORMDoctorRepository) FindByID(ctx context.Context, id uint) (*entity.Doctor, error) {
	var doctor entity.Doctor
	err := r.dbc(ctx).Preload("User").First(&doctor, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrDoctorNotFound
	}
	return &doctor, err
}

func (r *GORMDoctorRepository) FindByUserID(ctx context.Context, userID uint) (*entity.Doctor, error) {
	var doctor entity.Doctor
	err := r.dbc(ctx).Preload("User").Where("user_id = ?", userID).First(&doctor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrDoctorNotFound
	}
	return &doctor, err
}

func (r *GORMDoctorRepository) FindByLicenseNumber(ctx context.Context, license string) (*entity.Doctor, error) {
	var doctor entity.Doctor
	err := r.dbc(ctx).Where("license_number = ?", license).First(&doctor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrDoctorNotFound
	}
	return &doctor, err
}

func (r *GORMDoctorRepository) Update(ctx context.Context, doctor *entity.Doctor) error {
	return r.dbc(ctx).Save(doctor).Error
}

func (r *GORMDoctorRepository) List(ctx context.Context, p pagination.Params) ([]*entity.Doctor, int64, error) {
	var doctors []*entity.Doctor
	var total int64

	q := r.dbc(ctx).Model(&entity.Doctor{}).
		Preload("User").
		Joins("JOIN users ON users.id = doctors.user_id").
		Where("users.is_active = ?", true)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(p.Offset).Limit(p.Limit).Order("rating DESC").Find(&doctors).Error
	return doctors, total, err
}

func (r *GORMDoctorRepository) ListVerified(ctx context.Context, p pagination.Params) ([]*entity.Doctor, int64, error) {
	var doctors []*entity.Doctor
	var total int64

	q := r.dbc(ctx).Model(&entity.Doctor{}).
		Preload("User").
		Joins("JOIN users ON users.id = doctors.user_id").
		Where("doctors.status = ? AND users.is_active = ?", entity.DoctorStatusVerified, true)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(p.Offset).Limit(p.Limit).Order("rating DESC").Find(&doctors).Error
	return doctors, total, err
}

func (r *GORMDoctorRepository) ListAvailable(ctx context.Context, specialty string, p pagination.Params) ([]*entity.Doctor, int64, error) {
	var doctors []*entity.Doctor
	var total int64

	q := r.dbc(ctx).Model(&entity.Doctor{}).
		Preload("User").
		Joins("JOIN users ON users.id = doctors.user_id").
		Where("doctors.status = ? AND doctors.is_available = ? AND users.is_active = ?",
			entity.DoctorStatusVerified, true, true)

	if specialty != "" {
		q = q.Where("doctors.specialty = ?", specialty)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(p.Offset).Limit(p.Limit).Order("doctors.rating DESC").Find(&doctors).Error
	return doctors, total, err
}

func (r *GORMDoctorRepository) SetAvailability(ctx context.Context, doctorID uint, available bool) error {
	return r.dbc(ctx).
		Model(&entity.Doctor{}).
		Where("id = ?", doctorID).
		Update("is_available", available).Error
}

func (r *GORMDoctorRepository) UpdateStatus(ctx context.Context, doctorID uint, status entity.DoctorStatus) error {
	return r.dbc(ctx).
		Model(&entity.Doctor{}).
		Where("id = ?", doctorID).
		Update("status", status).Error
}

func (r *GORMDoctorRepository) UpdateRating(ctx context.Context, doctorID uint, rating float64, totalReviews int) error {
	return r.dbc(ctx).
		Model(&entity.Doctor{}).
		Where("id = ?", doctorID).
		Updates(map[string]interface{}{
			"rating":        rating,
			"total_reviews": totalReviews,
		}).Error
}

func (r *GORMDoctorRepository) IncrementConsultations(ctx context.Context, doctorID uint) error {
	return r.dbc(ctx).
		Model(&entity.Doctor{}).
		Where("id = ?", doctorID).
		UpdateColumn("total_consultations", gorm.Expr("total_consultations + 1")).Error
}

func (r *GORMDoctorRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.Doctor{}).Count(&count).Error
	return count, err
}

func (r *GORMDoctorRepository) CountPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.Doctor{}).Where("status = ?", entity.DoctorStatusPending).Count(&count).Error
	return count, err
}

func (r *GORMDoctorRepository) FindAll(ctx context.Context) ([]*entity.Doctor, error) {
	var doctors []*entity.Doctor
	err := r.dbc(ctx).Preload("User").Find(&doctors).Error
	return doctors, err
}
