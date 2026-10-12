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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthService struct {
	pool                 *pgxpool.Pool
	repo                 *repository.AuthRepository
	jwtManager           *security.JWTManager
	argonParams          *security.Argon2Params
	refreshTokenDuration time.Duration
	mailer               mailer.Mailer
	frontendURL          string
}

func NewAuthService(
	pool *pgxpool.Pool,
	repo *repository.AuthRepository,
	jwtManager *security.JWTManager,
	argonParams *security.Argon2Params,
	mailClient mailer.Mailer,
	frontendURL string,
) *AuthService {
	if argonParams == nil {
		argonParams = security.DefaultArgon2Params
	}
	if mailClient == nil {
		mailClient = &mailer.NoopMailer{}
	}
	frontendURL = strings.TrimRight(frontendURL, "/")
	if frontendURL == "" {
		frontendURL = "http://localhost:4200"
	}
	return &AuthService{
		pool:                 pool,
		repo:                 repo,
		jwtManager:           jwtManager,
		argonParams:          argonParams,
		refreshTokenDuration: 7 * 24 * time.Hour,
		mailer:               mailClient,
		frontendURL:          frontendURL,
	}
}

func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.UserDTO, error) {
	instance := "/v1/users/register"

	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, domain.ProblemDetails{
			Type:     "https://motoguard.co.ke/errors/conflict",
			Title:    "Conflict",
			Status:   409,
			Detail:   "A user with this email already exists",
			Instance: instance,
		}
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, domain.NewInternalServerError(instance)
	}

	hashedPassword, err := security.HashPassword(req.Password, s.argonParams)
	if err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		Name:         req.Name,
		NationalID:   req.NationalID,
		PhoneNumber:  req.PhoneNumber,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, domain.NewInternalServerError(instance)
	}

	return &domain.UserDTO{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        "owner",
		NationalID:  user.NationalID,
		PhoneNumber: user.PhoneNumber,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (*domain.SessionData, string, error) {
	instance := "/v1/sessions"

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			slog.Warn("Login failed: user not found", "email", email)
			return nil, "", domain.NewUnauthorizedError("Invalid email or password", instance)
		}
		slog.Error("Login failed: database error", "email", email, "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}

	matched, err := security.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		slog.Error("Login password verification failed: malformed hash", "email", email, "error", err)
		return nil, "", domain.NewUnauthorizedError("Invalid email or password", instance)
	}
	if !matched {
		slog.Warn("Login failed: invalid password", "email", email)
		return nil, "", domain.NewUnauthorizedError("Invalid email or password", instance)
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID)
	if err != nil {
		slog.Error("Login failed: access token error", "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}

	rawRefreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		slog.Error("Login failed: refresh token error", "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}

	refreshTokenHash := security.HashTokenSHA256(rawRefreshToken)
	sessionID := uuid.New().String()
	now := time.Now().UTC()

	ua := userAgent
	if len(ua) > 255 {
		ua = ua[:255]
	}
	ipAddr := ip
	if len(ipAddr) > 45 {
		ipAddr = ipAddr[:45]
	}

	session := &domain.Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        now.Add(s.refreshTokenDuration),
		UserAgent:        ua,
		IPAddress:        ipAddr,
		CreatedAt:        now,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("Login failed: pool.Begin error", "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateSession(ctx, tx, session); err != nil {
		slog.Error("Login failed: CreateSession error", "user_id", user.ID, "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("Login failed: tx.Commit error", "user_id", user.ID, "error", err)
		return nil, "", domain.NewInternalServerError(instance)
	}

	slog.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)

	sessionData := &domain.SessionData{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		User: domain.UserDTO{
			ID:          user.ID,
			Email:       user.Email,
			Name:        user.Name,
			Role:        "owner",
			NationalID:  user.NationalID,
			PhoneNumber: user.PhoneNumber,
		},
	}

	return sessionData, rawRefreshToken, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken, ip, userAgent string) (*domain.SessionData, string, error) {
	instance := "/v1/sessions/current"

	if rawRefreshToken == "" {
		return nil, "", domain.NewUnauthorizedError("Refresh token missing or invalid", instance)
	}

	tokenHash := security.HashTokenSHA256(rawRefreshToken)
	session, user, err := s.repo.GetSessionWithUser(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", domain.NewUnauthorizedError("Session not found or expired", instance)
		}
		return nil, "", domain.NewInternalServerError(instance)
	}

	if !session.RevokedAt.IsZero() {
		return nil, "", domain.NewUnauthorizedError("Session has been revoked", instance)
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, "", domain.NewUnauthorizedError("Session has expired", instance)
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}

	newRawRefreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}

	ua := userAgent
	if len(ua) > 255 {
		ua = ua[:255]
	}
	ipAddr := ip
	if len(ipAddr) > 45 {
		ipAddr = ipAddr[:45]
	}

	newSession := &domain.Session{
		ID:               uuid.New().String(),
		UserID:           user.ID,
		RefreshTokenHash: security.HashTokenSHA256(newRawRefreshToken),
		ExpiresAt:        time.Now().UTC().Add(s.refreshTokenDuration),
		UserAgent:        ua,
		IPAddress:        ipAddr,
		CreatedAt:        time.Now().UTC(),
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.RevokeSession(ctx, tx, session.ID); err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}

	if err := s.repo.CreateSession(ctx, tx, newSession); err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", domain.NewInternalServerError(instance)
	}

	sessionData := &domain.SessionData{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		User: domain.UserDTO{
			ID:          user.ID,
			Email:       user.Email,
			Name:        user.Name,
			Role:        "owner",
			NationalID:  user.NationalID,
			PhoneNumber: user.PhoneNumber,
		},
	}

	return sessionData, newRawRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	tokenHash := security.HashTokenSHA256(rawRefreshToken)
	session, _, err := s.repo.GetSessionWithUser(ctx, tokenHash)
	if err != nil {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.RevokeSession(ctx, tx, session.ID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil
	}

	rawToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return fmt.Errorf("generate reset token error: %w", err)
	}

	tokenHash := security.HashTokenSHA256(rawToken)
	now := time.Now().UTC()

	prt := &domain.PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(30 * time.Minute),
		CreatedAt: now,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreatePasswordResetToken(ctx, tx, prt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password/%s", s.frontendURL, rawToken)
	userEmail := user.Email
	userName := user.Name

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := s.mailer.SendPasswordResetEmail(bgCtx, userEmail, userName, resetURL); err != nil {
			slog.Error("failed to send password reset email", "email", userEmail, "error", err)
		}
	}()

	return nil
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string) error {
	instance := "/v1/password-resets"

	if len(newPassword) < 8 {
		return domain.NewBadRequestError("Password must be at least 8 characters long", instance, map[string][]string{
			"password": {"Minimum length is 8 characters"},
		})
	}

	tokenHash := security.HashTokenSHA256(rawToken)
	prt, err := s.repo.GetPasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.NewBadRequestError("Password reset token is invalid or has expired", instance, nil)
		}
		return domain.NewInternalServerError(instance)
	}

	if !prt.UsedAt.IsZero() {
		return domain.NewBadRequestError("Password reset token has already been used", instance, nil)
	}

	if time.Now().UTC().After(prt.ExpiresAt) {
		return domain.NewBadRequestError("Password reset token has expired", instance, nil)
	}

	newHash, err := security.HashPassword(newPassword, s.argonParams)
	if err != nil {
		return domain.NewInternalServerError(instance)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.NewInternalServerError(instance)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.MarkPasswordResetUsed(ctx, tx, prt.ID); err != nil {
		return domain.NewInternalServerError(instance)
	}

	if err := s.repo.UpdateUserPassword(ctx, tx, prt.UserID, newHash); err != nil {
		return domain.NewInternalServerError(instance)
	}

	if err := s.repo.RevokeAllUserSessions(ctx, tx, prt.UserID); err != nil {
		return domain.NewInternalServerError(instance)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.NewInternalServerError(instance)
	}

	user, err := s.repo.GetUserByID(ctx, prt.UserID)
	if err == nil && user != nil {
		userEmail := user.Email
		userName := user.Name
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := s.mailer.SendPasswordChangedEmail(bgCtx, userEmail, userName); err != nil {
				slog.Error("failed to send password changed confirmation email", "email", userEmail, "error", err)
			}
		}()
	}

	return nil
}

func (s *AuthService) GetCurrentSession(ctx context.Context, userID string) (*domain.UserDTO, error) {
	instance := "/v1/sessions/current"

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.NewNotFoundError("User not found", instance)
		}
		return nil, domain.NewInternalServerError(instance)
	}

	userDTO := &domain.UserDTO{
		ID:          user.ID,
		Email:       user.Email,
		Name:        user.Name,
		Role:        "owner",
		NationalID:  user.NationalID,
		PhoneNumber: user.PhoneNumber,
	}

	return userDTO, nil
}
