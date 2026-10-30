-- +goose Up
-- Exporting is its own right: reading a screen does not imply taking a copy
-- of the figures out of the system.
INSERT INTO permissions (id, description) VALUES
    ('exports.create', 'Export figures with their provenance (CSV)');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin',           'exports.create'),
    ('supervisor',      'exports.create'),
    ('airtel_viewer',   'exports.create'),
    ('airtel_reviewer', 'exports.create');
