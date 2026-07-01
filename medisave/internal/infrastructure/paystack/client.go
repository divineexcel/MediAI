package paystack

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(secretKey, baseURL string) *Client {
	return &Client{
		secretKey: secretKey,
		baseURL:   baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// ─── Request / Response types ─────────────────────────────────────────────────

type InitTransactionReq struct {
	Email       string `json:"email"`
	Amount      int64  `json:"amount"` // kobo
	Reference   string `json:"reference"`
	CallbackURL string `json:"callback_url,omitempty"`
}

type InitTransactionData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

type InitTransactionResp struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    InitTransactionData `json:"data"`
}

type VerifyData struct {
	Status    string `json:"status"` // "success" | "failed" | "abandoned"
	Amount    int64  `json:"amount"` // kobo
	Reference string `json:"reference"`
	PaidAt    string `json:"paid_at"`
}

type VerifyResp struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    VerifyData `json:"data"`
}

type TransferRecipientReq struct {
	Type          string `json:"type"`
	Name          string `json:"name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Currency      string `json:"currency"`
}

type TransferRecipientResp struct {
	Status bool   `json:"status"`
	Data   struct {
		RecipientCode string `json:"recipient_code"`
	} `json:"data"`
}

type InitiateTransferReq struct {
	Source    string `json:"source"`
	Amount    int64  `json:"amount"` // kobo
	Recipient string `json:"recipient"`
	Reason    string `json:"reason"`
	Reference string `json:"reference"`
}

type WebhookEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// ─── Methods ─────────────────────────────────────────────────────────────────

func (c *Client) InitializeTransaction(req *InitTransactionReq) (*InitTransactionData, error) {
	body, _ := json.Marshal(req)
	resp, err := c.post("/transaction/initialize", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result InitTransactionResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack: %s", result.Message)
	}
	return &result.Data, nil
}

func (c *Client) VerifyTransaction(reference string) (*VerifyData, error) {
	resp, err := c.get(fmt.Sprintf("/transaction/verify/%s", reference))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VerifyResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if !result.Status {
		return nil, fmt.Errorf("paystack: %s", result.Message)
	}
	return &result.Data, nil
}

func (c *Client) CreateTransferRecipient(name, accountNo, bankCode string) (string, error) {
	req := TransferRecipientReq{
		Type:          "nuban",
		Name:          name,
		AccountNumber: accountNo,
		BankCode:      bankCode,
		Currency:      "NGN",
	}
	body, _ := json.Marshal(req)
	resp, err := c.post("/transferrecipient", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result TransferRecipientResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.Status {
		return "", fmt.Errorf("paystack: failed to create transfer recipient")
	}
	return result.Data.RecipientCode, nil
}

func (c *Client) InitiateTransfer(amountKobo int64, recipientCode, reference, reason string) error {
	req := InitiateTransferReq{
		Source:    "balance",
		Amount:    amountKobo,
		Recipient: recipientCode,
		Reason:    reason,
		Reference: reference,
	}
	body, _ := json.Marshal(req)
	resp, err := c.post("/transfer", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Status {
		return fmt.Errorf("paystack: %s", result.Message)
	}
	return nil
}

func (c *Client) ValidateSignature(body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(c.secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (c *Client) ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) post(path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	return c.httpClient.Do(req)
}

// ReadBody reads the full response body as bytes (for webhook processing).
func ReadBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}
