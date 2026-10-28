package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// IncidentService runs the fault register: report, acknowledge, assign, work,
// restore, verify, close, with a history of every step.
type IncidentService struct {
	incidents ports.IncidentRepository
	sites     ports.SiteRepository
	users     ports.UserRepository
	tx        ports.TxManager
	audit     *AuditService
	clock     ports.Clock
}

func NewIncidentService(
	incidents ports.IncidentRepository, sites ports.SiteRepository, users ports.UserRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock,
) *IncidentService {
	return &IncidentService{incidents: incidents, sites: sites, users: users, tx: tx, audit: audit, clock: clock}
}

// IncidentView is an incident with the history the caller may see.
type IncidentView struct {
	Incident domain.Incident
	Updates  []domain.IncidentUpdate
}

type IncidentInput struct {
	SiteID      string
	Category    domain.IncidentCategory
	Severity    domain.Severity
	Impact      domain.Impact
	Title       string
	Description string
	StartedAt   time.Time
	// DetectedAt defaults to StartedAt.
	DetectedAt *time.Time
}

type IncidentUpdateInput struct {
	Body            string
	CustomerVisible bool
	NextAction      *string
	NextUpdateDue   *time.Time
	// RestorationTarget is a plan only; it never counts as restoration.
	RestorationTarget *time.Time
}

type AssignInput struct {
	OwnerID       string
	NextAction    string
	NextUpdateDue *time.Time
	Note          string
}

type ListIncidentsInput struct {
	SiteID   string
	OwnerID  string
	Statuses []domain.IncidentStatus
	Severity domain.Severity
	OpenOnly bool
	Overdue  bool
	Limit    int
	Offset   int
}

func (s *IncidentService) canView(actor domain.Actor) bool {
	return actor.Can(domain.PermIncidentsRead) || actor.Can(domain.PermPublishedRead)
}

// scope applies site scope and, for the customer, the published-only rule.
func (s *IncidentService) scope(actor domain.Actor) ports.IncidentFilter {
	return ports.IncidentFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(),
		PublishedOnly: !actor.Can(domain.PermIncidentsRead),
	}
}

// load fetches an incident and hides it from people outside its site scope,
// and from the customer until it is published.
func (s *IncidentService) load(ctx context.Context, actor domain.Actor, id string, forUpdate bool) (domain.Incident, error) {
	if !s.canView(actor) {
		return domain.Incident{}, apperrors.ErrForbidden
	}
	i, err := s.incidents.GetByID(ctx, id, forUpdate)
	if err != nil {
		return domain.Incident{}, err
	}
	if !actor.CanAccessSite(i.SiteID) {
		return domain.Incident{}, apperrors.ErrNotFound
	}
	if !actor.Can(domain.PermIncidentsRead) && i.PublishedAt == nil {
		return domain.Incident{}, apperrors.ErrNotFound
	}
	return i, nil
}

func (s *IncidentService) Report(ctx context.Context, actor domain.Actor, in IncidentInput) (domain.Incident, error) {
	if !actor.Can(domain.PermIncidentsCreate) {
		return domain.Incident{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(in.SiteID) {
		return domain.Incident{}, apperrors.ErrNotFound
	}
	now := s.clock.Now()
	inc := domain.Incident{
		SiteID: in.SiteID, Category: in.Category, Severity: in.Severity, Impact: in.Impact,
		Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description),
		StartedAt: in.StartedAt, DetectedAt: in.StartedAt, ReporterID: actor.UserID,
	}
	if in.DetectedAt != nil {
		inc.DetectedAt = *in.DetectedAt
	}
	if problems := inc.ValidateNew(now); len(problems) > 0 {
		return domain.Incident{}, invalidProblems(problems)
	}

	var created domain.Incident
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.sites.GetByID(ctx, in.SiteID); err != nil {
			return err
		}
		var err error
		if created, err = s.incidents.Create(ctx, inc); err != nil {
			return err
		}
		if _, err := s.incidents.AddUpdate(ctx, domain.IncidentUpdate{
			IncidentID: created.ID, AuthorID: actor.UserID, Kind: domain.UpdateStatus,
			ToStatus: domain.IncidentReported, Body: "Incident reported",
		}); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.reported", created, nil))
	})
	return created, err
}

func (s *IncidentService) event(actor domain.Actor, action string, i domain.Incident, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{"site_id": i.SiteID, "number": i.Number, "severity": string(i.Severity)}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "incident", EntityID: i.ID, Metadata: meta}
}

// Get returns an incident with its history. The customer sees only the
// updates written for them.
func (s *IncidentService) Get(ctx context.Context, actor domain.Actor, id string) (IncidentView, error) {
	i, err := s.load(ctx, actor, id, false)
	if err != nil {
		return IncidentView{}, err
	}
	updates, err := s.incidents.Updates(ctx, id, !actor.Can(domain.PermIncidentsRead))
	if err != nil {
		return IncidentView{}, err
	}
	return IncidentView{Incident: i, Updates: updates}, nil
}

