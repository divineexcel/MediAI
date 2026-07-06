package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	"github.com/medisave/app/internal/infrastructure/paystack"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/logger"
	"github.com/medisave/app/pkg/pagination"
	"github.com/medisave/app/pkg/utils"
)

type WalletService interface {
	GetWallet(ctx context.Context, userID uint) (*entity.Wallet, error)
	GetTransactions(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Transaction, int64, error)
	GetTransaction(ctx context.Context, userID uint, txID uint) (*entity.Transaction, error)
	InitializeDeposit(ctx context.Context, userID uint, email string, req *dto.DepositInitRequest) (*dto.DepositInitResponse, error)
	VerifyDeposit(ctx context.Context, userID uint, reference string) (*entity.Transaction, error)
	ProcessWebhook(ctx context.Context, body []byte, signature string) error
	Withdraw(ctx context.Context, userID uint, req *dto.WithdrawRequest) (*entity.Transaction, error)
	CreateSavingsGoal(ctx context.Context, userID uint, req *dto.CreateSavingsGoalRequest) (*entity.HealthSavingsGoal, error)
	GetSavingsGoals(ctx context.Context, userID uint, p pagination.Params) ([]*entity.HealthSavingsGoal, int64, error)
	ContributeToGoal(ctx context.Context, userID uint, goalID uint, req *dto.ContributeToGoalRequest) (*entity.HealthSavingsGoal, error)
}

type walletService struct {
	walletRepo   repository.WalletRepository
	txRepo       repository.TransactionRepository
	savingsRepo  repository.SavingsRepository
	patientRepo  repository.PatientRepository
	paystack     *paystack.Client
}

func NewWalletService(
	walletRepo repository.WalletRepository,
	txRepo repository.TransactionRepository,
	savingsRepo repository.SavingsRepository,
	patientRepo repository.PatientRepository,
	paystackClient *paystack.Client,
) WalletService {
	return &walletService{
		walletRepo:  walletRepo,
		txRepo:      txRepo,
		savingsRepo: savingsRepo,
		patientRepo: patientRepo,
		paystack:    paystackClient,
	}
}

func (s *walletService) GetWallet(ctx context.Context, userID uint) (*entity.Wallet, error) {
	return s.walletRepo.FindByUserID(ctx, userID)
}

func (s *walletService) GetTransactions(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Transaction, int64, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return s.txRepo.ListByWalletID(ctx, wallet.ID, p)
}

func (s *walletService) GetTransaction(ctx context.Context, userID uint, txID uint) (*entity.Transaction, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return nil, err
	}
	// Ensure the transaction belongs to this wallet
	if tx.WalletID != wallet.ID {
		return nil, pkgerrors.ErrAccessDenied
	}
	return tx, nil
}

