package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deterministic id namespaces (UUIDv5), so re-ingesting the same document is a no-op.
var (
	factNS   = uuid.MustParse("3a0c0b6e-1d2f-5a4b-8c6d-7e8f9a0b1c20")
	aliasNS  = uuid.MustParse("3a0c0b6e-1d2f-5a4b-8c6d-7e8f9a0b1c21")
	assignNS = uuid.MustParse("3a0c0b6e-1d2f-5a4b-8c6d-7e8f9a0b1c22")
)

// Document is the parsed adapter --out payload (AdapterOutput).
type Document struct {
	EnvelopeRaw   json.RawMessage `json:"envelope"`
	Envelope      Envelope        `json:"-"`
	Facts         []Fact          `json:"facts"`
	Entities      []Entity        `json:"entities"`
	EntityAliases []Alias         `json:"entityAliases"`
	ControlTotals []ControlTotal  `json:"controlTotals"`
}

// ControlTotal is an official published total used as the coverage denominator; scope labels
// what coverage against it means (e.g. "procurement facts vs total budget").
type ControlTotal struct {
	Jurisdiction    string `json:"jurisdiction"`
	FiscalYear      string `json:"fiscalYear"`
	Flow            string `json:"flow"`
	OfficialTotal   string `json:"officialTotal"`
	Currency        string `json:"currency"`
	Scope           string `json:"scope"`
	RawSha256       string `json:"rawSha256"`
	DerivationQuery string `json:"derivationQuery"`
}

type Envelope struct {
	RunID          string        `json:"runId"`
	AdapterID      string        `json:"adapterId"`
	AdapterVersion string        `json:"adapterVersion"`
	Jurisdiction   string        `json:"jurisdiction"`
	FiscalYear     string        `json:"fiscalYear"`
	FetchedAt      string        `json:"fetchedAt"`
	ResultHash     string        `json:"resultHash"`
	RawSnapshots   []RawSnapshot `json:"rawSnapshots"`
}

type RawSnapshot struct {
	Sha256     string `json:"sha256"`
	URL        string `json:"url"`
	Bytes      int64  `json:"bytes"`
	HTTPStatus int    `json:"httpStatus"`
}

type Assignment struct {
	SchemeID   string   `json:"schemeId"`
	Code       string   `json:"code"`
	AssignedBy string   `json:"assignedBy"`
	Version    int      `json:"version"`
	Confidence *float64 `json:"confidence"`
	Basis      *string  `json:"basis"`
}

type Fact struct {
	Jurisdiction    string       `json:"jurisdiction"`
	FiscalYear      string       `json:"fiscalYear"`
	Flow            string       `json:"flow"`
	Grain           string       `json:"grain"`
	Amount          string       `json:"amount"`
	Currency        string       `json:"currency"`
	DerivationQuery string       `json:"derivationQuery"`
	FactHash        string       `json:"factHash"`
	PayeeEntity     *string      `json:"payeeEntity"`
	PayerEntity     *string      `json:"payerEntity"`
	OccurredOn      *string      `json:"occurredOn"`
	Description     *string      `json:"description"`
	RawSha256       *string      `json:"rawSha256"`
	Supersedes      *string      `json:"supersedes"`
	Assignments     []Assignment `json:"assignments"`
}

type Entity struct {
	EntityID      string  `json:"entityId"`
	Kind          string  `json:"kind"`
	CanonicalName string  `json:"canonicalName"`
	UEI           *string `json:"uei"`
	EIN           *string `json:"ein"`
	Jurisdiction  *string `json:"jurisdiction"`
}

type Alias struct {
	EntityID   *string         `json:"entityId"`
	NameRaw    string          `json:"nameRaw"`
	MatchedBy  string          `json:"matchedBy"`
	Confidence *float64        `json:"confidence"`
	Source     json.RawMessage `json:"source"`
}

// ParseDocument parses an adapter --out document.
func ParseDocument(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	}
	if err := json.Unmarshal(doc.EnvelopeRaw, &doc.Envelope); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	return &doc, nil
}

// FactID is the deterministic fact_id derived from a fact's content hash.
func FactID(factHash string) string { return uuid.NewSHA1(factNS, []byte(factHash)).String() }

// AssignmentID is the deterministic assignment_id (D21): UUIDv5 over fact|scheme|code|version,
// so re-applying the same assignment is an ON CONFLICT no-op.
func AssignmentID(factID, schemeID, code string, version int) string {
	return uuid.NewSHA1(assignNS, []byte(fmt.Sprintf("%s|%s|%s|%d", factID, schemeID, code, version))).String()
}

// IngestResult summarizes what was written.
type IngestResult struct {
	RunID         string
	Facts         int
	Entities      int
	Aliases       int
	Assignments   int
	Snapshots     int
	ControlTotals int
}

