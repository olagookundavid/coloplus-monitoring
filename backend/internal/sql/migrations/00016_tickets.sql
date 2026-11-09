-- +goose Up
-- Support tickets: a request the customer raises about a site (a fault, a refill, an
-- access problem) and Coloplus works. Not a fault record: the incident register
-- stays Coloplus's own, and a ticket can be linked to the incident it became.
INSERT INTO permissions (id, description) VALUES
    ('tickets.read',   'Read support tickets within scope'),
    ('tickets.raise',  'Raise tickets, comment and reopen a resolved ticket (customer)'),
    ('tickets.manage', 'Work tickets: start, resolve, comment, add internal notes, link an incident (Coloplus)');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('viewer',   'tickets.read'),
    ('viewer',   'tickets.raise'),
    ('reviewer', 'tickets.read'),
    ('reviewer', 'tickets.raise'),
    ('admin',           'tickets.read'),
    ('supervisor',      'tickets.read'),
    ('supervisor',      'tickets.manage');

CREATE TABLE tickets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number        BIGINT GENERATED ALWAYS AS IDENTITY,
    site_id       UUID        NOT NULL REFERENCES sites (id),
    category      TEXT        NOT NULL CHECK (category IN (
        'power_outage', 'low_fuel', 'generator_fault', 'equipment_fault', 'access_restriction', 'other')),
    -- The priority the customer asked for. It is the customer's word, not Coloplus's
    -- severity; severity belongs to the incident.
    priority      TEXT        NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    subject       TEXT        NOT NULL,
    description   TEXT        NOT NULL,
    -- Where to reach the requester. Shown to Coloplus; never emailed to.
    contact_email TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved')),
    raised_by     UUID        NOT NULL REFERENCES users (id),
    assignee_id   UUID REFERENCES users (id),
    incident_id   UUID REFERENCES incidents (id),
    resolution    TEXT        NOT NULL DEFAULT '',
    resolved_at   TIMESTAMPTZ,
    resolved_by   UUID REFERENCES users (id),
    reopen_count  INT         NOT NULL DEFAULT 0,
    -- A retried request (a double click, a dropped connection) lands on the
    -- ticket it already created.
    client_key    TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status <> 'resolved' OR (resolved_at IS NOT NULL AND resolution <> ''))
);

CREATE INDEX tickets_site_idx   ON tickets (site_id, created_at DESC);
CREATE INDEX tickets_status_idx ON tickets (status) WHERE status <> 'resolved';
CREATE UNIQUE INDEX tickets_client_key_key ON tickets (raised_by, client_key) WHERE client_key <> '';

-- Everything said or done on a ticket, from both sides. Internal entries are
-- Coloplus's own notes and never reach the customer.
CREATE TABLE ticket_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq         BIGINT GENERATED ALWAYS AS IDENTITY,
    ticket_id   UUID        NOT NULL REFERENCES tickets (id),
    author_id   UUID        NOT NULL REFERENCES users (id),
    party       TEXT        NOT NULL CHECK (party IN ('customer', 'coloplus')),
    kind        TEXT        NOT NULL CHECK (kind IN ('comment', 'status', 'link')),
    from_status TEXT        NOT NULL DEFAULT '',
    to_status   TEXT        NOT NULL DEFAULT '',
    body        TEXT        NOT NULL DEFAULT '',
    internal    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (NOT internal OR party = 'coloplus')
);

CREATE INDEX ticket_events_ticket_idx ON ticket_events (ticket_id, seq);

-- +goose StatementBegin
CREATE FUNCTION tickets_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'tickets cannot be deleted';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER tickets_guard
    BEFORE DELETE ON tickets
    FOR EACH ROW EXECUTE FUNCTION tickets_guard();

-- +goose StatementBegin
CREATE FUNCTION ticket_events_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ticket history is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER ticket_events_guard
    BEFORE UPDATE OR DELETE ON ticket_events
    FOR EACH ROW EXECUTE FUNCTION ticket_events_guard();
