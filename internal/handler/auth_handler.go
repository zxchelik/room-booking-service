package handler

import (
	"net/http"

	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) DummyLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role model.Role `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	token, err := h.auth.DummyLogin(r.Context(), req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string     `json:"email"`
		Password string     `json:"password"`
		Role     model.Role `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	user, err := h.auth.Register(r.Context(), req.Email, req.Password, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}
