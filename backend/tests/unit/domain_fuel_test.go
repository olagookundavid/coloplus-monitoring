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
