package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// MaintenanceService runs the planned-maintenance workflow. An approved window
// is taken out of a site's eligible time, which can only help the figures, so
// it needs notice, a second person's approval and stays within a monthly cap.
type MaintenanceService struct {
	windows  ports.MaintenanceRepository
	sites    ports.SiteRepository
	users    ports.UserRepository
	tx       ports.TxManager
	audit    *AuditService
	clock    ports.Clock
	notifier *NotificationService
	loc      *time.Location
	baseURL  string
	rules    domain.MaintenanceRules
}

func NewMaintenanceService(
	windows ports.MaintenanceRepository, sites ports.SiteRepository, users ports.UserRepository, tx ports.TxManager,
	audit *AuditService, clock ports.Clock, notifier *NotificationService, loc *time.Location, baseURL string,
	rules domain.MaintenanceRules,
) *MaintenanceService {
	return &MaintenanceService{
		windows: windows, sites: sites, users: users, tx: tx, audit: audit, clock: clock, notifier: notifier,
		loc: loc, baseURL: strings.TrimRight(baseURL, "/"), rules: rules,
	}
}

// Rules returns the limits requests are held to, for the form to show.
func (s *MaintenanceService) Rules() domain.MaintenanceRules { return s.rules }

type MaintenanceInput struct {
	StartsAt, EndsAt time.Time
	Reason           string
}

// Request asks for a window. It is checked against the notice period, the
// length limit, the one-day limit and the site's monthly cap, and it may not
// overlap another window of the same site.
func (s *MaintenanceService) Request(ctx context.Context, actor domain.Actor, siteID string, in MaintenanceInput) (domain.MaintenanceWindow, error) {
	if !actor.Can(domain.PermMaintenanceRequest) {
		return domain.MaintenanceWindow{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.MaintenanceWindow{}, apperrors.ErrNotFound
	}
	site, err := s.sites.GetByID(ctx, siteID)
	if err != nil {
		return domain.MaintenanceWindow{}, err
	}
	if site.Status == domain.SiteSuspended {
		return domain.MaintenanceWindow{}, invalid("this site is suspended")
	}
	now := s.clock.Now()
	in.Reason = strings.TrimSpace(in.Reason)

	var created domain.MaintenanceWindow
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		monthStart := time.Date(in.StartsAt.In(s.loc).Year(), in.StartsAt.In(s.loc).Month(), 1, 0, 0, 0, 0, s.loc)
		used, err := s.windows.Booked(ctx, siteID, monthStart, monthStart.AddDate(0, 1, 0))
		if err != nil {
			return err
		}
		if problems := s.rules.Validate(in.StartsAt, in.EndsAt, in.Reason, now, s.loc, used); len(problems) > 0 {
			return invalidProblems(problems)
		}
		if n, err := s.windows.Overlapping(ctx, siteID, in.StartsAt, in.EndsAt); err != nil {
			return err
		} else if n > 0 {
			return conflict("this site already has planned maintenance at that time")
		}
		if created, err = s.windows.Create(ctx, domain.MaintenanceWindow{
			SiteID: siteID, StartsAt: in.StartsAt, EndsAt: in.EndsAt, Reason: in.Reason,
			RequestedBy: actor.UserID, RequestedAt: now,
		}); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "maintenance.requested", EntityType: "maintenance_window", EntityID: created.ID,
			Metadata: map[string]any{"site_id": siteID, "starts_at": in.StartsAt.UTC().Format(time.RFC3339), "ends_at": in.EndsAt.UTC().Format(time.RFC3339)},
		})
	})
	if err != nil {
		return domain.MaintenanceWindow{}, err
	}

	// Tell the people who can approve it, except the person who asked.
	approvers, err := s.users.UsersWithPermission(ctx, domain.PermMaintenanceApprove, siteID)
	if err == nil {
		var to []domain.User
		for _, u := range approvers {
			if u.ID != actor.UserID {
				to = append(to, u)
			}
		}
		subject := "Planned maintenance to approve: " + siteLabel(site)
		body := fmt.Sprintf("Planned maintenance has been asked for at %s.\n\nWhen: %s\nWhy: %s\n\nWhile it is approved this time is taken out of the site's eligible time, so please check it is genuine and was asked for in good time.\n\nOpen it: %s\n",
			siteLabel(site), s.when(created), created.Reason, s.baseURL+"/maintenance")
		_, _ = s.notifier.EnqueueFor(ctx, to, "maintenance.requested", "maintenance-requested:"+created.ID, subject, body)
	}
	return created, nil
}

func (s *MaintenanceService) when(w domain.MaintenanceWindow) string {
	return fmt.Sprintf("%s to %s", w.StartsAt.In(s.loc).Format("2 Jan 2006 15:04"), w.EndsAt.In(s.loc).Format("15:04"))
}

// Approve accepts a requested window. Never the person who asked.
func (s *MaintenanceService) Approve(ctx context.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
	return s.decide(ctx, actor, id, domain.MaintenanceApproved, note)
}

