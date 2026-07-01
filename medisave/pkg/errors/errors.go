package errors

import "errors"

var (
	// Auth
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account is deactivated")
	ErrAccountUnverified  = errors.New("account is not verified")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("access denied")

	// User
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already registered")
	ErrPhoneExists        = errors.New("phone number already registered")

	// Patient
	ErrPatientNotFound    = errors.New("patient not found")

	// Doctor
	ErrDoctorNotFound     = errors.New("doctor not found")
	ErrDoctorNotVerified  = errors.New("doctor not yet verified")
	ErrDoctorUnavailable  = errors.New("doctor is not available")
	ErrLicenseExists      = errors.New("license number already registered")

	// Wallet
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInsufficientFunds  = errors.New("insufficient wallet balance")
	ErrWalletInactive     = errors.New("wallet is inactive")

	// Appointment
	ErrAppointmentNotFound      = errors.New("appointment not found")
	ErrAppointmentConflict      = errors.New("doctor has an appointment at this time")
	ErrAppointmentNotPending    = errors.New("appointment is not in a cancellable state")
	ErrAppointmentNotInProgress = errors.New("appointment is not in progress")
	ErrScheduledTooSoon         = errors.New("appointment must be scheduled at least 30 minutes in the future")
	ErrInvalidScheduleFormat    = errors.New("invalid date format — use ISO 8601, e.g. 2024-12-25T10:00:00Z")
	ErrCompletedOnly            = errors.New("action only allowed on completed appointments")

	// Consultation
	ErrConsultationNotFound  = errors.New("consultation not found")
	ErrConsultationInactive  = errors.New("consultation is not currently active")

	// Medical Record
	ErrRecordNotFound = errors.New("medical record not found")
	ErrAccessDenied   = errors.New("you do not have access to this record")

	// AI
	ErrAIServiceUnavailable = errors.New("AI service is temporarily unavailable")

	// Emergency
	ErrEmergencyNotFound = errors.New("emergency not found")

	// Reminder
	ErrReminderNotFound = errors.New("reminder not found")

	// Review
	ErrReviewExists   = errors.New("you have already reviewed this consultation")
	ErrReviewNotFound = errors.New("review not found")

	// Generic
	ErrNotFound       = errors.New("resource not found")
	ErrInternalServer = errors.New("an internal server error occurred")
	ErrBadRequest     = errors.New("invalid request")
)
