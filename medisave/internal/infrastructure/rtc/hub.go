// Package rtc implements a WebRTC signaling hub over WebSocket.
// Each consultation room holds exactly two peers; all signaling messages
// (offer / answer / ice-candidate) are forwarded transparently.
package rtc

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 64 * 1024 // 64 KB — enough for any SDP
	maxPeers   = 2
)

// ─── Message ─────────────────────────────────────────────────────────────────

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func marshal(t string, payload interface{}) []byte {
	var raw json.RawMessage
	if payload != nil {
		raw, _ = json.Marshal(payload)
	}
	b, _ := json.Marshal(Message{Type: t, Payload: raw})
	return b
}

// ─── Room ────────────────────────────────────────────────────────────────────

type room struct {
	mu      sync.Mutex
	clients []*Client
}

func (r *room) add(c *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.clients) >= maxPeers {
		return false
	}
	r.clients = append(r.clients, c)
	return true
}

func (r *room) remove(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, cl := range r.clients {
		if cl == c {
			r.clients = append(r.clients[:i], r.clients[i+1:]...)
			return
		}
	}
}

func (r *room) others(c *Client) []*Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Client
	for _, cl := range r.clients {
		if cl != c {
			out = append(out, cl)
		}
	}
	return out
}

func (r *room) size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.clients)
}

// ─── Hub ─────────────────────────────────────────────────────────────────────

type Hub struct {
	mu        sync.RWMutex
	rooms     map[string]*room
	userConns map[uint][]*UserClient
}

func NewHub() *Hub {
	return &Hub{
		rooms:     make(map[string]*room),
		userConns: make(map[uint][]*UserClient),
	}
}

func (h *Hub) RegisterUser(userID uint, conn *websocket.Conn) *UserClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	client := &UserClient{
		hub:    h,
		userID: userID,
		conn:   conn,
		outbox: make(chan []byte, 64),
	}
	h.userConns[userID] = append(h.userConns[userID], client)
	return client
}

func (h *Hub) UnregisterUser(userID uint, client *UserClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.userConns[userID]
	for i, c := range conns {
		if c == client {
			h.userConns[userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.userConns[userID]) == 0 {
		delete(h.userConns, userID)
	}
}

func (h *Hub) NotifyUser(userID uint, msgType string, payload interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.userConns[userID]
	if !ok {
		return
	}
	msg := marshal(msgType, payload)
	for _, c := range conns {
		c.send(msg)
	}
}

func (h *Hub) getOrCreate(roomID string) *room {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.rooms[roomID]
	if !ok {
		r = &room{}
		h.rooms[roomID] = r
	}
	return r
}

func (h *Hub) cleanup(roomID string, r *room) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.size() == 0 {
		delete(h.rooms, roomID)
	}
}

// Join registers a client in its room, returning existing peer count (0 or 1).
// Returns -1 if the room is already full.
func (h *Hub) Join(c *Client) int {
	r := h.getOrCreate(c.roomID)
	existing := r.size()
	if !r.add(c) {
		return -1
	}
	// Notify existing peers
	for _, peer := range r.others(c) {
		peer.send(marshal("peer-joined", nil))
	}
	return existing
}

// Leave removes a client and notifies remaining peers.
func (h *Hub) Leave(c *Client) {
	r := h.getOrCreate(c.roomID)
	r.remove(c)
	for _, peer := range r.others(c) {
		peer.send(marshal("peer-left", nil))
	}
	h.cleanup(c.roomID, r)
}

// Relay forwards a signal message to all other peers in the room.
func (h *Hub) Relay(from *Client, msg []byte) {
	r := h.getOrCreate(from.roomID)
	for _, peer := range r.others(from) {
		peer.send(msg)
	}
}

// ─── Client ──────────────────────────────────────────────────────────────────

type Client struct {
	hub    *Hub
	roomID string
	conn   *websocket.Conn
	outbox chan []byte
}

func NewClient(hub *Hub, roomID string, conn *websocket.Conn) *Client {
	return &Client{hub: hub, roomID: roomID, conn: conn, outbox: make(chan []byte, 64)}
}

func (c *Client) send(msg []byte) {
	select {
	case c.outbox <- msg:
	default:
	}
}

// ReadPump reads inbound messages and either relays or handles join/leave.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Leave(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Register in hub
	existing := c.hub.Join(c)
	if existing == -1 {
		c.conn.WriteMessage(websocket.TextMessage, marshal("room-full", nil))
		c.conn.Close()
		return
	}
	c.send(marshal("joined", map[string]int{"peers": existing}))

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// Relay everything (offer / answer / ice-candidate) to other peer
		c.hub.Relay(c, raw)
	}
}

// WritePump drains the outbox and sends ping frames to keep the connection alive.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.outbox:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ─── UserClient ──────────────────────────────────────────────────────────────

type UserClient struct {
	hub    *Hub
	userID uint
	conn   *websocket.Conn
	outbox chan []byte
}

func (c *UserClient) send(msg []byte) {
	select {
	case c.outbox <- msg:
	default:
	}
}

func (c *UserClient) ReadPump() {
	defer func() {
		c.hub.UnregisterUser(c.userID, c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *UserClient) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.outbox:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
