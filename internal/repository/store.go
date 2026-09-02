package repository

import (
	"context"

	"github.com/oziev02/help-desk/internal/domain"
)

type UserStore interface {
	CreateUser(ctx context.Context, email, passwordHash, fullName string, roles []domain.Role) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	ListExecutors(ctx context.Context) ([]domain.User, error)
	GrantRole(ctx context.Context, userID string, role domain.Role) error
	RevokeRole(ctx context.Context, userID string, role domain.Role) error
	ListRoles(ctx context.Context, userID string) ([]domain.Role, error)
	EnsureDemoUsers(ctx context.Context) error
}

type CategoryStore interface {
	CreateCategory(ctx context.Context, name string) (domain.Category, error)
	ListCategories(ctx context.Context, activeOnly bool) ([]domain.Category, error)
	GetCategoryByID(ctx context.Context, id string) (domain.Category, error)
}

type TicketStore interface {
	CreateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error)
	GetTicketByID(ctx context.Context, id string) (domain.Ticket, error)
	GetTicketByIDForUpdate(ctx context.Context, tx Tx, id string) (domain.Ticket, error)
	UpdateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error)
	UpdateTicketInTx(ctx context.Context, tx Tx, ticket domain.Ticket) (domain.Ticket, error)
	ListTickets(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error)
	SoftDeleteTicket(ctx context.Context, id string) error
	AddComment(ctx context.Context, comment domain.Comment) (domain.Comment, error)
	ListComments(ctx context.Context, ticketID string) ([]domain.Comment, error)
	LinkTickets(ctx context.Context, ticketID, linkedID string) error
	UnlinkTickets(ctx context.Context, ticketID, linkedID string) error
	ListLinks(ctx context.Context, ticketID string) ([]domain.TicketLink, error)
	ReportTickets(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error)
}

type TicketFilter struct {
	Status   *domain.TicketStatus
	Assignee *string
	Overdue  bool
}

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type DB interface {
	UserStore
	CategoryStore
	TicketStore
	Begin(ctx context.Context) (Tx, error)
	Ping(ctx context.Context) error
	Close()
}
