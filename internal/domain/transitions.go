package domain

import "slices"

var allowedTransitions = map[TicketStatus][]TicketStatus{
	StatusNew:        {StatusAssigned, StatusCancelled},
	StatusAssigned:   {StatusInProgress, StatusNew, StatusCancelled},
	StatusInProgress: {StatusResolved, StatusNew, StatusCancelled},
	StatusResolved:   {StatusClosed},
	StatusClosed:     {StatusReopened},
	StatusReopened:   {StatusAssigned, StatusCancelled},
	StatusCancelled:  {},
}

func CanTransition(from, to TicketStatus) bool {
	return slices.Contains(allowedTransitions[from], to)
}

func isAssignee(actor Actor, ticket Ticket) bool {
	return ticket.AssigneeID != nil && *ticket.AssigneeID == actor.UserID
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
		return actor.HasRole(RoleExecutor) && isAssignee(actor, ticket)
	case StatusResolved:
		return actor.HasRole(RoleExecutor) && isAssignee(actor, ticket)
	case StatusClosed:
		// заявитель подтверждает выполнение (оценка/комментарий проверяются в сервисе)
		return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID
	case StatusReopened:
		return CanReopen(actor, ticket)
	case StatusNew:
		// возврат на new = отказ исполнителя (reason в сервисе)
		return actor.HasRole(RoleExecutor) && isAssignee(actor, ticket) &&
			(from == StatusAssigned || from == StatusInProgress)
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
	if ticket.ReopenCount >= 1 {
		return false
	}
	if actor.IsAdmin() || actor.HasRole(RoleDispatcher) {
		return true
	}
	return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID
}

func CanRefuse(actor Actor, ticket Ticket) bool {
	if actor.IsAdmin() {
		return ticket.Status == StatusAssigned || ticket.Status == StatusInProgress
	}
	return actor.HasRole(RoleExecutor) && isAssignee(actor, ticket) &&
		(ticket.Status == StatusAssigned || ticket.Status == StatusInProgress)
}

func CanComplete(actor Actor, ticket Ticket) bool {
	if ticket.Status != StatusResolved {
		return false
	}
	if actor.IsAdmin() {
		return true
	}
	return actor.HasRole(RoleUser) && ticket.AuthorID == actor.UserID
}

func CanViewReports(actor Actor) bool {
	return actor.IsAdmin() || actor.HasRole(RoleManager) || actor.HasRole(RoleDispatcher)
}

func CanManageRoles(actor Actor) bool {
	return actor.IsAdmin()
}

func CanManageCategories(actor Actor) bool {
	return actor.IsAdmin()
}

func CanManageRooms(actor Actor) bool {
	return actor.IsAdmin()
}

func CanLinkTickets(actor Actor) bool {
	return actor.IsAdmin() || actor.HasRole(RoleDispatcher)
}

func CanListAllTickets(actor Actor) bool {
	return actor.IsAdmin() || actor.HasRole(RoleDispatcher) || actor.HasRole(RoleManager)
}

func CanViewTicket(actor Actor, ticket Ticket) bool {
	if CanListAllTickets(actor) {
		return true
	}
	if ticket.AuthorID == actor.UserID {
		return true
	}
	return isAssignee(actor, ticket)
}

func CanCommentOnTicket(actor Actor, ticket Ticket) bool {
	return CanViewTicket(actor, ticket)
}

func CanSoftDeleteTicket(actor Actor, ticket Ticket) bool {
	if actor.IsAdmin() {
		return true
	}
	if ticket.AuthorID != actor.UserID {
		return false
	}
	return ticket.Status == StatusNew || ticket.Status == StatusCancelled
}