// Reject turns a requested window down; the reason is required.
func (s *MaintenanceService) Reject(ctx context.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
	if len(strings.TrimSpace(note)) < 5 {
		return domain.MaintenanceWindow{}, invalid("say why the window is being rejected")
	}
	return s.decide(ctx, actor, id, domain.MaintenanceRejected, note)
}

// Cancel withdraws a requested or approved window that has not started. The
// person who asked or an approver may cancel; the reason is required.
func (s *MaintenanceService) Cancel(ctx context.Context, actor domain.Actor, id, note string) (domain.MaintenanceWindow, error) {
	if len(strings.TrimSpace(note)) < 5 {
		return domain.MaintenanceWindow{}, invalid("say why the window is being cancelled")
	}
	return s.decide(ctx, actor, id, domain.MaintenanceCancelled, note)
}

func (s *MaintenanceService) decide(ctx context.Context, actor domain.Actor, id string, to domain.MaintenanceStatus, note string) (domain.MaintenanceWindow, error) {
	note = strings.TrimSpace(note)
	var out domain.MaintenanceWindow
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		w, err := s.windows.GetByID(ctx, id, true)
		if err != nil {
			return err
		}
		if !actor.CanAccessSite(w.SiteID) || (!actor.Can(domain.PermMaintenanceApprove) && !actor.Can(domain.PermMaintenanceRequest)) {
			return apperrors.ErrNotFound
		}
		var from []domain.MaintenanceStatus
		switch to {
		case domain.MaintenanceApproved, domain.MaintenanceRejected:
			if !actor.Can(domain.PermMaintenanceApprove) {
				return apperrors.ErrForbidden
			}
			if w.RequestedBy == actor.UserID {
				return forbidden("you cannot decide your own request; another person has to")
			}
			from = []domain.MaintenanceStatus{domain.MaintenanceRequested}
		case domain.MaintenanceCancelled:
			if w.RequestedBy != actor.UserID && !actor.Can(domain.PermMaintenanceApprove) {
				return apperrors.ErrForbidden
			}
			if !w.StartsAt.After(s.clock.Now()) {
				return conflict("this maintenance has already started, so it cannot be cancelled")
			}
			from = []domain.MaintenanceStatus{domain.MaintenanceRequested, domain.MaintenanceApproved}
		}
		if to == domain.MaintenanceApproved && !w.StartsAt.After(s.clock.Now()) {
			return conflict("this window has already started; ask for a new one")
		}
		ok, err := s.windows.Decide(ctx, id, from, to, actor.UserID, note, s.clock.Now())
		if err != nil {
			return err
		}
		if !ok {
			return conflict("this window is %s, so it cannot become %s", w.Status, to)
		}
		if out, err = s.windows.GetByID(ctx, id, false); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "maintenance." + string(to), EntityType: "maintenance_window", EntityID: id,
			Metadata: map[string]any{"site_id": w.SiteID, "note": note},
		})
	})
	if err != nil {
		return domain.MaintenanceWindow{}, err
	}

	// Tell the person who asked how it was decided (not when they did it themselves).
	if out.RequestedBy != actor.UserID {
		if requester, err := s.users.GetByID(ctx, out.RequestedBy); err == nil && requester.Active {
			if site, err := s.sites.GetByID(ctx, out.SiteID); err == nil {
				subject := fmt.Sprintf("Planned maintenance %s: %s", to, siteLabel(site))
				body := fmt.Sprintf("Your planned maintenance at %s (%s) is now %s.\n\n%s\n", siteLabel(site), s.when(out), to, orDash(note))
				_, _ = s.notifier.EnqueueFor(ctx, []domain.User{requester}, "maintenance.decided", "maintenance-"+string(to)+":"+out.ID, subject, body)
			}
		}
	}
	return out, nil
}

type ListMaintenanceInput struct {
	SiteID        string
	Status        domain.MaintenanceStatus
	From, To      *time.Time
	Limit, Offset int
}

// List returns windows in the caller's scope, newest start first.
func (s *MaintenanceService) List(ctx context.Context, actor domain.Actor, in ListMaintenanceInput) ([]domain.MaintenanceWindow, int, error) {
	if !actor.Can(domain.PermMaintenanceRead) {
		return nil, 0, apperrors.ErrForbidden
	}
	if in.Status != "" && !domain.ValidMaintenanceStatus(in.Status) {
		return nil, 0, invalid("%q is not a known status", in.Status)
	}
	if in.SiteID != "" && !actor.CanAccessSite(in.SiteID) {
		return nil, 0, apperrors.ErrNotFound
	}
	limit, offset := Page(in.Limit, in.Offset)
	return s.windows.List(ctx, ports.MaintenanceFilter{
		AllSites: actor.AllSites, SiteIDs: actor.SiteIDList(), SiteID: in.SiteID, Status: in.Status,
		From: in.From, To: in.To, Limit: limit, Offset: offset,
	})
}
