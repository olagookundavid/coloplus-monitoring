package services

import (
	"context"
	"errors"
	"strings"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// TicketService runs support tickets: a request Airtel raises about a site and
// Coloplus works. A ticket is not a fault record. It can be linked to the
// incident Coloplus opens for it, and Airtel only learns of that incident once
// it has been published to them.
type TicketService struct {
	tickets   ports.TicketRepository
	incidents ports.IncidentRepository
	sites     ports.SiteRepository
	users     ports.UserRepository
	tx        ports.TxManager
	audit     *AuditService
	clock     ports.Clock
	notifier  *NotificationService
	baseURL   string
}

func NewTicketService(
	tickets ports.TicketRepository, incidents ports.IncidentRepository, sites ports.SiteRepository,
	users ports.UserRepository, tx ports.TxManager, audit *AuditService, clock ports.Clock,
	notifier *NotificationService, baseURL string,
) *TicketService {
	return &TicketService{
		tickets: tickets, incidents: incidents, sites: sites, users: users, tx: tx, audit: audit,
		clock: clock, notifier: notifier, baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *TicketService) link(t domain.Ticket) string { return s.baseURL + "/support-tickets/" + t.ID }

// TicketView is a ticket with the history the caller may see.
type TicketView struct {
	Ticket domain.Ticket
	Events []domain.TicketEvent
}

type TicketInput struct {
	SiteID       string
	Category     domain.TicketCategory
	Priority     domain.TicketPriority
	Subject      string
	Description  string
	ContactEmail string
	// ClientKey, if given, makes a retried request return the ticket the first
	// one made instead of a second ticket.
	ClientKey string
}

type ListTicketsInput struct {
	SiteID     string
	Statuses   []domain.TicketStatus
	Category   domain.TicketCategory
	Priority   domain.TicketPriority
	IncidentID string
	OpenOnly   bool
	Limit      int
	Offset     int
}

// IsStaff says whether the caller sees Coloplus's side of a ticket: who raised
// and works it and the internal notes. Airtel users never do.
func (s *TicketService) IsStaff(actor domain.Actor) bool {
	return !actor.Can(domain.PermPublishedRead)
}

func (s *TicketService) scope(actor domain.Actor) ports.TicketFilter {
	return ports.TicketFilter{AllSites: actor.AllSites, SiteIDs: actor.SiteIDList()}
}

// load fetches a ticket and hides it from people outside its site scope.
func (s *TicketService) load(ctx context.Context, actor domain.Actor, id string, forUpdate bool) (domain.Ticket, error) {
	if !actor.Can(domain.PermTicketsRead) {
		return domain.Ticket{}, apperrors.ErrForbidden
	}
	t, err := s.tickets.GetByID(ctx, id, forUpdate)
	if err != nil {
		return domain.Ticket{}, err
	}
	if !actor.CanAccessSite(t.SiteID) {
		return domain.Ticket{}, apperrors.ErrNotFound
	}
	return t, nil
}

func (s *TicketService) event(actor domain.Actor, action string, t domain.Ticket, extra map[string]any) domain.AuditEvent {
	meta := map[string]any{"site_id": t.SiteID, "number": t.Number}
	for k, v := range extra {
		meta[k] = v
	}
	return domain.AuditEvent{ActorID: actor.UserID, Action: action, EntityType: "ticket", EntityID: t.ID, Metadata: meta}
}

// Raise opens a ticket at a site in the caller's scope. The contact email
// defaults to the requester's own address; Coloplus never emails the contact
// address, only the requester's account.
func (s *TicketService) Raise(ctx context.Context, actor domain.Actor, in TicketInput) (domain.Ticket, error) {
	if !actor.Can(domain.PermTicketsRaise) {
		return domain.Ticket{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(in.SiteID) {
		return domain.Ticket{}, apperrors.ErrNotFound
	}
	key := strings.TrimSpace(in.ClientKey)
	if len(key) > 100 {
		return domain.Ticket{}, invalid("the request key is too long")
	}
	t := domain.Ticket{
		SiteID: in.SiteID, Category: in.Category, Priority: in.Priority,
		Subject: strings.TrimSpace(in.Subject), Description: strings.TrimSpace(in.Description),
		ContactEmail: strings.TrimSpace(in.ContactEmail), RaisedBy: actor.UserID, ClientKey: key,
		CreatedAt: s.clock.Now(),
	}
	if problems := t.ValidateNew(); len(problems) > 0 {
		return domain.Ticket{}, invalidProblems(problems)
	}

	if key != "" {
		if existing, err := s.tickets.ByClientKey(ctx, actor.UserID, key); err == nil {
			return existing, nil
		} else if !isNotFound(err) {
			return domain.Ticket{}, err
		}
	}

	var created domain.Ticket
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.sites.GetByID(ctx, in.SiteID); err != nil {
			return err
		}
		if t.ContactEmail == "" {
			u, err := s.users.GetByID(ctx, actor.UserID)
			if err != nil {
				return err
			}
			t.ContactEmail = u.Email
		}
		var err error
		if created, err = s.tickets.Create(ctx, t); err != nil {
			return err
		}
		e, err := s.tickets.AddEvent(ctx, domain.TicketEvent{
			TicketID: created.ID, AuthorID: actor.UserID, Party: domain.PartyAirtel, Kind: domain.TicketEventStatus,
			ToStatus: domain.TicketOpen, Body: created.Description, CreatedAt: t.CreatedAt,
		})
		if err != nil {
			return err
		}
		if err := s.audit.RecordInTx(ctx, s.event(actor, "ticket.raised", created,
			map[string]any{"category": string(created.Category), "priority": string(created.Priority)})); err != nil {
			return err
		}
		return s.notifyOtherSide(ctx, created, e)
	})
	if err != nil && key != "" && errors.Is(err, apperrors.ErrConflict) {
		// Two identical requests raced; the other one won. Return its ticket.
		if existing, ferr := s.tickets.ByClientKey(ctx, actor.UserID, key); ferr == nil {
			return existing, nil
		}
	}
	return created, err
}

// Get returns a ticket with its history. Airtel does not see Coloplus's
// internal notes.
func (s *TicketService) Get(ctx context.Context, actor domain.Actor, id string) (TicketView, error) {
	t, err := s.load(ctx, actor, id, false)
	if err != nil {
		return TicketView{}, err
	}
	events, err := s.tickets.Events(ctx, id, !s.IsStaff(actor))
	if err != nil {
		return TicketView{}, err
	}
	return TicketView{Ticket: t, Events: events}, nil
}

func (s *TicketService) List(ctx context.Context, actor domain.Actor, in ListTicketsInput) ([]domain.Ticket, int, error) {
	if !actor.Can(domain.PermTicketsRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	f := s.scope(actor)
	f.SiteID, f.Statuses, f.Category, f.Priority, f.IncidentID, f.OpenOnly =
		in.SiteID, in.Statuses, in.Category, in.Priority, in.IncidentID, in.OpenOnly
	f.Limit, f.Offset = Page(in.Limit, in.Offset)
	if f.SiteID != "" && !actor.CanAccessSite(f.SiteID) {
		return []domain.Ticket{}, 0, nil
	}
	return s.tickets.List(ctx, f)
}

func (s *TicketService) Summary(ctx context.Context, actor domain.Actor) (ports.TicketCounts, error) {
	if !actor.Can(domain.PermTicketsRead) {
		return ports.TicketCounts{}, apperrors.ErrForbidden
	}
	return s.tickets.Counts(ctx, s.scope(actor))
}

// Comment adds to the conversation. Coloplus may mark a comment internal, so
// Airtel never sees it. A resolved ticket can still be commented on.
func (s *TicketService) Comment(ctx context.Context, actor domain.Actor, id, body string, internal bool) (domain.TicketEvent, error) {
	party, ok := domain.TicketPartyOf(actor)
	if !ok {
		return domain.TicketEvent{}, apperrors.ErrForbidden
	}
	if internal && party != domain.PartyColoplus {
		return domain.TicketEvent{}, apperrors.ErrForbidden
	}
	if problems := domain.ValidateTicketComment(body); len(problems) > 0 {
		return domain.TicketEvent{}, invalidProblems(problems)
	}

	var out domain.TicketEvent
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		t, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		out, err = s.tickets.AddEvent(ctx, domain.TicketEvent{
			TicketID: id, AuthorID: actor.UserID, Party: party, Kind: domain.TicketEventComment,
			FromStatus: t.Status, ToStatus: t.Status, Body: strings.TrimSpace(body), Internal: internal,
			CreatedAt: s.clock.Now(),
		})
		if err != nil {
			return err
		}
		if internal {
			return nil
		}
		return s.notifyOtherSide(ctx, t, out)
	})
	return out, err
}

// Transition moves a ticket along its workflow. Repeating a step that has
// already happened changes nothing, so a retry is safe.
//
//	Coloplus: open -> in progress -> resolved (a resolved ticket can be worked again)
//	Airtel:   reopen a resolved ticket (back to open)
func (s *TicketService) Transition(ctx context.Context, actor domain.Actor, id string, step domain.TicketStep) (domain.Ticket, error) {
	party, ok := domain.TicketPartyOf(actor)
	if !ok {
		return domain.Ticket{}, apperrors.ErrForbidden
	}
	if !domain.ValidTicketStatus(step.To) {
		return domain.Ticket{}, invalid("choose a valid next status")
	}

	var out domain.Ticket
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		t, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if t.Status == step.To {
			out = t
			return nil
		}
		if !t.CanMove(party, step.To) {
			return conflict("a ticket that is %s cannot be moved to %s by %s",
				ticketStatusWord(t.Status), ticketStatusWord(step.To), party)
		}
		if problems := t.CheckStep(party, step); len(problems) > 0 {
			return invalidProblems(problems)
		}
		now := s.clock.Now()
		from := t.Status
		t.Apply(party, actor.UserID, step, now)
		e, err := s.tickets.AddEvent(ctx, domain.TicketEvent{
			TicketID: t.ID, AuthorID: actor.UserID, Party: party, Kind: domain.TicketEventStatus,
			FromStatus: from, ToStatus: t.Status, Body: strings.TrimSpace(step.Note), CreatedAt: now,
		})
		if err != nil {
			return err
		}
		if err := s.tickets.Save(ctx, t); err != nil {
			return err
		}
		if err := s.audit.RecordInTx(ctx, s.event(actor, "ticket."+string(t.Status), t, map[string]any{"from": string(from)})); err != nil {
			return err
		}
		if err := s.notifyOtherSide(ctx, t, e); err != nil {
			return err
		}
		out, err = s.tickets.GetByID(ctx, id, false)
		return err
	})
	return out, err
}

