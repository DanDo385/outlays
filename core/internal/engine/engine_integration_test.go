//go:build integration

package engine_test

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
	"reflect"
	"runtime"
	"testing"

	"github.com/djmagro/outlays/core/internal/api"
	"github.com/djmagro/outlays/core/internal/classify"
	"github.com/djmagro/outlays/core/internal/engine"
	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ownerURL = "postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable"
	appURL   = "postgres://app_login:app_pw@localhost:5433/fiscal?sslmode=disable"

	jur  = "us-ca"
	year = "2014-15"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// TestIntegrationEngineEquivalence is the S10 acceptance: export the partition as
// content-addressed Parquet, then prove the DuckDB path returns byte-identical view
// responses — vendor aggregation (scheme=payee) and, with S9 assignments live, the cofog
// rollup — through the same endpoint with no response-shape change.
func TestIntegrationEngineEquivalence(t *testing.T) {
	ctx := context.Background()
	owner, err := store.Connect(ctx, envOr("MIGRATE_DATABASE_URL", ownerURL))
	if err != nil {
		t.Skipf("Postgres not reachable (%v)", err)
	}
	defer owner.Close()
	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfig{
		Endpoint:  envOr("S3_ENDPOINT_HOSTPORT", "localhost:9000"),
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
	mappingPath := filepath.Join(root, "data", "cofog", "us-ca-procurement.json")
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
	}, year); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// COFOG assignments live (S9), so the cofog view is a real multi-node rollup.
	m, err := classify.LoadMapping(mappingPath, jur)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := classify.Apply(ctx, app, m, jur, year, "spending", false); err != nil {
		t.Fatalf("classify: %v", err)
	}

	// --- Export: content-addressed artifacts + registry row. ---
	res, err := engine.Export(ctx, app, obj, jur, year, t.TempDir())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(res.Artifacts) != 4 || res.ExportID == "" {
		t.Fatalf("export result incomplete: %+v", res)
	}
	rowsByName := map[string]int64{}
	for _, a := range res.Artifacts {
		rowsByName[a.Name] = a.Rows
		if ok, err := obj.Exists(ctx, a.Key); err != nil || !ok {
			t.Errorf("artifact %s not in object store (exists=%v err=%v)", a.Key, ok, err)
		}
		if a.Key != engine.ParquetKey(jur, year, a.Sha256) {
			t.Errorf("artifact key %s not content-addressed by its sha", a.Key)
		}
	}
	if rowsByName["facts"] != 989 {
		t.Errorf("facts artifact rows = %d, want 989", rowsByName["facts"])
	}

	// Re-export: identical content ⇒ identical hashes (deterministic write), new registry row.
	res2, err := engine.Export(ctx, app, obj, jur, year, t.TempDir())
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	for i := range res.Artifacts {
		if res.Artifacts[i].Sha256 != res2.Artifacts[i].Sha256 {
			t.Errorf("re-export %s sha changed: %s -> %s (non-deterministic parquet)",
				res.Artifacts[i].Name, res.Artifacts[i].Sha256, res2.Artifacts[i].Sha256)
		}
	}
	if res2.ExportID == res.ExportID {
		t.Error("re-export did not append a new registry row")
	}

	// --- Equivalence: same endpoint, both engines, byte-identical JSON. ---
	duck := &engine.Duck{Pool: app, Obj: obj, CacheDir: t.TempDir()}
	ts := httptest.NewServer((&api.Server{Pool: app, Duck: duck}).Router())
	defer ts.Close()

	for _, scheme := range []string{"payee", "cofog", "us_ca_department", "us_ca_acquisition_type"} {
		pgBody := getBody(t, ts.URL+"/v1/us-ca/2014-15/view?scheme="+scheme+"&flow=spending")
		duckBody := getBody(t, ts.URL+"/v1/us-ca/2014-15/view?scheme="+scheme+"&flow=spending&engine=duckdb")

		var pgView, duckView map[string]any
		if err := json.Unmarshal(pgBody, &pgView); err != nil {
			t.Fatalf("%s: pg decode: %v", scheme, err)
		}
		if err := json.Unmarshal(duckBody, &duckView); err != nil {
			t.Fatalf("%s: duck decode: %v", scheme, err)
		}
		if !reflect.DeepEqual(pgView, duckView) {
			t.Errorf("scheme %s: engines disagree\n  pg:   %s\n  duck: %s", scheme, pgBody, duckBody)
			continue
		}
		if string(pgBody) != string(duckBody) {
			t.Errorf("scheme %s: responses semantically equal but not byte-identical", scheme)
		}
		t.Logf("scheme %-22s total %v nodes %d — engines identical",
			scheme, pgView["total"], len(pgView["nodes"].([]any)))
	}

	// engine=duckdb on a partition with no export -> explicit 409, not a silent fallback.
	resp, err := http.Get(ts.URL + "/v1/us-ca/2013-14/view?scheme=payee&flow=spending&engine=duckdb")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("missing export -> %d, want 409", resp.StatusCode)
	}
}

func getBody(t *testing.T, url string) []byte {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s -> %d: %s", url, resp.StatusCode, body)
	}
	return body
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}
