package services

import (
	"context"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// AttachmentConfig sizes the evidence flow.
type AttachmentConfig struct {
	MaxBytes    int64
	UploadTTL   time.Duration
	DownloadTTL time.Duration
	// PendingTTL is how long an upload may stay unconfirmed before it is
	// treated as abandoned.
	PendingTTL time.Duration
}

// AttachmentService holds evidence files on tickets, incidents, disputes and
// site acceptance. The bytes never pass through this server: a person asks for
// a short-lived upload link, sends the file straight to private storage, and
// confirms. Who may attach or see a file follows who may act on, or see, the
// thing it is attached to, and a file marked internal is never shown to the customer.
type AttachmentService struct {
	attachments ports.AttachmentRepository
	store       ports.ObjectStore // nil when storage is not configured
	tickets     ports.TicketRepository
	incidents   ports.IncidentRepository
	disputes    ports.DisputeRepository
	acceptances ports.AcceptanceRepository
	// reports is set with WithReports, for evidence on daily reports.
	reports ports.ReportRepository
	tx      ports.TxManager
	audit   *AuditService
	clock   ports.Clock
	cfg     AttachmentConfig
}

func NewAttachmentService(
	attachments ports.AttachmentRepository, store ports.ObjectStore, tickets ports.TicketRepository,
	incidents ports.IncidentRepository, disputes ports.DisputeRepository, acceptances ports.AcceptanceRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, cfg AttachmentConfig,
) *AttachmentService {
	return &AttachmentService{
		attachments: attachments, store: store, tickets: tickets, incidents: incidents, disputes: disputes,
		acceptances: acceptances, tx: tx, audit: audit, clock: clock, cfg: cfg,
	}
}

// WithReports lets files be attached to daily reports.
func (s *AttachmentService) WithReports(r ports.ReportRepository) *AttachmentService {
	s.reports = r
	return s
}

// IsStaff says whether the caller sees Coloplus's side: internal files and who
// sent what. Customer users never do.
func (s *AttachmentService) IsStaff(actor domain.Actor) bool {
	return !actor.Can(domain.PermPublishedRead)
}

// Enabled says whether file storage is set up.
func (s *AttachmentService) Enabled() bool { return s.store != nil }

func (s *AttachmentService) unavailable() error {
	return unavailableErr("file upload is not set up yet")
}

type AttachmentInput struct {
	OwnerType   domain.AttachmentOwner
	OwnerID     string
	Filename    string
	ContentType string
	SizeBytes   int64
	SHA256      string
	// Shared lets the customer see a file Coloplus sends. The customer's own files are
	// always shared with Coloplus.
	Shared bool
}

// AttachmentUpload is a pending file and the link to send it to.
type AttachmentUpload struct {
	Attachment domain.Attachment
	Upload     ports.PresignedUpload
}

// AttachmentLink is a short-lived link to read one file.
type AttachmentLink struct {
	URL       string
	ExpiresAt time.Time
	Filename  string
}

// owner is what a file hangs on, as the caller may use it.
type owner struct {
	siteID   string
	canWrite bool
	// frozen owners (a closed incident or dispute) take no new evidence.
	frozen bool
	// coloplusOnly owners keep every file internal, whatever the sender asked.
	coloplusOnly bool
}

// resolve finds the owner and applies the caller's right to see it. A thing
// the caller may not see is not found, so ids cannot be probed.
func (s *AttachmentService) resolve(ctx context.Context, actor domain.Actor, typ domain.AttachmentOwner, id string) (owner, error) {
	switch typ {
	case domain.AttachTicket:
		if !actor.Can(domain.PermTicketsRead) {
			return owner{}, apperrors.ErrForbidden
		}
		t, err := s.tickets.GetByID(ctx, id, false)
		if err != nil {
			return owner{}, err
		}
		if !actor.CanAccessSite(t.SiteID) {
			return owner{}, apperrors.ErrNotFound
		}
		return owner{siteID: t.SiteID, canWrite: actor.Can(domain.PermTicketsRaise) || actor.Can(domain.PermTicketsManage)}, nil

	case domain.AttachIncident:
		staff := actor.Can(domain.PermIncidentsRead)
		if !staff && !actor.Can(domain.PermPublishedRead) {
			return owner{}, apperrors.ErrForbidden
		}
		i, err := s.incidents.GetByID(ctx, id, false)
		if err != nil {
			return owner{}, err
		}
		if !actor.CanAccessSite(i.SiteID) || (!staff && i.PublishedAt == nil) {
			return owner{}, apperrors.ErrNotFound
		}
		return owner{
			siteID: i.SiteID, frozen: !i.Open(),
			canWrite: staff && (actor.Can(domain.PermIncidentsCreate) || actor.Can(domain.PermIncidentsManage)),
		}, nil

	case domain.AttachDispute:
		if !actor.Can(domain.PermDisputesRead) {
			return owner{}, apperrors.ErrForbidden
		}
		d, err := s.disputes.GetByID(ctx, id, false)
		if err != nil {
			return owner{}, err
		}
		if !actor.CanAccessSite(d.SiteID) {
			return owner{}, apperrors.ErrNotFound
		}
		return owner{
			siteID: d.SiteID, frozen: !d.Open(),
			canWrite: actor.Can(domain.PermDisputesRaise) || actor.Can(domain.PermDisputesManage),
		}, nil

	case domain.AttachAcceptance:
		// The owner id is the site's: a site has one acceptance.
		if !actor.Can(domain.PermAcceptanceRead) {
			return owner{}, apperrors.ErrForbidden
		}
		if !actor.CanAccessSite(id) {
			return owner{}, apperrors.ErrNotFound
		}
		if _, err := s.acceptances.GetBySite(ctx, id, false); err != nil {
			return owner{}, err
		}
		return owner{siteID: id, canWrite: actor.Can(domain.PermAcceptanceManage) || actor.Can(domain.PermAcceptanceReview)}, nil

	case domain.AttachReport:
		// Evidence for a daily report is Coloplus's own. The customer reaches a
		// report only once approved, and never its evidence.
		if s.reports == nil || !actor.Can(domain.PermReportsRead) {
			return owner{}, apperrors.ErrForbidden
		}
		r, err := s.reports.GetByID(ctx, id, false)
		if err != nil {
			return owner{}, err
		}
		if !actor.CanAccessSite(r.SiteID) {
			return owner{}, apperrors.ErrNotFound
		}
		return owner{
			siteID: r.SiteID, coloplusOnly: true,
			// An approved report is frozen; a correction is a new revision.
			frozen:   r.Status == domain.ReportApproved || r.Status == domain.ReportSuperseded,
			canWrite: actor.Can(domain.PermReportsCreate),
		}, nil
	}
	return owner{}, invalid("choose what the file belongs to: a ticket, incident, dispute, daily report or acceptance")
}

// Request starts an upload: it checks the file is acceptable and that the
// caller may attach it here, records it as pending and returns the link.
func (s *AttachmentService) Request(ctx context.Context, actor domain.Actor, in AttachmentInput) (AttachmentUpload, error) {
	if s.store == nil {
		return AttachmentUpload{}, s.unavailable()
	}
	party := domain.PartyCustomer
	if s.IsStaff(actor) {
		party = domain.PartyColoplus
	}
	a := domain.Attachment{
		OwnerType: in.OwnerType, OwnerID: strings.TrimSpace(in.OwnerID),
		Filename:    domain.SafeFilename(in.Filename),
		ContentType: strings.ToLower(strings.TrimSpace(in.ContentType)),
		SizeBytes:   in.SizeBytes, SHA256: strings.ToLower(strings.TrimSpace(in.SHA256)),
		Party: party, Internal: party == domain.PartyColoplus && !in.Shared,
		UploadedBy: actor.UserID, CreatedAt: s.clock.Now(),
	}
	if problems := a.ValidateNew(s.cfg.MaxBytes); len(problems) > 0 {
		return AttachmentUpload{}, invalidProblems(problems)
	}

	var created domain.Attachment
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		o, err := s.resolve(ctx, actor, a.OwnerType, a.OwnerID)
		if err != nil {
			return err
		}
		switch {
		case !o.canWrite:
			return apperrors.ErrForbidden
		case o.frozen:
			return conflict("this is closed, so no more files can be added")
		}
		if n, err := s.attachments.CountForOwner(ctx, a.OwnerType, a.OwnerID); err != nil {
			return err
		} else if n >= domain.MaxAttachmentsPerOwner {
			return conflict("this already has the most files allowed (%d)", domain.MaxAttachmentsPerOwner)
		}
		if n, err := s.attachments.CountPendingBy(ctx, actor.UserID); err != nil {
			return err
		} else if n >= domain.MaxPendingPerUser {
			return conflict("you have %d uploads waiting to be confirmed; finish those first", n)
		}
		a.SiteID = o.siteID
		if o.coloplusOnly {
			a.Internal = true
		}
		if a.StorageKey, err = domain.NewStorageKey(o.siteID); err != nil {
			return err
		}
		created, err = s.attachments.Create(ctx, a)
		return err
	})
	if err != nil {
		return AttachmentUpload{}, err
	}

	up, err := s.store.PresignUpload(ports.UploadRequest{
		Key: created.StorageKey, ContentType: created.ContentType, SizeBytes: created.SizeBytes,
		SHA256Hex: created.SHA256, TTL: s.cfg.UploadTTL,
	})
	if err != nil {
		return AttachmentUpload{}, err
	}
	return AttachmentUpload{Attachment: created, Upload: up}, nil
}

