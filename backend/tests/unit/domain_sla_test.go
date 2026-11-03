package unit

import (
	"testing"

	"coloplus-backend/internal/domain"
)

// The target judges the reported figure, but a site that only reaches it by
// time nobody reported must be told apart from one that confirmed it; and with
// no target, or no figure, nothing is judged (never "met" by default).
func TestSLA_AssumedTimeIsNeverPassedOffAsConfirmed(t *testing.T) {
	site := testSite()
	site.UnreportedTime = domain.UnreportedAvailable
	sla := domain.SLA{TargetPercent: 99.5, AtRiskPoints: 0.2}

	// Nothing reported: 100% by assumption, 0% confirmed.
	sum := domain.Summarise(site, day, nil, lagos)
	got := sla.Assess(sum)
	if got == nil || got.Status != domain.SLAMet || got.ConfirmedMeetsTarget {
		t.Fatalf("an unreported day meets the target only by assumption: %+v", got)
	}

	// A 30-minute outage in a 24-hour day is 97.9%: missed.
	out := []domain.PowerInterval{outage(at(1, 0), at(1, 30))}
	if got := sla.Assess(domain.Summarise(site, day, out, lagos)); got == nil || got.Status != domain.SLAMissed {
		t.Fatalf("30 minutes down in a day should miss 99.5%%: %+v", got)
	}

	if (domain.SLA{}).Assess(sum) != nil {
		t.Fatal("with no target nothing is judged")
	}
	strict := testSite() // unreported time unknown
	if sla.Assess(domain.Summarise(strict, day, nil, lagos)) != nil {
		t.Fatal("a figure that does not apply is not judged")
	}
}
