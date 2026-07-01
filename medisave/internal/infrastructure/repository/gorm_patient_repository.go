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

type GORMPatientRepository struct {
	db *gorm.DB
}

func NewGORMPatientRepository(db *gorm.DB) domainrepo.PatientRepository {
	return &GORMPatientRepository{db: db}
}

func (r *GORMPatientRepository) Create(ctx context.Context, patient *entity.Patient) error {
	return r.db.WithContext(ctx).Create(patient).Error
}

func (r *GORMPatientRepository) FindByID(ctx context.Context, id uint) (*entity.Patient, error) {
	var patient entity.Patient
	err := r.db.WithContext(ctx).Preload("User").First(&patient, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrPatientNotFound
	}
	return &patient, err
}

func (r *GORMPatientRepository) FindByUserID(ctx context.Context, userID uint) (*entity.Patient, error) {
	var patient entity.Patient
	err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&patient).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrPatientNotFound
	}
	return &patient, err
}

func (r *GORMPatientRepository) Update(ctx context.Context, patient *entity.Patient) error {
	return r.db.WithContext(ctx).Save(patient).Error
}

func (r *GORMPatientRepository) List(ctx context.Context, p pagination.Params) ([]*entity.Patient, int64, error) {
	var patients []*entity.Patient
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.Patient{}).Preload("User")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(p.Offset).Limit(p.Limit).Find(&patients).Error
	return patients, total, err
}

func (r *GORMPatientRepository) UpdateHealthScore(ctx context.Context, patientID uint, score int) error {
	return r.db.WithContext(ctx).
		Model(&entity.Patient{}).
		Where("id = ?", patientID).
		Update("health_score", score).Error
}

func (r *GORMPatientRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Patient{}).Count(&count).Error
	return count, err
}
