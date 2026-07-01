package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/medisave/app/internal/infrastructure/rtc"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type CallHandler struct {
	hub *rtc.Hub
}

func NewCallHandler(hub *rtc.Hub) *CallHandler {
	return &CallHandler{hub: hub}
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
