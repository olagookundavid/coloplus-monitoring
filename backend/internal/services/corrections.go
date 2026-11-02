package services

import (
	"context"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// CorrectionService lets a reporter ask to correct a single already-recorded
// incident field (when it started, when it was actually restored), and a
// supervisor approve or reject that request. Approving writes the corrected
// value to the incident and the decision to the correction row, in one
// transaction; the original value stays on the correction row permanently.
type CorrectionService struct {
	corrections ports.IncidentCorrectionRepository
	incidents   ports.IncidentRepository
	tx          ports.TxManager
	audit       *AuditService
	clock       ports.Clock
}

func NewCorrectionService(
	corrections ports.IncidentCorrectionRepository, incidents ports.IncidentRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock,
) *CorrectionService {
	return &CorrectionService{corrections: corrections, incidents: incidents, tx: tx, audit: audit, clock: clock}
}

func (s *CorrectionService) event(actor domain.Actor, action string, c domain.IncidentCorrection, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{"site_id": c.SiteID, "incident_id": c.IncidentID, "number": c.Number, "field": string(c.Field)}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "incident_correction", EntityID: c.ID, Metadata: meta}
}

type RequestCorrectionInput struct {
	IncidentID     string
	Field          domain.CorrectionField
	CorrectedValue time.Time
	Reason         string
}

// Request files a correction against a single incident field. The current
// value is snapshotted from the live incident row inside the transaction,
// never taken from the client's claim of what the original is.
func (s *CorrectionService) Request(ctx context.Context, actor domain.Actor, in RequestCorrectionInput) (domain.IncidentCorrection, error) {
	if !actor.Can(domain.PermIncidentsCreate) {
		return domain.IncidentCorrection{}, apperrors.ErrForbidden
	}
	reason := strings.TrimSpace(in.Reason)

	var created domain.IncidentCorrection
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		i, err := s.incidents.GetByID(ctx, in.IncidentID, true)
		if err != nil {
			return err
		}
		if !actor.CanAccessSite(i.SiteID) {
			return apperrors.ErrNotFound
		}
		if !i.Open() {
			return conflict("this incident is closed; it can no longer be corrected here")
		}
		original, ok := domain.FieldValue(i, in.Field)
		if !ok {
			return invalid("restoration has not been recorded yet")
		}
		now := s.clock.Now()
		c := domain.IncidentCorrection{
			IncidentID: i.ID, SiteID: i.SiteID, Field: in.Field,
			OriginalValue: original, CorrectedValue: in.CorrectedValue.UTC(), Reason: reason,
			RequestedBy: actor.UserID,
		}
		if problems := c.ValidateNew(now); len(problems) > 0 {
			return invalidProblems(problems)
		}
		if created, err = s.corrections.Create(ctx, c); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.correction_requested", created, map[string]any{
			"original": original, "corrected": c.CorrectedValue,
		}))
	})
	return created, err
}

// load fetches a correction and hides it from people outside its site scope.
func (s *CorrectionService) load(ctx context.Context, actor domain.Actor, id string, forUpdate bool) (domain.IncidentCorrection, error) {
	if !actor.Can(domain.PermIncidentsRead) {
		return domain.IncidentCorrection{}, apperrors.ErrForbidden
	}
	c, err := s.corrections.GetByID(ctx, id, forUpdate)
	if err != nil {
		return domain.IncidentCorrection{}, err
	}
	if !actor.CanAccessSite(c.SiteID) {
		return domain.IncidentCorrection{}, apperrors.ErrNotFound
	}
	return c, nil
}

func (s *CorrectionService) Get(ctx context.Context, actor domain.Actor, id string) (domain.IncidentCorrection, error) {
	return s.load(ctx, actor, id, false)
}

type ListIncidentCorrectionsInput struct {
	SiteID     string
	IncidentID string
	Statuses   []domain.CorrectionStatus
	Limit      int
	Offset     int
}

func (s *CorrectionService) List(ctx context.Context, actor domain.Actor, in ListIncidentCorrectionsInput) ([]domain.IncidentCorrection, int, error) {
	if !actor.Can(domain.PermIncidentsRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := ports.IncidentCorrectionFilter{AllSites: actor.AllSites, SiteIDs: actor.SiteIDList()}
	f.SiteID, f.IncidentID, f.Statuses = in.SiteID, in.IncidentID, in.Statuses
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	if f.SiteID != "" && !actor.CanAccessSite(f.SiteID) {
		return []domain.IncidentCorrection{}, 0, nil
	}
	return s.corrections.List(ctx, f)
}

// Approve applies the corrected value to the incident and records the
// decision, in one transaction. No UI calls this yet -- supervisor approval
// for incident corrections is a separate, later piece of work -- but it is
// built and permission-gated correctly now.
func (s *CorrectionService) Approve(ctx context.Context, actor domain.Actor, id string) (domain.IncidentCorrection, error) {
	if !actor.Can(domain.PermIncidentsManage) {
		return domain.IncidentCorrection{}, apperrors.ErrForbidden
	}
	var out domain.IncidentCorrection
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if !c.Open() {
			return conflict("this correction has already been decided")
		}
		if c.RequestedBy == actor.UserID {
			return forbidden("you cannot approve your own correction request")
		}
		i, err := s.incidents.GetByID(ctx, c.IncidentID, true)
		if err != nil {
			return err
		}
		// Re-checked here: the incident may have closed since the request was
		// filed, and the closed-incident DB trigger would reject this update.
		if !i.Open() {
			return conflict("this incident is closed; it can no longer be corrected here")
		}
		if problems := c.CheckApply(i); len(problems) > 0 {
			return invalidProblems(problems)
		}
		now := s.clock.Now()
		c.ApplyTo(&i)
		i.UpdatedAt = now
		if err := s.incidents.Save(ctx, i); err != nil {
			return err
		}
		c.Status, c.ReviewedBy, c.ReviewedAt, c.UpdatedAt = domain.CorrectionApproved, actor.UserID, &now, now
		if err := s.corrections.Save(ctx, c); err != nil {
			return err
		}
		out = c
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.correction_approved", c, nil))
	})
	return out, err
}

// Reject declines the correction request; the incident is left unchanged.
func (s *CorrectionService) Reject(ctx context.Context, actor domain.Actor, id, note string) (domain.IncidentCorrection, error) {
	if !actor.Can(domain.PermIncidentsManage) {
		return domain.IncidentCorrection{}, apperrors.ErrForbidden
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return domain.IncidentCorrection{}, invalid("say why this correction is being declined")
	}

	var out domain.IncidentCorrection
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		c, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if !c.Open() {
			return conflict("this correction has already been decided")
		}
		if c.RequestedBy == actor.UserID {
			return forbidden("you cannot decide your own correction request")
		}
		now := s.clock.Now()
		c.Status, c.ReviewedBy, c.ReviewedAt, c.ReviewNote, c.UpdatedAt =
			domain.CorrectionRejected, actor.UserID, &now, note, now
		if err := s.corrections.Save(ctx, c); err != nil {
			return err
		}
		out = c
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.correction_rejected", c, map[string]any{"note": note}))
	})
	return out, err
}
