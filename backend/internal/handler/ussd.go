package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"motoguard-backend/internal/africastalking"
	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type USSDHandler struct {
	ussdService *service.USSDService
	atClient    africastalking.Client
}

func NewUSSDHandler(ussdService *service.USSDService, atClient africastalking.Client) *USSDHandler {
	return &USSDHandler{
		ussdService: ussdService,
		atClient:    atClient,
	}
}

func (h *USSDHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.HandleUSSD)
	r.Post("/callback", h.HandleUSSD)
	r.Get("/callback", h.HandleUSSD) // Support GET test pings

	return r
}

func (h *USSDHandler) SMSRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/incoming", h.HandleIncomingSMS)
	r.Post("/callback", h.HandleIncomingSMS)
	r.Get("/outbox", h.GetOutboxSMS)

	return r
}

// HandleUSSD processes Africa's Talking USSD callback.
func (h *USSDHandler) HandleUSSD(w http.ResponseWriter, r *http.Request) {
	var req domain.USSDRequest

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	_ = r.ParseForm()

	if req.PhoneNumber == "" {
		req.PhoneNumber = getParam(r, "phoneNumber", "phone_number", "phone")
	}
	if req.SessionID == "" {
		req.SessionID = getParam(r, "sessionId", "session_id", "sessionId")
	}
	if req.ServiceCode == "" {
		req.ServiceCode = getParam(r, "serviceCode", "service_code")
	}
	if req.Text == "" {
		req.Text = getParam(r, "text", "Text")
	}

	if req.PhoneNumber == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("END Phone number is required for USSD session."))
		return
	}

	respText := h.ussdService.HandleUSSD(r.Context(), req)

	// Africa's Talking expects plain text beginning with "CON " or "END "
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(respText))
}

// HandleIncomingSMS processes 2-Way SMS webhooks from Africa's Talking.
func (h *USSDHandler) HandleIncomingSMS(w http.ResponseWriter, r *http.Request) {
	var req domain.SMSIncomingRequest

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		_ = r.ParseForm()
		req.From = r.FormValue("from")
		req.To = r.FormValue("to")
		req.Text = r.FormValue("text")
		req.Date = r.FormValue("date")
		req.ID = r.FormValue("id")
		req.LinkID = r.FormValue("linkId")
	}

	if req.From == "" {
		req.From = r.URL.Query().Get("from")
		req.Text = r.URL.Query().Get("text")
	}

	if req.From == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Sender 'from' number is required"})
		return
	}

	replyMessage := h.ussdService.HandleIncomingSMS(r.Context(), req)

	// Respond with JSON or plain text
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "received",
		"reply":   replyMessage,
		"to":      req.From,
		"message": replyMessage,
	})
}

// GetOutboxSMS returns recent outbound SMS log for simulator/testing UI.
func (h *USSDHandler) GetOutboxSMS(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs := h.atClient.GetRecentOutboundSMS(limit)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":  logs,
		"total": len(logs),
	})
}

// SimulatorPage serves the local interactive Africa's Talking Sandbox Simulator.
func (h *USSDHandler) SimulatorPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(simulatorHTML))
}

func getParam(r *http.Request, keys ...string) string {
	for _, k := range keys {
		if v := r.FormValue(k); v != "" {
			return strings.TrimSpace(v)
		}
		if v := r.URL.Query().Get(k); v != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
