//go:build perf

package integration

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"coloplus-backend/internal/domain"
)

// The pilot target is a dashboard answer under 3 seconds (p95) with 1,000
// sites and 50 people at once (PRD section 11). This builds that portfolio in
// a scratch database and times the dashboard calls. It is not part of the
// normal run: point TEST_DATABASE_URL at an empty scratch database and run
//
//	go test -tags perf -run TestPerf -v ./tests/integration/
//
// The handler is called in-process, so network and TLS time are not included.
func TestPerf_DashboardAt1000SitesAnd50Users(t *testing.T) {
	w := newWorkflow(t)

	t.Log("building 1,000 sites x 30 approved days x 3 intervals ...")
	start := time.Now()
	w.exec(`INSERT INTO sites (airtel_code, coloplus_code, name, state, region, power_sources, status)
	        SELECT 'PA'||g, 'PC'||g, 'Perf '||g, 'Lagos', (ARRAY['Lagos','Abuja','Kano','Port Harcourt'])[1+g%4],
	               ARRAY['grid','generator'], 'active'
	        FROM generate_series(1, 1000) g`)
	w.exec(`INSERT INTO daily_reports (site_id, report_date, status, author_id)
	        SELECT s.id, d::date, 'draft', $1
	        FROM sites s CROSS JOIN generate_series(current_date - 30, current_date - 1, interval '1 day') d
	        WHERE s.name LIKE 'Perf %'`, w.reporterID)
	for _, iv := range [][4]string{
		{"0", "11.5", "served", "grid"}, {"11.5", "12", "outage", ""}, {"12", "23.5", "served", "generator"},
	} {
		w.exec(`INSERT INTO power_intervals (report_id, start_at, end_at, state, source)
		        SELECT r.id, (r.report_date::timestamp AT TIME ZONE 'Africa/Lagos') + ($1::numeric * interval '1 hour'),
		               (r.report_date::timestamp AT TIME ZONE 'Africa/Lagos') + ($2::numeric * interval '1 hour'), $3, $4
		        FROM daily_reports r JOIN sites s ON s.id = r.site_id WHERE s.name LIKE 'Perf %' AND r.status = 'draft'`,
			iv[0], iv[1], iv[2], iv[3])
	}
	w.exec(`UPDATE daily_reports SET status = 'approved', reviewed_at = now() WHERE status = 'draft'
	        AND site_id IN (SELECT id FROM sites WHERE name LIKE 'Perf %')`)
	w.exec(`ANALYZE`)
	t.Logf("built in %s", time.Since(start).Round(time.Second))

	email, uid := w.seedUser(domain.RoleAirtelViewer)
	w.exec(`INSERT INTO user_sites (user_id, site_id) SELECT $1, id FROM sites WHERE name LIKE 'Perf %'`, uid)
	airtel := w.login(email)

	day, _ := w.reportDay(1)
	from, _ := w.reportDay(30)
	calls := map[string]string{
		"overview, one day":    "/api/v1/dashboard/overview?date=" + day,
		"overview, 30 days":    "/api/v1/dashboard/overview?from=" + from + "&to=" + day,
		"trend, 30 days":       "/api/v1/dashboard/trend?from=" + from + "&to=" + day,
		"overview, one region": "/api/v1/dashboard/overview?date=" + day + "&region=Lagos",
	}
	names := make([]string, 0, len(calls))
	for n := range calls {
		names = append(names, n)
	}
	sort.Strings(names)

	// One person at a time first, as a baseline.
	for _, n := range names {
		var ms []time.Duration
		for i := 0; i < 8; i++ {
			t0 := time.Now()
			w.expect(w.call("GET", calls[n], nil, airtel), 200)
			ms = append(ms, time.Since(t0))
		}
		t.Logf("alone       %-22s p50 %-8s max %s", n, pct(ms, 50), pct(ms, 100))
	}

	// Then 50 people at once, each making a few of the calls.
	var mu sync.Mutex
	all := map[string][]time.Duration{}
	var wg sync.WaitGroup
	t0 := time.Now()
	for p := 0; p < 50; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for i := 0; i < 4; i++ {
				n := names[(p+i)%len(names)]
				s := time.Now()
				r := w.call("GET", calls[n], nil, airtel)
				d := time.Since(s)
				mu.Lock()
				all[n] = append(all[n], d)
				if r.Code != 200 {
					t.Errorf("%s answered %d", n, r.Code)
				}
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	t.Logf("50 people x 4 calls finished in %s", time.Since(t0).Round(time.Millisecond))
	var worst time.Duration
	for _, n := range names {
		p95 := pct(all[n], 95)
		t.Logf("50 at once  %-22s p50 %-8s p95 %-8s", n, pct(all[n], 50), p95)
		if p95 > worst {
			worst = p95
		}
	}
	if worst > 3*time.Second {
		t.Fatalf("p95 %s is over the 3 s target", worst)
	}
	fmt.Println("worst p95:", worst)
}

func pct(d []time.Duration, p int) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := (len(s)*p + 99) / 100
	if i < 1 {
		i = 1
	}
	return s[i-1].Round(time.Millisecond)
}
