package dto

import "time"

type SendMessageRequest struct {
	Message     string `json:"message"      validate:"required,min=1,max=2000"`
	MessageType string `json:"message_type" validate:"omitempty,oneof=text image file prescription"`
}

type ConsultationNotesRequest struct {
	DoctorNotes   string `json:"doctor_notes"   validate:"required,min=10"`
	Diagnosis     string `json:"diagnosis"      validate:"omitempty"`
	Treatment     string `json:"treatment"      validate:"omitempty"`
	FollowUpDate  string `json:"follow_up_date" validate:"omitempty"`
	FollowUpNotes string `json:"follow_up_notes" validate:"omitempty"`
}

type AddPrescriptionRequest struct {
	MedicineName string `json:"medicine_name" validate:"required"`
	Dosage       string `json:"dosage"        validate:"required"`
	Frequency    string `json:"frequency"     validate:"required"`
	Duration     string `json:"duration"      validate:"required"`
	Instructions string `json:"instructions"  validate:"omitempty"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type ConsultationMessageResponse struct {
	ID            uint      `json:"id"`
	SenderID      uint      `json:"sender_id"`
	SenderName    string    `json:"sender_name"`
	SenderRole    string    `json:"sender_role"`
	Message       string    `json:"message"`
	MessageType   string    `json:"message_type"`
	AttachmentURL string    `json:"attachment_url"`
	IsRead        bool      `json:"is_read"`
	CreatedAt     time.Time `json:"created_at"`
}

type PrescriptionResponse struct {
	ID           uint   `json:"id"`
	MedicineName string `json:"medicine_name"`
	Dosage       string `json:"dosage"`
	Frequency    string `json:"frequency"`
	Duration     string `json:"duration"`
	Instructions string `json:"instructions"`
	IsFilled     bool   `json:"is_filled"`
	DoctorName   string `json:"doctor_name"`
	CreatedAt    string `json:"created_at"`
}

type ConsultationResponse struct {
	ID            uint                          `json:"id"`
	AppointmentID uint                          `json:"appointment_id"`
	DoctorNotes   string                        `json:"doctor_notes"`
	Diagnosis     string                        `json:"diagnosis"`
	Treatment     string                        `json:"treatment"`
	FollowUpDate  *time.Time                    `json:"follow_up_date"`
	FollowUpNotes string                        `json:"follow_up_notes"`
	Messages      []ConsultationMessageResponse `json:"messages"`
	Prescriptions []PrescriptionResponse        `json:"prescriptions"`
}
