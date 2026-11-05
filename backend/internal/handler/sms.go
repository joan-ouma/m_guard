package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/service"
)

type SMSHandler struct {
	smsService *service.SMSService
}

func NewSMSHandler(smsService *service.SMSService) *SMSHandler {
	return &SMSHandler{smsService: smsService}
}

func (h *SMSHandler) HandleAfricaTalking(w http.ResponseWriter, r *http.Request) {
	var req domain.SMSIncomingRequest
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON callback payload", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form callback payload", http.StatusBadRequest)
			return
		}
		req = domain.SMSIncomingRequest{
			From:   r.Form.Get("from"),
			To:     r.Form.Get("to"),
			Text:   r.Form.Get("text"),
			Date:   r.Form.Get("date"),
			ID:     r.Form.Get("id"),
			LinkID: r.Form.Get("linkId"),
		}
	}
	if strings.TrimSpace(req.From) == "" {
		http.Error(w, "Missing sender number", http.StatusBadRequest)
		return
	}
	if h.smsService == nil {
		http.Error(w, "SMS service is not configured", http.StatusServiceUnavailable)
		return
	}
	if err := h.smsService.HandleIncoming(r.Context(), req); err != nil {
		slog.Error("Africa's Talking SMS callback failed", "error", err)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "SMS callback canceled", http.StatusRequestTimeout)
			return
		}
		http.Error(w, "Unable to process SMS callback", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
}
