package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/infrastructure/rtc"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/pagination"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type AppointmentHandler struct {
	apptService service.AppointmentService
	roomSvc     service.ConsultationRoomService
	hub         *rtc.Hub
}

func NewAppointmentHandler(apptService service.AppointmentService, roomSvc service.ConsultationRoomService, hub *rtc.Hub) *AppointmentHandler {
	return &AppointmentHandler{apptService: apptService, roomSvc: roomSvc, hub: hub}
}

// GET /api/v1/appointments
func (h *AppointmentHandler) List(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)
	status := c.Query("status")

	appts, total, err := h.apptService.List(c.Request.Context(), claims.UserID, claims.Role, status, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "appointments", appts, pagination.NewMeta(p, total))
}

// POST /api/v1/appointments  (patient only)
func (h *AppointmentHandler) Book(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.BookAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	result, err := h.apptService.Book(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	// Instantly notify the doctor of the incoming consultation request (for instant booking calls)
	h.hub.NotifyUser(result.Appointment.Doctor.User.ID, "incoming_call", result.Appointment)

	response.Created(c, "appointment booked", result)
}

// GET /api/v1/appointments/:id
func (h *AppointmentHandler) GetByID(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	appt, err := h.apptService.GetByID(c.Request.Context(), claims.UserID, claims.Role, id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "appointment", appt)
}

// PATCH /api/v1/appointments/:id/cancel
func (h *AppointmentHandler) Cancel(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	var req dto.CancelAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	// Fetch details first to know who to notify
	appt, err := h.apptService.GetByID(c.Request.Context(), claims.UserID, claims.Role, id)

	if err := h.apptService.Cancel(c.Request.Context(), claims.UserID, claims.Role, id, req.Reason); err != nil {
		middleware.MapError(c, err)
		return
	}

	if err == nil {
		targetUserID := appt.Patient.UserID
		if claims.UserID == appt.Patient.UserID {
			targetUserID = appt.Doctor.UserID
		}
		// Send decline notification
		h.hub.NotifyUser(targetUserID, "call_declined", map[string]interface{}{
			"appointment_id": id,
			"reason":         req.Reason,
		})
	}

	response.OK(c, "appointment cancelled", nil)
}

// PATCH /api/v1/appointments/:id/start
func (h *AppointmentHandler) Start(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	if err := h.apptService.Start(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}

	appt, err := h.apptService.GetByID(c.Request.Context(), claims.UserID, claims.Role, id)
	if err == nil {
		if claims.UserID == appt.Patient.UserID {
			// Patient is calling the doctor
			h.hub.NotifyUser(appt.Doctor.UserID, "incoming_call", appt)
		} else {
			// Doctor is accepting/starting the call
			h.hub.NotifyUser(appt.Patient.UserID, "call_accepted", appt)
		}
	}

	response.OK(c, "consultation started", nil)
}

// PATCH /api/v1/appointments/:id/complete  (doctor only)
func (h *AppointmentHandler) Complete(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	if err := h.apptService.Complete(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	// Best-effort: mark the LiveKit room as ended.
	_ = h.roomSvc.EndRoom(c.Request.Context(), id)
	response.OK(c, "consultation completed", nil)
}

// POST /api/v1/appointments/:id/review  (patient only)
func (h *AppointmentHandler) LeaveReview(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	var req dto.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	if err := h.apptService.LeaveReview(c.Request.Context(), claims.UserID, id, &req); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "review submitted", nil)
}

// ─── shared helper ────────────────────────────────────────────────────────────

func parseID(c *gin.Context, param string) (uint, error) {
	n, err := strconv.ParseUint(c.Param(param), 10, 64)
	return uint(n), err
}
