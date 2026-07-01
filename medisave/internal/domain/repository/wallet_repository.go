package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/pkg/pagination"
)

type WalletRepository interface {
	Create(ctx context.Context, wallet *entity.Wallet) error
	FindByUserID(ctx context.Context, userID uint) (*entity.Wallet, error)
	FindByID(ctx context.Context, id uint) (*entity.Wallet, error)
	UpdateBalance(ctx context.Context, walletID uint, amount float64) error
	UpdateEscrow(ctx context.Context, walletID uint, amount float64) error
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *entity.Transaction) error
	FindByID(ctx context.Context, id uint) (*entity.Transaction, error)
	FindByReference(ctx context.Context, ref string) (*entity.Transaction, error)
	ListByWalletID(ctx context.Context, walletID uint, p pagination.Params) ([]*entity.Transaction, int64, error)
	Update(ctx context.Context, tx *entity.Transaction) error
	UpdateStatus(ctx context.Context, txID uint, status entity.TransactionStatus) error
	CountAll(ctx context.Context) (int64, error)
	ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int64, error)
	TotalVolume(ctx context.Context) (float64, error)
}
