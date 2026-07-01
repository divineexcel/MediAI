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

// ─── Medical Record Repository ────────────────────────────────────────────────

type GORMMedicalRecordRepository struct {
	db *gorm.DB
}

func NewGORMMedicalRecordRepository(db *gorm.DB) domainrepo.MedicalRecordRepository {
	return &GORMMedicalRecordRepository{db: db}
}

func (r *GORMMedicalRecordRepository) Create(ctx context.Context, rec *entity.MedicalRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *GORMMedicalRecordRepository) FindByID(ctx context.Context, id uint) (*entity.MedicalRecord, error) {
	var rec entity.MedicalRecord
	err := r.db.WithContext(ctx).Preload("Patient.User").First(&rec, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrRecordNotFound
	}
	return &rec, err
}

func (r *GORMMedicalRecordRepository) ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.MedicalRecord, int64, error) {
	var list []*entity.MedicalRecord
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.MedicalRecord{}).Where("patient_id = ?", patientID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("record_date DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMMedicalRecordRepository) Update(ctx context.Context, rec *entity.MedicalRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

func (r *GORMMedicalRecordRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&entity.MedicalRecord{}, id).Error
}

// ─── Prescription Repository ──────────────────────────────────────────────────

type GORMPrescriptionRepository struct {
	db *gorm.DB
}

func NewGORMPrescriptionRepository(db *gorm.DB) domainrepo.PrescriptionRepository {
	return &GORMPrescriptionRepository{db: db}
}

func (r *GORMPrescriptionRepository) Create(ctx context.Context, p *entity.Prescription) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *GORMPrescriptionRepository) FindByID(ctx context.Context, id uint) (*entity.Prescription, error) {
	var p entity.Prescription
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &p, err
}

func (r *GORMPrescriptionRepository) ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.Prescription, int64, error) {
	var list []*entity.Prescription
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.Prescription{}).Where("patient_id = ?", patientID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMPrescriptionRepository) ListByConsultation(ctx context.Context, consultationID uint) ([]*entity.Prescription, error) {
	var list []*entity.Prescription
	err := r.db.WithContext(ctx).
		Where("consultation_id = ?", consultationID).
		Order("created_at ASC").Find(&list).Error
	return list, err
}

func (r *GORMPrescriptionRepository) MarkFilled(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&entity.Prescription{}).
		Where("id = ?", id).
		Update("is_filled", true).Error
}
