// Package engine is the Parquet + DuckDB analytical path (task S10, D29). Export writes a
// (jurisdiction, fiscal_year) partition as content-addressed Parquet artifacts in object
// storage and registers them in the append-only parquet_export table; Duck serves the same
// view queries from those artifacts behind the read API's internal engine flag. Postgres
// remains the system of record (D9) — every artifact is a named snapshot of it.
package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/marcboeker/go-duckdb/v2" // registers the "duckdb" database/sql driver

	"github.com/djmagro/outlays/core/internal/store"
)

// ParquetKey is the object key for one exported Parquet artifact (content-addressed).
func ParquetKey(jurisdiction, fiscalYear, sha string) string {
	return fmt.Sprintf("parquet/%s/%s/%s.parquet", jurisdiction, fiscalYear, sha)
}

// Artifact describes one exported Parquet file.
type Artifact struct {
	Name   string `json:"name"` // facts | assignments | codes | entities
	Sha256 string `json:"sha256"`
	Key    string `json:"key"`
	Rows   int64  `json:"rows"`
	Bytes  int64  `json:"bytes"`
}

// ExportResult is the outcome of one partition export.
type ExportResult struct {
	Jurisdiction string     `json:"jurisdiction"`
	FiscalYear   string     `json:"fiscalYear"`
	ExportID     string     `json:"exportId"`
	Artifacts    []Artifact `json:"artifacts"`
}

// table holds rows read from Postgres on their way into DuckDB. All values travel as text
// (amounts stay exact decimal strings, Hard Rule 2) and are cast by DuckDB on insert.
type table struct {
	name   string
	ddl    string
	insert string
	rows   [][]any
}

