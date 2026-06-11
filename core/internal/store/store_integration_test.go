//go:build integration

package store_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration test for S4. Requires the compose stack (Postgres + MinIO) and the built CA
// adapter + fixtures. Skips cleanly when any of those is unavailable so `go test ./...` stays
// green without infra. Run with the stack up:
//
//	docker compose -f deploy/docker-compose.yml up -d
//	( cd core && go test ./internal/store/ -run Integration -v )

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

const (
	defOwnerURL = "postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable"
	appPassword = "app_pw"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "pnpm-workspace.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found")
		}
		dir = parent
	}
}

// dataTables and a column on each, used to assert UPDATE/DELETE both raise.
var dataTables = map[string]string{
	"ingestion_run":             "status",
	"raw_snapshot":              "url",
	"entity":                    "canonical_name",
	"entity_alias":              "name_raw",
	"fiscal_fact":               "derivation_query",
	"classification_scheme":     "name",
	"classification_code":       "name",
	"classification_assignment": "version",
	"control_total":             "derivation_query",
	"lead":                      "rule_id",
	"parquet_export":            "facts_key",
}

func TestIntegrationIngestAndAppendOnly(t *testing.T) {
	ctx := context.Background()
	ownerURL := env("MIGRATE_DATABASE_URL", defOwnerURL)

	owner, err := store.Connect(ctx, ownerURL)
	if err != nil {
		t.Skipf("Postgres not reachable (%v); skipping integration test", err)
	}
	defer owner.Close()

	// Object storage.
	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfig{
		Endpoint:  env("S3_ENDPOINT_HOSTPORT", "localhost:9000"),
		AccessKey: env("S3_ACCESS_KEY", "minioadmin"),
		SecretKey: env("S3_SECRET_KEY", "minioadmin"),
		Bucket:    env("S3_BUCKET", "fiscal-raw"),
		Region:    env("S3_REGION", "us-east-1"),
	})
	if err != nil {
		t.Skipf("MinIO not reachable (%v); skipping", err)
	}

	// Build the S3 output by running the CA adapter in replay mode.
	root := repoRoot(t)
	cli := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "dist", "cli.js")
	fixtures := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "fixtures", "replay")
	node, nerr := exec.LookPath("node")
	if _, e := os.Stat(cli); e != nil || nerr != nil {
		t.Skip("CA adapter not built or node missing")
	}
	work := t.TempDir()
	outPath := filepath.Join(work, "out.json")
	rawDir := filepath.Join(work, "raw")
	cmd := exec.Command(node, cli, "fetch", "--year", "2014-15", "--raw-dir", rawDir, "--out", outPath)
	cmd.Env = append(os.Environ(), "OUTLAYS_REPLAY_DIR="+fixtures, "OUTLAYS_MAX_PAGES=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("adapter fetch failed: %v\n%s", err, out)
	}

	// Reset to a clean schema so the test is hermetic and repeatable (append-only blocks
	// TRUNCATE/DELETE, so we drop+recreate as owner). This is a throwaway compose DB.
	mustExec(t, ctx, owner, `DROP SCHEMA public CASCADE`)
	mustExec(t, ctx, owner, `CREATE SCHEMA public`)

	// Migrate (owner) and ensure the app login role exists with app_rw membership.
	if err := store.Migrate(ownerURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mustExec(t, ctx, owner, `DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='app_login') THEN CREATE ROLE app_login LOGIN PASSWORD '`+appPassword+`'; END IF; END $$;`)
	mustExec(t, ctx, owner, `ALTER ROLE app_login WITH LOGIN PASSWORD '`+appPassword+`'`)
	mustExec(t, ctx, owner, `GRANT app_rw TO app_login`)

	// Connect as the app (member of app_rw: SELECT/INSERT only) and ingest.
	appURL := env("APP_DATABASE_URL", "postgres://app_login:"+appPassword+"@localhost:5433/fiscal?sslmode=disable")
	app, err := store.Connect(ctx, appURL)
	if err != nil {
		t.Fatalf("connect app: %v", err)
	}
	defer app.Close()

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := store.ParseDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	res, err := store.Ingest(ctx, app, obj, doc, rawDir, "purchase-order-data")
	if err != nil {
		t.Fatalf("ingest (as app_rw member): %v", err)
	}
	t.Logf("ingested run=%s facts=%d entities=%d aliases=%d assignments=%d snapshots=%d",
		res.RunID, res.Facts, res.Entities, res.Aliases, res.Assignments, res.Snapshots)

	// (a) Row counts match the document.
	var factCount int
	if err := app.QueryRow(ctx, `SELECT count(*) FROM fiscal_fact WHERE run_id=$1`, res.RunID).Scan(&factCount); err != nil {
		t.Fatal(err)
	}
	if factCount != len(doc.Facts) {
		t.Errorf("fiscal_fact count = %d, want %d", factCount, len(doc.Facts))
	}

	// (b) Object store has the raw snapshot.
	key := store.RawKey(doc.Envelope.Jurisdiction, "purchase-order-data", doc.Envelope.FiscalYear, doc.Envelope.RawSnapshots[0].Sha256)
	if ok, err := obj.Exists(ctx, key); err != nil || !ok {
		t.Errorf("object %s exists=%v err=%v", key, ok, err)
	}

	// Ensure control_total and lead have a row so their triggers can be exercised.
	seedWorkflowRows(t, ctx, owner, res.RunID, doc.Envelope.RawSnapshots[0].Sha256, doc.Facts[0].FactHash)

	// (c) UPDATE and DELETE raise on every data table (the trigger, via owner).
	for table, col := range dataTables {
		if _, err := owner.Exec(ctx, `UPDATE `+table+` SET `+col+` = `+col); err == nil {
			t.Errorf("UPDATE %s did not raise (append-only trigger missing)", table)
		}
		if _, err := owner.Exec(ctx, `DELETE FROM `+table); err == nil {
			t.Errorf("DELETE %s did not raise (append-only trigger missing)", table)
		}
	}

	// (d) The app role lacks UPDATE/DELETE privilege (REVOKE layer).
	if _, err := app.Exec(ctx, `UPDATE fiscal_fact SET description = description`); err == nil {
		t.Error("app role UPDATE fiscal_fact did not raise (REVOKE missing)")
	}

	// (e) A correction chains via supersedes (a new append-only row).
	var origID, origRun, origRaw string
	if err := owner.QueryRow(ctx,
		`SELECT fact_id, run_id, raw_sha256 FROM fiscal_fact WHERE run_id=$1 LIMIT 1`, res.RunID,
	).Scan(&origID, &origRun, &origRaw); err != nil {
		t.Fatal(err)
	}
	correctedHash := "correction-of-" + origID
	correctedID := store.FactID(correctedHash)
	if _, err := app.Exec(ctx, `
		INSERT INTO fiscal_fact (fact_id, run_id, jurisdiction, fiscal_year, flow, grain, amount,
		                         currency, derivation_query, fact_hash, raw_sha256, supersedes)
		VALUES ($1,$2,'us-ca','2014-15','spending','award',$3,'USD',$4,$5,$6,$7)
		ON CONFLICT (fact_id) DO NOTHING`,
		correctedID, origRun, "0.0000", "correction; supersedes "+origID, correctedHash, origRaw, origID,
	); err != nil {
		t.Fatalf("insert correction (as app): %v", err)
	}
	var chained string
	if err := owner.QueryRow(ctx, `SELECT supersedes FROM fiscal_fact WHERE fact_id=$1`, correctedID).Scan(&chained); err != nil {
		t.Fatal(err)
	}
	if chained != origID {
		t.Errorf("correction supersedes = %s, want %s", chained, origID)
	}
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func seedWorkflowRows(t *testing.T, ctx context.Context, owner *pgxpool.Pool, runID, sha, factHash string) {
	t.Helper()
	if _, err := owner.Exec(ctx, `
		INSERT INTO control_total (jurisdiction, fiscal_year, flow, official_total, currency, scope, raw_sha256, derivation_query)
		VALUES ('us-ca','2014-15','spending','1000000.0000','USD','seed scope',$1,'seed for trigger test')
		ON CONFLICT (jurisdiction, fiscal_year, flow) DO NOTHING`, sha); err != nil {
		t.Fatalf("seed control_total: %v", err)
	}
	factID := store.FactID(factHash)
	_, _ = owner.Exec(ctx, `
		INSERT INTO lead (rule_id, fact_ids, generated_query, status)
		VALUES ('seed-rule', ARRAY[$1::uuid], 'seed for trigger test', 'draft')`, factID)
	if _, err := owner.Exec(ctx, `
		INSERT INTO parquet_export (jurisdiction, fiscal_year,
			facts_sha256, facts_key, facts_rows,
			assignments_sha256, assignments_key, assignments_rows,
			codes_sha256, codes_key, codes_rows,
			entities_sha256, entities_key, entities_rows)
		VALUES ('us-ca','2014-15','s','k',0,'s','k',0,'s','k',0,'s','k',0)`); err != nil {
		t.Fatalf("seed parquet_export: %v", err)
	}
}
