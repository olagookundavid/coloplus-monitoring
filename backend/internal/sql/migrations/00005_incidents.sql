-- +goose Up
INSERT INTO permissions (id, description) VALUES
    ('incidents.create', 'Report incidents, add notes and mark restoration for sites in scope'),
    ('incidents.read',   'Read internal incident records within scope'),
    ('incidents.manage', 'Acknowledge, assign, verify, close and publish incidents');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin',      'incidents.read'),
    ('reporter',   'incidents.create'),
    ('reporter',   'incidents.read'),
    ('supervisor', 'incidents.create'),
    ('supervisor', 'incidents.read'),
    ('supervisor', 'incidents.manage');

CREATE TABLE incidents (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number             BIGINT GENERATED ALWAYS AS IDENTITY,
    site_id            UUID        NOT NULL REFERENCES sites (id),
    category           TEXT        NOT NULL CHECK (category IN (
        'grid_loss', 'building_supply_loss', 'generator_failure', 'low_fuel', 'battery_failure',
        'rectifier_fault', 'electrical_fault', 'access_restriction', 'other')),
    severity           TEXT        NOT NULL CHECK (severity IN ('critical', 'major', 'minor')),
    -- A fault with no outage is recorded, but kept apart from service interruptions.
    impact             TEXT        NOT NULL CHECK (impact IN ('outage', 'no_outage')),
    title              TEXT        NOT NULL,
    description        TEXT        NOT NULL DEFAULT '',
    status             TEXT        NOT NULL DEFAULT 'reported' CHECK (status IN (
        'reported', 'acknowledged', 'assigned', 'in_progress', 'restored', 'verified', 'closed')),
    started_at         TIMESTAMPTZ NOT NULL,
    detected_at        TIMESTAMPTZ NOT NULL,
    reporter_id        UUID        NOT NULL REFERENCES users (id),
    owner_id           UUID REFERENCES users (id),
    next_action        TEXT        NOT NULL DEFAULT '',
    next_update_due    TIMESTAMPTZ,
    -- A target is a plan. Only restored_at, an observed time, counts as restoration.
    restoration_target TIMESTAMPTZ,
    restored_at        TIMESTAMPTZ,
    restored_by        UUID REFERENCES users (id),
    verified_at        TIMESTAMPTZ,
    verified_by        UUID REFERENCES users (id),
    closed_at          TIMESTAMPTZ,
    root_cause         TEXT        NOT NULL DEFAULT '',
    published_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (detected_at >= started_at),
    CHECK (status NOT IN ('restored', 'verified', 'closed') OR restored_at IS NOT NULL),
    CHECK (status NOT IN ('verified', 'closed') OR verified_at IS NOT NULL),
    CHECK (status <> 'closed' OR root_cause <> ''),
    -- Whoever marks restoration is not the person who verifies it.
    CHECK (verified_by IS NULL OR restored_by IS NULL OR verified_by <> restored_by)
);

CREATE INDEX incidents_site_idx   ON incidents (site_id, started_at DESC);
CREATE INDEX incidents_open_idx   ON incidents (status) WHERE status <> 'closed';
CREATE INDEX incidents_owner_idx  ON incidents (owner_id) WHERE owner_id IS NOT NULL;

-- Update history: every status change, note, assignment and customer update.
CREATE TABLE incident_updates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id      UUID        NOT NULL REFERENCES incidents (id),
    author_id        UUID        NOT NULL REFERENCES users (id),
    kind             TEXT        NOT NULL CHECK (kind IN ('note', 'status', 'assignment', 'plan', 'customer')),
    from_status      TEXT        NOT NULL DEFAULT '',
    to_status        TEXT        NOT NULL DEFAULT '',
    body             TEXT        NOT NULL DEFAULT '',
    customer_visible BOOLEAN     NOT NULL DEFAULT FALSE,
    -- When a restoration or verification was observed, as opposed to recorded.
    observed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX incident_updates_incident_idx ON incident_updates (incident_id, created_at);

-- +goose StatementBegin
CREATE FUNCTION incidents_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'incidents cannot be deleted';
    END IF;
    IF OLD.status = 'closed' THEN
        RAISE EXCEPTION 'a closed incident cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER incidents_guard
    BEFORE UPDATE OR DELETE ON incidents
    FOR EACH ROW EXECUTE FUNCTION incidents_guard();

-- +goose StatementBegin
CREATE FUNCTION incident_updates_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'incident history is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER incident_updates_append_only
    BEFORE UPDATE OR DELETE ON incident_updates
    FOR EACH ROW EXECUTE FUNCTION incident_updates_append_only();
