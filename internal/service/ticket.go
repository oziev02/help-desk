package service

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/repository"
)

type TicketService struct {
	db       repository.DB
	notifier Notifier
	logger   *slog.Logger
	slaHours int
}

func NewTicketService(db repository.DB, notifier Notifier, logger *slog.Logger, slaHours int) *TicketService {
	if notifier == nil {
		notifier = NoopNotifier{}
	}
	if slaHours <= 0 {
		slaHours = 48
	}
	return &TicketService{db: db, notifier: notifier, logger: logger, slaHours: slaHours}
}

type CreateTicketInput struct {
	Title       string
	Description string
	CategoryID  *string
	Room        *string
	RoomID      *string
	DueAt       *time.Time
}

type UpdateTicketInput struct {
	Title       *string
	Description *string
	CategoryID  *string
	Room        *string
	RoomID      *string
	DueAt       *time.Time
}

type AssignInput struct {
	AssigneeID string
}

type TransitionInput struct {
	Status domain.TicketStatus
}

type CompleteInput struct {
	Rating  int
	Comment string
}

type RefuseInput struct {
	Reason string
}

type LinkInput struct {
	LinkedTicketID string
}

func (s *TicketService) Create(ctx context.Context, actor domain.Actor, in CreateTicketInput) (domain.Ticket, error) {
	if strings.TrimSpace(in.Title) == "" {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	if in.CategoryID != nil {
		cat, err := s.db.GetCategoryByID(ctx, *in.CategoryID)
		if err != nil {
			return domain.Ticket{}, err
		}
		if !cat.IsActive {
			return domain.Ticket{}, domain.ErrInvalidInput
		}
	}

	roomName := in.Room
	if in.RoomID != nil {
		room, err := s.db.GetRoomByID(ctx, *in.RoomID)
		if err != nil {
			return domain.Ticket{}, err
		}
		if !room.IsActive {
			return domain.Ticket{}, domain.ErrInvalidInput
		}
		name := room.Name
		roomName = &name
	}

	dueAt := in.DueAt
	if dueAt == nil {
		t := time.Now().UTC().Add(time.Duration(s.slaHours) * time.Hour)
		dueAt = &t
	}

	ticket := domain.Ticket{
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		CategoryID:  in.CategoryID,
		Status:      domain.StatusNew,
		AuthorID:    actor.UserID,
		Room:        roomName,
		RoomID:      in.RoomID,
		DueAt:       dueAt,
	}
	return s.db.CreateTicket(ctx, ticket)
}

func (s *TicketService) Get(ctx context.Context, actor domain.Actor, id string) (domain.Ticket, []domain.Comment, []domain.TicketLink, error) {
	ticket, err := s.db.GetTicketByID(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	if !domain.CanViewTicket(actor, ticket) {
		return domain.Ticket{}, nil, nil, domain.ErrNotFound
	}
	comments, err := s.db.ListComments(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	links, err := s.db.ListLinks(ctx, id)
	if err != nil {
		return domain.Ticket{}, nil, nil, err
	}
	return ticket, comments, links, nil
}

func (s *TicketService) List(ctx context.Context, actor domain.Actor, filter repository.TicketFilter) ([]domain.Ticket, error) {
	if !domain.CanListAllTickets(actor) {
		uid := actor.UserID
		filter.VisibleToUserID = &uid
	}
	return s.db.ListTickets(ctx, filter)
}

func (s *TicketService) Update(ctx context.Context, actor domain.Actor, id string, in UpdateTicketInput) (domain.Ticket, error) {
	ticket, err := s.db.GetTicketByID(ctx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanEditTicket(actor, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	if in.Title != nil {
		ticket.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		ticket.Description = *in.Description
	}
	if in.CategoryID != nil {
		cat, err := s.db.GetCategoryByID(ctx, *in.CategoryID)
		if err != nil {
			return domain.Ticket{}, err
		}
		if !cat.IsActive {
			return domain.Ticket{}, domain.ErrInvalidInput
		}
		ticket.CategoryID = in.CategoryID
	}
	if in.Room != nil {
		ticket.Room = in.Room
	}
	if in.RoomID != nil {
		room, err := s.db.GetRoomByID(ctx, *in.RoomID)
		if err != nil {
			return domain.Ticket{}, err
		}
		if !room.IsActive {
			return domain.Ticket{}, domain.ErrInvalidInput
		}
		ticket.RoomID = in.RoomID
		name := room.Name
		ticket.Room = &name
	}
	if in.DueAt != nil {
		ticket.DueAt = in.DueAt
	}
	return s.db.UpdateTicket(ctx, ticket)
}

func (s *TicketService) Delete(ctx context.Context, actor domain.Actor, id string) error {
	ticket, err := s.db.GetTicketByID(ctx, id)
	if err != nil {
		return err
	}
	if !domain.CanSoftDeleteTicket(actor, ticket) {
		return domain.ErrForbidden
	}
	return s.db.SoftDeleteTicket(ctx, id)
}

func (s *TicketService) Assign(ctx context.Context, actor domain.Actor, id string, in AssignInput) (domain.Ticket, error) {
	if !domain.CanAssign(actor) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	if in.AssigneeID == "" {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Ticket{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ticket, err := s.db.GetTicketByIDForUpdate(ctx, tx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if ticket.Status != domain.StatusNew && ticket.Status != domain.StatusReopened && ticket.Status != domain.StatusAssigned {
		return domain.Ticket{}, domain.ErrInvalidTransition
	}

	assignee, err := s.db.GetUserByID(ctx, in.AssigneeID)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !assigneeHasRole(assignee, domain.RoleExecutor) && !assigneeHasRole(assignee, domain.RoleAdmin) {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	ticket.AssigneeID = &in.AssigneeID
	ticket.Status = domain.StatusAssigned
	ticket.RefuseReason = nil
	updated, err := s.db.UpdateTicketInTx(ctx, tx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Ticket{}, err
	}
	s.notify(func() error {
		return s.notifier.TicketAssigned(ctx, updated.ID, assignee.Email)
	})
	return updated, nil
}

func (s *TicketService) Transition(ctx context.Context, actor domain.Actor, id string, in TransitionInput) (domain.Ticket, error) {
	if !in.Status.Valid() {
		return domain.Ticket{}, domain.ErrInvalidInput
	}
	// assigned только через Assign; closed/new — через Complete/Refuse
	if in.Status == domain.StatusClosed || in.Status == domain.StatusNew || in.Status == domain.StatusAssigned {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Ticket{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ticket, err := s.db.GetTicketByIDForUpdate(ctx, tx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanActorTransition(actor, ticket.Status, in.Status, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	ticket.Status = in.Status
	updated, err := s.db.UpdateTicketInTx(ctx, tx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Ticket{}, err
	}

	s.notifyStatus(ctx, updated)
	return updated, nil
}

func (s *TicketService) Complete(ctx context.Context, actor domain.Actor, id string, in CompleteInput) (domain.Ticket, error) {
	if in.Rating < 1 || in.Rating > 5 || strings.TrimSpace(in.Comment) == "" {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Ticket{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ticket, err := s.db.GetTicketByIDForUpdate(ctx, tx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanComplete(actor, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	if !domain.CanActorTransition(actor, ticket.Status, domain.StatusClosed, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}

	rating := in.Rating
	ticket.Rating = &rating
	ticket.Status = domain.StatusClosed
	updated, err := s.db.UpdateTicketInTx(ctx, tx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	if _, err := s.db.AddCommentInTx(ctx, tx, domain.Comment{
		TicketID: id,
		AuthorID: actor.UserID,
		Body:     strings.TrimSpace(in.Comment),
	}); err != nil {
		return domain.Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Ticket{}, err
	}

	if updated.AssigneeID != nil {
		assignee, err := s.db.GetUserByID(ctx, *updated.AssigneeID)
		if err != nil {
			s.warnNotify("load assignee for complete", err)
		} else {
			s.notify(func() error {
				return s.notifier.TicketCompleted(ctx, updated.ID, assignee.Email, rating)
			})
		}
	}
	return updated, nil
}

func (s *TicketService) Refuse(ctx context.Context, actor domain.Actor, id string, in RefuseInput) (domain.Ticket, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return domain.Ticket{}, domain.ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Ticket{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ticket, err := s.db.GetTicketByIDForUpdate(ctx, tx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanRefuse(actor, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	if !domain.CanActorTransition(actor, ticket.Status, domain.StatusNew, ticket) && !actor.IsAdmin() {
		return domain.Ticket{}, domain.ErrForbidden
	}

	ticket.RefuseReason = &reason
	ticket.AssigneeID = nil
	ticket.Status = domain.StatusNew
	updated, err := s.db.UpdateTicketInTx(ctx, tx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	if _, err := s.db.AddCommentInTx(ctx, tx, domain.Comment{
		TicketID: id,
		AuthorID: actor.UserID,
		Body:     "Отказ: " + reason,
	}); err != nil {
		return domain.Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Ticket{}, err
	}

	author, err := s.db.GetUserByID(ctx, updated.AuthorID)
	if err != nil {
		s.warnNotify("load author for refuse", err)
	} else {
		s.notify(func() error {
			return s.notifier.TicketRefused(ctx, updated.ID, reason, author.Email)
		})
	}
	return updated, nil
}

func (s *TicketService) Reopen(ctx context.Context, actor domain.Actor, id string) (domain.Ticket, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.Ticket{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ticket, err := s.db.GetTicketByIDForUpdate(ctx, tx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanReopen(actor, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	ticket.Status = domain.StatusReopened
	ticket.ReopenCount++
	ticket.Rating = nil
	updated, err := s.db.UpdateTicketInTx(ctx, tx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Ticket{}, err
	}

	s.notifyStatus(ctx, updated)
	return updated, nil
}

func (s *TicketService) Cancel(ctx context.Context, actor domain.Actor, id string) (domain.Ticket, error) {
	ticket, err := s.db.GetTicketByID(ctx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !domain.CanCancel(actor, ticket) {
		return domain.Ticket{}, domain.ErrForbidden
	}
	return s.Transition(ctx, actor, id, TransitionInput{Status: domain.StatusCancelled})
}

func (s *TicketService) AddComment(ctx context.Context, actor domain.Actor, id, body string) (domain.Comment, error) {
	if strings.TrimSpace(body) == "" {
		return domain.Comment{}, domain.ErrInvalidInput
	}
	ticket, err := s.db.GetTicketByID(ctx, id)
	if err != nil {
		return domain.Comment{}, err
	}
	if !domain.CanCommentOnTicket(actor, ticket) {
		return domain.Comment{}, domain.ErrNotFound
	}
	return s.db.AddComment(ctx, domain.Comment{
		TicketID: id,
		AuthorID: actor.UserID,
		Body:     strings.TrimSpace(body),
	})
}

func (s *TicketService) Link(ctx context.Context, actor domain.Actor, id string, in LinkInput) error {
	if !domain.CanLinkTickets(actor) {
		return domain.ErrForbidden
	}
	if in.LinkedTicketID == "" {
		return domain.ErrInvalidInput
	}
	if id == in.LinkedTicketID {
		return domain.ErrSelfLink
	}
	if _, err := s.db.GetTicketByID(ctx, id); err != nil {
		return err
	}
	if _, err := s.db.GetTicketByID(ctx, in.LinkedTicketID); err != nil {
		return err
	}
	return s.db.LinkTickets(ctx, id, in.LinkedTicketID)
}

func (s *TicketService) Unlink(ctx context.Context, actor domain.Actor, id, linkedID string) error {
	if !domain.CanLinkTickets(actor) {
		return domain.ErrForbidden
	}
	return s.db.UnlinkTickets(ctx, id, linkedID)
}

func (s *TicketService) Report(ctx context.Context, actor domain.Actor, filter repository.TicketFilter) ([]domain.Ticket, error) {
	if !domain.CanViewReports(actor) {
		return nil, domain.ErrForbidden
	}
	return s.db.ReportTickets(ctx, filter)
}

func (s *TicketService) notifyStatus(ctx context.Context, ticket domain.Ticket) {
	author, err := s.db.GetUserByID(ctx, ticket.AuthorID)
	if err != nil {
		s.warnNotify("load author for status notify", err)
		return
	}
	s.notify(func() error {
		return s.notifier.TicketStatusChanged(ctx, ticket.ID, string(ticket.Status), author.Email)
	})
}

func (s *TicketService) notify(fn func() error) {
	if err := fn(); err != nil && s.logger != nil {
		s.logger.Warn("notification failed", "error", err)
	}
}

func (s *TicketService) warnNotify(msg string, err error) {
	if s.logger != nil {
		s.logger.Warn(msg, "error", err)
	}
}

func assigneeHasRole(u domain.User, role domain.Role) bool {
	return slices.Contains(u.Roles, role)
}
