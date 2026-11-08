package config

import (
	"fmt"
	"os"
	"strconv"
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
	OpenAIAPIKey         string
	OpenAIModel          string
	OpenAIBaseURL        string
	ChatAIDailyLimit     int
	SMSAPIKey            string
	SMSUsername          string
	SMSShortcode         string
	SMSRoutingKeyword    string
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

	chatAILimit, err := strconv.Atoi(getEnv("CHAT_AI_DAILY_LIMIT", "1000"))
	if err != nil || chatAILimit < 0 {
		return Config{}, fmt.Errorf("CHAT_AI_DAILY_LIMIT must be a non-negative whole number")
	}

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
		OpenAIAPIKey:         getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:          getEnv("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIBaseURL:        getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		ChatAIDailyLimit:     chatAILimit,
		SMSAPIKey:            getEnv("SMS_API_KEY", ""),
		SMSUsername:          getEnv("SMS_USERNAME", ""),
		SMSShortcode:         getEnv("SMS_SHORTCODE", ""),
		SMSRoutingKeyword:    getEnv("SMS_ROUTING_KEYWORD", "kamilimu"),
	}, nil
}
