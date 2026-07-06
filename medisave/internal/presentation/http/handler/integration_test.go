package handler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/infrastructure/database/migrations"
	repo "github.com/medisave/app/internal/infrastructure/repository"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/pagination"
)

func TestIntegration_DatabasePersistenceAndTransactions(t *testing.T) {
	// Create a temp directory for the test database
	tempDir, err := os.MkdirTemp("", "medisave-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "medisave_test.db")

	// Helper function to open connection and run migrations
	connectDB := func() *gorm.DB {
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		require.NoError(t, err)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		db.Exec("PRAGMA foreign_keys=ON")
		return db
	}

	// 1. Initialize first DB connection and migrate
	db := connectDB()
	require.NoError(t, migrations.Run(db))

	// Instantiate services
	jwtManager := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)
	userRepo := repo.NewGORMUserRepository(db)
	patientRepo := repo.NewGORMPatientRepository(db)
	doctorRepo := repo.NewGORMDoctorRepository(db)
	walletRepo := repo.NewGORMWalletRepository(db)
	txRepo := repo.NewGORMTransactionRepository(db)
	notifRepo := repo.NewGORMNotificationRepository(db)
	apptRepo := repo.NewGORMAppointmentRepository(db)
	consultRepo := repo.NewGORMConsultationRepository(db)
	prescRepo := repo.NewGORMPrescriptionRepository(db)
	reviewRepo := repo.NewGORMReviewRepository(db)
	txer := repo.NewGORMTransactor(db)

	authSvc := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager, txer)
	apptSvc := service.NewAppointmentService(
		apptRepo, consultRepo, prescRepo, reviewRepo,
		patientRepo, doctorRepo, walletRepo, txRepo, notifRepo, txer,
	)

	ctx := context.Background()

	// 2. Register Patient
	patientReg := &dto.RegisterRequest{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@example.com",
		Password:  "SecurePassword@123",
		Phone:     "+2348033333333",
		Role:      "patient",
	}
	patientResp, err := authSvc.RegisterPatient(ctx, patientReg)
	require.NoError(t, err)
	assert.Equal(t, "john.doe@example.com", patientResp.User.Email)

	// 3. Register Doctor
	doctorReg := &dto.DoctorRegisterRequest{
		RegisterRequest: dto.RegisterRequest{
			FirstName: "Jane",
			LastName:  "Smith",
			Email:     "jane.smith@example.com",
			Password:  "SecurePassword@123",
			Phone:     "+2348044444444",
			Role:      "doctor",
		},
		LicenseNumber:     "MD12345",
		Specialty:         "General",
		YearsOfExperience: 5,
		ConsultationFee:   5000.0,
		Hospital:          "General Hospital",
		WorkIDURL:         "http://example.com/workid.jpg",
		MedicalLicenseURL: "http://example.com/license.jpg",
	}
	doctorResp, err := authSvc.RegisterDoctor(ctx, doctorReg)
	require.NoError(t, err)
	assert.Equal(t, "jane.smith@example.com", doctorResp.User.Email)

	// Verify doctor status is pending, verify doctor user id
	docRecord, err := doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusPending, docRecord.Status)

	// Make doctor verified and available for the test
	docRecord.Status = entity.DoctorStatusVerified
	docRecord.IsAvailable = true
	require.NoError(t, doctorRepo.Update(ctx, docRecord))

	// Find patient wallet and credit balance for appointment booking
	pWallet, err := walletRepo.FindByUserID(ctx, patientResp.User.ID)
	require.NoError(t, err)
	require.NoError(t, walletRepo.UpdateBalance(ctx, pWallet.ID, 10000.0))

	// 4. Book Appointment (uses WithinTransaction internally)
	bookReq := &dto.BookAppointmentRequest{
		DoctorID:       docRecord.ID,
		Type:           "video",
		ScheduledAt:    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		ChiefComplaint: "Headache",
	}
	bookResp, err := apptSvc.Book(ctx, patientResp.User.ID, bookReq)
	require.NoError(t, err)
	assert.NotEmpty(t, bookResp.Appointment.ID)
	assert.Equal(t, 5000.0, bookResp.Appointment.ConsultationFee)

	// Check final balances
	pWalletAfter, err := walletRepo.FindByUserID(ctx, patientResp.User.ID)
	require.NoError(t, err)
	// Wallet had 0 initially, credited with 10,000, debited 5,000 => final should be 5,000
	assert.Equal(t, 5000.0, pWalletAfter.Balance)

	docWalletAfter, err := walletRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	// Doctor wallet had 0 initially, credited with 5,000 => final should be 5,000
	assert.Equal(t, 5000.0, docWalletAfter.Balance)

	// Check that transaction logs were created
	txCount, err := txRepo.CountAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), txCount) // 1 debit transaction + 1 credit transaction

	// Close database connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	// 5. RESTART SIMULATION: Connect to the same SQLite file again
	dbRestarted := connectDB()
	defer func() {
		sDB, _ := dbRestarted.DB()
		if sDB != nil {
			sDB.Close()
		}
	}()

	userRepoR := repo.NewGORMUserRepository(dbRestarted)
	patientRepoR := repo.NewGORMPatientRepository(dbRestarted)
	doctorRepoR := repo.NewGORMDoctorRepository(dbRestarted)
	walletRepoR := repo.NewGORMWalletRepository(dbRestarted)
	txRepoR := repo.NewGORMTransactionRepository(dbRestarted)
	apptRepoR := repo.NewGORMAppointmentRepository(dbRestarted)

	// 6. VERIFY PERSISTENCE
	// Verify User & Patient records
	userP, err := userRepoR.FindByEmail(ctx, "john.doe@example.com")
	require.NoError(t, err)
	assert.Equal(t, "John", userP.FirstName)

	patientR, err := patientRepoR.FindByUserID(ctx, userP.ID)
	require.NoError(t, err)
	assert.Equal(t, userP.ID, patientR.UserID)

	// Verify Doctor record
	userD, err := userRepoR.FindByEmail(ctx, "jane.smith@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Jane", userD.FirstName)

	doctorR, err := doctorRepoR.FindByUserID(ctx, userD.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusVerified, doctorR.Status)

	// Verify Wallet balances
	walletP, err := walletRepoR.FindByUserID(ctx, userP.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, walletP.Balance)

	walletD, err := walletRepoR.FindByUserID(ctx, userD.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, walletD.Balance)

	// Verify Appointment persistence
	apptCount, err := apptRepoR.CountAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), apptCount)

	// Verify Transaction logs persistence
	txCountR, err := txRepoR.CountAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), txCountR)
}

