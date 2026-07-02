package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/pkg/logger"
)

type USSDHandler struct {
	svc service.USSDService
}

func NewUSSDHandler(svc service.USSDService) *USSDHandler {
	return &USSDHandler{svc: svc}
}

// POST /api/v1/ussd/session — Africa's Talking callback (no auth).
// Response format: "CON <text>" to continue, "END <text>" to terminate.
func (h *USSDHandler) Session(c *gin.Context) {
	var req dto.USSDRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(200, "END Service temporarily unavailable.")
		return
	}
	logger.Info("USSD session",
		zap.String("session", req.SessionID),
		zap.String("phone", req.PhoneNumber),
		zap.String("text", req.Text),
	)
	resp, _ := h.svc.Handle(c.Request.Context(), &req)
	c.String(200, resp)
}

// GET /api/v1/ussd/test — development sandbox; drives the menu via query params.
func (h *USSDHandler) Test(c *gin.Context) {
	req := &dto.USSDRequest{
		SessionID:   c.DefaultQuery("session", "test-session-001"),
		ServiceCode: "*384*123#",
		PhoneNumber: c.DefaultQuery("phone", "+2348000000000"),
		Text:        c.DefaultQuery("text", ""),
	}
	resp, _ := h.svc.Handle(c.Request.Context(), req)
	c.JSON(200, gin.H{
		"service_code": req.ServiceCode,
		"phone":        req.PhoneNumber,
		"text":         req.Text,
		"response":     resp,
	})
}
