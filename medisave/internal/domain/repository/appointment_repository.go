package repository

import (
	"context"
	"time"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type AppointmentRepository interface {
	Create(ctx context.Context, a *entity.Appointment) error
	FindByID(ctx context.Context, id uint) (*entity.Appointment, error)
	Update(ctx context.Context, a *entity.Appointment) error
	ListByPatient(ctx context.Context, patientID uint, status string, p pagination.Params) ([]*entity.Appointment, int64, error)
	ListByDoctor(ctx context.Context, doctorID uint, status string, p pagination.Params) ([]*entity.Appointment, int64, error)
	ListTodayByDoctor(ctx context.Context, doctorID uint) ([]*entity.Appointment, error)
	FindConflict(ctx context.Context, doctorID uint, scheduled time.Time) (*entity.Appointment, error)
	UpdateStatus(ctx context.Context, id uint, status entity.AppointmentStatus) error
	CountByDoctor(ctx context.Context, doctorID uint) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	ListAll(ctx context.Context, p pagination.Params) ([]*entity.Appointment, int64, error)
}

type ConsultationRepository interface {
	Create(ctx context.Context, c *entity.Consultation) error
	FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.Consultation, error)
	Update(ctx context.Context, c *entity.Consultation) error
	CreateMessage(ctx context.Context, m *entity.ConsultationMessage) error
	ListMessages(ctx context.Context, appointmentID uint) ([]*entity.ConsultationMessage, error)
	MarkMessagesRead(ctx context.Context, appointmentID uint, readerID uint) error
}
