package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address        string
	DatabaseURL    string
	WebOrigin      string
	APIPublicURL   string
	CookieSecure   bool
	CookieSameSite string
	CSRFSecret     string
	GoogleClientID string
	GoogleSecret   string
	GitHubClientID string
	GitHubSecret   string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPassword   string
	SMTPFrom       string
}

func Load() (Config, error) {
	cfg := Config{
		Address: env("SERVER_ADDRESS", ":3001"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		WebOrigin:    strings.TrimRight(env("WEB_ORIGIN", "http://localhost:3000"), "/"),
		APIPublicURL: strings.TrimRight(env("API_PUBLIC_URL", "http://localhost:3001"), "/"),

		CSRFSecret:     os.Getenv("CSRF_SECRET"),
		CookieSameSite: env("COOKIE_SAME_SITE", "lax"),

		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID: os.Getenv("GITHUB_CLIENT_ID"), GitHubSecret: os.Getenv("GITHUB_CLIENT_SECRET"),

		SMTPHost: env("SMTP_HOST", "localhost"), SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: env("SMTP_FROM", "Wiant <no-reply@localhost>"),
	}

	var err error
	cfg.CookieSecure, err = boolEnv("COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	cfg.SMTPPort, err = strconv.Atoi(env("SMTP_PORT", "1025"))
	if err != nil {
		return Config{}, fmt.Errorf("SMTP_PORT: %w", err)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.CSRFSecret) < 32 {
		return Config{}, errors.New("CSRF_SECRET must be at least 32 characters")
	}
	if cfg.CookieSameSite != "lax" && cfg.CookieSameSite != "none" {
		return Config{}, errors.New("COOKIE_SAME_SITE must be lax or none")
	}
	if cfg.CookieSameSite == "none" && !cfg.CookieSecure {
		return Config{}, errors.New("COOKIE_SAME_SITE=none requires COOKIE_SECURE=true")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
