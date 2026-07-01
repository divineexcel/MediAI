package dto

import "github.com/medisave/app/internal/domain/entity"

type RegisterRequest struct {
	FirstName string      `json:"first_name" validate:"required,min=2,max=50"`
	LastName  string      `json:"last_name"  validate:"required,min=2,max=50"`
	Email     string      `json:"email"      validate:"required,email"`
	Phone     string      `json:"phone"      validate:"required,min=11,max=15"`
	Password  string      `json:"password"   validate:"required,min=8"`
	Role      entity.Role `json:"role"       validate:"required,oneof=patient doctor"`
}

// DoctorExtras are required only when role == doctor
type DoctorRegisterRequest struct {
	RegisterRequest
	LicenseNumber     string  `json:"license_number"      validate:"required"`
	Specialty         string  `json:"specialty"           validate:"required"`
	YearsOfExperience int     `json:"years_of_experience" validate:"required,min=0"`
	ConsultationFee   float64 `json:"consultation_fee"    validate:"required,min=0"`
	Hospital          string  `json:"hospital"            validate:"required"`
	WorkIDURL         string  `json:"work_id_url"         validate:"required"`
	MedicalLicenseURL string  `json:"medical_license_url" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"        validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type AuthUserResponse struct {
	ID              uint        `json:"id"`
	UUID            string      `json:"uuid"`
	FirstName       string      `json:"first_name"`
	LastName        string      `json:"last_name"`
	Email           string      `json:"email"`
	Phone           string      `json:"phone"`
	Role            entity.Role `json:"role"`
	IsVerified      bool        `json:"is_verified"`
	ProfilePhotoURL string      `json:"profile_photo_url"`
}

type AuthResponse struct {
	User   AuthUserResponse  `json:"user"`
	Tokens TokenPairResponse `json:"tokens"`
}
