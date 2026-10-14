package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"motoguard-backend/internal/config"
	"motoguard-backend/internal/database"
	"motoguard-backend/internal/handler"
	"motoguard-backend/internal/logger"
	"motoguard-backend/internal/mailer"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/mqtt"
	"motoguard-backend/internal/notification"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"
	"motoguard-backend/internal/sse"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load error", "error", err)
		os.Exit(1)
	}

	defer logger.Setup(cfg.LogFile)()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database pool error", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	var mailClient mailer.Mailer
	if cfg.MailjetAPIKey != "" && cfg.MailjetSecretKey != "" {
		slog.Info("Initializing Mailjet mailer",
			"sender_name", cfg.MailjetSenderName,
			"sender_email", cfg.MailjetSenderEmail,
		)
		mailClient = mailer.NewMailjetMailer(cfg.MailjetAPIKey, cfg.MailjetSecretKey, cfg.MailjetSenderEmail, cfg.MailjetSenderName, cfg.MailjetReplyToEmail)
	} else {
		slog.Info("Mailjet API credentials not configured, falling back to NoopMailer")
		mailClient = &mailer.NoopMailer{}
	}

	jwtManager := security.NewJWTManager(cfg.JWTSecret, 15*time.Minute)
	authRepo := repository.NewAuthRepository(pool)
	authService := service.NewAuthService(pool, authRepo, jwtManager, nil, mailClient, cfg.FrontendURL)
	authHandler := handler.NewAuthHandler(authService)

	sseBroker := sse.NewBroker()
	defer sseBroker.Stop()

	// Initialize AWS SNS Notification Service
	var snsClient notification.SNSClient
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		slog.Info("Initializing AWS SNS client", "region", cfg.AWSRegion, "platform_app_arn", cfg.AWSSNSPlatformAppARN)
		snsClient = notification.NewAWSSNSClient(cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
	} else {
		slog.Info("AWS credentials not configured, falling back to NoopSNSClient")
		snsClient = &notification.NoopSNSClient{}
	}

	notificationRepo := notification.NewRepository(pool)
	notificationService := notification.NewService(snsClient, notificationRepo, cfg.AWSSNSPlatformAppARN, cfg.AWSSNSTopicARN)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	bikeRepo := repository.NewBikeRepository(pool)
	bikeService := service.NewBikeService(pool, bikeRepo)
	bikeService.SetSSEBroker(sseBroker)
	bikeService.SetMailer(mailClient)
	bikeService.SetFrontendURL(cfg.FrontendURL)
	bikeHandler := handler.NewBikeHandler(bikeService)

	telemetryRepo := repository.NewTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(pool, telemetryRepo)
	telemetryService.SetBikeRepo(bikeRepo)
	telemetryService.SetSSEBroker(sseBroker)
	telemetryService.SetNotificationService(notificationService)

	mqttSubscriber, err := mqtt.NewMQTTSubscriber(cfg.MQTTBrokerURL, "motoguard-backend-ingest", cfg.MQTTUsername, cfg.MQTTPassword, telemetryService)
	if err != nil {
		slog.Error("mqtt client error", "error", err)
	} else {
		bikeService.SetCommandPublisher(mqttSubscriber)
		if err := mqttSubscriber.Start(ctx); err != nil {
			slog.Error("mqtt subscriber start error", "error", err)
		} else {
			defer mqttSubscriber.Stop()
		}
	}

	sseHandler := handler.NewSSEHandler(sseBroker, bikeService)
	healthHandler := handler.NewHealthHandler(pool)

	router := handler.NewRouter(handler.RouterConfig{
		AuthHandler:         authHandler,
		BikeHandler:         bikeHandler,
		HealthHandler:       healthHandler,
		NotificationHandler: notificationHandler,
		SSEHandler:          sseHandler,
		AuthMiddleware:      middleware.AuthMiddleware(jwtManager),
		RateLimitMiddleware: middleware.RateLimitMiddleware(5, time.Minute),
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
}
