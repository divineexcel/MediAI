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

type GORMReminderRepository struct {
	db *gorm.DB
}

func NewGORMReminderRepository(db *gorm.DB) domainrepo.ReminderRepository {
	return &GORMReminderRepository{db: db}
}

func (r *GORMReminderRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMReminderRepository) Create(ctx context.Context, rem *entity.MedicationReminder) error {
	return r.dbc(ctx).Create(rem).Error
}

func (r *GORMReminderRepository) FindByID(ctx context.Context, id uint) (*entity.MedicationReminder, error) {
	var rem entity.MedicationReminder
	err := r.dbc(ctx).First(&rem, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrReminderNotFound
	}
	return &rem, err
}

func (r *GORMReminderRepository) ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.MedicationReminder, int64, error) {
	var list []*entity.MedicationReminder
	var total int64

	q := r.dbc(ctx).Model(&entity.MedicationReminder{}).Where("patient_id = ?", patientID)
	q.Count(&total)
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMReminderRepository) ListActive(ctx context.Context) ([]*entity.MedicationReminder, error) {
	var list []*entity.MedicationReminder
	err := r.dbc(ctx).
		Where("is_active = true AND end_date >= ?", time.Now()).
		Find(&list).Error
	return list, err
}

func (r *GORMReminderRepository) Update(ctx context.Context, rem *entity.MedicationReminder) error {
	return r.dbc(ctx).Save(rem).Error
}

func (r *GORMReminderRepository) Deactivate(ctx context.Context, id uint) error {
	return r.dbc(ctx).
		Model(&entity.MedicationReminder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error
}

func (r *GORMReminderRepository) CreateLog(ctx context.Context, log *entity.ReminderLog) error {
	return r.dbc(ctx).Create(log).Error
}

func (r *GORMReminderRepository) UpdateLogStatus(ctx context.Context, logID uint, status entity.ReminderLogStatus) error {
	now := time.Now()
	return r.dbc(ctx).
		Model(&entity.ReminderLog{}).
		Where("id = ?", logID).
		Updates(map[string]interface{}{"status": status, "action_at": now}).Error
}

func (r *GORMReminderRepository) ListLogs(ctx context.Context, reminderID uint, p pagination.Params) ([]*entity.ReminderLog, int64, error) {
	var list []*entity.ReminderLog
	var total int64

	q := r.dbc(ctx).Model(&entity.ReminderLog{}).Where("reminder_id = ?", reminderID)
	q.Count(&total)
	err := q.Order("scheduled_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMReminderRepository) ListDueReminders(ctx context.Context, from, to time.Time) ([]*entity.MedicationReminder, error) {
	var list []*entity.MedicationReminder
	err := r.dbc(ctx).
		Where("is_active = true AND start_date <= ? AND end_date >= ?", to, from).
		Find(&list).Error
	return list, err
}
