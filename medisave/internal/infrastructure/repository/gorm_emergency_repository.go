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

type GORMEmergencyRepository struct {
	db *gorm.DB
}

func NewGORMEmergencyRepository(db *gorm.DB) domainrepo.EmergencyRepository {
	return &GORMEmergencyRepository{db: db}
}

func (r *GORMEmergencyRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMEmergencyRepository) Create(ctx context.Context, e *entity.Emergency) error {
	return r.dbc(ctx).Create(e).Error
}

func (r *GORMEmergencyRepository) FindByID(ctx context.Context, id uint) (*entity.Emergency, error) {
	var e entity.Emergency
	err := r.dbc(ctx).Preload("Patient").First(&e, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &e, err
}

func (r *GORMEmergencyRepository) Update(ctx context.Context, e *entity.Emergency) error {
	return r.dbc(ctx).Save(e).Error
}

func (r *GORMEmergencyRepository) ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.Emergency, int64, error) {
	var list []*entity.Emergency
	var total int64

	q := r.dbc(ctx).Model(&entity.Emergency{}).Where("patient_id = ?", patientID)
	q.Count(&total)
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMEmergencyRepository) ListActive(ctx context.Context) ([]*entity.Emergency, error) {
	var list []*entity.Emergency
	err := r.dbc(ctx).Preload("Patient").
		Where("status = ?", entity.EmergencyStatusActive).
		Order("created_at DESC").
		Find(&list).Error
	return list, err
}

func (r *GORMEmergencyRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.Emergency{}).Where("status = ?", entity.EmergencyStatusActive).Count(&count).Error
	return count, err
}

// ─── EMERGENCY CONTACTS ──────────────────────────────────────────────────────

type GORMEmergencyContactRepository struct {
	db *gorm.DB
}

func NewGORMEmergencyContactRepository(db *gorm.DB) domainrepo.EmergencyContactRepository {
	return &GORMEmergencyContactRepository{db: db}
}

func (r *GORMEmergencyContactRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMEmergencyContactRepository) Create(ctx context.Context, c *entity.EmergencyContact) error {
	return r.dbc(ctx).Create(c).Error
}

func (r *GORMEmergencyContactRepository) ListByPatient(ctx context.Context, patientID uint) ([]*entity.EmergencyContact, error) {
	var list []*entity.EmergencyContact
	err := r.dbc(ctx).
		Where("patient_id = ?", patientID).
		Order("is_primary DESC, created_at ASC").
		Find(&list).Error
	return list, err
}

func (r *GORMEmergencyContactRepository) FindByID(ctx context.Context, id uint) (*entity.EmergencyContact, error) {
	var c entity.EmergencyContact
	err := r.dbc(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &c, err
}

func (r *GORMEmergencyContactRepository) Update(ctx context.Context, c *entity.EmergencyContact) error {
	return r.dbc(ctx).Save(c).Error
}

func (r *GORMEmergencyContactRepository) Delete(ctx context.Context, id uint) error {
	return r.dbc(ctx).Delete(&entity.EmergencyContact{}, id).Error
}

func (r *GORMEmergencyContactRepository) SetPrimary(ctx context.Context, patientID uint, contactID uint) error {
	return r.dbc(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.EmergencyContact{}).
			Where("patient_id = ?", patientID).
			Update("is_primary", false).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&entity.EmergencyContact{}).
			Where("id = ? AND patient_id = ?", contactID, patientID).
			Updates(map[string]interface{}{"is_primary": true, "updated_at": now}).Error
	})
}
