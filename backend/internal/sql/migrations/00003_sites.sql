-- +goose Up
CREATE TABLE sites (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    airtel_code          TEXT             NOT NULL,
    coloplus_code        TEXT             NOT NULL,
    name                 TEXT             NOT NULL,
    address              TEXT             NOT NULL DEFAULT '',
    region               TEXT             NOT NULL DEFAULT '',
    state                TEXT             NOT NULL,
    site_type            TEXT             NOT NULL DEFAULT '',
    power_sources        TEXT[]           NOT NULL,
    service_start_minute INT              NOT NULL DEFAULT 0,
    service_end_minute   INT              NOT NULL DEFAULT 1440,
    freshness_hours      INT              NOT NULL DEFAULT 30,
    status               TEXT             NOT NULL DEFAULT 'pending_acceptance'
                         CHECK (status IN ('pending_acceptance', 'active', 'suspended')),
    commissioning_date   DATE,
    latitude             DOUBLE PRECISION,
    longitude            DOUBLE PRECISION,
    created_at           TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ      NOT NULL DEFAULT now(),
    CHECK (service_start_minute >= 0 AND service_end_minute <= 1440
           AND service_start_minute < service_end_minute),
    CHECK ((latitude IS NULL) = (longitude IS NULL))
);

-- Duplicate Airtel site codes are rejected across the portfolio.
CREATE UNIQUE INDEX sites_airtel_code_key ON sites (lower(airtel_code));
CREATE UNIQUE INDEX sites_coloplus_code_key ON sites (lower(coloplus_code));
CREATE INDEX sites_state_idx ON sites (state);
CREATE INDEX sites_status_idx ON sites (status);

CREATE TABLE user_sites (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, site_id)
);

CREATE INDEX user_sites_site_idx ON user_sites (site_id);
