package unit

import (
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

func fuelSite(lph float64) domain.Site { return domain.Site{ConsumptionLPH: &lph} }

func hours(m map[string]float64) map[string]*float64 {
	out := map[string]*float64{}
	for k, v := range m {
		v := v
		out[k] = &v
	}
	return out
}

// 9 AM Lagos on a day of Nov 2026.
func fuelAt(day, hour int) time.Time { return time.Date(2026, 11, day, hour, 0, 0, 0, lagos) }

// The tank figure is an estimate built from what people entered and the agreed
// rate. It must follow the arithmetic exactly, and never turn what is not
// known into zero.
func TestEstimateFuel_FollowsTheAgreedRateAndNeverGuessesMissingDays(t *testing.T) {
	site := fuelSite(1) // 1 litre per running hour
	asOf := fuelAt(4, 12)

	// 1,000 litres delivered; the generator then ran 20 hours: 980 left.
	got := domain.EstimateFuel(site,
		[]domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 1000, OccurredAt: fuelAt(1, 9)}},
		hours(map[string]float64{"2026-11-01": 0, "2026-11-02": 10, "2026-11-03": 10}), asOf, lagos)
	if !got.Available || *got.Litres != 980 || got.Provisional || got.HoursRun != 20 || got.ConsumedLitres != 20 {
		t.Fatalf("1,000 L less 20 hours at 1 L/h should be 980, got %+v (%v)", got, got.Litres)
	}
	if !got.AssumedEmptyStart {
		t.Fatal("a first delivery with no reading assumes an empty tank, and must say so")
	}
	if *got.HoursRemaining != 980 {
		t.Fatalf("980 L at 1 L/h is 980 hours, got %v", *got.HoursRemaining)
	}

	// 100 litres, same 20 hours: 80.
	got = domain.EstimateFuel(site,
		[]domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 100, OccurredAt: fuelAt(1, 9)}},
		hours(map[string]float64{"2026-11-01": 0, "2026-11-02": 10, "2026-11-03": 10}), asOf, lagos)
	if *got.Litres != 80 {
		t.Fatalf("100 L less 20 hours should be 80, got %v", *got.Litres)
	}

	// A day with no approved hours is unknown, not zero: the level is then the
	// most the tank could hold, and the estimate says it is provisional.
	got = domain.EstimateFuel(site,
		[]domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 1000, OccurredAt: fuelAt(1, 9)}},
		hours(map[string]float64{"2026-11-01": 0, "2026-11-03": 10}), asOf, lagos)
	if *got.Litres != 990 || got.UnreportedDays != 1 || !got.Provisional {
		t.Fatalf("a missing day must be flagged, not counted as no use: %+v", got)
	}

	// A tank reading replaces the estimate and starts the count again.
	got = domain.EstimateFuel(site, []domain.FuelEntry{
		{Kind: domain.FuelDelivery, Litres: 1000, OccurredAt: fuelAt(1, 9)},
		{Kind: domain.FuelReading, Litres: 700, OccurredAt: fuelAt(2, 18)},
	}, hours(map[string]float64{"2026-11-01": 0, "2026-11-02": 10, "2026-11-03": 5}), asOf, lagos)
	if *got.Litres != 695 || got.AssumedEmptyStart || got.AnchorKind != domain.FuelReading {
		t.Fatalf("700 L read, then 5 h used = 695, got %+v (%v)", got, got.Litres)
	}

	// Fuel can run out, and that is shown as empty, not negative.
	got = domain.EstimateFuel(site,
		[]domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 10, OccurredAt: fuelAt(1, 9)}},
		hours(map[string]float64{"2026-11-01": 0, "2026-11-02": 24, "2026-11-03": 24}), asOf, lagos)
	if *got.Litres != 0 || !got.EstimatedEmpty {
		t.Fatalf("a tank cannot go below empty: %+v", got)
	}

	// A voided entry is ignored; no agreed rate, or nothing recorded, is
	// "unavailable" with the reason, never a number.
	voided := time.Now()
	got = domain.EstimateFuel(site,
		[]domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 5000, OccurredAt: fuelAt(1, 9), VoidedAt: &voided}}, nil, asOf, lagos)
	if got.Available || got.Litres != nil {
		t.Fatalf("only a voided entry means nothing recorded: %+v", got)
	}
	got = domain.EstimateFuel(domain.Site{}, []domain.FuelEntry{{Kind: domain.FuelDelivery, Litres: 100, OccurredAt: fuelAt(1, 9)}}, nil, asOf, lagos)
	if got.Available || got.Litres != nil || got.Reason == "" {
		t.Fatalf("no agreed rate must be unavailable, with a reason: %+v", got)
	}
}

