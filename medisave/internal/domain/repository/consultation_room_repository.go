package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
)

type ConsultationRoomRepository interface {
	FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.ConsultationRoom, error)
	Create(ctx context.Context, room *entity.ConsultationRoom) error
	End(ctx context.Context, appointmentID uint) error
}
