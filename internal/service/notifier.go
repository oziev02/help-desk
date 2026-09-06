package service

import (
	"context"
	"log/slog"
)

type Notifier interface {
	TicketAssigned(ctx context.Context, ticketID, assigneeEmail string) error
	TicketStatusChanged(ctx context.Context, ticketID, status, recipientEmail string) error
	TicketRefused(ctx context.Context, ticketID, reason, recipientEmail string) error
	TicketCompleted(ctx context.Context, ticketID, recipientEmail string, rating int) error
}

type NoopNotifier struct{}

func (NoopNotifier) TicketAssigned(context.Context, string, string) error { return nil }
func (NoopNotifier) TicketStatusChanged(context.Context, string, string, string) error {
	return nil
}
func (NoopNotifier) TicketRefused(context.Context, string, string, string) error { return nil }
func (NoopNotifier) TicketCompleted(context.Context, string, string, int) error  { return nil }

type LogNotifier struct {
	logger *slog.Logger
}

func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

func (n *LogNotifier) TicketAssigned(ctx context.Context, ticketID, assigneeEmail string) error {
	n.logger.InfoContext(ctx, "notification: ticket assigned",
		"ticket_id", ticketID,
		"assignee_email", assigneeEmail,
	)
	return nil
}

func (n *LogNotifier) TicketStatusChanged(ctx context.Context, ticketID, status, recipientEmail string) error {
	n.logger.InfoContext(ctx, "notification: ticket status changed",
		"ticket_id", ticketID,
		"status", status,
		"recipient_email", recipientEmail,
	)
	return nil
}

func (n *LogNotifier) TicketRefused(ctx context.Context, ticketID, reason, recipientEmail string) error {
	n.logger.InfoContext(ctx, "notification: ticket refused",
		"ticket_id", ticketID,
		"reason", reason,
		"recipient_email", recipientEmail,
	)
	return nil
}

func (n *LogNotifier) TicketCompleted(ctx context.Context, ticketID, recipientEmail string, rating int) error {
	n.logger.InfoContext(ctx, "notification: ticket completed",
		"ticket_id", ticketID,
		"recipient_email", recipientEmail,
		"rating", rating,
	)
	return nil
}
