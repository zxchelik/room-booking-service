package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

type fakeDB struct {
	execFn func(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (f fakeDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (f fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected query call")
}

func (f fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected queryrow call")
}

func TestDummyLoginReturnsFixedAdminToken(t *testing.T) {
	t.Parallel()

	var execCalls int
	service := NewAuthService(fakeDB{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			execCalls++
			return pgconn.CommandTag{}, nil
		},
	}, repository.NewUserRepository(), "secret", time.Hour, false)
	service.now = func() time.Time {
		return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	}

	token, err := service.DummyLogin(context.Background(), model.RoleAdmin)
	if err != nil {
		t.Fatalf("DummyLogin() error = %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("expected one upsert call, got %d", execCalls)
	}

	parsed, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		return service.JWTSecret(), nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("unexpected claims type %T", parsed.Claims)
	}
	if claims["user_id"] != DummyAdminID {
		t.Fatalf("expected admin id %s, got %v", DummyAdminID, claims["user_id"])
	}
	if claims["role"] != string(model.RoleAdmin) {
		t.Fatalf("expected admin role, got %v", claims["role"])
	}
}

func TestDummyLoginRejectsUnknownRole(t *testing.T) {
	t.Parallel()

	service := NewAuthService(fakeDB{}, repository.NewUserRepository(), "secret", time.Hour, false)
	if _, err := service.DummyLogin(context.Background(), "manager"); err == nil {
		t.Fatal("expected error for unknown role")
	}
}
