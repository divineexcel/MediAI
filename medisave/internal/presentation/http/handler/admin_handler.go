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

type AdminHandler struct {
	adminService service.AdminService
}

func NewAdminHandler(adminService service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// GET /api/v1/admin/dashboard
func (h *AdminHandler) Dashboard(c *gin.Context) {
	analytics, err := h.adminService.GetAnalytics(c.Request.Context())
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "admin dashboard", analytics)
}

// GET /api/v1/admin/analytics
func (h *AdminHandler) Analytics(c *gin.Context) {
	analytics, err := h.adminService.GetAnalytics(c.Request.Context())
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "analytics", analytics)
}

// GET /api/v1/admin/patients
func (h *AdminHandler) ListPatients(c *gin.Context) {
	p := pagination.FromContext(c)
	patients, total, err := h.adminService.ListPatients(c.Request.Context(), p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "patients", patients, pagination.NewMeta(p, total))
}

// GET /api/v1/admin/patients/:id
func (h *AdminHandler) GetPatient(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid patient id", nil)
		return
	}
	patient, err := h.adminService.GetPatient(c.Request.Context(), id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "patient", patient)
}

// GET /api/v1/admin/doctors
func (h *AdminHandler) ListDoctors(c *gin.Context) {
	p := pagination.FromContext(c)
	doctors, total, err := h.adminService.ListDoctors(c.Request.Context(), p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "doctors", doctors, pagination.NewMeta(p, total))
}

// GET /api/v1/admin/doctors/:id
func (h *AdminHandler) GetDoctor(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid doctor id", nil)
		return
	}
	doctor, err := h.adminService.GetDoctor(c.Request.Context(), id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "doctor", doctor)
}

// PATCH /api/v1/admin/doctors/:id/verify
func (h *AdminHandler) VerifyDoctor(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid doctor id", nil)
		return
	}

	var req dto.VerifyDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	if err := h.adminService.VerifyDoctor(c.Request.Context(), id, &req); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "doctor status updated", nil)
}

// GET /api/v1/admin/transactions
func (h *AdminHandler) ListTransactions(c *gin.Context) {
	p := pagination.FromContext(c)
	txs, total, err := h.adminService.ListTransactions(c.Request.Context(), p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "transactions", txs, pagination.NewMeta(p, total))
}

// GET /api/v1/admin/appointments
func (h *AdminHandler) ListAppointments(c *gin.Context) {
	p := pagination.FromContext(c)
	appts, total, err := h.adminService.ListAppointments(c.Request.Context(), p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "appointments", appts, pagination.NewMeta(p, total))
}

// GET /api/v1/admin/emergencies
func (h *AdminHandler) ListEmergencies(c *gin.Context) {
	emergencies, err := h.adminService.ListEmergencies(c.Request.Context())
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "emergencies", emergencies)
}

// POST /api/v1/admin/campaigns
func (h *AdminHandler) SendCampaign(c *gin.Context) {
	var req dto.HealthCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	if err := h.adminService.SendCampaign(c.Request.Context(), &req); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "campaign queued", nil)
}
