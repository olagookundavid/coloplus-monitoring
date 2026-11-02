package services

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"
)

// fuelLookbackDays bounds how far back generator hours are read.
const fuelLookbackDays = 400

// fuelHours reads the approved generator hours for the given sites, from the
// day of each one's first live entry (bounded) to today. A day with no approved
// report is absent, which the estimate treats as unknown rather than zero.
func fuelHours(ctx context.Context, reports ports.ReportRepository, entries map[string][]domain.FuelEntry, now time.Time, loc *time.Location) (map[string]map[string]*float64, error) {
	toDay := func(t time.Time) time.Time {
		y, m, d := t.In(loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	today := toDay(now)
	bound := today.AddDate(0, 0, -fuelLookbackDays)
	from := today
	ids := make([]string, 0, len(entries))
	for id, es := range entries {
		ids = append(ids, id)
		for _, e := range es {
			if d := toDay(e.OccurredAt); d.Before(from) {
				from = d
			}
		}
	}
	if from.Before(bound) {
		from = bound
	}
	raw, err := reports.GeneratorHoursByDay(ctx, ids, from, today)
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]*float64, len(raw))
	for site, days := range raw {
		m := make(map[string]*float64, len(days))
		for day, h := range days {
			h := h
			m[day] = &h
		}
		out[site] = m
	}
	return out, nil
}

// siteFuel is one site's estimate and the entry that opened the present
// episode: the most recently recorded one, so any new entry starts afresh.
type siteFuel struct {
	Estimate    domain.FuelEstimate
	LastEntryID string
}

// fuelEstimates works out the tank estimate for many sites with two queries,
// for the sites that have an agreed rate and at least one live entry. Sites
// without are left out: there is nothing to say about them.
func fuelEstimates(
	ctx context.Context, fuel ports.FuelRepository, reports ports.ReportRepository,
	sites []domain.Site, now time.Time, loc *time.Location,
) (map[string]siteFuel, error) {
	ids := make([]string, 0, len(sites))
	bySite := make(map[string]domain.Site, len(sites))
	for _, s := range sites {
		if s.ConsumptionLPH != nil {
			ids = append(ids, s.ID)
			bySite[s.ID] = s
		}
	}
	entries, err := fuel.ListLive(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return map[string]siteFuel{}, nil
	}
	hours, err := fuelHours(ctx, reports, entries, now, loc)
	if err != nil {
		return nil, err
	}
	out := make(map[string]siteFuel, len(entries))
	for id, es := range entries {
		last := es[0]
		for _, e := range es {
			if e.CreatedAt.After(last.CreatedAt) {
				last = e
			}
		}
		out[id] = siteFuel{Estimate: domain.EstimateFuel(bySite[id], es, hours[id], now, loc), LastEntryID: last.ID}
	}
	return out, nil
}
