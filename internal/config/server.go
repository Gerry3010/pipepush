package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServerConfig holds all server configuration loaded from environment variables.
type ServerConfig struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiry    time.Duration
	BaseURL      string
	VAPIDPublic  string
	VAPIDPrivate string
	VAPIDEmail   string
	// FCM is optional — empty means FCM is disabled
	FCMServiceAccountJSON string
	// StaticDir, if set, serves the built web app (SPA) from this directory.
	StaticDir string
}

// LoadServerConfig loads config from environment variables.
func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		BaseURL:               getEnv("BASE_URL", "http://localhost:8080"),
		VAPIDPublic:           os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivate:          os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDEmail:            os.Getenv("VAPID_EMAIL"),
		FCMServiceAccountJSON: os.Getenv("FCM_SERVICE_ACCOUNT_JSON"),
		StaticDir:             os.Getenv("STATIC_DIR"),
	}

	expirySecs := getEnv("JWT_EXPIRY_SECONDS", strconv.Itoa(int((24 * time.Hour).Seconds())))
	secs, err := strconv.Atoi(expirySecs)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_SECONDS: %w", err)
	}
	cfg.JWTExpiry = time.Duration(secs) * time.Second

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// VAPIDEnabled reports whether Web Push VAPID keys are configured.
func (c *ServerConfig) VAPIDEnabled() bool {
	return c.VAPIDPublic != "" && c.VAPIDPrivate != ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
