package dto

import "time"

type SOSRequest struct {
	Latitude    float64 `json:"latitude"    validate:"required"`
	Longitude   float64 `json:"longitude"   validate:"required"`
	Description string  `json:"description" validate:"omitempty,max=500"`
}

type ResolveEmergencyRequest struct {
	Status string `json:"status" validate:"required,oneof=resolved false_alarm"`
}

type EmergencyContactRequest struct {
	Name         string `json:"name"         validate:"required,min=2,max=100"`
	Phone        string `json:"phone"        validate:"required,min=11,max=15"`
	Relationship string `json:"relationship" validate:"required"`
	IsPrimary    bool   `json:"is_primary"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type EmergencyResponse struct {
	ID               uint       `json:"id"`
	Status           string     `json:"status"`
	Latitude         float64    `json:"latitude"`
	Longitude        float64    `json:"longitude"`
	Address          string     `json:"address"`
	NearestHospital  string     `json:"nearest_hospital"`
	Description      string     `json:"description"`
	ContactsNotified bool       `json:"contacts_notified"`
	SMSSent          bool       `json:"sms_sent"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type EmergencyContactResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Relationship string `json:"relationship"`
	IsPrimary    bool   `json:"is_primary"`
}
