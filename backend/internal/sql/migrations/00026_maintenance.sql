-- +goose Up
-- Planned maintenance. An approved window is taken out of a site's eligible
-- time, so it can only help the figures if a second person has agreed to it,
-- it was asked for in advance, and it stays within a monthly cap. A window is
-- never edited after it is decided; a mistake is cancelled and asked again.
CREATE TABLE maintenance_windows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq           BIGINT GENERATED ALWAYS AS IDENTITY,
    site_id       UUID        NOT NULL REFERENCES sites (id),
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    reason        TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'requested'
                  CHECK (status IN ('requested', 'approved', 'rejected', 'cancelled')),
    requested_by  UUID        NOT NULL REFERENCES users (id),
    requested_at  TIMESTAMPTZ NOT NULL,
    decided_by    UUID REFERENCES users (id),
    decided_at    TIMESTAMPTZ,
    decision_note TEXT        NOT NULL DEFAULT '',
    CHECK (ends_at > starts_at),
    CHECK ((status = 'requested') = (decided_at IS NULL)),
    CHECK (decided_by IS NULL OR status = 'cancelled' OR decided_by <> requested_by)
);

CREATE INDEX maintenance_site_time_idx ON maintenance_windows (site_id, starts_at);
CREATE INDEX maintenance_open_idx      ON maintenance_windows (seq DESC) WHERE status = 'requested';

INSERT INTO permissions (id, description) VALUES
    ('maintenance.read',    'See planned maintenance windows'),
    ('maintenance.request', 'Ask for a planned maintenance window'),
    ('maintenance.approve', 'Approve or reject a planned maintenance window');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin', 'maintenance.read'), ('admin', 'maintenance.request'), ('admin', 'maintenance.approve'),
    ('supervisor', 'maintenance.read'), ('supervisor', 'maintenance.request'), ('supervisor', 'maintenance.approve'),
    ('reporter', 'maintenance.read'), ('reporter', 'maintenance.request'),
    ('viewer', 'maintenance.read'), ('reviewer', 'maintenance.read'), ('auditor', 'maintenance.read');
