package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zxchelik/room-booking-service/internal/model"
)

type SlotRepository struct{}

func NewSlotRepository() *SlotRepository {
	return &SlotRepository{}
}

func (r *SlotRepository) InsertMany(ctx context.Context, db DBTX, slots []model.Slot) error {
	if len(slots) == 0 {
		return nil
	}

	var builder strings.Builder
	args := make([]any, 0, len(slots)*4)
	builder.WriteString(`INSERT INTO slots (id, room_id, start_time, end_time) VALUES `)
	for i, slot := range slots {
		if i > 0 {
			builder.WriteString(", ")
		}
		offset := i * 4
		builder.WriteString(fmt.Sprintf("($%d, $%d, $%d, $%d)", offset+1, offset+2, offset+3, offset+4))
		args = append(args, slot.ID, slot.RoomID, slot.Start, slot.End)
	}
	builder.WriteString(` ON CONFLICT (id) DO NOTHING`)

	_, err := db.Exec(ctx, builder.String(), args...)
	return err
}

func (r *SlotRepository) GetLatestStartByRoom(ctx context.Context, db DBTX, roomID string) (*time.Time, error) {
	var latest sql.NullTime
	if err := db.QueryRow(ctx, `SELECT MAX(start_time) FROM slots WHERE room_id = $1`, roomID).Scan(&latest); err != nil {
		return nil, err
	}
	if !latest.Valid {
		return nil, nil
	}
	value := latest.Time
	return &value, nil
}

func (r *SlotRepository) GetByID(ctx context.Context, db DBTX, slotID string) (model.Slot, error) {
	var slot model.Slot
	err := db.QueryRow(ctx, `
		SELECT id, room_id, start_time, end_time
		FROM slots
		WHERE id = $1
	`, slotID).Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
	if err != nil {
		if isNoRows(err) {
			return model.Slot{}, ErrNotFound
		}
		return model.Slot{}, err
	}
	return slot, nil
}

// LockByID fetches a slot and acquires a row-level lock (SELECT FOR UPDATE).
// Must be called inside a transaction to prevent concurrent bookings on the same slot.
func (r *SlotRepository) LockByID(ctx context.Context, db DBTX, slotID string) (model.Slot, error) {
	var slot model.Slot
	err := db.QueryRow(ctx, `
		SELECT id, room_id, start_time, end_time
		FROM slots
		WHERE id = $1
		FOR UPDATE
	`, slotID).Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End)
	if err != nil {
		if isNoRows(err) {
			return model.Slot{}, ErrNotFound
		}
		return model.Slot{}, err
	}
	return slot, nil
}

func (r *SlotRepository) ListAvailableByRoomAndDate(ctx context.Context, db DBTX, roomID string, date time.Time, now time.Time) ([]model.Slot, error) {
	rows, err := db.Query(ctx, `
		SELECT s.id, s.room_id, s.start_time, s.end_time
		FROM slots s
		LEFT JOIN bookings b
		  ON b.slot_id = s.id
		 AND b.status = 'active'
		WHERE s.room_id = $1
		  AND s.start_time >= $2
		  AND s.start_time < $3
		  AND s.start_time >= $4
		  AND b.id IS NULL
		ORDER BY s.start_time ASC
	`, roomID, date, date.Add(24*time.Hour), now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []model.Slot
	for rows.Next() {
		var slot model.Slot
		if err := rows.Scan(&slot.ID, &slot.RoomID, &slot.Start, &slot.End); err != nil {
			return nil, err
		}
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}
