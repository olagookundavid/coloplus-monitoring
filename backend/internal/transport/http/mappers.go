package httpapi

import (
	"fmt"
	"time"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"
)

func toUserDTO(u domain.User) dto.UserDTO {
	return dto.UserDTO{
		ID: u.ID, Email: u.Email, FullName: u.FullName, Role: string(u.Role),
		Active: u.Active, MFAEnabled: u.TOTPEnabled, Invited: u.Invited(), LastLoginAt: u.LastLoginAt,
	}
}

func toSiteDTO(s domain.Site) dto.SiteDTO {
	sources := make([]string, len(s.PowerSources))
	for i, p := range s.PowerSources {
		sources[i] = string(p)
	}
	out := dto.SiteDTO{
		ID: s.ID, AirtelCode: s.AirtelCode, ColoplusCode: s.ColoplusCode, Name: s.Name, Address: s.Address,
		Region: s.Region, State: s.State, SiteType: s.SiteType, PowerSources: sources,
		ServiceStartMinute: s.ServiceStartMinute, ServiceEndMinute: s.ServiceEndMinute,
		FreshnessHours: s.FreshnessHours, Status: string(s.Status),
		Latitude: s.Latitude, Longitude: s.Longitude, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.CommissioningDate != nil {
		d := s.CommissioningDate.Format("2006-01-02")
		out.CommissioningDate = &d
	}
	return out
}

func fromSiteRequest(r dto.SiteRequest) (domain.Site, error) {
	sources := make([]domain.PowerSource, len(r.PowerSources))
	for i, p := range r.PowerSources {
		sources[i] = domain.PowerSource(p)
	}
	site := domain.Site{
		AirtelCode: r.AirtelCode, ColoplusCode: r.ColoplusCode, Name: r.Name, Address: r.Address,
		Region: r.Region, State: r.State, SiteType: r.SiteType, PowerSources: sources,
		ServiceStartMinute: r.ServiceStartMinute, ServiceEndMinute: r.ServiceEndMinute,
		FreshnessHours: r.FreshnessHours, Status: domain.SiteStatus(r.Status),
		Latitude: r.Latitude, Longitude: r.Longitude,
	}
	if r.CommissioningDate != nil && *r.CommissioningDate != "" {
		d, err := time.Parse("2006-01-02", *r.CommissioningDate)
		if err != nil {
			return domain.Site{}, badRequest("the commissioning date must look like 2026-09-01")
		}
		site.CommissioningDate = &d
	}
	return site, nil
}

func fromIntervalDTOs(in []dto.IntervalDTO) []domain.PowerInterval {
	out := make([]domain.PowerInterval, len(in))
	for i, iv := range in {
		basis := domain.Basis(iv.Basis)
		if basis == "" {
			basis = domain.BasisObserved
		}
		out[i] = domain.PowerInterval{
			Start: iv.Start, End: iv.End, State: domain.IntervalState(iv.State),
			Source: domain.PowerSource(iv.Source), Basis: basis,
		}
	}
	return out
}

func toIntervalDTOs(in []domain.PowerInterval, loc *time.Location) []dto.IntervalDTO {
	out := make([]dto.IntervalDTO, len(in))
	for i, iv := range in {
		out[i] = dto.IntervalDTO{
			ID: iv.ID, Start: iv.Start.In(loc), End: iv.End.In(loc),
			State: string(iv.State), Source: string(iv.Source), Basis: string(iv.Basis),
		}
	}
	return out
}

func toReportDTO(r domain.DailyReport, loc *time.Location) dto.ReportDTO {
	return dto.ReportDTO{
		ID: r.ID, SiteID: r.SiteID, ReportDate: r.Date.Format("2006-01-02"), Revision: r.Revision,
		Status: string(r.Status), AuthorID: r.AuthorID, Notes: r.Notes,
		Intervals: toIntervalDTOs(r.Intervals, loc), SubmittedAt: r.SubmittedAt, ReviewedAt: r.ReviewedAt,
		ReviewerID: r.ReviewerID, ReviewComment: r.ReviewComment, SupersedesID: r.SupersedesID,
		CorrectionReason: r.CorrectionReason, RowVersion: r.RowVersion, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// toPublishedDTO is the customer view: approved facts, no internal commentary.
func toPublishedDTO(r domain.DailyReport, loc *time.Location) dto.PublishedReportDTO {
	return dto.PublishedReportDTO{
		ID: r.ID, SiteID: r.SiteID, ReportDate: r.Date.Format("2006-01-02"), Revision: r.Revision,
		Status: string(r.Status), Intervals: toIntervalDTOs(r.Intervals, loc), ApprovedAt: r.ReviewedAt,
	}
}

func toSummaryDTO(s domain.DailySummary) dto.SummaryDTO {
	return dto.SummaryDTO{
		EligibleMinutes: s.EligibleMinutes, ServedMinutes: s.ServedMinutes,
		OutageMinutes: s.OutageMinutes, UnknownMinutes: s.UnknownMinutes,
		CoveragePct: round2(s.Coverage), ReportedAvailabilityPct: round2(s.ReportedAvailability),
		ConfirmedServedSharePct: round2(s.ConfirmedServedShare), Provisional: s.Provisional(),
	}
}

// round2 rounds for display only; the domain keeps full precision.
func round2(v *float64) *float64 {
	if v == nil {
		return nil
	}
	r := float64(int64(*v*100+0.5)) / 100
	return &r
}

func toCalculationDTO(c services.Calculation, loc *time.Location) dto.CalculationDTO {
	warnings := c.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return dto.CalculationDTO{
		Formula: c.Formula, Version: c.Version, Basis: "calculation from reported inputs",
		InputReport: c.ReportID, InputRev: c.Revision,
		WindowStart: c.From.In(loc), WindowEnd: c.To.In(loc), Timezone: loc.String(), ComputedAt: c.ComputedAt,
		Explanation: "Eligible minutes are the site's service window on this day. Served and outage minutes are the union of " +
			"the observed intervals in the approved report, clipped to that window, so overlapping or duplicate entries " +
			"never count twice. Unknown minutes are what remains: they are neither confirmed uptime nor confirmed downtime. " +
			"Coverage is (served + outage) / eligible. Reported availability is served / (served + outage). " +
			"Confirmed served share is served / eligible.",
		Warnings: warnings,
	}
}

func toOverviewResponse(o services.Overview, loc *time.Location) dto.OverviewResponse {
	rows := make([]dto.SiteRowDTO, len(o.Sites))
	for i, r := range o.Sites {
		row := dto.SiteRowDTO{
			SiteID: r.Site.ID, AirtelCode: r.Site.AirtelCode, ColoplusCode: r.Site.ColoplusCode, Name: r.Site.Name,
			State: r.Site.State, Status: string(r.Site.Status), Condition: string(r.Condition),
		}
		if r.AsOf != nil {
			t := r.AsOf.In(loc)
			row.AsOf = &t
		}
		if r.Summary != nil {
			sum := toSummaryDTO(*r.Summary)
			row.Summary = &sum
		}
		rows[i] = row
	}
	return dto.OverviewResponse{
		Date: o.Date.Format("2006-01-02"), Timezone: o.Timezone, GeneratedAt: o.GeneratedAt,
		ActiveSites: o.Counts.Active, PendingSites: o.Counts.PendingAcceptance, SuspendedSites: o.Counts.Suspended,
		SitesReported: o.SitesReported, SitesNotReported: o.SitesNotReported,
		SitesAwaitingUpdate: o.SitesAwaitingUpdate, SitesOutageReported: o.SitesOutageReported,
		Portfolio: toSummaryDTO(o.Portfolio), Sites: rows,
	}
}

func incidentRef(number int64) string { return fmt.Sprintf("INC-%06d", number) }

func toIncidentDTO(i domain.Incident, now time.Time) dto.IncidentDTO {
	return dto.IncidentDTO{
		ID: i.ID, Ref: incidentRef(i.Number), SiteID: i.SiteID, Category: string(i.Category),
		Severity: string(i.Severity), Impact: string(i.Impact), Title: i.Title, Description: i.Description,
		Status: string(i.Status), StartedAt: i.StartedAt, DetectedAt: i.DetectedAt, ReporterID: i.ReporterID,
		OwnerID: i.OwnerID, NextAction: i.NextAction, NextUpdateDue: i.NextUpdateDue, Overdue: i.Overdue(now),
		RestorationTarget: i.RestorationTarget, RestoredAt: i.RestoredAt, RestoredBy: i.RestoredBy,
		VerifiedAt: i.VerifiedAt, VerifiedBy: i.VerifiedBy, ClosedAt: i.ClosedAt, RootCause: i.RootCause,
		PublishedAt: i.PublishedAt, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func toPublishedIncidentDTO(i domain.Incident, updates []domain.IncidentUpdate) dto.PublishedIncidentDTO {
	out := dto.PublishedIncidentDTO{
		ID: i.ID, Ref: incidentRef(i.Number), SiteID: i.SiteID, Category: string(i.Category),
		Severity: string(i.Severity), Impact: string(i.Impact), Title: i.Title, Status: string(i.Status),
		StartedAt: i.StartedAt, RestorationTarget: i.RestorationTarget, RestoredAt: i.RestoredAt,
		Verified: i.VerifiedAt != nil, ClosedAt: i.ClosedAt,
	}
	for _, u := range updates {
		if u.CustomerVisible {
			out.Updates = append(out.Updates, dto.PublishedUpdateDTO{At: u.CreatedAt, Body: u.Body})
		}
	}
	return out
}

func toIncidentUpdateDTO(u domain.IncidentUpdate) dto.IncidentUpdateDTO {
	return dto.IncidentUpdateDTO{
		ID: u.ID, AuthorID: u.AuthorID, Kind: string(u.Kind), FromStatus: string(u.FromStatus),
		ToStatus: string(u.ToStatus), Body: u.Body, CustomerVisible: u.CustomerVisible,
		ObservedAt: u.ObservedAt, CreatedAt: u.CreatedAt,
	}
}

func disputeRef(number int64) string { return fmt.Sprintf("DSP-%06d", number) }

func dateString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// toDisputeDTO builds the view for the caller's side. Airtel never gets
// Coloplus staff ids: who owns or resolved a dispute is Coloplus's own detail.
func toDisputeDTO(d domain.Dispute, staff bool) dto.DisputeDTO {
	out := dto.DisputeDTO{
		ID: d.ID, Ref: disputeRef(d.Number), SiteID: d.SiteID, ReportID: d.ReportID, IntervalID: d.IntervalID,
		Metric: string(d.Metric), PeriodFrom: dateString(d.PeriodFrom), PeriodTo: dateString(d.PeriodTo),
		Category: string(d.Category), Title: d.Title, Description: d.Description, Status: string(d.Status),
		WaitingOn: string(d.WaitingOn()), Outcome: string(d.Outcome), Resolution: d.Resolution,
		ResolutionReportID: d.ResolutionReportID, ResolvedAt: d.ResolvedAt, AcknowledgedAt: d.AcknowledgedAt,
		ReopenCount: d.ReopenCount, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	if staff {
		out.RaisedBy, out.OwnerID, out.ResolvedBy = d.RaisedBy, d.OwnerID, d.ResolvedBy
	}
	return out
}

func toDisputeEventDTO(e domain.DisputeEvent, staff bool) dto.DisputeEventDTO {
	out := dto.DisputeEventDTO{
		ID: e.ID, Party: string(e.Party), Kind: string(e.Kind), FromStatus: string(e.FromStatus),
		ToStatus: string(e.ToStatus), Body: e.Body, CreatedAt: e.CreatedAt,
	}
	if staff {
		out.Internal, out.AuthorID = e.Internal, e.AuthorID
	}
	return out
}

func toAcceptanceDTO(a domain.Acceptance, now time.Time) dto.AcceptanceDTO {
	return dto.AcceptanceDTO{
		ID: a.ID, SiteID: a.SiteID, Stage: string(a.Stage), HeldFrom: string(a.HeldFrom),
		WaitingOn: string(a.Responsible), NextAction: a.NextAction, DueAt: a.DueAt, Overdue: a.Overdue(now),
		Dossier: a.Dossier, Reason: a.Reason, HoldReason: a.HoldReason, ReadyAgeHours: int(a.ReadyAge(now).Hours()),
		RequestedAt: a.RequestedAt, BuiltAt: a.BuiltAt, ReadyAt: a.ReadyAt, InspectedAt: a.InspectedAt,
		DecidedAt: a.DecidedAt, UpdatedAt: a.UpdatedAt,
	}
}

func toAcceptanceEventDTO(e domain.AcceptanceEvent) dto.AcceptanceEventDTO {
	return dto.AcceptanceEventDTO{
		ID: e.ID, Party: string(e.Party), FromStage: string(e.FromStage), ToStage: string(e.ToStage),
		Body: e.Body, CreatedAt: e.CreatedAt,
	}
}
