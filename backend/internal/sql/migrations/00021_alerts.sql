-- +goose Up
-- The in-app alert centre: every alert that goes out by email to a person also
-- waits for them here, until they acknowledge it. Alerts are about the work
-- (a late report, an overdue incident, low fuel, a reply on a ticket) and carry
-- no one-time links; account emails (invitations, resets) never come here.
CREATE TABLE alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq             BIGINT GENERATED ALWAYS AS IDENTITY,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind            TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    body            TEXT        NOT NULL DEFAULT '',
    -- The same reason to alert the same person is only ever recorded once.
    dedupe_key      TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    UNIQUE (user_id, dedupe_key)
);

CREATE INDEX alerts_user_open_idx ON alerts (user_id, seq DESC) WHERE acknowledged_at IS NULL;
CREATE INDEX alerts_user_idx      ON alerts (user_id, seq DESC);

INSERT INTO permissions (id, description) VALUES
    ('alerts.read', 'See and acknowledge your own alerts');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin', 'alerts.read'), ('reporter', 'alerts.read'), ('supervisor', 'alerts.read'),
    ('viewer', 'alerts.read'), ('reviewer', 'alerts.read'), ('auditor', 'alerts.read');
