package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/response"
	"github.com/medisave/app/pkg/validator"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	authResp, err := h.authService.RegisterPatient(c.Request.Context(), &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.Created(c, "registration successful", authResp)
}

// POST /api/v1/auth/register/doctor
func (h *AuthHandler) RegisterDoctor(c *gin.Context) {
	var req dto.DoctorRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	authResp, err := h.authService.RegisterDoctor(c.Request.Context(), &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.Created(c, "doctor registration successful. your account is pending admin verification", authResp)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	authResp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "login successful", authResp)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "token refreshed", tokens)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT is stateless: logout is client-side token removal.
	// A Redis blacklist can be wired here in production.
	response.OK(c, "logged out successfully", nil)
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)

	user, err := h.authService.GetCurrentUser(c.Request.Context(), claims.UserID)
	if err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "user retrieved", dto.AuthUserResponse{
		ID:              user.ID,
		UUID:            user.UUID,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Email:           user.Email,
		Phone:           user.Phone,
		Role:            user.Role,
		IsVerified:      user.IsVerified,
		ProfilePhotoURL: user.ProfilePhotoURL,
	})
}

// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	// Email/SMS delivery is wired in Step 15.
	// For now, returns success to not leak user existence.
	response.OK(c, "if your account exists, a reset link has been sent", nil)
}

// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}
	if errs := validator.Validate(req); len(errs) > 0 {
		response.UnprocessableEntity(c, "validation failed", errs)
		return
	}

	claims := middleware.ClaimsFromContext(c)

	if err := h.authService.ChangePassword(c.Request.Context(), claims.UserID, &req); err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "password changed successfully", nil)
}

// POST /api/v1/auth/upload-document
// Accepts multipart/form-data with field "document". Returns the stored URL.
// No auth required — called during doctor registration before account exists.
func (h *AuthHandler) UploadDocument(c *gin.Context) {
	const maxSize = 10 << 20 // 10 MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

	file, header, err := c.Request.FormFile("document")
	if err != nil {
		response.BadRequest(c, "document file is required", err.Error())
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".pdf": true}
	if !allowed[ext] {
		response.BadRequest(c, "only JPG, PNG, and PDF files are allowed", nil)
		return
	}

	uploadDir := "./data/uploads/documents"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		response.InternalError(c, "could not create upload directory")
		return
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), uuid.NewString()[:8], ext)
	dest := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(header, dest); err != nil {
		response.InternalError(c, "failed to save file")
		return
	}

	url := "/uploads/documents/" + filename
	response.Created(c, "document uploaded", gin.H{"url": url})
}

// PATCH /api/v1/auth/fcm-token
func (h *AuthHandler) UpdateFCMToken(c *gin.Context) {
	var req dto.UpdateFCMTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	claims := middleware.ClaimsFromContext(c)

	if err := h.authService.UpdateFCMToken(c.Request.Context(), claims.UserID, req.FCMToken); err != nil {
		middleware.MapError(c, err)
		return
	}

	response.OK(c, "fcm token updated", nil)
}
