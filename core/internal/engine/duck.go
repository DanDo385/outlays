package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/djmagro/outlays/core/internal/store"
)

// Duck serves view queries from the latest registered Parquet export of a partition.
// Postgres is consulted only for the parquet_export registry lookup; the aggregation itself
// runs in DuckDB over content-addressed local copies of the artifacts.
type Duck struct {
	Pool     *pgxpool.Pool
	Obj      *store.ObjectStore
	CacheDir string
}

// ErrNoExport is returned when a partition has no registered Parquet export yet.
var ErrNoExport = fmt.Errorf("no parquet export registered for partition")

// partitionFiles are the local artifact paths for one export.
type partitionFiles struct {
	facts, assignments, codes, entities string
}

// ViewByScheme mirrors store.ViewByScheme on the DuckDB engine (same shape, same semantics).
func (d *Duck) ViewByScheme(ctx context.Context, jur, year, flow, scheme string) (*store.View, error) {
	pf, err := d.partition(ctx, jur, year)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	v := &store.View{Jurisdiction: jur, FiscalYear: year, Flow: flow, SchemeID: scheme, Currency: "USD", Nodes: []store.Node{}}
	return scanDuckView(ctx, db, v, schemeViewSQL(pf), jur, year, flow, scheme, scheme)
}

// ViewByPayee mirrors store.ViewByPayee on the DuckDB engine.
func (d *Duck) ViewByPayee(ctx context.Context, jur, year, flow string) (*store.View, error) {
	pf, err := d.partition(ctx, jur, year)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()
	v := &store.View{Jurisdiction: jur, FiscalYear: year, Flow: flow, SchemeID: "payee", Currency: "USD", Nodes: []store.Node{}}
	return scanDuckView(ctx, db, v, payeeViewSQL(pf), jur, year, flow)
}

// partition resolves the latest export for (jur, year) and ensures local cached copies,
// verified against their content hashes.
func (d *Duck) partition(ctx context.Context, jur, year string) (*partitionFiles, error) {
	var shas, keys [4]string
	err := d.Pool.QueryRow(ctx, `
		SELECT facts_sha256, facts_key, assignments_sha256, assignments_key,
		       codes_sha256, codes_key, entities_sha256, entities_key
		FROM parquet_export WHERE jurisdiction=$1 AND fiscal_year=$2
		ORDER BY exported_at DESC, export_id LIMIT 1`, jur, year,
	).Scan(&shas[0], &keys[0], &shas[1], &keys[1], &shas[2], &keys[2], &shas[3], &keys[3])
	if err == pgx.ErrNoRows {
		return nil, ErrNoExport
	}
	if err != nil {
		return nil, fmt.Errorf("lookup parquet export: %w", err)
	}

	var paths [4]string
	for i := range shas {
		p, err := d.ensureCached(ctx, shas[i], keys[i])
		if err != nil {
			return nil, err
		}
		paths[i] = p
	}
	return &partitionFiles{facts: paths[0], assignments: paths[1], codes: paths[2], entities: paths[3]}, nil
}

// ensureCached downloads an artifact to CacheDir/<sha256>.parquet once, verifying its hash.
func (d *Duck) ensureCached(ctx context.Context, sha, key string) (string, error) {
	if err := os.MkdirAll(d.CacheDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(d.CacheDir, sha+".parquet")
	if _, err := os.Stat(path); err == nil {
		return path, nil // content-addressed: present means verified at write time
	}
	data, err := d.Obj.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", key, err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != sha {
		return "", fmt.Errorf("artifact %s hash mismatch: got %s", key, got)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// schemeViewSQL is store.ViewByScheme translated to DuckDB over Parquet: identical
// semantics (current facts, latest-version assignment de-dup, explicit unclassified
// bucket) and identical deterministic ordering.
func schemeViewSQL(pf *partitionFiles) string {
	return `
	WITH fy AS (
		SELECT f.fact_id, f.amount
		FROM read_parquet('` + sqlEscape(pf.facts) + `') f
		WHERE f.jurisdiction = ? AND f.fiscal_year = ? AND f.flow = ?
		  AND NOT EXISTS (SELECT 1 FROM read_parquet('` + sqlEscape(pf.facts) + `') s WHERE s.supersedes = f.fact_id)
	),
	asg AS (
		SELECT DISTINCT ON (a.fact_id) a.fact_id, a.code
		FROM read_parquet('` + sqlEscape(pf.assignments) + `') a
		JOIN fy ON fy.fact_id = a.fact_id
		WHERE a.scheme_id = ?
		ORDER BY a.fact_id, a.version DESC, a.code
	)
	SELECT
		COALESCE(asg.code, '` + store.UnclassifiedCode + `') AS code,
		COALESCE(cc.name, CASE WHEN asg.code IS NULL THEN 'Unclassified' ELSE asg.code END) AS label,
		count(*) AS fact_count,
		CAST(CAST(COALESCE(sum(fy.amount), 0) AS DECIMAL(24,4)) AS VARCHAR) AS amount
	FROM fy
	LEFT JOIN asg ON asg.fact_id = fy.fact_id
	LEFT JOIN read_parquet('` + sqlEscape(pf.codes) + `') cc ON cc.scheme_id = ? AND cc.code = asg.code
	GROUP BY 1, 2
	ORDER BY sum(fy.amount) DESC NULLS LAST, 1`
}

// payeeViewSQL is store.ViewByPayee translated to DuckDB over Parquet.
func payeeViewSQL(pf *partitionFiles) string {
	return `
	WITH fy AS (
		SELECT f.fact_id, f.amount, f.payee_entity
		FROM read_parquet('` + sqlEscape(pf.facts) + `') f
		WHERE f.jurisdiction = ? AND f.fiscal_year = ? AND f.flow = ?
		  AND NOT EXISTS (SELECT 1 FROM read_parquet('` + sqlEscape(pf.facts) + `') s WHERE s.supersedes = f.fact_id)
	)
	SELECT
		COALESCE(e.entity_id, '` + store.UnclassifiedCode + `') AS code,
		COALESCE(e.canonical_name, 'Unclassified') AS label,
		count(*) AS fact_count,
		CAST(CAST(COALESCE(sum(fy.amount), 0) AS DECIMAL(24,4)) AS VARCHAR) AS amount
	FROM fy
	LEFT JOIN read_parquet('` + sqlEscape(pf.entities) + `') e ON e.entity_id = fy.payee_entity
	GROUP BY 1, 2
	ORDER BY sum(fy.amount) DESC NULLS LAST, 1`
}

// scanDuckView runs a view query and assembles the same store.View the Postgres path
// produces; amounts are normalized to 4dp decimal strings via exact minor-units math.
func scanDuckView(ctx context.Context, db *sql.DB, v *store.View, q string, args ...any) (*store.View, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdb view: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n store.Node
		if err := rows.Scan(&n.Code, &n.Label, &n.FactCount, &n.Amount); err != nil {
			return nil, err
		}
		n.Amount = store.AddDecimals(n.Amount, "0.0000") // normalize to exactly 4dp
		n.Currency = "USD"
		v.Nodes = append(v.Nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	total, unmapped := "0.0000", "0.0000"
	for _, n := range v.Nodes {
		total = store.AddDecimals(total, n.Amount)
		if n.Code == store.UnclassifiedCode {
			unmapped = n.Amount
		}
	}
	v.Total = total
	v.Unmapped = unmapped
	return v, nil
}
