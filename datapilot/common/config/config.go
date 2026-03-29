package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all configuration values for a DataPilot microservice.
type Config struct {
	ServiceName         string
	HTTPPort            string
	MySQLDSN            string
	JWTSecret           string
	FileStoragePath     string
	LogLevel            string
	AllowedOrigins      string
	FileServiceURL      string
	SchedulerServiceURL string
}

// LoadConfig reads environment variables (with optional .env file fallback)
// and returns a populated Config. Returns a descriptive error if any required
// key is missing.
func LoadConfig() (*Config, error) {
	// Best-effort load of .env file; ignore error if file doesn't exist.
	_ = godotenv.Load()

	required := []string{"SERVICE_NAME", "HTTP_PORT", "MYSQL_DSN", "JWT_SECRET"}
	for _, key := range required {
		if os.Getenv(key) == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	fileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if fileStoragePath == "" {
		fileStoragePath = "/data/files"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	fileServiceURL := os.Getenv("FILE_SERVICE_URL")
	if fileServiceURL == "" {
		fileServiceURL = "http://file-service:8081"
	}

	schedulerServiceURL := os.Getenv("SCHEDULER_SERVICE_URL")
	if schedulerServiceURL == "" {
		schedulerServiceURL = "http://scheduler-service:8082"
	}

	return &Config{
		ServiceName:         os.Getenv("SERVICE_NAME"),
		HTTPPort:            os.Getenv("HTTP_PORT"),
		MySQLDSN:            os.Getenv("MYSQL_DSN"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		FileStoragePath:     fileStoragePath,
		LogLevel:            logLevel,
		AllowedOrigins:      allowedOrigins,
		FileServiceURL:      fileServiceURL,
		SchedulerServiceURL: schedulerServiceURL,
	}, nil
}
