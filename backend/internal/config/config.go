package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBHost              string
	DBPort              string
	DBName              string
	DBUser              string
	DBPassword          string
	DBSSLMode           string
	JWTSecret           string
	JWTAccessExpiry     string
	JWTRefreshExpiry    string
	ServerPort          string
	ServerEnv           string
	CORSOrigins         string
	BcryptCost          int
	MaxLoginAttempts    int
	LockDurationMinutes int
	SuperAdminEmail     string
	SuperAdminPassword  string
	SuperAdminUsername  string
}

func Load() *Config {
	bcryptCost, _ := strconv.Atoi(getEnv("BCRYPT_COST", "12"))
	maxAttempts, _ := strconv.Atoi(getEnv("MAX_LOGIN_ATTEMPTS", "5"))
	lockDuration, _ := strconv.Atoi(getEnv("LOCK_DURATION_MINUTES", "30"))

	return &Config{
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBName:              getEnv("DB_NAME", "ispcms"),
		DBUser:              getEnv("DB_USER", "ispcms"),
		DBPassword:          getEnv("DB_PASSWORD", "secret"),
		DBSSLMode:           getEnv("DB_SSL_MODE", "disable"),
		JWTSecret:           getEnv("JWT_SECRET", "change-me-super-secret-key-min-32-chars"),
		JWTAccessExpiry:     getEnv("JWT_ACCESS_EXPIRY", "15m"),
		JWTRefreshExpiry:    getEnv("JWT_REFRESH_EXPIRY", "7d"),
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		ServerEnv:           getEnv("SERVER_ENV", "development"),
		CORSOrigins:         getEnv("CORS_ORIGINS", "http://localhost:3000"),
		BcryptCost:          bcryptCost,
		MaxLoginAttempts:    maxAttempts,
		LockDurationMinutes: lockDuration,
		SuperAdminEmail:     getEnv("SUPER_ADMIN_EMAIL", "admin@ispcms.local"),
		SuperAdminPassword:  getEnv("SUPER_ADMIN_PASSWORD", "Admin@1234"),
		SuperAdminUsername:  getEnv("SUPER_ADMIN_USERNAME", "superadmin"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
