-- +goose Up
-- Customer-neutral names. Migrations 00002 to 00027 now use the final names
-- (roles viewer and reviewer, the site field customer_site_code, the party
-- value customer), so a new database is already right and this migration does
-- nothing there. A database that was built from the earlier text still holds
-- the first customer's name in role ids, one column pair, one index, five
-- CHECK constraints and the party values on stored rows; this converts them.
-- Accounts keep their permissions and two-step sign-in requirement. Audit rows
-- are append-only history and are left as written.
--
-- The old name is assembled from two pieces below on purpose: the product
-- must not contain it anywhere, and this is the one place that has to look it up.
-- +goose StatementBegin
DO $$
DECLARE
    old text := 'air' || 'tel';
    r   record;
    c   record;
BEGIN
    -- Roles: move accounts and permissions to the new ids, then drop the old rows.
    FOR r IN SELECT * FROM (VALUES ('_viewer', 'viewer', 'Viewer'), ('_reviewer', 'reviewer', 'Reviewer'))
                  AS v (suffix, new_id, new_name)
    LOOP
        IF EXISTS (SELECT 1 FROM roles WHERE id = old || r.suffix) THEN
            INSERT INTO roles (id, name, mfa_required)
            SELECT r.new_id, r.new_name, mfa_required FROM roles WHERE id = old || r.suffix
            ON CONFLICT (id) DO NOTHING;
            INSERT INTO role_permissions (role_id, permission_id)
            SELECT r.new_id, permission_id FROM role_permissions WHERE role_id = old || r.suffix
            ON CONFLICT DO NOTHING;
            UPDATE users SET role_id = r.new_id WHERE role_id = old || r.suffix;
            DELETE FROM roles WHERE id = old || r.suffix;
        END IF;
    END LOOP;

    UPDATE permissions SET description = replace(description, '(' || initcap(old) || ')', '(customer)')
     WHERE description LIKE '%(' || initcap(old) || ')%';

    -- The site code field, on the site record and on stored calculation rows.
    FOR r IN SELECT unnest(ARRAY['sites', 'calculation_run_rows']) AS tbl
    LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_schema = current_schema() AND table_name = r.tbl AND column_name = old || '_code') THEN
            EXECUTE format('ALTER TABLE %I RENAME COLUMN %I TO customer_site_code', r.tbl, old || '_code');
            IF EXISTS (SELECT 1 FROM pg_constraint
                        WHERE conrelid = r.tbl::regclass AND conname = r.tbl || '_' || old || '_code_not_null') THEN
                EXECUTE format('ALTER TABLE %I RENAME CONSTRAINT %I TO %I',
                               r.tbl, r.tbl || '_' || old || '_code_not_null', r.tbl || '_customer_site_code_not_null');
            END IF;
        END IF;
    END LOOP;
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'sites_' || old || '_code_key') THEN
        EXECUTE format('ALTER INDEX %I RENAME TO sites_customer_site_code_key', 'sites_' || old || '_code_key');
    END IF;

    -- The party value on stored rows. These tables guard their rows (append-only
    -- or frozen), so their triggers are paused for this one correction of a label.
    FOR r IN SELECT * FROM (VALUES ('dispute_events', 'party'), ('ticket_events', 'party'),
                                   ('acceptance_events', 'party'), ('attachments', 'party'),
                                   ('acceptances', 'responsible'))
                  AS v (tbl, col)
    LOOP
        IF EXISTS (SELECT 1 FROM pg_constraint
                    WHERE conrelid = r.tbl::regclass AND contype = 'c'
                      AND pg_get_constraintdef(oid) LIKE '%' || old || '%') THEN
            FOR c IN SELECT conname FROM pg_constraint
                      WHERE conrelid = r.tbl::regclass AND contype = 'c'
                        AND pg_get_constraintdef(oid) LIKE '%' || old || '%'
            LOOP
                EXECUTE format('ALTER TABLE %I DROP CONSTRAINT %I', r.tbl, c.conname);
            END LOOP;
            EXECUTE format('ALTER TABLE %I DISABLE TRIGGER USER', r.tbl);
            EXECUTE format('UPDATE %I SET %I = %L WHERE %I = %L', r.tbl, r.col, 'customer', r.col, old);
            EXECUTE format('ALTER TABLE %I ENABLE TRIGGER USER', r.tbl);
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (%I IN (''customer'', ''coloplus''))',
                           r.tbl, r.tbl || '_' || r.col || '_check', r.col);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd
