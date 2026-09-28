package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	CORSOrigins   []string
	MQTTBrokerURL string
}

func Load() (Config, error) {
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok || dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	jwtSecret, ok := os.LookupEnv("JWT_SECRET")
	if !ok || jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok || port == "" {
		port = "8080"
	}

	corsRaw, ok := os.LookupEnv("CORS_ORIGINS")
	var origins []string
	if ok && corsRaw != "" {
		for _, o := range strings.Split(corsRaw, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	} else {
		origins = []string{"http://localhost:4200"}
	}

	mqttURL, ok := os.LookupEnv("MQTT_BROKER_URL")
	if !ok || mqttURL == "" {
		mqttURL = "tcp://localhost:1883"
	}

	return Config{
		Port:          port,
		DatabaseURL:   dbURL,
		JWTSecret:     jwtSecret,
		CORSOrigins:   origins,
		MQTTBrokerURL: mqttURL,
	}, nil
}

