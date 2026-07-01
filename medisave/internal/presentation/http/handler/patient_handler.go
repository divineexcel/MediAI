package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/pagination"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type PatientHandler struct {
	patientService service.PatientService
	notifService   service.NotificationService
}

func NewPatientHandler(patientService service.PatientService, notifService service.NotificationService) *PatientHandler {
	return &PatientHandler{patientService: patientService, notifService: notifService}
}

// GET /api/v1/patients/dashboard
func (h *PatientHandler) Dashboard(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	dash, err := h.patientService.GetDashboard(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "dashboard loaded", dash)
}

// GET /api/v1/patients/profile
func (h *PatientHandler) GetProfile(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	patient, err := h.patientService.GetProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "profile loaded", buildPatientResponse(patient))
}

// PUT /api/v1/patients/profile
func (h *PatientHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdatePatientProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	patient, err := h.patientService.UpdateProfile(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "profile updated", buildPatientResponse(patient))
}

// GET /api/v1/patients/health-score
func (h *PatientHandler) GetHealthScore(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	patient, err := h.patientService.GetProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "health score", gin.H{
		"score":       patient.HealthScore,
		"category":    scoreCategory(patient.HealthScore),
		"description": scoreDescription(patient.HealthScore),
	})
}

// GET /api/v1/patients/notifications
func (h *PatientHandler) GetNotifications(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	notifs, total, err := h.notifService.ListByUser(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.Paginated(c, "notifications loaded", notifs, pagination.NewMeta(p, total))
}

// PATCH /api/v1/patients/notifications/:id/read
func (h *PatientHandler) MarkNotificationRead(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid notification id", nil)
		return
	}

	if err := h.notifService.MarkRead(c.Request.Context(), uint(id), claims.UserID); err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "notification marked as read", nil)
}

// PATCH /api/v1/patients/notifications/read-all
func (h *PatientHandler) MarkAllNotificationsRead(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	if err := h.notifService.MarkAllRead(c.Request.Context(), claims.UserID); err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "all notifications marked as read", nil)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildPatientResponse(patient interface{}) interface{} {
	return patient
}

func scoreCategory(score int) string {
	switch {
	case score >= 80:
		return "Excellent"
	case score >= 60:
		return "Good"
	case score >= 40:
		return "Fair"
	default:
		return "Poor"
	}
}

func scoreDescription(score int) string {
	switch {
	case score >= 80:
		return "Your health metrics are looking great. Keep it up!"
	case score >= 60:
		return "Your health is good. A few improvements can make it great."
	case score >= 40:
		return "Your health needs attention. Consider booking a doctor."
	default:
		return "Please consult a doctor as soon as possible."
	}
}
