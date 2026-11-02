-- +goose Up
-- How a site's time is counted when nobody reported it. Owner direction: a site
-- is available by default and is only made unavailable when an engineer or the
-- site's owner says so, with a reason. 'unknown' keeps the stricter reading,
-- where unreported time is neither up nor down, for any site or customer whose
-- terms require it.
ALTER TABLE sites
    ADD COLUMN unreported_time TEXT NOT NULL DEFAULT 'available' CHECK (unreported_time IN ('available', 'unknown'));

-- Why the site was unavailable that day. Required when a report with an outage
-- is submitted; reports approved before this existed simply have none.
ALTER TABLE daily_reports ADD COLUMN outage_reason TEXT NOT NULL DEFAULT '';
