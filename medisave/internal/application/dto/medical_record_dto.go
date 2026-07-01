package dto

import "time"

type CreateMedicalRecordRequest struct {
	RecordType  string `json:"record_type"  validate:"required,oneof=consultation_note prescription lab_report imaging vaccination discharge_summary other"`
	Title       string `json:"title"        validate:"required,min=3,max=200"`
	Description string `json:"description"  validate:"omitempty"`
	FileURL     string `json:"file_url"     validate:"omitempty,url"`
	RecordDate  string `json:"record_date"  validate:"required"`
}

type MedicalRecordFilterRequest struct {
	RecordType string `form:"record_type"`
	From       string `form:"from"`
	To         string `form:"to"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type MedicalRecordResponse struct {
	ID             uint       `json:"id"`
	RecordType     string     `json:"record_type"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	FileURL        string     `json:"file_url"`
	RecordDate     time.Time  `json:"record_date"`
	IsShared       bool       `json:"is_shared"`
	DoctorName     string     `json:"doctor_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
