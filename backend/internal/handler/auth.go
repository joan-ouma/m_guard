package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		writeError(w, r, domain.NewBadRequestError("Email and password are required", r.URL.Path, nil))
		return
	}

	ip := middleware.GetClientIP(r)
	sessionData, refreshToken, err := h.service.Login(r.Context(), req.Email, req.Password, ip, r.UserAgent())
	if err != nil {
		writeError(w, r, err)
		return
	}

	setRefreshTokenCookie(w, refreshToken, 7*24*time.Hour)
	writeJSON(w, http.StatusOK, domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	rawRefreshToken := extractRefreshToken(r)
	ip := middleware.GetClientIP(r)

	sessionData, newRefreshToken, err := h.service.RefreshToken(r.Context(), rawRefreshToken, ip, r.UserAgent())
	if err != nil {
		writeError(w, r, err)
		return
	}

	setRefreshTokenCookie(w, newRefreshToken, 7*24*time.Hour)
	writeJSON(w, http.StatusOK, domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	rawRefreshToken := extractRefreshToken(r)
	_ = h.service.Logout(r.Context(), rawRefreshToken)

	setRefreshTokenCookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Unauthorized", r.URL.Path))
		return
	}

	user, err := h.service.GetCurrentSession(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user": *user,
		},
	})
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req domain.PasswordResetRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil))
		return
	}

	email := strings.TrimSpace(req.Email)
	if email != "" {
		_ = h.service.RequestPasswordReset(r.Context(), email)
	}

	writeJSON(w, http.StatusAccepted, domain.MessageResponse{
		Message: "If the email is registered, password reset instructions have been sent.",
	})
}

func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		token = strings.TrimPrefix(r.URL.Path, "/v1/password-resets/")
		token = strings.Trim(token, "/")
	}
	if unescaped, err := url.PathUnescape(token); err == nil {
		token = unescaped
	}
	token = strings.TrimSpace(token)

	if token == "" {
		writeError(w, r, domain.NewBadRequestError("Missing reset token in path", r.URL.Path, nil))
		return
	}

	var req domain.PasswordResetConfirmDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil))
		return
	}

	if err := h.service.ConfirmPasswordReset(r.Context(), token, req.Password); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, domain.MessageResponse{
		Message: "Password has been successfully updated. All active sessions have been revoked.",
	})
}

func extractRefreshToken(r *http.Request) string {
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.RefreshToken
}

func setRefreshTokenCookie(w http.ResponseWriter, token string, maxAge time.Duration) {
	expires := time.Now().Add(maxAge)
	age := int(maxAge.Seconds())
	if maxAge < 0 {
		expires = time.Unix(0, 0)
		age = -1
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/v1/sessions",
		Expires:  expires,
		MaxAge:   age,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
}