func (s *IncidentService) List(ctx context.Context, actor domain.Actor, in ListIncidentsInput) ([]domain.Incident, int, error) {
	if !s.canView(actor) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := s.scope(actor)
	f.SiteID, f.OwnerID, f.Statuses, f.Severity, f.OpenOnly = in.SiteID, in.OwnerID, in.Statuses, in.Severity, in.OpenOnly
	if in.Overdue {
		now := s.clock.Now()
		f.OverdueAt = &now
	}
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	if f.SiteID != "" && !actor.CanAccessSite(f.SiteID) {
		return []domain.Incident{}, 0, nil
	}
	return s.incidents.List(ctx, f)
}

// Summary gives the headline counts for the caller's scope.
func (s *IncidentService) Summary(ctx context.Context, actor domain.Actor) (ports.IncidentCounts, error) {
	if !s.canView(actor) {
		return ports.IncidentCounts{}, apperrors.ErrForbidden
	}
	return s.incidents.Counts(ctx, s.scope(actor), s.clock.Now())
}

// Assign names the owner and, optionally, the next action and when the next
// update is due. Assigning an unacknowledged incident moves it to assigned.
func (s *IncidentService) Assign(ctx context.Context, actor domain.Actor, id string, in AssignInput) (domain.Incident, error) {
	if !actor.Can(domain.PermIncidentsManage) {
		return domain.Incident{}, apperrors.ErrForbidden
	}
	var out domain.Incident
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		i, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if !i.Open() {
			return conflict("this incident is closed")
		}
		if err := s.checkOwner(ctx, i.SiteID, in.OwnerID); err != nil {
			return err
		}
		now := s.clock.Now()
		if in.NextUpdateDue != nil && !in.NextUpdateDue.After(now) {
			return invalid("the next update time must be in the future")
		}
		from := i.Status
		i.OwnerID = in.OwnerID
		if in.NextAction != "" {
			i.NextAction = in.NextAction
		}
		if in.NextUpdateDue != nil {
			i.NextUpdateDue = in.NextUpdateDue
		}
		if i.AssignmentAdvances() {
			i.Status = domain.IncidentAssigned
		}
		i.UpdatedAt = now
		if err := s.incidents.Save(ctx, i); err != nil {
			return err
		}
		if _, err := s.incidents.AddUpdate(ctx, domain.IncidentUpdate{
			IncidentID: id, AuthorID: actor.UserID, Kind: domain.UpdateAssignment,
			FromStatus: from, ToStatus: i.Status, Body: in.Note,
		}); err != nil {
			return err
		}
		out = i
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.assigned", i, map[string]any{"owner_id": in.OwnerID}))
	})
	return out, err
}

// checkOwner makes sure the owner is a real, active person who can see this
// incident, so nothing is assigned to someone who cannot act on it.
func (s *IncidentService) checkOwner(ctx context.Context, siteID, ownerID string) error {
	owner, err := s.users.GetByID(ctx, ownerID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return invalid("the chosen owner does not exist")
		}
		return err
	}
	if !owner.Active {
		return invalid("the chosen owner is deactivated")
	}
	perms, err := s.users.PermissionsForRole(ctx, owner.Role)
	if err != nil {
		return err
	}
	var canRead, allSites bool
	for _, p := range perms {
		switch p {
		case domain.PermIncidentsRead:
			canRead = true
		case domain.PermSitesAll:
			allSites = true
		}
	}
	if !canRead {
		return invalid("the chosen owner cannot work on incidents")
	}
	if allSites {
		return nil
	}
	assigned, err := s.users.SiteIDsForUser(ctx, ownerID)
	if err != nil {
		return err
	}
	for _, id := range assigned {
		if id == siteID {
			return nil
		}
	}
	return invalid("the chosen owner is not assigned to this site")
}

