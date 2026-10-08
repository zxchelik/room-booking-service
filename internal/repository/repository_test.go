package repository

import (
	"context"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"

	"github.com/zxchelik/room-booking-service/internal/model"
)

func TestRoomRepositoryList(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{"id", "name", "description", "capacity", "created_at"}).
		AddRow("room-1", "Alpha", "Quiet room", 6, now)
	mock.ExpectQuery("SELECT id, name, description, capacity, created_at FROM rooms").WillReturnRows(rows)

	repo := NewRoomRepository()
	rooms, err := repo.List(context.Background(), mock)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}
	if rooms[0].ID != "room-1" || rooms[0].Name != "Alpha" {
		t.Fatalf("unexpected room: %+v", rooms[0])
	}
}

func TestRoomRepositoryExists(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs("room-1").WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(true),
	)

	exists, err := NewRoomRepository().Exists(context.Background(), mock, "room-1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatal("expected room to exist")
	}
}

func TestScheduleRepositoryGetByRoomID(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "room_id", "days_of_week", "to_char", "to_char"}).
		AddRow("schedule-1", "room-1", []int16{1, 3, 5}, "09:00", "12:00")
	mock.ExpectQuery("SELECT id, room_id, days_of_week").WithArgs("room-1").WillReturnRows(rows)

	repo := NewScheduleRepository()
	record, err := repo.GetByRoomID(context.Background(), mock, "room-1")
	if err != nil {
		t.Fatalf("GetByRoomID() error = %v", err)
	}
	if record.StartTime != "09:00" || record.EndTime != "12:00" {
		t.Fatalf("unexpected time range: %+v", record)
	}
	if len(record.DaysOfWeek) != 3 || record.DaysOfWeek[1] != 3 {
		t.Fatalf("unexpected days: %+v", record.DaysOfWeek)
	}
}

func TestScheduleRepositoryCreate(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO schedules").WithArgs("schedule-1", "room-1", []int16{1, 3}, "09:00", "12:00").WillReturnRows(
		pgxmock.NewRows([]string{"id"}).AddRow("schedule-1"),
	)

	record, err := NewScheduleRepository().Create(context.Background(), mock, ScheduleRecord{
		ID:         "schedule-1",
		RoomID:     "room-1",
		DaysOfWeek: []int{1, 3},
		StartTime:  "09:00",
		EndTime:    "12:00",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if record.ID != "schedule-1" {
		t.Fatalf("unexpected schedule: %+v", record)
	}
}

func TestSlotRepositoryListAvailableByRoomAndDate(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	date := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{"id", "room_id", "start_time", "end_time"}).
		AddRow("slot-1", "room-1", date.Add(9*time.Hour), date.Add(9*time.Hour+30*time.Minute))
	mock.ExpectQuery("SELECT s.id, s.room_id, s.start_time, s.end_time").WithArgs("room-1", date, date.Add(24*time.Hour), date.Add(-time.Hour)).WillReturnRows(rows)

	repo := NewSlotRepository()
	slots, err := repo.ListAvailableByRoomAndDate(context.Background(), mock, "room-1", date, date.Add(-time.Hour))
	if err != nil {
		t.Fatalf("ListAvailableByRoomAndDate() error = %v", err)
	}
	if len(slots) != 1 || slots[0].ID != "slot-1" {
		t.Fatalf("unexpected slots: %+v", slots)
	}
}

func TestSlotRepositoryGetByIDAndLatest(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	start := time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	mock.ExpectQuery("SELECT id, room_id, start_time, end_time").WithArgs("slot-1").WillReturnRows(
		pgxmock.NewRows([]string{"id", "room_id", "start_time", "end_time"}).AddRow("slot-1", "room-1", start, end),
	)
	mock.ExpectQuery("SELECT MAX\\(start_time\\) FROM slots").WithArgs("room-1").WillReturnRows(
		pgxmock.NewRows([]string{"max"}).AddRow(start),
	)

	repo := NewSlotRepository()
	slot, err := repo.GetByID(context.Background(), mock, "slot-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if slot.ID != "slot-1" {
		t.Fatalf("unexpected slot: %+v", slot)
	}

	latest, err := repo.GetLatestStartByRoom(context.Background(), mock, "room-1")
	if err != nil {
		t.Fatalf("GetLatestStartByRoom() error = %v", err)
	}
	if latest == nil || !latest.Equal(start) {
		t.Fatalf("unexpected latest slot time: %v", latest)
	}
}

func TestBookingRepositoryCreateAndCount(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("INSERT INTO bookings").WithArgs("booking-1", "slot-1", "user-1", model.BookingStatusActive, (*string)(nil)).WillReturnRows(
		pgxmock.NewRows([]string{"created_at"}).AddRow(now),
	)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM bookings").WillReturnRows(
		pgxmock.NewRows([]string{"count"}).AddRow(3),
	)

	repo := NewBookingRepository()
	booking, err := repo.Create(context.Background(), mock, model.Booking{
		ID:     "booking-1",
		SlotID: "slot-1",
		UserID: "user-1",
		Status: model.BookingStatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if booking.CreatedAt == nil || !booking.CreatedAt.Equal(now) {
		t.Fatalf("unexpected created_at: %+v", booking.CreatedAt)
	}

	total, err := repo.CountAll(context.Background(), mock)
	if err != nil {
		t.Fatalf("CountAll() error = %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 bookings, got %d", total)
	}
}

func TestBookingRepositoryListFutureByUser(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{"id", "slot_id", "user_id", "status", "conference_link", "created_at"}).
		AddRow("booking-1", "slot-1", "user-1", "active", nil, now)
	mock.ExpectQuery("SELECT b.id, b.slot_id, b.user_id, b.status").WithArgs("user-1", now).WillReturnRows(rows)

	repo := NewBookingRepository()
	bookings, err := repo.ListFutureByUser(context.Background(), mock, "user-1", now)
	if err != nil {
		t.Fatalf("ListFutureByUser() error = %v", err)
	}
	if len(bookings) != 1 || bookings[0].ID != "booking-1" {
		t.Fatalf("unexpected bookings: %+v", bookings)
	}
}

func TestBookingRepositoryGetCancelAndListAll(t *testing.T) {
	t.Parallel()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT id, slot_id, user_id, status, conference_link, created_at FROM bookings WHERE id = \\$1").
		WithArgs("booking-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "slot_id", "user_id", "status", "conference_link", "created_at"}).
			AddRow("booking-1", "slot-1", "user-1", "active", "https://conference.local/bookings/booking-1", now))
	mock.ExpectQuery("UPDATE bookings").WithArgs("booking-1").WillReturnRows(
		pgxmock.NewRows([]string{"id", "slot_id", "user_id", "status", "conference_link", "created_at"}).
			AddRow("booking-1", "slot-1", "user-1", "cancelled", nil, now),
	)
	mock.ExpectQuery("SELECT id, slot_id, user_id, status, conference_link, created_at FROM bookings").
		WithArgs(20, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "slot_id", "user_id", "status", "conference_link", "created_at"}).
			AddRow("booking-1", "slot-1", "user-1", "cancelled", nil, now))

	repo := NewBookingRepository()
	booking, err := repo.GetByID(context.Background(), mock, "booking-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if booking.ConferenceLink == nil {
		t.Fatal("expected conference link")
	}

	cancelled, err := repo.Cancel(context.Background(), mock, "booking-1")
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != model.BookingStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}

	bookings, err := repo.ListAll(context.Background(), mock, 1, 20)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(bookings))
	}
}
