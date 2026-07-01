package entity

import "time"

type MedicalRecord struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID       uint      `gorm:"not null" json:"patient_id"`
	Patient         Patient   `gorm:"foreignKey:PatientID" json:"patient"`
	ConsultationID  *uint     `json:"consultation_id"`
	DoctorID        *uint     `json:"doctor_id"`
	RecordType      string    `gorm:"not null" json:"record_type"`
	Title           string    `gorm:"not null" json:"title"`
	Description     string    `json:"description"`
	FileURL         string    `json:"file_url"`
	RecordDate      time.Time `gorm:"not null" json:"record_date"`
	IsShared        bool      `gorm:"default:false" json:"is_shared"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Prescription struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConsultationID uint      `gorm:"not null" json:"consultation_id"`
	PatientID      uint      `gorm:"not null" json:"patient_id"`
	DoctorID       uint      `gorm:"not null" json:"doctor_id"`
	MedicineName   string    `gorm:"not null" json:"medicine_name"`
	Dosage         string    `gorm:"not null" json:"dosage"`
	Frequency      string    `gorm:"not null" json:"frequency"`
	Duration       string    `gorm:"not null" json:"duration"`
	Instructions   string    `json:"instructions"`
	IsFilled       bool      `gorm:"default:false" json:"is_filled"`
	CreatedAt      time.Time `json:"created_at"`
}
