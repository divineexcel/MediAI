package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/response"
)

// MapError translates domain errors to HTTP responses.
// Use this in every handler instead of writing switch statements repeatedly.
func MapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, pkgerrors.ErrInvalidCredentials):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, pkgerrors.ErrAccountInactive):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, pkgerrors.ErrAccountUnverified):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, pkgerrors.ErrTokenExpired),
		errors.Is(err, pkgerrors.ErrTokenInvalid):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, pkgerrors.ErrForbidden):
		response.Forbidden(c, err.Error())

	case errors.Is(err, pkgerrors.ErrEmailExists),
		errors.Is(err, pkgerrors.ErrPhoneExists),
		errors.Is(err, pkgerrors.ErrLicenseExists),
		errors.Is(err, pkgerrors.ErrReviewExists):
		response.Conflict(c, err.Error())

	case errors.Is(err, pkgerrors.ErrUserNotFound),
		errors.Is(err, pkgerrors.ErrPatientNotFound),
		errors.Is(err, pkgerrors.ErrDoctorNotFound),
		errors.Is(err, pkgerrors.ErrWalletNotFound),
		errors.Is(err, pkgerrors.ErrAppointmentNotFound),
		errors.Is(err, pkgerrors.ErrConsultationNotFound),
		errors.Is(err, pkgerrors.ErrRecordNotFound),
		errors.Is(err, pkgerrors.ErrEmergencyNotFound),
		errors.Is(err, pkgerrors.ErrReminderNotFound),
		errors.Is(err, pkgerrors.ErrReviewNotFound),
		errors.Is(err, pkgerrors.ErrNotFound):
		response.NotFound(c, err.Error())

	case errors.Is(err, pkgerrors.ErrInsufficientFunds):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrWalletInactive):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrDoctorUnavailable):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrDoctorNotVerified):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrAppointmentConflict):
		response.Conflict(c, err.Error())
	case errors.Is(err, pkgerrors.ErrAppointmentNotPending):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrAppointmentNotInProgress):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrScheduledTooSoon):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrInvalidScheduleFormat):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrCompletedOnly):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrConsultationInactive):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrAccessDenied):
		response.Forbidden(c, err.Error())
	case errors.Is(err, pkgerrors.ErrBadRequest):
		response.BadRequest(c, err.Error(), nil)
	case errors.Is(err, pkgerrors.ErrInternalServer):
		response.InternalError(c, "something went wrong on our end, please try again")

	default:
		response.InternalError(c, "an unexpected error occurred")
	}
}