// A reading is checked against what the estimate expected, and a large gap is
// flagged for review. It is a check, never a correction.
func TestCompareReading_FlagsOnlyGapsTooBigToBeRounding(t *testing.T) {
	site := fuelSite(1)
	prior := []domain.FuelEntry{{ID: "d", Kind: domain.FuelDelivery, Litres: 200, OccurredAt: fuelAt(1, 9)}}
	hrs := hours(map[string]float64{"2026-11-01": 0, "2026-11-02": 10, "2026-11-03": 10})
	reading := func(l float64) domain.FuelEntry {
		return domain.FuelEntry{ID: "r", Kind: domain.FuelReading, Litres: l, OccurredAt: fuelAt(4, 9)}
	}

	// 200 delivered, 20 hours used: 180 expected.
	small := domain.CompareReading(site, prior, hrs, reading(175), lagos)
	if small == nil || small.EstimatedBefore != 180 || small.Difference != -5 || small.ReviewWarning {
		t.Fatalf("5 L short of 180 is not worth a review: %+v", small)
	}
	big := domain.CompareReading(site, prior, hrs, reading(100), lagos)
	if big == nil || big.Difference != -80 || !big.ReviewWarning {
		t.Fatalf("80 L short of 180 must be flagged: %+v", big)
	}
	// With nothing earlier there is nothing to compare with.
	if v := domain.CompareReading(site, nil, hrs, reading(100), lagos); v != nil {
		t.Fatalf("a first reading has no estimate to compare with: %+v", v)
	}
	// A delivery is not a reading.
	if v := domain.CompareReading(site, prior, hrs, domain.FuelEntry{Kind: domain.FuelDelivery, Litres: 50, OccurredAt: fuelAt(4, 9)}, lagos); v != nil {
		t.Fatal("only a tank reading is compared")
	}
}

// Quality is built only from what is on record: how many days were reported,
// how many reports were late, and how much of the time nobody confirmed.
func TestAssessQuality(t *testing.T) {
	rep := func(late, supersedes string) domain.DailyReport {
		return domain.DailyReport{LateReason: late, SupersedesID: supersedes}
	}
	sum := domain.DailySummary{EligibleMinutes: 1000, UnknownMinutes: 250}

	good := domain.AssessQuality(20, []domain.DailyReport{rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""),
		rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", "x")}, sum)
	if good.Level != domain.QualityGood || good.Completeness != 100 || good.Corrections != 1 || good.UnconfirmedShare != 25 {
		t.Fatalf("every day reported is good: %+v", good)
	}
	mostlyLate := domain.AssessQuality(10, []domain.DailyReport{rep("r", ""), rep("r", ""), rep("r", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", ""), rep("", "")}, sum)
	if mostlyLate.Level != domain.QualityFair || mostlyLate.LateReports != 3 {
		t.Fatalf("complete but 30%% late is only fair: %+v", mostlyLate)
	}
	if none := domain.AssessQuality(7, nil, sum); none.Level != domain.QualityPoor || none.Completeness != 0 {
		t.Fatalf("nothing reported is poor: %+v", none)
	}
}
