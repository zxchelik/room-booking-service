package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/zxchelik/room-booking-service/internal/service"
)

type BookingHandler struct {
	bookings *service.BookingService
}

func NewBookingHandler(bookings *service.BookingService) *BookingHandler {
	return &BookingHandler{bookings: bookings}
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	var req struct {
		SlotID               string `json:"slotId"`
		CreateConferenceLink bool   `json:"createConferenceLink"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	if req.SlotID == "" {
		writeError(w, service.InvalidRequest("slotId is required"))
		return
	}
	if _, err := uuid.Parse(req.SlotID); err != nil {
		writeError(w, service.InvalidRequest("slotId must be a valid UUID"))
		return
	}

	booking, err := h.bookings.Create(r.Context(), claims, req.SlotID, req.CreateConferenceLink)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"booking": booking})
}

func (h *BookingHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, service.InvalidRequest("page must be a positive integer"))
			return
		}
		page = value
	}

	pageSize := 20
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, service.InvalidRequest("pageSize must be a positive integer"))
			return
		}
		pageSize = value
	}

	bookings, pagination, err := h.bookings.ListAll(r.Context(), claims, page, pageSize)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"bookings":   bookings,
		"pagination": pagination,
	})
}

func (h *BookingHandler) ListMy(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	bookings, err := h.bookings.ListMy(r.Context(), claims)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}

func (h *BookingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	bookingID := chi.URLParam(r, "bookingId")
	if _, err := uuid.Parse(bookingID); err != nil {
		writeError(w, service.InvalidRequest("bookingId must be a valid UUID"))
		return
	}
	booking, err := h.bookings.Cancel(r.Context(), claims, bookingID)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"booking": booking})
}
