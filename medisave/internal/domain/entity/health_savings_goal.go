package entity

import "time"

type SavingsGoalStatus string

const (
	GoalStatusActive    SavingsGoalStatus = "active"
	GoalStatusCompleted SavingsGoalStatus = "completed"
	GoalStatusCancelled SavingsGoalStatus = "cancelled"
)

type HealthSavingsGoal struct {
	ID            uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID     uint              `gorm:"not null" json:"patient_id"`
	WalletID      uint              `gorm:"not null" json:"wallet_id"`
	Title         string            `gorm:"not null" json:"title"`
	Description   string            `json:"description"`
	TargetAmount  float64           `gorm:"not null" json:"target_amount"`
	SavedAmount   float64           `gorm:"default:0" json:"saved_amount"`
	Frequency     string            `gorm:"not null" json:"frequency"`
	AutoSaveAmount float64          `json:"auto_save_amount"`
	Status        SavingsGoalStatus `gorm:"default:'active'" json:"status"`
	TargetDate    time.Time         `gorm:"not null" json:"target_date"`
	CompletedAt   *time.Time        `json:"completed_at"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
