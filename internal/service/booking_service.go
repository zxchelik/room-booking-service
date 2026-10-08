package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
)

type BookingService struct {
	db       *pgxpool.Pool
	bookings *repository.BookingRepository
	slots    *repository.SlotRepository
	now      func() time.Time
}

func NewBookingService(db *pgxpool.Pool, bookings *repository.BookingRepository, slots *repository.SlotRepository) *BookingService {
	return &BookingService{
		db:       db,
		bookings: bookings,
		slots:    slots,
		now:      time.Now,
	}
}

func (s *BookingService) Create(ctx context.Context, claims model.AuthClaims, slotID string, createConferenceLink bool) (model.Booking, error) {
	if claims.Role != model.RoleUser {
		return model.Booking{}, Forbidden("booking is available only for role user")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return model.Booking{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	slot, err := s.slots.LockByID(ctx, tx, slotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Booking{}, SlotNotFound("slot not found")
		}
		return model.Booking{}, err
	}
	if err := validateBookingCreation(claims, slot, s.now().UTC()); err != nil {
		return model.Booking{}, err
	}

	booking := model.Booking{
		ID:     uuid.NewString(),
		SlotID: slotID,
		UserID: claims.UserID,
		Status: model.BookingStatusActive,
	}
	if createConferenceLink {
		link := buildConferenceLink(booking.ID)
		booking.ConferenceLink = &link
	}

	booking, err = s.bookings.Create(ctx, tx, booking)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.Booking{}, SlotAlreadyBooked("slot is already booked")
		}
		return model.Booking{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Booking{}, err
	}
	return booking, nil
}

func (s *BookingService) ListAll(ctx context.Context, claims model.AuthClaims, page, pageSize int) ([]model.Booking, model.Pagination, error) {
	if claims.Role != model.RoleAdmin {
		return nil, model.Pagination{}, Forbidden("admin role is required")
	}
	if err := validatePagination(page, pageSize); err != nil {
		return nil, model.Pagination{}, err
	}

	bookings, err := s.bookings.ListAll(ctx, s.db, page, pageSize)
	if err != nil {
		return nil, model.Pagination{}, err
	}
	total, err := s.bookings.CountAll(ctx, s.db)
	if err != nil {
		return nil, model.Pagination{}, err
	}

	return bookings, model.Pagination{Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *BookingService) ListMy(ctx context.Context, claims model.AuthClaims) ([]model.Booking, error) {
	if claims.Role != model.RoleUser {
		return nil, Forbidden("user role is required")
	}
	return s.bookings.ListFutureByUser(ctx, s.db, claims.UserID, s.now().UTC())
}

func (s *BookingService) Cancel(ctx context.Context, claims model.AuthClaims, bookingID string) (model.Booking, error) {
	if claims.Role != model.RoleUser {
		return model.Booking{}, Forbidden("user role is required")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return model.Booking{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	booking, err := s.bookings.GetByID(ctx, tx, bookingID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Booking{}, BookingNotFound("booking not found")
		}
		return model.Booking{}, err
	}
	if err := validateBookingCancellation(claims, booking); err != nil {
		return model.Booking{}, err
	}
	if booking.Status == model.BookingStatusCancelled {
		if err := tx.Commit(ctx); err != nil {
			return model.Booking{}, err
		}
		return booking, nil
	}

	booking, err = s.bookings.Cancel(ctx, tx, bookingID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Booking{}, BookingNotFound("booking not found")
		}
		return model.Booking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Booking{}, err
	}
	return booking, nil
}

func AsHTTPError(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return &Error{Code: CodeInternalError, Message: http.StatusText(http.StatusInternalServerError), Status: http.StatusInternalServerError, Err: err}
}

func validateBookingCreation(claims model.AuthClaims, slot model.Slot, now time.Time) error {
	if claims.Role != model.RoleUser {
		return Forbidden("booking is available only for role user")
	}
	if slot.Start.Before(now) {
		return InvalidRequest("cannot create booking for a slot in the past")
	}
	return nil
}

func validateBookingCancellation(claims model.AuthClaims, booking model.Booking) error {
	if claims.Role != model.RoleUser {
		return Forbidden("user role is required")
	}
	if booking.UserID != claims.UserID {
		return Forbidden("cannot cancel another user's booking")
	}
	return nil
}

func validatePagination(page, pageSize int) error {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return InvalidRequest("invalid pagination parameters")
	}
	return nil
}

func buildConferenceLink(bookingID string) string {
	return fmt.Sprintf("https://conference.local/bookings/%s", bookingID)
}