// Ingest persists a document: uploads raw snapshots to object storage, then writes all rows in
// one append-only transaction (deterministic ids ⇒ idempotent). rawDir holds the adapter's
// <sha>.bin / <sha>.meta.json files; dataset names the object-storage partition.
func Ingest(ctx context.Context, pool *pgxpool.Pool, obj *ObjectStore, doc *Document, rawDir, dataset string) (*IngestResult, error) {
	env := doc.Envelope
	fetchedAt, err := time.Parse(time.RFC3339, env.FetchedAt)
	if err != nil {
		return nil, fmt.Errorf("envelope.fetchedAt: %w", err)
	}

	// 1. Upload raw snapshots (content-addressed keys are idempotent).
	for _, s := range env.RawSnapshots {
		bin, err := os.ReadFile(filepath.Join(rawDir, s.Sha256+".bin"))
		if err != nil {
			return nil, fmt.Errorf("read snapshot %s: %w", s.Sha256, err)
		}
		if err := obj.Put(ctx, RawKey(env.Jurisdiction, dataset, env.FiscalYear, s.Sha256), bin, "application/octet-stream"); err != nil {
			return nil, err
		}
		if meta, err := os.ReadFile(filepath.Join(rawDir, s.Sha256+".meta.json")); err == nil {
			if err := obj.Put(ctx, RawMetaKey(env.Jurisdiction, dataset, env.FiscalYear, s.Sha256), meta, "application/json"); err != nil {
				return nil, err
			}
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 2. ingestion_run (insert-once; succeeded).
	if _, err := tx.Exec(ctx, `
		INSERT INTO ingestion_run (run_id, adapter_id, adapter_version, jurisdiction, fiscal_year, completed_at, status, envelope)
		VALUES ($1,$2,$3,$4,$5,$6,'succeeded',$7)
		ON CONFLICT (run_id) DO NOTHING`,
		env.RunID, env.AdapterID, env.AdapterVersion, env.Jurisdiction, env.FiscalYear, time.Now(), []byte(doc.EnvelopeRaw),
	); err != nil {
		return nil, fmt.Errorf("insert ingestion_run: %w", err)
	}

	// 3. raw_snapshot rows.
	for _, s := range env.RawSnapshots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO raw_snapshot (sha256, storage_key, url, http_status, bytes, fetched_at, run_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (sha256) DO NOTHING`,
			s.Sha256, RawKey(env.Jurisdiction, dataset, env.FiscalYear, s.Sha256), s.URL, s.HTTPStatus, s.Bytes, fetchedAt, env.RunID,
		); err != nil {
			return nil, fmt.Errorf("insert raw_snapshot: %w", err)
		}
	}

	// 4. entities.
	if err := copyUpsert(ctx, tx, "entity",
		[]string{"entity_id", "kind", "canonical_name", "uei", "ein", "jurisdiction"},
		"entity_id", entityRows(doc.Entities)); err != nil {
		return nil, err
	}

	// 5. entity aliases (deterministic alias_id).
	if err := copyUpsert(ctx, tx, "entity_alias",
		[]string{"alias_id", "entity_id", "name_raw", "matched_by", "confidence", "source"},
		"alias_id", aliasRows(doc.EntityAliases)); err != nil {
		return nil, err
	}

	// 6. classification_code: ensure every (scheme, code) used exists (append-only INSERT).
	seen := map[string]bool{}
	for _, f := range doc.Facts {
		for _, a := range f.Assignments {
			key := a.SchemeID + "\x00" + a.Code
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, err := tx.Exec(ctx, `
				INSERT INTO classification_code (scheme_id, code, parent_code, name)
				VALUES ($1,$2,NULL,$2) ON CONFLICT (scheme_id, code) DO NOTHING`, a.SchemeID, a.Code); err != nil {
				return nil, fmt.Errorf("insert classification_code: %w", err)
			}
		}
	}

	// 7. fiscal_fact + 8. classification_assignment.
	factRows, assignRows, err := factAndAssignmentRows(env.RunID, doc.Facts)
	if err != nil {
		return nil, err
	}
	if err := copyUpsert(ctx, tx, "fiscal_fact",
		[]string{"fact_id", "run_id", "jurisdiction", "fiscal_year", "flow", "grain",
			"payer_entity", "payee_entity", "amount", "currency", "occurred_on",
			"description", "raw_sha256", "derivation_query", "fact_hash", "supersedes"},
		"fact_id", factRows); err != nil {
		return nil, err
	}
	if err := copyUpsert(ctx, tx, "classification_assignment",
		[]string{"assignment_id", "fact_id", "scheme_id", "code", "assigned_by", "confidence", "basis", "version"},
		"assignment_id", assignRows); err != nil {
		return nil, err
	}

	// 9. control totals (PK (jurisdiction, fiscal_year, flow) ⇒ idempotent insert-once).
	for _, ct := range doc.ControlTotals {
		total, perr := pgNumeric(ct.OfficialTotal)
		if perr != nil {
			return nil, fmt.Errorf("control total %q: %w", ct.OfficialTotal, perr)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO control_total (jurisdiction, fiscal_year, flow, official_total, currency, scope, raw_sha256, derivation_query)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (jurisdiction, fiscal_year, flow) DO NOTHING`,
			ct.Jurisdiction, ct.FiscalYear, ct.Flow, total, ct.Currency, ct.Scope, ct.RawSha256, ct.DerivationQuery,
		); err != nil {
			return nil, fmt.Errorf("insert control_total: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &IngestResult{
		RunID: env.RunID, Facts: len(factRows), Entities: len(doc.Entities),
		Aliases: len(doc.EntityAliases), Assignments: len(assignRows), Snapshots: len(env.RawSnapshots),
		ControlTotals: len(doc.ControlTotals),
	}, nil
}

// copyUpsert COPYs rows into a trigger-free temp table, then INSERT ... SELECT ON CONFLICT
// DO NOTHING into the real (append-only) table. This is the batched COPY writer.
func copyUpsert(ctx context.Context, tx pgx.Tx, table string, cols []string, conflictCol string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	tmp := "tmp_" + table
	if _, err := tx.Exec(ctx, fmt.Sprintf(`CREATE TEMP TABLE %s (LIKE %s INCLUDING DEFAULTS) ON COMMIT DROP`, tmp, table)); err != nil {
		return fmt.Errorf("create temp %s: %w", table, err)
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{tmp}, cols, pgx.CopyFromRows(rows)); err != nil {
		return fmt.Errorf("copy into %s: %w", table, err)
	}
	collist := ""
	for i, c := range cols {
		if i > 0 {
			collist += ", "
		}
		collist += c
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (%s) SELECT %s FROM %s ON CONFLICT (%s) DO NOTHING`,
		table, collist, collist, tmp, conflictCol)); err != nil {
		return fmt.Errorf("insert-select %s: %w", table, err)
	}
	return nil
}

func entityRows(entities []Entity) [][]any {
	rows := make([][]any, 0, len(entities))
	for _, e := range entities {
		rows = append(rows, []any{pgUUID(e.EntityID), e.Kind, e.CanonicalName, e.UEI, e.EIN, e.Jurisdiction})
	}
	return rows
}

func aliasRows(aliases []Alias) [][]any {
	rows := make([][]any, 0, len(aliases))
	for _, a := range aliases {
		entityID := ""
		if a.EntityID != nil {
			entityID = *a.EntityID
		}
		aliasID := uuid.NewSHA1(aliasNS, []byte(entityID+"|"+a.NameRaw)).String()
		src := a.Source
		if len(src) == 0 {
			src = json.RawMessage("{}")
		}
		rows = append(rows, []any{pgUUID(aliasID), pgUUIDPtr(a.EntityID), a.NameRaw, a.MatchedBy, pgFloat(a.Confidence), string(src)})
	}
	return rows
}

func factAndAssignmentRows(runID string, facts []Fact) (factRows, assignRows [][]any, err error) {
	for _, f := range facts {
		factID := FactID(f.FactHash)
		amount, perr := pgNumeric(f.Amount)
		if perr != nil {
			return nil, nil, fmt.Errorf("amount %q: %w", f.Amount, perr)
		}
		factRows = append(factRows, []any{
			pgUUID(factID), pgUUID(runID), f.Jurisdiction, f.FiscalYear, f.Flow, f.Grain,
			pgUUIDPtr(f.PayerEntity), pgUUIDPtr(f.PayeeEntity), amount, f.Currency,
			pgDate(f.OccurredOn), f.Description, f.RawSha256, f.DerivationQuery, f.FactHash, pgUUIDPtr(f.Supersedes),
		})
		for _, a := range f.Assignments {
			assignID := AssignmentID(factID, a.SchemeID, a.Code, a.Version)
			assignRows = append(assignRows, []any{
				pgUUID(assignID), pgUUID(factID), a.SchemeID, a.Code, a.AssignedBy, pgFloat(a.Confidence), a.Basis, a.Version,
			})
		}
	}
	return factRows, assignRows, nil
}

// --- pgtype helpers ---

func pgUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

func pgUUIDPtr(s *string) pgtype.UUID {
	if s == nil {
		return pgtype.UUID{}
	}
	return pgUUID(*s)
}

func pgNumeric(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return n, err
	}
	return n, nil
}

func pgFloat(f *float64) pgtype.Numeric {
	var n pgtype.Numeric
	if f == nil {
		return n
	}
	_ = n.Scan(fmt.Sprintf("%g", *f))
	return n
}

func pgDate(s *string) pgtype.Date {
	var d pgtype.Date
	if s == nil {
		return d
	}
	_ = d.Scan(*s)
	return d
}
