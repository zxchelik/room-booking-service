package handler

import (
	"net/http"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type RoomHandler struct {
	rooms *service.RoomService
}

func NewRoomHandler(rooms *service.RoomService) *RoomHandler {
	return &RoomHandler{rooms: rooms}
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"rooms": rooms})
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, err := ClaimsFromContext(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if claims.Role != model.RoleAdmin {
		writeError(w, service.Forbidden("admin role is required"))
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
		Capacity    *int    `json:"capacity"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	room, err := h.rooms.Create(r.Context(), model.Room{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"room": room})
}
