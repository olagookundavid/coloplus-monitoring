-- +goose Up
-- Coloplus management or auditor (PRD section 4): reads portfolio reports and
-- the audit history within assigned scope. Audit access is separate from
-- editing rights, so this role holds no create, review or manage permission.
-- It sees internal reports, so it needs two-step sign-in like staff do.
INSERT INTO roles (id, name, mfa_required) VALUES ('auditor', 'Coloplus management or auditor', TRUE);

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('auditor', 'sites.read'),
    ('auditor', 'reports.read'),
    ('auditor', 'audit.read');
