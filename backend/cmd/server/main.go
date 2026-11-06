package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"motoguard-backend/internal/assistant"
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
	"motoguard-backend/internal/sms"
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
	}

	jwtManager := security.NewJWTManager(cfg.JWTSecret, 15*time.Minute)
	authRepo := repository.NewAuthRepository(pool)
	authService := service.NewAuthService(pool, authRepo, jwtManager, nil, mailClient, cfg.FrontendURL)
	authHandler := handler.NewAuthHandler(authService)
	go authService.RunAccountCleanup(ctx)

	sseBroker := sse.NewBroker()
	defer sseBroker.Stop()

	// Initialize AWS SNS Notification Service
	slog.Info("Initializing AWS SNS client", "region", cfg.AWSRegion, "platform_app_arn", cfg.AWSSNSPlatformAppARN)
	snsClient := notification.NewAWSSNSClient(cfg.AWSRegion, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)

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

	var cmdPub service.CommandPublisher
	mqttSubscriber, err := mqtt.NewMQTTSubscriber(cfg.MQTTBrokerURL, "motoguard-backend-ingest", cfg.MQTTUsername, cfg.MQTTPassword, telemetryService, bikeService)
	if err != nil {
		slog.Error("mqtt client error", "error", err)
	} else {
		cmdPub = mqttSubscriber
		bikeService.SetCommandPublisher(mqttSubscriber)
		if err := mqttSubscriber.Start(ctx); err != nil {
			slog.Error("mqtt subscriber start error", "error", err)
		} else {
			defer mqttSubscriber.Stop()
		}

		// Resend immobilizer commands the device has not confirmed
		go bikeService.RunCommandRetries(ctx)
	}

	// Initialize USSD Service for non-smartphone riders
	ussdRepo := repository.NewUSSDRepository(pool)
	ussdService := service.NewUSSDService(ussdRepo, cmdPub, sseBroker, notificationService)
	ussdHandler := handler.NewUSSDHandler(ussdService)

	// Bike recovery: theft reports and saved contacts
	recoveryRepo := repository.NewRecoveryRepository(pool)
	recoveryService := service.NewRecoveryService(recoveryRepo)
	if cfg.SMSAPIKey != "" && cfg.SMSUsername != "" {
		slog.Info("Initializing SMS sender for lost-bike alerts", "username", cfg.SMSUsername, "shortcode", cfg.SMSShortcode)
		recoveryService.SetSMSSender(sms.NewAfricasTalkingClient(cfg.SMSUsername, cfg.SMSAPIKey, cfg.SMSShortcode))
	} else {
		slog.Warn("SMS_API_KEY or SMS_USERNAME not set; lost-bike texts to contacts are disabled")
	}
	recoveryHandler := handler.NewRecoveryHandler(recoveryService)

	// Guided chatbot: menu answers come from the same user-scoped bike data
	chatRepo := repository.NewChatRepository(pool)
	var chatComposer service.ChatComposer
	if cfg.OpenAIAPIKey != "" {
		chatComposer = assistant.NewBudgetedComposer(
			assistant.NewOpenAIComposer(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL),
			chatRepo,
			cfg.ChatAIDailyLimit,
		)
	} else {
		slog.Warn("OPENAI_API_KEY not set; chatbot will use built-in wording")
	}
	chatService := service.NewChatService(bikeService, recoveryService, chatRepo, chatComposer)
	go chatService.RunRetention(ctx)
	chatHandler := handler.NewChatHandler(chatService)

	sseHandler := handler.NewSSEHandler(sseBroker, bikeService)
	healthHandler := handler.NewHealthHandler(pool)

	router := handler.NewRouter(handler.RouterConfig{
		AuthHandler:         authHandler,
		BikeHandler:         bikeHandler,
		HealthHandler:       healthHandler,
		NotificationHandler: notificationHandler,
		SSEHandler:          sseHandler,
		USSDHandler:         ussdHandler,
		RecoveryHandler:     recoveryHandler,
		ChatHandler:         chatHandler,
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
