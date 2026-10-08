package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/zxchelik/room-booking-service/internal/model"
)

type RoomRepository struct{}

func NewRoomRepository() *RoomRepository {
	return &RoomRepository{}
}

func (r *RoomRepository) Create(ctx context.Context, db DBTX, room model.Room) (model.Room, error) {
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		INSERT INTO rooms (id, name, description, capacity)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`, room.ID, room.Name, room.Description, room.Capacity).Scan(&createdAt)
	if err != nil {
		return model.Room{}, err
	}
	room.CreatedAt = &createdAt
	return room, nil
}

func (r *RoomRepository) List(ctx context.Context, db DBTX) ([]model.Room, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, description, capacity, created_at
		FROM rooms
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		var room model.Room
		var description sql.NullString
		var capacity sql.NullInt32
		var createdAt time.Time
		if err := rows.Scan(&room.ID, &room.Name, &description, &capacity, &createdAt); err != nil {
			return nil, err
		}
		if description.Valid {
			value := description.String
			room.Description = &value
		}
		if capacity.Valid {
			value := int(capacity.Int32)
			room.Capacity = &value
		}
		room.CreatedAt = &createdAt
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}

func (r *RoomRepository) Exists(ctx context.Context, db DBTX, roomID string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = $1)`, roomID).Scan(&exists)
	return exists, err
}