func TestIntegration_DoctorCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "medisave-cancel-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "medisave_test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, migrations.Run(db))
	defer sqlDB.Close()

	jwtManager := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)
	userRepo := repo.NewGORMUserRepository(db)
	patientRepo := repo.NewGORMPatientRepository(db)
	doctorRepo := repo.NewGORMDoctorRepository(db)
	walletRepo := repo.NewGORMWalletRepository(db)
	txRepo := repo.NewGORMTransactionRepository(db)
	notifRepo := repo.NewGORMNotificationRepository(db)
	apptRepo := repo.NewGORMAppointmentRepository(db)
	consultRepo := repo.NewGORMConsultationRepository(db)
	prescRepo := repo.NewGORMPrescriptionRepository(db)
	reviewRepo := repo.NewGORMReviewRepository(db)
	txer := repo.NewGORMTransactor(db)

	authSvc := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager, txer)
	apptSvc := service.NewAppointmentService(
		apptRepo, consultRepo, prescRepo, reviewRepo,
		patientRepo, doctorRepo, walletRepo, txRepo, notifRepo, txer,
	)

	ctx := context.Background()

	// Register Patient
	pResp, err := authSvc.RegisterPatient(ctx, &dto.RegisterRequest{
		FirstName: "John", LastName: "Doe", Email: "john@example.com", Password: "SecurePassword@123", Phone: "+2348011111111", Role: "patient",
	})
	require.NoError(t, err)

	// Register Doctor
	dResp, err := authSvc.RegisterDoctor(ctx, &dto.DoctorRegisterRequest{
		RegisterRequest: dto.RegisterRequest{
			FirstName: "Jane", LastName: "Smith", Email: "jane@example.com", Password: "SecurePassword@123", Phone: "+2348022222222", Role: "doctor",
		},
		LicenseNumber: "MD111", Specialty: "GP", YearsOfExperience: 3, ConsultationFee: 2000.0, Hospital: "Gen",
	})
	require.NoError(t, err)

	// Verify and set doctor available
	doc, err := doctorRepo.FindByUserID(ctx, dResp.User.ID)
	require.NoError(t, err)
	doc.Status = entity.DoctorStatusVerified
	doc.IsAvailable = true
	require.NoError(t, doctorRepo.Update(ctx, doc))

	// Credit patient balance
	pWallet, err := walletRepo.FindByUserID(ctx, pResp.User.ID)
	require.NoError(t, err)
	require.NoError(t, walletRepo.UpdateBalance(ctx, pWallet.ID, 5000.0))

	// Book appointment
	bookResp, err := apptSvc.Book(ctx, pResp.User.ID, &dto.BookAppointmentRequest{
		DoctorID: doc.ID, Type: "video", ScheduledAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339), ChiefComplaint: "Fever",
	})
	require.NoError(t, err)

	// Doctor cancels appointment
	err = apptSvc.Cancel(ctx, dResp.User.ID, entity.RoleDoctor, bookResp.Appointment.ID, "Doctor urgent surgery")
	require.NoError(t, err)

	// Verify status is cancelled
	appt, err := apptRepo.FindByID(ctx, bookResp.Appointment.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.AppointmentStatusCancelled, appt.Status)
	assert.Equal(t, "Doctor urgent surgery", appt.CancelReason)

	// Verify refund: Patient balance should be back to 5000.0, Doctor balance should be 0.0
	pWalletAfter, err := walletRepo.FindByUserID(ctx, pResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, pWalletAfter.Balance)

	dWalletAfter, err := walletRepo.FindByUserID(ctx, dResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, 0.0, dWalletAfter.Balance)
}

