package livekit

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Client generates LiveKit access tokens. It requires no network calls —
// tokens are pure JWTs signed with the API secret and validated by the
// LiveKit server on connection.
type Client struct {
	WSURL     string
	APIKey    string
	APISecret string
}

func NewClient(wsURL, apiKey, apiSecret string) *Client {
	return &Client{WSURL: wsURL, APIKey: apiKey, APISecret: apiSecret}
}

type videoGrant struct {
	RoomJoin       bool   `json:"roomJoin,omitempty"`
	Room           string `json:"room,omitempty"`
	CanPublish     bool   `json:"canPublish,omitempty"`
	CanSubscribe   bool   `json:"canSubscribe,omitempty"`
	CanPublishData bool   `json:"canPublishData,omitempty"`
	RoomCreate     bool   `json:"roomCreate,omitempty"`
	RoomAdmin      bool   `json:"roomAdmin,omitempty"`
}

type livekitClaims struct {
	Video *videoGrant `json:"video,omitempty"`
	jwt.RegisteredClaims
}

// TokenForRoom returns a signed LiveKit access token.
// identity should be unique per participant (e.g. "doctor-42" or "patient-7").
// isHost grants room-create and room-admin permissions (doctor role).
func (c *Client) TokenForRoom(roomName, identity string, isHost bool) (string, error) {
	now := time.Now()
	claims := livekitClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.APIKey,
			Subject:   identity,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(4 * time.Hour)),
		},
		Video: &videoGrant{
			RoomJoin:       true,
			Room:           roomName,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
			RoomCreate:     isHost,
			RoomAdmin:      isHost,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.APISecret))
}

// RoomName returns the canonical LiveKit room name for a given appointment.
func RoomName(appointmentID uint) string {
	return fmt.Sprintf("consultation-%d", appointmentID)
}
