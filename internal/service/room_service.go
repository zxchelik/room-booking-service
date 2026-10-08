package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

type RoomService struct {
	rooms *repository.RoomRepository
	db    repository.DBTX
	now   func() time.Time
}

func NewRoomService(db repository.DBTX, rooms *repository.RoomRepository) *RoomService {
	return &RoomService{
		db:    db,
		rooms: rooms,
		now:   time.Now,
	}
}

func (s *RoomService) Create(ctx context.Context, room model.Room) (model.Room, error) {
	room.Name = strings.TrimSpace(room.Name)
	if room.Name == "" {
		return model.Room{}, InvalidRequest("name is required")
	}
	if room.Capacity != nil && *room.Capacity <= 0 {
		return model.Room{}, InvalidRequest("capacity must be greater than zero")
	}

	room.ID = uuid.NewString()
	now := s.now().UTC()
	room.CreatedAt = &now
	return s.rooms.Create(ctx, s.db, room)
}

func (s *RoomService) List(ctx context.Context) ([]model.Room, error) {
	return s.rooms.List(ctx, s.db)
}
