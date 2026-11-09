package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
)

const (
	// MaxSiteImportRows bounds one import so it stays a single short transaction.
	MaxSiteImportRows = 1000
	// MaxSiteImportBytes bounds the file text; the request body limit is 1 MB.
	MaxSiteImportBytes = 800 << 10
)

// SiteImportColumns are the columns a site import file may carry. Only
// customer_site_code must be present; a new site also needs the columns Validate
// requires. On an existing site a blank cell leaves the value unchanged.
var SiteImportColumns = []string{
	"customer_site_code", "coloplus_code", "name", "state", "power_sources", "region", "address", "site_type",
	"responsible_team", "service_start", "service_end", "freshness_hours", "status", "commissioning_date",
	"latitude", "longitude", "consumption_lph", "tank_capacity_litres", "unreported_time",
	"generator_capacity", "access_contact_name", "access_contact_phone", "supervisor_name",
}

type SiteImportAction string

const (
	ImportCreate    SiteImportAction = "create"
	ImportUpdate    SiteImportAction = "update"
	ImportUnchanged SiteImportAction = "unchanged"
	ImportError     SiteImportAction = "error"
)

// SiteImportRow is the outcome for one data row. Line is the line in the file
// (the header is line 1).
type SiteImportRow struct {
	Line             int
	CustomerSiteCode string
	Action           SiteImportAction
	Problems         []string
}

// SiteImportResult is all-or-nothing: Applied is true only when every row was
// good and it was not a dry run.
type SiteImportResult struct {
	DryRun                                bool
	Applied                               bool
	Created, Updated, Unchanged, Rejected int
	Rows                                  []SiteImportRow
}

// ImportCSV loads sites from a CSV file, creating new ones and updating
// existing ones matched on the customer site code. With dryRun it only reports what
// would happen. Any bad row means nothing is written.
func (s *SiteService) ImportCSV(ctx context.Context, actor domain.Actor, data []byte, dryRun bool) (SiteImportResult, error) {
	if !actor.Can(domain.PermSitesManage) {
		return SiteImportResult{}, apperrors.ErrForbidden
	}
	if len(data) > MaxSiteImportBytes {
		return SiteImportResult{}, invalid("the file is larger than %d KB", MaxSiteImportBytes>>10)
	}
	header, records, err := readSiteCSV(data)
	if err != nil {
		return SiteImportResult{}, err
	}

	existing, err := s.sites.CodeIndex(ctx)
	if err != nil {
		return SiteImportResult{}, err
	}
	byCustomerCode, byColoplus := map[string]string{}, map[string]string{}
	for _, c := range existing {
		byCustomerCode[strings.ToLower(c.CustomerSiteCode)] = c.ID
		byColoplus[strings.ToLower(c.ColoplusCode)] = c.ID
	}
	ids := make([]string, 0, len(records))
	for _, rec := range records {
		if id, ok := byCustomerCode[strings.ToLower(strings.TrimSpace(rec.cell(header, "customer_site_code")))]; ok {
			ids = append(ids, id)
		}
	}
	current := map[string]domain.Site{}
	if len(ids) > 0 {
		loaded, err := s.sites.GetMany(ctx, ids)
		if err != nil {
			return SiteImportResult{}, err
		}
		for _, site := range loaded {
			current[site.ID] = site
		}
	}

	res := SiteImportResult{DryRun: dryRun}
	type plan struct {
		site   domain.Site
		action SiteImportAction
	}
	var plans []plan
	seenCustomerCode, seenColoplus := map[string]int{}, map[string]int{}

	for _, rec := range records {
		row := SiteImportRow{Line: rec.line, CustomerSiteCode: strings.TrimSpace(rec.cell(header, "customer_site_code"))}
		site, action, problems := buildImportSite(rec, header, byCustomerCode, current)
		// A Coloplus code must not already belong to another site, in the
		// register or earlier in this file.
		if site.ColoplusCode != "" {
			key := strings.ToLower(site.ColoplusCode)
			if owner, ok := byColoplus[key]; ok && owner != site.ID {
				problems = append(problems, fmt.Sprintf("the Coloplus code %q already belongs to another site", site.ColoplusCode))
			}
			if first, ok := seenColoplus[key]; ok {
				problems = append(problems, fmt.Sprintf("the Coloplus code %q is repeated (also on line %d)", site.ColoplusCode, first))
			}
			seenColoplus[key] = rec.line
		}
		if key := strings.ToLower(row.CustomerSiteCode); key != "" {
			if first, ok := seenCustomerCode[key]; ok {
				problems = append(problems, fmt.Sprintf("the customer site code %q is repeated (also on line %d)", row.CustomerSiteCode, first))
			}
			seenCustomerCode[key] = rec.line
		}
		if len(problems) > 0 {
			row.Action, row.Problems = ImportError, problems
			res.Rejected++
		} else {
			row.Action = action
			plans = append(plans, plan{site, action})
			switch action {
			case ImportCreate:
				res.Created++
			case ImportUpdate:
				res.Updated++
			default:
				res.Unchanged++
			}
		}
		res.Rows = append(res.Rows, row)
	}
	if res.Rejected > 0 || dryRun {
		return res, nil
	}

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		for _, p := range plans {
			switch p.action {
			case ImportCreate:
				created, err := s.sites.Create(ctx, p.site)
				if err != nil {
					return err
				}
				if err := s.audit.RecordInTx(ctx, domain.AuditEvent{
					ActorID: actor.UserID, Action: "site.created", EntityType: "site", EntityID: created.ID,
					Metadata: map[string]any{"customer_site_code": created.CustomerSiteCode, "via": "import"},
				}); err != nil {
					return err
				}
			case ImportUpdate:
				if err := s.sites.Update(ctx, p.site); err != nil {
					return err
				}
				if err := s.audit.RecordInTx(ctx, domain.AuditEvent{
					ActorID: actor.UserID, Action: "site.updated", EntityType: "site", EntityID: p.site.ID,
					Metadata: map[string]any{"status": string(p.site.Status), "via": "import"},
				}); err != nil {
					return err
				}
			}
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "sites.imported", EntityType: "site", EntityID: "",
			Metadata: map[string]any{"created": res.Created, "updated": res.Updated, "unchanged": res.Unchanged},
		})
	})
	if err != nil {
		return SiteImportResult{}, err
	}
	res.Applied = true
	return res, nil
}

