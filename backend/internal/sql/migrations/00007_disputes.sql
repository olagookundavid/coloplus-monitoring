-- +goose Up
-- The Airtel reviewer is an Airtel viewer who may also raise disputes, add
-- review comments and acknowledge a resolution (PRD section 4).
INSERT INTO roles (id, name) VALUES ('airtel_reviewer', 'Airtel reviewer');

INSERT INTO permissions (id, description) VALUES
    ('disputes.read',   'Read disputes within scope'),
    ('disputes.raise',  'Raise disputes, comment, withdraw, reopen and acknowledge a resolution (Airtel)'),
    ('disputes.manage', 'Review, request evidence, agree a correction and resolve disputes (Coloplus)');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('airtel_reviewer', 'sites.read'),
    ('airtel_reviewer', 'published.read'),
    ('airtel_reviewer', 'disputes.read'),
    ('airtel_reviewer', 'disputes.raise'),
    ('airtel_viewer',   'disputes.read'),
    ('admin',           'disputes.read'),
    ('supervisor',      'disputes.read'),
    ('supervisor',      'disputes.manage');

CREATE TABLE disputes (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number               BIGINT GENERATED ALWAYS AS IDENTITY,
    site_id              UUID        NOT NULL REFERENCES sites (id),
    -- What is disputed: the site as a whole, a published report, or one
    -- interval of it. A report is the revision Airtel was looking at.
    report_id            UUID REFERENCES daily_reports (id),
    interval_id          UUID REFERENCES power_intervals (id),
    metric               TEXT        NOT NULL DEFAULT '' CHECK (metric IN (
        '', 'availability', 'coverage', 'outage_minutes', 'served_minutes', 'other')),
    period_from          DATE,
    period_to            DATE,
    category             TEXT        NOT NULL CHECK (category IN (
        'incorrect_outage', 'missing_outage', 'incorrect_served', 'wrong_duration',
        'wrong_metric', 'missing_report', 'other')),
    title                TEXT        NOT NULL,
    description          TEXT        NOT NULL DEFAULT '',
    status               TEXT        NOT NULL DEFAULT 'open' CHECK (status IN (
        'open', 'under_review', 'awaiting_evidence', 'agreed_correction', 'resolved', 'closed')),
    raised_by            UUID        NOT NULL REFERENCES users (id),
    owner_id             UUID REFERENCES users (id),
    -- Coloplus's side: how it was resolved, and by whom.
    outcome              TEXT        NOT NULL DEFAULT '' CHECK (outcome IN ('', 'upheld', 'not_upheld', 'withdrawn')),
    resolution           TEXT        NOT NULL DEFAULT '',
    resolution_report_id UUID REFERENCES daily_reports (id),
    resolved_at          TIMESTAMPTZ,
    resolved_by          UUID REFERENCES users (id),
    -- Airtel's side: its agreement is a separate act from Coloplus's approval.
    acknowledged_at      TIMESTAMPTZ,
    acknowledged_by      UUID REFERENCES users (id),
    reopen_count         INT         NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (interval_id IS NULL OR report_id IS NOT NULL),
    CHECK ((period_from IS NULL) = (period_to IS NULL)),
    CHECK (period_from IS NULL OR period_from <= period_to),
    CHECK (status <> 'resolved' OR (outcome IN ('upheld', 'not_upheld') AND resolved_at IS NOT NULL)),
    CHECK (status NOT IN ('open', 'under_review', 'awaiting_evidence', 'agreed_correction') OR outcome = ''),
    -- Closed means Airtel acknowledged the resolution, or withdrew the dispute.
    CHECK (status <> 'closed' OR outcome = 'withdrawn' OR (outcome IN ('upheld', 'not_upheld') AND acknowledged_at IS NOT NULL)),
    -- An upheld dispute about a report must end in a corrected revision.
    CHECK (NOT (outcome = 'upheld' AND report_id IS NOT NULL) OR resolution_report_id IS NOT NULL)
);

CREATE INDEX disputes_site_idx   ON disputes (site_id, created_at DESC);
CREATE INDEX disputes_report_idx ON disputes (report_id) WHERE report_id IS NOT NULL;
CREATE INDEX disputes_open_idx   ON disputes (status) WHERE status <> 'closed';

-- The same thing is not disputed twice at once: a retried or double-clicked
-- request lands on the dispute that already exists.
CREATE UNIQUE INDEX disputes_open_target_key ON disputes (
    site_id,
    COALESCE(report_id, '00000000-0000-0000-0000-000000000000'),
    COALESCE(interval_id, '00000000-0000-0000-0000-000000000000'),
    metric, category,
    COALESCE(period_from, DATE '1900-01-01'), COALESCE(period_to, DATE '1900-01-01')
) WHERE status <> 'closed';

-- Everything said or done on a dispute, from both parties. Internal entries
-- are Coloplus's own notes and never reach Airtel.
CREATE TABLE dispute_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Orders the history exactly, even for entries made in one transaction.
    seq         BIGINT GENERATED ALWAYS AS IDENTITY,
    dispute_id  UUID        NOT NULL REFERENCES disputes (id),
    author_id   UUID        NOT NULL REFERENCES users (id),
    party       TEXT        NOT NULL CHECK (party IN ('airtel', 'coloplus')),
    kind        TEXT        NOT NULL CHECK (kind IN ('comment', 'status')),
    from_status TEXT        NOT NULL DEFAULT '',
    to_status   TEXT        NOT NULL DEFAULT '',
    body        TEXT        NOT NULL DEFAULT '',
    internal    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NOT internal OR party = 'coloplus')
);

CREATE INDEX dispute_events_dispute_idx ON dispute_events (dispute_id, seq);

-- +goose StatementBegin
CREATE FUNCTION disputes_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'disputes cannot be deleted';
    END IF;
    IF OLD.status = 'closed' THEN
        RAISE EXCEPTION 'a closed dispute cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER disputes_guard
    BEFORE UPDATE OR DELETE ON disputes
    FOR EACH ROW EXECUTE FUNCTION disputes_guard();

-- +goose StatementBegin
CREATE FUNCTION dispute_events_guard() RETURNS trigger AS $$
DECLARE
    parent_status TEXT;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'dispute history is append-only';
    END IF;
    SELECT status INTO parent_status FROM disputes WHERE id = NEW.dispute_id;
    IF parent_status = 'closed' THEN
        RAISE EXCEPTION 'a closed dispute cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER dispute_events_guard
    BEFORE INSERT OR UPDATE OR DELETE ON dispute_events
    FOR EACH ROW EXECUTE FUNCTION dispute_events_guard();
