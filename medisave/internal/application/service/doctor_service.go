package service

import (
	"context"
	"strings"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type DoctorService interface {
	GetProfile(ctx context.Context, userID uint) (*entity.Doctor, error)
	GetDoctorByID(ctx context.Context, doctorID uint) (*entity.Doctor, error)
	UpdateProfile(ctx context.Context, userID uint, req *dto.UpdateDoctorProfileRequest) (*entity.Doctor, error)
	GetDashboard(ctx context.Context, userID uint) (*dto.DoctorDashboardResponse, error)
	ToggleAvailability(ctx context.Context, userID uint, available bool) error
	GetTodayAppointments(ctx context.Context, userID uint) ([]*entity.Appointment, error)
	List(ctx context.Context, filter dto.DoctorListFilter, p pagination.Params) ([]*entity.Doctor, int64, error)
}

type doctorService struct {
	doctorRepo repository.DoctorRepository
	userRepo   repository.UserRepository
	walletRepo repository.WalletRepository
	notifRepo  repository.NotificationRepository
	apptRepo   repository.AppointmentRepository
}

func NewDoctorService(
	doctorRepo repository.DoctorRepository,
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	notifRepo repository.NotificationRepository,
	apptRepo repository.AppointmentRepository,
) DoctorService {
	return &doctorService{
		doctorRepo: doctorRepo,
		userRepo:   userRepo,
		walletRepo: walletRepo,
		notifRepo:  notifRepo,
		apptRepo:   apptRepo,
	}
}

func (s *doctorService) GetProfile(ctx context.Context, userID uint) (*entity.Doctor, error) {
	return s.doctorRepo.FindByUserID(ctx, userID)
}

func (s *doctorService) GetDoctorByID(ctx context.Context, doctorID uint) (*entity.Doctor, error) {
	return s.doctorRepo.FindByID(ctx, doctorID)
}

func (s *doctorService) UpdateProfile(ctx context.Context, userID uint, req *dto.UpdateDoctorProfileRequest) (*entity.Doctor, error) {
	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user := &doctor.User
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

	if req.SubSpecialty != "" {
		doctor.SubSpecialty = req.SubSpecialty
	}
	if req.Hospital != "" {
		doctor.Hospital = req.Hospital
	}
	if req.ConsultationFee > 0 {
		doctor.ConsultationFee = req.ConsultationFee
	}
	if req.Bio != "" {
		doctor.Bio = req.Bio
	}
	if req.Education != "" {
		doctor.Education = req.Education
	}
	if req.Certifications != "" {
		doctor.Certifications = req.Certifications
	}
	if req.Languages != "" {
		doctor.Languages = req.Languages
	}

	if err := s.doctorRepo.Update(ctx, doctor); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return doctor, nil
}

func (s *doctorService) GetDashboard(ctx context.Context, userID uint) (*dto.DoctorDashboardResponse, error) {
	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		wallet = &entity.Wallet{Balance: 0}
	}

	unreadCount, _ := s.notifRepo.CountUnread(ctx, userID)

	todayAppts, _ := s.apptRepo.ListTodayByDoctor(ctx, doctor.ID)
	pendingCount, _ := s.apptRepo.CountByDoctor(ctx, doctor.ID)

	return &dto.DoctorDashboardResponse{
		Doctor:              buildDoctorProfileResponse(doctor),
		WalletBalance:       wallet.Balance,
		TodayAppointments:   len(todayAppts),
		PendingAppointments: int(pendingCount),
		TotalEarnings:       wallet.Balance,
		UnreadNotifications: unreadCount,
		Rating:              doctor.Rating,
		TotalConsultations:  doctor.TotalConsultations,
	}, nil
}

func (s *doctorService) ToggleAvailability(ctx context.Context, userID uint, available bool) error {
	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.doctorRepo.SetAvailability(ctx, doctor.ID, available)
}

func (s *doctorService) GetTodayAppointments(ctx context.Context, userID uint) ([]*entity.Appointment, error) {
	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.apptRepo.ListTodayByDoctor(ctx, doctor.ID)
}

func (s *doctorService) List(ctx context.Context, filter dto.DoctorListFilter, p pagination.Params) ([]*entity.Doctor, int64, error) {
	if filter.Available || filter.Specialty != "" {
		return s.doctorRepo.ListAvailable(ctx, filter.Specialty, p)
	}
	return s.doctorRepo.ListVerified(ctx, p)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildDoctorProfileResponse(d *entity.Doctor) dto.DoctorProfileResponse {
	return dto.DoctorProfileResponse{
		ID: d.ID,
		User: dto.AuthUserResponse{
			ID:              d.User.ID,
			UUID:            d.User.UUID,
			FirstName:       d.User.FirstName,
			LastName:        d.User.LastName,
			Email:           d.User.Email,
			Phone:           d.User.Phone,
			Role:            d.User.Role,
			IsVerified:      d.User.IsVerified,
			ProfilePhotoURL: d.User.ProfilePhotoURL,
		},
		LicenseNumber:      d.LicenseNumber,
		Specialty:          d.Specialty,
		SubSpecialty:       d.SubSpecialty,
		YearsOfExperience:  d.YearsOfExperience,
		Hospital:           d.Hospital,
		ConsultationFee:    d.ConsultationFee,
		IsAvailable:        d.IsAvailable,
		Status:             d.Status,
		Bio:                d.Bio,
		Education:          d.Education,
		Certifications:     d.Certifications,
		Languages:          d.Languages,
		Rating:             d.Rating,
		TotalReviews:       d.TotalReviews,
		TotalConsultations: d.TotalConsultations,
	}
}
