package dto

import (
	"time"

	"github.com/medisave/app/internal/domain/entity"
)

type CreateReminderRequest struct {
	MedicineName  string `json:"medicine_name"  validate:"required,min=2,max=100"`
	Dosage        string `json:"dosage"         validate:"required"`
	Frequency     string `json:"frequency"      validate:"required,oneof=once daily twice_daily three_times_daily weekly"`
	MorningTime   string `json:"morning_time"   validate:"omitempty"`
	AfternoonTime string `json:"afternoon_time" validate:"omitempty"`
	NightTime     string `json:"night_time"     validate:"omitempty"`
	StartDate     string `json:"start_date"     validate:"required"`
	EndDate       string `json:"end_date"       validate:"required"`
	Instructions  string `json:"instructions"   validate:"omitempty"`
}

type ReminderLogActionRequest struct {
	Status string `json:"status" validate:"required,oneof=taken skipped"`
	Notes  string `json:"notes"  validate:"omitempty,max=300"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type ReminderResponse struct {
	ID            uint                      `json:"id"`
	MedicineName  string                    `json:"medicine_name"`
	Dosage        string                    `json:"dosage"`
	Frequency     entity.ReminderFrequency  `json:"frequency"`
	MorningTime   string                    `json:"morning_time"`
	AfternoonTime string                    `json:"afternoon_time"`
	NightTime     string                    `json:"night_time"`
	StartDate     time.Time                 `json:"start_date"`
	EndDate       time.Time                 `json:"end_date"`
	Instructions  string                    `json:"instructions"`
	IsActive      bool                      `json:"is_active"`
}

type ReminderAdherenceResponse struct {
	TotalDoses   int     `json:"total_doses"`
	TakenDoses   int     `json:"taken_doses"`
	SkippedDoses int     `json:"skipped_doses"`
	MissedDoses  int     `json:"missed_doses"`
	AdherenceRate float64 `json:"adherence_rate"`
}
