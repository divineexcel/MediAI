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

type MedicalRecordHandler struct {
	recordService service.MedicalRecordService
}

func NewMedicalRecordHandler(recordService service.MedicalRecordService) *MedicalRecordHandler {
	return &MedicalRecordHandler{recordService: recordService}
}

// GET /api/v1/records
func (h *MedicalRecordHandler) List(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	records, total, err := h.recordService.List(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "medical records", records, pagination.NewMeta(p, total))
}

// GET /api/v1/records/:id
func (h *MedicalRecordHandler) GetByID(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid record id", nil)
		return
	}

	rec, err := h.recordService.GetByID(c.Request.Context(), claims.UserID, id)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "medical record", rec)
}

// POST /api/v1/records
func (h *MedicalRecordHandler) Create(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	var req dto.CreateMedicalRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	rec, err := h.recordService.Create(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Created(c, "record created", rec)
}

// DELETE /api/v1/records/:id
func (h *MedicalRecordHandler) Delete(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid record id", nil)
		return
	}

	if err := h.recordService.Delete(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "record deleted", nil)
}

// GET /api/v1/records/prescriptions
func (h *MedicalRecordHandler) ListPrescriptions(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	p := pagination.FromContext(c)

	prescriptions, total, err := h.recordService.ListPrescriptions(c.Request.Context(), claims.UserID, p)
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.Paginated(c, "prescriptions", prescriptions, pagination.NewMeta(p, total))
}

// PATCH /api/v1/records/prescriptions/:id/fill
func (h *MedicalRecordHandler) MarkPrescriptionFilled(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := parseID(c, "id")
	if err != nil {
		response.BadRequest(c, "invalid prescription id", nil)
		return
	}

	if err := h.recordService.MarkPrescriptionFilled(c.Request.Context(), claims.UserID, id); err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "prescription marked as filled", nil)
}
