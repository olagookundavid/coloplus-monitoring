-- +goose Up
INSERT INTO permissions (id, description) VALUES
    ('acceptance.read',   'Read commissioning and acceptance records within scope'),
    ('acceptance.manage', 'Request, build, submit a dossier for, hold and resume a site acceptance (Coloplus)'),
    ('acceptance.review', 'Record inspection, acceptance or rejection of a site (Airtel)');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin',           'acceptance.read'),
    ('reporter',        'acceptance.read'),
    ('supervisor',      'acceptance.read'),
    ('supervisor',      'acceptance.manage'),
    ('airtel_viewer',   'acceptance.read'),
    ('airtel_reviewer', 'acceptance.read'),
    ('airtel_reviewer', 'acceptance.review');

-- One acceptance process per site. Its stage is kept apart from the site's
-- operational record: nothing Coloplus does counts as Airtel's acceptance.
CREATE TABLE acceptances (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id      UUID        NOT NULL UNIQUE REFERENCES sites (id),
    stage        TEXT        NOT NULL DEFAULT 'requested' CHECK (stage IN (
        'requested', 'built', 'ready_for_inspection', 'inspected', 'accepted', 'rejected', 'on_hold')),
    -- The stage a hold will return to.
    held_from    TEXT        NOT NULL DEFAULT '',
    -- Whose move it is.
    responsible  TEXT        NOT NULL DEFAULT 'coloplus' CHECK (responsible IN ('coloplus', 'airtel')),
    next_action  TEXT        NOT NULL DEFAULT '',
    due_at       TIMESTAMPTZ,
    -- Coloplus's submission when it declares the site ready for inspection.
    dossier      TEXT        NOT NULL DEFAULT '',
    -- Why it was rejected, and why it is on hold.
    reason       TEXT        NOT NULL DEFAULT '',
    hold_reason  TEXT        NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL,
    built_at     TIMESTAMPTZ,
    ready_at     TIMESTAMPTZ,
    inspected_at TIMESTAMPTZ,
    decided_at   TIMESTAMPTZ,
    decided_by   UUID REFERENCES users (id),
    created_by   UUID        NOT NULL REFERENCES users (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (stage <> 'accepted' OR (decided_at IS NOT NULL AND decided_by IS NOT NULL)),
    CHECK (stage <> 'rejected' OR reason <> ''),
    CHECK (stage <> 'on_hold' OR (held_from <> '' AND hold_reason <> '')),
    CHECK (stage NOT IN ('ready_for_inspection', 'inspected') OR (ready_at IS NOT NULL AND dossier <> ''))
);

CREATE INDEX acceptances_open_idx ON acceptances (stage) WHERE stage <> 'accepted';

CREATE TABLE acceptance_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq           BIGINT GENERATED ALWAYS AS IDENTITY,
    acceptance_id UUID        NOT NULL REFERENCES acceptances (id),
    author_id     UUID        NOT NULL REFERENCES users (id),
    party         TEXT        NOT NULL CHECK (party IN ('airtel', 'coloplus')),
    from_stage    TEXT        NOT NULL DEFAULT '',
    to_stage      TEXT        NOT NULL,
    body          TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX acceptance_events_idx ON acceptance_events (acceptance_id, seq);

-- +goose StatementBegin
CREATE FUNCTION acceptances_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'acceptance records cannot be deleted';
    END IF;
    IF OLD.stage = 'accepted' THEN
        RAISE EXCEPTION 'an accepted site cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER acceptances_guard
    BEFORE UPDATE OR DELETE ON acceptances
    FOR EACH ROW EXECUTE FUNCTION acceptances_guard();

-- +goose StatementBegin
CREATE FUNCTION acceptance_events_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'acceptance history is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER acceptance_events_append_only
    BEFORE UPDATE OR DELETE ON acceptance_events
    FOR EACH ROW EXECUTE FUNCTION acceptance_events_append_only();