func (s *walletService) InitializeDeposit(ctx context.Context, userID uint, email string, req *dto.DepositInitRequest) (*dto.DepositInitResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("deposit initialization failed: wallet lookup error", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if !wallet.IsActive {
		logger.Warn("deposit initialization failed: wallet inactive", zap.Uint("user_id", userID), zap.Uint("wallet_id", wallet.ID))
		return nil, pkgerrors.ErrWalletInactive
	}

	reference := utils.GenerateReference("DEP")

	var authURL, accessCode string
	if s.paystack.IsDummy() {
		// Mock deposit initialization for development/sandbox
		authURL = "/patient/wallet"
		accessCode = "mock_access_code_" + reference
	} else {
		amountKobo := int64(req.Amount * 100) // convert NGN to kobo
		data, err := s.paystack.InitializeTransaction(&paystack.InitTransactionReq{
			Email:     email,
			Amount:    amountKobo,
			Reference: reference,
		})
		if err != nil {
			logger.Error("deposit initialization failed: gateway error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
			return nil, fmt.Errorf("payment gateway error: %w", err)
		}
		authURL = data.AuthorizationURL
		accessCode = data.AccessCode
		reference = data.Reference
	}

	// Record as a pending transaction
	tx := &entity.Transaction{
		Reference:     reference,
		WalletID:      wallet.ID,
		Type:          entity.TxTypeDeposit,
		Amount:        req.Amount,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance, // will be updated on verification
		Status:        entity.TxStatusPending,
		Description:   fmt.Sprintf("Wallet top-up via Paystack — ₦%.2f", req.Amount),
		PaystackRef:   reference,
	}
	if err := s.txRepo.Create(ctx, tx); err != nil {
		logger.Error("deposit initialization failed: transaction write error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	logger.Info("deposit initialized successfully",
		zap.Uint("wallet_id", wallet.ID),
		zap.String("ref", reference),
		zap.Float64("amount", req.Amount),
	)
	return &dto.DepositInitResponse{
		AuthorizationURL: authURL,
		AccessCode:       accessCode,
		Reference:        reference,
	}, nil
}

func (s *walletService) VerifyDeposit(ctx context.Context, userID uint, reference string) (*entity.Transaction, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("deposit verification failed: wallet lookup error", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}

	// Check the pending transaction exists and belongs to this wallet
	tx, err := s.txRepo.FindByReference(ctx, reference)
	if err != nil {
		logger.Warn("deposit verification failed: transaction ref not found", zap.Uint("wallet_id", wallet.ID), zap.String("ref", reference))
		return nil, pkgerrors.ErrNotFound
	}
	if tx.WalletID != wallet.ID {
		logger.Warn("deposit verification failed: wallet ownership mismatch", zap.Uint("wallet_id", wallet.ID), zap.Uint("tx_wallet_id", tx.WalletID))
		return nil, pkgerrors.ErrAccessDenied
	}
	if tx.Status == entity.TxStatusSuccess {
		return tx, nil // idempotent — already credited
	}

	var amountNGN float64
	if s.paystack.IsDummy() {
		// Mock successful verification in development
		amountNGN = tx.Amount
	} else {
		data, err := s.paystack.VerifyTransaction(reference)
		if err != nil {
			logger.Error("deposit verification failed: gateway verification error", zap.String("ref", reference), zap.Error(err))
			return nil, fmt.Errorf("payment gateway error: %w", err)
		}

		if data.Status != "success" {
			logger.Warn("deposit verification failed: gateway status not success", zap.String("ref", reference), zap.String("status", data.Status))
			_ = s.txRepo.UpdateStatus(ctx, tx.ID, entity.TxStatusFailed)
			return nil, fmt.Errorf("payment %s", data.Status)
		}
		amountNGN = float64(data.Amount) / 100
	}

	// Credit wallet
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, amountNGN); err != nil {
		logger.Error("deposit verification failed: wallet credit error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	// Re-fetch wallet to get current balance after update
	wallet, err = s.walletRepo.FindByID(ctx, wallet.ID)
	if err != nil {
		logger.Error("deposit verification failed: wallet reload error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	// Update transaction to success
	tx.Status = entity.TxStatusSuccess
	tx.Amount = amountNGN
	tx.BalanceAfter = wallet.Balance
	if err := s.txRepo.Update(ctx, tx); err != nil {
		logger.Error("deposit verification failed: transaction status update error", zap.String("ref", reference), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	logger.Info("deposit verified and credited successfully",
		zap.Uint("wallet_id", wallet.ID),
		zap.String("ref", reference),
		zap.Float64("amount", amountNGN),
		zap.Float64("new_balance", wallet.Balance),
	)
	return tx, nil
}

func (s *walletService) ProcessWebhook(ctx context.Context, body []byte, signature string) error {
	if !s.paystack.ValidateSignature(body, signature) {
		return fmt.Errorf("invalid webhook signature")
	}

	event, err := s.paystack.ParseWebhookEvent(body)
	if err != nil {
		return err
	}

	switch event.Event {
	case "charge.success":
		var data struct {
			Reference string `json:"reference"`
			Amount    int64  `json:"amount"` // kobo
		}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}

		tx, err := s.txRepo.FindByReference(ctx, data.Reference)
		if err != nil || tx.Status == entity.TxStatusSuccess {
			return nil // unknown or already processed
		}

		wallet, err := s.walletRepo.FindByID(ctx, tx.WalletID)
		if err != nil {
			return err
		}

		amountNGN := float64(data.Amount) / 100
		if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, amountNGN); err != nil {
			return err
		}

		wallet, err = s.walletRepo.FindByID(ctx, tx.WalletID)
		if err == nil {
			tx.Status = entity.TxStatusSuccess
			tx.Amount = amountNGN
			tx.BalanceAfter = wallet.Balance
			_ = s.txRepo.Update(ctx, tx)
		}

	case "transfer.success":
		var data struct{ Reference string `json:"reference"` }
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}
		tx, err := s.txRepo.FindByReference(ctx, data.Reference)
		if err == nil {
			_ = s.txRepo.UpdateStatus(ctx, tx.ID, entity.TxStatusSuccess)
		}

	case "transfer.failed", "transfer.reversed":
		var data struct{ Reference string `json:"reference"` }
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return err
		}
		tx, err := s.txRepo.FindByReference(ctx, data.Reference)
		if err == nil {
			// Refund the deducted amount
			_ = s.walletRepo.UpdateBalance(ctx, tx.WalletID, tx.Amount)
			_ = s.txRepo.UpdateStatus(ctx, tx.ID, entity.TxStatusReversed)
		}
	}

	return nil
}

func (s *walletService) Withdraw(ctx context.Context, userID uint, req *dto.WithdrawRequest) (*entity.Transaction, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.Error("withdrawal failed: wallet lookup error", zap.Uint("user_id", userID), zap.Error(err))
		return nil, err
	}
	if !wallet.IsActive {
		logger.Warn("withdrawal failed: wallet inactive", zap.Uint("wallet_id", wallet.ID))
		return nil, pkgerrors.ErrWalletInactive
	}
	if wallet.Balance < req.Amount {
		logger.Warn("withdrawal failed: insufficient balance",
			zap.Uint("wallet_id", wallet.ID),
			zap.Float64("balance", wallet.Balance),
			zap.Float64("requested", req.Amount),
		)
		return nil, pkgerrors.ErrInsufficientFunds
	}

	reference := utils.GenerateReference("WTH")

	narration := req.Narration
	if narration == "" {
		narration = "MediSave wallet withdrawal"
	}

	isDummy := s.paystack.IsDummy()

	if !isDummy {
		// Create transfer recipient on Paystack
		recipientCode, err := s.paystack.CreateTransferRecipient(req.AccountName, req.AccountNo, req.BankCode)
		if err != nil {
			logger.Error("withdrawal failed: recipient creation error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
			return nil, fmt.Errorf("payment gateway error: %w", err)
		}

		// Debit wallet optimistically (refunded on transfer.reversed webhook)
		if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, -req.Amount); err != nil {
			logger.Error("withdrawal failed: wallet debit error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
			return nil, pkgerrors.ErrInternalServer
		}

		tx := &entity.Transaction{
			Reference:     reference,
			WalletID:      wallet.ID,
			Type:          entity.TxTypeWithdrawal,
			Amount:        req.Amount,
			BalanceBefore: wallet.Balance,
			BalanceAfter:  wallet.Balance - req.Amount,
			Status:        entity.TxStatusPending,
			Description:   narration,
		}
		if err := s.txRepo.Create(ctx, tx); err != nil {
			logger.Error("withdrawal failed: transaction write error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
			return nil, pkgerrors.ErrInternalServer
		}

		// Initiate Paystack transfer (async — result comes via webhook)
		amountKobo := int64(req.Amount * 100)
		go func() {
			err := s.paystack.InitiateTransfer(amountKobo, recipientCode, reference, narration)
			if err != nil {
				logger.Error("withdrawal background transfer initiation failed", zap.String("ref", reference), zap.Error(err))
			}
		}()

		logger.Info("withdrawal initiated (pending callback)",
			zap.Uint("wallet_id", wallet.ID),
			zap.String("ref", reference),
			zap.Float64("amount", req.Amount),
		)
		return tx, nil
	}

	// ─── Dev/Dummy mode: debit wallet and mark success immediately ─
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, -req.Amount); err != nil {
		logger.Error("withdrawal failed: dummy mode wallet debit error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	tx := &entity.Transaction{
		Reference:     reference,
		WalletID:      wallet.ID,
		Type:          entity.TxTypeWithdrawal,
		Amount:        req.Amount,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance - req.Amount,
		Status:        entity.TxStatusSuccess,
		Description:   narration + " (demo)",
	}
	if err := s.txRepo.Create(ctx, tx); err != nil {
		logger.Error("withdrawal failed: dummy mode transaction write error", zap.Uint("wallet_id", wallet.ID), zap.Error(err))
		return nil, pkgerrors.ErrInternalServer
	}

	logger.Info("withdrawal processed immediately in dummy mode",
		zap.Uint("wallet_id", wallet.ID),
		zap.String("ref", reference),
		zap.Float64("amount", req.Amount),
	)
	return tx, nil
}

func (s *walletService) CreateSavingsGoal(ctx context.Context, userID uint, req *dto.CreateSavingsGoalRequest) (*entity.HealthSavingsGoal, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	targetDate, err := time.Parse("2006-01-02", req.TargetDate)
	if err != nil {
		return nil, fmt.Errorf("invalid target_date format, use YYYY-MM-DD")
	}
	if targetDate.Before(time.Now()) {
		return nil, fmt.Errorf("target_date must be in the future")
	}

	goal := &entity.HealthSavingsGoal{
		PatientID:      patient.ID,
		WalletID:       wallet.ID,
		Title:          req.Title,
		Description:    req.Description,
		TargetAmount:   req.TargetAmount,
		SavedAmount:    0,
		Frequency:      req.Frequency,
		AutoSaveAmount: req.AutoSaveAmount,
		Status:         entity.GoalStatusActive,
		TargetDate:     targetDate,
	}

	if err := s.savingsRepo.Create(ctx, goal); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return goal, nil
}

func (s *walletService) GetSavingsGoals(ctx context.Context, userID uint, p pagination.Params) ([]*entity.HealthSavingsGoal, int64, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, pkgerrors.ErrPatientNotFound
	}
	return s.savingsRepo.ListByPatient(ctx, patient.ID, p)
}

func (s *walletService) ContributeToGoal(ctx context.Context, userID uint, goalID uint, req *dto.ContributeToGoalRequest) (*entity.HealthSavingsGoal, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet.Balance < req.Amount {
		return nil, pkgerrors.ErrInsufficientFunds
	}

	goal, err := s.savingsRepo.FindByID(ctx, goalID)
	if err != nil {
		return nil, pkgerrors.ErrNotFound
	}
	if goal.WalletID != wallet.ID {
		return nil, pkgerrors.ErrAccessDenied
	}
	if goal.Status != entity.GoalStatusActive {
		return nil, fmt.Errorf("savings goal is not active")
	}

	// Deduct from wallet
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, -req.Amount); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Record savings transaction
	reference := utils.GenerateReference("SAV")
	tx := &entity.Transaction{
		Reference:       reference,
		WalletID:        wallet.ID,
		Type:            entity.TxTypeSavings,
		Amount:          req.Amount,
		BalanceBefore:   wallet.Balance,
		BalanceAfter:    wallet.Balance - req.Amount,
		Status:          entity.TxStatusSuccess,
		Description:     fmt.Sprintf("Contribution to savings goal: %s", goal.Title),
		RelatedEntityID: goal.ID,
	}
	_ = s.txRepo.Create(ctx, tx)

	// Credit the goal
	if err := s.savingsRepo.UpdateSavedAmount(ctx, goal.ID, req.Amount); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Mark as completed if target reached
	goal.SavedAmount += req.Amount
	if goal.SavedAmount >= goal.TargetAmount {
		_ = s.savingsRepo.UpdateStatus(ctx, goal.ID, entity.GoalStatusCompleted)
		goal.Status = entity.GoalStatusCompleted
	}

	return goal, nil
}
