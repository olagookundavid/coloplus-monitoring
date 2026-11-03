package integration

import (
	"strings"
	"testing"

	"coloplus-backend/internal/domain"
)

func (s *stack) postCSV(path, csv, tok string) reply {
	s.t.Helper()
	dry := strings.Contains(path, "dry_run=true")
	return s.call("POST", "/api/v1/sites/import", map[string]any{"csv": csv, "dry_run": dry}, tok)
}

func TestSiteImport_AllOrNothingDryRunAndBlankCellsKeepValues(t *testing.T) {
	s := newStack(t)
	adminEmail, _ := s.seedUser(domain.RoleAdmin)
	admin := s.login(adminEmail)
	repEmail, _ := s.seedUser(domain.RoleReporter)
	reporter := s.login(repEmail)

	a, b, cp1, cp2 := uniq("IMP"), uniq("IMP"), uniq("CPI"), uniq("CPI")
	head := "airtel_code,coloplus_code,name,state,power_sources,consumption_lph,service_start,service_end\n"
	good := head + a + "," + cp1 + ",Alpha,Lagos,grid;generator,4.5,00:00,24:00\n" + b + "," + cp2 + ",Beta,Ogun,grid,,06:00,18:00\n"

	// Not for field staff.
	s.expect(s.postCSV("/api/v1/sites/import", good, reporter), 403)

	// A dry run writes nothing.
	r := s.expect(s.postCSV("/api/v1/sites/import?dry_run=true", good, admin), 200)
	if r.str("created") != "2" || r.str("applied") != "false" {
		t.Fatalf("dry run = %s", r.Raw)
	}
	if got := s.expect(s.call("GET", "/api/v1/sites?q="+a, nil, admin), 200); got.str("total") != "0" {
		t.Fatalf("a dry run created a site: %s", got.Raw)
	}

	// One bad row (unknown source, line 3) means no row is written.
	bad := head + a + "," + cp1 + ",Alpha,Lagos,grid,,,\n" + b + "," + cp2 + ",Beta,Ogun,wind,,,\n"
	r = s.expect(s.postCSV("/api/v1/sites/import", bad, admin), 200)
	if r.str("applied") != "false" || r.str("rejected") != "1" {
		t.Fatalf("bad file = %s", r.Raw)
	}
	if got := s.expect(s.call("GET", "/api/v1/sites?q="+a, nil, admin), 200); got.str("total") != "0" {
		t.Fatalf("a rejected file still created a site: %s", got.Raw)
	}

	// The good file is applied; running it again changes nothing.
	r = s.expect(s.postCSV("/api/v1/sites/import", good, admin), 200)
	if r.str("applied") != "true" || r.str("created") != "2" {
		t.Fatalf("import = %s", r.Raw)
	}
	r = s.expect(s.postCSV("/api/v1/sites/import", good, admin), 200)
	if r.str("created") != "0" || r.str("unchanged") != "2" {
		t.Fatalf("re-import = %s", r.Raw)
	}

	// An update with blank cells leaves the fuel rate and window alone.
	upd := "airtel_code,name,consumption_lph\n" + a + ",Alpha renamed,\n"
	s.expect(s.postCSV("/api/v1/sites/import", upd, admin), 200)
	got := s.expect(s.call("GET", "/api/v1/sites?q="+a, nil, admin), 200)
	site := got.list("sites")[0].(map[string]any)
	if site["name"] != "Alpha renamed" || site["consumption_lph"] != 4.5 {
		t.Fatalf("update lost a value: %v", site)
	}
}
