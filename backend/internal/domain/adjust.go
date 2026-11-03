package domain

import "time"

// DayAdjust is what a day's figures need beyond its own report.
type DayAdjust struct {
	// Excluded are approved planned-maintenance windows. Their time leaves the
	// eligible time, and anything reported inside them is not counted.
	Excluded []Span
	// Carried is an outage that was still going at the end of the day before
	// and has not been contradicted by a report: it is assumed to continue, and
	// is shown as assumed, never as reported.
	Carried *Span
}

// clippedExclusions returns the exclusions cut to from..to, dropping empty ones.
func (a DayAdjust) clippedExclusions(from, to time.Time) []Span {
	var out []Span
	for _, e := range mergeSpans(a.Excluded) {
		if e.Start.Before(from) {
			e.Start = from
		}
		if e.End.After(to) {
			e.End = to
		}
		if e.End.After(e.Start) {
			out = append(out, e)
		}
	}
	return out
}

// CarryIn works out the outage assumed to continue into date from the day
// before. It applies when the previous day's last observation was an outage
// that ran to the end of that day's service window. The carried time then runs
// from the start of date's window to the first thing date's own report
// observed, or to the end of the window when there is no report. It stops
// there: it carries one day at most and never from a day that was itself
// only carried into. Returns nil when nothing carries.
func CarryIn(site Site, date time.Time, prev, cur []PowerInterval, hasCur bool, loc *time.Location) *Span {
	prevDate := date.AddDate(0, 0, -1)
	prevStart, _ := DayBounds(prevDate, loc)
	prevEnd := prevStart.Add(time.Duration(site.ServiceEndMinute) * time.Minute)

	var lastEnd time.Time
	for _, iv := range prev {
		if iv.Basis == BasisObserved && iv.End.After(lastEnd) {
			lastEnd = iv.End
		}
	}
	if lastEnd.IsZero() || lastEnd.Before(prevEnd) {
		return nil
	}
	reachesEnd := false
	for _, iv := range prev {
		if iv.Basis == BasisObserved && iv.State == StateOutage && !iv.End.Before(prevEnd) {
			reachesEnd = true
		}
	}
	if !reachesEnd {
		return nil
	}
	// A served interval that also ends at the close of the day contradicts it.
	for _, iv := range prev {
		if iv.Basis == BasisObserved && iv.State == StateServed && !iv.End.Before(prevEnd) {
			return nil
		}
	}

	dayStart, _ := DayBounds(date, loc)
	from := dayStart.Add(time.Duration(site.ServiceStartMinute) * time.Minute)
	to := dayStart.Add(time.Duration(site.ServiceEndMinute) * time.Minute)
	end := to
	if hasCur {
		for _, iv := range cur {
			if iv.Basis == BasisObserved && iv.Start.Before(end) {
				end = iv.Start
			}
		}
	}
	if !end.After(from) {
		return nil
	}
	return &Span{from, end}
}