// Complete confirms an upload: the stored file must be the size and digest the
// sender declared, or it is discarded. Repeating it is safe.
func (s *AttachmentService) Complete(ctx context.Context, actor domain.Actor, id string) (domain.Attachment, error) {
	if s.store == nil {
		return domain.Attachment{}, s.unavailable()
	}
	a, err := s.attachments.GetByID(ctx, id, false)
	if err != nil {
		return domain.Attachment{}, err
	}
	// Only the sender confirms; to anyone else it does not exist.
	if a.UploadedBy != actor.UserID {
		return domain.Attachment{}, apperrors.ErrNotFound
	}
	switch a.Status {
	case domain.AttachmentReady:
		return a, nil
	case domain.AttachmentPending:
	default:
		return domain.Attachment{}, conflict("that upload was cancelled; start again")
	}
	if _, err := s.resolve(ctx, actor, a.OwnerType, a.OwnerID); err != nil {
		return domain.Attachment{}, err
	}

	// Ask the store outside any transaction.
	info, err := s.store.Stat(ctx, a.StorageKey)
	if err != nil {
		if isNotFound(err) {
			return domain.Attachment{}, invalid("the file has not been uploaded yet; upload it, then confirm")
		}
		return domain.Attachment{}, err
	}
	if info.SizeBytes != a.SizeBytes || (info.SHA256Hex != "" && info.SHA256Hex != a.SHA256) {
		_ = s.store.Delete(ctx, a.StorageKey)
		err := s.tx.WithTx(ctx, func(ctx context.Context) error {
			if _, err := s.attachments.SetStatus(ctx, a.ID, domain.AttachmentRejected, s.clock.Now()); err != nil {
				return err
			}
			return s.audit.RecordInTx(ctx, s.event(actor, "attachment.rejected", a, map[string]any{"stored_bytes": info.SizeBytes}))
		})
		if err != nil {
			return domain.Attachment{}, err
		}
		return domain.Attachment{}, invalid("the uploaded file does not match what you described, so it was discarded; start again")
	}

	var out domain.Attachment
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		cur, err := s.attachments.GetByID(ctx, id, true)
		if err != nil {
			return err
		}
		if cur.Status == domain.AttachmentReady {
			out = cur
			return nil
		}
		ok, err := s.attachments.SetStatus(ctx, id, domain.AttachmentReady, s.clock.Now())
		if err != nil {
			return err
		}
		if !ok {
			return conflict("that upload was cancelled; start again")
		}
		if out, err = s.attachments.GetByID(ctx, id, false); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, s.event(actor, "attachment.added", out, map[string]any{"sha256": out.SHA256}))
	})
	return out, err
}

