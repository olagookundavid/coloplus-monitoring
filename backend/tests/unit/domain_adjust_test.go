package unit

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func availableSite() domain.Site {
	s := testSite()
	s.UnreportedTime = domain.UnreportedAvailable
	return s
}

// Approved planned maintenance leaves the eligible time, and nothing reported
// inside it counts for or against the site.
func TestSummarise_PlannedMaintenanceLeavesTheEligibleTime(t *testing.T) {
	ivs := []domain.PowerInterval{
		served(at(0, 0), at(10, 0), domain.SourceGrid),
		outage(at(10, 0), at(11, 0)), // inside the window: not counted
		outage(at(13, 0), at(13, 30)),
	}
	adj := domain.DayAdjust{Excluded: []domain.Span{{Start: at(10, 0), End: at(12, 0)}}}
	sum := domain.SummariseAdjusted(availableSite(), day, ivs, lagos, adj)

	if sum.EligibleMinutes != 1320 || sum.ExcludedMinutes != 120 {
		t.Fatalf("eligible %d excluded %d, want 1320 and 120", sum.EligibleMinutes, sum.ExcludedMinutes)
	}
	if sum.OutageMinutes != 30 || sum.ServedMinutes != 600 {
		t.Fatalf("outage %d served %d: the hour inside the window must not count", sum.OutageMinutes, sum.ServedMinutes)
	}
	near(t, "availability", sum.ReportedAvailability, 100*float64(1320-30)/1320)
	if sum.AssumedMinutes != 1320-600-30 {
		t.Fatalf("assumed = %d", sum.AssumedMinutes)
	}
}

// An outage still going at the end of the day is assumed to continue until a
// report says otherwise. It counts against availability, is shown as assumed,
// never counts as coverage, and carries one day at most.
func TestCarryIn_AnOutageAtMidnightIsAssumedToContinueUntilReported(t *testing.T) {
	site := availableSite()
	nextDay := day.AddDate(0, 0, 1)
	atNext := func(h, m int) time.Time { return time.Date(2026, 12, 2, h, m, 0, 0, lagos) }
	prev := []domain.PowerInterval{served(at(0, 0), at(22, 0), domain.SourceGrid), outage(at(22, 0), at(24, 0))}

	// The next day's report starts at 02:00: midnight to 02:00 is carried.
	cur := []domain.PowerInterval{served(atNext(2, 0), atNext(24, 0), domain.SourceGrid)}
	carry := domain.CarryIn(site, nextDay, prev, cur, true, lagos)
	if carry == nil || !carry.Start.Equal(atNext(0, 0)) || !carry.End.Equal(atNext(2, 0)) {
		t.Fatalf("carry = %+v, want 00:00 to 02:00", carry)
	}
	sum := domain.SummariseAdjusted(site, nextDay, cur, lagos, domain.DayAdjust{Carried: carry})
	if sum.CarriedMinutes != 120 || sum.OutageMinutes != 0 {
		t.Fatalf("carried %d outage %d: carried time is assumed, not reported", sum.CarriedMinutes, sum.OutageMinutes)
	}
	near(t, "availability", sum.ReportedAvailability, 100*float64(1440-120)/1440)
	near(t, "coverage", sum.Coverage, 100*float64(1320)/1440) // carried time is not confirmed

	// No report that day: the whole window is carried.
	if c := domain.CarryIn(site, nextDay, prev, nil, false, lagos); c == nil || !c.End.Equal(atNext(24, 0)) {
		t.Fatalf("with no report the whole day carries: %+v", c)
	}
	// A report that starts at midnight says what happened; nothing carries.
	if c := domain.CarryIn(site, nextDay, prev, []domain.PowerInterval{served(atNext(0, 0), atNext(24, 0), domain.SourceGrid)}, true, lagos); c != nil {
		t.Fatalf("a report covering midnight settles it: %+v", c)
	}
	// The day before did not end in an outage: nothing carries.
	ok := []domain.PowerInterval{outage(at(10, 0), at(11, 0)), served(at(11, 0), at(24, 0), domain.SourceGrid)}
	if c := domain.CarryIn(site, nextDay, ok, nil, false, lagos); c != nil {
		t.Fatalf("restored before the day ended: %+v", c)
	}
}
