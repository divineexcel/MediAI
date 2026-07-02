package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/presentation/http/middleware"
	"github.com/medisave/app/pkg/response"
)

// RoomHandler provides endpoints for LiveKit room token generation.
type RoomHandler struct {
	roomSvc service.ConsultationRoomService
}

func NewRoomHandler(roomSvc service.ConsultationRoomService) *RoomHandler {
	return &RoomHandler{roomSvc: roomSvc}
}

// GET /api/v1/appointments/:id/room-token
// Returns a signed LiveKit token so the caller can join the consultation room.
// Only the assigned doctor and patient receive a token; all others get 403.
func (h *RoomHandler) GetToken(c *gin.Context) {
	claims := middleware.ClaimsFromContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid appointment id", nil)
		return
	}

	resp, err := h.roomSvc.GetOrCreateRoom(c.Request.Context(), claims.UserID, claims.Role, uint(id))
	if err != nil {
		middleware.MapError(c, err)
		return
	}
	response.OK(c, "room token", resp)
}
