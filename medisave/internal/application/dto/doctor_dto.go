package dto

import "github.com/medisave/app/internal/domain/entity"

type UpdateDoctorProfileRequest struct {
	FirstName         string  `json:"first_name"          validate:"omitempty,min=2,max=50"`
	LastName          string  `json:"last_name"           validate:"omitempty,min=2,max=50"`
	Phone             string  `json:"phone"               validate:"omitempty,min=11,max=15"`
	SubSpecialty      string  `json:"sub_specialty"       validate:"omitempty"`
	Hospital          string  `json:"hospital"            validate:"omitempty"`
	ConsultationFee   float64 `json:"consultation_fee"    validate:"omitempty,min=0"`
	Bio               string  `json:"bio"                 validate:"omitempty,max=1000"`
	Education         string  `json:"education"           validate:"omitempty"`
	Certifications    string  `json:"certifications"      validate:"omitempty"`
	Languages         string  `json:"languages"           validate:"omitempty"`
}

type ToggleAvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

type DoctorListFilter struct {
	Specialty string `form:"specialty"`
	Available bool   `form:"available"`
	Search    string `form:"search"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type DoctorProfileResponse struct {
	ID                 uint                `json:"id"`
	User               AuthUserResponse     `json:"user"`
	LicenseNumber      string              `json:"license_number"`
	Specialty          string              `json:"specialty"`
	SubSpecialty       string              `json:"sub_specialty"`
	YearsOfExperience  int                 `json:"years_of_experience"`
	Hospital           string              `json:"hospital"`
	ConsultationFee    float64             `json:"consultation_fee"`
	IsAvailable        bool                `json:"is_available"`
	Status             entity.DoctorStatus `json:"status"`
	Remarks            string              `json:"remarks"`
	Bio                string              `json:"bio"`
	Education          string              `json:"education"`
	Certifications     string              `json:"certifications"`
	Languages          string              `json:"languages"`
	Rating             float64             `json:"rating"`
	TotalReviews       int                 `json:"total_reviews"`
	TotalConsultations int                 `json:"total_consultations"`
}

type DoctorDashboardResponse struct {
	Doctor               DoctorProfileResponse `json:"doctor"`
	WalletBalance        float64               `json:"wallet_balance"`
	TodayAppointments    int                   `json:"today_appointments"`
	PendingAppointments  int                   `json:"pending_appointments"`
	TotalEarnings        float64               `json:"total_earnings"`
	UnreadNotifications  int64                 `json:"unread_notifications"`
	Rating               float64               `json:"rating"`
	TotalConsultations   int                   `json:"total_consultations"`
}

type DoctorEarningsResponse struct {
	TotalEarnings   float64 `json:"total_earnings"`
	ThisMonthEarnings float64 `json:"this_month_earnings"`
	LastMonthEarnings float64 `json:"last_month_earnings"`
	PendingEscrow   float64 `json:"pending_escrow"`
}
