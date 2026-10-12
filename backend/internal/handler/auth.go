package handler

import (
	"encoding/json"
	"net/http"
<<<<<<< HEAD
	"net/url"
=======
>>>>>>> origin/main
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
<<<<<<< HEAD
=======
	"motoguard-backend/internal/security"
>>>>>>> origin/main
	"motoguard-backend/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

<<<<<<< HEAD
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil))
=======
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
>>>>>>> origin/main
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
<<<<<<< HEAD
	if req.Email == "" || req.Password == "" {
		writeError(w, r, domain.NewBadRequestError("Email and password are required", r.URL.Path, nil))
=======
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
>>>>>>> origin/main
		return
	}

	ip := middleware.GetClientIP(r)
<<<<<<< HEAD
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
=======
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

>>>>>>> origin/main
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
<<<<<<< HEAD
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		writeError(w, r, domain.NewUnauthorizedError("Unauthorized", r.URL.Path))
=======
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil {
		domain.NewUnauthorizedError("Unauthorized", r.URL.Path).WriteJSON(w)
>>>>>>> origin/main
		return
	}

	user, err := h.service.GetCurrentSession(r.Context(), claims.UserID)
	if err != nil {
<<<<<<< HEAD
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
=======
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
>>>>>>> origin/main
		Message: "If the email is registered, password reset instructions have been sent.",
	})
}

func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
<<<<<<< HEAD
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
=======
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/v1/password-resets/")
	token = strings.TrimSpace(token)
	if token == "" {
		domain.NewBadRequestError("Missing reset token in path", r.URL.Path, nil).WriteJSON(w)
>>>>>>> origin/main
		return
	}

	var req domain.PasswordResetConfirmDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
<<<<<<< HEAD
		writeError(w, r, domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil))
=======
		domain.NewBadRequestError("Invalid request payload", r.URL.Path, nil).WriteJSON(w)
>>>>>>> origin/main
		return
	}

	if err := h.service.ConfirmPasswordReset(r.Context(), token, req.Password); err != nil {
<<<<<<< HEAD
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
=======
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
>>>>>>> origin/main
