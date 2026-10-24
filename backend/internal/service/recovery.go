package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"
)

const (
	MaxAttachmentBytes = 5 << 20 // 5 MiB receipt photo or PDF
	MaxContactsPerUser = 50
	maxFutureSkew      = 5 * time.Minute
	maxIDLength        = 32
)

var (
	incidentModes = map[string]bool{
		"violent_robbery":    true,
		"theft_from_parking": true,
		"theft_unattended":   true,
		"hijacking":          true,
		"other":              true,
	}
	contactCategories = map[string]bool{"emergency": true, "insurance": true}
	obNumberPattern   = regexp.MustCompile(`^[A-Za-z0-9/\-\s]{3,30}$`)
	phonePattern      = regexp.MustCompile(`^\+?[0-9\s-]{7,15}$`)
	allowedAttachment = map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/webp":      true,
		"application/pdf": true,
	}
)

type RecoveryService struct {
	repo *repository.RecoveryRepository
}

func NewRecoveryService(repo *repository.RecoveryRepository) *RecoveryService {
	return &RecoveryService{repo: repo}
}

type IncidentListResult struct {
	Data       []domain.TheftIncidentDTO
	Pagination domain.PaginationMeta
}

func (s *RecoveryService) CreateIncident(ctx context.Context, userID string, req domain.CreateIncidentRequest) (domain.TheftIncidentDTO, error) {
	instance := "/v1/incidents"
	errs := map[string][]string{}

	if !incidentModes[req.Mode] {
		errs["mode"] = []string{"Choose one of the listed ways the bike was lost."}
	}

	lostAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.LostAt))
	if err != nil {
		errs["lost_at"] = []string{"Enter the time as an RFC 3339 timestamp."}
	} else if lostAt.After(time.Now().Add(maxFutureSkew)) {
		errs["lost_at"] = []string{"The time lost cannot be in the future."}
	}

	location := strings.TrimSpace(req.LocationLost)
	if location == "" || len(location) > 255 {
		errs["location_lost"] = []string{"Enter where the bike was lost (up to 255 characters)."}
	}

	if (req.Latitude == nil) != (req.Longitude == nil) {
		errs["latitude"] = []string{"Send both latitude and longitude, or neither."}
	} else if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
			errs["latitude"] = []string{"Coordinates are out of range."}
		}
	}

	bikeID := strings.TrimSpace(req.BikeID)
	if bikeID != "" {
		if !isValidID(bikeID) {
			errs["bike_id"] = []string{"Invalid bike id."}
		} else {
			owned, err := s.repo.BikeOwnedBy(ctx, userID, bikeID)
			if err != nil {
				slog.Error("CreateIncident bike ownership check failed", "user_id", userID, "error", err)
				return domain.TheftIncidentDTO{}, domain.NewInternalServerError(instance)
			}
			if !owned {
				errs["bike_id"] = []string{"This bike is not registered to your account."}
			}
		}
	}

	if len(errs) > 0 {
		return domain.TheftIncidentDTO{}, domain.NewBadRequestError("Recovery report is invalid", instance, errs)
	}

	created, err := s.repo.CreateIncident(ctx, domain.TheftIncident{
		ID:           security.NewID("inc_"),
		UserID:       userID,
		BikeID:       bikeID,
		Mode:         req.Mode,
		LostAt:       lostAt,
		LocationLost: location,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
	})
	if err != nil {
		slog.Error("CreateIncident insert failed", "user_id", userID, "error", err)
		return domain.TheftIncidentDTO{}, domain.NewInternalServerError(instance)
	}

	return toIncidentDTO(created), nil
}

