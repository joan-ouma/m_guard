package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiURL = "https://api.africastalking.com/version1/messaging"

type Client struct {
	username string
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

func NewClient(username, apiKey, from string) *Client {
	return &Client{
		username: strings.TrimSpace(username),
		apiKey:   strings.TrimSpace(apiKey),
		from:     strings.TrimSpace(from),
		endpoint: apiURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) SendSMS(ctx context.Context, to, message string) error {
	if c.username == "" || c.apiKey == "" {
		return fmt.Errorf("Africa's Talking SMS credentials are not configured")
	}
	if strings.TrimSpace(to) == "" || strings.TrimSpace(message) == "" {
		return fmt.Errorf("SMS recipient and message are required")
	}

	values := url.Values{}
	values.Set("username", c.username)
	values.Set("to", strings.TrimSpace(to))
	values.Set("message", message)
	if c.from != "" {
		values.Set("from", c.from)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("create Africa's Talking SMS request: %w", err)
	}
	req.Header.Set("apiKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send Africa's Talking SMS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Africa's Talking SMS API returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		SMSMessageData struct {
			Recipients []struct {
				StatusCode int    `json:"statusCode"`
				Status     string `json:"status"`
			} `json:"Recipients"`
		} `json:"SMSMessageData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Africa's Talking SMS response: %w", err)
	}
	if len(result.SMSMessageData.Recipients) == 0 {
		return fmt.Errorf("Africa's Talking SMS response contained no recipient result")
	}
	for _, recipient := range result.SMSMessageData.Recipients {
		if recipient.StatusCode != 101 || !strings.EqualFold(recipient.Status, "Success") {
			return fmt.Errorf("Africa's Talking SMS delivery was not accepted (status %d: %s)", recipient.StatusCode, recipient.Status)
		}
	}
	return nil
}
