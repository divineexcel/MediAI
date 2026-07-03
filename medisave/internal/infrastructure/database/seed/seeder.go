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
// If the database was seeded with an older version of this file, the repair helpers
// will bring the existing rows into the state the current code expects.
func Run(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entity.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("check users existence: %w", err)
	}
	if count > 0 {
		logger.Info("database already has data, skipping seeding")
		return nil
	}

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
	if err := seedAppointments(db); err != nil {
		return fmt.Errorf("seed appointments: %w", err)
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
	firstName  string
	lastName   string
	email      string
	phone      string
	license    string
	specialty  string
	subSpec    string
	hospital   string
	fee        float64
	experience int
	bio        string
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
			// Patient already exists — repair deposit transaction if it was missing
			// from an earlier version of the seeder that did not create it.
			if repairErr := repairPatientDeposit(db, s.email); repairErr != nil {
				logger.Warn("could not repair patient deposit",
					zap.String("email", s.email),
					zap.Error(repairErr),
				)
			}
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
			Balance:   50000,
			Currency:  "NGN",
			IsActive:  true,
		}

		if err := db.Create(wallet).Error; err != nil {
			return err
		}

		depositTx := &entity.Transaction{
			Reference:     "SEED-DEPOSIT-" + user.UUID[:8],
			WalletID:      wallet.ID,
			Type:          entity.TxTypeDeposit,
			Amount:        50000,
			BalanceBefore: 0,
			BalanceAfter:  50000,
			Status:        entity.TxStatusSuccess,
			Description:   "Welcome bonus — initial wallet funding",
		}
		if err := db.Create(depositTx).Error; err != nil {
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

// repairPatientDeposit ensures a patient seeded by an older version of this file
// (which omitted the deposit transaction) gets the deposit and correct wallet balance.
// It is a no-op when the deposit already exists.
func repairPatientDeposit(db *gorm.DB, email string) error {
	var user entity.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return err
	}

	var wallet entity.Wallet
	if err := db.Where("user_id = ?", user.ID).First(&wallet).Error; err != nil {
		return err
	}

	var txCount int64
	db.Model(&entity.Transaction{}).
		Where("wallet_id = ? AND type = ?", wallet.ID, entity.TxTypeDeposit).
		Count(&txCount)
	if txCount > 0 {
		return nil // deposit already present — nothing to repair
	}

	// Create the missing deposit and bring the wallet to the expected 50 000 balance.
	depositTx := &entity.Transaction{
		Reference:     "SEED-DEPOSIT-" + user.UUID[:8],
		WalletID:      wallet.ID,
		Type:          entity.TxTypeDeposit,
		Amount:        50000,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  50000,
		Status:        entity.TxStatusSuccess,
		Description:   "Welcome bonus — initial wallet funding (repair)",
	}
	if err := db.Create(depositTx).Error; err != nil {
		return err
	}

	if err := db.Model(&entity.Wallet{}).
		Where("id = ?", wallet.ID).
		Update("balance", 50000).Error; err != nil {
		return err
	}

	logger.Info("repaired patient wallet deposit",
		zap.String("email", email),
		zap.Float64("old_balance", wallet.Balance),
		zap.Float64("new_balance", 50000),
	)
	return nil
}

// ─── APPOINTMENTS ─────────────────────────────────────────────────────────────

type apptDef struct {
	patientEmail string
	doctorEmail  string
	apptType     entity.AppointmentType
	status       entity.AppointmentStatus
	offsetDays   int // negative = past, positive = future
	complaint    string
	notes        string
}

var appointmentSeeds = []apptDef{
	{
		patientEmail: "emeka@test.medisave.ng",
		doctorEmail:  "dr.obi@medisave.ng",
		apptType:     entity.AppointmentTypeVideo,
		status:       entity.AppointmentStatusCompleted,
		offsetDays:   -14,
		complaint:    "Persistent headache and fatigue for 2 weeks",
		notes:        "Tension headaches likely stress-related. Recommended rest, hydration, and ibuprofen as needed. Review in 4 weeks.",
	},
	{
		patientEmail: "emeka@test.medisave.ng",
		doctorEmail:  "dr.bello@medisave.ng",
		apptType:     entity.AppointmentTypeVideo,
		status:       entity.AppointmentStatusCompleted,
		offsetDays:   -7,
		complaint:    "Intermittent chest tightness",
		notes:        "ECG and BP within normal range. Likely anxiety-induced. Follow up in 4 weeks if symptoms persist.",
	},
	{
		patientEmail: "blessing@test.medisave.ng",
		doctorEmail:  "dr.obi@medisave.ng",
		apptType:     entity.AppointmentTypeChat,
		status:       entity.AppointmentStatusCompleted,
		offsetDays:   -10,
		complaint:    "High fever and body aches for 3 days",
		notes:        "Malaria RDT positive. Prescribed artemether-lumefantrine 6-dose pack. Advised rest, fluids, and repeat RDT after 48h.",
	},
	{
		patientEmail: "blessing@test.medisave.ng",
		doctorEmail:  "dr.eze@medisave.ng",
		apptType:     entity.AppointmentTypeChat,
		status:       entity.AppointmentStatusConfirmed,
		offsetDays:   2,
		complaint:    "Routine child wellness and vaccination check",
		notes:        "",
	},
	{
		patientEmail: "ibrahim@test.medisave.ng",
		doctorEmail:  "dr.aliyu@medisave.ng",
		apptType:     entity.AppointmentTypeVideo,
		status:       entity.AppointmentStatusCompleted,
		offsetDays:   -5,
		complaint:    "Itchy skin rash spreading on arms and torso",
		notes:        "Allergic contact dermatitis. Prescribed topical hydrocortisone 1% cream and cetirizine 10mg daily for 7 days.",
	},
	{
		patientEmail: "ibrahim@test.medisave.ng",
		doctorEmail:  "dr.obi@medisave.ng",
		apptType:     entity.AppointmentTypeChat,
		status:       entity.AppointmentStatusPending,
		offsetDays:   3,
		complaint:    "Annual health screening",
		notes:        "",
	},
}

