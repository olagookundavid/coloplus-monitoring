-- +goose Up
CREATE TABLE roles (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE permissions (
    id          TEXT PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE role_permissions (
    role_id       TEXT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

INSERT INTO roles (id, name) VALUES
    ('admin',         'Coloplus administrator'),
    ('reporter',      'Field officer or site reporter'),
    ('supervisor',    'Operations supervisor'),
    ('airtel_viewer', 'Airtel viewer');

INSERT INTO permissions (id, description) VALUES
    ('users.manage',    'Create users, change roles and site assignments'),
    ('sites.all',       'Access every site regardless of assignment'),
    ('sites.read',      'View sites within scope'),
    ('sites.manage',    'Create and edit site records'),
    ('reports.create',  'Create, edit and submit daily reports for assigned sites'),
    ('reports.read',    'Read internal daily reports within scope'),
    ('reports.review',  'Approve or return submitted daily reports'),
    ('published.read',  'Read approved, published records only'),
    ('audit.read',      'Search the audit log');

-- Administrators manage but cannot approve (they hold no reports.review), so
-- nobody can approve their own submissions by also being an admin.
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin',         'users.manage'),
    ('admin',         'sites.all'),
    ('admin',         'sites.read'),
    ('admin',         'sites.manage'),
    ('admin',         'reports.read'),
    ('admin',         'audit.read'),
    ('reporter',      'sites.read'),
    ('reporter',      'reports.create'),
    ('reporter',      'reports.read'),
    ('supervisor',    'sites.read'),
    ('supervisor',    'reports.read'),
    ('supervisor',    'reports.review'),
    ('airtel_viewer', 'sites.read'),
    ('airtel_viewer', 'published.read');

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    role_id       TEXT        NOT NULL REFERENCES roles (id),
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    totp_secret   TEXT        NOT NULL DEFAULT '',
    totp_enabled  BOOLEAN     NOT NULL DEFAULT FALSE,
    failed_logins INT         NOT NULL DEFAULT 0,
    locked_until  TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_key ON users (lower(email));
