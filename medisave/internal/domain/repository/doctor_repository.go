package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type DoctorRepository interface {
	Create(ctx context.Context, doctor *entity.Doctor) error
	FindByID(ctx context.Context, id uint) (*entity.Doctor, error)
	FindByUserID(ctx context.Context, userID uint) (*entity.Doctor, error)
	FindByLicenseNumber(ctx context.Context, license string) (*entity.Doctor, error)
	Update(ctx context.Context, doctor *entity.Doctor) error
	List(ctx context.Context, status string, p pagination.Params) ([]*entity.Doctor, int64, error)
	ListVerified(ctx context.Context, p pagination.Params) ([]*entity.Doctor, int64, error)
	ListAvailable(ctx context.Context, specialty string, p pagination.Params) ([]*entity.Doctor, int64, error)
	SetAvailability(ctx context.Context, doctorID uint, available bool) error
	UpdateStatus(ctx context.Context, doctorID uint, status entity.DoctorStatus) error
	CountAll(ctx context.Context) (int64, error)
	CountPending(ctx context.Context) (int64, error)
	UpdateRating(ctx context.Context, doctorID uint, rating float64, totalReviews int) error
	IncrementConsultations(ctx context.Context, doctorID uint) error
	FindAll(ctx context.Context) ([]*entity.Doctor, error)
}
