package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/oziev02/help-desk/internal/domain"
)

const maxJSONBody = 1 << 20 // 1 MiB

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"

	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		code = "not_found"
		message = "resource not found"
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusUnauthorized
		code = "unauthorized"
		message = "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		code = "forbidden"
		message = "forbidden"
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		code = "invalid_input"
		message = "invalid input"
	case errors.Is(err, domain.ErrInvalidTransition):
		status = http.StatusBadRequest
		code = "invalid_transition"
		message = "invalid status transition"
	case errors.Is(err, domain.ErrInvalidCredentials):
		status = http.StatusUnauthorized
		code = "invalid_credentials"
		message = "invalid credentials"
	case errors.Is(err, domain.ErrSelfLink):
		status = http.StatusBadRequest
		code = "self_link"
		message = "cannot link ticket to itself"
	case errors.Is(err, domain.ErrEmailTaken):
		status = http.StatusConflict
		code = "email_taken"
		message = "email already taken"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		code = "conflict"
		message = "conflict"
	}

	var resp ErrorResponse
	resp.Error.Code = code
	resp.Error.Message = message
	writeJSON(w, status, resp)
}

func respondError(w http.ResponseWriter, logger *slog.Logger, msg string, err error) {
	logHandlerError(logger, msg, err)
	writeError(w, err)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxJSONBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(body) > maxJSONBody {
		return domain.ErrInvalidInput
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func logHandlerError(logger *slog.Logger, msg string, err error) {
	if logger == nil {
		return
	}
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrForbidden) ||
		errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrInvalidTransition) ||
		errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrUnauthorized) ||
		errors.Is(err, domain.ErrEmailTaken) || errors.Is(err, domain.ErrConflict) ||
		errors.Is(err, domain.ErrSelfLink) {
		return
	}
	logger.Error(msg, "err", err)
}
