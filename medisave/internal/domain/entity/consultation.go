package entity

import "time"

type Consultation struct {
	ID              uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	AppointmentID   uint        `gorm:"uniqueIndex;not null" json:"appointment_id"`
	Appointment     Appointment `gorm:"foreignKey:AppointmentID" json:"appointment"`
	DoctorNotes     string      `json:"doctor_notes"`
	Diagnosis       string      `json:"diagnosis"`
	Treatment       string      `json:"treatment"`
	FollowUpDate    *time.Time  `json:"follow_up_date"`
	FollowUpNotes   string      `json:"follow_up_notes"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type ConsultationMessage struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AppointmentID  uint      `gorm:"not null" json:"appointment_id"`
	SenderID       uint      `gorm:"not null" json:"sender_id"`
	SenderRole     Role      `gorm:"not null" json:"sender_role"`
	Message        string    `gorm:"not null" json:"message"`
	MessageType    string    `gorm:"default:'text'" json:"message_type"`
	AttachmentURL  string    `json:"attachment_url"`
	IsRead         bool      `gorm:"default:false" json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}
