package domain

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ProblemDetails struct {
	Type     string              `json:"type,omitempty"`
	Title    string              `json:"title"`
	Status   int                 `json:"status"`
	Detail   string              `json:"detail,omitempty"`
	Instance string              `json:"instance,omitempty"`
	Errors   map[string][]string `json:"errors,omitempty"`
}

func (p ProblemDetails) Error() string {
	return p.Detail
}

func (p ProblemDetails) WriteJSON(w http.ResponseWriter) {
	if p.Status >= 500 {
		slog.Error("ProblemDetails internal error", "status", p.Status, "title", p.Title, "detail", p.Detail, "instance", p.Instance)
		p.Detail = "Internal server error"
	} else if p.Status >= 400 {
		slog.Warn("ProblemDetails client error", "status", p.Status, "title", p.Title, "detail", p.Detail, "instance", p.Instance)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func NewMethodNotAllowedError(detail string, instance string) ProblemDetails {
	if detail == "" {
		detail = "Method not allowed"
	}
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Method Not Allowed",
		Status:   http.StatusMethodNotAllowed,
		Detail:   detail,
		Instance: instance,
	}
}

func NewUnauthorizedError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	}
}

func NewBadRequestError(detail string, instance string, errors map[string][]string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
		Errors:   errors,
	}
}

func NewNotFoundError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	}
}

func NewTooManyRequestsError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   detail,
		Instance: instance,
	}
}

func NewInternalServerError(instance string) ProblemDetails {
	return NewInternalServerErrorWithDetail("Internal server error", instance)
}

func NewInternalServerErrorWithDetail(detail, instance string) ProblemDetails {
	if detail == "" {
		detail = "Internal server error"
	}
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: instance,
	}
}

func NewForbiddenError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	}
}

func NewConflictError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "about:blank",
		Title:    "Conflict",
		Status:   http.StatusConflict,
		Detail:   detail,
		Instance: instance,
	}
}

