-- +goose Up
-- A register of every export that was issued: what it was, for which period,
-- by whom, how many rows, and the SHA-256 of the file. When storage is set up
-- the exact file is kept too, so "what did we send them?" has one answer and an
-- issued report is never regenerated into something different. Rows are never
-- changed or removed.
CREATE TABLE issued_exports (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq         BIGINT GENERATED ALWAYS AS IDENTITY,
    kind        TEXT        NOT NULL,
    filename    TEXT        NOT NULL,
    params      JSONB       NOT NULL DEFAULT '{}',
    row_count   INT         NOT NULL CHECK (row_count >= 0),
    size_bytes  BIGINT      NOT NULL CHECK (size_bytes >= 0),
    sha256      TEXT        NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    -- Where the file itself is kept; empty if storage was off or unreachable.
    storage_key TEXT        NOT NULL DEFAULT '',
    issued_by   UUID        NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX issued_exports_by_user_idx ON issued_exports (issued_by, seq DESC);
CREATE INDEX issued_exports_idx         ON issued_exports (seq DESC);

-- +goose StatementBegin
CREATE FUNCTION issued_exports_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'an issued export cannot be changed or removed';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER issued_exports_guard
    BEFORE UPDATE OR DELETE ON issued_exports
    FOR EACH ROW EXECUTE FUNCTION issued_exports_guard();
