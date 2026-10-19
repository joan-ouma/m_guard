package mailer

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
	textTemplate "text/template"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

type GeofenceAlertData struct {
	BikeRegistration string
	BikeModel        string
	GeofenceName     string
	BreachType       string
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

type TrackerAttachedEmailData struct {
	RiderName        string
	RiderEmail       string
	BikeRegistration string
	BikeMake         string
	BikeModel        string
	IMEI             string
	AttachedAt       string
	DashboardURL     string
	ReplyToEmail     string
}

type Mailer interface {
	SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error
	SendPasswordChangedEmail(ctx context.Context, toEmail, toName string) error
	SendGeofenceBreachAlertEmail(ctx context.Context, toEmail, toName string, data GeofenceAlertData) error
	SendBikeTouchedAlertEmail(ctx context.Context, toEmail, toName string, data BikeTouchedAlertData) error
	SendTrackerAttachedEmail(ctx context.Context, toEmail, toName string, data TrackerAttachedEmailData) error
}

type MailjetMailer struct {
	apiKey        string
	secretKey     string
	senderEmail   string
	senderName    string
	replyToEmail  string
	endpoint      string
	httpClient    *http.Client
	htmlTemplates *htmlTemplate.Template
	textTemplates *textTemplate.Template
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

	htmlTmpl, err := htmlTemplate.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		slog.Warn("Mailjet error parsing html templates", "error", err)
	}

	textTmpl, err := textTemplate.ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		slog.Warn("Mailjet error parsing text templates", "error", err)
	}

	return &MailjetMailer{
		apiKey:        apiKey,
		secretKey:     secretKey,
		senderEmail:   senderEmail,
		senderName:    senderName,
		replyToEmail:  replyToEmail,
		endpoint:      "https://api.mailjet.com/v3.1/send",
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		htmlTemplates: htmlTmpl,
		textTemplates: textTmpl,
	}
}

func (m *MailjetMailer) SetEndpoint(endpoint string) {
	m.endpoint = endpoint
}

func (m *MailjetMailer) SetHTTPClient(client *http.Client) {
	m.httpClient = client
}

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

func (m *MailjetMailer) renderHTML(name string, data any) (string, error) {
	if m.htmlTemplates == nil {
		return "", fmt.Errorf("html templates not loaded")
	}
	var buf bytes.Buffer
	if err := m.htmlTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render html template %s: %w", name, err)
	}
	return buf.String(), nil
}

func (m *MailjetMailer) renderText(name string, data any) (string, error) {
	if m.textTemplates == nil {
		return "", fmt.Errorf("text templates not loaded")
	}
	var buf bytes.Buffer
	if err := m.textTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render text template %s: %w", name, err)
	}
	return buf.String(), nil
}

func (m *MailjetMailer) SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error {
	displayName := toName
	if displayName == "" {
		displayName = "User"
	}

	data := map[string]any{
		"Name":         displayName,
		"ResetURL":     resetURL,
		"ReplyToEmail": m.replyToEmail,
	}

	htmlPart, err := m.renderHTML("password_reset.html", data)
	if err != nil {
		return err
	}

	textPart, err := m.renderText("password_reset.tmpl", data)
	if err != nil {
		return err
	}

	subject := "Reset your MotoGuard password"
	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendPasswordChangedEmail(ctx context.Context, toEmail, toName string) error {
	displayName := toName
	if displayName == "" {
		displayName = "User"
	}

	data := map[string]any{
		"Name":         displayName,
		"ReplyToEmail": m.replyToEmail,
	}

	htmlPart, err := m.renderHTML("password_changed.html", data)
	if err != nil {
		return err
	}

	textPart, err := m.renderText("password_changed.tmpl", data)
	if err != nil {
		return err
	}

	subject := "Your MotoGuard password has been changed"
	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendGeofenceBreachAlertEmail(ctx context.Context, toEmail, toName string, data GeofenceAlertData) error {
	displayName := toName
	if displayName == "" {
		displayName = "Fleet Operator"
	}

	htmlPart, err := m.renderHTML("geofence_breach.html", data)
	if err != nil {
		return err
	}

	textPart, err := m.renderText("geofence_breach.tmpl", data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("[ALERT] Geofence Breach: %s - %s", data.BikeRegistration, data.GeofenceName)
	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendBikeTouchedAlertEmail(ctx context.Context, toEmail, toName string, data BikeTouchedAlertData) error {
	displayName := toName
	if displayName == "" {
		displayName = "Fleet Operator"
	}

	htmlPart, err := m.renderHTML("bike_touched.html", data)
	if err != nil {
		return err
	}

	textPart, err := m.renderText("bike_touched.tmpl", data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("[ALERT] Unauthorized Movement / Tamper: %s", data.BikeRegistration)
	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}

func (m *MailjetMailer) SendTrackerAttachedEmail(ctx context.Context, toEmail, toName string, data TrackerAttachedEmailData) error {
	displayName := toName
	if displayName == "" {
		displayName = "Rider"
	}

	htmlPart, err := m.renderHTML("tracker_attached.html", data)
	if err != nil {
		return err
	}

	textPart, err := m.renderText("tracker_attached.tmpl", data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Tracker Attached to %s - MotoGuard", data.BikeRegistration)
	return m.send(ctx, toEmail, displayName, subject, textPart, htmlPart)
}
