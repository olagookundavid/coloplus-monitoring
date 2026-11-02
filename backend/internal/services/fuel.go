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

// FuelService keeps each site's fuel record and the estimate built from it.
// Engineers say what was delivered and what the tank held; the system works
// out what is left from the generator hours reported each day and the
// litres-per-hour agreed for the site. The result is always an estimate and is
// labelled as one.
type FuelService struct {
	fuel    ports.FuelRepository
	sites   ports.SiteRepository
	reports ports.ReportRepository
	tx      ports.TxManager
	audit   *AuditService
	clock   ports.Clock
	loc     *time.Location
}

func NewFuelService(
	fuel ports.FuelRepository, sites ports.SiteRepository, reports ports.ReportRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, loc *time.Location,
) *FuelService {
	return &FuelService{fuel: fuel, sites: sites, reports: reports, tx: tx, audit: audit, clock: clock, loc: loc}
}

const (
	fuelListLimit = 500
	// fuelLookback bounds how far back the generator hours are read.
	fuelLookbackDays = 400
)

// IsStaff says whether the caller sees Coloplus's side: notes, who recorded
// an entry, and voided entries. A customer sees what was recorded and the estimate.
func (s *FuelService) IsStaff(actor domain.Actor) bool { return !actor.Can(domain.PermPublishedRead) }

type FuelInput struct {
	Kind       domain.FuelKind
	Litres     float64
	OccurredAt time.Time
	Note       string
	ClientKey  string
}

// FuelView is a site's fuel record with the estimate.
type FuelView struct {
	Site     domain.Site
	Estimate domain.FuelEstimate
	Entries  []domain.FuelEntry
}

func (s *FuelService) scopedSite(ctx context.Context, actor domain.Actor, siteID string) (domain.Site, error) {
	if !actor.Can(domain.PermFuelRead) {
		return domain.Site{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.Site{}, apperrors.ErrNotFound
	}
	return s.sites.GetByID(ctx, siteID)
}

// Record adds a delivery or a tank reading. A retry with the same client key
// returns the entry already made.
func (s *FuelService) Record(ctx context.Context, actor domain.Actor, siteID string, in FuelInput) (domain.FuelEntry, error) {
	if !actor.Can(domain.PermFuelRecord) {
		return domain.FuelEntry{}, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.FuelEntry{}, apperrors.ErrNotFound
	}
	key := strings.TrimSpace(in.ClientKey)
	if len(key) > 100 {
		return domain.FuelEntry{}, invalid("the request key is too long")
	}
	if key != "" {
		if existing, err := s.fuel.ByClientKey(ctx, actor.UserID, key); err == nil {
			return existing, nil
		} else if !isNotFound(err) {
			return domain.FuelEntry{}, err
		}
	}
	site, err := s.sites.GetByID(ctx, siteID)
	if err != nil {
		return domain.FuelEntry{}, err
	}
	now := s.clock.Now()
	e := domain.FuelEntry{
		SiteID: siteID, Kind: in.Kind, Litres: in.Litres, OccurredAt: in.OccurredAt,
		Note: strings.TrimSpace(in.Note), RecordedBy: actor.UserID, ClientKey: key, CreatedAt: now,
	}
	if problems := e.ValidateNew(site, now); len(problems) > 0 {
		return domain.FuelEntry{}, invalidProblems(problems)
	}

	var created domain.FuelEntry
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if created, err = s.fuel.Create(ctx, e); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "fuel.recorded", EntityType: "fuel_entry", EntityID: created.ID,
			Metadata: map[string]any{
				"site_id": siteID, "kind": string(created.Kind), "litres": created.Litres,
				"occurred_at": created.OccurredAt.UTC().Format(time.RFC3339),
			},
		})
	})
	if err != nil && key != "" && errors.Is(err, apperrors.ErrConflict) {
		if existing, ferr := s.fuel.ByClientKey(ctx, actor.UserID, key); ferr == nil {
			return existing, nil
		}
	}
	return created, err
}

// Status returns a site's estimate and its recent entries. Coloplus staff also
// see voided entries; a customer sees only what stands.
func (s *FuelService) Status(ctx context.Context, actor domain.Actor, siteID string) (FuelView, error) {
	site, err := s.scopedSite(ctx, actor, siteID)
	if err != nil {
		return FuelView{}, err
	}
	entries, err := s.fuel.List(ctx, siteID, s.IsStaff(actor), fuelListLimit)
	if err != nil {
		return FuelView{}, err
	}
	est, err := s.estimate(ctx, site, entries)
	if err != nil {
		return FuelView{}, err
	}
	return FuelView{Site: site, Estimate: est, Entries: entries}, nil
}

// estimate reads the approved generator hours for the days since the first
// entry and works out the level. A day with no approved report, or none with
// hours, stays unknown; it is never read as no use.
func (s *FuelService) estimate(ctx context.Context, site domain.Site, entries []domain.FuelEntry) (domain.FuelEstimate, error) {
	now := s.clock.Now()
	var earliest time.Time
	for _, e := range entries {
		if !e.Voided() && (earliest.IsZero() || e.OccurredAt.Before(earliest)) {
			earliest = e.OccurredAt
		}
	}
	if earliest.IsZero() || site.ConsumptionLPH == nil {
		return domain.EstimateFuel(site, entries, nil, now, s.loc), nil
	}
	toDay := func(t time.Time) time.Time {
		y, m, d := t.In(s.loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	today := toDay(now)
	from := toDay(earliest)
	if min := today.AddDate(0, 0, -fuelLookbackDays); from.Before(min) {
		from = min
	}
	reps, err := s.reports.ApprovedInRange(ctx, []string{site.ID}, from, today)
	if err != nil {
		return domain.FuelEstimate{}, err
	}
	hours := make(map[string]*float64, len(reps))
	for _, r := range reps {
		h := domain.GeneratorHoursOf(r)
		hours[r.Date.Format("2006-01-02")] = &h
	}
	return domain.EstimateFuel(site, entries, hours, now, s.loc), nil
}

// Void marks an entry as a mistake. It stays in the record, with the reason,
// and no longer counts toward the estimate.
func (s *FuelService) Void(ctx context.Context, actor domain.Actor, id, reason string) (domain.FuelEntry, error) {
	if !actor.Can(domain.PermFuelManage) {
		return domain.FuelEntry{}, apperrors.ErrForbidden
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 {
		return domain.FuelEntry{}, invalid("say why this entry is being voided")
	}
	var out domain.FuelEntry
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		e, err := s.fuel.GetByID(ctx, id, true)
		if err != nil {
			return err
		}
		if !actor.CanAccessSite(e.SiteID) {
			return apperrors.ErrNotFound
		}
		if e.Voided() {
			out = e
			return nil
		}
		if _, err := s.fuel.Void(ctx, id, actor.UserID, reason, s.clock.Now()); err != nil {
			return err
		}
		if out, err = s.fuel.GetByID(ctx, id, false); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "fuel.voided", EntityType: "fuel_entry", EntityID: id,
			Metadata: map[string]any{"site_id": e.SiteID, "reason": reason},
		})
	})
	return out, err
}
