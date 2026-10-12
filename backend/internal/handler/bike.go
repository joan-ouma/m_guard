package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"
)

type BikeHandler struct {
	service *service.BikeService
}

func NewBikeHandler(s *service.BikeService) *BikeHandler {
	return &BikeHandler{service: s}
}

func (h *BikeHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BikeHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
	subPath := strings.TrimPrefix(r.URL.Path, "/v1/bikes/")
	subPath = strings.TrimSpace(subPath)
	if subPath == "" {
		domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if strings.HasSuffix(subPath, "/breadcrumbs") {
		bikeID := strings.TrimSuffix(subPath, "/breadcrumbs")
		bikeID = strings.Trim(bikeID, "/")
		if r.Method == http.MethodGet {
			h.GetBreadcrumbs(w, r, bikeID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if strings.HasSuffix(subPath, "/events") {
		bikeID := strings.TrimSuffix(subPath, "/events")
		bikeID = strings.Trim(bikeID, "/")
		if r.Method == http.MethodGet {
			h.GetEvents(w, r, bikeID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if strings.HasSuffix(subPath, "/immobilize") {
		bikeID := strings.TrimSuffix(subPath, "/immobilize")
		bikeID = strings.Trim(bikeID, "/")
		if r.Method == http.MethodPost {
			h.Immobilize(w, r, bikeID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bikeID := subPath
	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r, bikeID)
	case http.MethodPut, http.MethodPatch:
		h.Update(w, r, bikeID)
	case http.MethodDelete:
		h.Delete(w, r, bikeID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BikeHandler) GetBreadcrumbs(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	limit := 1000
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

	breadcrumbs, err := h.service.GetBreadcrumbs(r.Context(), claims, bikeID, startDate, endDate, limit)
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
	_ = json.NewEncoder(w).Encode(map[string]any{"data": breadcrumbs})
}

func (h *BikeHandler) GetEvents(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	limit := 1000
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

	events, err := h.service.GetEvents(r.Context(), claims, bikeID, startDate, endDate, limit)
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
	_ = json.NewEncoder(w).Encode(map[string]any{"data": events})
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

func (h *BikeHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	resp, err := h.service.ListBikes(r.Context(), claims)
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

func (h *BikeHandler) GetByID(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
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

func (h *BikeHandler) Update(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
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

func (h *BikeHandler) Delete(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
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

func (h *BikeHandler) Immobilize(w http.ResponseWriter, r *http.Request, bikeID string) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
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
