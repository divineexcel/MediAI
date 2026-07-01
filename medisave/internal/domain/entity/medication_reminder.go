package entity

import "time"

type ReminderFrequency string
type ReminderLogStatus string

const (
	FrequencyOnce      ReminderFrequency = "once"
	FrequencyDaily     ReminderFrequency = "daily"
	FrequencyTwiceDaily ReminderFrequency = "twice_daily"
	FrequencyThreeDaily ReminderFrequency = "three_times_daily"
	FrequencyWeekly    ReminderFrequency = "weekly"

	ReminderTaken  ReminderLogStatus = "taken"
	ReminderSkipped ReminderLogStatus = "skipped"
	ReminderMissed ReminderLogStatus = "missed"
)

type MedicationReminder struct {
	ID             uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID      uint              `gorm:"not null" json:"patient_id"`
	PrescriptionID *uint             `json:"prescription_id"`
	MedicineName   string            `gorm:"not null" json:"medicine_name"`
	Dosage         string            `gorm:"not null" json:"dosage"`
	Frequency      ReminderFrequency `gorm:"not null" json:"frequency"`
	MorningTime    string            `json:"morning_time"`
	AfternoonTime  string            `json:"afternoon_time"`
	NightTime      string            `json:"night_time"`
	StartDate      time.Time         `gorm:"not null" json:"start_date"`
	EndDate        time.Time         `gorm:"not null" json:"end_date"`
	Instructions   string            `json:"instructions"`
	IsActive       bool              `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type ReminderLog struct {
	ID         uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	ReminderID uint              `gorm:"not null" json:"reminder_id"`
	PatientID  uint              `gorm:"not null" json:"patient_id"`
	ScheduledAt time.Time        `gorm:"not null" json:"scheduled_at"`
	ActionAt   *time.Time        `json:"action_at"`
	Status     ReminderLogStatus `gorm:"default:'missed'" json:"status"`
	Notes      string            `json:"notes"`
	CreatedAt  time.Time         `json:"created_at"`
}
