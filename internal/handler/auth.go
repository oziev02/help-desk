package handler

import (
	"log/slog"
	"net/http"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/handler/dto"
	"github.com/oziev02/help-desk/internal/service"
)

type AuthHandler struct {
	auth   *service.AuthService
	logger *slog.Logger
}

func NewAuthHandler(auth *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "auth handler", domain.ErrInvalidInput)
		return
	}
	token, user, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respondError(w, h.logger, "auth handler", err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AuthResponse{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresIn:   token.ExpiresIn,
		User:        toUserResponse(user),
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "auth handler", domain.ErrInvalidInput)
		return
	}
	token, user, err := h.auth.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		respondError(w, h.logger, "register failed", err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.AuthResponse{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresIn:   token.ExpiresIn,
		User:        toUserResponse(user),
	})
}

func toUserResponse(u domain.User) dto.UserResponse {
	roles := make([]string, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = string(role)
	}
	return dto.UserResponse{
		ID:       u.ID,
		Email:    u.Email,
		FullName: u.FullName,
		Roles:    roles,
	}
}
