package config

import (
	"os"
	"strconv"
)

// DatabaseConfig holds PostgreSQL database connection settings.
type DatabaseConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	Name               string
	SSLMode            string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeSec int
}

// MinIOConfig holds object storage settings for MinIO.
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// AppConfig is the centralized configuration struct for the application.
// It is populated from environment variables. Sensitive values are not hardcoded.
type AppConfig struct {
	AppHost  string
	Port     string
	Database DatabaseConfig
	MinIO    MinIOConfig
}

// Load reads configuration from environment variables.
// A .env file can be auto-loaded by importing: _ "github.com/joho/godotenv/autoload"
// This function does not require a .env file; real environment variables take precedence.
func Load() *AppConfig {
	return &AppConfig{
		AppHost: getEnv("APP_HOST", "localhost:8080"),
		Port:    getEnv("PORT", "8080"), // default only for non-sensitive value
		Database: DatabaseConfig{
			Host:               getEnv("DB_HOST", ""),
			Port:               getEnv("DB_PORT", "5432"),
			User:               getEnv("DB_USER", ""),
			Password:           getEnv("DB_PASSWORD", ""),
			Name:               getEnv("DB_NAME", ""),
			SSLMode:            getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:       getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:       getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetimeSec: getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 300),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", ""),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", ""),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
		},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return def
}
