package postgres

import (
	"context"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// TicketRepository implements ports.TicketRepository.
type TicketRepository struct{ db *DB }

func NewTicketRepository(db *DB) *TicketRepository { return &TicketRepository{db: db} }

// The incident is joined so the caller can tell whether it has been published
// to the customer without a query per ticket.
const ticketColumns = `t.id::text, t.number, t.site_id::text, t.category, t.priority, t.subject, t.description,
	t.contact_email, t.status, t.raised_by::text, COALESCE(t.assignee_id::text, ''), t.resolution, t.resolved_at,
	COALESCE(t.resolved_by::text, ''), t.reopen_count, t.client_key, COALESCE(t.incident_id::text, ''),
	COALESCE(i.number, 0), COALESCE(i.published_at IS NOT NULL, FALSE), t.created_at, t.updated_at`

const ticketFrom = ` FROM tickets t LEFT JOIN incidents i ON i.id = t.incident_id`

func scanTicket(row pgx.Row) (domain.Ticket, error) {
	var t domain.Ticket
	var category, priority, status string
	err := row.Scan(&t.ID, &t.Number, &t.SiteID, &category, &priority, &t.Subject, &t.Description,
		&t.ContactEmail, &status, &t.RaisedBy, &t.AssigneeID, &t.Resolution, &t.ResolvedAt,
		&t.ResolvedBy, &t.ReopenCount, &t.ClientKey, &t.IncidentID,
		&t.IncidentNumber, &t.IncidentPublished, &t.CreatedAt, &t.UpdatedAt)
	t.Category, t.Priority, t.Status = domain.TicketCategory(category), domain.TicketPriority(priority), domain.TicketStatus(status)
	return t, err
}

func (r *TicketRepository) Create(ctx context.Context, t domain.Ticket) (domain.Ticket, error) {
	var id string
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO tickets (site_id, category, priority, subject, description, contact_email, raised_by, client_key,
		                      created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9) RETURNING id::text`,
		t.SiteID, string(t.Category), string(t.Priority), t.Subject, t.Description, t.ContactEmail, t.RaisedBy,
		t.ClientKey, t.CreatedAt.UTC()).Scan(&id)
	if err != nil {
		return domain.Ticket{}, mapErr(err, "this request was already sent")
	}
	return r.GetByID(ctx, id, false)
}

func (r *TicketRepository) ByClientKey(ctx context.Context, raisedBy, key string) (domain.Ticket, error) {
	t, err := scanTicket(r.db.q(ctx).QueryRow(ctx,
		`SELECT `+ticketColumns+ticketFrom+` WHERE t.raised_by = $1 AND t.client_key = $2`, raisedBy, key))
	return t, mapErr(err, "")
}

func (r *TicketRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.Ticket, error) {
	q := `SELECT ` + ticketColumns + ticketFrom + ` WHERE t.id = $1`
	if forUpdate {
		// Lock the ticket only; the joined incident is read, not changed.
		q += ` FOR UPDATE OF t`
	}
	t, err := scanTicket(r.db.q(ctx).QueryRow(ctx, q, id))
	return t, mapErr(err, "")
}

func (r *TicketRepository) Save(ctx context.Context, t domain.Ticket) error {
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE tickets SET status = $2, assignee_id = NULLIF($3, '')::uuid, resolution = $4, resolved_at = $5,
		        resolved_by = NULLIF($6, '')::uuid, reopen_count = $7, incident_id = NULLIF($8, '')::uuid, updated_at = $9
		 WHERE id = $1`,
		t.ID, string(t.Status), t.AssigneeID, t.Resolution, t.ResolvedAt, t.ResolvedBy, t.ReopenCount,
		t.IncidentID, t.UpdatedAt.UTC())
	if err != nil {
		return mapErr(err, "")
	}
	if tag.RowsAffected() == 0 {
		return mapErr(pgx.ErrNoRows, "")
	}
	return nil
}

func (r *TicketRepository) AddEvent(ctx context.Context, e domain.TicketEvent) (domain.TicketEvent, error) {
	err := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO ticket_events (ticket_id, author_id, party, kind, from_status, to_status, body, internal, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id::text`,
		e.TicketID, e.AuthorID, string(e.Party), string(e.Kind), string(e.FromStatus), string(e.ToStatus),
		e.Body, e.Internal, e.CreatedAt.UTC()).Scan(&e.ID)
	return e, mapErr(err, "")
}

func (r *TicketRepository) Events(ctx context.Context, ticketID string, publicOnly bool) ([]domain.TicketEvent, error) {
	q := `SELECT id::text, ticket_id::text, author_id::text, party, kind, from_status, to_status, body, internal, created_at
	      FROM ticket_events WHERE ticket_id = $1`
	if publicOnly {
		q += ` AND NOT internal`
	}
	rows, err := r.db.q(ctx).Query(ctx, q+` ORDER BY seq`, ticketID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.TicketEvent{}
	for rows.Next() {
		var e domain.TicketEvent
		var party, kind, from, to string
		if err := rows.Scan(&e.ID, &e.TicketID, &e.AuthorID, &party, &kind, &from, &to, &e.Body, &e.Internal, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning ticket event: %w", err)
		}
		e.Party, e.Kind = domain.DisputeParty(party), domain.TicketEventKind(kind)
		e.FromStatus, e.ToStatus = domain.TicketStatus(from), domain.TicketStatus(to)
		out = append(out, e)
	}
	return out, rows.Err()
}

func ticketWhere(f ports.TicketFilter) (string, []any) {
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", len(args))))
	}
	if !f.AllSites {
		add("t.site_id = ANY(?::uuid[])", nonNil(f.SiteIDs))
	}
	if f.SiteID != "" {
		add("t.site_id = ?::uuid", f.SiteID)
	}
	if f.IncidentID != "" {
		add("t.incident_id = ?::uuid", f.IncidentID)
	}
	if f.Category != "" {
		add("t.category = ?", string(f.Category))
	}
	if f.Priority != "" {
		add("t.priority = ?", string(f.Priority))
	}
	if len(f.Statuses) > 0 {
		statuses := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			statuses[i] = string(s)
		}
		add("t.status = ANY(?::text[])", statuses)
	}
	if f.OpenOnly {
		where = append(where, "t.status <> 'resolved'")
	}
	if len(where) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

func (r *TicketRepository) List(ctx context.Context, f ports.TicketFilter) ([]domain.Ticket, int, error) {
	clause, args := ticketWhere(f)
	q := r.db.q(ctx)

	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM tickets t`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+ticketColumns+ticketFrom+clause+
			fmt.Sprintf(` ORDER BY t.created_at DESC, t.number DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	out := []domain.Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning ticket: %w", err)
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (r *TicketRepository) Counts(ctx context.Context, f ports.TicketFilter) (ports.TicketCounts, error) {
	clause, args := ticketWhere(f)
	var c ports.TicketCounts
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE t.status = 'open'),
		        count(*) FILTER (WHERE t.status = 'in_progress'),
		        count(*) FILTER (WHERE t.status = 'resolved'),
		        count(*) FILTER (WHERE t.status <> 'resolved' AND t.priority = 'critical')
		 FROM tickets t`+clause, args...).Scan(&c.Open, &c.InProgress, &c.Resolved, &c.CriticalOpen)
	return c, mapErr(err, "")
}
