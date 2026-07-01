package entity

import "time"

type Review struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID      uint      `gorm:"not null" json:"patient_id"`
	Patient        Patient   `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID       uint      `gorm:"not null" json:"doctor_id"`
	Doctor         Doctor    `gorm:"foreignKey:DoctorID" json:"doctor"`
	AppointmentID  uint      `gorm:"uniqueIndex;not null" json:"appointment_id"`
	Rating         int       `gorm:"not null" json:"rating"` // 1-5
	Comment        string    `json:"comment"`
	IsVisible      bool      `gorm:"default:true" json:"is_visible"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
