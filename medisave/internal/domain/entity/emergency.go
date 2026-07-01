package entity

import "time"

type EmergencyStatus string

const (
	EmergencyStatusActive   EmergencyStatus = "active"
	EmergencyStatusResolved EmergencyStatus = "resolved"
	EmergencyStatusFalse    EmergencyStatus = "false_alarm"
)

type Emergency struct {
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID        uint            `gorm:"not null" json:"patient_id"`
	Patient          Patient         `gorm:"foreignKey:PatientID" json:"patient"`
	Status           EmergencyStatus `gorm:"default:'active'" json:"status"`
	Latitude         float64         `json:"latitude"`
	Longitude        float64         `json:"longitude"`
	Address          string          `json:"address"`
	NearestHospital  string          `json:"nearest_hospital"`
	Description      string          `json:"description"`
	ContactsNotified bool            `gorm:"default:false" json:"contacts_notified"`
	SMSSent          bool            `gorm:"default:false" json:"sms_sent"`
	PushSent         bool            `gorm:"default:false" json:"push_sent"`
	ResolvedAt       *time.Time      `json:"resolved_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type EmergencyContact struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID    uint      `gorm:"not null" json:"patient_id"`
	Name         string    `gorm:"not null" json:"name"`
	Phone        string    `gorm:"not null" json:"phone"`
	Relationship string    `json:"relationship"`
	IsPrimary    bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
