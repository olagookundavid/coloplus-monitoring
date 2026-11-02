-- +goose Up
-- Fuel the manual way: each site has an agreed consumption rate (litres per
-- hour that its generator runs) and, optionally, a tank size. Engineers record
-- deliveries and tank readings; the system estimates what is left from the
-- generator hours reported each day. An estimate is always labelled as one.
ALTER TABLE sites
    ADD COLUMN consumption_lph      NUMERIC(8,2)  CHECK (consumption_lph IS NULL OR consumption_lph > 0),
    ADD COLUMN tank_capacity_litres NUMERIC(10,1) CHECK (tank_capacity_litres IS NULL OR tank_capacity_litres > 0);

INSERT INTO permissions (id, description) VALUES
    ('fuel.read',   'See a site''s fuel entries and estimated level within scope'),
    ('fuel.record', 'Record fuel deliveries and tank readings for sites in scope'),
    ('fuel.manage', 'Void a mistaken fuel entry, with a reason');

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('admin',           'fuel.read'),
    ('reporter',        'fuel.read'),
    ('reporter',        'fuel.record'),
    ('supervisor',      'fuel.read'),
    ('supervisor',      'fuel.record'),
    ('supervisor',      'fuel.manage'),
    ('airtel_viewer',   'fuel.read'),
    ('airtel_reviewer', 'fuel.read');

-- What came into the tank, or what the tank held when someone looked.
CREATE TABLE fuel_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seq         BIGINT GENERATED ALWAYS AS IDENTITY,
    site_id     UUID          NOT NULL REFERENCES sites (id),
    kind        TEXT          NOT NULL CHECK (kind IN ('delivery', 'reading')),
    litres      NUMERIC(10,1) NOT NULL CHECK (litres >= 0 AND (kind <> 'delivery' OR litres > 0)),
    occurred_at TIMESTAMPTZ   NOT NULL,
    -- Visible to Coloplus only.
    note        TEXT          NOT NULL DEFAULT '',
    recorded_by UUID          NOT NULL REFERENCES users (id),
    -- A retried request (a phone that lost its connection) lands on the entry
    -- it already made.
    client_key  TEXT          NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    voided_at   TIMESTAMPTZ,
    voided_by   UUID REFERENCES users (id),
    void_reason TEXT          NOT NULL DEFAULT '',
    CHECK ((voided_at IS NULL) = (voided_by IS NULL)),
    CHECK (voided_at IS NULL OR void_reason <> '')
);

CREATE INDEX fuel_entries_site_idx ON fuel_entries (site_id, occurred_at DESC, seq DESC);
CREATE UNIQUE INDEX fuel_entries_client_key_key ON fuel_entries (recorded_by, client_key) WHERE client_key <> '';

-- A fuel entry is a record of what was said. It can be voided once, with a
-- reason, and never edited or deleted.
-- +goose StatementBegin
CREATE FUNCTION fuel_entries_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'fuel entries cannot be deleted';
    END IF;
    IF OLD.voided_at IS NOT NULL THEN
        RAISE EXCEPTION 'a voided fuel entry cannot be changed';
    END IF;
    IF NEW.site_id IS DISTINCT FROM OLD.site_id OR NEW.kind IS DISTINCT FROM OLD.kind
       OR NEW.litres IS DISTINCT FROM OLD.litres OR NEW.occurred_at IS DISTINCT FROM OLD.occurred_at
       OR NEW.note IS DISTINCT FROM OLD.note OR NEW.recorded_by IS DISTINCT FROM OLD.recorded_by THEN
        RAISE EXCEPTION 'a fuel entry cannot be edited; void it and record a new one';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER fuel_entries_guard
    BEFORE UPDATE OR DELETE ON fuel_entries
    FOR EACH ROW EXECUTE FUNCTION fuel_entries_guard();
