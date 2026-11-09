-- +goose Up
-- A stored calculation run freezes a period's figures for every site at one
-- moment, with the formula version and a fingerprint of the approved reports
-- they were worked out from. Later corrections do not change a run; they show
-- up when the run is compared with today's figures. Runs are never changed or
-- removed.
CREATE TABLE calculation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq             BIGINT GENERATED ALWAYS AS IDENTITY,
    kind            TEXT        NOT NULL,
    period_from     DATE        NOT NULL,
    period_to       DATE        NOT NULL,
    region          TEXT        NOT NULL DEFAULT '',
    label           TEXT        NOT NULL DEFAULT '',
    formula         TEXT        NOT NULL,
    formula_version TEXT        NOT NULL,
    site_count      INT         NOT NULL CHECK (site_count >= 0),
    created_by      UUID        NOT NULL REFERENCES users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (period_from <= period_to)
);

CREATE INDEX calculation_runs_idx ON calculation_runs (seq DESC);

CREATE TABLE calculation_run_rows (
    run_id                UUID             NOT NULL REFERENCES calculation_runs (id),
    site_id               UUID             NOT NULL REFERENCES sites (id),
    -- The site as it was named then, so the run reads the same later.
    customer_site_code           TEXT             NOT NULL,
    site_name             TEXT             NOT NULL,
    eligible_minutes      INT              NOT NULL,
    served_minutes        INT              NOT NULL,
    outage_minutes        INT              NOT NULL,
    unknown_minutes       INT              NOT NULL,
    assumed_minutes       INT              NOT NULL,
    days_in_period        INT              NOT NULL,
    days_reported         INT              NOT NULL,
    reported_availability DOUBLE PRECISION,
    confirmed_served_share DOUBLE PRECISION,
    coverage              DOUBLE PRECISION,
    generator_hours       DOUBLE PRECISION NOT NULL,
    incidents_started     INT              NOT NULL,
    -- SHA-256 over the ids of the approved reports the figures came from.
    fingerprint           TEXT             NOT NULL,
    PRIMARY KEY (run_id, site_id)
);

-- +goose StatementBegin
CREATE FUNCTION calculation_runs_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'a stored calculation run cannot be changed or removed';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER calculation_runs_guard
    BEFORE UPDATE OR DELETE ON calculation_runs
    FOR EACH ROW EXECUTE FUNCTION calculation_runs_guard();
CREATE TRIGGER calculation_run_rows_guard
    BEFORE UPDATE OR DELETE ON calculation_run_rows
    FOR EACH ROW EXECUTE FUNCTION calculation_runs_guard();
