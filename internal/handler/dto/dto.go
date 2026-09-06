package dto

import "time"

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserResponse `json:"user"`
}

type UserResponse struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Roles    []string `json:"roles"`
}

type CreateTicketRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	CategoryID  *string `json:"category_id,omitempty"`
	Room        *string `json:"room,omitempty"`
	RoomID      *string `json:"room_id,omitempty"`
	DueAt       *string `json:"due_at,omitempty"`
}

type UpdateTicketRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	CategoryID  *string `json:"category_id,omitempty"`
	Room        *string `json:"room,omitempty"`
	RoomID      *string `json:"room_id,omitempty"`
	DueAt       *string `json:"due_at,omitempty"`
}

type AssignRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type TransitionRequest struct {
	Status string `json:"status"`
}

type CompleteRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type RefuseRequest struct {
	Reason string `json:"reason"`
}

type CommentRequest struct {
	Body string `json:"body"`
}

type RoleRequest struct {
	Role string `json:"role"`
}

type CategoryRequest struct {
	Name string `json:"name"`
}

type RoomRequest struct {
	Name     string `json:"name"`
	Building string `json:"building"`
	Floor    *int   `json:"floor,omitempty"`
}

type UpdateRoomRequest struct {
	Name     *string `json:"name,omitempty"`
	Building *string `json:"building,omitempty"`
	Floor    *int    `json:"floor,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type LinkRequest struct {
	LinkedTicketID string `json:"linked_ticket_id"`
}

type TicketResponse struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	CategoryID   *string           `json:"category_id,omitempty"`
	Status       string            `json:"status"`
	AuthorID     string            `json:"author_id"`
	AssigneeID   *string           `json:"assignee_id,omitempty"`
	Room         *string           `json:"room,omitempty"`
	RoomID       *string           `json:"room_id,omitempty"`
	DueAt        *time.Time        `json:"due_at,omitempty"`
	Overdue      bool              `json:"overdue"`
	Rating       *int              `json:"rating,omitempty"`
	RefuseReason *string           `json:"refuse_reason,omitempty"`
	ReopenCount  int               `json:"reopen_count"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Comments     []CommentResponse `json:"comments,omitempty"`
	Links        []LinkResponse    `json:"links,omitempty"`
}

type CommentResponse struct {
	ID        string    `json:"id"`
	TicketID  string    `json:"ticket_id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type LinkResponse struct {
	LinkedTicketID string    `json:"linked_ticket_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type CategoryResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type RoomResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Building string `json:"building"`
	Floor    *int   `json:"floor,omitempty"`
	IsActive bool   `json:"is_active"`
}

type RolesResponse struct {
	Roles []string `json:"roles"`
}
