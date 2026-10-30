package domain

import (
	"fmt"
	"sort"
	"time"
)

type ReportStatus string

const (
	ReportDraft      ReportStatus = "draft"
	ReportSubmitted  ReportStatus = "submitted"
	ReportApproved   ReportStatus = "approved"
	ReportRejected   ReportStatus = "rejected"
	ReportSuperseded ReportStatus = "superseded"
)

func ValidReportStatus(s ReportStatus) bool {
	switch s {
	case ReportDraft, ReportSubmitted, ReportApproved, ReportRejected, ReportSuperseded:
		return true
	}
	return false
}

type IntervalState string

const (
	StateServed IntervalState = "served"
	StateOutage IntervalState = "outage"
)

// Basis says how a value is known. Only observed intervals count towards
// availability; a declared schedule can support a forecast but never
// historical uptime.
type Basis string

const (
	BasisObserved Basis = "observed"
	BasisDeclared Basis = "declared"
)

type PowerInterval struct {
	ID     string
	Start  time.Time
	End    time.Time
	State  IntervalState
	Source PowerSource // required when served
	Basis  Basis
}

// DailyReport is one revision of one site's report for one reporting day.
// An approved report is immutable; a correction is a new, linked revision.
type DailyReport struct {
	ID       string
	SiteID   string
	Date     time.Time // calendar date; only year, month and day matter
	Revision int
	Status   ReportStatus

	AuthorID  string
	Notes     string
	Intervals []PowerInterval

	SubmittedAt   *time.Time
	ReviewedAt    *time.Time
	ReviewerID    string
	ReviewComment string

	SupersedesID     string
	CorrectionReason string

	// RowVersion is the optimistic-concurrency token: every edit must quote
	// the version it was based on.
	RowVersion int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Editable reports whether the reporter may still change the report.
func (r DailyReport) Editable() bool {
	return r.Status == ReportDraft || r.Status == ReportRejected
}

// DayBounds returns the start (inclusive) and end (exclusive) of the
// reporting day in loc.
func DayBounds(date time.Time, loc *time.Location) (time.Time, time.Time) {
	y, m, d := date.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, loc)
	return start, time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// ValidateIntervals checks a report's intervals against the PRD's entry
// rules and returns every problem found, worded for the reporter.
func ValidateIntervals(site Site, date time.Time, intervals []PowerInterval, now time.Time, loc *time.Location) []string {
	var problems []string
	dayStart, dayEnd := DayBounds(date, loc)

	for i, iv := range intervals {
		n := i + 1
		switch {
		case !iv.End.After(iv.Start):
			problems = append(problems, fmt.Sprintf("interval %d: the end must be after the start", n))
			continue
		case iv.Start.Second() != 0 || iv.End.Second() != 0 || iv.Start.Nanosecond() != 0 || iv.End.Nanosecond() != 0:
			problems = append(problems, fmt.Sprintf("interval %d: times must be whole minutes", n))
			continue
		}
		if iv.Start.Before(dayStart) || iv.End.After(dayEnd) {
			problems = append(problems, fmt.Sprintf(
				"interval %d: must stay within %s. Enter the part after midnight in the next day's report", n, date.Format("2 Jan 2006")))
		}
		if iv.End.After(now) {
			problems = append(problems, fmt.Sprintf("interval %d: an observation cannot end in the future", n))
		}
		switch iv.State {
		case StateServed:
			if iv.Source == "" {
				problems = append(problems, fmt.Sprintf("interval %d: say which power source served the site", n))
			} else if !site.HasSource(iv.Source) {
				problems = append(problems, fmt.Sprintf("interval %d: %q is not an installed source at this site", n, iv.Source))
			}
		case StateOutage:
			if iv.Source != "" {
				problems = append(problems, fmt.Sprintf("interval %d: an outage has no serving source", n))
			}
		default:
			problems = append(problems, fmt.Sprintf("interval %d: the state must be served or outage", n))
		}
		if iv.Basis != BasisObserved && iv.Basis != BasisDeclared {
			problems = append(problems, fmt.Sprintf("interval %d: the basis must be observed or declared", n))
		}
	}

	// A served and an outage interval may never overlap: the site cannot be
	// both. Only observed intervals are compared.
	for i, a := range intervals {
		for j := i + 1; j < len(intervals); j++ {
			b := intervals[j]
			if a.Basis != BasisObserved || b.Basis != BasisObserved || a.State == b.State {
				continue
			}
			if a.State != StateServed && a.State != StateOutage || b.State != StateServed && b.State != StateOutage {
				continue
			}
			if a.Start.Before(b.End) && b.Start.Before(a.End) {
				problems = append(problems, fmt.Sprintf("intervals %d and %d contradict each other: the site cannot be served and in outage at the same time", i+1, j+1))
			}
		}
	}
	return problems
}

// Span is a half-open time range.
type Span struct{ Start, End time.Time }

// mergeSpans returns the union of spans as sorted, non-overlapping spans.
// Duplicates and overlaps collapse so no minute is counted twice.
func mergeSpans(spans []Span) []Span {
	if len(spans) == 0 {
		return nil
	}
	sorted := append([]Span(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })

	merged := []Span{sorted[0]}
	for _, s := range sorted[1:] {
		last := &merged[len(merged)-1]
		if !s.Start.After(last.End) {
			if s.End.After(last.End) {
				last.End = s.End
			}
			continue
		}
		merged = append(merged, s)
	}
	return merged
}

func clipMinutes(spans []Span, from, to time.Time) int {
	var total time.Duration
	for _, s := range mergeSpans(spans) {
		start, end := s.Start, s.End
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		if end.After(start) {
			total += end.Sub(start)
		}
	}
	return int(total / time.Minute)
}

// DailySummary holds the PRD's metrics for one site and one day. A nil
// percentage means "not applicable" or "unavailable", never zero.
type DailySummary struct {
	EligibleMinutes int // S
	ServedMinutes   int // U
	OutageMinutes   int // D
	UnknownMinutes  int // X = S - U - D

	// Coverage is 100 x (U+D) / S.
	Coverage *float64
	// ReportedAvailability is 100 x U / (U+D).
	ReportedAvailability *float64
	// ConfirmedServedShare is 100 x U / S, the availability lower bound.
	ConfirmedServedShare *float64
}

// Provisional is true when unknown time remains.
func (s DailySummary) Provisional() bool { return s.UnknownMinutes > 0 }

// Summarise computes the daily metrics from observed intervals, clipped to
// the site's service window on the reporting day.
func Summarise(site Site, date time.Time, intervals []PowerInterval, loc *time.Location) DailySummary {
	dayStart, _ := DayBounds(date, loc)
	from := dayStart.Add(time.Duration(site.ServiceStartMinute) * time.Minute)
	to := dayStart.Add(time.Duration(site.ServiceEndMinute) * time.Minute)
	return summarise(from, to, intervals)
}

func summarise(from, to time.Time, intervals []PowerInterval) DailySummary {
	var served, outage []Span
	for _, iv := range intervals {
		if iv.Basis != BasisObserved {
			continue
		}
		switch iv.State {
		case StateServed:
			served = append(served, Span{iv.Start, iv.End})
		case StateOutage:
			outage = append(outage, Span{iv.Start, iv.End})
		}
	}

	s := int(to.Sub(from) / time.Minute)
	u := clipMinutes(served, from, to)
	d := clipMinutes(outage, from, to)
	sum := DailySummary{EligibleMinutes: s, ServedMinutes: u, OutageMinutes: d, UnknownMinutes: s - u - d}

	if s > 0 {
		sum.Coverage = pct(u+d, s)
		sum.ConfirmedServedShare = pct(u, s)
	}
	if u+d > 0 {
		sum.ReportedAvailability = pct(u, u+d)
	}
	return sum
}

func pct(part, whole int) *float64 {
	v := 100 * float64(part) / float64(whole)
	return &v
}

// PortfolioSummary pools sites by eligible minutes rather than averaging
// their percentages, as the PRD requires.
func PortfolioSummary(sites []DailySummary) DailySummary {
	var total DailySummary
	for _, s := range sites {
		total.EligibleMinutes += s.EligibleMinutes
		total.ServedMinutes += s.ServedMinutes
		total.OutageMinutes += s.OutageMinutes
		total.UnknownMinutes += s.UnknownMinutes
	}
	if total.EligibleMinutes > 0 {
		total.Coverage = pct(total.ServedMinutes+total.OutageMinutes, total.EligibleMinutes)
		total.ConfirmedServedShare = pct(total.ServedMinutes, total.EligibleMinutes)
	}
	if total.ServedMinutes+total.OutageMinutes > 0 {
		total.ReportedAvailability = pct(total.ServedMinutes, total.ServedMinutes+total.OutageMinutes)
	}
	return total
}

// SiteCondition is the plain label shown to customers.
type SiteCondition string

const (
	ConditionAvailable      SiteCondition = "available"
	ConditionOutageReported SiteCondition = "outage_reported"
	ConditionAwaitingUpdate SiteCondition = "awaiting_update"
)

// LastObservedEnd is the end of the latest observed interval, the moment the
// site was last known to be in a state. ok is false when nothing was observed.
func (r DailyReport) LastObservedEnd() (end time.Time, ok bool) {
	for _, iv := range r.Intervals {
		if iv.Basis == BasisObserved && (!ok || iv.End.After(end)) {
			end, ok = iv.End, true
		}
	}
	return end, ok
}

// Condition decides the customer-facing label from the latest approved
// report. An overdue or empty report is "awaiting update" and can never look
// like confirmed availability.
func Condition(site Site, latest *DailyReport, now time.Time) SiteCondition {
	if latest == nil {
		return ConditionAwaitingUpdate
	}
	asOf, ok := latest.LastObservedEnd()
	if !ok || now.Sub(asOf) > time.Duration(site.FreshnessHours)*time.Hour {
		return ConditionAwaitingUpdate
	}
	if lastKnownState(latest) == StateOutage {
		return ConditionOutageReported
	}
	return ConditionAvailable
}

// lastKnownState is the state of the latest observed interval.
func lastKnownState(r *DailyReport) IntervalState {
	var last *PowerInterval
	for i := range r.Intervals {
		iv := &r.Intervals[i]
		if iv.Basis == BasisObserved && (last == nil || iv.End.After(last.End)) {
			last = iv
		}
	}
	if last == nil {
		return ""
	}
	return last.State
}
