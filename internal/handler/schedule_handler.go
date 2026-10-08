package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type ScheduleHandler struct {
	schedules *service.ScheduleService
}

func NewScheduleHandler(schedules *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{schedules: schedules}
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if claims.Role != model.RoleAdmin {
		writeError(w, service.Forbidden("admin role is required"))
		return
	}

	roomID := chi.URLParam(r, "roomId")
	if _, err := uuid.Parse(roomID); err != nil {
		writeError(w, service.InvalidRequest("roomId must be a valid UUID"))
		return
	}
	var req model.Schedule
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	schedule, err := h.schedules.Create(r.Context(), roomID, req)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"schedule": schedule})
}
