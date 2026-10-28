-- +goose Up
CREATE TABLE daily_reports (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id           UUID        NOT NULL REFERENCES sites (id),
    report_date       DATE        NOT NULL,
    revision          INT         NOT NULL DEFAULT 1,
    status            TEXT        NOT NULL DEFAULT 'draft'
                      CHECK (status IN ('draft', 'submitted', 'approved', 'rejected', 'superseded')),
    author_id         UUID        NOT NULL REFERENCES users (id),
    notes             TEXT        NOT NULL DEFAULT '',
    submitted_at      TIMESTAMPTZ,
    reviewed_at       TIMESTAMPTZ,
    reviewer_id       UUID REFERENCES users (id),
    review_comment    TEXT        NOT NULL DEFAULT '',
    supersedes_id     UUID REFERENCES daily_reports (id),
    correction_reason TEXT        NOT NULL DEFAULT '',
    row_version       INT         NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (site_id, report_date, revision),
    -- The approver is never the author.
    CHECK (reviewer_id IS NULL OR reviewer_id <> author_id)
);

-- One report in progress, and one live approved report, per site and day.
CREATE UNIQUE INDEX daily_reports_in_progress_key
    ON daily_reports (site_id, report_date) WHERE status IN ('draft', 'submitted', 'rejected');
CREATE UNIQUE INDEX daily_reports_approved_key
    ON daily_reports (site_id, report_date) WHERE status = 'approved';
CREATE INDEX daily_reports_site_date_idx ON daily_reports (site_id, report_date DESC);
CREATE INDEX daily_reports_status_idx ON daily_reports (status);

CREATE TABLE power_intervals (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID        NOT NULL REFERENCES daily_reports (id) ON DELETE CASCADE,
    start_at  TIMESTAMPTZ NOT NULL,
    end_at    TIMESTAMPTZ NOT NULL,
    state     TEXT        NOT NULL CHECK (state IN ('served', 'outage')),
    source    TEXT        NOT NULL DEFAULT '',
    basis     TEXT        NOT NULL DEFAULT 'observed' CHECK (basis IN ('observed', 'declared')),
    CHECK (end_at > start_at),
    CHECK ((state = 'served') = (source <> ''))
);

CREATE INDEX power_intervals_report_idx ON power_intervals (report_id);

-- An approved record is immutable. The only change it may undergo is being
-- superseded by an approved correction. Superseded records never change.
-- +goose StatementBegin
CREATE FUNCTION daily_reports_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status IN ('approved', 'superseded') THEN
            RAISE EXCEPTION 'approved reports cannot be deleted';
        END IF;
        RETURN OLD;
    END IF;

    IF OLD.status = 'superseded' THEN
        RAISE EXCEPTION 'superseded reports cannot be changed';
    END IF;
    IF OLD.status = 'approved' AND NEW.status <> 'superseded' THEN
        RAISE EXCEPTION 'approved reports cannot be changed; create a correction instead';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER daily_reports_guard
    BEFORE UPDATE OR DELETE ON daily_reports
    FOR EACH ROW EXECUTE FUNCTION daily_reports_guard();

-- +goose StatementBegin
CREATE FUNCTION power_intervals_guard() RETURNS trigger AS $$
DECLARE
    parent_status TEXT;
    parent_id     UUID;
BEGIN
    IF TG_OP = 'INSERT' THEN
        parent_id := NEW.report_id;
    ELSE
        parent_id := OLD.report_id;
    END IF;

    SELECT status INTO parent_status FROM daily_reports WHERE id = parent_id;
    IF parent_status IN ('approved', 'superseded') THEN
        RAISE EXCEPTION 'intervals of an approved report cannot be changed';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER power_intervals_guard
    BEFORE INSERT OR UPDATE OR DELETE ON power_intervals
    FOR EACH ROW EXECUTE FUNCTION power_intervals_guard();
