package service

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/hash"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/logger"
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
	txer        repository.Transactor
}

func NewAuthService(
	userRepo repository.UserRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	walletRepo repository.WalletRepository,
	jwtManager *pkgjwt.Manager,
	txer repository.Transactor,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		patientRepo: patientRepo,
		doctorRepo:  doctorRepo,
		walletRepo:  walletRepo,
		jwtManager:  jwtManager,
		txer:        txer,
	}
}

// ─── REGISTER PATIENT ────────────────────────────────────────────────────────

func (s *authService) RegisterPatient(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	email := utils.NormEmail(req.Email)
	logger.Info("registering patient", zap.String("email", email))

	if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
		logger.Warn("patient registration failed: email already exists", zap.String("email", email))
		return nil, pkgerrors.ErrEmailExists
	}
	if exists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone); exists {
		logger.Warn("patient registration failed: phone already exists", zap.String("phone", req.Phone))
		return nil, pkgerrors.ErrPhoneExists
	}

	hashedPw, err := hash.Password(req.Password)
	if err != nil {
		logger.Error("failed to hash password for patient registration", zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	var user *entity.User
	err = s.txer.WithinTransaction(ctx, func(txCtx context.Context) error {
		user = &entity.User{
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
		if err := s.userRepo.Create(txCtx, user); err != nil {
			logger.Error("failed to create user during patient registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
		}

		patient := &entity.Patient{UserID: user.ID}
		if err := s.patientRepo.Create(txCtx, patient); err != nil {
			logger.Error("failed to create patient record during registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
		}

		if err := s.walletRepo.Create(txCtx, &entity.Wallet{
			UserID:    user.ID,
			OwnerType: entity.WalletOwnerPatient,
			Currency:  "NGN",
			IsActive:  true,
		}); err != nil {
			logger.Error("failed to create wallet during patient registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	logger.Info("patient registered successfully", zap.String("email", email), zap.Uint("user_id", user.ID))
	return s.buildAuthResponse(user)
}

// ─── REGISTER DOCTOR ─────────────────────────────────────────────────────────

func (s *authService) RegisterDoctor(ctx context.Context, req *dto.DoctorRegisterRequest) (*dto.AuthResponse, error) {
	email := utils.NormEmail(req.Email)
	logger.Info("registering doctor", zap.String("email", email), zap.String("license", req.LicenseNumber))

	if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
		logger.Warn("doctor registration failed: email already exists", zap.String("email", email))
		return nil, pkgerrors.ErrEmailExists
	}
	if exists, _ := s.userRepo.ExistsByPhone(ctx, req.Phone); exists {
		logger.Warn("doctor registration failed: phone already exists", zap.String("phone", req.Phone))
		return nil, pkgerrors.ErrPhoneExists
	}

	// Check license number uniqueness
	if _, err := s.doctorRepo.FindByLicenseNumber(ctx, req.LicenseNumber); err == nil {
		logger.Warn("doctor registration failed: license already exists", zap.String("license", req.LicenseNumber))
		return nil, pkgerrors.ErrLicenseExists
	}

	hashedPw, err := hash.Password(req.Password)
	if err != nil {
		logger.Error("failed to hash password for doctor registration", zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	var user *entity.User
	err = s.txer.WithinTransaction(ctx, func(txCtx context.Context) error {
		user = &entity.User{
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
		if err := s.userRepo.Create(txCtx, user); err != nil {
			logger.Error("failed to create user during doctor registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
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
		if err := s.doctorRepo.Create(txCtx, doctor); err != nil {
			logger.Error("failed to create doctor record during registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
		}

		if err := s.walletRepo.Create(txCtx, &entity.Wallet{
			UserID:    user.ID,
			OwnerType: entity.WalletOwnerDoctor,
			Currency:  "NGN",
			IsActive:  true,
		}); err != nil {
			logger.Error("failed to create wallet during doctor registration", zap.String("email", email), zap.Error(err))
			return pkgerrors.ErrInternalServer
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	logger.Info("doctor registered successfully", zap.String("email", email), zap.Uint("user_id", user.ID))
	return s.buildAuthResponse(user)
}

// ─── LOGIN ───────────────────────────────────────────────────────────────────

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.AuthResponse, error) {
	email := utils.NormEmail(req.Email)
	logger.Info("attempting login", zap.String("email", email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Warn("login failed: user not found", zap.String("email", email))
		return nil, pkgerrors.ErrInvalidCredentials
	}

	if !user.IsActive {
		logger.Warn("login failed: account inactive", zap.String("email", email), zap.Uint("user_id", user.ID))
		return nil, pkgerrors.ErrAccountInactive
	}

	if !hash.CheckPassword(req.Password, user.PasswordHash) {
		logger.Warn("login failed: incorrect password", zap.String("email", email), zap.Uint("user_id", user.ID))
		return nil, pkgerrors.ErrInvalidCredentials
	}

	logger.Info("login successful", zap.String("email", email), zap.Uint("user_id", user.ID))
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
