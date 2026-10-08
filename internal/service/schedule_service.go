package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

type ScheduleService struct {
	db             *pgxpool.Pool
	rooms          *repository.RoomRepository
	schedules      *repository.ScheduleRepository
	slots          *repository.SlotRepository
	slotWindowDays int
	now            func() time.Time
}

func NewScheduleService(db *pgxpool.Pool, rooms *repository.RoomRepository, schedules *repository.ScheduleRepository, slots *repository.SlotRepository, slotWindowDays int) *ScheduleService {
	return &ScheduleService{
		db:             db,
		rooms:          rooms,
		schedules:      schedules,
		slots:          slots,
		slotWindowDays: slotWindowDays,
		now:            time.Now,
	}
}

func (s *ScheduleService) Create(ctx context.Context, roomID string, schedule model.Schedule) (model.Schedule, error) {
	if schedule.RoomID != "" && schedule.RoomID != roomID {
		return model.Schedule{}, InvalidRequest("roomId in path and body must match")
	}

	days, err := normalizeDays(schedule.DaysOfWeek)
	if err != nil {
		return model.Schedule{}, err
	}
	if err := validateScheduleBounds(schedule.StartTime, schedule.EndTime); err != nil {
		return model.Schedule{}, err
	}

	exists, err := s.rooms.Exists(ctx, s.db, roomID)
	if err != nil {
		return model.Schedule{}, err
	}
	if !exists {
		return model.Schedule{}, RoomNotFound("room not found")
	}

	record := repository.ScheduleRecord{
		ID:         uuid.NewString(),
		RoomID:     roomID,
		DaysOfWeek: days,
		StartTime:  schedule.StartTime,
		EndTime:    schedule.EndTime,
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return model.Schedule{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := s.schedules.Create(ctx, tx, record); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Schedule{}, ScheduleExists("schedule for this room already exists and cannot be changed")
		}
		return model.Schedule{}, err
	}

	today := truncateToDay(s.now())
	until := today.AddDate(0, 0, s.slotWindowDays-1)
	slots, err := GenerateSlots(record, today, until)
	if err != nil {
		return model.Schedule{}, err
	}
	if err := s.slots.InsertMany(ctx, tx, slots); err != nil {
		return model.Schedule{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Schedule{}, err
	}

	return model.Schedule{
		ID:         record.ID,
		RoomID:     record.RoomID,
		DaysOfWeek: record.DaysOfWeek,
		StartTime:  record.StartTime,
		EndTime:    record.EndTime,
	}, nil
}

func validateScheduleBounds(start, end string) error {
	startOffset, err := parseClock(start)
	if err != nil {
		return err
	}
	endOffset, err := parseClock(end)
	if err != nil {
		return err
	}
	if startOffset >= endOffset {
		return InvalidRequest("startTime must be before endTime")
	}
	if startOffset%slotDuration != 0 || endOffset%slotDuration != 0 {
		return InvalidRequest("schedule bounds must align to 30-minute slots")
	}
	if (endOffset-startOffset)%slotDuration != 0 {
		return InvalidRequest("schedule duration must be divisible by 30 minutes")
	}
	return nil
}

func normalizeDays(days []int) ([]int, error) {
	if len(days) == 0 {
		return nil, InvalidRequest("daysOfWeek must not be empty")
	}

	unique := make(map[int]struct{}, len(days))
	result := make([]int, 0, len(days))
	for _, day := range days {
		if day < 1 || day > 7 {
			return nil, InvalidRequest("daysOfWeek values must be in range 1..7")
		}
		if _, exists := unique[day]; exists {
			continue
		}
		unique[day] = struct{}{}
		result = append(result, day)
	}
	sort.Ints(result)
	return result, nil
}

func (s *ScheduleService) ensureSlots(ctx context.Context, roomID string, requestedDate time.Time) error {
	record, err := s.schedules.GetByRoomID(ctx, s.db, roomID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}

	targetDay := truncateToDay(requestedDate)
	windowEnd := truncateToDay(s.now()).AddDate(0, 0, s.slotWindowDays-1)
	if windowEnd.After(targetDay) {
		targetDay = windowEnd
	}

	latest, err := s.slots.GetLatestStartByRoom(ctx, s.db, roomID)
	if err != nil {
		return err
	}

	startDay := truncateToDay(s.now())
	if latest != nil {
		startDay = truncateToDay(*latest).AddDate(0, 0, 1)
	}
	if startDay.After(targetDay) {
		return nil
	}

	generated, err := GenerateSlots(record, startDay, targetDay)
	if err != nil {
		return err
	}
	return s.slots.InsertMany(ctx, s.db, generated)
}

func (s *ScheduleService) EnsureSlots(ctx context.Context, roomID string, requestedDate time.Time) error {
	return s.ensureSlots(ctx, roomID, requestedDate)
}
