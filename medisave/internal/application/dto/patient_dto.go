package dto

import (
	"time"

	"github.com/medisave/app/internal/domain/entity"
)

type UpdatePatientProfileRequest struct {
	FirstName         string `json:"first_name"          validate:"omitempty,min=2,max=50"`
	LastName          string `json:"last_name"           validate:"omitempty,min=2,max=50"`
	Phone             string `json:"phone"               validate:"omitempty,min=11,max=15"`
	DateOfBirth       string `json:"date_of_birth"       validate:"omitempty"`
	Gender            string `json:"gender"              validate:"omitempty,oneof=male female other"`
	BloodGroup        string `json:"blood_group"         validate:"omitempty"`
	Genotype          string `json:"genotype"            validate:"omitempty"`
	Allergies         string `json:"allergies"           validate:"omitempty"`
	ChronicConditions string `json:"chronic_conditions"  validate:"omitempty"`
	Address           string `json:"address"             validate:"omitempty"`
	State             string `json:"state"               validate:"omitempty"`
	LGA               string `json:"lga"                 validate:"omitempty"`
	NHISNumber        string `json:"nhis_number"         validate:"omitempty"`
}

type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token" validate:"required"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type PatientProfileResponse struct {
	ID                uint               `json:"id"`
	User              AuthUserResponse    `json:"user"`
	DateOfBirth       *time.Time         `json:"date_of_birth"`
	Gender            string             `json:"gender"`
	BloodGroup        entity.BloodGroup  `json:"blood_group"`
	Genotype          string             `json:"genotype"`
	Allergies         string             `json:"allergies"`
	ChronicConditions string             `json:"chronic_conditions"`
	Address           string             `json:"address"`
	State             string             `json:"state"`
	LGA               string             `json:"lga"`
	NHISNumber        string             `json:"nhis_number"`
	HealthScore       int                `json:"health_score"`
}

type PatientDashboardResponse struct {
	Patient              PatientProfileResponse `json:"patient"`
	WalletBalance        float64               `json:"wallet_balance"`
	UpcomingAppointments int                   `json:"upcoming_appointments"`
	UnreadNotifications  int64                 `json:"unread_notifications"`
	ActiveReminders      int                   `json:"active_reminders"`
	HealthScore          int                   `json:"health_score"`
	SavingsGoals         int                   `json:"savings_goals"`
}
