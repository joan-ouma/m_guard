package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

const maxJSONBodyBytes = 64 << 10 // 64 KiB is plenty for a report or a contact

type RecoveryHandler struct {
	service *service.RecoveryService
}

func NewRecoveryHandler(s *service.RecoveryService) *RecoveryHandler {
	return &RecoveryHandler{service: s}
}

// IncidentRoutes is mounted at /v1/incidents.
func (h *RecoveryHandler) IncidentRoutes(auth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(auth)

	r.Get("/", h.ListIncidents)
	r.Post("/", h.CreateIncident)
	r.Delete("/{id}", h.DeleteIncident)
	r.Put("/{id}/ob", h.SetOB)
	r.Post("/{id}/resolve", h.ResolveIncident)

	return r
}

// ContactRoutes is mounted at /v1/contacts.
func (h *RecoveryHandler) ContactRoutes(auth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(auth)

	r.Get("/", h.ListContacts)
	r.Post("/", h.CreateContact)
	r.Put("/{id}", h.UpdateContact)
	r.Delete("/{id}", h.DeleteContact)

	return r
}

func (h *RecoveryHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	limit := parseLimit(r.URL.Query(), 25)
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	res, err := h.service.ListIncidents(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":       res.Data,
		"pagination": res.Pagination,
	})
}

func (h *RecoveryHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.CreateIncidentRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	dto, err := h.service.CreateIncident(r.Context(), claims.UserID, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": dto})
}

// SetOB accepts multipart/form-data with an ob_number field and an optional attachment file.
func (h *RecoveryHandler) SetOB(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	// Allow headroom for form fields on top of the file limit.
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAttachmentBytes+(64<<10))
	if err := r.ParseMultipartForm(service.MaxAttachmentBytes); err != nil {
		domain.NewBadRequestError("Upload is too large or malformed", r.URL.Path, map[string][]string{
			"attachment": {"Files must be 5 MB or smaller."},
		}).WriteJSON(w)
		return
	}

	var file *domain.IncidentAttachment
	upload, header, err := r.FormFile("attachment")
	switch {
	case err == nil:
		defer upload.Close()
		data, readErr := io.ReadAll(io.LimitReader(upload, service.MaxAttachmentBytes+1))
		if readErr != nil {
			domain.NewBadRequestError("Could not read the attachment", r.URL.Path, nil).WriteJSON(w)
			return
		}
		file = &domain.IncidentAttachment{FileName: header.Filename, Data: data}
	case err == http.ErrMissingFile:
		// Attachment is optional.
	default:
		domain.NewBadRequestError("Could not read the attachment", r.URL.Path, nil).WriteJSON(w)
		return
	}

	if err := h.service.SetOB(r.Context(), claims.UserID, chi.URLParam(r, "id"), r.FormValue("ob_number"), file); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "saved"}})
}

func (h *RecoveryHandler) ResolveIncident(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	if err := h.service.ResolveIncident(r.Context(), claims.UserID, chi.URLParam(r, "id")); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RecoveryHandler) DeleteIncident(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	if err := h.service.DeleteIncident(r.Context(), claims.UserID, chi.URLParam(r, "id")); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RecoveryHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	dtos, err := h.service.ListContacts(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": dtos})
}

func (h *RecoveryHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.CreateContactRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	dto, err := h.service.CreateContact(r.Context(), claims.UserID, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": dto})
}

func (h *RecoveryHandler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	var req domain.CreateContactRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	dto, err := h.service.UpdateContact(r.Context(), claims.UserID, chi.URLParam(r, "id"), req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": dto})
}

func (h *RecoveryHandler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFromRequest(r)
	if !ok {
		domain.NewUnauthorizedError("Authentication required", r.URL.Path).WriteJSON(w)
		return
	}

	if err := h.service.DeleteContact(r.Context(), claims.UserID, chi.URLParam(r, "id")); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func claimsFromRequest(r *http.Request) (*security.JWTClaims, bool) {
	claims, ok := r.Context().Value(middleware.UserClaimsKey).(*security.JWTClaims)
	if !ok || claims == nil || claims.UserID == "" {
		return nil, false
	}
	return claims, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		domain.NewBadRequestError("Invalid request body", r.URL.Path, nil).WriteJSON(w)
		return false
	}
	return true
}
