package entity

import "time"

type AppointmentStatus string
type AppointmentType string

const (
	AppointmentStatusPending    AppointmentStatus = "pending"
	AppointmentStatusConfirmed  AppointmentStatus = "confirmed"
	AppointmentStatusInProgress AppointmentStatus = "in_progress"
	AppointmentStatusCompleted  AppointmentStatus = "completed"
	AppointmentStatusCancelled  AppointmentStatus = "cancelled"
	AppointmentStatusNoShow     AppointmentStatus = "no_show"

	AppointmentTypeChat  AppointmentType = "chat"
	AppointmentTypeVoice AppointmentType = "voice"
	AppointmentTypeVideo AppointmentType = "video"
)

type Appointment struct {
	ID              uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID       uint              `gorm:"not null" json:"patient_id"`
	Patient         Patient           `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID        uint              `gorm:"not null" json:"doctor_id"`
	Doctor          Doctor            `gorm:"foreignKey:DoctorID" json:"doctor"`
	Type            AppointmentType   `gorm:"default:'chat'" json:"type"`
	Status          AppointmentStatus `gorm:"default:'pending'" json:"status"`
	ScheduledAt     time.Time         `gorm:"not null" json:"scheduled_at"`
	StartedAt       *time.Time        `json:"started_at"`
	CompletedAt     *time.Time        `json:"completed_at"`
	CallDuration    int               `gorm:"default:0" json:"call_duration"`
	ConsultationFee float64           `gorm:"not null" json:"consultation_fee"`
	TransactionID   uint              `json:"transaction_id"`
	ChiefComplaint  string            `json:"chief_complaint"`
	Notes           string            `json:"notes"`
	CancelReason    string            `json:"cancel_reason"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
