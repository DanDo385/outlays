// Package classify loads a reviewed COFOG mapping file (data/cofog/*.json, a research
// deliverable this code never modifies) and applies it to existing facts as versioned,
// append-only classification_assignment rows (task S9).
//
// Mapping semantics (Hard Rule 5 — never invent a classification):
//   - An entry whose cofogCode is "unmapped" is a deliberate, reviewed non-mapping (e.g.
//     acquisition types describe purchased inputs, not government functions). It produces
//     no assignment rows and is reported explicitly as reviewed-unmapped.
//   - A source category absent from the file is unreviewed: no rows, reported explicitly.
//   - Mapped entries become assigned_by='rule' rows whose basis carries the rule id, the
//     entry's citation, the source category, the verbatim reviewer confidence, and the
//     SHA-256 over the JCS-canonical entry JSON.
//
// Versioning: a fact's first cofog assignment is version 1; if the reviewed entry for its
// category changes (code, confidence, or basis), a new row is appended at latest+1 — the
// view's DISTINCT ON (fact_id) ... ORDER BY version DESC picks it up (D24). Re-applying an
// unchanged mapping inserts nothing (deterministic assignment ids, D21). A fact whose
// latest cofog assignment was made by a human is never overridden by this loader.
package classify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djmagro/outlays/core/internal/store"
	"github.com/djmagro/outlays/core/internal/verify"
)

// SchemePrefixes maps a jurisdiction to the mapping-file key prefixes it may use and the
// per-source classification scheme each prefix resolves to (in-source constants, Hard Rule 8).
var SchemePrefixes = map[string]map[string]string{
	"us-ca": {
		"department":       "us_ca_department",
		"acquisition_type": "us_ca_acquisition_type",
	},
}

// confidenceScale translates the reviewer's ordinal confidence into the NUMERIC column.
// The verbatim word is preserved in basis; this scale is the documented translation.
var confidenceScale = map[string]string{"low": "0.25", "medium": "0.5", "high": "0.75"}

// UnmappedCode marks a reviewed entry that deliberately assigns no COFOG function.
const UnmappedCode = "unmapped"

var cofogCodes = map[string]bool{
	"01": true, "02": true, "03": true, "04": true, "05": true,
	"06": true, "07": true, "08": true, "09": true, "10": true,
}

// Entry is one reviewed mapping entry as it appears in the file.
type Entry struct {
	CofogCode  string `json:"cofogCode"`
	Confidence string `json:"confidence"`
	Basis      string `json:"basis"`
	Note       string `json:"note"`
}

// Category is a parsed, validated entry bound to its source scheme.
type Category struct {
	Key        string // raw file key, e.g. "department: Franchise Tax Board"
	SchemeID   string // e.g. us_ca_department
	Code       string // source-scheme code, e.g. "Franchise Tax Board"
	Entry      Entry
	EntrySha   string // SHA-256 over the JCS-canonical entry JSON
	BasisJSON  string // canonical basis string stored on assignment rows (mapped entries only)
	Confidence string // numeric confidence ("0.25"/"0.5"/"0.75"; mapped entries only)
}

// Mapped reports whether the entry assigns a real COFOG code.
func (c Category) Mapped() bool { return c.Entry.CofogCode != UnmappedCode }

// Mapping is a loaded, validated mapping file.
type Mapping struct {
	Path       string
	Sha256     string // over the exact file bytes; reported, not embedded in basis
	RuleID     string
	Categories []Category // sorted by Key
}

