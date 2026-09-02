package domain

var allowedTransitions = map[TicketStatus][]TicketStatus{
	StatusNew:        {StatusAssigned, StatusCancelled},
	StatusAssigned:   {StatusInProgress, StatusCancelled},
	StatusInProgress: {StatusResolved, StatusCancelled},
	StatusResolved:   {StatusClosed},
	StatusClosed:     {StatusReopened},
	StatusReopened:   {StatusAssigned, StatusCancelled},
	StatusCancelled:  {},
}

func CanTransition(from, to TicketStatus) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

func CanActorTransition(actor Actor, from, to TicketStatus, ticket Ticket) bool {
	if !CanTransition(from, to) {
		return false
	}

	if actor.IsAdmin() {
		return true
	}

	switch to {
	case StatusAssigned:
		return actor.HasRole(RoleDispatcher)
	case StatusInProgress:
		return actor.HasRole(RoleExecutor)
	case StatusResolved:
		return actor.HasRole(RoleExecutor)
	case StatusClosed:
		return actor.HasRole(RoleDispatcher)
	case StatusReopened:
		if actor.HasRole(RoleDispatcher) {
			return true
		}
		return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID && from == StatusClosed
	case StatusCancelled:
		if actor.HasRole(RoleDispatcher) {
			return true
		}
		return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID
	default:
		return false
	}
}

func CanAssign(actor Actor) bool {
	return actor.HasRole(RoleDispatcher)
}

func CanEditTicket(actor Actor, ticket Ticket) bool {
	if actor.IsAdmin() || actor.HasRole(RoleDispatcher) {
		return true
	}
	if ticket.AuthorID != actor.UserID {
		return false
	}
	return ticket.Status == StatusNew || ticket.Status == StatusReopened
}

func CanCancel(actor Actor, ticket Ticket) bool {
	if actor.IsAdmin() || actor.HasRole(RoleDispatcher) {
		return true
	}
	return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID &&
		ticket.Status != StatusClosed && ticket.Status != StatusCancelled
}

func CanReopen(actor Actor, ticket Ticket) bool {
	if ticket.Status != StatusClosed {
		return false
	}
	if actor.IsAdmin() || actor.HasRole(RoleDispatcher) {
		return true
	}
	return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID
}

func CanManageRoles(actor Actor) bool {
	return actor.IsAdmin()
}

func CanManageCategories(actor Actor) bool {
	return actor.IsAdmin()
}
