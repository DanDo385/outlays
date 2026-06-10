package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UnclassifiedCode is the synthetic node code for facts lacking an assignment in a scheme.
const UnclassifiedCode = "__unclassified__"

// currentFacts is the WHERE fragment selecting non-superseded facts for a jurisdiction-year.
// Correction rows (which set supersedes) are themselves current; the rows they replace are not.
const currentFactsCTE = `
	SELECT f.fact_id, f.amount, f.currency, f.payee_entity
	FROM fiscal_fact f
	WHERE f.jurisdiction = $1 AND f.fiscal_year = $2 AND f.flow = $3
	  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)`

// Node is one row of a computed view.
type Node struct {
	Code      string `json:"code"`
	Label     string `json:"label"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	FactCount int64  `json:"factCount"`
}

// View is a one-level rollup over a scheme (or the payee tree) for a jurisdiction-year-flow.
type View struct {
	Jurisdiction string `json:"jurisdiction"`
	FiscalYear   string `json:"fiscalYear"`
	Flow         string `json:"flow"`
	SchemeID     string `json:"schemeId"`
	Total        string `json:"total"`
	Currency     string `json:"currency"`
	Unmapped     string `json:"unmapped"`
	Nodes        []Node `json:"nodes"`
}

// SchemeExists reports whether a classification scheme is registered.
func SchemeExists(ctx context.Context, pool *pgxpool.Pool, scheme string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM classification_scheme WHERE scheme_id=$1)`, scheme).Scan(&ok)
	return ok, err
}

// ViewByScheme groups current facts by their (de-duplicated) assignment code in the scheme,
// with an explicit unclassified bucket for facts that have no assignment in that scheme.
func ViewByScheme(ctx context.Context, pool *pgxpool.Pool, jur, year, flow, scheme string) (*View, error) {
	q := `
	WITH fy AS (` + currentFactsCTE + `),
	asg AS (
		SELECT DISTINCT ON (a.fact_id) a.fact_id, a.code
		FROM classification_assignment a
		JOIN fy ON fy.fact_id = a.fact_id
		WHERE a.scheme_id = $4
		ORDER BY a.fact_id, a.version DESC, a.code
	)
	SELECT
		COALESCE(asg.code, '` + UnclassifiedCode + `') AS code,
		COALESCE(cc.name, CASE WHEN asg.code IS NULL THEN 'Unclassified' ELSE asg.code END) AS label,
		count(*) AS fact_count,
		COALESCE(sum(fy.amount), 0)::numeric(24,4)::text AS amount
	FROM fy
	LEFT JOIN asg ON asg.fact_id = fy.fact_id
	LEFT JOIN classification_code cc ON cc.scheme_id = $4 AND cc.code = asg.code
	GROUP BY 1, 2
	ORDER BY sum(fy.amount) DESC NULLS LAST`
	v := &View{Jurisdiction: jur, FiscalYear: year, Flow: flow, SchemeID: scheme, Currency: "USD", Nodes: []Node{}}
	return scanView(ctx, pool, v, q, jur, year, flow, scheme)
}

// ViewByPayee groups current facts by payee entity (the vendor tree), with an unclassified
// bucket for facts that have no payee.
func ViewByPayee(ctx context.Context, pool *pgxpool.Pool, jur, year, flow string) (*View, error) {
	q := `
	WITH fy AS (` + currentFactsCTE + `)
	SELECT
		COALESCE(e.entity_id::text, '` + UnclassifiedCode + `') AS code,
		COALESCE(e.canonical_name, 'Unclassified') AS label,
		count(*) AS fact_count,
		COALESCE(sum(fy.amount), 0)::numeric(24,4)::text AS amount
	FROM fy
	LEFT JOIN entity e ON e.entity_id = fy.payee_entity
	GROUP BY 1, 2
	ORDER BY sum(fy.amount) DESC NULLS LAST`
	v := &View{Jurisdiction: jur, FiscalYear: year, Flow: flow, SchemeID: "payee", Currency: "USD", Nodes: []Node{}}
	return scanView(ctx, pool, v, q, jur, year, flow)
}

