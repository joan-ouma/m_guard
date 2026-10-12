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
	CORSOrigins         []string
	MQTTBrokerURL       string
	MQTTUsername        string
	MQTTPassword        string
	MailjetAPIKey       string
	MailjetSecretKey    string
	MailjetSenderEmail  string
	MailjetReplyToEmail string
	MailjetSenderName   string
	FrontendURL         string
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	var items []string
	for _, item := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}

	if len(items) == 0 {
		return fallback
	}
	return items
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

	origins := getEnvSlice("CORS_ORIGINS", []string{
		"https://mxguard.jullie.tech",
		"http://mxguard.jullie.tech",
		"*.jullie.tech",
		"*.vercel.app",
	})

	defaultFrontendURL := "https://mxguard.jullie.tech"

	return Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         dbURL,
		JWTSecret:           jwtSecret,
		CORSOrigins:         origins,
		MQTTBrokerURL:       getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTUsername:        getEnv("MQTT_USERNAME", "tracker"),
		MQTTPassword:        getEnv("MQTT_PASSWORD", "~:!Nu2>#6TDe5Y+"),
		MailjetAPIKey:       getEnv("MAILJET_API_KEY", ""),
		MailjetSecretKey:    getEnv("MAILJET_SECRET_KEY", ""),
		MailjetSenderEmail:  getEnv("MAILJET_SENDER_EMAIL", "no-reply@jullie.tech"),
		MailjetReplyToEmail: getEnv("MAILJET_REPLY_TO_EMAIL", "hello@jullie.tech"),
		MailjetSenderName:   getEnv("MAILJET_SENDER_NAME", "MGuard"),
		FrontendURL:         getEnv("FRONTEND_URL", defaultFrontendURL),
	}, nil
}