// LoadMapping reads and validates a reviewed mapping file for a jurisdiction. The file is
// read only — corrections to it are proposed via NOTES.md, never made here.
func LoadMapping(path, jurisdiction string) (*Mapping, error) {
	prefixes, ok := SchemePrefixes[jurisdiction]
	if !ok {
		return nil, fmt.Errorf("no scheme prefixes registered for jurisdiction %q", jurisdiction)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mapping: %w", err)
	}
	sum := sha256.Sum256(raw)

	var entries map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse mapping: %w", err)
	}

	m := &Mapping{
		Path:   path,
		Sha256: hex.EncodeToString(sum[:]),
		RuleID: "cofog-map/" + strings.TrimSuffix(filepath.Base(path), ".json"),
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		prefix, code, found := strings.Cut(key, ": ")
		if !found || code == "" {
			return nil, fmt.Errorf("entry %q: key must be \"<prefix>: <category>\"", key)
		}
		schemeID, ok := prefixes[prefix]
		if !ok {
			return nil, fmt.Errorf("entry %q: unknown prefix %q for jurisdiction %s", key, prefix, jurisdiction)
		}

		dec := json.NewDecoder(strings.NewReader(string(entries[key])))
		dec.DisallowUnknownFields()
		var e Entry
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("entry %q: %w", key, err)
		}
		if e.CofogCode != UnmappedCode && !cofogCodes[e.CofogCode] {
			return nil, fmt.Errorf("entry %q: cofogCode %q is not a seeded COFOG code or %q", key, e.CofogCode, UnmappedCode)
		}
		if _, ok := confidenceScale[e.Confidence]; !ok {
			return nil, fmt.Errorf("entry %q: confidence %q not in {low, medium, high}", key, e.Confidence)
		}
		if strings.TrimSpace(e.Basis) == "" {
			return nil, fmt.Errorf("entry %q: basis (citation) is required", key)
		}

		entrySha, err := verify.JCSSha256(entries[key])
		if err != nil {
			return nil, fmt.Errorf("entry %q: canonicalize: %w", key, err)
		}
		c := Category{Key: key, SchemeID: schemeID, Code: code, Entry: e, EntrySha: entrySha}
		if c.Mapped() {
			c.Confidence = confidenceScale[e.Confidence]
			c.BasisJSON, err = basisJSON(m.RuleID, key, e, entrySha)
			if err != nil {
				return nil, fmt.Errorf("entry %q: basis: %w", key, err)
			}
		}
		m.Categories = append(m.Categories, c)
	}
	return m, nil
}

// basisJSON builds the canonical basis string for an assignment row: rule id + citation
// (Hard Rule 5), the source category, the reviewer's verbatim confidence, and the hash of
// the exact reviewed entry. The whole-file hash is deliberately not embedded so unrelated
// file edits do not churn versions; the entry hash pins the reviewed content precisely.
func basisJSON(ruleID, sourceCategory string, e Entry, entrySha string) (string, error) {
	b, err := json.Marshal(map[string]string{
		"ruleId":         ruleID,
		"citation":       e.Basis,
		"sourceCategory": sourceCategory,
		"confidence":     e.Confidence,
		"entrySha256":    entrySha,
	})
	if err != nil {
		return "", err
	}
	canon, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	return string(canon), nil
}

