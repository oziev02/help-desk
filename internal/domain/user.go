package domain

import "time"

type Role string

const (
	RoleUser       Role = "user"
	RoleDispatcher Role = "dispatcher"
	RoleExecutor   Role = "executor"
	RoleAdmin      Role = "admin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleDispatcher, RoleExecutor, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Roles        []Role
	CreatedAt    time.Time
}

type Category struct {
	ID        string
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

type TicketStatus string

const (
	StatusNew        TicketStatus = "new"
	StatusAssigned   TicketStatus = "assigned"
	StatusInProgress TicketStatus = "in_progress"
	StatusResolved   TicketStatus = "resolved"
	StatusClosed     TicketStatus = "closed"
	StatusReopened   TicketStatus = "reopened"
	StatusCancelled  TicketStatus = "cancelled"
)

func (s TicketStatus) Valid() bool {
	switch s {
	case StatusNew, StatusAssigned, StatusInProgress, StatusResolved, StatusClosed, StatusReopened, StatusCancelled:
		return true
	default:
		return false
	}
}

type Ticket struct {
	ID          string
	Title       string
	Description string
	CategoryID  *string
	Status      TicketStatus
	AuthorID    string
	AssigneeID  *string
	Room        *string
	DueAt       *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Comment struct {
	ID        string
	TicketID  string
	AuthorID  string
	Body      string
	CreatedAt time.Time
}

type TicketLink struct {
	TicketID       string
	LinkedTicketID string
	CreatedAt      time.Time
}

type Actor struct {
	UserID string
	Roles  []Role
}

func (a Actor) HasRole(role Role) bool {
	for _, r := range a.Roles {
		if r == role || r == RoleAdmin {
			return true
		}
	}
	return false
}

func (a Actor) IsAdmin() bool {
	return a.HasRole(RoleAdmin)
}