func scanView(ctx context.Context, pool *pgxpool.Pool, v *View, q string, args ...any) (*View, error) {
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.Code, &n.Label, &n.FactCount, &n.Amount); err != nil {
			return nil, err
		}
		n.Currency = "USD"
		v.Nodes = append(v.Nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Total + unmapped derived from the nodes (they already reconcile).
	total, unmapped := addDecimals("0.0000", "0.0000"), "0.0000"
	for _, n := range v.Nodes {
		total = addDecimals(total, n.Amount)
		if n.Code == UnclassifiedCode {
			unmapped = n.Amount
		}
	}
	v.Total = total
	v.Unmapped = unmapped
	return v, nil
}

// EntityHit is a search result.
type EntityHit struct {
	EntityID      string  `json:"entityId"`
	Kind          string  `json:"kind"`
	CanonicalName string  `json:"canonicalName"`
	Jurisdiction  *string `json:"jurisdiction"`
}

// SearchEntities finds entities by canonical-name substring (case-insensitive).
func SearchEntities(ctx context.Context, pool *pgxpool.Pool, q string, limit int) ([]EntityHit, error) {
	rows, err := pool.Query(ctx, `
		SELECT entity_id::text, kind, canonical_name, jurisdiction
		FROM entity WHERE canonical_name ILIKE '%' || $1 || '%'
		ORDER BY canonical_name LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := []EntityHit{}
	for rows.Next() {
		var h EntityHit
		if err := rows.Scan(&h.EntityID, &h.Kind, &h.CanonicalName, &h.Jurisdiction); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// EntityFlows is one entity's spending broken down by department (the cross-cut that shows a
// vendor across departments).
type EntityFlows struct {
	EntityID      string `json:"entityId"`
	CanonicalName string `json:"canonicalName"`
	FiscalYear    string `json:"fiscalYear"`
	Total         string `json:"total"`
	Currency      string `json:"currency"`
	ByDepartment  []Node `json:"byDepartment"`
}

// EntityFlowsByDepartment returns, for an entity in a year, its facts grouped by the
// us_ca_department scheme.
func EntityFlowsByDepartment(ctx context.Context, pool *pgxpool.Pool, entityID, year string) (*EntityFlows, error) {
	var name string
	if err := pool.QueryRow(ctx, `SELECT canonical_name FROM entity WHERE entity_id=$1`, entityID).Scan(&name); err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		WITH fy AS (
			SELECT f.fact_id, f.amount FROM fiscal_fact f
			WHERE f.payee_entity = $1 AND f.fiscal_year = $2
			  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)
		),
		asg AS (
			SELECT DISTINCT ON (a.fact_id) a.fact_id, a.code FROM classification_assignment a
			JOIN fy ON fy.fact_id = a.fact_id WHERE a.scheme_id = 'us_ca_department'
			ORDER BY a.fact_id, a.version DESC, a.code
		)
		SELECT COALESCE(asg.code, '`+UnclassifiedCode+`'),
		       COALESCE(asg.code, 'Unclassified'),
		       count(*), COALESCE(sum(fy.amount),0)::numeric(24,4)::text
		FROM fy LEFT JOIN asg ON asg.fact_id = fy.fact_id
		GROUP BY 1,2 ORDER BY sum(fy.amount) DESC NULLS LAST`, entityID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ef := &EntityFlows{EntityID: entityID, CanonicalName: name, FiscalYear: year, Currency: "USD", ByDepartment: []Node{}}
	total := "0.0000"
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.Code, &n.Label, &n.FactCount, &n.Amount); err != nil {
			return nil, err
		}
		n.Currency = "USD"
		ef.ByDepartment = append(ef.ByDepartment, n)
		total = addDecimals(total, n.Amount)
	}
	ef.Total = total
	return ef, rows.Err()
}

