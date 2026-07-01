package seed

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/hash"
	"github.com/medisave/app/pkg/logger"
)

// Run inserts development seed data. Safe to call multiple times — skips if data exists.
func Run(db *gorm.DB) error {
	logger.Info("running database seeder")

	if err := seedAdmin(db); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if err := seedDoctors(db); err != nil {
		return fmt.Errorf("seed doctors: %w", err)
	}
	if err := seedPatients(db); err != nil {
		return fmt.Errorf("seed patients: %w", err)
	}

	logger.Info("seeder complete")
	return nil
}

// ─── ADMIN ───────────────────────────────────────────────────────────────────

func seedAdmin(db *gorm.DB) error {
	var count int64
	db.Model(&entity.User{}).Where("role = ?", entity.RoleAdmin).Count(&count)
	if count > 0 {
		return nil
	}

	pw, err := hash.Password("Admin@MediSave2024")
	if err != nil {
		return err
	}

	admin := &entity.User{
		UUID:         uuid.NewString(),
		FirstName:    "System",
		LastName:     "Administrator",
		Email:        "admin@medisave.ng",
		Phone:        "+2348000000000",
		PasswordHash: pw,
		Role:         entity.RoleAdmin,
		IsVerified:   true,
		IsActive:     true,
	}

	if err := db.WithContext(context.Background()).Create(admin).Error; err != nil {
		return err
	}

	logger.Info("admin seeded", zap.String("email", admin.Email))
	return nil
}

// ─── DOCTORS ─────────────────────────────────────────────────────────────────

type doctorSeed struct {
	firstName   string
	lastName    string
	email       string
	phone       string
	license     string
	specialty   string
	subSpec     string
	hospital    string
	fee         float64
	experience  int
	bio         string
}

var doctorSeeds = []doctorSeed{
	{
		firstName: "Chukwuemeka", lastName: "Obi",
		email: "dr.obi@medisave.ng", phone: "+2348012345678",
		license: "MDCN-2018-001234", specialty: "General Practice",
		hospital: "National Hospital Abuja", fee: 5000, experience: 8,
		bio: "Experienced GP with focus on preventive care and chronic disease management.",
	},
	{
		firstName: "Amina", lastName: "Bello",
		email: "dr.bello@medisave.ng", phone: "+2348023456789",
		license: "MDCN-2015-005678", specialty: "Cardiology",
		subSpec: "Heart Failure", hospital: "University of Abuja Teaching Hospital",
		fee: 15000, experience: 11,
		bio: "Consultant cardiologist specializing in heart failure and hypertension.",
	},
	{
		firstName: "Ngozi", lastName: "Eze",
		email: "dr.eze@medisave.ng", phone: "+2348034567890",
		license: "MDCN-2019-009012", specialty: "Pediatrics",
		hospital: "Garki Hospital Abuja", fee: 7500, experience: 7,
		bio: "Dedicated pediatrician committed to child health and development.",
	},
	{
		firstName: "Babatunde", lastName: "Adeyemi",
		email: "dr.adeyemi@medisave.ng", phone: "+2348045678901",
		license: "MDCN-2012-003456", specialty: "Obstetrics & Gynaecology",
		hospital: "Maitama District Hospital", fee: 12000, experience: 14,
		bio: "Specialist in maternal health and high-risk pregnancies.",
	},
	{
		firstName: "Fatima", lastName: "Aliyu",
		email: "dr.aliyu@medisave.ng", phone: "+2348056789012",
		license: "MDCN-2020-007890", specialty: "Dermatology",
		hospital: "Private Practice, Wuse 2 Abuja", fee: 10000, experience: 6,
		bio: "Skin health specialist treating acne, eczema, and cosmetic concerns.",
	},
}

