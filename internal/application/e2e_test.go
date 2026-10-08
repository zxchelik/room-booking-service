package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/zxchelik/room-booking-service/internal/service"
)

func TestE2ECreateRoomScheduleAndBooking(t *testing.T) {
	t.Parallel()

	client, baseURL, cleanup := startTestServer(t)
	defer cleanup()

	adminToken := dummyLogin(t, client, baseURL, "admin")
	userToken := dummyLogin(t, client, baseURL, "user")

	roomID := createRoom(t, client, baseURL, adminToken)
	createSchedule(t, client, baseURL, adminToken, roomID)
	slotID := firstSlotID(t, client, baseURL, userToken, roomID)

	resp := doJSON(t, client, http.MethodPost, baseURL+"/bookings/create", userToken, map[string]any{
		"slotId": slotID,
	})
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		Booking struct {
			ID     string `json:"id"`
			SlotID string `json:"slotId"`
			UserID string `json:"userId"`
			Status string `json:"status"`
		} `json:"booking"`
	}
	decodeResponse(t, resp, &body)

	if body.Booking.ID == "" {
		t.Fatal("expected booking id")
	}
	if body.Booking.SlotID != slotID {
		t.Fatalf("expected slotId %s, got %s", slotID, body.Booking.SlotID)
	}
	if body.Booking.UserID != service.DummyUserID {
		t.Fatalf("expected user id %s, got %s", service.DummyUserID, body.Booking.UserID)
	}
	if body.Booking.Status != "active" {
		t.Fatalf("expected active status, got %s", body.Booking.Status)
	}
}

func TestE2ECancelBookingIsIdempotent(t *testing.T) {
	t.Parallel()

	client, baseURL, cleanup := startTestServer(t)
	defer cleanup()

	adminToken := dummyLogin(t, client, baseURL, "admin")
	userToken := dummyLogin(t, client, baseURL, "user")

	roomID := createRoom(t, client, baseURL, adminToken)
	createSchedule(t, client, baseURL, adminToken, roomID)
	slotID := firstSlotID(t, client, baseURL, userToken, roomID)
	bookingID := createBooking(t, client, baseURL, userToken, slotID)

	cancelURL := baseURL + "/bookings/" + bookingID + "/cancel"
	first := doJSON(t, client, http.MethodPost, cancelURL, userToken, nil)
	defer func() {
		_ = first.Body.Close()
	}()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on first cancel, got %d", first.StatusCode)
	}

	var firstBody struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
	}
	decodeResponse(t, first, &firstBody)
	if firstBody.Booking.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %s", firstBody.Booking.Status)
	}

	second := doJSON(t, client, http.MethodPost, cancelURL, userToken, nil)
	defer func() {
		_ = second.Body.Close()
	}()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on second cancel, got %d", second.StatusCode)
	}

	var secondBody struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
	}
	decodeResponse(t, second, &secondBody)
	if secondBody.Booking.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %s", secondBody.Booking.Status)
	}
}

func startTestServer(t *testing.T) (*http.Client, string, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()
	container, err := startPostgresContainer(ctx)
	if err != nil {
		t.Skipf("docker is unavailable for e2e tests: %v", err)
	}

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	app, err := New(ctx, Config{
		HTTPPort:         "8080",
		AppEnv:           "test",
		DatabaseURL:      connectionString,
		JWTSecret:        "test-secret",
		JWTTTL:           time.Hour,
		AutoMigrate:      true,
		SlotWindowDays:   14,
		EnableDummyLogin: true,
	})
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("create app: %v", err)
	}

	server := httptest.NewServer(app.Router)
	cleanup := func() {
		server.Close()
		app.Close()
		_ = container.Terminate(ctx)
	}
	return server.Client(), server.URL, cleanup
}

func startPostgresContainer(ctx context.Context) (container *tcpostgres.PostgresContainer, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()

	return tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("room_booking"),
		tcpostgres.WithUsername("room_booking"),
		tcpostgres.WithPassword("room_booking"),
		tcpostgres.BasicWaitStrategies(),
	)
}

func dummyLogin(t *testing.T, client *http.Client, baseURL, role string) string {
	t.Helper()

	resp := doJSON(t, client, http.MethodPost, baseURL+"/dummyLogin", "", map[string]string{"role": role})
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dummyLogin expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Token string `json:"token"`
	}
	decodeResponse(t, resp, &body)
	return body.Token
}

func createRoom(t *testing.T, client *http.Client, baseURL, token string) string {
	t.Helper()

	resp := doJSON(t, client, http.MethodPost, baseURL+"/rooms/create", token, map[string]any{
		"name":        "Room A",
		"description": "Focus room",
		"capacity":    6,
	})
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("rooms/create expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
	}
	decodeResponse(t, resp, &body)
	return body.Room.ID
}

func createSchedule(t *testing.T, client *http.Client, baseURL, token, roomID string) {
	t.Helper()

	targetDate := time.Now().UTC().Add(24 * time.Hour)
	resp := doJSON(t, client, http.MethodPost, baseURL+"/rooms/"+roomID+"/schedule/create", token, map[string]any{
		"roomId":     roomID,
		"daysOfWeek": []int{isoWeekday(targetDate)},
		"startTime":  "09:00",
		"endTime":    "10:00",
	})
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("schedule/create expected 201, got %d", resp.StatusCode)
	}
}

func firstSlotID(t *testing.T, client *http.Client, baseURL, token, roomID string) string {
	t.Helper()

	targetDate := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02")
	resp := doJSON(t, client, http.MethodGet, baseURL+"/rooms/"+roomID+"/slots/list?date="+targetDate, token, nil)
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("slots/list expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Slots []struct {
			ID string `json:"id"`
		} `json:"slots"`
	}
	decodeResponse(t, resp, &body)
	if len(body.Slots) == 0 {
		t.Fatal("expected at least one slot")
	}
	return body.Slots[0].ID
}

func createBooking(t *testing.T, client *http.Client, baseURL, token, slotID string) string {
	t.Helper()

	resp := doJSON(t, client, http.MethodPost, baseURL+"/bookings/create", token, map[string]any{"slotId": slotID})
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("bookings/create expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		Booking struct {
			ID string `json:"id"`
		} `json:"booking"`
	}
	decodeResponse(t, resp, &body)
	return body.Booking.ID
}

func doJSON(t *testing.T, client *http.Client, method, url, token string, body any) *http.Response {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeResponse(t *testing.T, resp *http.Response, target any) {
	t.Helper()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func isoWeekday(t time.Time) int {
	weekday := int(t.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}
