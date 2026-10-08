package service

import (
	"context"
	"testing"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

func TestRoomServiceCreateRejectsBlankName(t *testing.T) {
	t.Parallel()

	svc := NewRoomService(fakeDB{}, repository.NewRoomRepository())
	if _, err := svc.Create(context.Background(), model.Room{Name: "   "}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRoomServiceCreateRejectsInvalidCapacity(t *testing.T) {
	t.Parallel()

	svc := NewRoomService(fakeDB{}, repository.NewRoomRepository())
	capacity := 0
	if _, err := svc.Create(context.Background(), model.Room{Name: "Room", Capacity: &capacity}); err == nil {
		t.Fatal("expected validation error")
	}
}
