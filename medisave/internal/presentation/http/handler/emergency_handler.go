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

type EmergencyHandler struct {
	emergencyService service.EmergencyService
}

func NewEmergencyHandler(emergencyService service.EmergencyService) *EmergencyHandler {
	return &EmergencyHandler{emergencyService: emergencyService}
}

// POST /api/v1/emergency/sos
func (h *EmergencyHandler) SOS(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.SOSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	emergency, err := h.emergencyService.TriggerSOS(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "sos triggered", emergency)
}

// PATCH /api/v1/emergency/:id/resolve
func (h *EmergencyHandler) Resolve(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid emergency id", nil)
		return
	}

	var req dto.ResolveEmergencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	if err := h.emergencyService.Resolve(c.Request.Context(), claims.UserID, id, &req); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "emergency resolved", nil)
}

// GET /api/v1/emergency/history
func (h *EmergencyHandler) GetHistory(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	history, total, err := h.emergencyService.GetHistory(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "emergency history", history, pagination.NewMeta(p, total))
}

// GET /api/v1/emergency/contacts
func (h *EmergencyHandler) GetContacts(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	contacts, err := h.emergencyService.GetContacts(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "emergency contacts", contacts)
}

// POST /api/v1/emergency/contacts
func (h *EmergencyHandler) AddContact(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.EmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	contact, err := h.emergencyService.AddContact(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "contact added", contact)
}

// PUT /api/v1/emergency/contacts/:id
func (h *EmergencyHandler) UpdateContact(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid contact id", nil)
		return
	}

	var req dto.EmergencyContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	contact, err := h.emergencyService.UpdateContact(c.Request.Context(), claims.UserID, id, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "contact updated", contact)
}

// DELETE /api/v1/emergency/contacts/:id
func (h *EmergencyHandler) DeleteContact(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid contact id", nil)
		return
	}

	if err := h.emergencyService.DeleteContact(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "contact deleted", nil)
}

// PATCH /api/v1/emergency/contacts/:id/primary
func (h *EmergencyHandler) SetPrimaryContact(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid contact id", nil)
		return
	}

	if err := h.emergencyService.SetPrimaryContact(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "primary contact updated", nil)
}
