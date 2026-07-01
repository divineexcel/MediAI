package handler

import (
	"io"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/pagination"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type WalletHandler struct {
	walletService service.WalletService
}

func NewWalletHandler(walletService service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

// GET /api/v1/wallet
func (h *WalletHandler) GetWallet(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	wallet, err := h.walletService.GetWallet(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "wallet loaded", dto.WalletResponse{
		ID:       wallet.ID,
		Balance:  wallet.Balance,
		Escrow:   wallet.Escrow,
		Currency: wallet.Currency,
		IsActive: wallet.IsActive,
	})
}

// POST /api/v1/wallet/deposit/initialize
func (h *WalletHandler) InitializeDeposit(c *gin.Context) {
	var req dto.DepositInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	result, err := h.walletService.InitializeDeposit(c.Request.Context(), claims.UserID, claims.Email, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "deposit initialized", result)
}

// POST /api/v1/wallet/deposit/verify
func (h *WalletHandler) VerifyDeposit(c *gin.Context) {
	var req dto.DepositVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	tx, err := h.walletService.VerifyDeposit(c.Request.Context(), claims.UserID, req.Reference)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "deposit verified and wallet credited", buildTxResponse(tx))
}

// POST /api/v1/wallet/withdraw
func (h *WalletHandler) Withdraw(c *gin.Context) {
	var req dto.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	tx, err := h.walletService.Withdraw(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "withdrawal initiated", buildTxResponse(tx))
}

// GET /api/v1/wallet/transactions
func (h *WalletHandler) GetTransactions(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	txs, total, err := h.walletService.GetTransactions(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	var out []dto.TransactionResponse
	for _, tx := range txs {
		out = append(out, buildTxResponse(tx))
	}

	response.Paginated(c, "transactions loaded", out, pagination.NewMeta(p, total))
}

// GET /api/v1/wallet/transactions/:id
func (h *WalletHandler) GetTransaction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid transaction id", nil)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	tx, err := h.walletService.GetTransaction(c.Request.Context(), claims.UserID, uint(id))
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "transaction loaded", buildTxResponse(tx))
}

// POST /api/v1/wallet/savings
func (h *WalletHandler) CreateSavingsGoal(c *gin.Context) {
	var req dto.CreateSavingsGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	goal, err := h.walletService.CreateSavingsGoal(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.Created(c, "savings goal created", buildGoalResponse(goal))
}

// GET /api/v1/wallet/savings
func (h *WalletHandler) GetSavingsGoals(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	goals, total, err := h.walletService.GetSavingsGoals(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	var out []dto.SavingsGoalResponse
	for _, g := range goals {
		out = append(out, buildGoalResponse(g))
	}

	response.Paginated(c, "savings goals loaded", out, pagination.NewMeta(p, total))
}

// POST /api/v1/wallet/savings/:id/contribute
func (h *WalletHandler) ContributeToGoal(c *gin.Context) {
	goalID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid goal id", nil)
		return
	}

	var req dto.ContributeToGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	goal, err := h.walletService.ContributeToGoal(c.Request.Context(), claims.UserID, uint(goalID), &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "contribution added", buildGoalResponse(goal))
}

// POST /api/v1/wallet/webhook  (Paystack — no auth middleware)
func (h *WalletHandler) PaystackWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(200, gin.H{"status": "ok"}) // always 200 to Paystack
		return
	}

	signature := c.GetHeader("x-paystack-signature")
	_ = h.walletService.ProcessWebhook(c.Request.Context(), body, signature)

	c.JSON(200, gin.H{"status": "ok"})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildTxResponse(tx *entity.Transaction) dto.TransactionResponse {
	return dto.TransactionResponse{
		ID:            tx.ID,
		Reference:     tx.Reference,
		Type:          tx.Type,
		Amount:        tx.Amount,
		BalanceBefore: tx.BalanceBefore,
		BalanceAfter:  tx.BalanceAfter,
		Status:        tx.Status,
		Description:   tx.Description,
		CreatedAt:     tx.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func buildGoalResponse(g *entity.HealthSavingsGoal) dto.SavingsGoalResponse {
	var pct float64
	if g.TargetAmount > 0 {
		pct = (g.SavedAmount / g.TargetAmount) * 100
	}
	days := int(time.Until(g.TargetDate).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return dto.SavingsGoalResponse{
		ID:             g.ID,
		Title:          g.Title,
		Description:    g.Description,
		TargetAmount:   g.TargetAmount,
		SavedAmount:    g.SavedAmount,
		ProgressPct:    pct,
		Frequency:      g.Frequency,
		AutoSaveAmount: g.AutoSaveAmount,
		Status:         string(g.Status),
		TargetDate:     g.TargetDate,
		DaysRemaining:  days,
	}
}
