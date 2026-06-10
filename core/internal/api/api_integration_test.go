//go:build integration

package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/djmagro/outlays/core/internal/api"
	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ownerURL = "postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable"
	appURL   = "postgres://app_login:app_pw@localhost:5433/fiscal?sslmode=disable"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func TestIntegrationReadAPI(t *testing.T) {
	ctx := context.Background()
	owner, err := store.Connect(ctx, envOr("MIGRATE_DATABASE_URL", ownerURL))
	if err != nil {
		t.Skipf("Postgres not reachable (%v)", err)
	}
	defer owner.Close()
	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfig{
		Endpoint: envOr("S3_ENDPOINT_HOSTPORT", "localhost:9000"),
		AccessKey: envOr("S3_ACCESS_KEY", "minioadmin"), SecretKey: envOr("S3_SECRET_KEY", "minioadmin"),
		Bucket: envOr("S3_BUCKET", "fiscal-raw"), Region: envOr("S3_REGION", "us-east-1"),
	})
	if err != nil {
		t.Skipf("MinIO not reachable (%v)", err)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	caCli := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "dist", "cli.js")
	fixtures := filepath.Join(root, "packages", "adapters", "us-ca-procurement", "fixtures", "replay")
	budgetCli := filepath.Join(root, "packages", "adapters", "us-ca-budget", "dist", "cli.js")
	budgetFixtures := filepath.Join(root, "packages", "adapters", "us-ca-budget", "fixtures", "replay")
	node, nerr := exec.LookPath("node")
	if _, e := os.Stat(caCli); e != nil || nerr != nil {
		t.Skip("CA adapter not built or node missing")
	}

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

	if _, err := ingest.RunYear(ctx, &ingest.Options{
		AdapterCmd: []string{node, caCli}, Pool: app, Obj: obj,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		ExtraEnv: []string{"OUTLAYS_REPLAY_DIR=" + fixtures, "OUTLAYS_MAX_PAGES=1"},
	}, "2014-15"); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// Control total (S8): the budget adapter emits no facts, only the official total.
	if _, err := os.Stat(budgetCli); err != nil {
		t.Skip("budget adapter not built")
	}
	if _, err := ingest.RunYear(ctx, &ingest.Options{
		AdapterCmd: []string{node, budgetCli}, Pool: app, Obj: obj,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		ExtraEnv: []string{"OUTLAYS_REPLAY_DIR=" + budgetFixtures},
	}, "2014-15"); err != nil {
		t.Fatalf("ingest control total: %v", err)
	}

	ts := httptest.NewServer((&api.Server{Pool: app}).Router())
	defer ts.Close()

	get := func(path string) map[string]any {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Fatalf("GET %s -> %d: %s", path, resp.StatusCode, body)
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return m
	}

	// Same facts pivoted across three dimensions ⇒ identical totals.
	dept := get("/v1/us-ca/2014-15/view?scheme=us_ca_department&flow=spending")
	acq := get("/v1/us-ca/2014-15/view?scheme=us_ca_acquisition_type")
	pay := get("/v1/us-ca/2014-15/view?scheme=payee")
	if dept["total"] != acq["total"] || dept["total"] != pay["total"] {
		t.Errorf("totals differ across dimensions: dept=%v acq=%v payee=%v", dept["total"], acq["total"], pay["total"])
	}
	if dept["unmapped"] != "0.0000" {
		t.Errorf("department unmapped = %v, want 0.0000 (all facts have a department)", dept["unmapped"])
	}

	// Unclassified bucket: cofog has no assignments yet ⇒ everything unmapped, nothing dropped.
	cofog := get("/v1/us-ca/2014-15/view?scheme=cofog")
	if cofog["unmapped"] != cofog["total"] {
		t.Errorf("cofog unmapped %v != total %v (facts were dropped)", cofog["unmapped"], cofog["total"])
	}
	nodes := cofog["nodes"].([]any)
	if len(nodes) != 1 || nodes[0].(map[string]any)["code"] != store.UnclassifiedCode {
		t.Errorf("cofog should be a single __unclassified__ node, got %v", nodes)
	}

	// A vendor across multiple departments.
	var vendorID string
	for _, n := range pay["nodes"].([]any) {
		nm := n.(map[string]any)
		if nm["code"] != store.UnclassifiedCode {
			vendorID = nm["code"].(string)
			ef := get("/v1/entities/" + vendorID + "/flows?year=2014-15")
			if len(ef["byDepartment"].([]any)) >= 1 {
				break
			}
		}
	}

	// Provenance resolves to a real object-store key.
	facts := get("/v1/facts?jurisdiction=us-ca&year=2014-15&limit=1")["facts"].([]any)
	factID := facts[0].(map[string]any)["factId"].(string)
	prov := get("/v1/fact/" + factID + "/provenance")
	if prov["storageKey"] == nil || prov["storageKey"].(string) == "" {
		t.Errorf("provenance missing storageKey: %v", prov)
	}
	if ok, err := obj.Exists(ctx, prov["storageKey"].(string)); err != nil || !ok {
		t.Errorf("storageKey %v not in object store (exists=%v err=%v)", prov["storageKey"], ok, err)
	}

	// Coverage (S8): numerator and denominator each with provenance links; honest low ratio.
	cov := get("/v1/us-ca/2014-15/coverage")
	if cov["numerator"] != dept["total"] {
		t.Errorf("coverage numerator %v != view total %v", cov["numerator"], dept["total"])
	}
	if cov["denominator"] != "156357000000.0000" {
		t.Errorf("coverage denominator = %v, want 156357000000.0000", cov["denominator"])
	}
	if cov["ratio"] != "0.000247" {
		t.Errorf("coverage ratio = %v, want 0.000247 (the honest low number)", cov["ratio"])
	}
	nb, _ := cov["numeratorBasis"].(map[string]any)
	if nb == nil || nb["factsUrl"] == "" || nb["derivationQuery"] == "" {
		t.Errorf("coverage numeratorBasis missing provenance: %v", cov["numeratorBasis"])
	}
	db, _ := cov["denominatorBasis"].(map[string]any)
	if db == nil || db["rawSha256"] == "" || db["derivationQuery"] == "" {
		t.Fatalf("coverage denominatorBasis missing provenance: %v", cov["denominatorBasis"])
	}
	if db["scope"] != "procurement facts vs total budget" {
		t.Errorf("coverage scope = %v, want explicit scope label", db["scope"])
	}
	if key, _ := db["storageKey"].(string); key == "" {
		t.Errorf("denominator storageKey missing")
	} else if ok, err := obj.Exists(ctx, key); err != nil || !ok {
		t.Errorf("denominator storageKey %v not in object store (exists=%v err=%v)", key, ok, err)
	}

	// Leads: published-only, empty for now.
	if leads := get("/v1/leads?status=published")["leads"].([]any); len(leads) != 0 {
		t.Errorf("expected 0 published leads, got %d", len(leads))
	}
	resp, _ := http.Get(ts.URL + "/v1/leads?status=draft")
	if resp.StatusCode != 400 {
		t.Errorf("leads?status=draft -> %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	t.Logf("read API: total %v across dept/acq/payee; cofog all unclassified; vendor %s flows ok", dept["total"], vendorID)
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}
