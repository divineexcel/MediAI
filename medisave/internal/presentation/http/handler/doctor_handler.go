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

type DoctorHandler struct {
	doctorService service.DoctorService
}

func NewDoctorHandler(doctorService service.DoctorService) *DoctorHandler {
	return &DoctorHandler{doctorService: doctorService}
}

// GET /api/v1/doctors  (public)
func (h *DoctorHandler) List(c *gin.Context) {
	var filter dto.DoctorListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.BadRequest(c, "invalid query params", err.Error())
		return
	}

	p := pagination.FromContext(c)
	doctors, total, err := h.doctorService.List(c.Request.Context(), filter, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.Paginated(c, "doctors loaded", doctors, pagination.NewMeta(p, total))
}

// GET /api/v1/doctors/:id  (public)
func (h *DoctorHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid doctor id", nil)
		return
	}

	doctor, err := h.doctorService.GetDoctorByID(c.Request.Context(), uint(id))
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "doctor loaded", buildDoctorResponse(doctor))
}

// GET /api/v1/doctors/me/dashboard  (doctor only)
func (h *DoctorHandler) Dashboard(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	dash, err := h.doctorService.GetDashboard(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "dashboard loaded", dash)
}

// GET /api/v1/doctors/me/profile  (doctor only)
func (h *DoctorHandler) GetMyProfile(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	doctor, err := h.doctorService.GetProfile(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "profile loaded", buildDoctorResponse(doctor))
}

// PUT /api/v1/doctors/me/profile  (doctor only)
func (h *DoctorHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateDoctorProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	doctor, err := h.doctorService.UpdateProfile(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "profile updated", buildDoctorResponse(doctor))
}

// PATCH /api/v1/doctors/me/availability  (doctor only)
func (h *DoctorHandler) ToggleAvailability(c *gin.Context) {
	var req dto.ToggleAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	claims := middleware.ClaimsFromContext(c)

	if err := h.doctorService.ToggleAvailability(c.Request.Context(), claims.UserID, req.IsAvailable); err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "availability updated", gin.H{"is_available": req.IsAvailable})
}

// GET /api/v1/doctors/me/today  (doctor only)
func (h *DoctorHandler) TodayAppointments(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	appts, err := h.doctorService.GetTodayAppointments(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "today appointments", appts)
}

// GET /api/v1/doctors/me/analytics  (doctor only)
func (h *DoctorHandler) Analytics(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	dash, err := h.doctorService.GetDashboard(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "analytics loaded", gin.H{
		"total_consultations": dash.TotalConsultations,
		"total_earnings":      dash.TotalEarnings,
		"rating":              dash.Rating,
		"wallet_balance":      dash.WalletBalance,
	})
}

// GET /api/v1/doctors/:id/reviews  (public)
func (h *DoctorHandler) GetReviews(c *gin.Context) {
	// Populated in Step 9 (Review module)
	response.OK(c, "reviews loaded", gin.H{"reviews": []interface{}{}, "total": 0})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildDoctorResponse(d interface{}) interface{} {
	return d
}
