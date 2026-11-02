package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// WithSnapshots turns on the register of issued exports. The store is optional:
// without it the register keeps each file's fingerprint but not the file.
func (s *ExportService) WithSnapshots(repo ports.IssuedExportRepository, store ports.ObjectStore) *ExportService {
	s.issued, s.snapshots = repo, store
	return s
}

// IssueInput is a finished export, ready to be handed over.
type IssueInput struct {
	Kind     string
	Filename string
	Params   map[string]any
	RowCount int
	Content  []byte
}

const issuedLinkTTL = 2 * time.Minute

// Issue records an export that is about to be handed over: its fingerprint and,
// when storage is up, the exact file. A storage failure never stops the export,
// since the person is waiting on it; the register then holds the fingerprint
// only, and says so. Recording the fingerprint itself must succeed, because an
// export that cannot be traced is not issued.
func (s *ExportService) Issue(ctx context.Context, actor domain.Actor, in IssueInput) (domain.IssuedExport, error) {
	if !actor.Can(domain.PermExportsCreate) {
		return domain.IssuedExport{}, apperrors.ErrForbidden
	}
	if s.issued == nil {
		return domain.IssuedExport{}, nil
	}
	sum := sha256.Sum256(in.Content)
	e := domain.IssuedExport{
		Kind: in.Kind, Filename: in.Filename, Params: in.Params, RowCount: in.RowCount,
		SizeBytes: int64(len(in.Content)), SHA256: hex.EncodeToString(sum[:]), IssuedBy: actor.UserID, CreatedAt: s.clock.Now(),
	}
	if s.snapshots != nil {
		if key, err := domain.NewObjectKey("exports"); err == nil {
			// Outside any transaction: this is a call to a third party.
			if err := s.snapshots.Put(ctx, key, "text/csv", in.Content); err == nil {
				e.StorageKey = key
			}
		}
	}
	return s.issued.Create(ctx, e)
}

type IssuedList struct {
	Items         []domain.IssuedExport
	Total         int
	Limit, Offset int
}

// Issued lists exports that were handed out: a person's own, or everyone's for
// someone who can read the audit log.
func (s *ExportService) Issued(ctx context.Context, actor domain.Actor, limit, offset int) (IssuedList, error) {
	if s.issued == nil {
		return IssuedList{}, apperrors.ErrNotFound
	}
	own := ""
	switch {
	case actor.Can(domain.PermAuditRead):
	case actor.Can(domain.PermExportsCreate):
		own = actor.UserID
	default:
		return IssuedList{}, apperrors.ErrForbidden
	}
	limit, offset = Page(limit, offset)
	items, total, err := s.issued.List(ctx, own, limit, offset)
	return IssuedList{Items: items, Total: total, Limit: limit, Offset: offset}, err
}

// IssuedLink gives a short-lived link to the exact file that was issued. Only
// the person who issued it, or someone who can read the audit log, may have it.
func (s *ExportService) IssuedLink(ctx context.Context, actor domain.Actor, id string) (url string, exp time.Time, e domain.IssuedExport, err error) {
	if s.issued == nil || s.snapshots == nil {
		return "", time.Time{}, e, apperrors.ErrNotFound
	}
	if !actor.Can(domain.PermAuditRead) && !actor.Can(domain.PermExportsCreate) {
		return "", time.Time{}, e, apperrors.ErrForbidden
	}
	if e, err = s.issued.GetByID(ctx, id); err != nil {
		return "", time.Time{}, e, err
	}
	if e.IssuedBy != actor.UserID && !actor.Can(domain.PermAuditRead) {
		return "", time.Time{}, e, apperrors.ErrNotFound
	}
	if !e.Stored() {
		return "", time.Time{}, e, apperrors.ErrNotFound
	}
	url, exp, err = s.snapshots.PresignDownload(ports.DownloadRequest{
		Key: e.StorageKey, Filename: e.Filename, ContentType: "text/csv", TTL: issuedLinkTTL,
	})
	return url, exp, e, err
}
