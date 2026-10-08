package service

import (
	"testing"
	"time"

	"github.com/zxchelik/room-booking-service/internal/repository"
)

func TestNormalizeDaysSortsAndDeduplicates(t *testing.T) {
	t.Parallel()

	got, err := normalizeDays([]int{5, 1, 5, 3})
	if err != nil {
		t.Fatalf("normalizeDays() error = %v", err)
	}

	want := []int{1, 3, 5}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestNormalizeDaysRejectsOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := normalizeDays([]int{0, 1}); err == nil {
		t.Fatal("expected error for invalid day")
	}
}

func TestValidateScheduleBoundsRejectsUnalignedTime(t *testing.T) {
	t.Parallel()

	if err := validateScheduleBounds("09:15", "10:00"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGenerateSlotsCreatesStableUUIDs(t *testing.T) {
	t.Parallel()

	record := repository.ScheduleRecord{
		RoomID:     "room-1",
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}
	day := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	first, err := GenerateSlots(record, day, day)
	if err != nil {
		t.Fatalf("GenerateSlots() error = %v", err)
	}
	second, err := GenerateSlots(record, day, day)
	if err != nil {
		t.Fatalf("GenerateSlots() second error = %v", err)
	}

	if len(first) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(first))
	}
	if first[0].ID != second[0].ID || first[1].ID != second[1].ID {
		t.Fatal("expected stable UUIDs across generations")
	}
	if !first[0].Start.Equal(time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first slot start: %s", first[0].Start)
	}
}

func TestGenerateSlotsSkipsDaysOutsideSchedule(t *testing.T) {
	t.Parallel()

	record := repository.ScheduleRecord{
		RoomID:     "room-1",
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	}
	day := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)

	slots, err := GenerateSlots(record, day, day)
	if err != nil {
		t.Fatalf("GenerateSlots() error = %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected 0 slots, got %d", len(slots))
	}
}
