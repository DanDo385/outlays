-- +goose Up
INSERT INTO classification_scheme (scheme_id, name, hierarchical) VALUES
  ('us_fed_awarding_agency', 'Federal awarding agency (USAspending source-coded)', false),
  ('us_fed_award_type', 'Federal award type (USAspending source-coded)', false);

-- +goose Down
SET session_replication_role = replica;
DELETE FROM classification_scheme WHERE scheme_id IN ('us_fed_awarding_agency', 'us_fed_award_type');
SET session_replication_role = origin;
