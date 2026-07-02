package entity

import "time"

type HealthCampaign struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title      string    `gorm:"not null" json:"title"`
	Message    string    `gorm:"not null" json:"message"`
	Category   string    `gorm:"not null" json:"category"`
	TargetRole string    `gorm:"not null" json:"target_role"`
	Location   string    `gorm:"not null" json:"location"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
