package postgres

import (
	"context"
	"fmt"
	"strings"

	"coloplus-backend/internal/domain"
	"coloplus-backend/internal/ports"

	"github.com/jackc/pgx/v5"
)

// SiteRepository implements ports.SiteRepository.
type SiteRepository struct{ db *DB }

func NewSiteRepository(db *DB) *SiteRepository { return &SiteRepository{db: db} }

const siteColumns = `id::text, airtel_code, coloplus_code, name, address, region, state, site_type, power_sources,
	service_start_minute, service_end_minute, freshness_hours, status, commissioning_date, latitude, longitude,
	created_at, updated_at`

func scanSite(row pgx.Row) (domain.Site, error) {
	var s domain.Site
	var sources []string
	var status string
	err := row.Scan(&s.ID, &s.AirtelCode, &s.ColoplusCode, &s.Name, &s.Address, &s.Region, &s.State, &s.SiteType, &sources,
		&s.ServiceStartMinute, &s.ServiceEndMinute, &s.FreshnessHours, &status, &s.CommissioningDate, &s.Latitude, &s.Longitude,
		&s.CreatedAt, &s.UpdatedAt)
	s.Status = domain.SiteStatus(status)
	for _, p := range sources {
		s.PowerSources = append(s.PowerSources, domain.PowerSource(p))
	}
	return s, err
}

func sourceStrings(in []domain.PowerSource) []string {
	out := make([]string, len(in))
	for i, p := range in {
		out[i] = string(p)
	}
	return out
}

const siteConflictMsg = "a site with this Airtel code or Coloplus code already exists"

func (r *SiteRepository) Create(ctx context.Context, s domain.Site) (domain.Site, error) {
	row := r.db.q(ctx).QueryRow(ctx,
		`INSERT INTO sites (airtel_code, coloplus_code, name, address, region, state, site_type, power_sources,
		        service_start_minute, service_end_minute, freshness_hours, status, commissioning_date, latitude, longitude)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING `+siteColumns,
		s.AirtelCode, s.ColoplusCode, s.Name, s.Address, s.Region, s.State, s.SiteType, sourceStrings(s.PowerSources),
		s.ServiceStartMinute, s.ServiceEndMinute, s.FreshnessHours, string(s.Status), s.CommissioningDate, s.Latitude, s.Longitude)
	created, err := scanSite(row)
	if err != nil {
		return domain.Site{}, mapErr(err, siteConflictMsg)
	}
	return created, nil
}

func (r *SiteRepository) GetByID(ctx context.Context, id string) (domain.Site, error) {
	s, err := scanSite(r.db.q(ctx).QueryRow(ctx, `SELECT `+siteColumns+` FROM sites WHERE id = $1`, id))
	return s, mapErr(err, "")
}

func (r *SiteRepository) Update(ctx context.Context, s domain.Site) error {
	_, err := r.db.q(ctx).Exec(ctx,
		`UPDATE sites SET airtel_code=$2, coloplus_code=$3, name=$4, address=$5, region=$6, state=$7, site_type=$8,
		        power_sources=$9, service_start_minute=$10, service_end_minute=$11, freshness_hours=$12, status=$13,
		        commissioning_date=$14, latitude=$15, longitude=$16, updated_at=now()
		 WHERE id = $1`,
		s.ID, s.AirtelCode, s.ColoplusCode, s.Name, s.Address, s.Region, s.State, s.SiteType, sourceStrings(s.PowerSources),
		s.ServiceStartMinute, s.ServiceEndMinute, s.FreshnessHours, string(s.Status), s.CommissioningDate, s.Latitude, s.Longitude)
	return mapErr(err, siteConflictMsg)
}

func (r *SiteRepository) List(ctx context.Context, f ports.SiteFilter) ([]domain.Site, int, error) {
	var where []string
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(clause, "?", fmt.Sprintf("$%d", len(args))))
	}

	if !f.AllSites {
		add("id = ANY(?::uuid[])", nonNil(f.SiteIDs))
	}
	if f.Query != "" {
		add("(airtel_code ILIKE ? OR coloplus_code ILIKE ? OR name ILIKE ?)", "%"+escapeLike(f.Query)+"%")
	}
	if f.State != "" {
		add("lower(state) = lower(?)", f.State)
	}
	if f.Region != "" {
		add("lower(region) = lower(?)", f.Region)
	}
	if f.Status != "" {
		add("status = ?", string(f.Status))
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	q := r.db.q(ctx)
	var total int
	if err := q.QueryRow(ctx, `SELECT count(*) FROM sites`+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapErr(err, "")
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := q.Query(ctx,
		`SELECT `+siteColumns+` FROM sites`+clause+
			fmt.Sprintf(` ORDER BY airtel_code LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, mapErr(err, "")
	}
	defer rows.Close()

	sites := []domain.Site{}
	for rows.Next() {
		s, err := scanSite(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning site: %w", err)
		}
		sites = append(sites, s)
	}
	return sites, total, rows.Err()
}

func (r *SiteRepository) GetMany(ctx context.Context, ids []string) ([]domain.Site, error) {
	rows, err := r.db.q(ctx).Query(ctx, `SELECT `+siteColumns+` FROM sites WHERE id = ANY($1::uuid[])`, nonNil(ids))
	if err != nil {
		return nil, mapErr(err, "")
	}
	defer rows.Close()

	var sites []domain.Site
	for rows.Next() {
		s, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

func (r *SiteRepository) CountExisting(ctx context.Context, ids []string) (int, error) {
	var n int
	err := r.db.q(ctx).QueryRow(ctx, `SELECT count(*) FROM sites WHERE id = ANY($1::uuid[])`, nonNil(ids)).Scan(&n)
	return n, mapErr(err, "")
}

// nonNil keeps a nil slice from being sent as SQL NULL.
func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
