package entity

import "time"

type BloodGroup string

const (
	BloodGroupAPos  BloodGroup = "A+"
	BloodGroupANeg  BloodGroup = "A-"
	BloodGroupBPos  BloodGroup = "B+"
	BloodGroupBNeg  BloodGroup = "B-"
	BloodGroupABPos BloodGroup = "AB+"
	BloodGroupABNeg BloodGroup = "AB-"
	BloodGroupOPos  BloodGroup = "O+"
	BloodGroupONeg  BloodGroup = "O-"
)

type Patient struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint       `gorm:"uniqueIndex;not null" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID" json:"user"`
	DateOfBirth    *time.Time `json:"date_of_birth"`
	Gender         string     `json:"gender"`
	BloodGroup     BloodGroup `json:"blood_group"`
	Genotype       string     `json:"genotype"`
	Allergies      string     `json:"allergies"`
	ChronicConditions string  `json:"chronic_conditions"`
	Address        string     `json:"address"`
	State          string     `json:"state"`
	LGA            string     `json:"lga"`
	NHISNumber     string     `json:"nhis_number"`
	HealthScore    int        `gorm:"default:0" json:"health_score"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
