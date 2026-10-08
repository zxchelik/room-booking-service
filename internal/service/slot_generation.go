package service

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

const slotDuration = 30 * time.Minute

var slotNamespace = uuid.MustParse("19f71cd3-f64d-49b0-a4f8-9c6ff78208d7")

func GenerateSlots(record repository.ScheduleRecord, fromDate, toDate time.Time) ([]model.Slot, error) {
	if toDate.Before(fromDate) {
		return nil, nil
	}

	startOffset, err := parseClock(record.StartTime)
	if err != nil {
		return nil, err
	}
	endOffset, err := parseClock(record.EndTime)
	if err != nil {
		return nil, err
	}

	var slots []model.Slot
	for current := truncateToDay(fromDate); !current.After(truncateToDay(toDate)); current = current.Add(24 * time.Hour) {
		if !slices.Contains(record.DaysOfWeek, isoWeekday(current)) {
			continue
		}

		dayStart := current.Add(startOffset)
		dayEnd := current.Add(endOffset)
		for slotStart := dayStart; slotStart.Before(dayEnd); slotStart = slotStart.Add(slotDuration) {
			slotEnd := slotStart.Add(slotDuration)
			if slotEnd.After(dayEnd) {
				break
			}
			slotID := uuid.NewSHA1(slotNamespace, []byte(fmt.Sprintf("%s|%s", record.RoomID, slotStart.UTC().Format(time.RFC3339)))).String()
			slots = append(slots, model.Slot{
				ID:     slotID,
				RoomID: record.RoomID,
				Start:  slotStart.UTC(),
				End:    slotEnd.UTC(),
			})
		}
	}

	return slots, nil
}

func parseClock(value string) (time.Duration, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, InvalidRequest("time must be in HH:MM format")
	}
	return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute, nil
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func isoWeekday(t time.Time) int {
	weekday := int(t.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}
