package service

import (
	"context"
	"strings"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/repository"
)

type AdminService struct {
	db repository.DB
}

func NewAdminService(db repository.DB) *AdminService {
	return &AdminService{db: db}
}

func (s *AdminService) GrantRole(ctx context.Context, actor domain.Actor, userID string, role domain.Role) error {
	if !domain.CanManageRoles(actor) {
		return domain.ErrForbidden
	}
	if !role.Valid() {
		return domain.ErrInvalidInput
	}
	if _, err := s.db.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return s.db.GrantRole(ctx, userID, role)
}

func (s *AdminService) RevokeRole(ctx context.Context, actor domain.Actor, userID string, role domain.Role) error {
	if !domain.CanManageRoles(actor) {
		return domain.ErrForbidden
	}
	if !role.Valid() {
		return domain.ErrInvalidInput
	}
	return s.db.RevokeRole(ctx, userID, role)
}

func (s *AdminService) ListUserRoles(ctx context.Context, actor domain.Actor, userID string) ([]domain.Role, error) {
	if !domain.CanManageRoles(actor) {
		return nil, domain.ErrForbidden
	}
	return s.db.ListRoles(ctx, userID)
}

func (s *AdminService) CreateCategory(ctx context.Context, actor domain.Actor, name string) (domain.Category, error) {
	if !domain.CanManageCategories(actor) {
		return domain.Category{}, domain.ErrForbidden
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Category{}, domain.ErrInvalidInput
	}
	return s.db.CreateCategory(ctx, name)
}

func (s *AdminService) DeactivateCategory(ctx context.Context, actor domain.Actor, id string) (domain.Category, error) {
	if !domain.CanManageCategories(actor) {
		return domain.Category{}, domain.ErrForbidden
	}
	return s.db.SetCategoryActive(ctx, id, false)
}

func (s *AdminService) ListCategories(ctx context.Context, activeOnly bool) ([]domain.Category, error) {
	return s.db.ListCategories(ctx, activeOnly)
}

type CreateRoomInput struct {
	Name     string
	Building string
	Floor    *int
}

type UpdateRoomInput struct {
	Name     *string
	Building *string
	Floor    *int
	IsActive *bool
}

func (s *AdminService) CreateRoom(ctx context.Context, actor domain.Actor, in CreateRoomInput) (domain.Room, error) {
	if !domain.CanManageRooms(actor) {
		return domain.Room{}, domain.ErrForbidden
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	return s.db.CreateRoom(ctx, domain.Room{
		Name:     name,
		Building: strings.TrimSpace(in.Building),
		Floor:    in.Floor,
		IsActive: true,
	})
}

func (s *AdminService) UpdateRoom(ctx context.Context, actor domain.Actor, id string, in UpdateRoomInput) (domain.Room, error) {
	if !domain.CanManageRooms(actor) {
		return domain.Room{}, domain.ErrForbidden
	}
	room, err := s.db.GetRoomByID(ctx, id)
	if err != nil {
		return domain.Room{}, err
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return domain.Room{}, domain.ErrInvalidInput
		}
		room.Name = name
	}
	if in.Building != nil {
		room.Building = strings.TrimSpace(*in.Building)
	}
	if in.Floor != nil {
		room.Floor = in.Floor
	}
	if in.IsActive != nil {
		room.IsActive = *in.IsActive
	}
	return s.db.UpdateRoom(ctx, room)
}

func (s *AdminService) DeactivateRoom(ctx context.Context, actor domain.Actor, id string) (domain.Room, error) {
	if !domain.CanManageRooms(actor) {
		return domain.Room{}, domain.ErrForbidden
	}
	return s.db.SetRoomActive(ctx, id, false)
}

func (s *AdminService) ListRooms(ctx context.Context, activeOnly bool) ([]domain.Room, error) {
	return s.db.ListRooms(ctx, activeOnly)
}
