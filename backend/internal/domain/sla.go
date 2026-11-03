package domain

// SLAStatus says where a figure stands against the agreed availability target.
type SLAStatus string

const (
	SLAMet    SLAStatus = "met"
	SLAAtRisk SLAStatus = "at_risk"
	SLAMissed SLAStatus = "missed"
)

// SLA is the availability target. A zero target means none has been agreed and
// nothing is judged. The number is a proposal until the customer agrees it.
type SLA struct {
	// TargetPercent is the availability to meet over the period.
	TargetPercent float64
	// AtRiskPoints is how many percentage points above the target still counts
	// as at risk rather than met.
	AtRiskPoints float64
}

// SLAResult judges one figure. It never hides how much of the figure is only
// assumed: ConfirmedMeetsTarget says whether the reported time alone, with
// nothing assumed, would reach the target.
type SLAResult struct {
	TargetPercent        float64
	Status               SLAStatus
	ConfirmedMeetsTarget bool
}

// Assess judges a summary against the target; nil when no target is set or the
// figure does not apply (no eligible reported time), never "met" by default.
func (s SLA) Assess(sum DailySummary) *SLAResult {
	if s.TargetPercent <= 0 || sum.ReportedAvailability == nil {
		return nil
	}
	a := *sum.ReportedAvailability
	res := SLAResult{TargetPercent: s.TargetPercent}
	switch {
	case a < s.TargetPercent:
		res.Status = SLAMissed
	case a < s.TargetPercent+s.AtRiskPoints:
		res.Status = SLAAtRisk
	default:
		res.Status = SLAMet
	}
	res.ConfirmedMeetsTarget = sum.ConfirmedServedShare != nil && *sum.ConfirmedServedShare >= s.TargetPercent
	return &res
}
