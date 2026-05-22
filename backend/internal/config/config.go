package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv             string
	ServerAddr         string
	DatabaseDSN        string
	JWTSecret          string
	EncryptionKey      string
	CORSAllowedOrigins []string
}

func Load() Config {
	return Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		ServerAddr:         getEnv("SERVER_ADDR", ":8080"),
		DatabaseDSN:        getEnv("DATABASE_DSN", "host=127.0.0.1 user=postgres password=postgres dbname=singbox_manager port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		EncryptionKey:      getEnv("ENCRYPTION_KEY", "replace-with-32-byte-secret-key"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}
}

func getEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
