package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/medisave/app/internal/infrastructure/rtc"
	pkgjwt "github.com/medisave/app/pkg/jwt"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type CallHandler struct {
	hub        *rtc.Hub
	jwtManager *pkgjwt.Manager
}

func NewCallHandler(hub *rtc.Hub, jwtManager *pkgjwt.Manager) *CallHandler {
	return &CallHandler{hub: hub, jwtManager: jwtManager}
}

// GET /api/v1/consultations/:appointment_id/call/signal  (WebSocket)
func (h *CallHandler) Signal(c *gin.Context) {
	roomID := c.Param("appointment_id")
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := rtc.NewClient(h.hub, roomID, conn)
	go client.WritePump()
	client.ReadPump() // blocks until disconnect
}

// GET /api/v1/ws  (WebSocket notification and call overlay socket)
func (h *CallHandler) ConnectUserWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := h.hub.RegisterUser(claims.UserID, conn)
	go client.WritePump()
	client.ReadPump() // blocks until disconnect
}
