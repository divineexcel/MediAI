package entity

import "time"

type NotificationType string
type NotificationChannel string

const (
	NotifTypeAppointment  NotificationType = "appointment"
	NotifTypeMedication   NotificationType = "medication"
	NotifTypeWallet       NotificationType = "wallet"
	NotifTypeEmergency    NotificationType = "emergency"
	NotifTypeAI           NotificationType = "ai"
	NotifTypeConsultation NotificationType = "consultation"
	NotifTypeSystem       NotificationType = "system"

	ChannelPush    NotificationChannel = "push"
	ChannelSMS     NotificationChannel = "sms"
	ChannelInApp   NotificationChannel = "in_app"
)

type Notification struct {
	ID        uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint                `gorm:"not null" json:"user_id"`
	Type      NotificationType    `gorm:"not null" json:"type"`
	Channel   NotificationChannel `gorm:"not null" json:"channel"`
	Title     string              `gorm:"not null" json:"title"`
	Body      string              `gorm:"not null" json:"body"`
	Data      string              `json:"data"`
	IsRead    bool                `gorm:"default:false" json:"is_read"`
	IsSent    bool                `gorm:"default:false" json:"is_sent"`
	SentAt    *time.Time          `json:"sent_at"`
	CreatedAt time.Time           `json:"created_at"`
}
