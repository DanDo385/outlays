-- +goose Up
-- Append-only enforcement (Hard Rule 4): a BEFORE UPDATE OR DELETE trigger on every data
-- table that raises. Corrections are new rows (fiscal_fact.supersedes,
-- classification_assignment.version). History is evidence. The REVOKE layer for the app role
-- is in 00003_roles.sql.

-- +goose StatementBegin
CREATE FUNCTION reject_mutation() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'append-only: % on table % is not permitted (corrections are new rows)',
    TG_OP, TG_TABLE_NAME
    USING ERRCODE = 'restrict_violation';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER ingestion_run_append_only BEFORE UPDATE OR DELETE ON ingestion_run
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER raw_snapshot_append_only BEFORE UPDATE OR DELETE ON raw_snapshot
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER entity_append_only BEFORE UPDATE OR DELETE ON entity
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER entity_alias_append_only BEFORE UPDATE OR DELETE ON entity_alias
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER fiscal_fact_append_only BEFORE UPDATE OR DELETE ON fiscal_fact
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER classification_scheme_append_only BEFORE UPDATE OR DELETE ON classification_scheme
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER classification_code_append_only BEFORE UPDATE OR DELETE ON classification_code
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER classification_assignment_append_only BEFORE UPDATE OR DELETE ON classification_assignment
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER control_total_append_only BEFORE UPDATE OR DELETE ON control_total
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();
CREATE TRIGGER lead_append_only BEFORE UPDATE OR DELETE ON lead
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS lead_append_only ON lead;
DROP TRIGGER IF EXISTS control_total_append_only ON control_total;
DROP TRIGGER IF EXISTS classification_assignment_append_only ON classification_assignment;
DROP TRIGGER IF EXISTS classification_code_append_only ON classification_code;
DROP TRIGGER IF EXISTS classification_scheme_append_only ON classification_scheme;
DROP TRIGGER IF EXISTS fiscal_fact_append_only ON fiscal_fact;
DROP TRIGGER IF EXISTS entity_alias_append_only ON entity_alias;
DROP TRIGGER IF EXISTS entity_append_only ON entity;
DROP TRIGGER IF EXISTS raw_snapshot_append_only ON raw_snapshot;
DROP TRIGGER IF EXISTS ingestion_run_append_only ON ingestion_run;
DROP FUNCTION IF EXISTS reject_mutation();
