package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type ConsultationHandler struct {
	apptService service.AppointmentService
}

func NewConsultationHandler(apptService service.AppointmentService) *ConsultationHandler {
	return &ConsultationHandler{apptService: apptService}
}

// GET /api/v1/consultations/:appointment_id
func (h *ConsultationHandler) Get(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	consult, prescriptions, err := h.apptService.GetConsultation(c.Request.Context(), claims.UserID, apptID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "consultation", gin.H{
		"consultation":  consult,
		"prescriptions": prescriptions,
	})
}

// GET /api/v1/consultations/:appointment_id/messages
func (h *ConsultationHandler) GetMessages(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	msgs, err := h.apptService.GetMessages(c.Request.Context(), claims.UserID, apptID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "messages", msgs)
}

// POST /api/v1/consultations/:appointment_id/messages
func (h *ConsultationHandler) SendMessage(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	msg, err := h.apptService.SendMessage(c.Request.Context(), claims.UserID, claims.Role, apptID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "message sent", msg)
}

// PUT /api/v1/consultations/:appointment_id/notes  (doctor only)
func (h *ConsultationHandler) SaveNotes(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	var req dto.ConsultationNotesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	consult, err := h.apptService.SaveNotes(c.Request.Context(), claims.UserID, apptID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "notes saved", consult)
}

// POST /api/v1/consultations/:appointment_id/prescriptions  (doctor only)
func (h *ConsultationHandler) AddPrescription(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	var req dto.AddPrescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	prescription, err := h.apptService.AddPrescription(c.Request.Context(), claims.UserID, apptID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "prescription added", prescription)
}

// GET /api/v1/consultations/:appointment_id/prescriptions
func (h *ConsultationHandler) GetPrescriptions(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	apptID, err := parseID(c, "appointment_id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	prescriptions, err := h.apptService.GetPrescriptions(c.Request.Context(), claims.UserID, apptID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "prescriptions", prescriptions)
}
