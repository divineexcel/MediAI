package entity

import "time"

type ConsultationRoomStatus string

const (
	ConsultationRoomStatusActive ConsultationRoomStatus = "active"
	ConsultationRoomStatusEnded  ConsultationRoomStatus = "ended"
)

type ConsultationRoom struct {
	ID            uint                   `gorm:"primaryKey;autoIncrement" json:"id"`
	AppointmentID uint                   `gorm:"uniqueIndex;not null" json:"appointment_id"`
	RoomName      string                 `gorm:"not null" json:"room_name"`
	Status        ConsultationRoomStatus `gorm:"not null;default:'active'" json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	EndedAt       *time.Time             `json:"ended_at,omitempty"`
}
