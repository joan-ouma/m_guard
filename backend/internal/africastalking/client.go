package africastalking

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// SMSRecipient represents the delivery status of a single recipient.
type SMSRecipient struct {
	Number    string `json:"number"`
	Status    string `json:"status"`
	Cost      string `json:"cost"`
	MessageID string `json:"messageId"`
}

// SMSMessageData represents the container of SMS delivery data.
type SMSMessageData struct {
	Message    string         `json:"Message"`
	Recipients []SMSRecipient `json:"Recipients"`
}

// SMSResponse is the Africa's Talking API response structure.
type SMSResponse struct {
	SMSMessageData SMSMessageData `json:"SMSMessageData"`
}

// OutboundSMSLog records an SMS sent by the system (for testing & auditing).
type OutboundSMSLog struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	From      string    `json:"from"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Client defines the Africa's Talking operations interface.
type Client interface {
	SendSMS(ctx context.Context, to []string, message string, from string) (*SMSResponse, error)
	GetRecentOutboundSMS(limit int) []OutboundSMSLog
}

// ATClient is the HTTP client implementation for Africa's Talking.
type ATClient struct {
	username   string
	apiKey     string
	isSandbox  bool
	httpClient *http.Client
	logsMu     sync.RWMutex
	logs       []OutboundSMSLog
}

// NewClient creates a new Africa's Talking client.
func NewClient(username, apiKey string, isSandbox bool) *ATClient {
	if username == "" {
		username = "sandbox"
	}
	if username == "sandbox" {
		isSandbox = true
	}

	return &ATClient{
		username:   username,
		apiKey:     apiKey,
		isSandbox:  isSandbox,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logs:       make([]OutboundSMSLog, 0),
	}
}

// SendSMS is deactivated.
func (c *ATClient) SendSMS(ctx context.Context, to []string, message string, from string) (*SMSResponse, error) {
	// SMS dispatch deactivated
	return &SMSResponse{
		SMSMessageData: SMSMessageData{
			Message: "SMS dispatch is deactivated",
		},
	}, nil
}

// GetRecentOutboundSMS returns the most recent outbound SMS log entries.
func (c *ATClient) GetRecentOutboundSMS(limit int) []OutboundSMSLog {
	c.logsMu.RLock()
	defer c.logsMu.RUnlock()

	if limit <= 0 || limit > len(c.logs) {
		limit = len(c.logs)
	}

	res := make([]OutboundSMSLog, 0, limit)
	for i := len(c.logs) - 1; i >= 0 && len(res) < limit; i-- {
		res = append(res, c.logs[i])
	}
	return res
}