func TestIntegration_DoctorVerificationAndRejectionWorkflow(t *testing.T) {
	// Create a temp directory for the test database
	tempDir, err := os.MkdirTemp("", "medisave-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "medisave_test.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	db.Exec("PRAGMA foreign_keys=ON")

	require.NoError(t, migrations.Run(db))

	jwtManager := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)
	userRepo := repo.NewGORMUserRepository(db)
	patientRepo := repo.NewGORMPatientRepository(db)
	doctorRepo := repo.NewGORMDoctorRepository(db)
	walletRepo := repo.NewGORMWalletRepository(db)
	notifRepo := repo.NewGORMNotificationRepository(db)
	txRepo := repo.NewGORMTransactionRepository(db)
	apptRepo := repo.NewGORMAppointmentRepository(db)
	emergencyRepo := repo.NewGORMEmergencyRepository(db)
	campaignRepo := repo.NewGORMCampaignRepository(db)

	txer := repo.NewGORMTransactor(db)
	authSvc := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager, txer)
	adminSvc := service.NewAdminService(patientRepo, doctorRepo, userRepo, apptRepo, txRepo, emergencyRepo, notifRepo, campaignRepo, nil, txer)
	doctorSvc := service.NewDoctorService(doctorRepo, userRepo, walletRepo, notifRepo, apptRepo)

	ctx := context.Background()

	// 1. Register Doctor (starts as pending)
	doctorReg := &dto.DoctorRegisterRequest{
		RegisterRequest: dto.RegisterRequest{
			FirstName: "Dave",
			LastName:  "Reid",
			Email:     "dave.reid@example.com",
			Password:  "SecurePassword@123",
			Phone:     "+2348055555555",
			Role:      "doctor",
		},
		LicenseNumber:     "MD9999",
		Specialty:         "Pediatrics",
		YearsOfExperience: 8,
		ConsultationFee:   6000.0,
		Hospital:          "City Pediatrics",
		WorkIDURL:         "http://example.com/workid.jpg",
		MedicalLicenseURL: "http://example.com/license.jpg",
	}
	doctorResp, err := authSvc.RegisterDoctor(ctx, doctorReg)
	require.NoError(t, err)

	// Verify initial status is pending
	docRecord, err := doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusPending, docRecord.Status)
	assert.Empty(t, docRecord.Remarks)

	// Trying to toggle availability when pending should fail
	err = doctorSvc.ToggleAvailability(ctx, doctorResp.User.ID, true)
	assert.Error(t, err)

	// 2. Reject doctor via Admin Service
	rejectReq := &dto.VerifyDoctorRequest{
		Status:  "rejected",
		Remarks: "Credentials do not match the public database. Please upload a valid license.",
	}
	err = adminSvc.VerifyDoctor(ctx, docRecord.ID, rejectReq)
	require.NoError(t, err)

	// Verify rejected status and remarks
	docRecord, err = doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusRejected, docRecord.Status)
	assert.Equal(t, "Credentials do not match the public database. Please upload a valid license.", docRecord.Remarks)

	// Trying to toggle availability when rejected should fail
	err = doctorSvc.ToggleAvailability(ctx, doctorResp.User.ID, true)
	assert.Error(t, err)

	// Verify notification generated
	notifications, _, err := notifRepo.ListByUser(ctx, doctorResp.User.ID, pagination.Params{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.Contains(t, notifications[0].Title, "Rejected")
	assert.Contains(t, notifications[0].Body, "Feedback: Credentials do not match")

	// 3. Approve doctor via Admin Service
	approveReq := &dto.VerifyDoctorRequest{
		Status: "verified",
	}
	err = adminSvc.VerifyDoctor(ctx, docRecord.ID, approveReq)
	require.NoError(t, err)

	// Verify verified status
	docRecord, err = doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusVerified, docRecord.Status)

	// Toggle availability should succeed now
	err = doctorSvc.ToggleAvailability(ctx, doctorResp.User.ID, true)
	require.NoError(t, err)

	docRecord, err = doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.True(t, docRecord.IsAvailable)

	// 4. Suspend doctor via Admin Service
	suspendReq := &dto.VerifyDoctorRequest{
		Status:  "suspended",
		Remarks: "Complaints of unprofessional conduct",
	}
	err = adminSvc.VerifyDoctor(ctx, docRecord.ID, suspendReq)
	require.NoError(t, err)

	// Verify suspended status and remarks
	docRecord, err = doctorRepo.FindByUserID(ctx, doctorResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusSuspended, docRecord.Status)
	assert.Equal(t, "Complaints of unprofessional conduct", docRecord.Remarks)

	// Toggle availability when suspended should fail
	err = doctorSvc.ToggleAvailability(ctx, doctorResp.User.ID, true)
	assert.Error(t, err)
}

