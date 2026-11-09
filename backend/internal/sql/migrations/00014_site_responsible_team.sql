-- +goose Up
-- Who is accountable for the site (PRD section 8.1, "responsible team"). A team
-- name, not a person: The customer sees it, staff names stay inside Coloplus.
ALTER TABLE sites ADD COLUMN responsible_team TEXT NOT NULL DEFAULT '';

