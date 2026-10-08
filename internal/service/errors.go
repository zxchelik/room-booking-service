package service

import "fmt"

const (
	CodeInvalidRequest  = "INVALID_REQUEST"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeNotFound        = "NOT_FOUND"
	CodeRoomNotFound    = "ROOM_NOT_FOUND"
	CodeSlotNotFound    = "SLOT_NOT_FOUND"
	CodeSlotAlreadyBook = "SLOT_ALREADY_BOOKED"
	CodeBookingNotFound = "BOOKING_NOT_FOUND"
	CodeForbidden       = "FORBIDDEN"
	CodeScheduleExists  = "SCHEDULE_EXISTS"
	CodeInternalError   = "INTERNAL_ERROR"
)

type Error struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func InvalidRequest(message string) *Error {
	return &Error{Code: CodeInvalidRequest, Message: message, Status: 400}
}

func Unauthorized(message string) *Error {
	return &Error{Code: CodeUnauthorized, Message: message, Status: 401}
}

func RoomNotFound(message string) *Error {
	return &Error{Code: CodeRoomNotFound, Message: message, Status: 404}
}

func SlotNotFound(message string) *Error {
	return &Error{Code: CodeSlotNotFound, Message: message, Status: 404}
}

func BookingNotFound(message string) *Error {
	return &Error{Code: CodeBookingNotFound, Message: message, Status: 404}
}

func Forbidden(message string) *Error {
	return &Error{Code: CodeForbidden, Message: message, Status: 403}
}

func ScheduleExists(message string) *Error {
	return &Error{Code: CodeScheduleExists, Message: message, Status: 409}
}

func SlotAlreadyBooked(message string) *Error {
	return &Error{Code: CodeSlotAlreadyBook, Message: message, Status: 409}
}
