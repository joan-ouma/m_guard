package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandPublisher interface {
	PublishCommand(imei string, action string) error
}

type BikeService struct {
	pool   *pgxpool.Pool
	repo   *repository.BikeRepository
	cmdPub CommandPublisher
}

func NewBikeService(pool *pgxpool.Pool, repo *repository.BikeRepository) *BikeService {
	return &BikeService{
		pool: pool,
		repo: repo,
	}
}

func (s *BikeService) SetCommandPublisher(pub CommandPublisher) {
	s.cmdPub = pub
}

func (s *BikeService) CreateBike(ctx context.Context, claims *security.JWTClaims, req domain.CreateBikeRequest) (*domain.BikeDTO, error) {
	instance := "/v1/bikes"

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	regNumber := strings.TrimSpace(req.RegistrationNumber)
	imei := strings.TrimSpace(req.IMEI)

	validationErrors := make(map[string][]string)
	if regNumber == "" {
		validationErrors["registration_number"] = []string{"Registration number is required"}
	}
	if imei == "" {
		validationErrors["imei"] = []string{"IMEI is required"}
	}
	if len(validationErrors) > 0 {
		return nil, domain.NewBadRequestError("Validation failed", instance, validationErrors)
	}

	imeiExists, err := s.repo.ExistsIMEI(ctx, imei, "")
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}
	if imeiExists {
		return nil, domain.NewBadRequestError("Validation failed", instance, map[string][]string{
			"imei": {"IMEI is already registered to another motorcycle"},
		})
	}

	regExists, err := s.repo.ExistsRegistrationNumber(ctx, claims.UserID, regNumber, "")
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}
	if regExists {
		return nil, domain.NewBadRequestError("Validation failed", instance, map[string][]string{
			"registration_number": {"Registration number is already in use for your account"},
		})
	}

	year := req.Year
	if year <= 0 {
		year = time.Now().Year()
	}

	now := time.Now().UTC()
	bikeID := uuid.New().String()

	bike := domain.Bike{
		ID:                 bikeID,
		UserID:             claims.UserID,
		RegistrationNumber: regNumber,
		Label:              strings.TrimSpace(req.Label),
		Make:               strings.TrimSpace(req.Make),
		Model:              strings.TrimSpace(req.Model),
		Year:               year,
		Color:              strings.TrimSpace(req.Color),
		IMEI:               imei,
		Status:             "STOPPED",
		Ignition:           false,
		SpeedKMH:           0.0,
		HeadingDeg:         0.0,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		CurrentAddress:     strings.TrimSpace(req.CurrentAddress),
		SubCounty:          strings.TrimSpace(req.SubCounty),
		County:             strings.TrimSpace(req.County),
		BatteryPercent:     100,
		SignalQualityCSQ:   20,
		LastTelemetryAt:    nil,
		GeofenceID:         strings.TrimSpace(req.GeofenceID),
		DriverContactID:    strings.TrimSpace(req.DriverContactID),
		RiderName:          strings.TrimSpace(req.RiderName),
		RiderNationalID:    strings.TrimSpace(req.RiderNationalID),
		RiderPhone:         strings.TrimSpace(req.RiderPhone),
		IsImmobilized:      false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Create(ctx, tx, &bike); err != nil {
		slog.Error("CreateBike repo.Create error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("CreateBike tx.Commit error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dto := toBikeDTO(&bike)
	return &dto, nil
}

func (s *BikeService) GetBike(ctx context.Context, claims *security.JWTClaims, bikeID string) (*domain.BikeDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	bike, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	dto := toBikeDTO(bike)
	return &dto, nil
}

func (s *BikeService) ListBikes(ctx context.Context, claims *security.JWTClaims) (*domain.BikeListResponse, error) {
	instance := "/v1/bikes"

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	bikes, err := s.repo.ListByOwner(ctx, claims.UserID)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	dtos := make([]domain.BikeDTO, 0, len(bikes))
	defaultSelectedID := ""
	for i := range bikes {
		dto := toBikeDTO(&bikes[i])
		dtos = append(dtos, dto)
		if i == 0 {
			defaultSelectedID = dto.ID
		}
	}

	return &domain.BikeListResponse{
		Data: dtos,
		Meta: domain.BikeListMeta{
			Total:                 len(dtos),
			DefaultSelectedBikeID: defaultSelectedID,
		},
	}, nil
}

func (s *BikeService) UpdateBike(ctx context.Context, claims *security.JWTClaims, bikeID string, req domain.UpdateBikeRequest) (*domain.BikeDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	existing, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	regNumber := strings.TrimSpace(req.RegistrationNumber)
	if regNumber != "" && !strings.EqualFold(regNumber, existing.RegistrationNumber) {
		regExists, err := s.repo.ExistsRegistrationNumber(ctx, claims.UserID, regNumber, bikeID)
		if err != nil {
			return nil, domain.NewInternalServerError(instance)
		}
		if regExists {
			return nil, domain.NewBadRequestError("Validation failed", instance, map[string][]string{
				"registration_number": {"Registration number is already in use for your account"},
			})
		}
		existing.RegistrationNumber = regNumber
	}

	if strings.TrimSpace(req.Label) != "" {
		existing.Label = strings.TrimSpace(req.Label)
	}
	if strings.TrimSpace(req.Make) != "" {
		existing.Make = strings.TrimSpace(req.Make)
	}
	if strings.TrimSpace(req.Model) != "" {
		existing.Model = strings.TrimSpace(req.Model)
	}
	if req.Year > 0 {
		existing.Year = req.Year
	}
	if strings.TrimSpace(req.Color) != "" {
		existing.Color = strings.TrimSpace(req.Color)
	}
	if strings.TrimSpace(req.CurrentAddress) != "" {
		existing.CurrentAddress = strings.TrimSpace(req.CurrentAddress)
	}
	if strings.TrimSpace(req.SubCounty) != "" {
		existing.SubCounty = strings.TrimSpace(req.SubCounty)
	}
	if strings.TrimSpace(req.County) != "" {
		existing.County = strings.TrimSpace(req.County)
	}
	if req.Status == "RUNNING" || req.Status == "STOPPED" {
		existing.Status = req.Status
	}
	if req.GeofenceID != "" {
		existing.GeofenceID = strings.TrimSpace(req.GeofenceID)
	}
	if req.DriverContactID != "" {
		existing.DriverContactID = strings.TrimSpace(req.DriverContactID)
	}
	if req.RiderName != "" {
		existing.RiderName = strings.TrimSpace(req.RiderName)
	}
	if req.RiderNationalID != "" {
		existing.RiderNationalID = strings.TrimSpace(req.RiderNationalID)
	}
	if req.RiderPhone != "" {
		existing.RiderPhone = strings.TrimSpace(req.RiderPhone)
	}

	existing.UpdatedAt = time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Update(ctx, tx, existing); err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	dto := toBikeDTO(existing)
	return &dto, nil
}

func (s *BikeService) DeleteBike(ctx context.Context, claims *security.JWTClaims, bikeID string) error {
	instance := fmt.Sprintf("/v1/bikes/%s", bikeID)

	if claims == nil || claims.UserID == "" {
		return domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Delete(ctx, tx, claims.UserID, bikeID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.NewInternalServerError(instance)
	}

	return nil
}

func (s *BikeService) GetBreadcrumbs(ctx context.Context, claims *security.JWTClaims, bikeID string, startDate string, endDate string, limit int) ([]domain.BreadcrumbDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/breadcrumbs", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	_, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	breadcrumbs, err := s.repo.GetBreadcrumbs(ctx, claims.UserID, bikeID, startDate, endDate, limit)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	if breadcrumbs == nil {
		breadcrumbs = []domain.BreadcrumbDTO{}
	}

	return breadcrumbs, nil
}

func (s *BikeService) GetEvents(ctx context.Context, claims *security.JWTClaims, bikeID string, startDate string, endDate string, limit int) ([]domain.BikeEventDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/events", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	_, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	events, err := s.repo.GetEvents(ctx, claims.UserID, bikeID, startDate, endDate, limit)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	if events == nil {
		events = []domain.BikeEventDTO{}
	}

	return events, nil
}

func (s *BikeService) ImmobilizeBike(ctx context.Context, claims *security.JWTClaims, bikeID string, immobilize bool) (*domain.BikeDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/immobilize", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	imei, err := s.repo.SetImmobilized(ctx, claims.UserID, bikeID, immobilize)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	if s.cmdPub != nil && imei != "" {
		action := "IMMOBILIZE"
		if !immobilize {
			action = "REACTIVATE"
		}
		_ = s.cmdPub.PublishCommand(imei, action)
	}

	bike, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	dto := toBikeDTO(bike)
	return &dto, nil
}

