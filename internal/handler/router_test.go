package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
	"github.com/zxchelik/room-booking-service/internal/service"
)

func TestRouterInfoEndpoint(t *testing.T) {
	t.Parallel()

	router := newRouterForTests(t)
	req := httptest.NewRequest(http.MethodGet, "/_info", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterProtectedEndpointRequiresAuth(t *testing.T) {
	t.Parallel()

	router := newRouterForTests(t)
	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRoomsCreateRejectsNonAdmin(t *testing.T) {
	t.Parallel()

	router, token := newRouterAndUserToken(t)
	req := httptest.NewRequest(http.MethodPost, "/rooms/create", bytes.NewBufferString(`{"name":"Room A"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSlotsListRejectsMissingDate(t *testing.T) {
	t.Parallel()

	router, token := newRouterAndUserToken(t)
	req := httptest.NewRequest(http.MethodGet, "/rooms/11111111-1111-1111-1111-111111111111/slots/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestBookingsCreateRejectsInvalidUUID(t *testing.T) {
	t.Parallel()

	router, token := newRouterAndUserToken(t)
	req := httptest.NewRequest(http.MethodPost, "/bookings/create", bytes.NewBufferString(`{"slotId":"bad-id"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestScheduleCreateRejectsInvalidRoomUUID(t *testing.T) {
	t.Parallel()

	auth := service.NewAuthService(fakeDBForHandler{}, repository.NewUserRepository(), "secret", time.Hour, false)
	token, err := auth.DummyLogin(context.Background(), model.RoleAdmin)
	if err != nil {
		t.Fatalf("DummyLogin() error = %v", err)
	}

	router := NewRouter(Services{Auth: auth}, false)
	req := httptest.NewRequest(http.MethodPost, "/rooms/not-a-uuid/schedule/create", bytes.NewBufferString(`{"roomId":"not-a-uuid","daysOfWeek":[1],"startTime":"09:00","endTime":"10:00"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestBookingsCancelRejectsInvalidBookingUUID(t *testing.T) {
	t.Parallel()

	router, token := newRouterAndUserToken(t)
	req := httptest.NewRequest(http.MethodPost, "/bookings/not-a-uuid/cancel", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

type fakeDBForHandler struct{}

func (fakeDBForHandler) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (fakeDBForHandler) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected query call")
}

func (fakeDBForHandler) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected queryrow call")
}

func newRouterForTests(t *testing.T) http.Handler {
	t.Helper()

	auth := service.NewAuthService(fakeDBForHandler{}, repository.NewUserRepository(), "secret", time.Hour, false)
	return NewRouter(Services{Auth: auth}, false)
}

func newRouterAndUserToken(t *testing.T) (http.Handler, string) {
	t.Helper()

	auth := service.NewAuthService(fakeDBForHandler{}, repository.NewUserRepository(), "secret", time.Hour, false)
	token, err := auth.DummyLogin(context.Background(), model.RoleUser)
	if err != nil {
		t.Fatalf("DummyLogin() error = %v", err)
	}
	return NewRouter(Services{Auth: auth}, false), token
}