// FactRow is a paged fact listing row.
type FactRow struct {
	FactID       string  `json:"factId"`
	Jurisdiction string  `json:"jurisdiction"`
	FiscalYear   string  `json:"fiscalYear"`
	Flow         string  `json:"flow"`
	Grain        string  `json:"grain"`
	Amount       string  `json:"amount"`
	Currency     string  `json:"currency"`
	OccurredOn   *string `json:"occurredOn"`
	Description  *string `json:"description"`
	PayeeEntity  *string `json:"payeeEntity"`
	FactHash     string  `json:"factHash"`
}

// ListFacts returns a page of current facts filtered by jurisdiction/year/flow/payee and,
// optionally, by a view node (scheme, code) — the drill-down behind the UI. Mirroring the
// view endpoint: code __unclassified__ selects facts with no assignment in the scheme, and
// scheme 'payee' selects by payee entity (code = entity id, or __unclassified__ = no payee).
func ListFacts(ctx context.Context, pool *pgxpool.Pool, jur, year, flow, payee, scheme, code string, limit, offset int) ([]FactRow, error) {
	q := `
		SELECT fact_id::text, jurisdiction, fiscal_year, flow, grain, amount::text, currency,
		       occurred_on::text, description, payee_entity::text, fact_hash
		FROM fiscal_fact
		WHERE ($1='' OR jurisdiction=$1) AND ($2='' OR fiscal_year=$2)
		  AND ($3='' OR flow=$3) AND ($4='' OR payee_entity=$4::uuid)
		  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = fiscal_fact.fact_id)`
	args := []any{jur, year, flow, payee}
	switch {
	case scheme == "":
	case scheme == "payee" && code == UnclassifiedCode:
		q += ` AND payee_entity IS NULL`
	case scheme == "payee":
		args = append(args, code)
		q += fmt.Sprintf(` AND payee_entity = $%d::uuid`, len(args))
	case code == UnclassifiedCode:
		args = append(args, scheme)
		q += fmt.Sprintf(` AND NOT EXISTS (SELECT 1 FROM classification_assignment a
			WHERE a.fact_id = fiscal_fact.fact_id AND a.scheme_id = $%d)`, len(args))
	default:
		args = append(args, scheme, code)
		q += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM classification_assignment a
			WHERE a.fact_id = fiscal_fact.fact_id AND a.scheme_id = $%d AND a.code = $%d)`,
			len(args)-1, len(args))
	}
	args = append(args, limit, offset)
	q += fmt.Sprintf(` ORDER BY amount DESC, fact_id LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FactRow{}
	for rows.Next() {
		var f FactRow
		if err := rows.Scan(&f.FactID, &f.Jurisdiction, &f.FiscalYear, &f.Flow, &f.Grain, &f.Amount,
			&f.Currency, &f.OccurredOn, &f.Description, &f.PayeeEntity, &f.FactHash); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Provenance points a fact at its raw snapshot and hashes.
type Provenance struct {
	FactID          string  `json:"factId"`
	FactHash        string  `json:"factHash"`
	DerivationQuery string  `json:"derivationQuery"`
	RawSha256       *string `json:"rawSha256"`
	StorageKey      *string `json:"storageKey"`
	SnapshotURL     *string `json:"snapshotUrl"`
	HTTPStatus      *int    `json:"httpStatus"`
	Bytes           *int64  `json:"bytes"`
}

// FactProvenance joins a fact to its raw snapshot (object-store pointer + hashes).
func FactProvenance(ctx context.Context, pool *pgxpool.Pool, factID string) (*Provenance, error) {
	var p Provenance
	err := pool.QueryRow(ctx, `
		SELECT f.fact_id::text, f.fact_hash, f.derivation_query, f.raw_sha256,
		       rs.storage_key, rs.url, rs.http_status, rs.bytes
		FROM fiscal_fact f
		LEFT JOIN raw_snapshot rs ON rs.sha256 = f.raw_sha256
		WHERE f.fact_id = $1`, factID,
	).Scan(&p.FactID, &p.FactHash, &p.DerivationQuery, &p.RawSha256, &p.StorageKey, &p.SnapshotURL, &p.HTTPStatus, &p.Bytes)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// NumeratorBasis is the provenance of the coverage numerator: the exact aggregation the API
// ran, plus the facts listing where every underlying fact carries its own provenance link.
type NumeratorBasis struct {
	DerivationQuery string `json:"derivationQuery"`
	FactsURL        string `json:"factsUrl"`
}

// DenominatorBasis is the provenance of the coverage denominator: the control_total's scope
// label and its raw-snapshot pointer (hash, object-store key, source URL, derivation).
type DenominatorBasis struct {
	Scope           string  `json:"scope"`
	DerivationQuery string  `json:"derivationQuery"`
	RawSha256       string  `json:"rawSha256"`
	StorageKey      *string `json:"storageKey"`
	SnapshotURL     *string `json:"snapshotUrl"`
}

// Coverage is sum(transaction+award facts) / official_total for a jurisdiction-year. Both
// sides carry provenance; the denominator's scope labels what the comparison means (e.g.
// "procurement facts vs total budget") so a low or scope-mismatched number is never dressed
// up as precision.
type Coverage struct {
	Jurisdiction     string            `json:"jurisdiction"`
	FiscalYear       string            `json:"fiscalYear"`
	Numerator        string            `json:"numerator"`
	NumeratorBasis   NumeratorBasis    `json:"numeratorBasis"`
	Denominator      *string           `json:"denominator"`
	DenominatorBasis *DenominatorBasis `json:"denominatorBasis"`
	Ratio            *string           `json:"ratio"`
	Currency         string            `json:"currency"`
}

const coverageNumeratorSQL = `SELECT COALESCE(sum(amount),0)::numeric(24,4)::text FROM fiscal_fact f
		WHERE jurisdiction=$1 AND fiscal_year=$2 AND grain IN ('transaction','award')
		  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes = f.fact_id)`

// CoverageFor computes coverage; denominator/ratio/denominatorBasis are null until a
// control_total exists for the jurisdiction-year.
func CoverageFor(ctx context.Context, pool *pgxpool.Pool, jur, year string) (*Coverage, error) {
	c := &Coverage{
		Jurisdiction: jur, FiscalYear: year, Currency: "USD",
		NumeratorBasis: NumeratorBasis{
			DerivationQuery: fmt.Sprintf(
				"sum(amount) over current (non-superseded) transaction+award facts where jurisdiction=%q and fiscal_year=%q", jur, year),
			FactsURL: fmt.Sprintf("/v1/facts?jurisdiction=%s&year=%s", jur, year),
		},
	}
	if err := pool.QueryRow(ctx, coverageNumeratorSQL, jur, year).Scan(&c.Numerator); err != nil {
		return nil, err
	}

	var (
		denom, scope, ctDerivation, rawSha string
		storageKey, snapshotURL            *string
	)
	err := pool.QueryRow(ctx, `
		SELECT ct.official_total::text, ct.scope, ct.derivation_query, ct.raw_sha256,
		       rs.storage_key, rs.url
		FROM control_total ct
		LEFT JOIN raw_snapshot rs ON rs.sha256 = ct.raw_sha256
		WHERE ct.jurisdiction=$1 AND ct.fiscal_year=$2 AND ct.flow='spending'`, jur, year,
	).Scan(&denom, &scope, &ctDerivation, &rawSha, &storageKey, &snapshotURL)
	if err == pgx.ErrNoRows {
		return c, nil
	}
	if err != nil {
		return nil, err
	}

	c.Denominator = &denom
	c.DenominatorBasis = &DenominatorBasis{
		Scope: scope, DerivationQuery: ctDerivation, RawSha256: rawSha,
		StorageKey: storageKey, SnapshotURL: snapshotURL,
	}
	var ratio string
	if rerr := pool.QueryRow(ctx, `SELECT CASE WHEN $2::numeric=0 THEN NULL ELSE round($1::numeric/$2::numeric, 6)::text END`,
		c.Numerator, denom).Scan(&ratio); rerr == nil {
		c.Ratio = &ratio
	}
	return c, nil
}

// CompareRow is one jurisdiction's total for a scheme code.
type CompareRow struct {
	Jurisdiction string `json:"jurisdiction"`
	Amount       string `json:"amount"`
	FactCount    int64  `json:"factCount"`
}

// Compare totals a (scheme, code) across jurisdictions.
func Compare(ctx context.Context, pool *pgxpool.Pool, scheme, code string, jurisdictions []string) ([]CompareRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT f.jurisdiction, COALESCE(sum(f.amount),0)::numeric(24,4)::text, count(*)
		FROM fiscal_fact f
		JOIN classification_assignment a ON a.fact_id=f.fact_id AND a.scheme_id=$1 AND a.code=$2
		WHERE f.jurisdiction = ANY($3)
		  AND NOT EXISTS (SELECT 1 FROM fiscal_fact s WHERE s.supersedes=f.fact_id)
		GROUP BY f.jurisdiction ORDER BY f.jurisdiction`, scheme, code, jurisdictions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CompareRow{}
	for rows.Next() {
		var r CompareRow
		if err := rows.Scan(&r.Jurisdiction, &r.Amount, &r.FactCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Lead is a published lead (the only status ever exposed publicly).
type Lead struct {
	LeadID         string   `json:"leadId"`
	RuleID         string   `json:"ruleId"`
	FactIDs        []string `json:"factIds"`
	Score          *string  `json:"score"`
	GeneratedQuery string   `json:"generatedQuery"`
	ReviewNote     *string  `json:"reviewNote"`
}

// PublishedLeads returns only leads with status='published' (Hard Rule 6).
func PublishedLeads(ctx context.Context, pool *pgxpool.Pool) ([]Lead, error) {
	rows, err := pool.Query(ctx, `
		SELECT lead_id::text, rule_id, fact_ids::text[], score::text, generated_query, review_note
		FROM lead WHERE status='published' ORDER BY inserted_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Lead{}
	for rows.Next() {
		var l Lead
		if err := rows.Scan(&l.LeadID, &l.RuleID, &l.FactIDs, &l.Score, &l.GeneratedQuery, &l.ReviewNote); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Jurisdictions lists distinct jurisdictions present in facts.
func Jurisdictions(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	return scanStrings(ctx, pool, `SELECT DISTINCT jurisdiction FROM fiscal_fact ORDER BY 1`)
}

// Years lists distinct fiscal years for a jurisdiction, descending.
func Years(ctx context.Context, pool *pgxpool.Pool, jur string) ([]string, error) {
	return scanStrings(ctx, pool, `SELECT DISTINCT fiscal_year FROM fiscal_fact WHERE jurisdiction=$1 ORDER BY 1 DESC`, jur)
}

func scanStrings(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) ([]string, error) {
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// addDecimals adds two 4dp decimal strings without floats (integer minor-units math).
func addDecimals(a, b string) string {
	return formatMinor(parseMinor(a) + parseMinor(b))
}

func parseMinor(s string) int64 {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	intPart, frac := s, ""
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			intPart, frac = s[:i], s[i+1:]
			break
		}
	}
	for len(frac) < 4 {
		frac += "0"
	}
	frac = frac[:4]
	var n int64
	for _, c := range intPart {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	for _, c := range frac {
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n
}

func formatMinor(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	return fmt.Sprintf("%s%d.%04d", sign, n/10000, n%10000)
}
