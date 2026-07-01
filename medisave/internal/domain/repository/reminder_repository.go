package repository

import (
	"context"
	"time"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type ReminderRepository interface {
	Create(ctx context.Context, r *entity.MedicationReminder) error
	FindByID(ctx context.Context, id uint) (*entity.MedicationReminder, error)
	ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.MedicationReminder, int64, error)
	ListActive(ctx context.Context) ([]*entity.MedicationReminder, error)
	Update(ctx context.Context, r *entity.MedicationReminder) error
	Deactivate(ctx context.Context, id uint) error
	CreateLog(ctx context.Context, log *entity.ReminderLog) error
	UpdateLogStatus(ctx context.Context, logID uint, status entity.ReminderLogStatus) error
	ListLogs(ctx context.Context, reminderID uint, p pagination.Params) ([]*entity.ReminderLog, int64, error)
	ListDueReminders(ctx context.Context, from, to time.Time) ([]*entity.MedicationReminder, error)
}