func (s *AttachmentService) event(actor domain.Actor, action string, a domain.Attachment, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{
		"site_id": a.SiteID, "owner_type": string(a.OwnerType), "owner_id": a.OwnerID,
		"filename": a.Filename, "size_bytes": a.SizeBytes, "internal": a.Internal,
	}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "attachment", EntityID: a.ID, Metadata: meta}
}

// List returns the received files on one owner that the caller may see.
func (s *AttachmentService) List(ctx context.Context, actor domain.Actor, typ domain.AttachmentOwner, ownerID string) ([]domain.Attachment, error) {
	if _, err := s.resolve(ctx, actor, typ, ownerID); err != nil {
		return nil, err
	}
	return s.attachments.ListReady(ctx, typ, ownerID, s.IsStaff(actor))
}

// Download checks the caller may see the file and gives a link that works for
// a couple of minutes. Each one is recorded in the audit log.
func (s *AttachmentService) Download(ctx context.Context, actor domain.Actor, id string) (AttachmentLink, error) {
	if s.store == nil {
		return AttachmentLink{}, s.unavailable()
	}
	a, err := s.attachments.GetByID(ctx, id, false)
	if err != nil {
		return AttachmentLink{}, err
	}
	if _, err := s.resolve(ctx, actor, a.OwnerType, a.OwnerID); err != nil {
		return AttachmentLink{}, err
	}
	// Anything the caller may not see, or that never arrived, does not exist.
	if a.Status != domain.AttachmentReady || (a.Internal && !s.IsStaff(actor)) {
		return AttachmentLink{}, apperrors.ErrNotFound
	}
	url, exp, err := s.store.PresignDownload(ports.DownloadRequest{
		Key: a.StorageKey, Filename: a.Filename, ContentType: a.ContentType, TTL: s.cfg.DownloadTTL,
	})
	if err != nil {
		return AttachmentLink{}, err
	}
	_ = s.audit.Record(ctx, s.event(actor, "attachment.downloaded", a, nil))
	return AttachmentLink{URL: url, ExpiresAt: exp, Filename: a.Filename}, nil
}

// ExpireAbandoned clears uploads that were started and never confirmed. With
// dryRun it only counts them.
func (s *AttachmentService) ExpireAbandoned(ctx context.Context, dryRun bool) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	stale, err := s.attachments.StalePending(ctx, s.clock.Now().Add(-s.cfg.PendingTTL), 200)
	if err != nil {
		return 0, err
	}
	if dryRun {
		return len(stale), nil
	}
	cleared := 0
	for _, a := range stale {
		if ctx.Err() != nil {
			break
		}
		if err := s.store.Delete(ctx, a.StorageKey); err != nil {
			return cleared, err
		}
		if ok, err := s.attachments.SetStatus(ctx, a.ID, domain.AttachmentExpired, s.clock.Now()); err != nil {
			return cleared, err
		} else if ok {
			cleared++
		}
	}
	return cleared, nil
}
