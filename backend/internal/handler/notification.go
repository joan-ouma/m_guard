package handler

import (
	"encoding/json"
	"net/http"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/notification"

	"github.com/go-chi/chi/v5"
)

type NotificationHandler struct {
	service *notification.Service
}

func NewNotificationHandler(service *notification.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/sns-endpoint", h.RegisterDevice)
	r.Delete("/sns-endpoint", h.UnregisterDevice)
	r.Post("/test", h.SendTestNotification)

	return r
}

func (h *NotificationHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		domain.NewUnauthorizedError("Unauthorized", r.URL.Path).WriteJSON(w)
		return
	}

	var req notification.RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request body", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if req.Token == "" {
		domain.NewBadRequestError("token is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	endpointARN, err := h.service.RegisterDevice(r.Context(), notification.RegisterDeviceParams{
		UserID:      claims.UserID,
		Token:       req.Token,
		Platform:    req.Platform,
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		domain.NewInternalServerErrorWithDetail("Failed to register device endpoint: "+err.Error(), r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": notification.RegisterDeviceResponse{
			EndpointARN: endpointARN,
			Message:     "Device registered successfully for SNS push alerts",
		},
	})
}

func (h *NotificationHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	endpointARN := r.URL.Query().Get("endpoint_arn")
	if endpointARN == "" {
		domain.NewBadRequestError("endpoint_arn query param required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	_ = h.service.UnregisterDevice(r.Context(), endpointARN)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Device endpoint unregistered",
	})
}

func (h *NotificationHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		domain.NewUnauthorizedError("Unauthorized", r.URL.Path).WriteJSON(w)
		return
	}

	var req notification.TestNotificationRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	err := h.service.SendTestNotification(r.Context(), notification.SendTestAlertParams{
		UserID:      claims.UserID,
		Title:       req.Title,
		Body:        req.Body,
		PhoneNumber: req.PhoneNumber,
		BikeID:      req.BikeID,
	})
	if err != nil {
		domain.NewInternalServerErrorWithDetail("Failed to send test alert: "+err.Error(), r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Test push and SMS alert dispatched via AWS SNS",
	})
}
