package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type AIHandler struct {
	aiService service.AIService
}

func NewAIHandler(aiService service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// POST /api/v1/ai/chat
func (h *AIHandler) Chat(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	msg, err := h.aiService.Chat(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "message sent", msg)
}

// GET /api/v1/ai/conversations
func (h *AIHandler) GetConversations(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	convs, err := h.aiService.GetConversations(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "conversations", convs)
}

// GET /api/v1/ai/conversations/:id/messages
func (h *AIHandler) GetMessages(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid conversation id", nil)
		return
	}

	msgs, err := h.aiService.GetMessages(c.Request.Context(), claims.UserID, id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "messages", msgs)
}

// DELETE /api/v1/ai/conversations/:id
func (h *AIHandler) ClearConversation(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid conversation id", nil)
		return
	}

	if err := h.aiService.ClearConversation(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "conversation cleared", nil)
}