// Export snapshots one partition: full fact history (including superseded rows, so the
// DuckDB query applies the same supersedes semantics as Postgres), every assignment
// version, the classification code labels, and the referenced entities. Files are written
// deterministically (stable ORDER BY), named by the sha256 of their bytes, uploaded, and
// registered in parquet_export.
func Export(ctx context.Context, pool *pgxpool.Pool, obj *store.ObjectStore, jur, year, dir string) (*ExportResult, error) {
	tables, err := readPartition(ctx, pool, jur, year)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	defer db.Close()

	res := &ExportResult{Jurisdiction: jur, FiscalYear: year}
	for _, t := range tables {
		path := filepath.Join(dir, t.name+".parquet")
		if err := writeParquet(ctx, db, t, path); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		sha := hex.EncodeToString(sum[:])
		key := ParquetKey(jur, year, sha)
		if err := obj.Put(ctx, key, data, "application/vnd.apache.parquet"); err != nil {
			return nil, err
		}
		res.Artifacts = append(res.Artifacts, Artifact{
			Name: t.name, Sha256: sha, Key: key, Rows: int64(len(t.rows)), Bytes: int64(len(data)),
		})
	}

	a := map[string]Artifact{}
	for _, art := range res.Artifacts {
		a[art.Name] = art
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO parquet_export (jurisdiction, fiscal_year,
			facts_sha256, facts_key, facts_rows,
			assignments_sha256, assignments_key, assignments_rows,
			codes_sha256, codes_key, codes_rows,
			entities_sha256, entities_key, entities_rows)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING export_id::text`,
		jur, year,
		a["facts"].Sha256, a["facts"].Key, a["facts"].Rows,
		a["assignments"].Sha256, a["assignments"].Key, a["assignments"].Rows,
		a["codes"].Sha256, a["codes"].Key, a["codes"].Rows,
		a["entities"].Sha256, a["entities"].Key, a["entities"].Rows,
	).Scan(&res.ExportID); err != nil {
		return nil, fmt.Errorf("register export: %w", err)
	}
	return res, nil
}

func readPartition(ctx context.Context, pool *pgxpool.Pool, jur, year string) ([]table, error) {
	facts := table{
		name: "facts",
		ddl: `CREATE TABLE facts (
			fact_id VARCHAR, jurisdiction VARCHAR, fiscal_year VARCHAR, flow VARCHAR,
			grain VARCHAR, payer_entity VARCHAR, payee_entity VARCHAR, amount DECIMAL(24,4),
			currency VARCHAR, occurred_on DATE, description VARCHAR, raw_sha256 VARCHAR,
			derivation_query VARCHAR, fact_hash VARCHAR, supersedes VARCHAR)`,
		insert: `INSERT INTO facts VALUES (?,?,?,?,?,?,?,CAST(? AS DECIMAL(24,4)),?,CAST(? AS DATE),?,?,?,?,?)`,
	}
	if err := scanRows(ctx, pool, &facts, 15, `
		SELECT fact_id::text, jurisdiction, fiscal_year, flow, grain,
		       payer_entity::text, payee_entity::text, amount::text, currency,
		       occurred_on::text, description, raw_sha256, derivation_query, fact_hash, supersedes::text
		FROM fiscal_fact WHERE jurisdiction=$1 AND fiscal_year=$2
		ORDER BY fact_hash`, jur, year); err != nil {
		return nil, err
	}

	assignments := table{
		name: "assignments",
		ddl: `CREATE TABLE assignments (
			assignment_id VARCHAR, fact_id VARCHAR, scheme_id VARCHAR, code VARCHAR,
			assigned_by VARCHAR, confidence VARCHAR, basis VARCHAR, version INTEGER)`,
		insert: `INSERT INTO assignments VALUES (?,?,?,?,?,?,?,?)`,
	}
	if err := scanRows(ctx, pool, &assignments, 8, `
		SELECT a.assignment_id::text, a.fact_id::text, a.scheme_id, a.code,
		       a.assigned_by, a.confidence::text, a.basis, a.version
		FROM classification_assignment a
		JOIN fiscal_fact f ON f.fact_id = a.fact_id
		WHERE f.jurisdiction=$1 AND f.fiscal_year=$2
		ORDER BY a.fact_id, a.scheme_id, a.version, a.code`, jur, year); err != nil {
		return nil, err
	}

	codes := table{
		name:   "codes",
		ddl:    `CREATE TABLE codes (scheme_id VARCHAR, code VARCHAR, parent_code VARCHAR, name VARCHAR)`,
		insert: `INSERT INTO codes VALUES (?,?,?,?)`,
	}
	if err := scanRows(ctx, pool, &codes, 4, `
		SELECT scheme_id, code, parent_code, name FROM classification_code
		ORDER BY scheme_id, code`); err != nil {
		return nil, err
	}

	entities := table{
		name:   "entities",
		ddl:    `CREATE TABLE entities (entity_id VARCHAR, kind VARCHAR, canonical_name VARCHAR)`,
		insert: `INSERT INTO entities VALUES (?,?,?)`,
	}
	if err := scanRows(ctx, pool, &entities, 3, `
		SELECT DISTINCT e.entity_id::text, e.kind, e.canonical_name
		FROM entity e
		JOIN fiscal_fact f ON e.entity_id = f.payer_entity OR e.entity_id = f.payee_entity
		WHERE f.jurisdiction=$1 AND f.fiscal_year=$2
		ORDER BY 1`, jur, year); err != nil {
		return nil, err
	}

	return []table{facts, assignments, codes, entities}, nil
}

func scanRows(ctx context.Context, pool *pgxpool.Pool, t *table, ncols int, q string, args ...any) error {
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("read %s: %w", t.name, err)
	}
	defer rows.Close()
	for rows.Next() {
		vals := make([]any, ncols)
		ptrs := make([]any, ncols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan %s: %w", t.name, err)
		}
		t.rows = append(t.rows, vals)
	}
	return rows.Err()
}

func writeParquet(ctx context.Context, db *sql.DB, t table, path string) error {
	if _, err := db.ExecContext(ctx, t.ddl); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, t.insert)
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, row := range t.rows {
		if _, err := stmt.ExecContext(ctx, row...); err != nil {
			stmt.Close()
			tx.Rollback()
			return fmt.Errorf("insert: %w", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		fmt.Sprintf(`COPY %s TO '%s' (FORMAT PARQUET)`, t.name, sqlEscape(path))); err != nil {
		return fmt.Errorf("copy to parquet: %w", err)
	}
	return nil
}

// sqlEscape escapes a string for inclusion in a single-quoted DuckDB literal (paths only —
// every other value travels as a bound parameter).
func sqlEscape(s string) string { return strings.ReplaceAll(s, "'", "''") }
