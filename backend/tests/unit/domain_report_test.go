package unit

import (
	"math"
	"strings"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

var lagos = time.FixedZone("Africa/Lagos", 3600)

// day is 1 December 2026, the reporting day used throughout.
var day = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

func at(hour, minute int) time.Time {
	return time.Date(2026, 12, 1, hour, minute, 0, 0, lagos)
}

func served(from, to time.Time, src domain.PowerSource) domain.PowerInterval {
	return domain.PowerInterval{Start: from, End: to, State: domain.StateServed, Source: src, Basis: domain.BasisObserved}
}

func outage(from, to time.Time) domain.PowerInterval {
	return domain.PowerInterval{Start: from, End: to, State: domain.StateOutage, Basis: domain.BasisObserved}
}

func testSite() domain.Site {
	return domain.Site{
		ID: "site-1", AirtelCode: "LOS-001", ColoplusCode: "CP-001", Name: "Ikeja", State: "Lagos",
		PowerSources:       []domain.PowerSource{domain.SourceGrid, domain.SourceGenerator},
		ServiceStartMinute: 0, ServiceEndMinute: 1440, FreshnessHours: 30, Status: domain.SiteActive,
	}
}

func near(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %.2f", name, want)
	}
	if math.Abs(*got-want) > 0.005 {
		t.Fatalf("%s = %.4f, want %.2f", name, *got, want)
	}
}

// The worked example in PRD section 8: 24 eligible hours, 23 known served,
// 0.5 known outage, 0.5 unknown.
func TestSummarise_PRDWorkedExample(t *testing.T) {
	ivs := []domain.PowerInterval{
		served(at(0, 0), at(11, 30), domain.SourceGrid),
		outage(at(11, 30), at(12, 0)),
		served(at(12, 0), at(23, 30), domain.SourceGenerator),
		// 23:30 to 24:00 is unknown.
	}
	sum := domain.Summarise(testSite(), day, ivs, lagos)

	if sum.EligibleMinutes != 1440 || sum.ServedMinutes != 1380 || sum.OutageMinutes != 30 || sum.UnknownMinutes != 30 {
		t.Fatalf("minutes = S%d U%d D%d X%d", sum.EligibleMinutes, sum.ServedMinutes, sum.OutageMinutes, sum.UnknownMinutes)
	}
	near(t, "coverage", sum.Coverage, 97.92)
	near(t, "reported availability", sum.ReportedAvailability, 97.87)
	near(t, "confirmed served share", sum.ConfirmedServedShare, 95.83)
	if !sum.Provisional() {
		t.Fatal("a day with unknown time must be provisional")
	}
}

func TestSummarise_OverlapsAndDuplicatesCountOnce(t *testing.T) {
	ivs := []domain.PowerInterval{
		served(at(0, 0), at(6, 0), domain.SourceGrid),
		served(at(4, 0), at(8, 0), domain.SourceGenerator), // overlaps the first
		served(at(0, 0), at(6, 0), domain.SourceGrid),      // exact duplicate
	}
	sum := domain.Summarise(testSite(), day, ivs, lagos)
	if sum.ServedMinutes != 8*60 {
		t.Fatalf("served = %d minutes, want 480: overlaps must merge", sum.ServedMinutes)
	}
}

func TestSummarise_DeclaredIntervalsNeverCount(t *testing.T) {
	declared := served(at(0, 0), at(24-1, 59), domain.SourceGrid)
	declared.Basis = domain.BasisDeclared
	sum := domain.Summarise(testSite(), day, []domain.PowerInterval{declared}, lagos)
	if sum.ServedMinutes != 0 || sum.UnknownMinutes != 1440 {
		t.Fatalf("a declared schedule must not establish uptime: U%d X%d", sum.ServedMinutes, sum.UnknownMinutes)
	}
}

func TestSummarise_NoData_IsUnknownNeverZeroOrAvailable(t *testing.T) {
	sum := domain.Summarise(testSite(), day, nil, lagos)
	if sum.UnknownMinutes != 1440 {
		t.Fatalf("unknown = %d, want 1440", sum.UnknownMinutes)
	}
	if sum.ReportedAvailability != nil {
		t.Fatal("availability with no known minutes must be unavailable (nil), not 0 or 100")
	}
	near(t, "coverage", sum.Coverage, 0)
}

func TestSummarise_ClipsToServiceWindow(t *testing.T) {
	site := testSite()
	site.ServiceStartMinute, site.ServiceEndMinute = 8*60, 18*60 // 08:00 to 18:00
	ivs := []domain.PowerInterval{served(at(0, 0), at(20, 0), domain.SourceGrid)}

	sum := domain.Summarise(site, day, ivs, lagos)
	if sum.EligibleMinutes != 600 || sum.ServedMinutes != 600 || sum.UnknownMinutes != 0 {
		t.Fatalf("minutes = S%d U%d X%d, want S600 U600 X0", sum.EligibleMinutes, sum.ServedMinutes, sum.UnknownMinutes)
	}
}

