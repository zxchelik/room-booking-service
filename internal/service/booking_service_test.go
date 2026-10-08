package service

import (
	"testing"
	"time"

	"github.com/zxchelik/room-booking-service/internal/model"
)

func TestValidateBookingCreationRejectsAdmin(t *testing.T) {
	t.Parallel()

	err := validateBookingCreation(model.AuthClaims{Role: model.RoleAdmin}, model.Slot{}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for admin booking")
	}
}

func TestValidateBookingCreationRejectsPastSlot(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	err := validateBookingCreation(
		model.AuthClaims{Role: model.RoleUser},
		model.Slot{Start: now.Add(-time.Minute)},
		now,
	)
	if err == nil {
		t.Fatal("expected error for past slot")
	}
}

func TestValidateBookingCancellationRejectsForeignBooking(t *testing.T) {
	t.Parallel()

	err := validateBookingCancellation(
		model.AuthClaims{Role: model.RoleUser, UserID: "user-1"},
		model.Booking{UserID: "user-2"},
	)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestValidateBookingCancellationRejectsNonUserRole(t *testing.T) {
	t.Parallel()

	err := validateBookingCancellation(
		model.AuthClaims{Role: model.RoleAdmin, UserID: "user-1"},
		model.Booking{UserID: "user-1"},
	)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestValidatePaginationRejectsPageSizeTooLarge(t *testing.T) {
	t.Parallel()

	if err := validatePagination(1, 101); err == nil {
		t.Fatal("expected pagination validation error")
	}
}

func TestBuildConferenceLink(t *testing.T) {
	t.Parallel()

	got := buildConferenceLink("booking-1")
	want := "https://conference.local/bookings/booking-1"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAsHTTPErrorWrapsGenericError(t *testing.T) {
	t.Parallel()

	appErr := AsHTTPError(assertionError("boom"))
	if appErr.Code != CodeInternalError {
		t.Fatalf("expected internal error code, got %s", appErr.Code)
	}
}

func TestParseClockAndHelpers(t *testing.T) {
	t.Parallel()

	offset, err := parseClock("09:30")
	if err != nil {
		t.Fatalf("parseClock() error = %v", err)
	}
	if offset != 9*time.Hour+30*time.Minute {
		t.Fatalf("unexpected offset %s", offset)
	}

	day := truncateToDay(time.Date(2026, 3, 22, 15, 4, 0, 0, time.FixedZone("UTC+3", 3*3600)))
	if !day.Equal(time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected truncated day %s", day)
	}
	if isoWeekday(time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)) != 7 {
		t.Fatal("expected Sunday to map to 7")
	}
}

type assertionError string

func (e assertionError) Error() string {
	return string(e)
}
