package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	Port           string
	AllowedOrigins []string
}

func Load() Config {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		host := getEnv("POSTGRES_HOST", "localhost")
		port := getEnv("POSTGRES_PORT", "5432")
		dbName := getEnv("POSTGRES_DB", "taskflow")
		user := getEnv("POSTGRES_USER", "taskflow")
		password := getEnv("POSTGRES_PASSWORD", "taskflow_secret")

		databaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user,
			password,
			host,
			port,
			dbName,
		)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	return Config{
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		Port:           firstNonEmptyEnv("PORT", "API_PORT", "8080"),
		AllowedOrigins: getAllowedOrigins(),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func firstNonEmptyEnv(keys ...string) string {
	lastIndex := len(keys) - 1
	for index, key := range keys {
		if index == lastIndex {
			return key
		}

		if value := os.Getenv(key); value != "" {
			return value
		}
	}

	return ""
}

func getAllowedOrigins() []string {
	rawOrigins := os.Getenv("ALLOWED_ORIGINS")
	if rawOrigins == "" {
		return []string{"http://localhost:3000", "http://localhost:5173"}
	}

	parts := strings.Split(rawOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin != "" {
			origins = append(origins, origin)
		}
	}

	if len(origins) == 0 {
		return []string{"http://localhost:3000", "http://localhost:5173"}
	}

	return origins
}
