package config

import "os"

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	TMDBKey        string
	GoogleBooksKey string
	MALClientID    string
	CORSOrigins    string
	Environment    string
	Port           string
}

func Load() *Config {
	return &Config{
		DatabaseURL:    mustEnv("DATABASE_URL"),
		JWTSecret:      mustEnv("JWT_SECRET_KEY"),
		TMDBKey:        os.Getenv("TMDB_API_KEY"),
		GoogleBooksKey: os.Getenv("GOOGLE_BOOKS_API_KEY"),
		MALClientID:    os.Getenv("MAL_CLIENT_ID"),
		CORSOrigins:    envOr("CORS_ORIGINS", "http://localhost:5173"),
		Environment:    envOr("ENVIRONMENT", "development"),
		Port:           envOr("PORT", "8000"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

