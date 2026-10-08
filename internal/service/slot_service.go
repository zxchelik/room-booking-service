package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

type SlotService struct {
	db        *pgxpool.Pool
	rooms     *repository.RoomRepository
	schedules *repository.ScheduleRepository
	slots     *repository.SlotRepository
	scheduler *ScheduleService
	now       func() time.Time
}

func NewSlotService(db *pgxpool.Pool, rooms *repository.RoomRepository, schedules *repository.ScheduleRepository, slots *repository.SlotRepository, scheduler *ScheduleService) *SlotService {
	return &SlotService{
		db:        db,
		rooms:     rooms,
		schedules: schedules,
		slots:     slots,
		scheduler: scheduler,
		now:       time.Now,
	}
}

func (s *SlotService) ListAvailable(ctx context.Context, roomID string, date time.Time) ([]model.Slot, error) {
	exists, err := s.rooms.Exists(ctx, s.db, roomID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, RoomNotFound("room not found")
	}

	if err := s.scheduler.EnsureSlots(ctx, roomID, date); err != nil {
		return nil, err
	}

	if _, err := s.schedules.GetByRoomID(ctx, s.db, roomID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return []model.Slot{}, nil
		}
		return nil, err
	}

	return s.slots.ListAvailableByRoomAndDate(ctx, s.db, roomID, truncateToDay(date), s.now().UTC())
}
