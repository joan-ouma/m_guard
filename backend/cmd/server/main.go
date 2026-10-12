package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"motoguard-backend/internal/config"
	"motoguard-backend/internal/database"
	"motoguard-backend/internal/handler"
	"motoguard-backend/internal/mailer"
	"motoguard-backend/internal/middleware"
	"motoguard-backend/internal/mqtt"
	"motoguard-backend/internal/repository"
	"motoguard-backend/internal/security"
	"motoguard-backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database pool error: %v", err)
	}
	defer pool.Close()

	var mailClient mailer.Mailer
	if cfg.MailjetAPIKey != "" && cfg.MailjetSecretKey != "" {
		log.Printf("Initializing Mailjet mailer with sender %s <%s> and reply-to <%s>", cfg.MailjetSenderName, cfg.MailjetSenderEmail, cfg.MailjetReplyToEmail)
		mailClient = mailer.NewMailjetMailer(cfg.MailjetAPIKey, cfg.MailjetSecretKey, cfg.MailjetSenderEmail, cfg.MailjetSenderName, cfg.MailjetReplyToEmail)
	} else {
		log.Printf("Mailjet API credentials not configured, falling back to NoopMailer")
		mailClient = &mailer.NoopMailer{}
	}

	jwtManager := security.NewJWTManager(cfg.JWTSecret, 15*time.Minute)
	authRepo := repository.NewAuthRepository(pool)
	authService := service.NewAuthService(pool, authRepo, jwtManager, nil, mailClient, cfg.FrontendURL)
	authHandler := handler.NewAuthHandler(authService)

	bikeRepo := repository.NewBikeRepository(pool)
	bikeService := service.NewBikeService(pool, bikeRepo)
	bikeHandler := handler.NewBikeHandler(bikeService)

	telemetryRepo := repository.NewTelemetryRepository(pool)
	telemetryService := service.NewTelemetryService(pool, telemetryRepo)

	mqttSubscriber, err := mqtt.NewMQTTSubscriber(cfg.MQTTBrokerURL, "motoguard-backend-ingest", cfg.MQTTUsername, cfg.MQTTPassword, telemetryService)
	if err != nil {
		log.Printf("mqtt client error: %v", err)
	} else {
		bikeService.SetCommandPublisher(mqttSubscriber)
		if err := mqttSubscriber.Start(ctx); err != nil {
			log.Printf("mqtt subscriber start error: %v", err)
		} else {
			defer mqttSubscriber.Stop()
		}
	}

	router := handler.NewRouter(handler.RouterConfig{
		AuthHandler:         authHandler,
		BikeHandler:         bikeHandler,
		AuthMiddleware:      middleware.AuthMiddleware(jwtManager),
		RateLimitMiddleware: middleware.RateLimitMiddleware(5, time.Minute),
	})

	httpHandler := middleware.CORSMiddleware(cfg.CORSOrigins)(router)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen error: %v", err)
		}
	}()

	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