func (s *RecoveryService) ListIncidents(ctx context.Context, userID string, limit, offset int) (*IncidentListResult, error) {
	instance := "/v1/incidents"
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	items, err := s.repo.ListIncidents(ctx, userID, limit, offset)
	if err != nil {
		slog.Error("ListIncidents failed", "user_id", userID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}
	total, err := s.repo.CountIncidents(ctx, userID)
	if err != nil {
		slog.Error("CountIncidents failed", "user_id", userID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dtos := make([]domain.TheftIncidentDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, toIncidentDTO(it))
	}

	return &IncidentListResult{
		Data: dtos,
		Pagination: domain.PaginationMeta{
			Total:   total,
			Limit:   limit,
			HasMore: offset+len(items) < total,
		},
	}, nil
}

// SetOB records the OB number from the police receipt and an optional receipt file.
func (s *RecoveryService) SetOB(ctx context.Context, userID, incidentID, obNumber string, file *domain.IncidentAttachment) error {
	instance := "/v1/incidents/" + incidentID + "/ob"

	if !isValidID(incidentID) {
		return domain.NewNotFoundError("Incident not found", instance)
	}

	obNumber = strings.TrimSpace(obNumber)
	if !obNumberPattern.MatchString(obNumber) {
		return domain.NewBadRequestError("OB number is invalid", instance, map[string][]string{
			"ob_number": {"Use 3 to 30 letters, numbers, / or -."},
		})
	}

	var attachment *domain.IncidentAttachment
	if file != nil {
		if len(file.Data) == 0 {
			return domain.NewBadRequestError("Attachment is empty", instance, map[string][]string{
				"attachment": {"The file is empty."},
			})
		}
		if len(file.Data) > MaxAttachmentBytes {
			return domain.NewBadRequestError("Attachment is too large", instance, map[string][]string{
				"attachment": {"Files must be 5 MB or smaller."},
			})
		}
		// Trust the file's bytes over the browser-supplied type.
		sniffed := http.DetectContentType(file.Data[:min(512, len(file.Data))])
		if !allowedAttachment[sniffed] {
			return domain.NewBadRequestError("Attachment type is not allowed", instance, map[string][]string{
				"attachment": {"Upload a JPEG, PNG, WEBP image or a PDF."},
			})
		}
		attachment = &domain.IncidentAttachment{
			FileName:    sanitizeFileName(file.FileName),
			ContentType: sniffed,
			Data:        file.Data,
		}
	}

	if err := s.repo.SetOB(ctx, userID, incidentID, obNumber, attachment); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewNotFoundError("Incident not found", instance)
		}
		slog.Error("SetOB failed", "user_id", userID, "incident_id", incidentID, "error", err)
		return domain.NewInternalServerError(instance)
	}
	return nil
}

func (s *RecoveryService) DeleteIncident(ctx context.Context, userID, incidentID string) error {
	instance := "/v1/incidents/" + incidentID
	if !isValidID(incidentID) {
		return domain.NewNotFoundError("Incident not found", instance)
	}
	if err := s.repo.DeleteIncident(ctx, userID, incidentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewNotFoundError("Incident not found", instance)
		}
		slog.Error("DeleteIncident failed", "user_id", userID, "incident_id", incidentID, "error", err)
		return domain.NewInternalServerError(instance)
	}
	return nil
}

func (s *RecoveryService) CreateContact(ctx context.Context, userID string, req domain.CreateContactRequest) (domain.UserContactDTO, error) {
	instance := "/v1/contacts"
	name, phone, category, errs := validateContact(req)
	if len(errs) > 0 {
		return domain.UserContactDTO{}, domain.NewBadRequestError("Contact is invalid", instance, errs)
	}

	total, err := s.repo.CountContacts(ctx, userID)
	if err != nil {
		slog.Error("CountContacts failed", "user_id", userID, "error", err)
		return domain.UserContactDTO{}, domain.NewInternalServerError(instance)
	}
	if total >= MaxContactsPerUser {
		return domain.UserContactDTO{}, domain.NewBadRequestError("Contact limit reached", instance, map[string][]string{
			"contacts": {"You can save up to 50 contacts. Remove one to add another."},
		})
	}

	created, err := s.repo.CreateContact(ctx, domain.UserContact{
		ID:       security.NewID("con_"),
		UserID:   userID,
		Name:     name,
		Phone:    phone,
		Category: category,
	})
	if err != nil {
		slog.Error("CreateContact insert failed", "user_id", userID, "error", err)
		return domain.UserContactDTO{}, domain.NewInternalServerError(instance)
	}

	return toContactDTO(created), nil
}

