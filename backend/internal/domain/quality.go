package domain

// DataQuality says how much of a figure rests on reports rather than on the
// default. It is built only from facts already on record, with no judgement
// calls beyond the three thresholds below, which are proposed and open to
// being agreed with the customer.
type DataQuality struct {
	// Days in the period, and how many have an approved report.
	Days         int
	ReportedDays int
	// Completeness is ReportedDays as a share of Days, in percent.
	Completeness float64
	// LateReports counts approved reports that were submitted late (and so carry
	// a reason); Corrections counts approved corrections of earlier reports.
	LateReports int
	Corrections int
	// UnconfirmedShare is the share of eligible time no report confirmed, in
	// percent: counted as available (assumed) or left unknown, depending on the site.
	UnconfirmedShare float64
	Level            QualityLevel
}

type QualityLevel string

const (
	QualityGood QualityLevel = "good"
	QualityFair QualityLevel = "fair"
	QualityPoor QualityLevel = "poor"
)

const (
	// Complete enough to be good: at least 95% of days reported, and not more
	// than a fifth of the reports late. Fair: at least 80% of days. Below that, poor.
	qualityGoodCompleteness = 95.0
	qualityFairCompleteness = 80.0
	qualityMaxLateShare     = 0.20
)

// AssessQuality builds the indicator for a site over a period from its days,
// its approved reports in that period, and its summary for the period.
func AssessQuality(days int, reports []DailyReport, summary DailySummary) DataQuality {
	q := DataQuality{Days: days, ReportedDays: len(reports)}
	if days > 0 {
		q.Completeness = 100 * float64(q.ReportedDays) / float64(days)
	}
	for _, r := range reports {
		if r.LateReason != "" {
			q.LateReports++
		}
		if r.SupersedesID != "" {
			q.Corrections++
		}
	}
	if summary.EligibleMinutes > 0 {
		q.UnconfirmedShare = 100 * float64(summary.UnknownMinutes) / float64(summary.EligibleMinutes)
	}
	lateShare := 0.0
	if q.ReportedDays > 0 {
		lateShare = float64(q.LateReports) / float64(q.ReportedDays)
	}
	switch {
	case q.Completeness >= qualityGoodCompleteness && lateShare <= qualityMaxLateShare:
		q.Level = QualityGood
	case q.Completeness >= qualityFairCompleteness:
		q.Level = QualityFair
	default:
		q.Level = QualityPoor
	}
	return q
}
