package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultHTTPAddr          = ":8080"
	defaultDatabaseDSN       = "app.db"
	defaultMaxHeaderBytes    = 64 << 10
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type Config struct {
	HTTPAddr           string
	DatabaseDSN        string
	MaxHeaderBytes     int
	ReadHeaderTimeout  time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	ShutdownTimeout    time.Duration
	SessionSecure      bool
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           envOrDefault("HTTP_ADDR", defaultHTTPAddr),
		DatabaseDSN:        envOrDefault("DATABASE_DSN", envOrDefault("DSN", defaultDatabaseDSN)),
		MaxHeaderBytes:     defaultMaxHeaderBytes,
		ReadHeaderTimeout:  defaultReadHeaderTimeout,
		ReadTimeout:        defaultReadTimeout,
		WriteTimeout:       defaultWriteTimeout,
		IdleTimeout:        defaultIdleTimeout,
		ShutdownTimeout:    defaultShutdownTimeout,
		SessionSecure:      envOrDefault("SESSION_SECURE", "false") == "true",
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
	if value := os.Getenv("HTTP_MAX_HEADER_BYTES"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("config: HTTP_MAX_HEADER_BYTES: %w", err)
		}
		cfg.MaxHeaderBytes = parsed
	}

	durations := []struct {
		name   string
		target *time.Duration
	}{
		{"HTTP_READ_HEADER_TIMEOUT", &cfg.ReadHeaderTimeout},
		{"HTTP_READ_TIMEOUT", &cfg.ReadTimeout},
		{"HTTP_WRITE_TIMEOUT", &cfg.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", &cfg.IdleTimeout},
		{"SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
	}
	for _, item := range durations {
		value := os.Getenv(item.name)
		if value == "" {
			continue
		}
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("config: %s: %w", item.name, err)
		}
		*item.target = parsed
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if _, _, err := net.SplitHostPort(c.HTTPAddr); err != nil {
		return fmt.Errorf("config: HTTP_ADDR: %w", err)
	}
	if c.DatabaseDSN == "" {
		return fmt.Errorf("config: DATABASE_DSN must not be empty")
	}
	if c.MaxHeaderBytes <= 0 {
		return fmt.Errorf("config: HTTP_MAX_HEADER_BYTES must be positive")
	}
	googleValues := []string{c.GoogleClientID, c.GoogleClientSecret, c.GoogleRedirectURL}
	configured := 0
	for _, value := range googleValues {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(googleValues) {
		return fmt.Errorf("config: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be configured together")
	}
	for name, value := range map[string]time.Duration{
		"HTTP_READ_HEADER_TIMEOUT": c.ReadHeaderTimeout,
		"HTTP_READ_TIMEOUT":        c.ReadTimeout,
		"HTTP_WRITE_TIMEOUT":       c.WriteTimeout,
		"HTTP_IDLE_TIMEOUT":        c.IdleTimeout,
		"SHUTDOWN_TIMEOUT":         c.ShutdownTimeout,
	} {
		if value <= 0 {
			return fmt.Errorf("config: %s must be positive", name)
		}
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