func seedDoctors(db *gorm.DB) error {
	pw, err := hash.Password("Doctor@MediSave2024")
	if err != nil {
		return err
	}

	for _, s := range doctorSeeds {
		var count int64
		db.Model(&entity.User{}).Where("email = ?", s.email).Count(&count)
		if count > 0 {
			continue
		}

		user := &entity.User{
			UUID:         uuid.NewString(),
			FirstName:    s.firstName,
			LastName:     s.lastName,
			Email:        s.email,
			Phone:        s.phone,
			PasswordHash: pw,
			Role:         entity.RoleDoctor,
			IsVerified:   true,
			IsActive:     true,
		}

		if err := db.Create(user).Error; err != nil {
			return err
		}

		doctor := &entity.Doctor{
			UserID:             user.ID,
			LicenseNumber:      s.license,
			Specialty:          s.specialty,
			SubSpecialty:       s.subSpec,
			YearsOfExperience:  s.experience,
			Hospital:           s.hospital,
			ConsultationFee:    s.fee,
			IsAvailable:        true,
			Status:             entity.DoctorStatusVerified,
			Bio:                s.bio,
			Languages:          "English, Hausa, Yoruba, Igbo",
			Rating:             4.5,
			TotalReviews:       12,
			TotalConsultations: 48,
		}

		if err := db.Create(doctor).Error; err != nil {
			return err
		}

		wallet := &entity.Wallet{
			UserID:    user.ID,
			OwnerType: entity.WalletOwnerDoctor,
			Balance:   0,
			Currency:  "NGN",
			IsActive:  true,
		}

		if err := db.Create(wallet).Error; err != nil {
			return err
		}

		logger.Info("doctor seeded", zap.String("name", user.FullName()))
	}

	return nil
}

// ─── PATIENTS ────────────────────────────────────────────────────────────────

type patientSeed struct {
	firstName string
	lastName  string
	email     string
	phone     string
	gender    string
	blood     entity.BloodGroup
	state     string
	lga       string
	dob       time.Time
}

var patientSeeds = []patientSeed{
	{
		firstName: "Emeka", lastName: "Okafor",
		email: "emeka@test.medisave.ng", phone: "+2348060000001",
		gender: "male", blood: entity.BloodGroupOPos,
		state: "FCT", lga: "Abuja Municipal",
		dob: time.Date(1990, 3, 15, 0, 0, 0, 0, time.UTC),
	},
	{
		firstName: "Blessing", lastName: "Nwosu",
		email: "blessing@test.medisave.ng", phone: "+2348060000002",
		gender: "female", blood: entity.BloodGroupAPos,
		state: "FCT", lga: "Gwagwalada",
		dob: time.Date(1995, 7, 22, 0, 0, 0, 0, time.UTC),
	},
	{
		firstName: "Ibrahim", lastName: "Mohammed",
		email: "ibrahim@test.medisave.ng", phone: "+2348060000003",
		gender: "male", blood: entity.BloodGroupBPos,
		state: "FCT", lga: "Kuje",
		dob: time.Date(1985, 11, 8, 0, 0, 0, 0, time.UTC),
	},
}

func seedPatients(db *gorm.DB) error {
	pw, err := hash.Password("Patient@MediSave2024")
	if err != nil {
		return err
	}

	for _, s := range patientSeeds {
		var count int64
		db.Model(&entity.User{}).Where("email = ?", s.email).Count(&count)
		if count > 0 {
			continue
		}

		user := &entity.User{
			UUID:         uuid.NewString(),
			FirstName:    s.firstName,
			LastName:     s.lastName,
			Email:        s.email,
			Phone:        s.phone,
			PasswordHash: pw,
			Role:         entity.RolePatient,
			IsVerified:   true,
			IsActive:     true,
		}

		if err := db.Create(user).Error; err != nil {
			return err
		}

		dob := s.dob
		patient := &entity.Patient{
			UserID:      user.ID,
			DateOfBirth: &dob,
			Gender:      s.gender,
			BloodGroup:  s.blood,
			Genotype:    "AA",
			State:       s.state,
			LGA:         s.lga,
			HealthScore: 75,
		}

		if err := db.Create(patient).Error; err != nil {
			return err
		}

		wallet := &entity.Wallet{
			UserID:    user.ID,
			OwnerType: entity.WalletOwnerPatient,
			Balance:   10000,
			Currency:  "NGN",
			IsActive:  true,
		}

		if err := db.Create(wallet).Error; err != nil {
			return err
		}

		contact := &entity.EmergencyContact{
			PatientID:    patient.ID,
			Name:         "Family Contact",
			Phone:        "+2348099999999",
			Relationship: "Family",
			IsPrimary:    true,
		}

		if err := db.Create(contact).Error; err != nil {
			return err
		}

		logger.Info("patient seeded", zap.String("name", user.FullName()))
	}

	return nil
}
