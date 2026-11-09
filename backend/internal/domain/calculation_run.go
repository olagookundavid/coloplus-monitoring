package domain

import "time"

// CalculationRun is a period's figures for every site, frozen at one moment.
type CalculationRun struct {
	ID             string
	Kind           string
	PeriodFrom     time.Time
	PeriodTo       time.Time
	Region         string
	Label          string
	Formula        string
	FormulaVersion string
	SiteCount      int
	CreatedBy      string
	CreatedAt      time.Time
	Rows           []RunRow
}

// RunRow is one site's frozen figures. Percentages are nil where they did not
// apply, never zero.
type RunRow struct {
	SiteID, CustomerSiteCode, SiteName string

	EligibleMinutes, ServedMinutes, OutageMinutes, UnknownMinutes, AssumedMinutes int
	DaysInPeriod, DaysReported                                                    int

	ReportedAvailability, ConfirmedServedShare, Coverage *float64

	GeneratorHours   float64
	IncidentsStarted int
	// Fingerprint identifies the approved reports the row was worked out from.
	Fingerprint string
}

// SameFigures reports whether two rows state the same figures.
func (r RunRow) SameFigures(o RunRow) bool {
	return r.EligibleMinutes == o.EligibleMinutes && r.ServedMinutes == o.ServedMinutes &&
		r.OutageMinutes == o.OutageMinutes && r.UnknownMinutes == o.UnknownMinutes &&
		r.AssumedMinutes == o.AssumedMinutes && r.DaysReported == o.DaysReported &&
		r.GeneratorHours == o.GeneratorHours && r.IncidentsStarted == o.IncidentsStarted
}
