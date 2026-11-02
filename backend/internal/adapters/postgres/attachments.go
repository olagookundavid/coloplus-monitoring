package postgres

import (
	"context"
	"fmt"
	"time"

	"coloplus-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

// AttachmentRepository implements ports.AttachmentRepository.
type AttachmentRepository struct{ db *DB }

func NewAttachmentRepository(db *DB) *AttachmentRepository { return &AttachmentRepository{db: db} }

const attachmentColumns = `id::text, site_id::text, owner_type, owner_id::text, filename, content_type, size_bytes,
	sha256, storage_key, status, internal, party, uploaded_by::text, scan_status, created_at, confirmed_at`

func scanAttachment(row pgx.Row) (domain.Attachment, error) {
	var a domain.Attachment
	var owner, status, party string
	err := row.Scan(&a.ID, &a.SiteID, &owner, &a.OwnerID, &a.Filename, &a.ContentType, &a.SizeBytes,
		&a.SHA256, &a.StorageKey, &status, &a.Internal, &party, &a.UploadedBy, &a.ScanStatus, &a.CreatedAt, &a.ConfirmedAt)
	a.OwnerType, a.Status, a.Party = domain.AttachmentOwner(owner), domain.AttachmentStatus(status), domain.DisputeParty(party)
	return a, err
}

func (r *AttachmentRepository) Create(ctx context.Context, a domain.Attachment) (domain.Attachment, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO attachments (site_id, owner_type, owner_id, filename, content_type, size_bytes, sha256, storage_key,
		                          internal, party, uploaded_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING `+attachmentColumns,
		a.SiteID, string(a.OwnerType), a.OwnerID, a.Filename, a.ContentType, a.SizeBytes, a.SHA256, a.StorageKey,
		a.Internal, string(a.Party), a.UploadedBy, a.CreatedAt.UTC())
	created, err := scanAttachment(row)
	return created, mapErr(err, "")
}

func (r *AttachmentRepository) GetByID(ctx context.Context, id string, forUpdate bool) (domain.Attachment, error) {
	q := `SELECT ` + attachmentColumns + ` FROM attachments WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	a, err := scanAttachment(r.db.q(ctx).QueryRow(ctx, q, id))
	return a, mapErr(err, "")
}

func (r *AttachmentRepository) SetStatus(ctx context.Context, id string, status domain.AttachmentStatus, at time.Time) (bool, error) {
	var confirmed *time.Time
	if status == domain.AttachmentReady {
		t := at.UTC()
		confirmed = &t
	}
	tag, err := r.db.q(ctx).Exec(ctx,
		`UPDATE attachments SET status = $2, confirmed_at = $3 WHERE id = $1 AND status = 'pending'`,
		id, string(status), confirmed)
	if err != nil {
		return false, mapErr(err, "")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *AttachmentRepository) ListReady(ctx context.Context, owner domain.AttachmentOwner, ownerID string, withInternal bool) ([]domain.Attachment, error) {
	q := `SELECT ` + attachmentColumns + ` FROM attachments WHERE owner_type = $1 AND owner_id = $2 AND status = 'ready'`
	if !withInternal {
		q += ` AND NOT internal`
	}
	rows, err := r.db.q(ctx).Query(ctx, q+` ORDER BY confirmed_at, id`, string(owner), ownerID)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning attachment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AttachmentRepository) CountForOwner(ctx context.Context, owner domain.AttachmentOwner, ownerID string) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM attachments WHERE owner_type = $1 AND owner_id = $2 AND status IN ('pending', 'ready')`,
		string(owner), ownerID).Scan(&n)
	return n, mapErr(err, "")
}

func (r *AttachmentRepository) CountPendingBy(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx,
		`SELECT count(*) FROM attachments WHERE uploaded_by = $1 AND status = 'pending'`, userID).Scan(&n)
	return n, mapErr(err, "")
}

func (r *AttachmentRepository) StalePending(ctx context.Context, before time.Time, limit int) ([]domain.Attachment, error) {
	rows, err := r.db.q(ctx).Query(ctx,
		`SELECT `+attachmentColumns+` FROM attachments WHERE status = 'pending' AND created_at < $1
		 ORDER BY created_at LIMIT $2`, before.UTC(), limit)
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()
	out := []domain.Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning attachment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
