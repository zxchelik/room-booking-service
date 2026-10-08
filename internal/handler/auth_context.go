package handler

import (
	"context"

	handlermw "github.com/zxchelik/room-booking-service/internal/handler/middleware"
	"github.com/zxchelik/room-booking-service/internal/model"
)

func ClaimsFromContext(ctx context.Context) (model.AuthClaims, error) {
	return handlermw.ClaimsFromContext(ctx)
}
