package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/zxchelik/room-booking-service/internal/service"
)

type SlotHandler struct {
	slots *service.SlotService
}

func NewSlotHandler(slots *service.SlotService) *SlotHandler {
	return &SlotHandler{slots: slots}
}

func (h *SlotHandler) List(w http.ResponseWriter, r *http.Request) {
	dateRaw := r.URL.Query().Get("date")
	if dateRaw == "" {
		writeError(w, service.InvalidRequest("date query parameter is required"))
		return
	}

	date, err := time.Parse("2006-01-02", dateRaw)
	if err != nil {
		writeError(w, service.InvalidRequest("date query parameter must be in YYYY-MM-DD format"))
		return
	}

	roomID := chi.URLParam(r, "roomId")
	if _, err := uuid.Parse(roomID); err != nil {
		writeError(w, service.InvalidRequest("roomId must be a valid UUID"))
		return
	}
	slots, err := h.slots.ListAvailable(r.Context(), roomID, date.UTC())
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"slots": slots})
}
