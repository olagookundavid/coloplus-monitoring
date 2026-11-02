-- +goose Up
ALTER TABLE daily_reports
    ADD COLUMN generator_hours NUMERIC(4,2)
    CHECK (generator_hours IS NULL OR generator_hours BETWEEN 0 AND 24);

-- +goose Down
ALTER TABLE daily_reports DROP COLUMN generator_hours;
