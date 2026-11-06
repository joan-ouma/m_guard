// Package sms sends text messages through Africa's Talking.
package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const africasTalkingMessagingURL = "https://api.africastalking.com/version1/messaging"

// Result is the delivery status Africa's Talking reports for one recipient.
type Result struct {
	Number     string
	Status     string
	StatusCode int
	MessageID  string
}

// Sender sends one message to many recipients.
type Sender interface {
	Send(ctx context.Context, recipients []string, message string) ([]Result, error)
}

// AfricasTalkingClient sends SMS through the Africa's Talking messaging API.
type AfricasTalkingClient struct {
	username   string
	apiKey     string
	shortcode  string
	endpoint   string
	httpClient *http.Client
}

func NewAfricasTalkingClient(username, apiKey, shortcode string) *AfricasTalkingClient {
	return &AfricasTalkingClient{
		username:   username,
		apiKey:     apiKey,
		shortcode:  shortcode,
		endpoint:   africasTalkingMessagingURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type atRecipient struct {
	StatusCode int    `json:"statusCode"`
	Number     string `json:"number"`
	Status     string `json:"status"`
	MessageID  string `json:"messageId"`
}

type atResponse struct {
	SMSMessageData struct {
		Message    string        `json:"Message"`
		Recipients []atRecipient `json:"Recipients"`
	} `json:"SMSMessageData"`
}

// Send posts the message to all recipients in one request. Numbers must be in international
// format (for example +254712345678).
func (c *AfricasTalkingClient) Send(ctx context.Context, recipients []string, message string) ([]Result, error) {
	if len(recipients) == 0 {
		return nil, nil
	}

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("to", strings.Join(recipients, ","))
	form.Set("message", message)
	if c.shortcode != "" {
		form.Set("from", c.shortcode)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", c.apiKey)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("africas talking request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("africas talking read failed: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("africas talking http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed atResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("africas talking response parse failed: %w", err)
	}

	out := make([]Result, 0, len(parsed.SMSMessageData.Recipients))
	for _, r := range parsed.SMSMessageData.Recipients {
		out = append(out, Result{
			Number:     r.Number,
			Status:     r.Status,
			StatusCode: r.StatusCode,
			MessageID:  r.MessageID,
		})
	}
	return out, nil
}

// IsAccepted reports whether Africa's Talking queued the message for this recipient.
// Status code 100 and 101 mean the message is sent or queued for delivery.
func (r Result) IsAccepted() bool {
	return r.StatusCode == 100 || r.StatusCode == 101 || strings.EqualFold(r.Status, "Success")
}

// NormalizePhone converts a Kenyan number as typed by a rider (0712 345 678, 254712345678,
// +254 712 345 678) into international format. It returns false when the number is not usable.
func NormalizePhone(raw string) (string, bool) {
	s := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(strings.TrimSpace(raw))
	if s == "" {
		return "", false
	}

	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case strings.HasPrefix(s, "0"):
		s = "254" + s[1:]
	case len(s) == 9 && (s[0] == '7' || s[0] == '1'):
		s = "254" + s
	}

	if _, err := strconv.ParseUint(s, 10, 64); err != nil {
		return "", false
	}
	if len(s) < 8 || len(s) > 15 {
		return "", false
	}
	return "+" + s, true
}
