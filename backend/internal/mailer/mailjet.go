package mailer

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

type GeofenceAlertData struct {
	BikeRegistration string
	BikeModel        string
	GeofenceName     string
	BreachType       string // e.g., "Boundary Exit"
	Timestamp        string
	Address          string
	SpeedKMH         string
	DashboardURL     string
}

type BikeTouchedAlertData struct {
	BikeRegistration string
	BikeModel        string
	Timestamp        string
	Address          string
	BatteryPercent   int
	DashboardURL     string
}

type Mailer interface {
	SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error
	SendPasswordChangedEmail(ctx context.Context, toEmail, toName string) error
	SendGeofenceBreachAlertEmail(ctx context.Context, toEmail, toName string, data GeofenceAlertData) error
	SendBikeTouchedAlertEmail(ctx context.Context, toEmail, toName string, data BikeTouchedAlertData) error
}

type MailjetMailer struct {
	apiKey       string
	secretKey    string
	senderEmail  string
	senderName   string
	replyToEmail string
	endpoint     string
	httpClient   *http.Client
	templates    *htmlTemplate.Template
}

type mailjetRecipient struct {
	Email string `json:"Email"`
	Name  string `json:"Name,omitempty"`
}

type mailjetMessage struct {
	From     mailjetRecipient   `json:"From"`
	ReplyTo  *mailjetRecipient  `json:"ReplyTo,omitempty"`
	To       []mailjetRecipient `json:"To"`
	Subject  string             `json:"Subject"`
	TextPart string             `json:"TextPart"`
	HTMLPart string             `json:"HTMLPart"`
}

type mailjetPayload struct {
	Messages []mailjetMessage `json:"Messages"`
}

type mailjetResponseItem struct {
	Status string `json:"Status"`
	Errors []struct {
		ErrorMessage string `json:"ErrorMessage"`
		StatusCode   int    `json:"StatusCode"`
	} `json:"Errors,omitempty"`
}

type mailjetResponse struct {
	Messages []mailjetResponseItem `json:"Messages"`
}

func NewMailjetMailer(apiKey, secretKey, senderEmail, senderName, replyToEmail string) *MailjetMailer {
	if senderName == "" {
		senderName = "MotoGuard"
	}
	if senderEmail == "" {
		senderEmail = "no-reply@jullie.tech"
	}
	if replyToEmail == "" {
		replyToEmail = "hello@jullie.tech"
	}

	tmpl, err := htmlTemplate.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Printf("[MailjetMailer] Warning: error parsing email templates: %v", err)
	}

	return &MailjetMailer{
		apiKey:       apiKey,
		secretKey:    secretKey,
		senderEmail:  senderEmail,
		senderName:   senderName,
		replyToEmail: replyToEmail,
		endpoint:     "https://api.mailjet.com/v3.1/send",
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		templates:    tmpl,
	}
}

// SetEndpoint allows overriding the Mailjet API endpoint for testing.
func (m *MailjetMailer) SetEndpoint(endpoint string) {
	m.endpoint = endpoint
}

// SetHTTPClient allows overriding the HTTP client for testing.
func (m *MailjetMailer) SetHTTPClient(client *http.Client) {
	m.httpClient = client
}

// SetReplyToEmail updates the reply-to address.
func (m *MailjetMailer) SetReplyToEmail(replyToEmail string) {
	m.replyToEmail = replyToEmail
}

func (m *MailjetMailer) send(ctx context.Context, toEmail, toName, subject, textPart, htmlPart string) error {
	msg := mailjetMessage{
		From: mailjetRecipient{
			Email: m.senderEmail,
			Name:  m.senderName,
		},
		To: []mailjetRecipient{
			{
				Email: toEmail,
				Name:  toName,
			},
		},
		Subject:  subject,
		TextPart: textPart,
		HTMLPart: htmlPart,
	}

	if m.replyToEmail != "" {
		msg.ReplyTo = &mailjetRecipient{
			Email: m.replyToEmail,
			Name:  m.senderName + " Support",
		}
	}

	payload := mailjetPayload{
		Messages: []mailjetMessage{msg},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal mailjet payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create mailjet request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(m.apiKey, m.secretKey)

	res, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mailjet request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read mailjet response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("mailjet api error (status %d): %s", res.StatusCode, string(respBody))
	}

	var parsedResp mailjetResponse
	if err := json.Unmarshal(respBody, &parsedResp); err == nil {
		for _, msg := range parsedResp.Messages {
			if strings.ToLower(msg.Status) == "error" {
				var errMsgs []string
				for _, e := range msg.Errors {
					errMsgs = append(errMsgs, e.ErrorMessage)
				}
				return fmt.Errorf("mailjet message delivery error: %s", strings.Join(errMsgs, "; "))
			}
		}
	}

	return nil
}

