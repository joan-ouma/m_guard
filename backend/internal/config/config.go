package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	MQTTBrokerURL       string
	MQTTUsername        string
	MQTTPassword        string
	MailjetAPIKey       string
	MailjetSecretKey    string
	MailjetSenderEmail  string
	MailjetReplyToEmail string
	MailjetSenderName   string
	FrontendURL         string
	LogFile             string
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func Load() (Config, error) {
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	defaultFrontendURL := "https://mxguard.jullie.tech"

	return Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         dbURL,
		JWTSecret:           jwtSecret,
		MQTTBrokerURL:       getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTUsername:        getEnv("MQTT_USERNAME", "tracker"),
		MQTTPassword:        getEnv("MQTT_PASSWORD", "~:!Nu2>#6TDe5Y+"),
		MailjetAPIKey:       getEnv("MAILJET_API_KEY", ""),
		MailjetSecretKey:    getEnv("MAILJET_SECRET_KEY", ""),
		MailjetSenderEmail:  getEnv("MAILJET_SENDER_EMAIL", "no-reply@jullie.tech"),
		MailjetReplyToEmail: getEnv("MAILJET_REPLY_TO_EMAIL", "hello@jullie.tech"),
		MailjetSenderName:   getEnv("MAILJET_SENDER_NAME", "MGuard"),
		FrontendURL:         getEnv("FRONTEND_URL", defaultFrontendURL),
		LogFile:             getEnv("LOG_FILE", "/var/log/fleet/fleet.log"),
	}, nil
}
