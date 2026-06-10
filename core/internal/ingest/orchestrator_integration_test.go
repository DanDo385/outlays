//go:build integration

package ingest_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration test for S5. Requires the compose stack + built adapters + node; skips cleanly
// otherwise. Run with the stack up:
//   ( cd core && go test ./internal/ingest/ -run Integration -v )

const (
	ownerURL = "postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable"
	appURL   = "postgres://app_login:app_pw@localhost:5433/fiscal?sslmode=disable"
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

func TestIntegrationOrchestrator(t *testing.T) {
	ctx := context.Background()

	owner, err := store.Connect(ctx, envOr("MIGRATE_DATABASE_URL", ownerURL))
	if err != nil {
		t.Skipf("Postgres not reachable (%v); skipping", err)
	}
	defer owner.Close()

	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfig{
		Endpoint: envOr("S3_ENDPOINT_HOSTPORT", "localhost:9000"),
		AccessKey: envOr("S3_ACCESS_KEY", "minioadmin"), SecretKey: envOr("S3_SECRET_KEY", "minioadmin"),
		Bucket: envOr("S3_BUCKET", "fiscal-raw"), Region: envOr("S3_REGION", "us-east-1"),
	})
	if err != nil {
		t.Skipf("MinIO not reachable (%v); skipping", err)
	}

	root := repoRoot(t)
	caCli := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "dist", "cli.js")
	toyCli := filepath.Join(root, "packages", "adapters", "toy-fixture", "dist", "cli.js")
	fixtures := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "fixtures", "replay")
	node, nerr := exec.LookPath("node")
	if _, e := os.Stat(caCli); e != nil || nerr != nil {
		t.Skip("adapters not built or node missing")
	}

	// Fresh schema + app role.
	mustExec(t, ctx, owner, `DROP SCHEMA public CASCADE`)
	mustExec(t, ctx, owner, `CREATE SCHEMA public`)
	if err := store.Migrate(envOr("MIGRATE_DATABASE_URL", ownerURL)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mustExec(t, ctx, owner, `DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='app_login') THEN CREATE ROLE app_login LOGIN PASSWORD 'app_pw'; END IF; END $$;`)
	mustExec(t, ctx, owner, `ALTER ROLE app_login WITH LOGIN PASSWORD 'app_pw'`)
	mustExec(t, ctx, owner, `GRANT app_rw TO app_login`)

	app, err := store.Connect(ctx, envOr("APP_DATABASE_URL", appURL))
	if err != nil {
		t.Fatalf("connect app: %v", err)
	}
	defer app.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// (1) Success: CA adapter in replay mode ingests 2014-15.
	caOpts := &ingest.Options{
		AdapterCmd: []string{node, caCli}, Pool: app, Obj: obj, Logger: log,
		ExtraEnv: []string{"OUTLAYS_REPLAY_DIR=" + fixtures, "OUTLAYS_MAX_PAGES=1"},
	}
	outcomes, err := ingest.Backfill(ctx, caOpts, []string{"2014-15"}, 2)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Status != "succeeded" || outcomes[0].Facts == 0 {
		t.Fatalf("CA outcome = %+v, want succeeded with facts", outcomes)
	}
	if s, _ := store.RunStatus(ctx, app, outcomes[0].RunID); s != "succeeded" {
		t.Errorf("ingestion_run status = %q, want succeeded", s)
	}
	t.Logf("success run %s ingested %d facts", outcomes[0].RunID, outcomes[0].Facts)

	// (2) Forced failure (adapter fetch exits 1).
	failOutcome, err := ingest.RunYear(ctx, &ingest.Options{
		AdapterCmd: []string{node, toyCli}, Pool: app, Obj: obj, Logger: log,
		ExtraEnv: []string{"OUTLAYS_TEST_FAIL=1"},
	}, "2024-25")
	if err != nil {
		t.Fatalf("forced-failure run returned infra error: %v", err)
	}
	if failOutcome.Status != "failed" {
		t.Errorf("forced-failure outcome status = %q, want failed", failOutcome.Status)
	}
	if s, _ := store.RunStatus(ctx, app, failOutcome.RunID); s != "failed" {
		t.Errorf("forced-failure ingestion_run status = %q, want failed", s)
	}

	// (3) Exit code 2 (source unavailable / restricted).
	ex2Outcome, err := ingest.RunYear(ctx, &ingest.Options{
		AdapterCmd: []string{node, toyCli}, Pool: app, Obj: obj, Logger: log,
		ExtraEnv: []string{"OUTLAYS_TEST_FAIL=2"},
	}, "2024-25")
	if err != nil {
		t.Fatalf("exit-2 run returned infra error: %v", err)
	}
	if ex2Outcome.Status != "failed" {
		t.Errorf("exit-2 outcome status = %q, want failed", ex2Outcome.Status)
	}
	var exitCode int
	if err := app.QueryRow(ctx,
		`SELECT (envelope->>'exitCode')::int FROM ingestion_run WHERE run_id=$1`, ex2Outcome.RunID,
	).Scan(&exitCode); err != nil {
		t.Fatal(err)
	}
	if exitCode != 2 {
		t.Errorf("exit-2 run recorded exitCode = %d, want 2", exitCode)
	}
	t.Logf("forced-failure run %s and exit-2 run %s recorded as failed", failOutcome.RunID, ex2Outcome.RunID)
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
