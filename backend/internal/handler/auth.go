package handler

import (
	"encoding/json"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
			pd.WriteJSON(w)
			return
		}
		domain.NewInternalServerError(r.URL.Path).WriteJSON(w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/v1/sessions",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/v1/sessions",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(domain.AuthResponse{Data: *sessionData})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rawRefreshToken string
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		rawRefreshToken = cookie.Value
	}

	_ = h.service.Logout(r.Context(), rawRefreshToken)

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/v1/sessions",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
