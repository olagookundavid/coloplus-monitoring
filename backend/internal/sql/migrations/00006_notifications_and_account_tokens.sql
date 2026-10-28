-- +goose Up
-- A password change cuts off every session issued before it.
ALTER TABLE users ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

INSERT INTO permissions (id, description) VALUES
    ('notifications.read', 'See email delivery status and retry failed emails');
INSERT INTO role_permissions (role_id, permission_id) VALUES ('admin', 'notifications.read');

-- One-time links for choosing a password (invitation) or recovering access
-- (reset). Only a hash of the token is stored.
CREATE TABLE user_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT        NOT NULL CHECK (purpose IN ('invite', 'reset')),
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_tokens_user_idx ON user_tokens (user_id, purpose, created_at DESC);

-- The outbox. A business change and the email it implies are written in the
-- same transaction; a dispatcher sends afterwards, so no third party is
-- called while a transaction is open and a provider outage loses nothing.
CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind            TEXT        NOT NULL,
    recipient       TEXT        NOT NULL,
    subject         TEXT        NOT NULL,
    -- Cleared once the email is sent, because it can hold a one-time link.
    body            TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    attempts        INT         NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error      TEXT        NOT NULL DEFAULT '',
    provider_id     TEXT        NOT NULL DEFAULT '',
    -- The same reason to email the same person is only ever queued once.
    dedupe_key      TEXT        NOT NULL UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ
);

CREATE INDEX notifications_due_idx ON notifications (next_attempt_at) WHERE status = 'pending';
CREATE INDEX notifications_status_idx ON notifications (status, created_at DESC);
