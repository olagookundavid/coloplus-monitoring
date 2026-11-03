-- +goose Up
-- Closing a month locks it: no new report, correction or approval for any day in
-- it until an administrator reopens it and says why. Closing stores the month's
-- calculation run, so what was closed is on record. A month can be closed again
-- after being reopened, which makes a new row; the audit log keeps the history.
CREATE TABLE period_closures (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq           BIGINT GENERATED ALWAYS AS IDENTITY,
    month         DATE        NOT NULL CHECK (date_trunc('month', month) = month),
    closed_by     UUID        NOT NULL REFERENCES users (id),
    closed_at     TIMESTAMPTZ NOT NULL,
    run_id        UUID        NOT NULL REFERENCES calculation_runs (id),
    reopened_by   UUID REFERENCES users (id),
    reopened_at   TIMESTAMPTZ,
    reopen_reason TEXT        NOT NULL DEFAULT '',
    CHECK ((reopened_at IS NULL) = (reopened_by IS NULL))
);

-- Only one open closure per month.
CREATE UNIQUE INDEX period_closures_open_month_key ON period_closures (month) WHERE reopened_at IS NULL;
CREATE INDEX period_closures_idx ON period_closures (seq DESC);

INSERT INTO permissions (id, description) VALUES
    ('periods.manage', 'Close and reopen a reporting month');

INSERT INTO role_permissions (role_id, permission_id) VALUES ('admin', 'periods.manage');
