package domain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProblemDetails_WriteJSON(t *testing.T) {
	prob := NewBadRequestError("invalid email", "/v1/sessions", map[string][]string{
		"email": {"Email must be valid"},
	})

	rr := httptest.NewRecorder()
	prob.WriteJSON(rr)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected Content-Type application/problem+json, got %s", ct)
	}

	var decoded ProblemDetails
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if decoded.Detail != "invalid email" {
		t.Fatalf("expected detail 'invalid email', got %s", decoded.Detail)
	}
	if decoded.Errors["email"][0] != "Email must be valid" {
		t.Fatalf("unexpected field error value: %v", decoded.Errors)
	}
}

func TestProblemDetails_Constructors(t *testing.T) {
	tests := []struct {
		name       string
		err        ProblemDetails
		wantStatus int
		wantTitle  string
	}{
		{
			name:       "Unauthorized",
			err:        NewUnauthorizedError("bad token", "/path"),
			wantStatus: http.StatusUnauthorized,
			wantTitle:  "Unauthorized",
		},
		{
			name:       "NotFound",
			err:        NewNotFoundError("not found", "/path"),
			wantStatus: http.StatusNotFound,
			wantTitle:  "Not Found",
		},
		{
			name:       "TooManyRequests",
			err:        NewTooManyRequestsError("rate limited", "/path"),
			wantStatus: http.StatusTooManyRequests,
			wantTitle:  "Too Many Requests",
		},
		{
			name:       "InternalServerError",
			err:        NewInternalServerError("/path"),
			wantStatus: http.StatusInternalServerError,
			wantTitle:  "Internal Server Error",
		},
		{
			name:       "Forbidden",
			err:        NewForbiddenError("forbidden", "/path"),
			wantStatus: http.StatusForbidden,
			wantTitle:  "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.wantStatus {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.wantStatus, tt.err.Status)
			}
			if tt.err.Title != tt.wantTitle {
				t.Errorf("%s: expected title %s, got %s", tt.name, tt.wantTitle, tt.err.Title)
			}
			if tt.err.Error() != tt.err.Detail {
				t.Errorf("%s: expected Error() to match Detail", tt.name)
			}
		})
	}
}
