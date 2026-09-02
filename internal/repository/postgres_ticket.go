package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/oziev02/help-desk/internal/domain"
)

func scanTicket(row pgx.Row) (domain.Ticket, error) {
	var t domain.Ticket
	var status string
	err := row.Scan(
		&t.ID, &t.Title, &t.Description, &t.CategoryID, &status,
		&t.AuthorID, &t.AssigneeID, &t.Room, &t.DueAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return domain.Ticket{}, err
	}
	t.Status = domain.TicketStatus(status)
	return t, nil
}

func (p *Postgres) CreateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	return scanTicket(p.pool.QueryRow(ctx, `
		INSERT INTO tickets (title, description, category_id, status, author_id, assignee_id, room, due_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at`,
		ticket.Title, ticket.Description, ticket.CategoryID, ticket.Status,
		ticket.AuthorID, ticket.AssigneeID, ticket.Room, ticket.DueAt,
	))
}

func (p *Postgres) GetTicketByID(ctx context.Context, id string) (domain.Ticket, error) {
	t, err := scanTicket(p.pool.QueryRow(ctx, `
		SELECT id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at
		FROM tickets WHERE id = $1 AND deleted_at IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Ticket{}, domain.ErrNotFound
	}
	return t, err
}

func (p *Postgres) GetTicketByIDForUpdate(ctx context.Context, tx Tx, id string) (domain.Ticket, error) {
	ptx, ok := tx.(*pgxTx)
	if !ok {
		return domain.Ticket{}, fmt.Errorf("invalid transaction type")
	}
	t, err := scanTicket(ptx.tx.QueryRow(ctx, `
		SELECT id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at
		FROM tickets WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Ticket{}, domain.ErrNotFound
	}
	return t, err
}

func (p *Postgres) UpdateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	return scanTicket(p.pool.QueryRow(ctx, `
		UPDATE tickets SET title = $2, description = $3, category_id = $4, status = $5,
			assignee_id = $6, room = $7, due_at = $8, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at`,
		ticket.ID, ticket.Title, ticket.Description, ticket.CategoryID, ticket.Status,
		ticket.AssigneeID, ticket.Room, ticket.DueAt,
	))
}

func (p *Postgres) UpdateTicketInTx(ctx context.Context, tx Tx, ticket domain.Ticket) (domain.Ticket, error) {
	ptx, ok := tx.(*pgxTx)
	if !ok {
		return domain.Ticket{}, fmt.Errorf("invalid transaction type")
	}
	return scanTicket(ptx.tx.QueryRow(ctx, `
		UPDATE tickets SET title = $2, description = $3, category_id = $4, status = $5,
			assignee_id = $6, room = $7, due_at = $8, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at`,
		ticket.ID, ticket.Title, ticket.Description, ticket.CategoryID, ticket.Status,
		ticket.AssigneeID, ticket.Room, ticket.DueAt,
	))
}

func (p *Postgres) buildListQuery(filter TicketFilter) (string, []any) {
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, title, description, category_id, status, author_id, assignee_id, room, due_at, created_at, updated_at
		FROM tickets WHERE deleted_at IS NULL`)

	args := make([]any, 0, 3)
	n := 1
	if filter.Status != nil {
		fmt.Fprintf(&sb, " AND status = $%d", n)
		args = append(args, string(*filter.Status))
		n++
	}
	if filter.Assignee != nil {
		fmt.Fprintf(&sb, " AND assignee_id = $%d", n)
		args = append(args, *filter.Assignee)
		n++
	}
	if filter.Overdue {
		sb.WriteString(" AND due_at IS NOT NULL AND due_at < now() AND status NOT IN ('closed', 'cancelled')")
	}
	sb.WriteString(" ORDER BY created_at DESC")
	return sb.String(), args
}

func (p *Postgres) ListTickets(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error) {
	query, args := p.buildListQuery(filter)
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []domain.Ticket
	for rows.Next() {
		var t domain.Ticket
		var status string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.CategoryID, &status,
			&t.AuthorID, &t.AssigneeID, &t.Room, &t.DueAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Status = domain.TicketStatus(status)
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (p *Postgres) ReportTickets(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error) {
	return p.ListTickets(ctx, filter)
}

func (p *Postgres) SoftDeleteTicket(ctx context.Context, id string) error {
	tag, err := p.pool.Exec(ctx, `UPDATE tickets SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (p *Postgres) AddComment(ctx context.Context, comment domain.Comment) (domain.Comment, error) {
	err := p.pool.QueryRow(ctx, `
		INSERT INTO comments (ticket_id, author_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, ticket_id, author_id, body, created_at`,
		comment.TicketID, comment.AuthorID, comment.Body,
	).Scan(&comment.ID, &comment.TicketID, &comment.AuthorID, &comment.Body, &comment.CreatedAt)
	return comment, err
}

func (p *Postgres) ListComments(ctx context.Context, ticketID string) ([]domain.Comment, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, ticket_id, author_id, body, created_at
		FROM comments WHERE ticket_id = $1 ORDER BY created_at ASC`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (p *Postgres) LinkTickets(ctx context.Context, ticketID, linkedID string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO ticket_links (ticket_id, linked_ticket_id)
		VALUES ($1, $2), ($2, $1)
		ON CONFLICT DO NOTHING`, ticketID, linkedID)
	return err
}

func (p *Postgres) UnlinkTickets(ctx context.Context, ticketID, linkedID string) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM ticket_links
		WHERE (ticket_id = $1 AND linked_ticket_id = $2) OR (ticket_id = $2 AND linked_ticket_id = $1)`,
		ticketID, linkedID)
	return err
}

func (p *Postgres) ListLinks(ctx context.Context, ticketID string) ([]domain.TicketLink, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT ticket_id, linked_ticket_id, created_at
		FROM ticket_links WHERE ticket_id = $1`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []domain.TicketLink
	for rows.Next() {
		var l domain.TicketLink
		if err := rows.Scan(&l.TicketID, &l.LinkedTicketID, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (p *Postgres) CreateCategory(ctx context.Context, name string) (domain.Category, error) {
	var c domain.Category
	err := p.pool.QueryRow(ctx, `
		INSERT INTO categories (name) VALUES ($1)
		RETURNING id, name, is_active, created_at`, name,
	).Scan(&c.ID, &c.Name, &c.IsActive, &c.CreatedAt)
	return c, err
}

func (p *Postgres) ListCategories(ctx context.Context, activeOnly bool) ([]domain.Category, error) {
	query := `SELECT id, name, is_active, created_at FROM categories`
	if activeOnly {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY name`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (p *Postgres) GetCategoryByID(ctx context.Context, id string) (domain.Category, error) {
	var c domain.Category
	err := p.pool.QueryRow(ctx, `
		SELECT id, name, is_active, created_at FROM categories WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.IsActive, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Category{}, domain.ErrNotFound
	}
	return c, err
}