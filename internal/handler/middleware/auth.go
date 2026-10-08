package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type contextKey string

const claimsContextKey contextKey = "authClaims"

func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeUnauthorized(w, "missing bearer token")
				return
			}

			tokenString := strings.TrimPrefix(header, "Bearer ")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, service.Unauthorized("invalid token signing method")
				}
				return secret, nil
			})
			if err != nil || !token.Valid {
				writeUnauthorized(w, "invalid token")
				return
			}

			claimsMap, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeUnauthorized(w, "invalid token")
				return
			}

			userID, ok := claimsMap["user_id"].(string)
			if !ok || userID == "" {
				writeUnauthorized(w, "invalid token")
				return
			}
			roleRaw, ok := claimsMap["role"].(string)
			if !ok || roleRaw == "" {
				writeUnauthorized(w, "invalid token")
				return
			}

			claims := model.AuthClaims{
				UserID: userID,
				Role:   model.Role(roleRaw),
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (model.AuthClaims, error) {
	claims, ok := ctx.Value(claimsContextKey).(model.AuthClaims)
	if !ok {
		return model.AuthClaims{}, service.Unauthorized("authentication required")
	}
	return claims, nil
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    service.CodeUnauthorized,
			"message": message,
		},
	})
}
