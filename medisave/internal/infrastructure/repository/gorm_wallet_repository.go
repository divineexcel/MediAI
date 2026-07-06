package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"go.uber.org/zap"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/logger"
	"github.com/medisave/app/pkg/pagination"
)

type GORMWalletRepository struct {
	db *gorm.DB
}

func NewGORMWalletRepository(db *gorm.DB) domainrepo.WalletRepository {
	return &GORMWalletRepository{db: db}
}

func (r *GORMWalletRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMWalletRepository) Create(ctx context.Context, wallet *entity.Wallet) error {
	err := r.dbc(ctx).Create(wallet).Error
	if err != nil {
		logger.Error("GORMWalletRepository.Create failed", zap.Error(err), zap.Uint("user_id", wallet.UserID))
	}
	return err
}

func (r *GORMWalletRepository) FindByUserID(ctx context.Context, userID uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.dbc(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrWalletNotFound
	}
	return &wallet, err
}

func (r *GORMWalletRepository) FindByID(ctx context.Context, id uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.dbc(ctx).First(&wallet, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrWalletNotFound
	}
	return &wallet, err
}

func (r *GORMWalletRepository) UpdateBalance(ctx context.Context, walletID uint, amount float64) error {
	err := r.dbc(ctx).
		Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
	if err != nil {
		logger.Error("GORMWalletRepository.UpdateBalance failed", zap.Error(err), zap.Uint("wallet_id", walletID), zap.Float64("amount", amount))
	}
	return err
}

func (r *GORMWalletRepository) UpdateEscrow(ctx context.Context, walletID uint, amount float64) error {
	err := r.dbc(ctx).
		Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		UpdateColumn("escrow", gorm.Expr("escrow + ?", amount)).Error
	if err != nil {
		logger.Error("GORMWalletRepository.UpdateEscrow failed", zap.Error(err), zap.Uint("wallet_id", walletID), zap.Float64("amount", amount))
	}
	return err
}

// ─── Transaction Repository ───────────────────────────────────────────────────

type GORMTransactionRepository struct {
	db *gorm.DB
}

func NewGORMTransactionRepository(db *gorm.DB) domainrepo.TransactionRepository {
	return &GORMTransactionRepository{db: db}
}

func (r *GORMTransactionRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMTransactionRepository) Create(ctx context.Context, tx *entity.Transaction) error {
	err := r.dbc(ctx).Create(tx).Error
	if err != nil {
		logger.Error("GORMTransactionRepository.Create failed", zap.Error(err), zap.String("reference", tx.Reference))
	}
	return err
}

func (r *GORMTransactionRepository) FindByID(ctx context.Context, id uint) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.dbc(ctx).First(&tx, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &tx, err
}

func (r *GORMTransactionRepository) FindByReference(ctx context.Context, ref string) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.dbc(ctx).Where("reference = ?", ref).First(&tx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &tx, err
}

func (r *GORMTransactionRepository) ListByWalletID(ctx context.Context, walletID uint, p pagination.Params) ([]*entity.Transaction, int64, error) {
	var txs []*entity.Transaction
	var total int64

	q := r.dbc(ctx).Model(&entity.Transaction{}).Where("wallet_id = ?", walletID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&txs).Error
	return txs, total, err
}

func (r *GORMTransactionRepository) Update(ctx context.Context, tx *entity.Transaction) error {
	err := r.dbc(ctx).Save(tx).Error
	if err != nil {
		logger.Error("GORMTransactionRepository.Update failed", zap.Error(err), zap.Uint("tx_id", tx.ID))
	}
	return err
}

func (r *GORMTransactionRepository) UpdateStatus(ctx context.Context, txID uint, status entity.TransactionStatus) error {
	err := r.dbc(ctx).
		Model(&entity.Transaction{}).
		Where("id = ?", txID).
		Update("status", status).Error
	if err != nil {
		logger.Error("GORMTransactionRepository.UpdateStatus failed", zap.Error(err), zap.Uint("tx_id", txID), zap.String("status", string(status)))
	}
	return err
}

func (r *GORMTransactionRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.dbc(ctx).Model(&entity.Transaction{}).Count(&count).Error
	return count, err
}

func (r *GORMTransactionRepository) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int64, error) {
	var list []*entity.Transaction
	var total int64
	q := r.dbc(ctx).Model(&entity.Transaction{})
	q.Count(&total)
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMTransactionRepository) TotalVolume(ctx context.Context) (float64, error) {
	var total float64
	err := r.dbc(ctx).
		Model(&entity.Transaction{}).
		Where("status = ? AND type IN ?", entity.TxStatusSuccess, []string{string(entity.TxTypeDeposit), string(entity.TxTypePayment)}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
