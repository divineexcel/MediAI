package service

import (
	"context"
	"strings"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/hash"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/utils"
)

// AuthService defines the authentication contract.
type AuthService interface {
	RegisterPatient(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	RegisterDoctor(ctx context.Context, req *dto.DoctorRegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPairResponse, error)
	GetCurrentUser(ctx context.Context, userID uint) (*entity.User, error)
	ChangePassword(ctx context.Context, userID uint, req *dto.ChangePasswordRequest) error
	UpdateFCMToken(ctx context.Context, userID uint, token string) error
}

type authService struct {
	userRepo    repository.UserRepository
	patientRepo repository.PatientRepository
	doctorRepo  repository.DoctorRepository
	walletRepo  repository.WalletRepository
	jwtManager  *pkgjwt.Manager
}

func NewAuthService(
	userRepo repository.UserRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	walletRepo repository.WalletRepository,
	jwtManager *pkgjwt.Manager,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		patientRepo: patientRepo,
		doctorRepo:  doctorRepo,
		walletRepo:  walletRepo,
		jwtManager:  jwtManager,
	}
}

// ─── REGISTER PATIENT ────────────────────────────────────────────────────────

func (s *authService) RegisterPatient(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	email := utils.NormEmail(req.Email)

	if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
		return nil, pkgerrors.ErrEmailExists
	}
	if exists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone); exists {
		return nil, pkgerrors.ErrPhoneExists
	}

	hashedPw, err := hash.Password(req.Password)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	user := &entity.User{
		UUID:         utils.NewUUID(),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Email:        email,
		Phone:        req.Phone,
		PasswordHash: hashedPw,
		Role:         entity.RolePatient,
		IsVerified:   true,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	patient := &entity.Patient{UserID: user.ID}
	if err := s.patientRepo.Create(ctx, patient); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	if err := s.walletRepo.Create(ctx, &entity.Wallet{
		UserID:    user.ID,
		OwnerType: entity.WalletOwnerPatient,
		Currency:  "NGN",
		IsActive:  true,
	}); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return s.buildAuthResponse(user)
}

// ─── REGISTER DOCTOR ─────────────────────────────────────────────────────────

func (s *authService) RegisterDoctor(ctx context.Context, req *dto.DoctorRegisterRequest) (*dto.AuthResponse, error) {
	email := utils.NormEmail(req.Email)

	if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
		return nil, pkgerrors.ErrEmailExists
	}
	if exists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone); exists {
		return nil, pkgerrors.ErrPhoneExists
	}

	// Check license number uniqueness
	if _, err := s.doctorRepo.FindByLicenseNumber(ctx, req.LicenseNumber); err == nil {
		return nil, pkgerrors.ErrLicenseExists
	}

	hashedPw, err := hash.Password(req.Password)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	user := &entity.User{
		UUID:         utils.NewUUID(),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Email:        email,
		Phone:        req.Phone,
		PasswordHash: hashedPw,
		Role:         entity.RoleDoctor,
		IsVerified:   true,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	doctor := &entity.Doctor{
		UserID:            user.ID,
		LicenseNumber:     req.LicenseNumber,
		Specialty:         req.Specialty,
		YearsOfExperience: req.YearsOfExperience,
		ConsultationFee:   req.ConsultationFee,
		Hospital:          req.Hospital,
		WorkIDURL:         req.WorkIDURL,
		MedicalLicenseURL: req.MedicalLicenseURL,
		Status:            entity.DoctorStatusPending,
		IsAvailable:       false,
	}
	if err := s.doctorRepo.Create(ctx, doctor); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	if err := s.walletRepo.Create(ctx, &entity.Wallet{
		UserID:    user.ID,
		OwnerType: entity.WalletOwnerDoctor,
		Currency:  "NGN",
		IsActive:  true,
	}); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return s.buildAuthResponse(user)
}

// ─── LOGIN ───────────────────────────────────────────────────────────────────

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, utils.NormEmail(req.Email))
	if err != nil {
		// Return same error whether email or password is wrong — prevents user enumeration
		return nil, pkgerrors.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, pkgerrors.ErrAccountInactive
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		return nil, pkgerrors.ErrInvalidCredentials
	}

	return s.buildAuthResponse(user)
}

// ─── REFRESH TOKEN ───────────────────────────────────────────────────────────

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenPairResponse, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, pkgerrors.ErrTokenInvalid
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, pkgerrors.ErrUserNotFound
	}
	if !user.IsActive {
		return nil, pkgerrors.ErrAccountInactive
	}

	tokens, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return &dto.TokenPairResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
	}, nil
}

// ─── GET CURRENT USER ────────────────────────────────────────────────────────

func (s *authService) GetCurrentUser(ctx context.Context, userID uint) (*entity.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// ─── CHANGE PASSWORD ─────────────────────────────────────────────────────────

func (s *authService) ChangePassword(ctx context.Context, userID uint, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if !hash.CheckPassword(req.OldPassword, user.PasswordHash) {
		return pkgerrors.ErrInvalidCredentials
	}

	newHash, err := hash.Password(req.NewPassword)
	if err != nil {
		return pkgerrors.ErrInternalServer
	}

	user.PasswordHash = newHash
	return s.userRepo.Update(ctx, user)
}

// ─── UPDATE FCM TOKEN ────────────────────────────────────────────────────────

func (s *authService) UpdateFCMToken(ctx context.Context, userID uint, token string) error {
	return s.userRepo.UpdateFCMToken(ctx, userID, token)
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func (s *authService) buildAuthResponse(user *entity.User) (*dto.AuthResponse, error) {
	tokens, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return &dto.AuthResponse{
		User: dto.AuthUserResponse{
			ID:              user.ID,
			UUID:            user.UUID,
			FirstName:       user.FirstName,
			LastName:        user.LastName,
			Email:           user.Email,
			Phone:           user.Phone,
			Role:            user.Role,
			IsVerified:      user.IsVerified,
			ProfilePhotoURL: user.ProfilePhotoURL,
		},
		Tokens: dto.TokenPairResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt,
		},
	}, nil
}
