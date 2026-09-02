package domain_test

import (
	"testing"

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

func TestCanActorTransition_executorResolves(t *testing.T) {
	actor := domain.Actor{UserID: "e1", Roles: []domain.Role{domain.RoleExecutor}}
	ticket := domain.Ticket{Status: domain.StatusInProgress}

	assert.True(t, domain.CanActorTransition(actor, domain.StatusInProgress, domain.StatusResolved, ticket))
}

func TestCanReopen(t *testing.T) {
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	ticket := domain.Ticket{AuthorID: "u1", Status: domain.StatusClosed}

	assert.True(t, domain.CanReopen(user, ticket))
}

func TestCanEditTicket(t *testing.T) {
	user := domain.Actor{UserID: "u1", Roles: []domain.Role{domain.RoleUser}}
	ownNew := domain.Ticket{AuthorID: "u1", Status: domain.StatusNew}
	ownAssigned := domain.Ticket{AuthorID: "u1", Status: domain.StatusAssigned}

	assert.True(t, domain.CanEditTicket(user, ownNew))
	assert.False(t, domain.CanEditTicket(user, ownAssigned))
}
