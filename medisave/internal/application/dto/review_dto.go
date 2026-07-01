package dto

import "time"

type CreateReviewRequest struct {
	AppointmentID uint   `json:"appointment_id" validate:"required"`
	Rating        int    `json:"rating"         validate:"required,min=1,max=5"`
	Comment       string `json:"comment"        validate:"omitempty,max=1000"`
}

type ReviewResponse struct {
	ID          uint      `json:"id"`
	PatientName string    `json:"patient_name"`
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}
