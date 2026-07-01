package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type EmergencyRepository interface {
	Create(ctx context.Context, e *entity.Emergency) error
	FindByID(ctx context.Context, id uint) (*entity.Emergency, error)
	Update(ctx context.Context, e *entity.Emergency) error
	ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.Emergency, int64, error)
	ListActive(ctx context.Context) ([]*entity.Emergency, error)
	CountActive(ctx context.Context) (int64, error)
}

type EmergencyContactRepository interface {
	Create(ctx context.Context, c *entity.EmergencyContact) error
	ListByPatient(ctx context.Context, patientID uint) ([]*entity.EmergencyContact, error)
	FindByID(ctx context.Context, id uint) (*entity.EmergencyContact, error)
	Update(ctx context.Context, c *entity.EmergencyContact) error
	Delete(ctx context.Context, id uint) error
	SetPrimary(ctx context.Context, patientID uint, contactID uint) error
}
