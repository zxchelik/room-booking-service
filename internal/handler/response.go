package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/zxchelik/room-booking-service/internal/service"
)

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	appErr := service.AsHTTPError(err)
	respondJSON(w, appErr.Status, errorEnvelope{
		Error: errorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return service.InvalidRequest("request body is required")
		}
		return service.InvalidRequest("invalid request body")
	}
	if decoder.More() {
		return service.InvalidRequest("request body must contain a single JSON object")
	}
	return nil
}
