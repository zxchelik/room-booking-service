package application

import (
	"testing"
	"time"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_TTL", "")
	t.Setenv("AUTO_MIGRATE", "")
	t.Setenv("SLOT_WINDOW_DAYS", "")

	cfg := LoadConfig()
	if cfg.HTTPPort != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.HTTPPort)
	}
	if cfg.SlotWindowDays != 14 {
		t.Fatalf("expected default slot window 14, got %d", cfg.SlotWindowDays)
	}
	if !cfg.AutoMigrate {
		t.Fatal("expected auto migrate to default to true")
	}
}

func TestLoadConfigUsesDatabaseURLOverride(t *testing.T) {
	want := "postgres://user:pass@localhost:5432/custom?sslmode=disable"
	t.Setenv("DATABASE_URL", want)

	cfg := LoadConfig()
	if cfg.DatabaseURL != want {
		t.Fatalf("expected %s, got %s", want, cfg.DatabaseURL)
	}
}

func TestLoadConfigParsesOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("JWT_SECRET", "override-secret")
	t.Setenv("JWT_TTL", "2h")
	t.Setenv("AUTO_MIGRATE", "false")
	t.Setenv("SLOT_WINDOW_DAYS", "21")

	cfg := LoadConfig()
	if cfg.HTTPPort != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.HTTPPort)
	}
	if cfg.JWTSecret != "override-secret" {
		t.Fatalf("expected custom secret, got %s", cfg.JWTSecret)
	}
	if cfg.JWTTTL != 2*time.Hour {
		t.Fatalf("expected TTL 2h, got %s", cfg.JWTTTL)
	}
	if cfg.AutoMigrate {
		t.Fatal("expected auto migrate false")
	}
	if cfg.SlotWindowDays != 21 {
		t.Fatalf("expected slot window 21, got %d", cfg.SlotWindowDays)
	}
}
