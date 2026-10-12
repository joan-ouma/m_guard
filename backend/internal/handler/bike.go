package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/service"
)

type BikeHandler struct {
	service *service.BikeService
}

func NewBikeHandler(s *service.BikeService) *BikeHandler {
	return &BikeHandler{service: s}
}

func getBikeID(r *http.Request) string {
	id := r.PathValue("id")
	if id != "" {
		return id
	}
	// Fallback for manual subpath invocation
	subPath := strings.TrimPrefix(r.URL.Path, "/v1/bikes/")
	for _, suffix := range []string{"/breadcrumbs", "/events", "/immobilize"} {
		subPath = strings.TrimSuffix(subPath, suffix)
	}
	return strings.Trim(subPath, "/")
}

func (h *BikeHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	resp, err := h.service.ListBikes(r.Context(), claims)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *BikeHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	var req domain.CreateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request body", r.URL.Path, nil))
		return
	}

	dto, err := h.service.CreateBike(r.Context(), claims, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": dto})
}

func (h *BikeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	dto, err := h.service.GetBike(r.Context(), claims, bikeID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": dto})
}

func (h *BikeHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	var req domain.UpdateBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request body", r.URL.Path, nil))
		return
	}

	dto, err := h.service.UpdateBike(r.Context(), claims, bikeID, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": dto})
}

func (h *BikeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	if err := h.service.DeleteBike(r.Context(), claims, bikeID); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BikeHandler) GetBreadcrumbs(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	query := r.URL.Query()
	limit := parseLimit(query, 1000)
	startDate, endDate := parseDateRange(query)

	breadcrumbs, err := h.service.GetBreadcrumbs(r.Context(), claims, bikeID, startDate, endDate, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": breadcrumbs})
}

func (h *BikeHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	query := r.URL.Query()
	limit := parseLimit(query, 1000)
	startDate, endDate := parseDateRange(query)

	events, err := h.service.GetEvents(r.Context(), claims, bikeID, startDate, endDate, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": events})
}

func (h *BikeHandler) Immobilize(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Authentication required", r.URL.Path))
		return
	}

	bikeID := getBikeID(r)
	if bikeID == "" {
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	var req domain.ImmobilizeBikeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	immobilize := true
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action == "REACTIVATE" || action == "MOBILIZE" || action == "FALSE" {
		immobilize = false
	}

	bike, err := h.service.ImmobilizeBike(r.Context(), claims, bikeID, immobilize)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": bike})
}

// Backward-compatible collection & item dispatchers
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
		writeError(w, r, domain.NewBadRequestError("Motorcycle ID is required", r.URL.Path, nil))
		return
	}

	if strings.HasSuffix(subPath, "/breadcrumbs") {
		h.GetBreadcrumbs(w, r)
		return
	}
	if strings.HasSuffix(subPath, "/events") {
		h.GetEvents(w, r)
		return
	}
	if strings.HasSuffix(subPath, "/immobilize") {
		h.Immobilize(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r)
	case http.MethodPut, http.MethodPatch:
		h.Update(w, r)
	case http.MethodDelete:
		h.Delete(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
