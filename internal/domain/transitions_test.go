package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/oziev02/help-desk/internal/domain"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from domain.TicketStatus
		to   domain.TicketStatus
		want bool
	}{
		{"new to assigned", domain.StatusNew, domain.StatusAssigned, true},
		{"new to cancelled", domain.StatusNew, domain.StatusCancelled, true},
		{"new to resolved", domain.StatusNew, domain.StatusResolved, false},
		{"assigned to in_progress", domain.StatusAssigned, domain.StatusInProgress, true},
		{"assigned refuse to new", domain.StatusAssigned, domain.StatusNew, true},
		{"closed to reopened", domain.StatusClosed, domain.StatusReopened, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.CanTransition(tt.from, tt.to))
		})
	}
}

func TestCanActorTransition_userCannotChangeStatus(t *testing.T) {
	actor := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	ticket := domain.Ticket{AuthorID: "u1", Status: domain.StatusNew}

	assert.False(t, domain.CanActorTransition(actor, domain.StatusNew, domain.StatusInProgress, ticket))
	assert.False(t, domain.CanAssign(actor))
}

func TestCanActorTransition_dispatcherAssigns(t *testing.T) {
	actor := domain.Actor{UserID: "d1", Roles: []domain.Role{domain.RoleDispatcher}}
	ticket := domain.Ticket{Status: domain.StatusNew}

	assert.True(t, domain.CanAssign(actor))
	assert.True(t, domain.CanActorTransition(actor, domain.StatusNew, domain.StatusAssigned, ticket))
}

func TestCanActorTransition_executorOnlyIfAssignee(t *testing.T) {
	assignee := "e1"
	actor := domain.Actor{UserID: "e1", Roles: []domain.Role{domain.RoleExecutor}}
	other := domain.Actor{UserID: "e2", Roles: []domain.Role{domain.RoleExecutor}}
	ticket := domain.Ticket{Status: domain.StatusInProgress, AssigneeID: &assignee}

	assert.True(t, domain.CanActorTransition(actor, domain.StatusInProgress, domain.StatusResolved, ticket))
	assert.False(t, domain.CanActorTransition(other, domain.StatusInProgress, domain.StatusResolved, ticket))
}

func TestCanComplete_authorConfirms(t *testing.T) {
	author := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	other := domain.Actor{UserID: "u2", Roles: []domain.Role{domain.RoleUser}}
	ticket := domain.Ticket{AuthorID: "u1", Status: domain.StatusResolved}

	assert.True(t, domain.CanComplete(author, ticket))
	assert.False(t, domain.CanComplete(other, ticket))
	assert.True(t, domain.CanActorTransition(author, domain.StatusResolved, domain.StatusClosed, ticket))
}

func TestCanRefuse_assigneeOnly(t *testing.T) {
	assignee := "e1"
	actor := domain.Actor{UserID: "e1", Roles: []domain.Role{domain.RoleExecutor}}
	other := domain.Actor{UserID: "e2", Roles: []domain.Role{domain.RoleExecutor}}
	ticket := domain.Ticket{Status: domain.StatusAssigned, AssigneeID: &assignee}

	assert.True(t, domain.CanRefuse(actor, ticket))
	assert.False(t, domain.CanRefuse(other, ticket))
}

func TestCanReopen_onceOnly(t *testing.T) {
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	ticket := domain.Ticket{AuthorID: "u1", Status: domain.StatusClosed, ReopenCount: 0}
	assert.True(t, domain.CanReopen(user, ticket))

	ticket.ReopenCount = 1
	assert.False(t, domain.CanReopen(user, ticket))
}

func TestCanEditTicket(t *testing.T) {
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	ownNew := domain.Ticket{AuthorID: "u1", Status: domain.StatusNew}
	ownAssigned := domain.Ticket{AuthorID: "u1", Status: domain.StatusAssigned}

	assert.True(t, domain.CanEditTicket(user, ownNew))
	assert.False(t, domain.CanEditTicket(user, ownAssigned))
}

func TestCanViewTicket_scopes(t *testing.T) {
	author := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	other := domain.Actor{UserID: "u2", Roles: []domain.Role{domain.RoleUser}}
	assigneeID := "e1"
	executor := domain.Actor{UserID: "e1", Roles: []domain.Role{domain.RoleExecutor}}
	dispatcher := domain.Actor{UserID: "d1", Roles: []domain.Role{domain.RoleDispatcher}}
	ticket := domain.Ticket{AuthorID: "u1", AssigneeID: &assigneeID, Status: domain.StatusAssigned}

	assert.True(t, domain.CanViewTicket(author, ticket))
	assert.True(t, domain.CanViewTicket(executor, ticket))
	assert.True(t, domain.CanViewTicket(dispatcher, ticket))
	assert.False(t, domain.CanViewTicket(other, ticket))
	assert.False(t, domain.CanCommentOnTicket(other, ticket))
}

func TestCanSoftDeleteTicket(t *testing.T) {
	author := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	assert.True(t, domain.CanSoftDeleteTicket(author, domain.Ticket{AuthorID: "u1", Status: domain.StatusNew}))
	assert.False(t, domain.CanSoftDeleteTicket(author, domain.Ticket{AuthorID: "u1", Status: domain.StatusInProgress}))
}

func TestCanViewReports_manager(t *testing.T) {
	manager := domain.Actor{UserID: "m1", Roles: []domain.Role{domain.RoleManager}}
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	assert.True(t, domain.CanViewReports(manager))
	assert.False(t, domain.CanViewReports(user))
}

func TestCanLinkTickets(t *testing.T) {
	dispatcher := domain.Actor{UserID: "d1", Roles: []domain.Role{domain.RoleDispatcher}}
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	assert.True(t, domain.CanLinkTickets(dispatcher))
	assert.False(t, domain.CanLinkTickets(user))
}

func TestCanManageRooms(t *testing.T) {
	admin := domain.Actor{UserID: "a1", Roles: []domain.Role{domain.RoleAdmin}}
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	assert.True(t, domain.CanManageRooms(admin))
	assert.False(t, domain.CanManageRooms(user))
}

func TestTicketIsOverdue(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	open := domain.Ticket{Status: domain.StatusAssigned, DueAt: &past}
	closed := domain.Ticket{Status: domain.StatusClosed, DueAt: &past}
	ok := domain.Ticket{Status: domain.StatusNew, DueAt: &future}

	assert.True(t, open.IsOverdue(time.Now().UTC()))
	assert.False(t, closed.IsOverdue(time.Now().UTC()))
	assert.False(t, ok.IsOverdue(time.Now().UTC()))
}
