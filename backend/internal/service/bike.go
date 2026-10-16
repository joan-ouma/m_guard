package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/mailer"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandPublisher interface {
	PublishCommand(imei string, action string) error
}

type SSEBroker interface {
	BroadcastBikeUpdate(userID string, bike domain.BikeDTO)
	BroadcastAlert(userID string, alert any)
}

type BikeService struct {
	pool        *pgxpool.Pool
	repo        *repository.BikeRepository
	cmdPub      CommandPublisher
	sseBroker   SSEBroker
	mailer      mailer.Mailer
	frontendURL string
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

func (s *BikeService) SetSSEBroker(broker SSEBroker) {
	s.sseBroker = broker
}

func (s *BikeService) SetMailer(m mailer.Mailer) {
	s.mailer = m
}

func (s *BikeService) SetFrontendURL(url string) {
	s.frontendURL = url
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
		slog.Error("CreateBike ExistsIMEI error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}
	if imeiExists {
		return nil, domain.NewBadRequestError("Validation failed", instance, map[string][]string{
			"imei": {"IMEI is already registered to another motorcycle"},
		})
	}

	regExists, err := s.repo.ExistsRegistrationNumber(ctx, repository.ExistsRegParams{
		UserID:    claims.UserID,
		RegNumber: regNumber,
	})
	if err != nil {
		slog.Error("CreateBike ExistsRegistrationNumber error", "error", err)
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
	bikeID := security.NewID("bik_")

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
		BatteryPercent:     0,
		SignalQualityCSQ:   0,
		LastTelemetryAt:    nil,
		GeofenceID:         strings.TrimSpace(req.GeofenceID),
		DriverContactID:    strings.TrimSpace(req.DriverContactID),
		IsImmobilized:      false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("CreateBike pool.Begin error", "error", err)
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

	dto := ToBikeDTO(&bike)
	if s.sseBroker != nil {
		s.sseBroker.BroadcastBikeUpdate(dto.UserID, dto)
	}
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
		slog.Error("GetBike GetByID error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dto := ToBikeDTO(bike)
	return &dto, nil
}

func (s *BikeService) ListBikes(ctx context.Context, claims *security.JWTClaims, cursor string, limit int) (*domain.BikeListResponse, error) {
	instance := "/v1/bikes"

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	bikes, meta, err := s.repo.ListByOwner(ctx, repository.ListByOwnerParams{
		UserID: claims.UserID,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		slog.Error("ListBikes ListByOwner error", "user_id", claims.UserID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dtos := make([]domain.BikeDTO, 0, len(bikes))
	defaultSelectedID := ""
	for i := range bikes {
		dto := ToBikeDTO(&bikes[i])
		dtos = append(dtos, dto)
		if i == 0 {
			defaultSelectedID = dto.ID
		}
	}

	meta.DefaultSelectedBikeID = defaultSelectedID

	return &domain.BikeListResponse{
		Data: dtos,
		Meta: meta,
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
		slog.Error("UpdateBike GetByID error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	regNumber := strings.TrimSpace(req.RegistrationNumber)
	if regNumber != "" && !strings.EqualFold(regNumber, existing.RegistrationNumber) {
		regExists, err := s.repo.ExistsRegistrationNumber(ctx, repository.ExistsRegParams{
			UserID:        claims.UserID,
			RegNumber:     regNumber,
			ExcludeBikeID: bikeID,
		})
		if err != nil {
			slog.Error("UpdateBike ExistsRegistrationNumber error", "error", err)
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

	existing.UpdatedAt = time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("UpdateBike pool.Begin error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Update(ctx, tx, existing); err != nil {
		slog.Error("UpdateBike repo.Update error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("UpdateBike tx.Commit error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dto := ToBikeDTO(existing)
	if s.sseBroker != nil {
		s.sseBroker.BroadcastBikeUpdate(dto.UserID, dto)
	}
	return &dto, nil
}

func (s *BikeService) DeleteBike(ctx context.Context, claims *security.JWTClaims, bikeID string) error {
	instance := fmt.Sprintf("/v1/bikes/%s", bikeID)

	if claims == nil || claims.UserID == "" {
		return domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("DeleteBike pool.Begin error", "error", err)
		return domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Delete(ctx, tx, claims.UserID, bikeID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewNotFoundError("Motorcycle not found", instance)
		}
		slog.Error("DeleteBike repo.Delete error", "error", err)
		return domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("DeleteBike tx.Commit error", "error", err)
		return domain.NewInternalServerError(instance)
	}

	return nil
}

func (s *BikeService) GetBreadcrumbs(
	ctx context.Context,
	claims *security.JWTClaims,
	bikeID string,
	startDate string,
	endDate string,
	cursor int64,
	limit int,
) (*domain.PaginatedBreadcrumbsResponse, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/breadcrumbs", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	_, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		slog.Error("GetBreadcrumbs GetByID error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	breadcrumbs, pagination, err := s.repo.GetBreadcrumbs(ctx, repository.GetBreadcrumbsParams{
		UserID:    claims.UserID,
		BikeID:    bikeID,
		StartDate: startDate,
		EndDate:   endDate,
		Cursor:    cursor,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("GetBreadcrumbs repo.GetBreadcrumbs error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	if breadcrumbs == nil {
		breadcrumbs = []domain.BreadcrumbDTO{}
	}

	return &domain.PaginatedBreadcrumbsResponse{
		Data:       breadcrumbs,
		Pagination: pagination,
	}, nil
}

func (s *BikeService) GetEvents(
	ctx context.Context,
	claims *security.JWTClaims,
	bikeID string,
	startDate string,
	endDate string,
	cursor int64,
	limit int,
) (*domain.PaginatedEventsResponse, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/events", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	_, err := s.repo.GetByID(ctx, claims.UserID, bikeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		slog.Error("GetEvents GetByID error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	events, pagination, err := s.repo.GetEvents(ctx, repository.GetEventsParams{
		UserID:    claims.UserID,
		BikeID:    bikeID,
		StartDate: startDate,
		EndDate:   endDate,
		Cursor:    cursor,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("GetEvents repo.GetEvents error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	if events == nil {
		events = []domain.BikeEventDTO{}
	}

	return &domain.PaginatedEventsResponse{
		Data:       events,
		Pagination: pagination,
	}, nil
}

func (s *BikeService) GetAllEvents(
	ctx context.Context,
	claims *security.JWTClaims,
	severity string,
	cursor int64,
	limit int,
) (*domain.PaginatedEventsResponse, error) {
	instance := "/v1/bikes/events"

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	events, pagination, critCount, warnCount, infoCount, err := s.repo.GetAllEvents(ctx, repository.GetAllEventsParams{
		UserID:   claims.UserID,
		Severity: severity,
		Cursor:   cursor,
		Limit:    limit,
	})
	if err != nil {
		slog.Error("GetAllEvents repo.GetAllEvents error", "user_id", claims.UserID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	if events == nil {
		events = []domain.BikeEventDTO{}
	}

	return &domain.PaginatedEventsResponse{
		Data:          events,
		Pagination:    pagination,
		CriticalCount: critCount,
		WarningCount:  warnCount,
		InfoCount:     infoCount,
	}, nil
}

func (s *BikeService) ImmobilizeBike(ctx context.Context, claims *security.JWTClaims, bikeID string, immobilize bool) (*domain.BikeDTO, error) {
	instance := fmt.Sprintf("/v1/bikes/%s/immobilize", bikeID)

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	imei, err := s.repo.SetImmobilized(ctx, repository.SetImmobilizedParams{
		UserID:        claims.UserID,
		BikeID:        bikeID,
		IsImmobilized: immobilize,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("Motorcycle not found", instance)
		}
		slog.Error("ImmobilizeBike SetImmobilized error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
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
		slog.Error("ImmobilizeBike GetByID error", "user_id", claims.UserID, "bike_id", bikeID, "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dto := ToBikeDTO(bike)
	if s.sseBroker != nil {
		s.sseBroker.BroadcastBikeUpdate(dto.UserID, dto)
	}
	return &dto, nil
}

func ToBikeDTO(b *domain.Bike) domain.BikeDTO {
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
		Accuracy:           b.Accuracy,
		LocType:            b.LocType,
		CurrentAddress:     b.CurrentAddress,
		SubCounty:          b.SubCounty,
		County:             b.County,
		BatteryPercent:     b.BatteryPercent,
		SignalQualityCSQ:   b.SignalQualityCSQ,
		LastTelemetryAt:    b.LastTelemetryAt,
		GeofenceID:         b.GeofenceID,
		DriverContactID:    b.DriverContactID,
		IsImmobilized:      b.IsImmobilized,
		CreatedAt:          b.CreatedAt,
		UpdatedAt:          b.UpdatedAt,
	}
}

func (s *BikeService) VerifyTracker(ctx context.Context, imei string) (*domain.TrackerStatusDTO, error) {
	instance := "/v1/bikes/verify-tracker"
	cleanIMEI := strings.TrimSpace(imei)
	if cleanIMEI == "" {
		return nil, domain.NewBadRequestError("Tracker IMEI is required", instance, map[string][]string{
			"imei": {"IMEI code is required"},
		})
	}

	dto, err := s.repo.VerifyTracker(ctx, cleanIMEI)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError(fmt.Sprintf("Tracker hardware with IMEI %s does not exist in MotoGuard database", cleanIMEI), instance)
		}
		slog.Error("VerifyTracker repo error", "error", err, "imei", cleanIMEI)
		return nil, domain.NewInternalServerError(instance)
	}

	if dto.IsClaimed {
		return dto, domain.NewConflictError(fmt.Sprintf("Tracker with IMEI %s is already assigned to an active motorcycle and cannot be claimed", cleanIMEI), instance)
	}

	return dto, nil
}

func (s *BikeService) AttachBike(ctx context.Context, claims *security.JWTClaims, req domain.AttachBikeRequest) (*domain.BikeDTO, error) {
	instance := "/v1/bikes/attach"

	if claims == nil || claims.UserID == "" {
		return nil, domain.NewUnauthorizedError("Authentication token is missing", instance)
	}

	regNumber := strings.ToUpper(strings.TrimSpace(req.RegistrationNumber))
	imei := strings.TrimSpace(req.IMEI)
	riderName := strings.TrimSpace(req.RiderName)
	riderPhone := strings.TrimSpace(req.RiderPhone)
	riderEmail := strings.TrimSpace(req.RiderEmail)

	validationErrors := make(map[string][]string)
	if regNumber == "" {
		validationErrors["registration_number"] = []string{"Registration number is required"}
	}
	if imei == "" {
		validationErrors["imei"] = []string{"Tracker IMEI is required"}
	}
	if riderName == "" {
		validationErrors["rider_name"] = []string{"Rider name is required"}
	}
	if riderPhone == "" {
		validationErrors["rider_phone"] = []string{"Rider phone is required"}
	}
	if riderEmail == "" {
		validationErrors["rider_email"] = []string{"Rider email is required for tracking notifications"}
	}
	if len(validationErrors) > 0 {
		return nil, domain.NewBadRequestError("Validation failed", instance, validationErrors)
	}

	// Verify tracker is available
	trackerStatus, err := s.repo.VerifyTracker(ctx, imei)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError(fmt.Sprintf("Tracker hardware with IMEI %s not found", imei), instance)
		}
		slog.Error("AttachBike VerifyTracker error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}
	if trackerStatus.IsClaimed {
		return nil, domain.NewConflictError(fmt.Sprintf("Tracker %s is already claimed by another motorcycle", imei), instance)
	}

	// Check if user already has this registration number
	regExists, err := s.repo.ExistsRegistrationNumber(ctx, repository.ExistsRegParams{
		UserID:    claims.UserID,
		RegNumber: regNumber,
	})
	if err != nil {
		slog.Error("AttachBike ExistsRegistrationNumber error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}
	if regExists {
		return nil, domain.NewBadRequestError("Validation failed", instance, map[string][]string{
			"registration_number": {"Registration number is already registered under your account"},
		})
	}

	req.RegistrationNumber = regNumber
	req.IMEI = imei
	req.RiderName = riderName
	req.RiderPhone = riderPhone
	req.RiderEmail = riderEmail

	bike, err := s.repo.AttachBikeWithRider(ctx, claims.UserID, req)
	if err != nil {
		slog.Error("AttachBike AttachBikeWithRider error", "error", err)
		return nil, domain.NewInternalServerError(instance)
	}

	dto := ToBikeDTO(bike)

	// Send email notification to rider asynchronously
	if s.mailer != nil && riderEmail != "" {
		go func(recipientEmail, recipientName string, bDTO domain.BikeDTO) {
			ctxEmail, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			dashURL := s.frontendURL

			emailErr := s.mailer.SendTrackerAttachedEmail(ctxEmail, recipientEmail, recipientName, mailer.TrackerAttachedEmailData{
				RiderName:        recipientName,
				RiderEmail:       recipientEmail,
				BikeRegistration: bDTO.RegistrationNumber,
				BikeMake:         bDTO.Make,
				BikeModel:        bDTO.Model,
				IMEI:             bDTO.IMEI,
				AttachedAt:       time.Now().UTC().Format(time.RFC1123),
				DashboardURL:     dashURL,
			})
			if emailErr != nil {
				slog.Warn("Failed to send tracker attached email to rider", "email", recipientEmail, "error", emailErr)
				return
			}
			slog.Info("Sent tracker attached email to rider", "email", recipientEmail, "bike", bDTO.RegistrationNumber)
		}(riderEmail, riderName, dto)
	}

	// Broadcast over SSE
	if s.sseBroker != nil {
		s.sseBroker.BroadcastBikeUpdate(dto.UserID, dto)
	}

	return &dto, nil
}