// Transition moves an incident along its workflow. Repeating a step that has
// already happened changes nothing, so a retry after a dropped connection is
// safe. A field officer may record restoration; everything else, including
// verification, belongs to a supervisor, and verification must come from a
// different person than the one who recorded restoration.
func (s *IncidentService) Transition(ctx context.Context, actor domain.Actor, id string, t domain.Transition) (domain.Incident, error) {
	allowed := actor.Can(domain.PermIncidentsManage) ||
		(t.To == domain.IncidentRestored && actor.Can(domain.PermIncidentsCreate))
	if !allowed {
		return domain.Incident{}, apperrors.ErrForbidden
	}
	if t.To == domain.IncidentAssigned {
		return domain.Incident{}, invalid("name an owner to assign the incident")
	}
	if !domain.ValidIncidentStatus(t.To) || t.To == domain.IncidentReported {
		return domain.Incident{}, invalid("choose a valid next status")
	}

	var out domain.Incident
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		i, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if i.Status == t.To {
			out = i
			return nil
		}
		if !i.CanMoveTo(t.To) {
			return conflict("an incident that is %s cannot move to %s", statusWord(i.Status), statusWord(t.To))
		}
		now := s.clock.Now()
		if problems := i.CheckTransition(t, actor.UserID, now); len(problems) > 0 {
			return invalidProblems(problems)
		}
		from := i.Status
		i.ApplyTransition(t, actor.UserID, now)
		if _, err := s.incidents.AddUpdate(ctx, domain.IncidentUpdate{
			IncidentID: id, AuthorID: actor.UserID, Kind: domain.UpdateStatus,
			FromStatus: from, ToStatus: t.To, Body: strings.TrimSpace(t.Note), ObservedAt: t.At,
		}); err != nil {
			return err
		}
		if err := s.incidents.Save(ctx, i); err != nil {
			return err
		}
		out = i
		return s.audit.RecordInTx(ctx, s.event(actor, "incident."+string(t.To), i, map[string]any{"from": string(from)}))
	})
	return out, err
}

func statusWord(s domain.IncidentStatus) string { return strings.ReplaceAll(string(s), "_", " ") }

// AddUpdate appends a note. Supervisors can also change the plan (next action,
// next update time, restoration target) and write an update for the customer.
func (s *IncidentService) AddUpdate(ctx context.Context, actor domain.Actor, id string, in IncidentUpdateInput) (domain.IncidentUpdate, error) {
	plan := in.NextAction != nil || in.NextUpdateDue != nil || in.RestorationTarget != nil
	if !actor.Can(domain.PermIncidentsCreate) && !actor.Can(domain.PermIncidentsManage) {
		return domain.IncidentUpdate{}, apperrors.ErrForbidden
	}
	if (plan || in.CustomerVisible) && !actor.Can(domain.PermIncidentsManage) {
		return domain.IncidentUpdate{}, apperrors.ErrForbidden
	}
	body := strings.TrimSpace(in.Body)
	if body == "" && !plan {
		return domain.IncidentUpdate{}, invalid("write the update")
	}

	var out domain.IncidentUpdate
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		i, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if !i.Open() {
			return conflict("this incident is closed")
		}
		if in.CustomerVisible && i.PublishedAt == nil {
			return conflict("publish the incident to Airtel before sending them an update")
		}
		now := s.clock.Now()
		if in.NextUpdateDue != nil && !in.NextUpdateDue.After(now) {
			return invalid("the next update time must be in the future")
		}
		if in.RestorationTarget != nil && in.RestorationTarget.Before(i.StartedAt) {
			return invalid("the restoration target cannot be before the fault started")
		}
		if plan {
			if in.NextAction != nil {
				i.NextAction = strings.TrimSpace(*in.NextAction)
			}
			if in.NextUpdateDue != nil {
				i.NextUpdateDue = in.NextUpdateDue
			}
			if in.RestorationTarget != nil {
				i.RestorationTarget = in.RestorationTarget
			}
			i.UpdatedAt = now
			if err := s.incidents.Save(ctx, i); err != nil {
				return err
			}
		}
		kind := domain.UpdateNote
		switch {
		case in.CustomerVisible:
			kind = domain.UpdateCustomer
		case plan:
			kind = domain.UpdatePlan
		}
		out, err = s.incidents.AddUpdate(ctx, domain.IncidentUpdate{
			IncidentID: id, AuthorID: actor.UserID, Kind: kind, FromStatus: i.Status, ToStatus: i.Status,
			Body: body, CustomerVisible: in.CustomerVisible,
		})
		if err != nil {
			return err
		}
		if kind == domain.UpdateCustomer || plan {
			return s.audit.RecordInTx(ctx, s.event(actor, "incident.updated", i, map[string]any{"kind": string(kind)}))
		}
		return nil
	})
	return out, err
}

// Publish shares the incident with Airtel. It does not wait for the daily
// report to be approved. Publishing twice changes nothing.
func (s *IncidentService) Publish(ctx context.Context, actor domain.Actor, id string) (domain.Incident, error) {
	if !actor.Can(domain.PermIncidentsManage) {
		return domain.Incident{}, apperrors.ErrForbidden
	}
	var out domain.Incident
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		i, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if i.PublishedAt != nil {
			out = i
			return nil
		}
		if !i.Open() {
			return conflict("this incident is closed")
		}
		now := s.clock.Now()
		i.PublishedAt, i.UpdatedAt = &now, now
		if err := s.incidents.Save(ctx, i); err != nil {
			return err
		}
		out = i
		return s.audit.RecordInTx(ctx, s.event(actor, "incident.published", i, nil))
	})
	return out, err
}
