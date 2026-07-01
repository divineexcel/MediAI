package entity

import "time"

type DoctorStatus string

const (
	DoctorStatusPending  DoctorStatus = "pending"
	DoctorStatusVerified DoctorStatus = "verified"
	DoctorStatusSuspended DoctorStatus = "suspended"
)

type Doctor struct {
	ID                  uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID              uint         `gorm:"uniqueIndex;not null" json:"user_id"`
	User                User         `gorm:"foreignKey:UserID" json:"user"`
	LicenseNumber       string       `gorm:"uniqueIndex;not null" json:"license_number"`
	Specialty           string       `gorm:"not null" json:"specialty"`
	SubSpecialty        string       `json:"sub_specialty"`
	YearsOfExperience   int          `json:"years_of_experience"`
	Hospital            string       `json:"hospital"`
	ConsultationFee     float64      `gorm:"not null;default:0" json:"consultation_fee"`
	IsAvailable         bool         `gorm:"default:false" json:"is_available"`
	Status              DoctorStatus `gorm:"default:'pending'" json:"status"`
	Bio                 string       `json:"bio"`
	Education           string       `json:"education"`
	Certifications      string       `json:"certifications"`
	Languages           string       `json:"languages"`
	WorkIDURL           string       `json:"work_id_url"`
	MedicalLicenseURL   string       `json:"medical_license_url"`
	Rating              float64      `gorm:"default:0" json:"rating"`
	TotalReviews        int          `gorm:"default:0" json:"total_reviews"`
	TotalConsultations  int          `gorm:"default:0" json:"total_consultations"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}
