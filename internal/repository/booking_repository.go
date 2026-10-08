package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/zxchelik/room-booking-service/internal/model"
)

type BookingRepository struct{}

func NewBookingRepository() *BookingRepository {
	return &BookingRepository{}
}

func (r *BookingRepository) Create(ctx context.Context, db DBTX, booking model.Booking) (model.Booking, error) {
	var createdAt time.Time
	if err := db.QueryRow(ctx, `
		INSERT INTO bookings (id, slot_id, user_id, status, conference_link)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`, booking.ID, booking.SlotID, booking.UserID, booking.Status, booking.ConferenceLink).Scan(&createdAt); err != nil {
		return model.Booking{}, err
	}
	booking.CreatedAt = &createdAt
	return booking, nil
}

func (r *BookingRepository) GetByID(ctx context.Context, db DBTX, bookingID string) (model.Booking, error) {
	var booking model.Booking
	var conferenceLink sql.NullString
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		SELECT id, slot_id, user_id, status, conference_link, created_at
		FROM bookings
		WHERE id = $1
	`, bookingID).Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &conferenceLink, &createdAt)
	if err != nil {
		if isNoRows(err) {
			return model.Booking{}, ErrNotFound
		}
		return model.Booking{}, err
	}
	if conferenceLink.Valid {
		value := conferenceLink.String
		booking.ConferenceLink = &value
	}
	booking.CreatedAt = &createdAt
	return booking, nil
}

func (r *BookingRepository) Cancel(ctx context.Context, db DBTX, bookingID string) (model.Booking, error) {
	var booking model.Booking
	var conferenceLink sql.NullString
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		UPDATE bookings
		SET status = 'cancelled'
		WHERE id = $1
		RETURNING id, slot_id, user_id, status, conference_link, created_at
	`, bookingID).Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &conferenceLink, &createdAt)
	if err != nil {
		if isNoRows(err) {
			return model.Booking{}, ErrNotFound
		}
		return model.Booking{}, err
	}
	if conferenceLink.Valid {
		value := conferenceLink.String
		booking.ConferenceLink = &value
	}
	booking.CreatedAt = &createdAt
	return booking, nil
}

func (r *BookingRepository) ListAll(ctx context.Context, db DBTX, page, pageSize int) ([]model.Booking, error) {
	rows, err := db.Query(ctx, `
		SELECT id, slot_id, user_id, status, conference_link, created_at
		FROM bookings
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var booking model.Booking
		var conferenceLink sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &conferenceLink, &createdAt); err != nil {
			return nil, err
		}
		if conferenceLink.Valid {
			value := conferenceLink.String
			booking.ConferenceLink = &value
		}
		booking.CreatedAt = &createdAt
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (r *BookingRepository) CountAll(ctx context.Context, db DBTX) (int, error) {
	var total int
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&total)
	return total, err
}

func (r *BookingRepository) ListFutureByUser(ctx context.Context, db DBTX, userID string, now time.Time) ([]model.Booking, error) {
	rows, err := db.Query(ctx, `
		SELECT b.id, b.slot_id, b.user_id, b.status, b.conference_link, b.created_at
		FROM bookings b
		INNER JOIN slots s ON s.id = b.slot_id
		WHERE b.user_id = $1
		  AND s.start_time >= $2
		  AND b.status = 'active'
		ORDER BY s.start_time ASC, b.created_at ASC
	`, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []model.Booking
	for rows.Next() {
		var booking model.Booking
		var conferenceLink sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&booking.ID, &booking.SlotID, &booking.UserID, &booking.Status, &conferenceLink, &createdAt); err != nil {
			return nil, err
		}
		if conferenceLink.Valid {
			value := conferenceLink.String
			booking.ConferenceLink = &value
		}
		booking.CreatedAt = &createdAt
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
