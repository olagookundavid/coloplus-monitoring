package services

import (
	"context"
	"errors"
	"fmt"
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
	fuel     ports.FuelRepository
	sites    ports.SiteRepository
	reports  ports.ReportRepository
	users    ports.UserRepository
	tx       ports.TxManager
	audit    *AuditService
	clock    ports.Clock
	notifier *NotificationService
	loc      *time.Location
	baseURL  string
	// lowHours is the run time left at or under which the tank counts as low.
	lowHours float64
}

func NewFuelService(
	fuel ports.FuelRepository, sites ports.SiteRepository, reports ports.ReportRepository, users ports.UserRepository,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, notifier *NotificationService,
	loc *time.Location, baseURL string, lowHours float64,
) *FuelService {
	return &FuelService{
		fuel: fuel, sites: sites, reports: reports, users: users, tx: tx, audit: audit, clock: clock,
		notifier: notifier, loc: loc, baseURL: strings.TrimRight(baseURL, "/"), lowHours: lowHours,
	}
}

const fuelListLimit = 500

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

// FuelView is a site's fuel record with the estimate. Variances compares each
// recent tank reading with what the estimate expected just before it.
type FuelView struct {
	Site      domain.Site
	Estimate  domain.FuelEstimate
	Entries   []domain.FuelEntry
	Variances map[string]*domain.FuelVariance
	LowHours  float64
}

// fuelVarianceReadings is how many of the latest readings are compared.
const fuelVarianceReadings = 10

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
func (s *FuelService) Record(ctx context.Context, actor domain.Actor, siteID string, in FuelInput) (FuelRecorded, error) {
	e, v, err := s.record(ctx, actor, siteID, in)
	return FuelRecorded{Entry: e, Variance: v}, err
}

// FuelRecorded is a new entry and, for a tank reading, how it compares with
// what the estimate expected.
type FuelRecorded struct {
	Entry    domain.FuelEntry
	Variance *domain.FuelVariance
}

func (s *FuelService) record(ctx context.Context, actor domain.Actor, siteID string, in FuelInput) (domain.FuelEntry, *domain.FuelVariance, error) {
	if !actor.Can(domain.PermFuelRecord) {
		return domain.FuelEntry{}, nil, apperrors.ErrForbidden
	}
	if !actor.CanAccessSite(siteID) {
		return domain.FuelEntry{}, nil, apperrors.ErrNotFound
	}
	key := strings.TrimSpace(in.ClientKey)
	if len(key) > 100 {
		return domain.FuelEntry{}, nil, invalid("the request key is too long")
	}
	if key != "" {
		if existing, err := s.fuel.ByClientKey(ctx, actor.UserID, key); err == nil {
			return existing, nil, nil
		} else if !isNotFound(err) {
			return domain.FuelEntry{}, nil, err
		}
	}
	site, err := s.sites.GetByID(ctx, siteID)
	if err != nil {
		return domain.FuelEntry{}, nil, err
	}
	now := s.clock.Now()
	e := domain.FuelEntry{
		SiteID: siteID, Kind: in.Kind, Litres: in.Litres, OccurredAt: in.OccurredAt,
		Note: strings.TrimSpace(in.Note), RecordedBy: actor.UserID, ClientKey: key, CreatedAt: now,
	}
	if problems := e.ValidateNew(site, now); len(problems) > 0 {
		return domain.FuelEntry{}, nil, invalidProblems(problems)
	}

	// A tank reading is checked against what the estimate expected just before
	// it. This is read before the entry is stored, and changes nothing: the
	// reading wins and the estimate restarts from it.
	var variance *domain.FuelVariance
	if e.Kind == domain.FuelReading && site.ConsumptionLPH != nil {
		prior, err := s.fuel.List(ctx, siteID, false, fuelListLimit)
		if err != nil {
			return domain.FuelEntry{}, nil, err
		}
		_, hours, err := s.estimateWithHours(ctx, site, prior)
		if err != nil {
			return domain.FuelEntry{}, nil, err
		}
		variance = domain.CompareReading(site, prior, hours, e, s.loc)
	}

	var created domain.FuelEntry
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if created, err = s.fuel.Create(ctx, e); err != nil {
			return err
		}
		meta := map[string]any{
			"site_id": siteID, "kind": string(created.Kind), "litres": created.Litres,
			"occurred_at": created.OccurredAt.UTC().Format(time.RFC3339),
		}
		if variance != nil {
			meta["estimated_before"] = variance.EstimatedBefore
			meta["difference"] = variance.Difference
			meta["review_warning"] = variance.ReviewWarning
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "fuel.recorded", EntityType: "fuel_entry", EntityID: created.ID, Metadata: meta,
		})
	})
	if err != nil && key != "" && errors.Is(err, apperrors.ErrConflict) {
		if existing, ferr := s.fuel.ByClientKey(ctx, actor.UserID, key); ferr == nil {
			return existing, nil, nil
		}
	}
	return created, variance, err
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
	est, hours, err := s.estimateWithHours(ctx, site, entries)
	if err != nil {
		return FuelView{}, err
	}
	view := FuelView{Site: site, Estimate: est, Entries: entries, Variances: map[string]*domain.FuelVariance{}, LowHours: s.lowHours}
	if site.ConsumptionLPH != nil {
		n := 0
		for _, e := range entries { // newest first
			if e.Kind != domain.FuelReading || e.Voided() || n >= fuelVarianceReadings {
				continue
			}
			n++
			if v := domain.CompareReading(site, entries, hours, e, s.loc); v != nil {
				view.Variances[e.ID] = v
			}
		}
	}
	return view, nil
}

