-- +goose Up
ALTER TABLE sites
    ADD COLUMN generator_capacity   TEXT NOT NULL DEFAULT '',
    ADD COLUMN access_contact_name  TEXT NOT NULL DEFAULT '',
    ADD COLUMN access_contact_phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN supervisor_name      TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sites
    DROP COLUMN generator_capacity,
    DROP COLUMN access_contact_name,
    DROP COLUMN access_contact_phone,
    DROP COLUMN supervisor_name;
