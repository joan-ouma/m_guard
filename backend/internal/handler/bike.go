package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type BikeHandler struct {
	service       *service.BikeService
	verifyLimiter *middleware.TrackerRateLimiter
}

func NewBikeHandler(s *service.BikeService) *BikeHandler {
	return &BikeHandler{
		service:       s,
		verifyLimiter: middleware.NewTrackerRateLimiter(1*time.Minute, 5, 24*time.Hour),
	}
}

func (h *BikeHandler) Routes(auth func(http.Handler) http.Handler, sseStream http.HandlerFunc) chi.Router {
	r := chi.NewRouter()

	// Public tracker verification (rate-limited per IP) so unauthenticated users during signup can verify trackers
	r.Get("/verify-tracker", h.VerifyTracker)
	r.Post("/verify-tracker", h.VerifyTracker)

	// Protected routes below
	r.Group(func(pr chi.Router) {
		pr.Use(auth)
		pr.Get("/stream", sseStream)

		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/attach", h.Attach)

		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Patch("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/breadcrumbs", h.GetBreadcrumbs)
		r.Get("/{id}/events", h.GetEvents)
		r.Post("/{id}/immobilize", h.Immobilize)
	})

	return r
}

func (h *BikeHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	resp, err := h.service.ListBikes(r.Context(), claims, cursor, limit)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BikeHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.CreateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request body", r.URL.Path, nil).WriteJSON(w)
		return
	}

	dto, err := h.service.CreateBike(r.Context(), claims, req)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": dto})
}

func (h *BikeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	dto, err := h.service.GetBike(r.Context(), claims, bikeID)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": dto})
}

func (h *BikeHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	var req domain.UpdateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request body", r.URL.Path, nil).WriteJSON(w)
		return
	}

	dto, err := h.service.UpdateBike(r.Context(), claims, bikeID, req)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": dto})
}

func (h *BikeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if err := h.service.DeleteBike(r.Context(), claims, bikeID); err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BikeHandler) GetBreadcrumbs(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	var cursor int64
	if c := r.URL.Query().Get("cursor"); c != "" {
		if parsed, err := strconv.ParseInt(c, 10, 64); err == nil && parsed > 0 {
			cursor = parsed
		}
	}

	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	startDate := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDate := strings.TrimSpace(r.URL.Query().Get("endDate"))
	if dateParam := strings.TrimSpace(r.URL.Query().Get("date")); dateParam != "" {
		startDate = dateParam + "T00:00:00Z"
		endDate = dateParam + "T23:59:59Z"
	} else {
		if startDate != "" && !strings.Contains(startDate, "T") {
			startDate = startDate + "T00:00:00Z"
		}
		if endDate != "" && !strings.Contains(endDate, "T") {
			endDate = endDate + "T23:59:59Z"
		}
	}

	resp, err := h.service.GetBreadcrumbs(r.Context(), claims, bikeID, startDate, endDate, cursor, limit)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BikeHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	var cursor int64
	if c := r.URL.Query().Get("cursor"); c != "" {
		if parsed, err := strconv.ParseInt(c, 10, 64); err == nil && parsed > 0 {
			cursor = parsed
		}
	}

	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	startDate := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDate := strings.TrimSpace(r.URL.Query().Get("endDate"))
	if dateParam := strings.TrimSpace(r.URL.Query().Get("date")); dateParam != "" {
		startDate = dateParam + "T00:00:00Z"
		endDate = dateParam + "T23:59:59Z"
	} else {
		if startDate != "" && !strings.Contains(startDate, "T") {
			startDate = startDate + "T00:00:00Z"
		}
		if endDate != "" && !strings.Contains(endDate, "T") {
			endDate = endDate + "T23:59:59Z"
		}
	}

	resp, err := h.service.GetEvents(r.Context(), claims, bikeID, startDate, endDate, cursor, limit)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BikeHandler) Immobilize(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	var req domain.ImmobilizeBikeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	immobilize := true
	if strings.EqualFold(req.Action, "REACTIVATE") || strings.EqualFold(req.Action, "MOBILIZE") || strings.EqualFold(req.Action, "FALSE") {
		immobilize = false
	}

	bike, err := h.service.ImmobilizeBike(r.Context(), claims, bikeID, immobilize)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": bike})
}

func (h *BikeHandler) VerifyTracker(w http.ResponseWriter, r *http.Request) {
	ip := middleware.GetClientIP(r)

	// Check rate limit per IP
	allowed, retryAfter, lockedOut := h.verifyLimiter.CheckAllowed(ip)
	if !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		w.Header().Set("X-RateLimit-Locked", strconv.FormatBool(lockedOut))

		msg := fmt.Sprintf("Rate limited. Please wait %d seconds before attempting another tracker lookup.", retryAfter)
		if lockedOut {
			msg = "Excessive verification failures. Your IP address has been temporarily locked out from manual tracker lookups."
		}
		domain.NewTooManyRequestsError(msg, r.URL.Path).WriteJSON(w)
		return
	}

	var imei string
	if r.Method == http.MethodGet {
		imei = strings.TrimSpace(r.URL.Query().Get("imei"))
	} else {
		var req domain.VerifyTrackerRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		imei = strings.TrimSpace(req.IMEI)
		if imei == "" {
			imei = strings.TrimSpace(r.URL.Query().Get("imei"))
		}
	}

	if imei == "" {
		retryAfter, lockedOut := h.verifyLimiter.RecordFailure(ip)
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		w.Header().Set("X-RateLimit-Locked", strconv.FormatBool(lockedOut))
		domain.NewBadRequestError("Tracker IMEI code is required", r.URL.Path, map[string][]string{
			"imei": {"Please provide a valid 15-digit IMEI code"},
		}).WriteJSON(w)
		return
	}

	dto, err := h.service.VerifyTracker(r.Context(), imei)
	if err != nil {
		retryAfter, lockedOut := h.verifyLimiter.RecordFailure(ip)
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		w.Header().Set("X-RateLimit-Locked", strconv.FormatBool(lockedOut))
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	// Success: record success and enforce baseline 1 request per min
	h.verifyLimiter.RecordSuccess(ip)
	w.Header().Set("Retry-After", "60")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": dto})
}

func (h *BikeHandler) Attach(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.AttachBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
		return
	}

	dto, err := h.service.AttachBike(r.Context(), claims, req)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":    dto,
		"message": "Motorcycle attached to tracker successfully",
	})
}