// estimateWithHours reads the approved generator hours for the days since the
// first entry and works out the level, returning the hours too so a reading can
// be compared against the estimate. A day with no approved report stays
// unknown; it is never read as no use.
func (s *FuelService) estimateWithHours(ctx context.Context, site domain.Site, entries []domain.FuelEntry) (domain.FuelEstimate, map[string]*float64, error) {
	now := s.clock.Now()
	live := map[string][]domain.FuelEntry{}
	for _, e := range entries {
		if !e.Voided() {
			live[site.ID] = append(live[site.ID], e)
		}
	}
	if len(live) == 0 || site.ConsumptionLPH == nil {
		return domain.EstimateFuel(site, entries, nil, now, s.loc), nil, nil
	}
	all, err := fuelHours(ctx, s.reports, live, now, s.loc)
	if err != nil {
		return domain.FuelEstimate{}, nil, err
	}
	return domain.EstimateFuel(site, entries, all[site.ID], now, s.loc), all[site.ID], nil
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

// AlertLow emails the people who record fuel at a site when its estimate says
// the tank is low or empty. It sends once for each episode: a new entry (a
// delivery that fixes it, a reading that changes it) starts a new one, so a
// tank that stays low is not repeated and one that is refilled and runs low
// again is. The customer sees the level on screen and is not emailed. It
// returns the number of emails queued.
func (s *FuelService) AlertLow(ctx context.Context) (int, error) {
	now := s.clock.Now()
	sites, err := listAllActiveSites(ctx, s.sites)
	if err != nil {
		return 0, err
	}
	found, err := fuelEstimates(ctx, s.fuel, s.reports, sites, now, s.loc)
	if err != nil {
		return 0, err
	}
	queued := 0
	for _, site := range sites {
		f, ok := found[site.ID]
		if !ok || !f.Estimate.Low(s.lowHours) {
			continue
		}
		recipients, err := s.users.UsersWithPermission(ctx, domain.PermFuelRecord, site.ID)
		if err != nil {
			return queued, err
		}
		kind, subject, body := fuelLowEmail(site, f.Estimate, s.lowHours, s.baseURL+"/sites/"+site.ID)
		n, err := s.notifier.EnqueueFor(ctx, recipients, "fuel.low", fmt.Sprintf("%s:%s:%s", kind, site.ID, f.LastEntryID), subject, body)
		queued += n
		if err != nil {
			return queued, err
		}
	}
	return queued, nil
}
