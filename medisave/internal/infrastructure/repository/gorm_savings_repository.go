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

type GORMSavingsRepository struct {
	db *gorm.DB
}

func NewGORMSavingsRepository(db *gorm.DB) domainrepo.SavingsRepository {
	return &GORMSavingsRepository{db: db}
}

func (r *GORMSavingsRepository) Create(ctx context.Context, g *entity.HealthSavingsGoal) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *GORMSavingsRepository) FindByID(ctx context.Context, id uint) (*entity.HealthSavingsGoal, error) {
	var g entity.HealthSavingsGoal
	err := r.db.WithContext(ctx).First(&g, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &g, err
}

func (r *GORMSavingsRepository) ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.HealthSavingsGoal, int64, error) {
	var goals []*entity.HealthSavingsGoal
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.HealthSavingsGoal{}).Where("patient_id = ?", patientID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&goals).Error
	return goals, total, err
}

func (r *GORMSavingsRepository) Update(ctx context.Context, g *entity.HealthSavingsGoal) error {
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *GORMSavingsRepository) UpdateSavedAmount(ctx context.Context, id uint, amount float64) error {
	return r.db.WithContext(ctx).
		Model(&entity.HealthSavingsGoal{}).
		Where("id = ?", id).
		UpdateColumn("saved_amount", gorm.Expr("saved_amount + ?", amount)).Error
}

func (r *GORMSavingsRepository) UpdateStatus(ctx context.Context, id uint, status entity.SavingsGoalStatus) error {
	return r.db.WithContext(ctx).
		Model(&entity.HealthSavingsGoal{}).
		Where("id = ?", id).
		Update("status", status).Error
}
