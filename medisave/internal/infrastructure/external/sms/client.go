package sms

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/medisave/app/pkg/logger"
	"go.uber.org/zap"
)

type Client struct {
	apiKey     string
	username   string
	senderID   string
	gatewayURL string
	httpClient *http.Client
}

type Message struct {
	To      string
	Message string
}

func NewClient(apiKey, username, senderID, gatewayURL string) *Client {
	if gatewayURL == "" {
		gatewayURL = "https://api.africastalking.com/version1/messaging"
	}
	return &Client{
		apiKey:     apiKey,
		username:   username,
		senderID:   senderID,
		gatewayURL: gatewayURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send sends a single SMS. Falls back to logging if no API key configured.
func (c *Client) Send(to, message string) error {
	if c.apiKey == "" {
		logger.Info("SMS (dev mode — no API key)", zap.String("to", to), zap.String("message", message))
		return nil
	}
	return c.africasTalking([]Message{{To: to, Message: message}})
}

// SendBulk sends the same message to multiple recipients.
func (c *Client) SendBulk(recipients []string, message string) error {
	if c.apiKey == "" {
		for _, r := range recipients {
			logger.Info("SMS bulk (dev mode)", zap.String("to", r), zap.String("message", message))
		}
		return nil
	}
	msgs := make([]Message, len(recipients))
	for i, r := range recipients {
		msgs[i] = Message{To: r, Message: message}
	}
	return c.africasTalking(msgs)
}

func (c *Client) africasTalking(msgs []Message) error {
	recipients := make([]string, len(msgs))
	for i, m := range msgs {
		recipients[i] = m.To
	}

	// Africa's Talking requires form-encoded body
	body := url.Values{}
	body.Set("username", c.username)
	body.Set("to", strings.Join(recipients, ","))
	body.Set("message", msgs[0].Message) // bulk uses same message
	if c.senderID != "" {
		body.Set("from", c.senderID)
	}

	req, err := http.NewRequest("POST", c.gatewayURL, strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apiKey", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SMS gateway returned %d", resp.StatusCode)
	}
	return nil
}