func (m *MailjetMailer) renderTemplate(name string, data any) (string, error) {
	if m.templates == nil {
		return "", fmt.Errorf("templates not loaded")
	}
	var buf bytes.Buffer
	if err := m.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", name, err)
	}
	return buf.String(), nil
}

func (m *MailjetMailer) SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error {
	displayName := toName
	if displayName == "" {
		displayName = "User"
	}

	subject := "Reset your MotoGuard password"

	textPart := fmt.Sprintf(`Hello %s,

We received a request to reset the password for your account. You can set a new password by opening the link below:

%s

This link will expire in 30 minutes. If you did not request a password reset, you can safely ignore this email.

MotoGuard &bull; Fleet Tracking & Security`, displayName, resetURL)

	htmlPart, err := m.renderTemplate("password_reset.html", map[string]any{
		"Name":         displayName,
		"ResetURL":     resetURL,
		"ReplyToEmail": m.replyToEmail,
	})
	if err != nil {
		return err
	}

	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendPasswordChangedEmail(ctx context.Context, toEmail, toName string) error {
	displayName := toName
	if displayName == "" {
		displayName = "User"
	}

	subject := "Your MotoGuard password has been changed"

	textPart := fmt.Sprintf(`Hello %s,

Your MotoGuard account password was recently updated. All existing active sessions have been terminated.

If you made this change, you can safely disregard this email.

If you did not initiate this change, please contact support at %s immediately.

MotoGuard &bull; Fleet Tracking & Security`, displayName, m.replyToEmail)

	htmlPart, err := m.renderTemplate("password_changed.html", map[string]any{
		"Name":         displayName,
		"ReplyToEmail": m.replyToEmail,
	})
	if err != nil {
		return err
	}

	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendGeofenceBreachAlertEmail(ctx context.Context, toEmail, toName string, data GeofenceAlertData) error {
	displayName := toName
	if displayName == "" {
		displayName = "Fleet Operator"
	}

	subject := fmt.Sprintf("[ALERT] Geofence Breach: %s - %s", data.BikeRegistration, data.GeofenceName)

	textPart := fmt.Sprintf(`GEOFENCE BREACH ALERT

Bike: %s (%s)
Geofence: %s
Alert Type: %s
Time: %s
Location: %s
Speed: %s km/h

View Live Map: %s

MotoGuard &bull; Fleet Tracking & Security`,
		data.BikeRegistration, data.BikeModel, data.GeofenceName, data.BreachType,
		data.Timestamp, data.Address, data.SpeedKMH, data.DashboardURL)

	htmlPart, err := m.renderTemplate("geofence_breach.html", data)
	if err != nil {
		return err
	}

	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendBikeTouchedAlertEmail(ctx context.Context, toEmail, toName string, data BikeTouchedAlertData) error {
	displayName := toName
	if displayName == "" {
		displayName = "Fleet Operator"
	}

	subject := fmt.Sprintf("[ALERT] Unauthorized Movement / Tamper: %s", data.BikeRegistration)

	textPart := fmt.Sprintf(`SECURITY ALERT - BIKE MOVEMENT DETECTED

Bike: %s (%s)
Event: Movement / Tamper Sensor Triggered
Ignition: OFF
Time: %s
Location: %s
Battery: %d%%

View Dashboard: %s

MotoGuard &bull; Fleet Tracking & Security`,
		data.BikeRegistration, data.BikeModel,
		data.Timestamp, data.Address, data.BatteryPercent, data.DashboardURL)

	htmlPart, err := m.renderTemplate("bike_touched.html", data)
	if err != nil {
		return err
	}

	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

type NoopMailer struct{}

func (n *NoopMailer) SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error {
	log.Printf("[NoopMailer] Password reset email for %s <%s>: %s", toName, toEmail, resetURL)
	return nil
}

func (n *NoopMailer) SendPasswordChangedEmail(ctx context.Context, toEmail, toName string) error {
	log.Printf("[NoopMailer] Password changed confirmation email for %s <%s>", toName, toEmail)
	return nil
}

func (n *NoopMailer) SendGeofenceBreachAlertEmail(ctx context.Context, toEmail, toName string, data GeofenceAlertData) error {
	log.Printf("[NoopMailer] Geofence breach alert email for %s: %s", data.BikeRegistration, data.GeofenceName)
	return nil
}

func (n *NoopMailer) SendBikeTouchedAlertEmail(ctx context.Context, toEmail, toName string, data BikeTouchedAlertData) error {
	log.Printf("[NoopMailer] Bike touched alert email for %s", data.BikeRegistration)
	return nil
}
