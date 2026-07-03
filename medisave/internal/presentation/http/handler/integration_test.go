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

	authSvc := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager)
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
