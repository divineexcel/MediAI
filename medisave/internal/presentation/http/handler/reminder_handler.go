package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/pagination"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type ReminderHandler struct {
	reminderService service.ReminderService
}

func NewReminderHandler(reminderService service.ReminderService) *ReminderHandler {
	return &ReminderHandler{reminderService: reminderService}
}

// GET /api/v1/reminders
func (h *ReminderHandler) List(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	reminders, total, err := h.reminderService.List(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "reminders", reminders, pagination.NewMeta(p, total))
}

// POST /api/v1/reminders
func (h *ReminderHandler) Create(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.CreateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	reminder, err := h.reminderService.Create(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "reminder created", reminder)
}

// GET /api/v1/reminders/:id
func (h *ReminderHandler) GetByID(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid reminder id", nil)
		return
	}

	reminder, err := h.reminderService.GetByID(c.Request.Context(), claims.UserID, id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "reminder", reminder)
}

// PUT /api/v1/reminders/:id
func (h *ReminderHandler) Update(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid reminder id", nil)
		return
	}

	var req dto.CreateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	reminder, err := h.reminderService.Update(c.Request.Context(), claims.UserID, id, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "reminder updated", reminder)
}

// DELETE /api/v1/reminders/:id
func (h *ReminderHandler) Deactivate(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid reminder id", nil)
		return
	}

	if err := h.reminderService.Deactivate(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "reminder deactivated", nil)
}

// POST /api/v1/reminders/logs/:id/action
func (h *ReminderHandler) LogAction(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid reminder id", nil)
		return
	}

	var req dto.ReminderLogActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	if err := h.reminderService.LogAction(c.Request.Context(), claims.UserID, id, &req); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "action logged", nil)
}

// GET /api/v1/reminders/analytics
func (h *ReminderHandler) Analytics(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	analytics, err := h.reminderService.GetAnalytics(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "reminder analytics", analytics)
}
