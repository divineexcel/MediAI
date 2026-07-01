package entity

import (
	"time"
)

type Role string

const (
	RolePatient Role = "patient"
	RoleDoctor  Role = "doctor"
	RoleAdmin   Role = "admin"
)

type User struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID            string    `gorm:"uniqueIndex;not null" json:"uuid"`
	FirstName       string    `gorm:"not null" json:"first_name"`
	LastName        string    `gorm:"not null" json:"last_name"`
	Email           string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone           string    `gorm:"uniqueIndex;not null" json:"phone"`
	PasswordHash    string    `gorm:"not null" json:"-"`
	Role            Role      `gorm:"not null" json:"role"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	ProfilePhotoURL string    `json:"profile_photo_url"`
	FCMToken        string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
