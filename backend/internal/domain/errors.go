package domain

import (
	"encoding/json"
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
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

func NewUnauthorizedError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	}
}

func NewBadRequestError(detail string, instance string, errors map[string][]string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/bad-request",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
		Errors:   errors,
	}
}

func NewNotFoundError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/not-found",
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   detail,
		Instance: instance,
	}
}

func NewTooManyRequestsError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/too-many-requests",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   detail,
		Instance: instance,
	}
}

func NewInternalServerError(instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/internal-server-error",
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   "Internal server error",
		Instance: instance,
	}
}

func NewForbiddenError(detail string, instance string) ProblemDetails {
	return ProblemDetails{
		Type:     "https://motoguard.co.ke/errors/forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	}
}
