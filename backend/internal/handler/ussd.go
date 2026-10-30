package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type USSDHandler struct {
	ussdService *service.USSDService
}

func NewUSSDHandler(ussdService *service.USSDService) *USSDHandler {
	return &USSDHandler{
		ussdService: ussdService,
	}
}

func (h *USSDHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.HandleUSSD)
	r.Get("/", h.HandleUSSD) // Support GET test pings

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