func toBikeDTO(b *domain.Bike) domain.BikeDTO {
	return domain.BikeDTO{
		ID:                 b.ID,
		UserID:             b.UserID,
		RegistrationNumber: b.RegistrationNumber,
		Label:              b.Label,
		Make:               b.Make,
		Model:              b.Model,
		Year:               b.Year,
		Color:              b.Color,
		IMEI:               b.IMEI,
		Status:             b.Status,
		Ignition:           b.Ignition,
		SpeedKMH:           b.SpeedKMH,
		HeadingDeg:         b.HeadingDeg,
		Latitude:           b.Latitude,
		Longitude:          b.Longitude,
		CurrentAddress:     b.CurrentAddress,
		SubCounty:          b.SubCounty,
		County:             b.County,
		BatteryPercent:     b.BatteryPercent,
		SignalQualityCSQ:   b.SignalQualityCSQ,
		LastTelemetryAt:    b.LastTelemetryAt,
		GeofenceID:         b.GeofenceID,
		DriverContactID:    b.DriverContactID,
		RiderName:          b.RiderName,
		RiderNationalID:    b.RiderNationalID,
		RiderPhone:         b.RiderPhone,
		IsImmobilized:      b.IsImmobilized,
		CreatedAt:          b.CreatedAt,
		UpdatedAt:          b.UpdatedAt,
	}
}
