package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                     string
	DatabaseURL              string
	JWTSecret                string
	Environment              string
	DDOStampHeader           string
	ValidatorCredentialsFile string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/aktarbilet?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-jwt-key-ddo-aktarbilet-2026")
	env := getEnv("ENVIRONMENT", "development")
	stamp := getEnv("DDO_STAMP", "f85f26")
	credentialsFile := os.Getenv("VALIDATOR_CREDENTIALS_FILE")

	return &Config{
		Port:                     port,
		DatabaseURL:              dbURL,
		JWTSecret:                jwtSecret,
		Environment:              env,
		DDOStampHeader:           stamp,
		ValidatorCredentialsFile: credentialsFile,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
