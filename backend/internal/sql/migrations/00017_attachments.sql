-- +goose Up
-- Evidence files on tickets, incidents, disputes and site acceptance. The file
-- itself lives in a private R2 bucket; this row says what it is, who sent it,
-- who may see it, and its SHA-256. No permission rows are added: who may attach
-- or see a file follows who may act on, or see, the thing it is attached to.
CREATE TABLE attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id      UUID        NOT NULL REFERENCES sites (id),
    -- What it hangs on. For acceptance the owner is the site (one acceptance
    -- per site). Not a foreign key because the owner is in one of four tables.
    owner_type   TEXT        NOT NULL CHECK (owner_type IN ('ticket', 'incident', 'dispute', 'acceptance')),
    owner_id     UUID        NOT NULL,
    filename     TEXT        NOT NULL,
    content_type TEXT        NOT NULL,
    size_bytes   BIGINT      NOT NULL CHECK (size_bytes > 0),
    sha256       TEXT        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    -- A random key, never the file name: nothing a person typed is in a path.
    storage_key  TEXT        NOT NULL UNIQUE,
    status       TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'rejected', 'expired')),
    -- Coloplus-only evidence. The customer's own uploads are never internal.
    internal     BOOLEAN     NOT NULL DEFAULT FALSE,
    party        TEXT        NOT NULL CHECK (party IN ('customer', 'coloplus')),
    uploaded_by  UUID        NOT NULL REFERENCES users (id),
    -- Reserved for the virus scan (DECISIONS 016); nothing sets it yet.
    scan_status  TEXT        NOT NULL DEFAULT 'not_scanned' CHECK (scan_status IN ('not_scanned', 'clean', 'infected')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at TIMESTAMPTZ,
    CHECK (NOT internal OR party = 'coloplus'),
    CHECK (status <> 'ready' OR confirmed_at IS NOT NULL)
);

CREATE INDEX attachments_owner_idx   ON attachments (owner_type, owner_id) WHERE status = 'ready';
CREATE INDEX attachments_pending_idx ON attachments (created_at) WHERE status = 'pending';
CREATE INDEX attachments_user_pending_idx ON attachments (uploaded_by) WHERE status = 'pending';

-- Evidence is kept as it was received. A pending row becomes ready, rejected
-- or expired once; after that only the scan result may change.
-- +goose StatementBegin
CREATE FUNCTION attachments_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'attachments cannot be deleted';
    END IF;
    IF OLD.status <> 'pending' AND (
           NEW.status IS DISTINCT FROM OLD.status
        OR NEW.filename IS DISTINCT FROM OLD.filename
        OR NEW.size_bytes IS DISTINCT FROM OLD.size_bytes
        OR NEW.sha256 IS DISTINCT FROM OLD.sha256
        OR NEW.storage_key IS DISTINCT FROM OLD.storage_key
        OR NEW.internal IS DISTINCT FROM OLD.internal
        OR NEW.owner_id IS DISTINCT FROM OLD.owner_id) THEN
        RAISE EXCEPTION 'a received attachment cannot be changed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER attachments_guard
    BEFORE UPDATE OR DELETE ON attachments
    FOR EACH ROW EXECUTE FUNCTION attachments_guard();
