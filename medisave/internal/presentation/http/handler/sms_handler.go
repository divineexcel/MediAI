package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	smsclient "github.com/medisave/app/internal/infrastructure/external/sms"
	"github.com/medisave/app/pkg/logger"
	"github.com/medisave/app/pkg/response"
	"go.uber.org/zap"
)

type SMSHandler struct {
	smsClient *smsclient.Client
}

func NewSMSHandler(smsClient *smsclient.Client) *SMSHandler {
	return &SMSHandler{smsClient: smsClient}
}

type sendSMSRequest struct {
	To      string `json:"to"      binding:"required"`
	Message string `json:"message" binding:"required,min=1,max=480"`
}

type bulkSMSRequest struct {
	Recipients []string `json:"recipients" binding:"required,min=1"`
	Message    string   `json:"message"    binding:"required,min=1,max=480"`
}

// POST /api/v1/sms/webhook  (Africa's Talking / gateway inbound SMS callback — no auth)
func (h *SMSHandler) Webhook(c *gin.Context) {
	from := c.PostForm("from")
	text := c.PostForm("text")
	date := c.PostForm("date")

	logger.Info("inbound SMS",
		zap.String("from", from),
		zap.String("text", text),
		zap.String("date", date),
	)

	// Acknowledge receipt — gateway expects 200
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /api/v1/sms/send  (admin only)
func (h *SMSHandler) Send(c *gin.Context) {
	// Check for bulk vs single
	var bulk bulkSMSRequest
	if err := c.ShouldBindJSON(&bulk); err == nil && len(bulk.Recipients) > 0 {
		if err := h.smsClient.SendBulk(bulk.Recipients, bulk.Message); err != nil {
			response.InternalError(c, "failed to send bulk SMS")
			return
		}
		response.OK(c, "bulk SMS queued", gin.H{"recipients": len(bulk.Recipients)})
		return
	}

	var req sendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	if err := h.smsClient.Send(req.To, req.Message); err != nil {
		response.InternalError(c, "failed to send SMS")
		return
	}
	response.OK(c, "sms sent", gin.H{"to": req.To})
}
