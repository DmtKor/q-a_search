package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourusername/project/internal/tickets"
)

// Pool implements tickets.TicketsRepository using pgxpool.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool returns a repository using the given pool.
func NewPool(pool *pgxpool.Pool) *Pool {
	return &Pool{pool: pool}
}

func (p *Pool) Create(ctx context.Context, t *tickets.Ticket) error {
	query := `
		INSERT INTO tickets (query, category, confidence, status, assigned_to, resolution_notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	return p.pool.QueryRow(ctx, query,
		t.Query, strPtr(t.Category), float64Ptr(t.Confidence), t.Status, strPtr(t.AssignedTo), strPtr(t.ResolutionNotes),
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (p *Pool) GetByID(ctx context.Context, id string) (*tickets.Ticket, error) {
	query := `
		SELECT id, query, category, confidence, status, assigned_to,
		       created_at, updated_at, resolved_at, resolution_notes, converted_to_case_id
		FROM tickets WHERE id = $1`
	var t tickets.Ticket
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Query, &t.Category, &t.Confidence, &t.Status, &t.AssignedTo,
		&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt, &t.ResolutionNotes, &t.ConvertedToCaseID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tickets.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (p *Pool) List(ctx context.Context, filters tickets.ListFilters) ([]tickets.Ticket, error) {
	query := `
		SELECT id, query, category, confidence, status, assigned_to,
		       created_at, updated_at, resolved_at, resolution_notes, converted_to_case_id
		FROM tickets
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR category = $2)
		ORDER BY created_at DESC`
	rows, err := p.pool.Query(ctx, query, filters.Status, filters.Category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []tickets.Ticket
	for rows.Next() {
		var t tickets.Ticket
		if err := rows.Scan(
			&t.ID, &t.Query, &t.Category, &t.Confidence, &t.Status, &t.AssignedTo,
			&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt, &t.ResolutionNotes, &t.ConvertedToCaseID,
		); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func (p *Pool) Update(ctx context.Context, id string, u *tickets.TicketUpdate) (*tickets.Ticket, error) {
	// Build dynamic update: only set non-nil fields; set updated_at; set resolved_at when status = resolved|closed
	cur, err := p.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	status := cur.Status
	assignedTo := cur.AssignedTo
	resolutionNotes := cur.ResolutionNotes
	if u.Status != nil {
		status = *u.Status
	}
	if u.AssignedTo != nil {
		assignedTo = u.AssignedTo
	}
	if u.ResolutionNotes != nil {
		resolutionNotes = u.ResolutionNotes
	}
	query := `
		UPDATE tickets
		SET status = $1, assigned_to = $2, resolution_notes = $3,
		    resolved_at = CASE WHEN $4 IN ('resolved', 'closed') THEN COALESCE(resolved_at, NOW()) ELSE resolved_at END,
		    updated_at = NOW()
		WHERE id = $5
		RETURNING id, query, category, confidence, status, assigned_to,
		          created_at, updated_at, resolved_at, resolution_notes, converted_to_case_id`
	var t tickets.Ticket
	err = p.pool.QueryRow(ctx, query, status, strPtr(assignedTo), strPtr(resolutionNotes), status, id).Scan(
		&t.ID, &t.Query, &t.Category, &t.Confidence, &t.Status, &t.AssignedTo,
		&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt, &t.ResolutionNotes, &t.ConvertedToCaseID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tickets.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (p *Pool) SetConvertedToCaseID(ctx context.Context, ticketID, caseID string) error {
	res, err := p.pool.Exec(ctx,
		`UPDATE tickets SET converted_to_case_id = $1, updated_at = NOW() WHERE id = $2`,
		caseID, ticketID,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return tickets.ErrNotFound
	}
	return nil
}

func strPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func float64Ptr(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
