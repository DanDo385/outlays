-- +goose Up
-- App runtime role (Hard Rule 4, second layer): app_rw may SELECT and INSERT but never
-- UPDATE or DELETE. A login role for the app is granted app_rw out of band (see .env /
-- deploy), so no credentials live in migrations.

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_rw') THEN
    CREATE ROLE app_rw NOLOGIN;
  END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO app_rw;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA public TO app_rw;
REVOKE UPDATE, DELETE, TRUNCATE ON ALL TABLES IN SCHEMA public FROM app_rw;

-- Future tables created by the owner default to the same posture.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT ON TABLES TO app_rw;

-- +goose Down
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT ON TABLES FROM app_rw;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM app_rw;
REVOKE USAGE ON SCHEMA public FROM app_rw;
-- Role intentionally left in place (may own grants elsewhere); drop manually if required.
