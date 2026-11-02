package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

const (
	maxChatBodyBytes = 4 << 10
	// Each menu tap is one request; 30 per minute leaves room for normal browsing and caps OpenAI spend per rider.
	chatRequestsPerMinute = 30
)

// riderKey limits chat per signed-in user, falling back to client IP.
func riderKey(r *http.Request) string {
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims.UserID != "" {
		return "user:" + claims.UserID
	}
	return "ip:" + middleware.GetClientIP(r)
}

type ChatHandler struct {
	service *service.ChatService
}

func NewChatHandler(s *service.ChatService) *ChatHandler {
	return &ChatHandler{service: s}
}

func (h *ChatHandler) Routes(auth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Group(func(pr chi.Router) {
		pr.Use(auth)
		pr.Use(middleware.RateLimitByKey(riderKey, chatRequestsPerMinute, time.Minute))
		pr.Post("/", h.Respond)
		pr.Get("/history", h.History)
		pr.Delete("/history", h.ClearHistory)
	})
	return r
}

// Respond takes the rider's menu path and returns the next menu or the answer.
func (h *ChatHandler) Respond(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.ChatRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxChatBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request body", r.URL.Path, nil).WriteJSON(w)
		return
	}

	resp, err := h.service.Respond(r.Context(), claims, req.Path, req.Label)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

// History returns the rider's stored chat messages, oldest first. Optional query: limit.
func (h *ChatHandler) History(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	items, err := h.service.History(r.Context(), claims, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// ClearHistory deletes all of the rider's stored chat messages.
func (h *ChatHandler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	if err := h.service.ClearHistory(r.Context(), claims); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
