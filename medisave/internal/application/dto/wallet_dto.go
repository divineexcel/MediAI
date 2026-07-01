package dto

import "github.com/medisave/app/internal/domain/entity"

type DepositInitRequest struct {
	Amount float64 `json:"amount" validate:"required,min=100"`
}

type DepositVerifyRequest struct {
	Reference string `json:"reference" validate:"required"`
}

type WithdrawRequest struct {
	Amount      float64 `json:"amount"       validate:"required,min=100"`
	BankCode    string  `json:"bank_code"    validate:"required"`
	AccountNo   string  `json:"account_no"   validate:"required,len=10"`
	AccountName string  `json:"account_name" validate:"required"`
	Narration   string  `json:"narration"    validate:"omitempty,max=100"`
}

type SavingsContributeRequest struct {
	GoalID uint    `json:"goal_id" validate:"required"`
	Amount float64 `json:"amount"  validate:"required,min=100"`
}

// ─── RESPONSES ───────────────────────────────────────────────────────────────

type WalletResponse struct {
	ID        uint    `json:"id"`
	Balance   float64 `json:"balance"`
	Escrow    float64 `json:"escrow"`
	Currency  string  `json:"currency"`
	IsActive  bool    `json:"is_active"`
}

type TransactionResponse struct {
	ID            uint                      `json:"id"`
	Reference     string                    `json:"reference"`
	Type          entity.TransactionType    `json:"type"`
	Amount        float64                   `json:"amount"`
	BalanceBefore float64                   `json:"balance_before"`
	BalanceAfter  float64                   `json:"balance_after"`
	Status        entity.TransactionStatus  `json:"status"`
	Description   string                    `json:"description"`
	CreatedAt     string                    `json:"created_at"`
}

type DepositInitResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}
