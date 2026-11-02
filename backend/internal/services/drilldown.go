package services

import (
	"context"
	"sort"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
)

// OutageItem is one stretch of reported outage, with the report it came from.
type OutageItem struct {
	Site       domain.Site
	Start, End time.Time
	Minutes    int
	ReportID   string
	ReportDate time.Time
	Revision   int
	// Reason is why the site was unavailable. Coloplus staff only.
	Reason string
}

type OutageList struct {
	Period       Period
	GeneratedAt  time.Time
	TotalMinutes int
	Total        int
	Items        []OutageItem
	Limit        int
	Offset       int
}

const outageListMax = 500

// Outages lists the reported outages behind the dashboard's figures for a
// period: every stretch from the approved reports, merged so none is counted
// twice, newest first. It is what a click on an outage card opens.
func (s *DashboardService) Outages(ctx context.Context, actor domain.Actor, p Period, siteID string, limit, offset int) (OutageList, error) {
	if !actor.Can(domain.PermSitesRead) {
		return OutageList{}, apperrors.ErrForbidden
	}
	if siteID != "" && !actor.CanAccessSite(siteID) {
		return OutageList{}, apperrors.ErrNotFound
	}
	sc, err := s.load(ctx, actor, p)
	if err != nil {
		return OutageList{}, err
	}
	bySite := make(map[string]domain.Site, len(sc.sites))
	for _, site := range sc.sites {
		bySite[site.ID] = site
	}
	staff := actor.Can(domain.PermReportsRead)

	var items []OutageItem
	total := 0
	for _, r := range sc.byDay {
		site, ok := bySite[r.SiteID]
		if !ok || (siteID != "" && r.SiteID != siteID) {
			continue
		}
		for _, sp := range domain.OutageSpans(site, r.Date, r.Intervals, s.loc) {
			it := OutageItem{
				Site: site, Start: sp.Start, End: sp.End, Minutes: int(sp.End.Sub(sp.Start) / time.Minute),
				ReportID: r.ID, ReportDate: r.Date, Revision: r.Revision,
			}
			if staff {
				it.Reason = r.OutageReason
			}
			total += it.Minutes
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Start.After(items[j].Start) })

	if limit <= 0 || limit > outageListMax {
		limit = 100
	}
	out := OutageList{Period: p, GeneratedAt: sc.now, TotalMinutes: total, Total: len(items), Limit: limit, Offset: offset}
	if offset < 0 {
		offset = 0
	}
	if offset < len(items) {
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		out.Items = items[offset:end]
	}
	return out, nil
}

// IncidentOutage is a reported outage that overlaps an incident.
type IncidentOutage struct {
	Start, End time.Time
	// Minutes is the part of it inside the incident's window.
	Minutes    int
	ReportID   string
	ReportDate time.Time
	Revision   int
}

type IncidentOutages struct {
	IncidentID string
	// From and To are the incident's window: it started, and it was restored
	// (or now, while it is still open).
	From, To time.Time
	Open     bool
	// TotalMinutes is the outage time inside the window, each minute counted
	// once even where reports overlap. It is the downtime the incident links to,
	// without adding to the daily figures.
	TotalMinutes int
	Items        []IncidentOutage
}

// IncidentOutages links an incident to the outage the daily reports show
// during it, without counting any downtime twice. A caller sees an incident
// only as they could anywhere else: staff in their sites, "Airtel" users only
// once it is published.
func (s *DashboardService) IncidentOutages(ctx context.Context, actor domain.Actor, incidentID string) (IncidentOutages, error) {
	staff := actor.Can(domain.PermIncidentsRead)
	if !staff && !actor.Can(domain.PermPublishedRead) {
		return IncidentOutages{}, apperrors.ErrForbidden
	}
	inc, err := s.incidents.GetByID(ctx, incidentID, false)
	if err != nil {
		return IncidentOutages{}, err
	}
	if !actor.CanAccessSite(inc.SiteID) || (!staff && inc.PublishedAt == nil) {
		return IncidentOutages{}, apperrors.ErrNotFound
	}
	site, err := s.sites.GetByID(ctx, inc.SiteID)
	if err != nil {
		return IncidentOutages{}, err
	}

	now := s.clock.Now()
	from, to, open := inc.StartedAt, now, true
	if inc.RestoredAt != nil {
		to, open = *inc.RestoredAt, false
	}
	if to.Before(from) {
		to = from
	}
	toDay := func(t time.Time) time.Time {
		y, m, d := t.In(s.loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	reps, err := s.reports.ApprovedInRange(ctx, []string{inc.SiteID}, toDay(from), toDay(to))
	if err != nil {
		return IncidentOutages{}, err
	}

	out := IncidentOutages{IncidentID: inc.ID, From: from, To: to, Open: open}
	var all []domain.Span
	for _, r := range reps {
		for _, sp := range domain.OutageSpans(site, r.Date, r.Intervals, s.loc) {
			start, end := sp.Start, sp.End
			if start.Before(from) {
				start = from
			}
			if end.After(to) {
				end = to
			}
			if !end.After(start) {
				continue
			}
			all = append(all, domain.Span{Start: start, End: end})
			out.Items = append(out.Items, IncidentOutage{
				Start: sp.Start, End: sp.End, Minutes: int(end.Sub(start) / time.Minute),
				ReportID: r.ID, ReportDate: r.Date, Revision: r.Revision,
			})
		}
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].Start.Before(out.Items[j].Start) })
	out.TotalMinutes = domain.SpanMinutes(all, from, to)
	return out, nil
}
