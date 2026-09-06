package handler

import (
	"encoding/csv"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/oziev02/help-desk/internal/domain"
	"github.com/oziev02/help-desk/internal/handler/dto"
	"github.com/oziev02/help-desk/internal/middleware"
	"github.com/oziev02/help-desk/internal/repository"
	"github.com/oziev02/help-desk/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
	logger  *slog.Logger
}

func NewTicketHandler(tickets *service.TicketService, logger *slog.Logger) *TicketHandler {
	return &TicketHandler{tickets: tickets, logger: logger}
}

func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "list tickets", domain.ErrUnauthorized)
		return
	}
	filter := repository.TicketFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.TicketStatus(s)
		if !st.Valid() {
			respondError(w, h.logger, "list tickets", domain.ErrInvalidInput)
			return
		}
		filter.Status = &st
	}
	if a := r.URL.Query().Get("assignee_id"); a != "" {
		filter.Assignee = &a
	}
	if r.URL.Query().Get("overdue") == "true" {
		filter.Overdue = true
	}

	tickets, err := h.tickets.List(r.Context(), actor, filter)
	if err != nil {
		respondError(w, h.logger, "list tickets", err)
		return
	}
	resp := make([]dto.TicketResponse, len(tickets))
	for i, t := range tickets {
		resp[i] = toTicketResponse(t, nil, nil)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	var req dto.CreateTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	in := service.CreateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Room:        req.Room,
		RoomID:      req.RoomID,
	}
	if req.DueAt != nil {
		t, err := time.Parse(time.RFC3339, *req.DueAt)
		if err != nil {
			respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
			return
		}
		in.DueAt = &t
	}
	ticket, err := h.tickets.Create(r.Context(), actor, in)
	if err != nil {
		respondError(w, h.logger, "create ticket", err)
		return
	}
	writeJSON(w, http.StatusCreated, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "get ticket", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, comments, links, err := h.tickets.Get(r.Context(), actor, id)
	if err != nil {
		respondError(w, h.logger, "get ticket", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, comments, links))
}

func (h *TicketHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.UpdateTicketRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	in := service.UpdateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Room:        req.Room,
		RoomID:      req.RoomID,
	}
	if req.DueAt != nil {
		t, err := time.Parse(time.RFC3339, *req.DueAt)
		if err != nil {
			respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
			return
		}
		in.DueAt = &t
	}
	ticket, err := h.tickets.Update(r.Context(), actor, id, in)
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.tickets.Delete(r.Context(), actor, id); err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TicketHandler) Assign(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.AssignRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	ticket, err := h.tickets.Assign(r.Context(), actor, id, service.AssignInput{AssigneeID: req.AssigneeID})
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Transition(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.TransitionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	ticket, err := h.tickets.Transition(r.Context(), actor, id, service.TransitionInput{
		Status: domain.TicketStatus(req.Status),
	})
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Complete(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.CompleteRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	ticket, err := h.tickets.Complete(r.Context(), actor, id, service.CompleteInput{
		Rating:  req.Rating,
		Comment: req.Comment,
	})
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Refuse(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.RefuseRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	ticket, err := h.tickets.Refuse(r.Context(), actor, id, service.RefuseInput{Reason: req.Reason})
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, err := h.tickets.Reopen(r.Context(), actor, id)
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	ticket, err := h.tickets.Cancel(r.Context(), actor, id)
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusOK, toTicketResponse(ticket, nil, nil))
}

func (h *TicketHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.CommentRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	comment, err := h.tickets.AddComment(r.Context(), actor, id, req.Body)
	if err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	writeJSON(w, http.StatusCreated, toCommentResponse(comment))
}

func (h *TicketHandler) Link(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var req dto.LinkRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, h.logger, "ticket handler", domain.ErrInvalidInput)
		return
	}
	if err := h.tickets.Link(r.Context(), actor, id, service.LinkInput{LinkedTicketID: req.LinkedTicketID}); err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TicketHandler) Unlink(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	linkedID := chi.URLParam(r, "linkedId")
	if err := h.tickets.Unlink(r.Context(), actor, id, linkedID); err != nil {
		respondError(w, h.logger, "ticket handler", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TicketHandler) Report(w http.ResponseWriter, r *http.Request) {
	actor, ok := middleware.ActorFromContext(r.Context())
	if !ok {
		respondError(w, h.logger, "ticket handler", domain.ErrUnauthorized)
		return
	}
	filter := repository.TicketFilter{}
	if r.URL.Query().Get("overdue") == "true" {
		filter.Overdue = true
	}
	tickets, err := h.tickets.Report(r.Context(), actor, filter)
	if err != nil {
		respondError(w, h.logger, "report tickets", err)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=tickets.csv")
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"id", "title", "status", "author_id", "assignee_id", "room", "due_at", "rating", "overdue", "created_at"})
		now := time.Now().UTC()
		for _, t := range tickets {
			assignee := ""
			if t.AssigneeID != nil {
				assignee = *t.AssigneeID
			}
			room := ""
			if t.Room != nil {
				room = *t.Room
			}
			dueAt := ""
			if t.DueAt != nil {
				dueAt = t.DueAt.Format(time.RFC3339)
			}
			rating := ""
			if t.Rating != nil {
				rating = formatInt(*t.Rating)
			}
			_ = cw.Write([]string{
				t.ID, t.Title, string(t.Status), t.AuthorID, assignee, room, dueAt, rating,
				formatBool(t.IsOverdue(now)), t.CreatedAt.Format(time.RFC3339),
			})
		}
		cw.Flush()
		return
	}
	resp := make([]dto.TicketResponse, len(tickets))
	for i, t := range tickets {
		resp[i] = toTicketResponse(t, nil, nil)
	}
	writeJSON(w, http.StatusOK, resp)
}

func toTicketResponse(t domain.Ticket, comments []domain.Comment, links []domain.TicketLink) dto.TicketResponse {
	resp := dto.TicketResponse{
		ID:           t.ID,
		Title:        t.Title,
		Description:  t.Description,
		CategoryID:   t.CategoryID,
		Status:       string(t.Status),
		AuthorID:     t.AuthorID,
		AssigneeID:   t.AssigneeID,
		Room:         t.Room,
		RoomID:       t.RoomID,
		DueAt:        t.DueAt,
		Overdue:      t.IsOverdue(time.Now().UTC()),
		Rating:       t.Rating,
		RefuseReason: t.RefuseReason,
		ReopenCount:  t.ReopenCount,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	for _, c := range comments {
		resp.Comments = append(resp.Comments, toCommentResponse(c))
	}
	for _, l := range links {
		resp.Links = append(resp.Links, dto.LinkResponse{
			LinkedTicketID: l.LinkedTicketID,
			CreatedAt:      l.CreatedAt,
		})
	}
	return resp
}

func toCommentResponse(c domain.Comment) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        c.ID,
		TicketID:  c.TicketID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
	}
}

func formatInt(v int) string {
	return strconv.Itoa(v)
}

func formatBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