type importRecord struct {
	line  int
	cells []string
}

func (r importRecord) cell(header map[string]int, col string) string {
	i, ok := header[col]
	if !ok || i >= len(r.cells) {
		return ""
	}
	return strings.TrimSpace(r.cells[i])
}

func readSiteCSV(data []byte) (map[string]int, []importRecord, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	head, err := r.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, invalid("the file is empty")
	}
	if err != nil {
		return nil, nil, invalid("the file is not a readable CSV: %s", err.Error())
	}
	known := map[string]bool{}
	for _, c := range SiteImportColumns {
		known[c] = true
	}
	header := map[string]int{}
	for i, h := range head {
		name := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(h)), " ", "_")
		if name == "" {
			continue
		}
		if !known[name] {
			return nil, nil, invalid("unknown column %q; allowed columns are %s", strings.TrimSpace(h), strings.Join(SiteImportColumns, ", "))
		}
		if _, dup := header[name]; dup {
			return nil, nil, invalid("column %q appears twice", name)
		}
		header[name] = i
	}
	if _, ok := header["customer_site_code"]; !ok {
		return nil, nil, invalid("the customer_site_code column is required")
	}

	var records []importRecord
	for line := 2; ; line++ {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, invalid("line %d is not readable: %s", line, err.Error())
		}
		blank := true
		for _, c := range cells {
			if strings.TrimSpace(c) != "" {
				blank = false
			}
		}
		if blank {
			continue
		}
		if len(records) >= MaxSiteImportRows {
			return nil, nil, invalid("more than %d sites in one file; split it", MaxSiteImportRows)
		}
		records = append(records, importRecord{line: line, cells: cells})
	}
	if len(records) == 0 {
		return nil, nil, invalid("the file has a header but no sites")
	}
	return header, records, nil
}

