package dto

import (
	"time"

	"github.com/medisave/app/internal/domain/entity"
)

type BookAppointmentRequest struct {
	DoctorID       uint   `json:"doctor_id"       validate:"required"`
	Type           string `json:"type"            validate:"required,oneof=chat voice video"`
	ScheduledAt    string `json:"scheduled_at"    validate:"required"`
	ChiefComplaint string `json:"chief_complaint" validate:"required,min=10,max=500"`
}

type CancelAppointmentRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=300"`
}

type RescheduleAppointmentRequest struct {
	ScheduledAt string `json:"scheduled_at" validate:"required"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type AppointmentResponse struct {
	ID              uint                       `json:"id"`
	Patient         PatientProfileResponse     `json:"patient"`
	Doctor          DoctorProfileResponse      `json:"doctor"`
	Type            entity.AppointmentType     `json:"type"`
	Status          entity.AppointmentStatus   `json:"status"`
	ScheduledAt     time.Time                  `json:"scheduled_at"`
	StartedAt       *time.Time                 `json:"started_at"`
	CompletedAt     *time.Time                 `json:"completed_at"`
	CallDuration    int                        `json:"call_duration"`
	ConsultationFee float64                    `json:"consultation_fee"`
	ChiefComplaint  string                     `json:"chief_complaint"`
	Notes           string                     `json:"notes"`
	CreatedAt       time.Time                  `json:"created_at"`
}

type AppointmentBookedResponse struct {
	Appointment AppointmentResponse  `json:"appointment"`
	Transaction TransactionResponse  `json:"transaction"`
	Message     string               `json:"message"`
}
