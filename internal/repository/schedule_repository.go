package repository

import "context"

type ScheduleRecord struct {
	ID         string
	RoomID     string
	DaysOfWeek []int
	StartTime  string
	EndTime    string
}

type ScheduleRepository struct{}

func NewScheduleRepository() *ScheduleRepository {
	return &ScheduleRepository{}
}

func (r *ScheduleRepository) Create(ctx context.Context, db DBTX, schedule ScheduleRecord) (ScheduleRecord, error) {
	var days []int16
	for _, day := range schedule.DaysOfWeek {
		days = append(days, int16(day))
	}

	if err := db.QueryRow(ctx, `
		INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, schedule.ID, schedule.RoomID, days, schedule.StartTime, schedule.EndTime).Scan(&schedule.ID); err != nil {
		return ScheduleRecord{}, err
	}
	return schedule, nil
}

func (r *ScheduleRepository) GetByRoomID(ctx context.Context, db DBTX, roomID string) (ScheduleRecord, error) {
	row := db.QueryRow(ctx, `
		SELECT id, room_id, days_of_week, to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI')
		FROM schedules
		WHERE room_id = $1
	`, roomID)

	var record ScheduleRecord
	var days []int16
	if err := row.Scan(&record.ID, &record.RoomID, &days, &record.StartTime, &record.EndTime); err != nil {
		if isNoRows(err) {
			return ScheduleRecord{}, ErrNotFound
		}
		return ScheduleRecord{}, err
	}

	record.DaysOfWeek = make([]int, 0, len(days))
	for _, day := range days {
		record.DaysOfWeek = append(record.DaysOfWeek, int(day))
	}
	return record, nil
}
