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
	AWSRegion           string
	AWSAccessKeyID      string
	AWSSecretAccessKey  string
	AWSSNSPlatformAppARN string
	AWSSNSTopicARN       string
	ATAPIKey             string
	ATUsername           string
	ATSenderID           string
	ATIsSandbox          bool
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
	atUsername := getEnv("AFRICASTALKING_USERNAME", getEnv("AT_USERNAME", "sandbox"))
	atIsSandbox := atUsername == "sandbox" || strings.ToLower(getEnv("AFRICASTALKING_SANDBOX", "true")) == "true"

	return Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          dbURL,
		JWTSecret:            jwtSecret,
		MQTTBrokerURL:        getEnv("MQTT_BROKER_URL", "tcp://localhost:1883"),
		MQTTUsername:         getEnv("MQTT_USERNAME", ""),
		MQTTPassword:         getEnv("MQTT_PASSWORD", ""),
		MailjetAPIKey:        getEnv("MAILJET_API_KEY", ""),
		MailjetSecretKey:     getEnv("MAILJET_SECRET_KEY", ""),
		MailjetSenderEmail:   getEnv("MAILJET_SENDER_EMAIL", ""),
		MailjetReplyToEmail:  getEnv("MAILJET_REPLY_TO_EMAIL", ""),
		MailjetSenderName:    getEnv("MAILJET_SENDER_NAME", "MGuard"),
		FrontendURL:          getEnv("FRONTEND_URL", defaultFrontendURL),
		LogFile:              getEnv("LOG_FILE", "/var/log/fleet/fleet.log"),
		AWSRegion:            getEnv("AWS_REGION", "eu-west-1"),
		AWSAccessKeyID:       getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey:   getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSSNSPlatformAppARN: getEnv("AWS_SNS_PLATFORM_APP_ARN", ""),
		AWSSNSTopicARN:       getEnv("AWS_SNS_TOPIC_ARN", ""),
		ATAPIKey:             getEnv("AFRICASTALKING_API_KEY", getEnv("AT_API_KEY", "")),
		ATUsername:           atUsername,
		ATSenderID:           getEnv("AFRICASTALKING_SENDER_ID", getEnv("AT_SENDER_ID", "")),
		ATIsSandbox:          atIsSandbox,
	}, nil
}
