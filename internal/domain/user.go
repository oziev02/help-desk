package domain

import "time"

type Role string

const (
	RoleUser       Role = "user"       // заявитель
	RoleDispatcher Role = "dispatcher" // диспетчер
	RoleExecutor   Role = "executor"   // исполнитель
	RoleManager    Role = "manager"    // руководитель
	RoleAdmin      Role = "admin"      // администратор
)

func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleDispatcher, RoleExecutor, RoleManager, RoleAdmin:
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

type Room struct {
	ID        string
	Name      string
	Building  string
	Floor     *int
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
	ID           string
	Title        string
	Description  string
	CategoryID   *string
	Status       TicketStatus
	AuthorID     string
	AssigneeID   *string
	Room         *string
	RoomID       *string
	DueAt        *time.Time
	Rating       *int
	RefuseReason *string
	ReopenCount  int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (t Ticket) IsOverdue(now time.Time) bool {
	if t.DueAt == nil {
		return false
	}
	switch t.Status {
	case StatusClosed, StatusCancelled:
		return false
	}
	return t.DueAt.Before(now)
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
