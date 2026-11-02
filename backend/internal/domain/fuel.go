package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type FuelKind string

const (
	// FuelDelivery is fuel added to the tank: litres on top of what was there.
	FuelDelivery FuelKind = "delivery"
	// FuelReading is what the tank held when someone looked (a dip): the
	// litres in the tank, replacing any estimate.
	FuelReading FuelKind = "reading"
)

func ValidFuelKind(k FuelKind) bool { return k == FuelDelivery || k == FuelReading }

const (
	FuelFormula        = "fuel-estimate"
	FuelFormulaVersion = "1"
	maxFuelNote        = 500
	// A fuel entry may be back-dated this far, and no further.
	maxFuelBackdate = 400 * 24 * time.Hour
)

// FuelEntry is one thing said about a site's tank. Entries are never edited;
// a mistaken one is voided, with a reason, and left in the record.
type FuelEntry struct {
	ID         string
	SiteID     string
	Kind       FuelKind
	Litres     float64
	OccurredAt time.Time
	// Note is Coloplus's own and is never shown to the customer.
	Note       string
	RecordedBy string
	ClientKey  string
	CreatedAt  time.Time
	VoidedAt   *time.Time
	VoidedBy   string
	VoidReason string
}

func (e FuelEntry) Voided() bool { return e.VoidedAt != nil }

// ValidateNew lists what is wrong with an entry about to be recorded.
func (e FuelEntry) ValidateNew(site Site, now time.Time) []string {
	var problems []string
	if !ValidFuelKind(e.Kind) {
		problems = append(problems, "say whether this is a delivery or a tank reading")
	}
	switch {
	case math.IsNaN(e.Litres) || math.IsInf(e.Litres, 0) || e.Litres < 0:
		problems = append(problems, "the litres must be a number, zero or more")
	case e.Kind == FuelDelivery && e.Litres == 0:
		problems = append(problems, "a delivery is more than 0 litres")
	case math.Round(e.Litres*10) != e.Litres*10:
		problems = append(problems, "give litres to one decimal place at most")
	case site.TankCapacityLitres != nil && e.Litres > *site.TankCapacityLitres:
		problems = append(problems, fmt.Sprintf("that is more than the tank holds (%.0f litres); check the number", *site.TankCapacityLitres))
	}
	switch {
	case e.OccurredAt.IsZero():
		problems = append(problems, "say when it happened")
	case e.OccurredAt.After(now.Add(5 * time.Minute)):
		problems = append(problems, "it cannot have happened in the future")
	case now.Sub(e.OccurredAt) > maxFuelBackdate:
		problems = append(problems, "that is too long ago to record")
	}
	if utf8.RuneCountInString(strings.TrimSpace(e.Note)) > maxFuelNote {
		problems = append(problems, "the note is too long")
	}
	return problems
}

// FuelEstimate is what the system believes is in a site's tank, and how it
// got there. It is an estimate, never a meter reading: the level comes from a
// person's entries, less the agreed litres-per-hour for the hours the
// generator was reported running.
type FuelEstimate struct {
	// Available is false when there is nothing to estimate from; Reason says
	// why, in words for the person looking.
	Available bool
	Reason    string

	Litres *float64
	// HoursRemaining is how long the estimated level would run the generator.
	HoursRemaining *float64

	// Anchor is the entry the estimate starts from: the latest tank reading, or
	// the first delivery when nobody has taken a reading.
	AnchorKind FuelKind
	AnchorAt   time.Time
	// AssumedEmptyStart is true when there is no reading and the first entry
	// is a delivery: the tank is taken to have been empty before it.
	AssumedEmptyStart bool

	ConsumptionLPH float64
	// HoursRun and ConsumedLitres cover the days reported since the anchor.
	HoursRun       float64
	ConsumedLitres float64
	// UnreportedDays are days since the anchor with no approved generator
	// hours. Their fuel use is unknown, not zero, so while there are any the
	// level shown is the most the tank could hold.
	UnreportedDays int
	Provisional    bool

	// EstimatedEmpty means the estimate ran out. OverCapacity means it came to
	// more than the tank holds, so an entry is probably wrong.
	EstimatedEmpty bool
	OverCapacity   bool

	AsOf    time.Time
	Formula string
	Version string
}