// LinkIncident ties the ticket to the incident Coloplus opened for it. The
// incident must be at the same site. The link is noted internally; Airtel sees
// it on the ticket only once the incident has been published to them.
func (s *TicketService) LinkIncident(ctx context.Context, actor domain.Actor, id, incidentID string) (domain.Ticket, error) {
	if !actor.Can(domain.PermTicketsManage) {
		return domain.Ticket{}, apperrors.ErrForbidden
	}
	if strings.TrimSpace(incidentID) == "" {
		return domain.Ticket{}, invalid("choose the incident to link")
	}
	var out domain.Ticket
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		t, err := s.load(ctx, actor, id, true)
		if err != nil {
			return err
		}
		if t.IncidentID == incidentID {
			out = t
			return nil
		}
		inc, err := s.incidents.GetByID(ctx, incidentID, false)
		if err != nil {
			if isNotFound(err) {
				return invalid("that incident does not exist")
			}
			return err
		}
		if inc.SiteID != t.SiteID {
			return invalid("that incident is at a different site")
		}
		now := s.clock.Now()
		prev := t.IncidentID
		t.IncidentID, t.UpdatedAt = incidentID, now
		if err := s.tickets.Save(ctx, t); err != nil {
			return err
		}
		if _, err := s.tickets.AddEvent(ctx, domain.TicketEvent{
			TicketID: t.ID, AuthorID: actor.UserID, Party: domain.PartyColoplus, Kind: domain.TicketEventLink,
			FromStatus: t.Status, ToStatus: t.Status, Body: "Linked to " + incidentRef(inc.Number), Internal: true, CreatedAt: now,
		}); err != nil {
			return err
		}
		extra := map[string]any{"incident_id": incidentID}
		if prev != "" {
			extra["replaced_incident_id"] = prev
		}
		if err := s.audit.RecordInTx(ctx, s.event(actor, "ticket.incident_linked", t, extra)); err != nil {
			return err
		}
		out, err = s.tickets.GetByID(ctx, id, false)
		return err
	})
	return out, err
}

// notifyOtherSide emails the people on the other side of the conversation.
// Coloplus staff on the site hear about Airtel's moves; the person who raised
// the ticket hears about Coloplus's. Internal notes never come here, and the
// contact address typed into the ticket is never emailed.
func (s *TicketService) notifyOtherSide(ctx context.Context, t domain.Ticket, e domain.TicketEvent) error {
	var recipients []domain.User
	if e.Party == domain.PartyColoplus {
		u, err := s.users.GetByID(ctx, t.RaisedBy)
		if err != nil {
			return err
		}
		if u.Active {
			recipients = []domain.User{u}
		}
	} else {
		var err error
		if recipients, err = s.users.UsersWithPermission(ctx, domain.PermTicketsManage, t.SiteID); err != nil {
			return err
		}
	}
	if len(recipients) == 0 {
		return nil
	}
	site, err := s.sites.GetByID(ctx, t.SiteID)
	if err != nil {
		return err
	}
	subject, body := ticketEventEmail(t, site, e, s.link(t))
	_, err = s.notifier.EnqueueFor(ctx, recipients, "ticket.event", "ticket-event:"+e.ID, subject, body)
	return err
}