// CategoryReport is one source category that carries no COFOG assignment, listed explicitly.
type CategoryReport struct {
	SchemeID  string `json:"schemeId"`
	Category  string `json:"category"`
	Status    string `json:"status"` // reviewed_unmapped | unreviewed
	Note      string `json:"note,omitempty"`
	FactCount int64  `json:"factCount"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
}

// Conflict aggregates facts whose mapped source categories disagree on the COFOG code; the
// loader assigns nothing for them (ambiguity stays unmapped).
type Conflict struct {
	Candidates []string `json:"candidates"` // "scheme:category -> code", sorted
	FactCount  int64    `json:"factCount"`
}

// Reconciliation proves the cofog view drops nothing: mapped + unclassified == total, and
// the view matches an independent count/sum over current facts.
type Reconciliation struct {
	Total             string `json:"total"`
	Mapped            string `json:"mapped"`
	Unclassified      string `json:"unclassified"`
	FactsTotal        int64  `json:"factsTotal"`
	FactsMapped       int64  `json:"factsMapped"`
	FactsUnclassified int64  `json:"factsUnclassified"`
	Reconciles        bool   `json:"reconciles"`
}

// Report is the loader's output document.
type Report struct {
	MappingFile          string           `json:"mappingFile"`
	MappingSha256        string           `json:"mappingSha256"`
	RuleID               string           `json:"ruleId"`
	Jurisdiction         string           `json:"jurisdiction"`
	FiscalYear           string           `json:"fiscalYear"`
	Flow                 string           `json:"flow"`
	DryRun               bool             `json:"dryRun"`
	Entries              int              `json:"entries"`
	MappedEntries        int              `json:"mappedEntries"`
	ReviewedUnmapped     int              `json:"reviewedUnmappedEntries"`
	Inserted             int              `json:"inserted"`
	UpToDate             int              `json:"upToDate"`
	SkippedHumanOverride int              `json:"skippedHumanOverride"`
	Conflicts            []Conflict       `json:"conflicts"`
	UnmappedCategories   []CategoryReport `json:"unmappedCategories"`
	EntriesNotInData     []string         `json:"entriesNotInData"`
	Reconciliation       Reconciliation   `json:"reconciliation"`
}

const currentFactsCTE = `
	SELECT f.fact_id, f.amount FROM fiscal_fact f
	WHERE f.jurisdiction = $1 AND f.fiscal_year = $2 AND f.flow = $3
	  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)`

type candidate struct {
	srcScheme, srcCode, code, confidence, basis string
}

type planRow struct {
	factID, code, confidence, basis string
	version                         int
}

// Apply runs the pipeline against one jurisdiction-year-flow: plan candidate assignments,
// insert what changed (unless dryRun), and report unmapped categories plus an exact
// reconciliation of the resulting cofog view.
func Apply(ctx context.Context, pool *pgxpool.Pool, m *Mapping, jur, year, flow string, dryRun bool) (*Report, error) {
	r := &Report{
		MappingFile: m.Path, MappingSha256: m.Sha256, RuleID: m.RuleID,
		Jurisdiction: jur, FiscalYear: year, Flow: flow, DryRun: dryRun,
		Entries:   len(m.Categories),
		Conflicts: []Conflict{}, UnmappedCategories: []CategoryReport{}, EntriesNotInData: []string{},
	}
	byKey := map[string]Category{} // "scheme\x00code" -> category
	schemes := []string{}
	seenScheme := map[string]bool{}
	for _, c := range m.Categories {
		if c.Mapped() {
			r.MappedEntries++
		} else {
			r.ReviewedUnmapped++
		}
		byKey[c.SchemeID+"\x00"+c.Code] = c
		if !seenScheme[c.SchemeID] {
			seenScheme[c.SchemeID] = true
			schemes = append(schemes, c.SchemeID)
		}
	}

	// 1. Source categories actually present on current facts in scope.
	type dbCat struct {
		scheme, code, amount string
		count                int64
	}
	dbCats := []dbCat{}
	rows, err := pool.Query(ctx, `
		WITH fy AS (`+currentFactsCTE+`),
		src AS (
			SELECT DISTINCT ON (a.fact_id, a.scheme_id) a.fact_id, a.scheme_id, a.code
			FROM classification_assignment a JOIN fy ON fy.fact_id = a.fact_id
			WHERE a.scheme_id = ANY($4)
			ORDER BY a.fact_id, a.scheme_id, a.version DESC, a.code
		)
		SELECT src.scheme_id, src.code, count(*), COALESCE(sum(fy.amount),0)::numeric(24,4)::text
		FROM src JOIN fy ON fy.fact_id = src.fact_id
		GROUP BY 1, 2 ORDER BY 1, 2`, jur, year, flow, schemes)
	if err != nil {
		return nil, fmt.Errorf("source categories: %w", err)
	}
	for rows.Next() {
		var c dbCat
		if err := rows.Scan(&c.scheme, &c.code, &c.count, &c.amount); err != nil {
			rows.Close()
			return nil, err
		}
		dbCats = append(dbCats, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Plan: join mapped entries to facts, compare against each fact's latest cofog row.
	plan, err := buildPlan(ctx, pool, m, r, jur, year, flow)
	if err != nil {
		return nil, err
	}
	r.Inserted = len(plan)

	// 3. Insert (append-only; deterministic ids make re-runs no-ops).
	if !dryRun && len(plan) > 0 {
		if err := insertPlan(ctx, pool, plan); err != nil {
			return nil, err
		}
	}

	// 4. Unmapped categories, explicitly: reviewed-unmapped (in file) and unreviewed (absent).
	inData := map[string]bool{}
	for _, c := range dbCats {
		inData[c.scheme+"\x00"+c.code] = true
		cat, ok := byKey[c.scheme+"\x00"+c.code]
		switch {
		case !ok:
			r.UnmappedCategories = append(r.UnmappedCategories, CategoryReport{
				SchemeID: c.scheme, Category: c.code, Status: "unreviewed",
				FactCount: c.count, Amount: c.amount, Currency: "USD",
			})
		case !cat.Mapped():
			r.UnmappedCategories = append(r.UnmappedCategories, CategoryReport{
				SchemeID: c.scheme, Category: c.code, Status: "reviewed_unmapped", Note: cat.Entry.Note,
				FactCount: c.count, Amount: c.amount, Currency: "USD",
			})
		}
	}
	for _, c := range m.Categories {
		if !inData[c.SchemeID+"\x00"+c.Code] {
			r.EntriesNotInData = append(r.EntriesNotInData, c.Key)
		}
	}

	// 5. Reconciliation: the cofog view vs an independent count/sum over current facts.
	if err := reconcile(ctx, pool, r, jur, year, flow); err != nil {
		return nil, err
	}
	return r, nil
}

func buildPlan(ctx context.Context, pool *pgxpool.Pool, m *Mapping, r *Report, jur, year, flow string) ([]planRow, error) {
	values, args := []string{}, []any{jur, year, flow}
	for _, c := range m.Categories {
		if !c.Mapped() {
			continue
		}
		n := len(args)
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d::numeric,$%d)", n+1, n+2, n+3, n+4, n+5))
		args = append(args, c.SchemeID, c.Code, c.Entry.CofogCode, c.Confidence, c.BasisJSON)
	}
	if len(values) == 0 {
		return nil, nil
	}

	q := `
	WITH mapping(scheme_id, source_code, cofog_code, confidence, basis) AS (VALUES ` + strings.Join(values, ",") + `),
	fy AS (` + currentFactsCTE + `),
	cand AS (
		SELECT DISTINCT a.fact_id, m.scheme_id AS src_scheme, m.source_code, m.cofog_code, m.confidence, m.basis
		FROM classification_assignment a
		JOIN fy ON fy.fact_id = a.fact_id
		JOIN mapping m ON m.scheme_id = a.scheme_id AND m.source_code = a.code
	),
	latest AS (
		SELECT DISTINCT ON (l.fact_id) l.fact_id, l.code, l.confidence, l.basis, l.assigned_by, l.version
		FROM classification_assignment l JOIN fy ON fy.fact_id = l.fact_id
		WHERE l.scheme_id = 'cofog'
		ORDER BY l.fact_id, l.version DESC, l.code
	)
	SELECT c.fact_id::text, c.src_scheme, c.source_code, c.cofog_code, c.confidence::text, c.basis,
	       l.code, l.confidence::text, l.basis, l.assigned_by, COALESCE(l.version, 0)
	FROM cand c LEFT JOIN latest l ON l.fact_id = c.fact_id
	ORDER BY c.fact_id, c.src_scheme, c.source_code`

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	defer rows.Close()

	type factState struct {
		cands                               []candidate
		latestCode, latestConf, latestBasis *string
		latestBy                            *string
		latestVersion                       int
	}
	order := []string{}
	facts := map[string]*factState{}
	for rows.Next() {
		var (
			factID string
			c      candidate
			fs     factState
		)
		if err := rows.Scan(&factID, &c.srcScheme, &c.srcCode, &c.code, &c.confidence, &c.basis,
			&fs.latestCode, &fs.latestConf, &fs.latestBasis, &fs.latestBy, &fs.latestVersion); err != nil {
			return nil, err
		}
		st, ok := facts[factID]
		if !ok {
			st = &factState{latestCode: fs.latestCode, latestConf: fs.latestConf,
				latestBasis: fs.latestBasis, latestBy: fs.latestBy, latestVersion: fs.latestVersion}
			facts[factID] = st
			order = append(order, factID)
		}
		st.cands = append(st.cands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	plan := []planRow{}
	conflicts := map[string]*Conflict{}
	for _, factID := range order {
		st := facts[factID]
		distinct := map[string]bool{}
		for _, c := range st.cands {
			distinct[c.code] = true
		}
		if len(distinct) > 1 {
			// Ambiguity is left unmapped, never resolved by invention.
			sig := []string{}
			for _, c := range st.cands {
				sig = append(sig, fmt.Sprintf("%s:%s -> %s", c.srcScheme, c.srcCode, c.code))
			}
			sort.Strings(sig)
			key := strings.Join(sig, " | ")
			if conflicts[key] == nil {
				conflicts[key] = &Conflict{Candidates: sig}
			}
			conflicts[key].FactCount++
			continue
		}
		// Codes agree; deterministic winner is the first by (scheme, category) sort order.
		w := st.cands[0]
		if st.latestBy != nil && *st.latestBy == "human" {
			if !latestMatches(st.latestCode, st.latestConf, st.latestBasis, w) {
				r.SkippedHumanOverride++
			} else {
				r.UpToDate++
			}
			continue
		}
		if st.latestCode != nil && latestMatches(st.latestCode, st.latestConf, st.latestBasis, w) {
			r.UpToDate++
			continue
		}
		plan = append(plan, planRow{factID: factID, code: w.code, confidence: w.confidence,
			basis: w.basis, version: st.latestVersion + 1})
	}

	keys := make([]string, 0, len(conflicts))
	for k := range conflicts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		r.Conflicts = append(r.Conflicts, *conflicts[k])
	}
	return plan, nil
}

func latestMatches(code, conf, basis *string, w candidate) bool {
	return code != nil && *code == w.code &&
		conf != nil && *conf == w.confidence &&
		basis != nil && *basis == w.basis
}

func insertPlan(ctx context.Context, pool *pgxpool.Pool, plan []planRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	b := &pgx.Batch{}
	for _, p := range plan {
		b.Queue(`
			INSERT INTO classification_assignment
				(assignment_id, fact_id, scheme_id, code, assigned_by, confidence, basis, version)
			VALUES ($1, $2, 'cofog', $3, 'rule', $4::numeric, $5, $6)
			ON CONFLICT (assignment_id) DO NOTHING`,
			store.AssignmentID(p.factID, "cofog", p.code, p.version),
			p.factID, p.code, p.confidence, p.basis, p.version)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return fmt.Errorf("insert assignments: %w", err)
	}
	return tx.Commit(ctx)
}

func reconcile(ctx context.Context, pool *pgxpool.Pool, r *Report, jur, year, flow string) error {
	var factsTotal int64
	var factsSum string
	if err := pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(f.amount),0)::numeric(24,4)::text FROM fiscal_fact f
		WHERE f.jurisdiction = $1 AND f.fiscal_year = $2 AND f.flow = $3
		  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)`,
		jur, year, flow).Scan(&factsTotal, &factsSum); err != nil {
		return fmt.Errorf("independent fact count: %w", err)
	}

	v, err := store.ViewByScheme(ctx, pool, jur, year, flow, "cofog")
	if err != nil {
		return fmt.Errorf("cofog view: %w", err)
	}
	mapped, nodeSum := "0.0000", "0.0000"
	var mappedCount, nodeCount, unclassifiedCount int64
	for _, n := range v.Nodes {
		nodeSum = store.AddDecimals(nodeSum, n.Amount)
		nodeCount += n.FactCount
		if n.Code == store.UnclassifiedCode {
			unclassifiedCount = n.FactCount
		} else {
			mapped = store.AddDecimals(mapped, n.Amount)
			mappedCount += n.FactCount
		}
	}

	r.Reconciliation = Reconciliation{
		Total: v.Total, Mapped: mapped, Unclassified: v.Unmapped,
		FactsTotal: factsTotal, FactsMapped: mappedCount, FactsUnclassified: unclassifiedCount,
		Reconciles: nodeSum == v.Total &&
			v.Total == factsSum &&
			store.AddDecimals(mapped, v.Unmapped) == v.Total &&
			nodeCount == factsTotal &&
			mappedCount+unclassifiedCount == factsTotal,
	}
	return nil
}
