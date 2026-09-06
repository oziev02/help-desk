package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/handler/dto"
	"github.com/oziev02/help-desk/internal/middleware"
	"github.com/oziev02/help-desk/internal/service"
)

type AdminHandler struct {
	admin  *service.AdminService
	logger *slog.Logger
}

func NewAdminHandler(admin *service.AdminService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{admin: admin, logger: logger}
}

func (h *AdminHandler) GrantRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	userID := chi.URLParam(r, "id")
	var req dto.RoleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "admin handler", domain.ErrInvalidInput)
		return
	}
	if err := h.admin.GrantRole(r.Context(), actor, userID, domain.Role(req.Role)); err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	userID := chi.URLParam(r, "id")
	roleName := chi.URLParam(r, "role")
	if err := h.admin.RevokeRole(r.Context(), actor, userID, domain.Role(roleName)); err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	userID := chi.URLParam(r, "id")
	roles, err := h.admin.ListUserRoles(r.Context(), actor, userID)
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	resp := dto.RolesResponse{Roles: make([]string, len(roles))}
	for i, role := range roles {
		resp.Roles[i] = string(role)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	var req dto.CategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "admin handler", domain.ErrInvalidInput)
		return
	}
	category, err := h.admin.CreateCategory(r.Context(), actor, req.Name)
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	writeJSON(w, http.StatusCreated, toCategoryResponse(category))
}

func (h *AdminHandler) DeactivateCategory(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	category, err := h.admin.DeactivateCategory(r.Context(), actor, id)
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toCategoryResponse(category))
}

func (h *AdminHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.admin.ListCategories(r.Context(), true)
	if err != nil {
		respondError(w, h.logger, "list categories", err)
		return
	}
	resp := make([]dto.CategoryResponse, len(categories))
	for i, c := range categories {
		resp[i] = toCategoryResponse(c)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.admin.ListRooms(r.Context(), true)
	if err != nil {
		respondError(w, h.logger, "list rooms", err)
		return
	}
	resp := make([]dto.RoomResponse, len(rooms))
	for i, room := range rooms {
		resp[i] = toRoomResponse(room)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	var req dto.RoomRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "admin handler", domain.ErrInvalidInput)
		return
	}
	room, err := h.admin.CreateRoom(r.Context(), actor, service.CreateRoomInput{
		Name:     req.Name,
		Building: req.Building,
		Floor:    req.Floor,
	})
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	writeJSON(w, http.StatusCreated, toRoomResponse(room))
}

func (h *AdminHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.UpdateRoomRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "admin handler", domain.ErrInvalidInput)
		return
	}
	room, err := h.admin.UpdateRoom(r.Context(), actor, id, service.UpdateRoomInput{
		Name:     req.Name,
		Building: req.Building,
		Floor:    req.Floor,
		IsActive: req.IsActive,
	})
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toRoomResponse(room))
}

func (h *AdminHandler) DeactivateRoom(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "admin handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	room, err := h.admin.DeactivateRoom(r.Context(), actor, id)
	if err != nil {
		respondError(w, h.logger, "admin handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toRoomResponse(room))
}

func toCategoryResponse(c domain.Category) dto.CategoryResponse {
	return dto.CategoryResponse{ID: c.ID, Name: c.Name, IsActive: c.IsActive}
}

func toRoomResponse(r domain.Room) dto.RoomResponse {
	return dto.RoomResponse{
		ID:       r.ID,
		Name:     r.Name,
		Building: r.Building,
		Floor:    r.Floor,
		IsActive: r.IsActive,
	}
}
