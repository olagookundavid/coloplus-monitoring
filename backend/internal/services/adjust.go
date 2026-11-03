package services

import (
	"context"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"
)

// Adjustments are the facts, outside a day's own report, that figures need:
// approved planned maintenance (taken out of the eligible time) and an outage
// carried over from the day before (DECISIONS 060). One value is shared by every
// service that shows a figure, so a figure never differs from screen to screen.
type Adjustments struct {
	windows ports.MaintenanceRepository
	reports ports.ReportRepository
	loc     *time.Location
	// carry turns the midnight carry-over on or off.
	carry bool
}

func NewAdjustments(windows ports.MaintenanceRepository, reports ports.ReportRepository, loc *time.Location, carryOver bool) *Adjustments {
	return &Adjustments{windows: windows, reports: reports, loc: loc, carry: carryOver}
}

// dayAdjuster holds what was loaded for one request.
type dayAdjuster struct {
	carry   bool
	windows map[string][]domain.Span
	byDay   map[string]domain.DailyReport
}

// For loads the adjustments for the sites over from..to. byDay is the caller's
// approved reports for that range; the day before from is fetched here. A nil
// receiver returns an adjuster that changes nothing.
func (a *Adjustments) For(ctx context.Context, siteIDs []string, from, to time.Time, byDay map[string]domain.DailyReport) (*dayAdjuster, error) {
	out := &dayAdjuster{windows: map[string][]domain.Span{}, byDay: byDay}
	if a == nil || len(siteIDs) == 0 {
		return out, nil
	}
	out.carry = a.carry
	if a.windows != nil {
		start, _ := domain.DayBounds(from, a.loc)
		_, end := domain.DayBounds(to, a.loc)
		ws, err := a.windows.Approved(ctx, siteIDs, start, end)
		if err != nil {
			return nil, err
		}
		for _, w := range ws {
			out.windows[w.SiteID] = append(out.windows[w.SiteID], w.Span())
		}
	}
	if a.carry {
		day := from.AddDate(0, 0, -1)
		prev, err := a.reports.ApprovedInRange(ctx, siteIDs, day, day)
		if err != nil {
			return nil, err
		}
		merged := make(map[string]domain.DailyReport, len(byDay)+len(prev))
		for k, v := range byDay {
			merged[k] = v
		}
		for _, r := range prev {
			merged[ports.DayKey(r.SiteID, r.Date)] = r
		}
		out.byDay = merged
	}
	return out, nil
}

// summarise is the site's figures for one day, in the portal's time zone.
// hasReport says the intervals come from an approved report of that day.
func (d *dayAdjuster) summarise(loc *time.Location, site domain.Site, date time.Time, intervals []domain.PowerInterval, hasReport bool) domain.DailySummary {
	if d == nil {
		return domain.Summarise(site, date, intervals, loc)
	}
	adj := domain.DayAdjust{Excluded: d.windows[site.ID]}
	if d.carry {
		if prev, ok := d.byDay[ports.DayKey(site.ID, date.AddDate(0, 0, -1))]; ok {
			adj.Carried = domain.CarryIn(site, date, prev.Intervals, intervals, hasReport, loc)
		}
	}
	return domain.SummariseAdjusted(site, date, intervals, loc, adj)
}
