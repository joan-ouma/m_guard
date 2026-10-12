package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Name = strings.TrimSpace(req.Name)
	req.NationalID = strings.TrimSpace(req.NationalID)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		domain.NewBadRequestError("Email, name, and password are required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if len(req.Password) < 8 {
		domain.NewBadRequestError("Password must be at least 8 characters", r.URL.Path, nil).WriteJSON(w)
		return
	}

	userDto, err := h.service.Register(r.Context(), req)
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": userDto,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		domain.NewBadRequestError("Email and password are required", r.URL.Path, nil).WriteJSON(w)
		return
	}

	ip := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	sessionData, refreshToken, err := h.service.Login(r.Context(), req.Email, req.Password, ip, userAgent)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			if pd.Status >= 500 {
				slog.Error("Login 500 internal error", "email", req.Email, "detail", pd.Detail, "instance", pd.Instance)
			} else {
				slog.Warn("Login client error", "email", req.Email, "status", pd.Status, "detail", pd.Detail)
			}
			pd.WriteJSON(w)
			return
		}
		slog.Error("Login unhandled internal error", "email", req.Email, "error", err)
		domain.NewInternalServerErrorWithDetail(err.Error(), r.URL.Path).WriteJSON(w)
		return
	}

	setSessionCookie(w, r, refreshToken, time.Now().Add(7*24*time.Hour), int(7*24*time.Hour.Seconds()))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	var rawRefreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawRefreshToken = cookie.Value
	}

	if rawRefreshToken == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		rawRefreshToken = body.RefreshToken
	}

	ip := middleware.GetClientIP(r)
	userAgent := r.UserAgent()

	sessionData, newRefreshToken, err := h.service.RefreshToken(r.Context(), rawRefreshToken, ip, userAgent)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	setSessionCookie(w, r, newRefreshToken, time.Now().Add(7*24*time.Hour), int(7*24*time.Hour.Seconds()))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	var rawRefreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawRefreshToken = cookie.Value
	}

	_ = h.service.Logout(r.Context(), rawRefreshToken)

	setSessionCookie(w, r, "", time.Unix(0, 0), -1)

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Unauthorized", r.URL.Path).WriteJSON(w)
		return
	}

	user, err := h.service.GetCurrentSession(r.Context(), claims.UserID)
	if err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	response := struct {
		Data struct {
			User domain.UserDTO `json:"user"`
		} `json:"data"`
	}{
		Data: struct {
			User domain.UserDTO `json:"user"`
		}{
			User: *user,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.PasswordResetRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" {
		_ = h.service.RequestPasswordReset(r.Context(), req.Email)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(domain.MessageResponse{
		Message: "If the email is registered, password reset instructions have been sent.",
	})
}

func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		domain.NewMethodNotAllowedError("Method not allowed", r.URL.Path).WriteJSON(w)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/v1/password-resets/")
	token = strings.TrimSpace(token)
	if token == "" {
		domain.NewBadRequestError("Missing reset token in path", r.URL.Path, nil).WriteJSON(w)
		return
	}

	var req domain.PasswordResetConfirmDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if err := h.service.ConfirmPasswordReset(r.Context(), token, req.Password); err != nil {
		if pd, ok := err.(domain.ProblemDetails); ok {
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(domain.MessageResponse{
		Message: "Password has been successfully updated. All active sessions have been revoked.",
	})
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expires time.Time, maxAge int) {
	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/v1/sessions",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
	}
	if isHTTPS {
		cookie.Secure = true
		cookie.SameSite = http.SameSiteNoneMode
	} else {
		cookie.Secure = false
		cookie.SameSite = http.SameSiteLaxMode
	}
	http.SetCookie(w, cookie)
}