func seedAppointments(db *gorm.DB) error {
	var count int64
	db.Model(&entity.Appointment{}).Count(&count)
	if count > 0 {
		return nil
	}

	type patRec struct {
		patientID uint
		walletID  uint
	}
	patMap := make(map[string]patRec)
	for _, ps := range patientSeeds {
		var u entity.User
		if err := db.Where("email = ?", ps.email).First(&u).Error; err != nil {
			continue
		}
		var p entity.Patient
		if err := db.Where("user_id = ?", u.ID).First(&p).Error; err != nil {
			continue
		}
		var w entity.Wallet
		if err := db.Where("user_id = ?", u.ID).First(&w).Error; err != nil {
			continue
		}
		patMap[ps.email] = patRec{patientID: p.ID, walletID: w.ID}
	}

	type docRec struct {
		doctorID uint
		fee      float64
	}
	docMap := make(map[string]docRec)
	for _, ds := range doctorSeeds {
		var u entity.User
		if err := db.Where("email = ?", ds.email).First(&u).Error; err != nil {
			continue
		}
		var d entity.Doctor
		if err := db.Where("user_id = ?", u.ID).First(&d).Error; err != nil {
			continue
		}
		docMap[ds.email] = docRec{doctorID: d.ID, fee: d.ConsultationFee}
	}

	now := time.Now()

	for i, a := range appointmentSeeds {
		pat, ok := patMap[a.patientEmail]
		if !ok {
			continue
		}
		doc, ok := docMap[a.doctorEmail]
		if !ok {
			continue
		}

		scheduledAt := now.AddDate(0, 0, a.offsetDays)

		appt := &entity.Appointment{
			PatientID:       pat.patientID,
			DoctorID:        doc.doctorID,
			Type:            a.apptType,
			Status:          a.status,
			ScheduledAt:     scheduledAt,
			ConsultationFee: doc.fee,
			ChiefComplaint:  a.complaint,
			Notes:           a.notes,
		}

		if a.status == entity.AppointmentStatusCompleted {
			startedAt := scheduledAt
			completedAt := scheduledAt.Add(30 * time.Minute)
			appt.StartedAt = &startedAt
			appt.CompletedAt = &completedAt
		}

		if err := db.Create(appt).Error; err != nil {
			return err
		}

		if a.status == entity.AppointmentStatusCompleted {
			// Re-fetch the wallet to get the current live balance — a prior loop
			// iteration may have debited it.
			var w entity.Wallet
			if err := db.First(&w, pat.walletID).Error; err != nil {
				return err
			}

			if w.Balance < doc.fee {
				// Wallet has insufficient funds for this seed payment.
				// Log a warning and skip the debit rather than failing the entire
				// seeder and leaving appointments in a partially-seeded state.
				logger.Warn("skipping seed appointment payment: insufficient wallet balance",
					zap.String("patient", a.patientEmail),
					zap.String("doctor", a.doctorEmail),
					zap.Float64("balance", w.Balance),
					zap.Float64("fee", doc.fee),
				)
				continue
			}

			newBalance := w.Balance - doc.fee
			ref := fmt.Sprintf("SEED-APPT-%04d-%02d", appt.ID, i)
			payTx := &entity.Transaction{
				Reference:       ref,
				WalletID:        pat.walletID,
				Type:            entity.TxTypePayment,
				Amount:          doc.fee,
				BalanceBefore:   w.Balance,
				BalanceAfter:    newBalance,
				Status:          entity.TxStatusSuccess,
				Description:     fmt.Sprintf("Consultation payment — %s", a.doctorEmail),
				RelatedEntityID: appt.ID,
			}
			if err := db.Create(payTx).Error; err != nil {
				return err
			}
			if err := db.Model(&entity.Wallet{}).Where("id = ?", pat.walletID).
				Update("balance", newBalance).Error; err != nil {
				return err
			}
		}

		logger.Info("appointment seeded",
			zap.String("patient", a.patientEmail),
			zap.String("doctor", a.doctorEmail),
			zap.String("status", string(a.status)),
		)
	}

	return nil
}
