package service

import (
	"context"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
	smsclient "github.com/medisave/app/internal/infrastructure/external/sms"
)

type AdminService interface {
	GetAnalytics(ctx context.Context) (*dto.AdminAnalyticsResponse, error)
	ListPatients(ctx context.Context, p pagination.Params) ([]*entity.Patient, int64, error)
	GetPatient(ctx context.Context, patientID uint) (*entity.Patient, error)
	ListDoctors(ctx context.Context, p pagination.Params) ([]*entity.Doctor, int64, error)
	GetDoctor(ctx context.Context, doctorID uint) (*entity.Doctor, error)
	VerifyDoctor(ctx context.Context, doctorID uint, req *dto.VerifyDoctorRequest) error
	ListTransactions(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int64, error)
	ListAppointments(ctx context.Context, p pagination.Params) ([]*entity.Appointment, int64, error)
	ListEmergencies(ctx context.Context) ([]*entity.Emergency, error)
	SendCampaign(ctx context.Context, req *dto.HealthCampaignRequest) error
}

type adminService struct {
	patientRepo     repository.PatientRepository
	doctorRepo      repository.DoctorRepository
	apptRepo        repository.AppointmentRepository
	txRepo          repository.TransactionRepository
	emergencyRepo   repository.EmergencyRepository
	notifRepo       repository.NotificationRepository
	smsClient       *smsclient.Client
}

func NewAdminService(
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	apptRepo repository.AppointmentRepository,
	txRepo repository.TransactionRepository,
	emergencyRepo repository.EmergencyRepository,
	notifRepo repository.NotificationRepository,
	smsClient *smsclient.Client,
) AdminService {
	return &adminService{
		patientRepo:   patientRepo,
		doctorRepo:    doctorRepo,
		apptRepo:      apptRepo,
		txRepo:        txRepo,
		emergencyRepo: emergencyRepo,
		notifRepo:     notifRepo,
		smsClient:     smsClient,
	}
}

func (s *adminService) GetAnalytics(ctx context.Context) (*dto.AdminAnalyticsResponse, error) {
	totalPatients, _ := s.patientRepo.CountAll(ctx)
	totalDoctors, _   := s.doctorRepo.CountAll(ctx)
	pendingDoctors, _ := s.doctorRepo.CountPending(ctx)
	totalAppts, _     := s.apptRepo.CountAll(ctx)
	totalTx, _        := s.txRepo.CountAll(ctx)
	totalVolume, _    := s.txRepo.TotalVolume(ctx)
	activeEmergencies, _ := s.emergencyRepo.CountActive(ctx)

	return &dto.AdminAnalyticsResponse{
		TotalPatients:        totalPatients,
		TotalDoctors:         totalDoctors,
		TotalAppointments:    totalAppts,
		TotalTransactions:    totalTx,
		TotalVolume:          totalVolume,
		PendingDoctors:       pendingDoctors,
		ActiveEmergencies:    activeEmergencies,
		AIConversationsToday: 0, // placeholder until AI repo gets CountToday
	}, nil
}

func (s *adminService) ListPatients(ctx context.Context, p pagination.Params) ([]*entity.Patient, int64, error) {
	return s.patientRepo.List(ctx, p)
}

func (s *adminService) GetPatient(ctx context.Context, patientID uint) (*entity.Patient, error) {
	patient, err := s.patientRepo.FindByID(ctx, patientID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}
	return patient, nil
}

func (s *adminService) ListDoctors(ctx context.Context, p pagination.Params) ([]*entity.Doctor, int64, error) {
	return s.doctorRepo.List(ctx, p)
}

func (s *adminService) GetDoctor(ctx context.Context, doctorID uint) (*entity.Doctor, error) {
	doctor, err := s.doctorRepo.FindByID(ctx, doctorID)
	if err != nil {
		return nil, pkgerrors.ErrDoctorNotFound
	}
	return doctor, nil
}

func (s *adminService) VerifyDoctor(ctx context.Context, doctorID uint, req *dto.VerifyDoctorRequest) error {
	doctor, err := s.doctorRepo.FindByID(ctx, doctorID)
	if err != nil {
		return pkgerrors.ErrDoctorNotFound
	}

	var status entity.DoctorStatus
	switch req.Status {
	case "verified":
		status = entity.DoctorStatusVerified
	case "suspended":
		status = entity.DoctorStatusSuspended
	default:
		return pkgerrors.ErrBadRequest
	}

	if err := s.doctorRepo.UpdateStatus(ctx, doctor.ID, status); err != nil {
		return pkgerrors.ErrInternalServer
	}

	// Notify doctor
	var title, body string
	if status == entity.DoctorStatusVerified {
		title = "Profile Verified"
		body = "Congratulations! Your MediSave doctor profile has been verified. You can now start receiving patient appointments."
	} else {
		title = "Profile Suspended"
		body = "Your MediSave doctor account has been suspended. Please contact support for details."
		if req.Remarks != "" {
			body += " Reason: " + req.Remarks
		}
	}

	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  doctor.UserID,
		Type:    entity.NotifTypeSystem,
		Title:   title,
		Body:    body,
		Channel: entity.ChannelInApp,
	})

	return nil
}

func (s *adminService) ListTransactions(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int64, error) {
	return s.txRepo.ListAll(ctx, p)
}

func (s *adminService) ListAppointments(ctx context.Context, p pagination.Params) ([]*entity.Appointment, int64, error) {
	return s.apptRepo.ListAll(ctx, p)
}

func (s *adminService) ListEmergencies(ctx context.Context) ([]*entity.Emergency, error) {
	return s.emergencyRepo.ListActive(ctx)
}

func (s *adminService) SendCampaign(ctx context.Context, req *dto.HealthCampaignRequest) error {
	// In production: fetch all user phones by role and send bulk SMS
	// For now: log the campaign and return success
	// This would also send push notifications via Firebase
	_ = req
	return nil
}
