package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"motoguard-backend/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if pd, ok := err.(domain.ProblemDetails); ok {
		pd.WriteJSON(w)
		return
	}
	domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
}

func parseLimit(q url.Values, defaultLimit int) int {
	raw := q.Get("limit")
	if raw == "" {
		return defaultLimit
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultLimit
	}
	return parsed
}

func parseDateRange(q url.Values) (string, string) {
	dateParam := strings.TrimSpace(q.Get("date"))
	if dateParam != "" {
		return dateParam + "T00:00:00Z", dateParam + "T23:59:59Z"
	}

	startDate := strings.TrimSpace(q.Get("startDate"))
	if startDate != "" && !strings.Contains(startDate, "T") {
		startDate += "T00:00:00Z"
	}

	endDate := strings.TrimSpace(q.Get("endDate"))
	if endDate != "" && !strings.Contains(endDate, "T") {
		endDate += "T23:59:59Z"
	}

	return startDate, endDate
}

func getBikeID(r *http.Request) string {
	if id := r.PathValue("id"); id != "" {
		return id
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range parts {
		if part == "bikes" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
