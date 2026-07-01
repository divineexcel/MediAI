package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type SavingsRepository interface {
	Create(ctx context.Context, g *entity.HealthSavingsGoal) error
	FindByID(ctx context.Context, id uint) (*entity.HealthSavingsGoal, error)
	ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.HealthSavingsGoal, int64, error)
	Update(ctx context.Context, g *entity.HealthSavingsGoal) error
	UpdateSavedAmount(ctx context.Context, id uint, amount float64) error
	UpdateStatus(ctx context.Context, id uint, status entity.SavingsGoalStatus) error
}
