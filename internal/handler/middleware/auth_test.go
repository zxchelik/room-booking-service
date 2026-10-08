package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/zxchelik/room-booking-service/internal/model"
)

func TestRequireAuthRejectsMissingBearerToken(t *testing.T) {
	t.Parallel()

	handler := RequireAuth([]byte("secret"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthStoresClaimsInContext(t *testing.T) {
	t.Parallel()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-1",
		"role":    "user",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	handler := RequireAuth([]byte("secret"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ClaimsFromContext(r.Context())
		if err != nil {
			t.Fatalf("ClaimsFromContext() error = %v", err)
		}
		if claims != (model.AuthClaims{UserID: "user-1", Role: model.RoleUser}) {
			t.Fatalf("unexpected claims: %+v", claims)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestClaimsFromContextWithoutClaims(t *testing.T) {
	t.Parallel()

	if _, err := ClaimsFromContext(context.Background()); err == nil {
		t.Fatal("expected authentication error")
	}
}

func TestWriteUnauthorizedProducesJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	writeUnauthorized(rec, "invalid token")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var body map[string]map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"]["code"] != "UNAUTHORIZED" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
