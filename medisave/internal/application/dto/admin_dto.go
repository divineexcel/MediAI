package dto

type AdminAnalyticsResponse struct {
	TotalPatients      int64   `json:"total_patients"`
	TotalDoctors       int64   `json:"total_doctors"`
	TotalAppointments  int64   `json:"total_appointments"`
	TotalTransactions  int64   `json:"total_transactions"`
	TotalVolume        float64 `json:"total_volume"`
	PendingDoctors     int64   `json:"pending_doctors"`
	ActiveEmergencies  int64   `json:"active_emergencies"`
	AIConversationsToday int64 `json:"ai_conversations_today"`
}

type VerifyDoctorRequest struct {
	Status  string `json:"status"  validate:"required,oneof=verified suspended"`
	Remarks string `json:"remarks" validate:"omitempty"`
}

type HealthCampaignRequest struct {
	Title   string `json:"title"   validate:"required"`
	Body    string `json:"body"    validate:"required"`
	Target  string `json:"target"  validate:"required,oneof=all patients doctors"`
}
