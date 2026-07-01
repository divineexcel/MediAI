package entity

import "time"

type AISeverity string

const (
	AISeverityLow      AISeverity = "low"
	AISeverityModerate AISeverity = "moderate"
	AISeverityHigh     AISeverity = "high"
)

type AIConversation struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PatientID uint      `gorm:"not null" json:"patient_id"`
	SessionID string    `gorm:"not null" json:"session_id"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AIMessage struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID   uint       `gorm:"not null" json:"conversation_id"`
	Role             string     `gorm:"not null" json:"role"` // user | assistant
	Content          string     `gorm:"not null" json:"content"`
	Severity         *AISeverity `json:"severity,omitempty"`
	PossibleCondition string    `json:"possible_condition"`
	RecommendedAction string    `json:"recommended_action"`
	LabTests         string     `json:"lab_tests"`
	HomeCare         string     `json:"home_care"`
	EmergencyWarning string     `json:"emergency_warning"`
	ShowBookDoctor   bool       `gorm:"default:false" json:"show_book_doctor"`
	ShowFindHospital bool       `gorm:"default:false" json:"show_find_hospital"`
	ShowEmergencySOS bool       `gorm:"default:false" json:"show_emergency_sos"`
	CreatedAt        time.Time  `json:"created_at"`
}
