package dto

import "time"

type CreateSavingsGoalRequest struct {
	Title          string  `json:"title"            validate:"required,min=3,max=100"`
	Description    string  `json:"description"      validate:"omitempty"`
	TargetAmount   float64 `json:"target_amount"    validate:"required,min=500"`
	Frequency      string  `json:"frequency"        validate:"required,oneof=daily weekly monthly"`
	AutoSaveAmount float64 `json:"auto_save_amount" validate:"omitempty,min=0"`
	TargetDate     string  `json:"target_date"      validate:"required"`
}

type ContributeToGoalRequest struct {
	Amount float64 `json:"amount" validate:"required,min=100"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type SavingsGoalResponse struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	TargetAmount   float64    `json:"target_amount"`
	SavedAmount    float64    `json:"saved_amount"`
	ProgressPct    float64    `json:"progress_pct"`
	Frequency      string     `json:"frequency"`
	AutoSaveAmount float64    `json:"auto_save_amount"`
	Status         string     `json:"status"`
	TargetDate     time.Time  `json:"target_date"`
	DaysRemaining  int        `json:"days_remaining"`
}
