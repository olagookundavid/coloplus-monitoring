-- +goose Up
-- A reporter-raised request to correct a single already-recorded incident
-- field (when a fault actually started, or when it was actually restored),
-- with the original value kept permanently alongside the corrected one.
-- No new permissions: requesting reuses incidents.create (the same
-- permission that lets a reporter record restoration), approving/rejecting
-- reuses incidents.manage, exactly like every other incident supervisor act.
CREATE TABLE incident_corrections (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number           BIGINT GENERATED ALWAYS AS IDENTITY,
    incident_id      UUID        NOT NULL REFERENCES incidents (id),
    -- Denormalized for site-scope filtering without joining incidents on
    -- every list call -- same convention as disputes.site_id.
    site_id          UUID        NOT NULL REFERENCES sites (id),
    field            TEXT        NOT NULL CHECK (field IN ('started_at', 'restored_at')),
    -- Snapshotted server-side from the incident row at request time, never
    -- taken from the client.
    original_value   TIMESTAMPTZ NOT NULL,
    corrected_value  TIMESTAMPTZ NOT NULL,
    reason           TEXT        NOT NULL,
    status           TEXT        NOT NULL DEFAULT 'requested' CHECK (status IN (
        'requested', 'approved', 'rejected')),
    requested_by     UUID        NOT NULL REFERENCES users (id),
    reviewed_by      UUID REFERENCES users (id),
    reviewed_at      TIMESTAMPTZ,
    review_note      TEXT        NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (corrected_value <> original_value),
    CHECK (status <> 'requested' OR (reviewed_by IS NULL AND reviewed_at IS NULL)),
    CHECK (status = 'requested' OR (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL))
);

CREATE INDEX incident_corrections_site_idx     ON incident_corrections (site_id, created_at DESC);
CREATE INDEX incident_corrections_incident_idx ON incident_corrections (incident_id);

-- One open request per incident+field at a time -- a retried or
-- double-clicked submit lands on the request that already exists.
CREATE UNIQUE INDEX incident_corrections_open_target_key ON incident_corrections (
    incident_id, field
) WHERE status = 'requested';

-- +goose StatementBegin
CREATE FUNCTION incident_corrections_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'corrections cannot be deleted';
    END IF;
    IF OLD.status <> 'requested' THEN
        RAISE EXCEPTION 'a decided correction cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER incident_corrections_guard
    BEFORE UPDATE OR DELETE ON incident_corrections
    FOR EACH ROW EXECUTE FUNCTION incident_corrections_guard();
