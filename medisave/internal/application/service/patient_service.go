package service

import (
	"context"
	"strings"
	"time"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
)

type PatientService interface {
	GetProfile(ctx context.Context, userID uint) (*entity.Patient, error)
	UpdateProfile(ctx context.Context, userID uint, req *dto.UpdatePatientProfileRequest) (*entity.Patient, error)
	GetDashboard(ctx context.Context, userID uint) (*dto.PatientDashboardResponse, error)
}

type patientService struct {
	patientRepo  repository.PatientRepository
	userRepo     repository.UserRepository
	walletRepo   repository.WalletRepository
	notifRepo    repository.NotificationRepository
}

func NewPatientService(
	patientRepo repository.PatientRepository,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	notifRepo repository.NotificationRepository,
) PatientService {
	return &patientService{
		patientRepo: patientRepo,
		userRepo:    userRepo,
		walletRepo:  walletRepo,
		notifRepo:   notifRepo,
	}
}

func (s *patientService) GetProfile(ctx context.Context, userID uint) (*entity.Patient, error) {
	return s.patientRepo.FindByUserID(ctx, userID)
}

func (s *patientService) UpdateProfile(ctx context.Context, userID uint, req *dto.UpdatePatientProfileRequest) (*entity.Patient, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update User fields
	user := &patient.User
	if req.FirstName != "" {
		user.FirstName = strings.TrimSpace(req.FirstName)
	}
	if req.LastName != "" {
		user.LastName = strings.TrimSpace(req.LastName)
	}
	if req.Phone != "" {
		exists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone)
		if exists && user.Phone != req.Phone {
			return nil, pkgerrors.ErrPhoneExists
		}
		user.Phone = req.Phone
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Update Patient fields
	if req.DateOfBirth != "" {
		if dob, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			patient.DateOfBirth = &dob
		}
	}
	if req.Gender != "" {
		patient.Gender = req.Gender
	}
	if req.BloodGroup != "" {
		patient.BloodGroup = entity.BloodGroup(req.BloodGroup)
	}
	if req.Genotype != "" {
		patient.Genotype = req.Genotype
	}
	if req.Allergies != "" {
		patient.Allergies = req.Allergies
	}
	if req.ChronicConditions != "" {
		patient.ChronicConditions = req.ChronicConditions
	}
	if req.Address != "" {
		patient.Address = req.Address
	}
	if req.State != "" {
		patient.State = req.State
	}
	if req.LGA != "" {
		patient.LGA = req.LGA
	}
	if req.NHISNumber != "" {
		patient.NHISNumber = req.NHISNumber
	}

	if err := s.patientRepo.Update(ctx, patient); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return patient, nil
}

func (s *patientService) GetDashboard(ctx context.Context, userID uint) (*dto.PatientDashboardResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		wallet = &entity.Wallet{Balance: 0}
	}

	unreadCount, _ := s.notifRepo.CountUnread(ctx, userID)

	return &dto.PatientDashboardResponse{
		Patient: buildPatientProfileResponse(patient),
		WalletBalance:       wallet.Balance,
		UnreadNotifications: unreadCount,
		HealthScore:         patient.HealthScore,
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildPatientProfileResponse(p *entity.Patient) dto.PatientProfileResponse {
	return dto.PatientProfileResponse{
		ID: p.ID,
		User: dto.AuthUserResponse{
			ID:        p.User.ID,
			UUID:      p.User.UUID,
			FirstName: p.User.FirstName,
			LastName:  p.User.LastName,
			Email:     p.User.Email,
			Phone:     p.User.Phone,
			Role:      p.User.Role,
			IsVerified: p.User.IsVerified,
			ProfilePhotoURL: p.User.ProfilePhotoURL,
		},
		DateOfBirth:       p.DateOfBirth,
		Gender:            p.Gender,
		BloodGroup:        p.BloodGroup,
		Genotype:          p.Genotype,
		Allergies:         p.Allergies,
		ChronicConditions: p.ChronicConditions,
		Address:           p.Address,
		State:             p.State,
		LGA:               p.LGA,
		NHISNumber:        p.NHISNumber,
		HealthScore:       p.HealthScore,
	}
}
