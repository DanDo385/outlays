-- +goose Up
-- Seed classification schemes and the COFOG top level (ARCHITECTURE.md Section 3). Per-source
-- schemes (us_ca_*) are registered here; their codes are inserted by the ingest writer as
-- facts are loaded (assigned_by='source').

INSERT INTO classification_scheme (scheme_id, name, hierarchical) VALUES
  ('cofog', 'COFOG (Classification of the Functions of Government)', true),
  ('object_class', 'Object class', false),
  ('department', 'Department', false),
  ('fund', 'Fund', false),
  ('program', 'Program', false),
  ('recipient_type', 'Recipient type', false),
  ('tag', 'Tag', false),
  ('us_ca_department', 'California department (source-coded)', false),
  ('us_ca_acquisition_type', 'California acquisition type (source-coded)', false);

INSERT INTO classification_code (scheme_id, code, parent_code, name) VALUES
  ('cofog', '01', NULL, 'General public services'),
  ('cofog', '02', NULL, 'Defence'),
  ('cofog', '03', NULL, 'Public order and safety'),
  ('cofog', '04', NULL, 'Economic affairs'),
  ('cofog', '05', NULL, 'Environmental protection'),
  ('cofog', '06', NULL, 'Housing and community amenities'),
  ('cofog', '07', NULL, 'Health'),
  ('cofog', '08', NULL, 'Recreation, culture and religion'),
  ('cofog', '09', NULL, 'Education'),
  ('cofog', '10', NULL, 'Social protection');

-- +goose Down
-- Bypass the append-only triggers for this controlled rollback (owner/superuser only).
SET session_replication_role = replica;
DELETE FROM classification_code WHERE scheme_id = 'cofog';
DELETE FROM classification_scheme WHERE scheme_id IN
  ('cofog','object_class','department','fund','program','recipient_type','tag',
   'us_ca_department','us_ca_acquisition_type');
SET session_replication_role = origin;
