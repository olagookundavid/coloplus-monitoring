package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type SiteStatus string

const (
	SitePendingAcceptance SiteStatus = "pending_acceptance"
	SiteActive            SiteStatus = "active"
	SiteSuspended         SiteStatus = "suspended"
)

type PowerSource string

const (
	SourceGrid           PowerSource = "grid"
	SourceGenerator      PowerSource = "generator"
	SourceBattery        PowerSource = "battery"
	SourceSolar          PowerSource = "solar"
	SourceBuildingSupply PowerSource = "building_supply"
)

func ValidPowerSource(s PowerSource) bool {
	switch s {
	case SourceGrid, SourceGenerator, SourceBattery, SourceSolar, SourceBuildingSupply:
		return true
	}
	return false
}

func ValidSiteStatus(s SiteStatus) bool {
	switch s {
	case SitePendingAcceptance, SiteActive, SiteSuspended:
		return true
	}
	return false
}

const minutesPerDay = 24 * 60

// Site is the master record. Its ID is immutable and internal.
type Site struct {
	ID           string
	AirtelCode   string
	ColoplusCode string
	Name         string
	Address      string
	Region       string
	State        string
	SiteType     string
	PowerSources []PowerSource

	// The contractual service window as minutes from local midnight. A site
	// serviced around the clock is 0 to 1440.
	ServiceStartMinute int
	ServiceEndMinute   int

	// FreshnessHours is how long an approved report keeps the site "current".
	FreshnessHours int

	Status            SiteStatus
	CommissioningDate *time.Time
	Latitude          *float64
	Longitude         *float64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Validate checks the rules every site record must meet.
func (s Site) Validate() error {
	var problems []string
	add := func(msg string) { problems = append(problems, msg) }

	if strings.TrimSpace(s.AirtelCode) == "" {
		add("the Airtel site code is required")
	}
	if strings.TrimSpace(s.ColoplusCode) == "" {
		add("the Coloplus site code is required")
	}
	if strings.TrimSpace(s.Name) == "" {
		add("the site name is required")
	}
	if strings.TrimSpace(s.State) == "" {
		add("the state is required")
	}
	if len(s.PowerSources) == 0 {
		add("at least one installed power source is required")
	}
	seen := map[PowerSource]bool{}
	for _, p := range s.PowerSources {
		if !ValidPowerSource(p) {
			add(fmt.Sprintf("%q is not a known power source", p))
		}
		if seen[p] {
			add(fmt.Sprintf("power source %q is listed twice", p))
		}
		seen[p] = true
	}
	if s.ServiceStartMinute < 0 || s.ServiceEndMinute > minutesPerDay || s.ServiceStartMinute >= s.ServiceEndMinute {
		add("the service window must start before it ends, between 00:00 and 24:00")
	}
	if s.FreshnessHours < 1 {
		add("the freshness threshold must be at least 1 hour")
	}
	if !ValidSiteStatus(s.Status) {
		add(fmt.Sprintf("%q is not a known site status", s.Status))
	}
	if (s.Latitude == nil) != (s.Longitude == nil) {
		add("latitude and longitude must be given together")
	}
	if s.Latitude != nil && s.Longitude != nil && (*s.Latitude < -90 || *s.Latitude > 90 || *s.Longitude < -180 || *s.Longitude > 180) {
		add("the coordinates are out of range")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (s Site) HasSource(p PowerSource) bool {
	for _, have := range s.PowerSources {
		if have == p {
			return true
		}
	}
	return false
}
