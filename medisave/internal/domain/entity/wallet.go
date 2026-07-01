package entity

import "time"

type WalletOwnerType string

const (
	WalletOwnerPatient WalletOwnerType = "patient"
	WalletOwnerDoctor  WalletOwnerType = "doctor"
)

type Wallet struct {
	ID        uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint            `gorm:"uniqueIndex;not null" json:"user_id"`
	OwnerType WalletOwnerType `gorm:"not null" json:"owner_type"`
	Balance   float64         `gorm:"default:0;not null" json:"balance"`
	Escrow    float64         `gorm:"default:0;not null" json:"escrow"`
	Currency  string          `gorm:"default:'NGN';not null" json:"currency"`
	IsActive  bool            `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type TransactionType string
type TransactionStatus string

const (
	TxTypeDeposit              TransactionType = "deposit"
	TxTypeWithdrawal           TransactionType = "withdrawal"
	TxTypePayment              TransactionType = "payment"
	TxTypeRefund               TransactionType = "refund"
	TxTypeConsultationCredit   TransactionType = "consultation_credit"
	TxTypeSavings              TransactionType = "savings"

	TxStatusPending   TransactionStatus = "pending"
	TxStatusSuccess   TransactionStatus = "success"
	TxStatusFailed    TransactionStatus = "failed"
	TxStatusReversed  TransactionStatus = "reversed"
)

type Transaction struct {
	ID              uint              `gorm:"primaryKey;autoIncrement" json:"id"`
	Reference       string            `gorm:"uniqueIndex;not null" json:"reference"`
	WalletID        uint              `gorm:"not null" json:"wallet_id"`
	Wallet          Wallet            `gorm:"foreignKey:WalletID" json:"wallet"`
	Type            TransactionType   `gorm:"not null" json:"type"`
	Amount          float64           `gorm:"not null" json:"amount"`
	BalanceBefore   float64           `gorm:"not null" json:"balance_before"`
	BalanceAfter    float64           `gorm:"not null" json:"balance_after"`
	Status          TransactionStatus `gorm:"default:'pending'" json:"status"`
	Description     string            `json:"description"`
	Metadata        string            `json:"metadata"`
	PaystackRef     string            `json:"paystack_ref"`
	RelatedEntityID uint              `json:"related_entity_id"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}