func TestPortfolioSummary_WeightsByEligibleMinutes(t *testing.T) {
	// Site A: 100% over 1000 minutes. Site B: 0% over 10 minutes.
	// An unweighted mean would say 50%; pooling must say ~99%.
	a := domain.DailySummary{EligibleMinutes: 1000, ServedMinutes: 1000}
	b := domain.DailySummary{EligibleMinutes: 10, OutageMinutes: 10}
	p := domain.PortfolioSummary([]domain.DailySummary{a, b})
	near(t, "pooled availability", p.ReportedAvailability, 100*1000.0/1010)
}

func TestValidateIntervals(t *testing.T) {
	site := testSite()
	now := at(23, 0)

	cases := []struct {
		name string
		ivs  []domain.PowerInterval
		want string // substring of a problem; empty means valid
	}{
		{"valid", []domain.PowerInterval{served(at(1, 0), at(5, 0), domain.SourceGrid)}, ""},
		{"end before start", []domain.PowerInterval{served(at(5, 0), at(1, 0), domain.SourceGrid)}, "end must be after"},
		{"zero length", []domain.PowerInterval{served(at(5, 0), at(5, 0), domain.SourceGrid)}, "end must be after"},
		{"seconds", []domain.PowerInterval{served(at(1, 0), at(5, 0).Add(7*time.Second), domain.SourceGrid)}, "whole minutes"},
		{"future", []domain.PowerInterval{served(at(22, 0), at(23, 30), domain.SourceGrid)}, "future"},
		{"crosses midnight", []domain.PowerInterval{outage(at(22, 0).Add(-24*time.Hour), at(1, 0))}, "within"},
		{"served without source", []domain.PowerInterval{served(at(1, 0), at(2, 0), "")}, "which power source"},
		{"source not installed", []domain.PowerInterval{served(at(1, 0), at(2, 0), domain.SourceSolar)}, "not an installed source"},
		{"outage with source", []domain.PowerInterval{{Start: at(1, 0), End: at(2, 0), State: domain.StateOutage, Source: domain.SourceGrid, Basis: domain.BasisObserved}}, "no serving source"},
		{"served and outage overlap", []domain.PowerInterval{
			served(at(1, 0), at(5, 0), domain.SourceGrid), outage(at(4, 0), at(6, 0)),
		}, "contradict"},
		{"served then outage touching is fine", []domain.PowerInterval{
			served(at(1, 0), at(5, 0), domain.SourceGrid), outage(at(5, 0), at(6, 0)),
		}, ""},
		{"unknown state", []domain.PowerInterval{{Start: at(1, 0), End: at(2, 0), State: "on", Basis: domain.BasisObserved}}, "served or outage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := domain.ValidateIntervals(site, day, tc.ivs, now, lagos)
			joined := strings.Join(problems, "; ")
			if tc.want == "" && len(problems) > 0 {
				t.Fatalf("unexpected problems: %s", joined)
			}
			if tc.want != "" && !strings.Contains(joined, tc.want) {
				t.Fatalf("problems = %q, want one containing %q", joined, tc.want)
			}
		})
	}
}

func TestCondition_OverdueOrEmptyIsNeverAvailable(t *testing.T) {
	site := testSite() // freshness 30h
	now := time.Date(2026, 12, 3, 12, 0, 0, 0, lagos)

	fresh := &domain.DailyReport{Intervals: []domain.PowerInterval{served(at(0, 0), at(23, 0), domain.SourceGrid)}}
	fresh.Intervals[0].End = now.Add(-2 * time.Hour)
	fresh.Intervals[0].Start = now.Add(-10 * time.Hour)

	stale := &domain.DailyReport{Intervals: []domain.PowerInterval{served(at(0, 0), at(23, 0), domain.SourceGrid)}} // two days old
	ongoing := &domain.DailyReport{Intervals: []domain.PowerInterval{
		served(now.Add(-10*time.Hour), now.Add(-5*time.Hour), domain.SourceGrid),
		outage(now.Add(-5*time.Hour), now.Add(-1*time.Hour)),
	}}
	declaredOnly := &domain.DailyReport{Intervals: []domain.PowerInterval{
		{Start: now.Add(-3 * time.Hour), End: now.Add(-1 * time.Hour), State: domain.StateServed, Source: domain.SourceGrid, Basis: domain.BasisDeclared},
	}}

	cases := []struct {
		name   string
		report *domain.DailyReport
		want   domain.SiteCondition
	}{
		{"no report", nil, domain.ConditionAwaitingUpdate},
		{"report is overdue", stale, domain.ConditionAwaitingUpdate},
		{"only a declared schedule", declaredOnly, domain.ConditionAwaitingUpdate},
		{"fresh and served", fresh, domain.ConditionAvailable},
		{"ends in an outage", ongoing, domain.ConditionOutageReported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.Condition(site, tc.report, now); got != tc.want {
				t.Fatalf("condition = %s, want %s", got, tc.want)
			}
		})
	}
}