func TestIntegration_RegisterThenLoginAcrossRestart(t *testing.T) {
	// This test verifies that users registered before a DB restart
	// can successfully log in after reconnecting (simulating the dual-database fix).
	tempDir, err := os.MkdirTemp("", "medisave-reg-login-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "medisave_test.db")

	connectDB := func() *gorm.DB {
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		require.NoError(t, err)
		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		db.Exec("PRAGMA foreign_keys=ON")
		return db
	}

	// 1. First session: register users
	db1 := connectDB()
	require.NoError(t, migrations.Run(db1))

	jwtManager1 := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)
	userRepo1 := repo.NewGORMUserRepository(db1)
	patientRepo1 := repo.NewGORMPatientRepository(db1)
	doctorRepo1 := repo.NewGORMDoctorRepository(db1)
	walletRepo1 := repo.NewGORMWalletRepository(db1)
	txer1 := repo.NewGORMTransactor(db1)

	authSvc1 := service.NewAuthService(userRepo1, patientRepo1, doctorRepo1, walletRepo1, jwtManager1, txer1)

	ctx := context.Background()

	// Register Patient
	pResp, err := authSvc1.RegisterPatient(ctx, &dto.RegisterRequest{
		FirstName: "Alice",
		LastName:  "Johnson",
		Email:     "alice.j@example.com",
		Password:  "SecurePassword@123",
		Phone:     "+2348060000001",
		Role:      "patient",
	})
	require.NoError(t, err)
	require.NotEmpty(t, pResp.Tokens.AccessToken)

	// Register Doctor
	dResp, err := authSvc1.RegisterDoctor(ctx, &dto.DoctorRegisterRequest{
		RegisterRequest: dto.RegisterRequest{
			FirstName: "Bob",
			LastName:  "Williams",
			Email:     "bob.w@example.com",
			Password:  "SecurePassword@123",
			Phone:     "+2348060000002",
			Role:      "doctor",
		},
		LicenseNumber:     "MD-LOGIN-001",
		Specialty:         "Cardiology",
		YearsOfExperience: 10,
		ConsultationFee:   8000.0,
		Hospital:          "Heart Clinic",
	})
	require.NoError(t, err)
	require.NotEmpty(t, dResp.Tokens.AccessToken)

	// Close the first connection
	sqlDB1, err := db1.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB1.Close())

	// 2. Second session: login with same credentials on fresh connection
	db2 := connectDB()
	defer func() {
		sqlDB2, _ := db2.DB()
		if sqlDB2 != nil {
			sqlDB2.Close()
		}
	}()

	jwtManager2 := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)
	userRepo2 := repo.NewGORMUserRepository(db2)
	patientRepo2 := repo.NewGORMPatientRepository(db2)
	doctorRepo2 := repo.NewGORMDoctorRepository(db2)
	walletRepo2 := repo.NewGORMWalletRepository(db2)
	txer2 := repo.NewGORMTransactor(db2)

	authSvc2 := service.NewAuthService(userRepo2, patientRepo2, doctorRepo2, walletRepo2, jwtManager2, txer2)

	// Patient login after restart
	loginResp, err := authSvc2.Login(ctx, &dto.LoginRequest{
		Email:    "alice.j@example.com",
		Password: "SecurePassword@123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, loginResp.Tokens.AccessToken)
	assert.Equal(t, "alice.j@example.com", loginResp.User.Email)
	assert.Equal(t, entity.RolePatient, loginResp.User.Role)

	// Doctor login after restart
	docLoginResp, err := authSvc2.Login(ctx, &dto.LoginRequest{
		Email:    "bob.w@example.com",
		Password: "SecurePassword@123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, docLoginResp.Tokens.AccessToken)
	assert.Equal(t, "bob.w@example.com", docLoginResp.User.Email)
	assert.Equal(t, entity.RoleDoctor, docLoginResp.User.Role)

	// Verify doctor is pending and not yet available (registration default)
	docRecord, err := doctorRepo2.FindByUserID(ctx, docLoginResp.User.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.DoctorStatusPending, docRecord.Status)
	assert.False(t, docRecord.IsAvailable)
}