func (s *RecoveryService) UpdateContact(ctx context.Context, userID, contactID string, req domain.CreateContactRequest) (domain.UserContactDTO, error) {
	instance := "/v1/contacts/" + contactID
	if !isValidID(contactID) {
		return domain.UserContactDTO{}, domain.NewNotFoundError("Contact not found", instance)
	}

	name, phone, category, errs := validateContact(req)
	if len(errs) > 0 {
		return domain.UserContactDTO{}, domain.NewBadRequestError("Contact is invalid", instance, errs)
	}

	updated, err := s.repo.UpdateContact(ctx, domain.UserContact{
		ID:       contactID,
		UserID:   userID,
		Name:     name,
		Phone:    phone,
		Category: category,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.UserContactDTO{}, domain.NewNotFoundError("Contact not found", instance)
		}
		slog.Error("UpdateContact failed", "user_id", userID, "contact_id", contactID, "error", err)
		return domain.UserContactDTO{}, domain.NewInternalServerError(instance)
	}

	return toContactDTO(updated), nil
}

// validateContact trims the request and returns field errors, if any.
func validateContact(req domain.CreateContactRequest) (name, phone, category string, errs map[string][]string) {
	errs = map[string][]string{}

	name = strings.TrimSpace(req.Name)
	if name == "" || len(name) > 120 {
		errs["name"] = []string{"Enter a name (up to 120 characters)."}
	}

	phone = strings.TrimSpace(req.Phone)
	if !phonePattern.MatchString(phone) {
		errs["phone"] = []string{"Enter a valid phone number, for example 0712 345 678 or +254 712 345 678."}
	}

	category = strings.TrimSpace(req.Category)
	if !contactCategories[category] {
		errs["category"] = []string{"Choose emergency or insurance."}
	}

	return name, phone, category, errs
}

func (s *RecoveryService) ListContacts(ctx context.Context, userID string) ([]domain.UserContactDTO, error) {
	items, err := s.repo.ListContacts(ctx, userID)
	if err != nil {
		slog.Error("ListContacts failed", "user_id", userID, "error", err)
		return nil, domain.NewInternalServerError("/v1/contacts")
	}
	dtos := make([]domain.UserContactDTO, 0, len(items))
	for _, c := range items {
		dtos = append(dtos, toContactDTO(c))
	}
	return dtos, nil
}

func (s *RecoveryService) DeleteContact(ctx context.Context, userID, contactID string) error {
	instance := "/v1/contacts/" + contactID
	if !isValidID(contactID) {
		return domain.NewNotFoundError("Contact not found", instance)
	}
	if err := s.repo.DeleteContact(ctx, userID, contactID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewNotFoundError("Contact not found", instance)
		}
		slog.Error("DeleteContact failed", "user_id", userID, "contact_id", contactID, "error", err)
		return domain.NewInternalServerError(instance)
	}
	return nil
}

func toIncidentDTO(in domain.TheftIncident) domain.TheftIncidentDTO {
	return domain.TheftIncidentDTO{
		ID:             in.ID,
		BikeID:         in.BikeID,
		Mode:           in.Mode,
		LostAt:         in.LostAt,
		LocationLost:   in.LocationLost,
		Latitude:       in.Latitude,
		Longitude:      in.Longitude,
		OBNumber:       in.OBNumber,
		AttachmentName: in.AttachmentName,
		CreatedAt:      in.CreatedAt,
	}
}

func toContactDTO(c domain.UserContact) domain.UserContactDTO {
	return domain.UserContactDTO{
		ID:        c.ID,
		Name:      c.Name,
		Phone:     c.Phone,
		Category:  c.Category,
		CreatedAt: c.CreatedAt,
	}
}

// isValidID accepts the prefixed IDs the app generates (for example "bik_..."), at most 32 characters.
func isValidID(id string) bool {
	return id != "" && len(id) <= maxIDLength
}

// sanitizeFileName keeps only the base name and trims it to the column size.
func sanitizeFileName(name string) string {
	base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if base == "." || base == "/" || base == "" {
		return "receipt"
	}
	if len(base) > 255 {
		return base[len(base)-255:]
	}
	return base
}