// EstimateFuel works out the tank level at asOf from a site's entries (any
// order; voided ones are ignored) and the approved generator hours of each
// reporting day, keyed by "2006-01-02". An entry counts as the state of the
// tank at the end of its reporting day, and today's use is not counted until
// today is reported.
//
//	level at end of day = level at end of the day before
//	                      + deliveries that day
//	                      - agreed litres per hour x hours run that day
//
// A reading sets the level outright and starts the count again.
func EstimateFuel(site Site, entries []FuelEntry, hoursByDay map[string]*float64, asOf time.Time, loc *time.Location) FuelEstimate {
	out := FuelEstimate{AsOf: asOf, Formula: FuelFormula, Version: FuelFormulaVersion}
	if site.ConsumptionLPH == nil {
		out.Reason = "no fuel consumption rate has been agreed for this site yet"
		return out
	}
	out.ConsumptionLPH = *site.ConsumptionLPH

	live := make([]FuelEntry, 0, len(entries))
	for _, e := range entries {
		if !e.Voided() {
			live = append(live, e)
		}
	}
	if len(live) == 0 {
		out.Reason = "no fuel has been recorded for this site yet"
		return out
	}
	sort.SliceStable(live, func(i, j int) bool { return live[i].OccurredAt.Before(live[j].OccurredAt) })

	dayOf := func(t time.Time) time.Time {
		y, m, d := t.In(loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	key := func(d time.Time) string { return d.Format("2006-01-02") }
	today := dayOf(asOf)
	first := dayOf(live[0].OccurredAt)
	if first.After(today) {
		first = today
	}

	byDay := map[string][]FuelEntry{}
	for _, e := range live {
		byDay[key(dayOf(e.OccurredAt))] = append(byDay[key(dayOf(e.OccurredAt))], e)
	}

	var level float64
	out.AnchorKind, out.AnchorAt = live[0].Kind, live[0].OccurredAt
	out.AssumedEmptyStart = live[0].Kind == FuelDelivery
	emptied := false

	for day := first; !day.After(today); day = day.AddDate(0, 0, 1) {
		readToday := false
		for _, e := range byDay[key(day)] {
			switch e.Kind {
			case FuelReading:
				level, readToday, emptied = e.Litres, true, false
				out.AnchorKind, out.AnchorAt, out.AssumedEmptyStart = FuelReading, e.OccurredAt, false
				out.HoursRun, out.ConsumedLitres, out.UnreportedDays = 0, 0, 0
			case FuelDelivery:
				level += e.Litres
			}
		}
		// A reading already includes that day's use; today is not over yet.
		if readToday || day.Equal(today) {
			continue
		}
		h := hoursByDay[key(day)]
		if h == nil {
			out.UnreportedDays++
			continue
		}
		used := *site.ConsumptionLPH * *h
		out.HoursRun += *h
		out.ConsumedLitres += used
		level -= used
		if level < 0 {
			level, emptied = 0, true
		}
	}

	level = math.Round(level*10) / 10
	out.Available = true
	out.Litres = &level
	out.Provisional = out.UnreportedDays > 0
	out.EstimatedEmpty = emptied && level == 0
	if site.TankCapacityLitres != nil && level > *site.TankCapacityLitres {
		out.OverCapacity = true
	}
	if *site.ConsumptionLPH > 0 {
		h := math.Round(level / *site.ConsumptionLPH * 10) / 10
		out.HoursRemaining = &h
	}
	out.ConsumedLitres = math.Round(out.ConsumedLitres*10) / 10
	out.HoursRun = math.Round(out.HoursRun*10) / 10
	return out
}

// Low reports whether the estimate says the tank is nearly out: it ran out, or
// would run the generator for no more than lowHours. An estimate that is
// unavailable is never "low" (nor fine): there is nothing to say.
func (e FuelEstimate) Low(lowHours float64) bool {
	if !e.Available || e.Litres == nil {
		return false
	}
	if e.EstimatedEmpty || *e.Litres == 0 {
		return true
	}
	return e.HoursRemaining != nil && *e.HoursRemaining <= lowHours
}

// FuelVariance compares what a tank reading found with what the estimate
// expected just before it. It is a check on the agreed rate and on the
// entries, not a correction: the reading wins and the estimate restarts from it.
type FuelVariance struct {
	// EstimatedBefore is the level the estimate gave at the time of the reading.
	EstimatedBefore float64
	Reading         float64
	// Difference is reading minus estimate: negative when the tank held less
	// than expected (more fuel used, or fuel taken).
	Difference float64
	// Provisional is true when the estimate had days with no approved hours, so
	// it was only an upper bound.
	Provisional bool
	// ReviewWarning is set when the gap is too large to be rounding; someone
	// should look at the rate, the hours reported, or the entries.
	ReviewWarning bool
}

const (
	// fuelVarianceShare of the larger figure, and at least fuelVarianceFloor
	// litres, is what counts as a gap worth a look. Proposed defaults, to be
	// agreed with the customer.
	fuelVarianceShare = 0.10
	fuelVarianceFloor = 20.0
)

// CompareReading works out the variance for a tank reading, from the entries
// made before it. It is nil when there was nothing earlier to compare with.
func CompareReading(site Site, entries []FuelEntry, hoursByDay map[string]*float64, reading FuelEntry, loc *time.Location) *FuelVariance {
	if reading.Kind != FuelReading || reading.Voided() {
		return nil
	}
	var before []FuelEntry
	for _, e := range entries {
		if e.ID != reading.ID && !e.Voided() && e.OccurredAt.Before(reading.OccurredAt) {
			before = append(before, e)
		}
	}
	est := EstimateFuel(site, before, hoursByDay, reading.OccurredAt, loc)
	if !est.Available || est.Litres == nil {
		return nil
	}
	diff := math.Round((reading.Litres-*est.Litres)*10) / 10
	limit := math.Max(fuelVarianceFloor, fuelVarianceShare*math.Max(reading.Litres, *est.Litres))
	return &FuelVariance{
		EstimatedBefore: *est.Litres, Reading: reading.Litres, Difference: diff,
		Provisional: est.Provisional, ReviewWarning: math.Abs(diff) > limit,
	}
}