// buildImportSite overlays the row onto the existing site (or a new one) and
// validates the result.
func buildImportSite(rec importRecord, header map[string]int, byCustomerCode map[string]string, current map[string]domain.Site) (domain.Site, SiteImportAction, []string) {
	var problems []string
	var site domain.Site
	action := ImportCreate
	code := rec.cell(header, "customer_site_code")
	if code == "" {
		return site, ImportError, []string{"the customer site code is required"}
	}
	var before domain.Site
	if id, ok := byCustomerCode[strings.ToLower(code)]; ok {
		before = current[id]
		site = before
		action = ImportUpdate
	}

	bad := func(col, why string) { problems = append(problems, fmt.Sprintf("%s: %s", col, why)) }
	for _, col := range SiteImportColumns {
		v := rec.cell(header, col)
		if v == "" {
			continue
		}
		switch col {
		case "customer_site_code":
			site.CustomerSiteCode = v
		case "coloplus_code":
			site.ColoplusCode = v
		case "name":
			site.Name = v
		case "state":
			site.State = v
		case "region":
			site.Region = v
		case "address":
			site.Address = v
		case "site_type":
			site.SiteType = v
		case "responsible_team":
			site.ResponsibleTeam = v
		case "generator_capacity":
			site.GeneratorCapacity = v
		case "access_contact_name":
			site.AccessContactName = v
		case "access_contact_phone":
			site.AccessContactPhone = v
		case "supervisor_name":
			site.SupervisorName = v
		case "power_sources":
			site.PowerSources = nil
			for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ';' || r == '|' || r == ',' }) {
				site.PowerSources = append(site.PowerSources, domain.PowerSource(strings.ToLower(strings.TrimSpace(p))))
			}
		case "service_start", "service_end":
			m, ok := parseClockMinutes(v)
			if !ok {
				bad(col, fmt.Sprintf("%q is not a time like 06:00", v))
			} else if col == "service_start" {
				site.ServiceStartMinute = m
			} else {
				site.ServiceEndMinute = m
			}
		case "freshness_hours":
			n, err := strconv.Atoi(v)
			if err != nil {
				bad(col, fmt.Sprintf("%q is not a whole number", v))
			}
			site.FreshnessHours = n
		case "status":
			site.Status = domain.SiteStatus(strings.ToLower(v))
		case "unreported_time":
			site.UnreportedTime = domain.UnreportedPolicy(strings.ToLower(v))
		case "commissioning_date":
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				bad(col, fmt.Sprintf("%q is not a date like 2026-09-01", v))
			}
			site.CommissioningDate = &d
		case "latitude", "longitude", "consumption_lph", "tank_capacity_litres":
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				bad(col, fmt.Sprintf("%q is not a number", v))
				break
			}
			switch col {
			case "latitude":
				site.Latitude = &f
			case "longitude":
				site.Longitude = &f
			case "consumption_lph":
				site.ConsumptionLPH = &f
			default:
				site.TankCapacityLitres = &f
			}
		}
	}
	if len(problems) > 0 {
		return site, ImportError, problems
	}
	if action == ImportCreate {
		site = withDefaults(site)
	}
	if err := site.Validate(); err != nil {
		return site, ImportError, strings.Split(err.Error(), "; ")
	}
	if action == ImportUpdate && sameSite(before, site) {
		action = ImportUnchanged
	}
	return site, action, nil
}

func sameSite(a, b domain.Site) bool {
	a.CreatedAt, a.UpdatedAt, b.CreatedAt, b.UpdatedAt = time.Time{}, time.Time{}, time.Time{}, time.Time{}
	return reflect.DeepEqual(a, b)
}

// parseClockMinutes reads 06:00 or 24:00 as minutes from midnight.
func parseClockMinutes(v string) (int, bool) {
	h, m, ok := strings.Cut(v, ":")
	if !ok {
		return 0, false
	}
	hh, err1 := strconv.Atoi(h)
	mm, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hh < 0 || mm < 0 || mm > 59 || hh > 24 || (hh == 24 && mm != 0) {
		return 0, false
	}
	return hh*60 + mm, true
}
