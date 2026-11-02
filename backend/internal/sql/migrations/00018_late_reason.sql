-- +goose Up
-- A report submitted after its site would already show "Awaiting update" has to
-- say why. Kept with the report so the approver sees it; it is set when the
-- report is submitted, before approval locks the row.
ALTER TABLE daily_reports ADD COLUMN late_reason TEXT NOT NULL DEFAULT '';
