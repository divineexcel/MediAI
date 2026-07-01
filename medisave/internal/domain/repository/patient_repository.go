package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type PatientRepository interface {
	Create(ctx context.Context, patient *entity.Patient) error
	FindByID(ctx context.Context, id uint) (*entity.Patient, error)
	FindByUserID(ctx context.Context, userID uint) (*entity.Patient, error)
	Update(ctx context.Context, patient *entity.Patient) error
	List(ctx context.Context, p pagination.Params) ([]*entity.Patient, int64, error)
	UpdateHealthScore(ctx context.Context, patientID uint, score int) error
	CountAll(ctx context.Context) (int64, error)
}
