-- +goose Up
-- Lead review workflow, append-only (task S11; resolves the D20 open question). The lead
-- row is immutable: its status column records only the initial machine state ('draft').
-- Every human review action is a new lead_event row carrying the mandatory reviewer
-- handle; a lead's current status is its latest event, or the row's initial status when no
-- events exist. Nothing is public unless a human published it (Hard Rule 6).
--
-- lead.body holds the full generated lead document (title, summary, severity, citations,
-- limitations, safe wording, subject, params — the methodology library's minimum output
-- shape), so the public endpoint can serve citation and context without schema sprawl.

ALTER TABLE lead ADD COLUMN body JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE lead_event (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  lead_id UUID NOT NULL REFERENCES lead(lead_id),
  status TEXT NOT NULL CHECK (status IN ('draft','reviewed','published','dismissed')),
  reviewer TEXT NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lead_event_lead_idx ON lead_event (lead_id, created_at DESC);

CREATE TRIGGER lead_event_append_only BEFORE UPDATE OR DELETE ON lead_event
  FOR EACH ROW EXECUTE FUNCTION reject_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS lead_event_append_only ON lead_event;
DROP TABLE IF EXISTS lead_event;
ALTER TABLE lead DROP COLUMN IF EXISTS body;
