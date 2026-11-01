-- Local development only. Keep the database and role; remove all public tables.
DO $$
BEGIN
    IF current_database() <> 'cloudmusic_dev' THEN
        RAISE EXCEPTION 'Reset is only allowed for cloudmusic_dev';
    END IF;
END;
$$;

DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

-- CREATE TABLE lives in apps/server/migrations/00001_placeholder.sql.
