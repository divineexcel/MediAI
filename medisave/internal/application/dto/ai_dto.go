package dto

import (
	"time"

	"github.com/medisave/app/internal/domain/entity"
)

type AIChatRequest struct {
	Message string `json:"message" validate:"required,min=3,max=2000"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type AIMessageResponse struct {
	ID                uint               `json:"id"`
	Role              string             `json:"role"`
	Content           string             `json:"content"`
	Severity          *entity.AISeverity `json:"severity,omitempty"`
	PossibleCondition string             `json:"possible_condition,omitempty"`
	RecommendedAction string             `json:"recommended_action,omitempty"`
	LabTests          string             `json:"lab_tests,omitempty"`
	HomeCare          string             `json:"home_care,omitempty"`
	EmergencyWarning  string             `json:"emergency_warning,omitempty"`
	ShowBookDoctor    bool               `json:"show_book_doctor"`
	ShowFindHospital  bool               `json:"show_find_hospital"`
	ShowEmergencySOS  bool               `json:"show_emergency_sos"`
	Disclaimer        string             `json:"disclaimer,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

type AIConversationResponse struct {
	ID        uint                `json:"id"`
	SessionID string              `json:"session_id"`
	IsActive  bool                `json:"is_active"`
	Messages  []AIMessageResponse `json:"messages"`
	CreatedAt time.Time           `json:"created_at"`
}
