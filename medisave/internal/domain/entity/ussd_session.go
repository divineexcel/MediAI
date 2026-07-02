package entity

import "time"

// USSDSession tracks USSD session state between gateway callbacks.
// ExpiresAt enforces the 3-minute USSD session timeout.
// Data holds a JSON blob of transient menu data (selected IDs, partial inputs).
type USSDSession struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	SessionID string    `gorm:"uniqueIndex;not null"`
	Phone     string    `gorm:"not null;index"`
	UserID    uint      `gorm:"default:0"` // 0 = phone not in users table
	MenuState string    `gorm:"not null;default:'home'"`
	PrevMenu  string    `gorm:"not null;default:''"`
	Data      string    `gorm:"type:text;not null;default:'{}'"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
