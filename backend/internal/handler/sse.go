package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"
	"motoguard-backend/internal/sse"

	"github.com/google/uuid"
)

type SSEHandler struct {
	broker      *sse.Broker
	bikeService *service.BikeService
}

func NewSSEHandler(broker *sse.Broker, bikeService *service.BikeService) *SSEHandler {
	return &SSEHandler{
		broker:      broker,
		bikeService: bikeService,
	}
}

func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Disable write deadline for SSE stream in Go
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := sse.NewClient(uuid.NewString(), claims.UserID, 128)

	h.broker.Register(client)
	defer h.broker.Unregister(client)

	// Send initial snapshot of bikes immediately upon connection
	if bikesList, err := h.bikeService.ListBikes(r.Context(), claims, "", 100); err == nil {
		if dataBytes, err := json.Marshal(bikesList); err == nil {
			fmt.Fprintf(w, "event: bikes_init\ndata: %s\n\n", dataBytes)
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ctx := r.Context()
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-client.Send:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
