-- +goose Up
-- Two-step sign-in is required by role (PRD section 11): Coloplus's
-- administrators and supervisors, and every Airtel user. Code asks the role,
-- never its name.
ALTER TABLE roles ADD COLUMN mfa_required BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE roles SET mfa_required = TRUE
 WHERE id IN ('admin', 'supervisor', 'airtel_viewer', 'airtel_reviewer');

-- One-time codes for a lost phone. Only a hash is stored, and each code works
-- once.
CREATE TABLE mfa_recovery_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at    TIMESTAMPTZ,
    UNIQUE (user_id, code_hash)
);
