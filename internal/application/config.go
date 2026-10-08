package application

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const developmentJWTSecret = "development-only-room-booking-secret"

type Config struct {
	AppEnv           string
	HTTPPort         string
	DatabaseURL      string
	JWTSecret        string
	JWTTTL           time.Duration
	AutoMigrate      bool
	SlotWindowDays   int
	EnableDummyLogin bool
}

func LoadConfig() Config {
	appEnv := strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development")))
	dummyLoginDefault := appEnv == "development" || appEnv == "test"
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" && dummyLoginDefault {
		jwtSecret = developmentJWTSecret
	}

	host := getEnv("DB_HOST", "db")
	port := getEnv("DB_PORT", "5432")
	name := getEnv("DB_NAME", "room_booking")
	user := getEnv("DB_USER", "room_booking")
	password := getEnv("DB_PASSWORD", "room_booking")

	databaseURL := getEnv("DATABASE_URL", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, name))

	return Config{
		AppEnv:           appEnv,
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      databaseURL,
		JWTSecret:        jwtSecret,
		JWTTTL:           getDurationEnv("JWT_TTL", 24*time.Hour),
		AutoMigrate:      getBoolEnv("AUTO_MIGRATE", true),
		SlotWindowDays:   getIntEnv("SLOT_WINDOW_DAYS", 14),
		EnableDummyLogin: getBoolEnv("ENABLE_DUMMY_LOGIN", dummyLoginDefault),
	}
}

func (c Config) Validate() error {
	appEnv := strings.ToLower(strings.TrimSpace(c.AppEnv))
	if appEnv == "" {
		appEnv = "development"
	}
	if appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if appEnv == "production" {
		if c.EnableDummyLogin {
			return fmt.Errorf("ENABLE_DUMMY_LOGIN must be false in production")
		}
		jwtSecret := strings.TrimSpace(c.JWTSecret)
		if len([]byte(jwtSecret)) < 32 || jwtSecret == developmentJWTSecret {
			return fmt.Errorf("JWT_SECRET must be set to a non-development secret of at least 32 bytes in production")
		}
	}
	return nil
}

func (c Config) dummyLoginEnabled() bool {
	return c.EnableDummyLogin
}

func (c Config) adminRegistrationEnabled() bool {
	appEnv := strings.ToLower(strings.TrimSpace(c.AppEnv))
	return appEnv == "" || appEnv == "development" || appEnv == "test"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
