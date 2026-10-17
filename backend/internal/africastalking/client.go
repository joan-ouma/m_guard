package africastalking

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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

// SendSMS dispatches SMS message to one or more recipient phone numbers.
func (c *ATClient) SendSMS(ctx context.Context, to []string, message string, from string) (*SMSResponse, error) {
	if len(to) == 0 {
		return nil, fmt.Errorf("recipient list cannot be empty")
	}

	cleanedTo := make([]string, 0, len(to))
	for _, num := range to {
		trimmed := strings.TrimSpace(num)
		if trimmed != "" {
			cleanedTo = append(cleanedTo, trimmed)
		}
	}

	if len(cleanedTo) == 0 {
		return nil, fmt.Errorf("no valid phone numbers provided")
	}

	toJoin := strings.Join(cleanedTo, ",")

	// If no API key configured or running purely in offline/local mock mode
	if c.apiKey == "" {
		slog.Warn("Africa's Talking API key not configured; recording SMS to local simulator buffer",
			"to", toJoin,
			"message", message,
		)

		res := &SMSResponse{
			SMSMessageData: SMSMessageData{
				Message: fmt.Sprintf("Simulated delivery to %d recipients", len(cleanedTo)),
				Recipients: make([]SMSRecipient, len(cleanedTo)),
			},
		}

		c.logsMu.Lock()
		for i, num := range cleanedTo {
			simID := fmt.Sprintf("ATXid_sim_%d_%d", time.Now().UnixNano(), i)
			res.SMSMessageData.Recipients[i] = SMSRecipient{
				Number:    num,
				Status:    "Success (Simulated)",
				Cost:      "KES 0.00",
				MessageID: simID,
			}
			c.logs = append(c.logs, OutboundSMSLog{
				ID:        simID,
				To:        num,
				From:      from,
				Message:   message,
				Status:    "Delivered (Simulated)",
				CreatedAt: time.Now().UTC(),
			})
		}
		c.logsMu.Unlock()

		return res, nil
	}

	apiEndpoint := "https://api.africastalking.com/version1/messaging"
	if c.isSandbox {
		apiEndpoint = "https://api.sandbox.africastalking.com/version1/messaging"
	}

	formData := url.Values{}
	formData.Set("username", c.username)
	formData.Set("to", toJoin)
	formData.Set("message", message)
	if strings.TrimSpace(from) != "" {
		formData.Set("from", from)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create sms request error: %w", err)
	}

	req.Header.Set("apiKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Africa's Talking SMS HTTP request failed", "error", err, "endpoint", apiEndpoint)
		return nil, fmt.Errorf("sms http dispatch error: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read sms response body error: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("Africa's Talking returned non-2xx status", "status", resp.StatusCode, "body", string(bodyBytes))
		return nil, fmt.Errorf("sms api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var atResp SMSResponse
	if err := json.Unmarshal(bodyBytes, &atResp); err != nil {
		slog.Error("Failed to decode Africa's Talking response JSON", "error", err, "body", string(bodyBytes))
		return nil, fmt.Errorf("decode sms response error: %w", err)
	}

	// Record in memory audit log
	c.logsMu.Lock()
	for _, rec := range atResp.SMSMessageData.Recipients {
		c.logs = append(c.logs, OutboundSMSLog{
			ID:        rec.MessageID,
			To:        rec.Number,
			From:      from,
			Message:   message,
			Status:    rec.Status,
			CreatedAt: time.Now().UTC(),
		})
	}
	c.logsMu.Unlock()

	slog.Info("Africa's Talking SMS dispatched",
		"recipients_count", len(atResp.SMSMessageData.Recipients),
		"summary", atResp.SMSMessageData.Message,
	)

	return &atResp, nil
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
