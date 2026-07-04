package bootstrap

import (
	"gorm.io/gorm"

	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/infrastructure/paystack"
	"github.com/medisave/app/internal/infrastructure/rtc"
	"github.com/medisave/app/internal/presentation/http/handler"
	"github.com/medisave/app/internal/presentation/http/router"
	repo "github.com/medisave/app/internal/infrastructure/repository"
	aiclient   "github.com/medisave/app/internal/infrastructure/external/ai"
	lkclient   "github.com/medisave/app/internal/infrastructure/external/livekit"
	mapsclient "github.com/medisave/app/internal/infrastructure/external/maps"
	smsclient  "github.com/medisave/app/internal/infrastructure/external/sms"
	pkgjwt "github.com/medisave/app/pkg/jwt"
)

func NewContainer(db *gorm.DB, cfg *config.Config, jwtManager *pkgjwt.Manager) *router.Handlers {
	// ─── REPOSITORIES ────────────────────────────────────────────────────────
	userRepo    := repo.NewGORMUserRepository(db)
	patientRepo := repo.NewGORMPatientRepository(db)
	doctorRepo  := repo.NewGORMDoctorRepository(db)
	walletRepo  := repo.NewGORMWalletRepository(db)
	txRepo      := repo.NewGORMTransactionRepository(db)
	notifRepo   := repo.NewGORMNotificationRepository(db)
	savingsRepo := repo.NewGORMSavingsRepository(db)
	apptRepo    := repo.NewGORMAppointmentRepository(db)
	consultRepo := repo.NewGORMConsultationRepository(db)
	prescRepo   := repo.NewGORMPrescriptionRepository(db)
	reviewRepo  := repo.NewGORMReviewRepository(db)
	aiRepo           := repo.NewGORMAIRepository(db)
	emergencyRepo    := repo.NewGORMEmergencyRepository(db)
	emergencyContRepo := repo.NewGORMEmergencyContactRepository(db)
	reminderRepo      := repo.NewGORMReminderRepository(db)
	campaignRepo      := repo.NewGORMCampaignRepository(db)
	ussdRepo          := repo.NewGORMUSSDRepository(db)
	roomRepo          := repo.NewGORMConsultationRoomRepository(db)
	txer              := repo.NewGORMTransactor(db)

	// ─── INFRASTRUCTURE ──────────────────────────────────────────────────────
	paystackClient := paystack.NewClient(cfg.Paystack.SecretKey, cfg.Paystack.BaseURL)
	aiClient       := aiclient.NewClient(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.TimeoutSeconds)
	lkClient       := lkclient.NewClient(cfg.LiveKit.WSURL, cfg.LiveKit.APIKey, cfg.LiveKit.APISecret)
	mapsClient     := mapsclient.NewClient(cfg.Maps.PlacesAPIKey)
	smsClient      := smsclient.NewClient(cfg.SMS.APIKey, cfg.SMS.Username, cfg.SMS.SenderID, cfg.SMS.GatewayURL)

	// ─── SERVICES ────────────────────────────────────────────────────────────
	authSvc    := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager)
	notifSvc   := service.NewNotificationService(notifRepo)
	patientSvc := service.NewPatientService(patientRepo, userRepo, walletRepo, notifRepo)
	doctorSvc  := service.NewDoctorService(doctorRepo, userRepo, walletRepo, notifRepo, apptRepo)
	walletSvc  := service.NewWalletService(walletRepo, txRepo, savingsRepo, patientRepo, paystackClient)
	apptSvc    := service.NewAppointmentService(apptRepo, consultRepo, prescRepo, reviewRepo, patientRepo, doctorRepo, walletRepo, txRepo, notifRepo, txer)
	aiSvc      := service.NewAIService(aiRepo, patientRepo, aiClient)
	recordSvc     := service.NewMedicalRecordService(repo.NewGORMMedicalRecordRepository(db), prescRepo, patientRepo)
	emergencySvc  := service.NewEmergencyService(emergencyRepo, emergencyContRepo, patientRepo, notifRepo)
	reminderSvc   := service.NewReminderService(reminderRepo, patientRepo)
	adminSvc      := service.NewAdminService(patientRepo, doctorRepo, apptRepo, txRepo, emergencyRepo, notifRepo, campaignRepo, smsClient)
	ussdSvc       := service.NewUSSDService(ussdRepo, userRepo, patientRepo, doctorRepo, walletSvc, apptSvc, emergencySvc, reminderSvc, mapsClient)
	roomSvc       := service.NewConsultationRoomService(roomRepo, apptRepo, patientRepo, doctorRepo, lkClient)

	// ─── RTC ─────────────────────────────────────────────────────────────────
	rtcHub := rtc.NewHub()

	// ─── HANDLERS ────────────────────────────────────────────────────────────
	return &router.Handlers{
		Page:         handler.NewPageHandler(),
		Auth:         handler.NewAuthHandler(authSvc),
		Patient:      handler.NewPatientHandler(patientSvc, notifSvc),
		Doctor:       handler.NewDoctorHandler(doctorSvc),
		Wallet:       handler.NewWalletHandler(walletSvc),
		Appointment:  handler.NewAppointmentHandler(apptSvc, roomSvc, rtcHub),
		Consultation: handler.NewConsultationHandler(apptSvc),
		Record:       handler.NewMedicalRecordHandler(recordSvc),
		AI:           handler.NewAIHandler(aiSvc),
		Emergency:    handler.NewEmergencyHandler(emergencySvc),
		Reminder:     handler.NewReminderHandler(reminderSvc),
		Maps:         handler.NewMapsHandler(mapsClient),
		SMS:          handler.NewSMSHandler(smsClient),
		USSD:         handler.NewUSSDHandler(ussdSvc),
		Admin:        handler.NewAdminHandler(adminSvc),
		Call:         handler.NewCallHandler(rtcHub, jwtManager),
		Room:         handler.NewRoomHandler(roomSvc),
	}
}
