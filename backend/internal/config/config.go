package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	AppPort            string
	DatabasePath       string
	JWTSecret          string
	JWTExpiration      time.Duration
	AdminUsername      string
	AdminPassword      string
	EncryptionKey      string
	CORSAllowedOrigins []string
	EnableSwagger      bool
	RegistryContainer  string
	RegistryGCMode     string
	LoginRateLimit     int
	LoginRateWindow    time.Duration
	StaticDir          string
}

func Load() (*Config, error) {
	jwtExp, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION: %w", err)
	}

	loginWindow, err := time.ParseDuration(getEnv("LOGIN_RATE_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid LOGIN_RATE_WINDOW: %w", err)
	}

	loginLimit, err := strconv.Atoi(getEnv("LOGIN_RATE_LIMIT", "10"))
	if err != nil || loginLimit < 1 {
		loginLimit = 10
	}

	encKey := getEnv("ENCRYPTION_KEY", "")
	if len(encKey) < 16 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be at least 16 characters")
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" || jwtSecret == "change-this-secret" {
		if getEnv("APP_ENV", "production") == "production" {
			// Allow default in compose for first boot, but warn via empty check in main
		}
	}
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	origins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8080"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "production"),
		AppPort:            getEnv("APP_PORT", "8080"),
		DatabasePath:       getEnv("DATABASE_PATH", "/data/regpilot.db"),
		JWTSecret:          jwtSecret,
		JWTExpiration:      jwtExp,
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "change-me"),
		EncryptionKey:      encKey,
		CORSAllowedOrigins: origins,
		EnableSwagger:      getEnv("ENABLE_SWAGGER", "false") == "true",
		RegistryContainer:  getEnv("REGISTRY_CONTAINER_NAME", ""),
		RegistryGCMode:     getEnv("REGISTRY_GC_MODE", "local"),
		LoginRateLimit:     loginLimit,
		LoginRateWindow:    loginWindow,
		StaticDir:          getEnv("STATIC_DIR", "/app/static"),
	}

	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_USERNAME and ADMIN_PASSWORD are required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
