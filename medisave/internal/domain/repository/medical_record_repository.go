package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type MedicalRecordRepository interface {
	Create(ctx context.Context, r *entity.MedicalRecord) error
	FindByID(ctx context.Context, id uint) (*entity.MedicalRecord, error)
	ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.MedicalRecord, int64, error)
	Update(ctx context.Context, r *entity.MedicalRecord) error
	Delete(ctx context.Context, id uint) error
}

type PrescriptionRepository interface {
	Create(ctx context.Context, p *entity.Prescription) error
	FindByID(ctx context.Context, id uint) (*entity.Prescription, error)
	ListByPatient(ctx context.Context, patientID uint, p pagination.Params) ([]*entity.Prescription, int64, error)
	ListByConsultation(ctx context.Context, consultationID uint) ([]*entity.Prescription, error)
	MarkFilled(ctx context.Context, id uint) error
}
