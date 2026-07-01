package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type GORMWalletRepository struct {
	db *gorm.DB
}

func NewGORMWalletRepository(db *gorm.DB) domainrepo.WalletRepository {
	return &GORMWalletRepository{db: db}
}

func (r *GORMWalletRepository) Create(ctx context.Context, wallet *entity.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *GORMWalletRepository) FindByUserID(ctx context.Context, userID uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrWalletNotFound
	}
	return &wallet, err
}

func (r *GORMWalletRepository) FindByID(ctx context.Context, id uint) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).First(&wallet, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrWalletNotFound
	}
	return &wallet, err
}

func (r *GORMWalletRepository) UpdateBalance(ctx context.Context, walletID uint, amount float64) error {
	return r.db.WithContext(ctx).
		Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *GORMWalletRepository) UpdateEscrow(ctx context.Context, walletID uint, amount float64) error {
	return r.db.WithContext(ctx).
		Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		UpdateColumn("escrow", gorm.Expr("escrow + ?", amount)).Error
}

// ─── Transaction Repository ───────────────────────────────────────────────────

type GORMTransactionRepository struct {
	db *gorm.DB
}

func NewGORMTransactionRepository(db *gorm.DB) domainrepo.TransactionRepository {
	return &GORMTransactionRepository{db: db}
}

func (r *GORMTransactionRepository) Create(ctx context.Context, tx *entity.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *GORMTransactionRepository) FindByID(ctx context.Context, id uint) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.db.WithContext(ctx).First(&tx, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &tx, err
}

func (r *GORMTransactionRepository) FindByReference(ctx context.Context, ref string) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.db.WithContext(ctx).Where("reference = ?", ref).First(&tx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrNotFound
	}
	return &tx, err
}

func (r *GORMTransactionRepository) ListByWalletID(ctx context.Context, walletID uint, p pagination.Params) ([]*entity.Transaction, int64, error) {
	var txs []*entity.Transaction
	var total int64

	q := r.db.WithContext(ctx).Model(&entity.Transaction{}).Where("wallet_id = ?", walletID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&txs).Error
	return txs, total, err
}

func (r *GORMTransactionRepository) UpdateStatus(ctx context.Context, txID uint, status entity.TransactionStatus) error {
	return r.db.WithContext(ctx).
		Model(&entity.Transaction{}).
		Where("id = ?", txID).
		Update("status", status).Error
}

func (r *GORMTransactionRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Transaction{}).Count(&count).Error
	return count, err
}

func (r *GORMTransactionRepository) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int64, error) {
	var list []*entity.Transaction
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.Transaction{})
	q.Count(&total)
	err := q.Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMTransactionRepository) TotalVolume(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Model(&entity.Transaction{}).
		Where("status = 'completed' AND type IN ('deposit','appointment_payment')").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return total, err
}
