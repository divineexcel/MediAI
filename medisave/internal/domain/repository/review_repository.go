package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type ReviewRepository interface {
	Create(ctx context.Context, r *entity.Review) error
	FindByID(ctx context.Context, id uint) (*entity.Review, error)
	FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.Review, error)
	ListByDoctor(ctx context.Context, doctorID uint, p pagination.Params) ([]*entity.Review, int64, error)
	AverageRatingByDoctor(ctx context.Context, doctorID uint) (float64, int, error)
}
