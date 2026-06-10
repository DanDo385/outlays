//go:build integration

package classify_test

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
	"github.com/djmagro/outlays/core/internal/classify"
	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ownerURL = "postgres://fiscal_owner:change_me_too@localhost:5433/fiscal?sslmode=disable"
	appURL   = "postgres://app_login:app_pw@localhost:5433/fiscal?sslmode=disable"

	jur  = "us-ca"
	year = "2014-15"
	flow = "spending"
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// TestIntegrationClassify ingests the recorded CA replay fixture, applies the real reviewed
// COFOG mapping, and verifies S9 acceptance: versioned assignment rows with preserved
// confidence + basis, idempotent re-runs, version bumps on entry change, human precedence,
// the conflict guard, an exactly-reconciling cofog view, and the explicit unmapped listing.
func TestIntegrationClassify(t *testing.T) {
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

	m, err := classify.LoadMapping(mappingPath, jur)
	if err != nil {
		t.Fatalf("load mapping: %v", err)
	}

	// --- Dry run plans but writes nothing. ---
	dry, err := classify.Apply(ctx, app, m, jur, year, flow, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if dry.Inserted == 0 {
		t.Fatal("dry run planned 0 assignments; expected department-mapped facts")
	}
	if n := cofogRows(t, ctx, app); n != 0 {
		t.Fatalf("dry run wrote %d rows", n)
	}

	// --- First real apply. ---
	r1, err := classify.Apply(ctx, app, m, jur, year, flow, false)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if r1.Inserted != dry.Inserted {
		t.Errorf("apply inserted %d, dry run planned %d", r1.Inserted, dry.Inserted)
	}
	if len(r1.Conflicts) != 0 {
		t.Errorf("unexpected conflicts: %+v", r1.Conflicts)
	}
	if n := cofogRows(t, ctx, app); n != int64(r1.Inserted) {
		t.Errorf("cofog rows = %d, want %d", n, r1.Inserted)
	}

	// Expected count derived independently: facts whose department is a mapped entry.
	var mappedDepts []string
	for _, c := range m.Categories {
		if c.Mapped() && c.SchemeID == "us_ca_department" {
			mappedDepts = append(mappedDepts, c.Code)
		}
	}
	var want int64
	if err := app.QueryRow(ctx, `
		SELECT count(DISTINCT a.fact_id) FROM classification_assignment a
		JOIN fiscal_fact f ON f.fact_id = a.fact_id
		WHERE a.scheme_id='us_ca_department' AND a.code = ANY($1)
		  AND f.jurisdiction=$2 AND f.fiscal_year=$3 AND f.flow=$4`,
		mappedDepts, jur, year, flow).Scan(&want); err != nil {
		t.Fatal(err)
	}
	if int64(r1.Inserted) != want {
		t.Errorf("inserted %d, want %d (facts with a mapped department)", r1.Inserted, want)
	}

	// Rows are rule-assigned, version 1, with translated confidence + canonical basis.
	var badBy, badVer int64
	if err := app.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE assigned_by <> 'rule'),
		       count(*) FILTER (WHERE version <> 1)
		FROM classification_assignment WHERE scheme_id='cofog'`).Scan(&badBy, &badVer); err != nil {
		t.Fatal(err)
	}
	if badBy != 0 || badVer != 0 {
		t.Errorf("badAssignedBy=%d badVersion=%d", badBy, badVer)
	}
	var conf, basis string
	if err := app.QueryRow(ctx, `
		SELECT a.confidence::text, a.basis FROM classification_assignment a
		JOIN classification_assignment d ON d.fact_id = a.fact_id
		WHERE a.scheme_id='cofog' AND d.scheme_id='us_ca_department'
		  AND d.code='Franchise Tax Board' LIMIT 1`).Scan(&conf, &basis); err != nil {
		t.Fatalf("no cofog row for an FTB fact: %v", err)
	}
	if conf != "0.5" {
		t.Errorf("FTB confidence = %s, want 0.5 (medium)", conf)
	}
	var b map[string]string
	if err := json.Unmarshal([]byte(basis), &b); err != nil {
		t.Fatalf("basis not JSON: %v", err)
	}
	if b["ruleId"] != "cofog-map/us-ca-procurement" || b["confidence"] != "medium" ||
		b["sourceCategory"] != "department: Franchise Tax Board" || b["citation"] == "" || len(b["entrySha256"]) != 64 {
		t.Errorf("basis fields wrong: %s", basis)
	}

	// Reconciliation is exact: mapped + unclassified == total, zero dropped facts.
	rec := r1.Reconciliation
	if !rec.Reconciles {
		t.Errorf("view does not reconcile: %+v", rec)
	}
	if rec.FactsTotal != 989 {
		t.Errorf("factsTotal = %d, want 989 (replay fixture)", rec.FactsTotal)
	}
	if rec.FactsMapped != int64(r1.Inserted) || rec.FactsMapped+rec.FactsUnclassified != rec.FactsTotal {
		t.Errorf("fact counts do not reconcile: %+v", rec)
	}
	if store.AddDecimals(rec.Mapped, rec.Unclassified) != rec.Total {
		t.Errorf("amounts do not reconcile: %+v", rec)
	}

	// Unmapped categories listed explicitly: all five acquisition types + HHS Agency are
	// reviewed-unmapped by design; departments outside the reviewed file are unreviewed.
	got := map[string]string{}
	for _, u := range r1.UnmappedCategories {
		got[u.SchemeID+": "+u.Category] = u.Status
		if u.FactCount == 0 || u.Amount == "" {
			t.Errorf("unmapped category %s missing count/amount", u.Category)
		}
	}
	for _, acq := range []string{"NON-IT Goods", "IT Goods", "NON-IT Services", "IT Services", "IT Telecommunications"} {
		if got["us_ca_acquisition_type: "+acq] != "reviewed_unmapped" {
			t.Errorf("acquisition type %q not listed as reviewed_unmapped (got %q)", acq, got["us_ca_acquisition_type: "+acq])
		}
	}
	if got["us_ca_department: Health & Human Services Agency"] != "reviewed_unmapped" {
		t.Errorf("HHS Agency not reviewed_unmapped: %q", got["us_ca_department: Health & Human Services Agency"])
	}
	unreviewed := 0
	for _, u := range r1.UnmappedCategories {
		if u.Status == "unreviewed" {
			unreviewed++
		}
	}
	if unreviewed == 0 {
		t.Error("expected unreviewed departments beyond the reviewed top-25")
	}

	// --- Idempotency: re-applying the same mapping inserts nothing. ---
	r2, err := classify.Apply(ctx, app, m, jur, year, flow, false)
	if err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if r2.Inserted != 0 || r2.UpToDate != r1.Inserted {
		t.Errorf("re-apply inserted=%d upToDate=%d, want 0/%d", r2.Inserted, r2.UpToDate, r1.Inserted)
	}

	// --- Versioning: an entry change appends version 2 for that category's facts only. ---
	modPath := modifiedMapping(t, mappingPath, "department: Transportation, Department of",
		func(e map[string]string) { e["confidence"] = "high" })
	mod, err := classify.LoadMapping(modPath, jur)
	if err != nil {
		t.Fatal(err)
	}
	var transpoFacts int64
	if err := app.QueryRow(ctx, `
		SELECT count(DISTINCT fact_id) FROM classification_assignment
		WHERE scheme_id='us_ca_department' AND code='Transportation, Department of'`).Scan(&transpoFacts); err != nil {
		t.Fatal(err)
	}
	r3, err := classify.Apply(ctx, app, mod, jur, year, flow, false)
	if err != nil {
		t.Fatalf("apply modified: %v", err)
	}
	if int64(r3.Inserted) != transpoFacts {
		t.Errorf("modified apply inserted %d, want %d (Transportation facts)", r3.Inserted, transpoFacts)
	}
	var v2 int64
	var v2conf string
	if err := app.QueryRow(ctx, `
		SELECT count(*), min(confidence::text) FROM classification_assignment
		WHERE scheme_id='cofog' AND version=2`).Scan(&v2, &v2conf); err != nil {
		t.Fatal(err)
	}
	if v2 != transpoFacts || v2conf != "0.75" {
		t.Errorf("version-2 rows=%d conf=%s, want %d/0.75 (high)", v2, v2conf, transpoFacts)
	}
	if !r3.Reconciliation.Reconciles {
		t.Errorf("view does not reconcile after version bump: %+v", r3.Reconciliation)
	}

	// --- Human precedence: a later human assignment is never overridden by the rule. ---
	var humanFact string
	if err := app.QueryRow(ctx, `
		SELECT fact_id::text FROM classification_assignment
		WHERE scheme_id='us_ca_department' AND code='Transportation, Department of'
		ORDER BY fact_id LIMIT 1`).Scan(&humanFact); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Exec(ctx, `
		INSERT INTO classification_assignment
			(assignment_id, fact_id, scheme_id, code, assigned_by, basis, version)
		VALUES ($1, $2, 'cofog', '09', 'human', 'reviewer:integration-test', 99)`,
		store.AssignmentID(humanFact, "cofog", "09", 99), humanFact); err != nil {
		t.Fatalf("insert human assignment: %v", err)
	}
	r4, err := classify.Apply(ctx, app, mod, jur, year, flow, false)
	if err != nil {
		t.Fatalf("apply over human: %v", err)
	}
	if r4.Inserted != 0 || r4.SkippedHumanOverride != 1 {
		t.Errorf("apply over human: inserted=%d skippedHuman=%d, want 0/1", r4.Inserted, r4.SkippedHumanOverride)
	}

	// --- Conflict guard: disagreeing mapped categories assign nothing (dry run). ---
	conflictPath := writeJSON(t, map[string]map[string]string{
		"department: Transportation, Department of": {
			"cofogCode": "04", "confidence": "medium", "basis": "docs/cofog-references.md#reference-1--eurostat-cofog-manual"},
		"acquisition_type: NON-IT Goods": {
			"cofogCode": "01", "confidence": "low", "basis": "docs/cofog-references.md#mapping-policy"},
	})
	conflictMap, err := classify.LoadMapping(conflictPath, jur)
	if err != nil {
		t.Fatal(err)
	}
	before := cofogRows(t, ctx, app)
	rc, err := classify.Apply(ctx, app, conflictMap, jur, year, flow, true)
	if err != nil {
		t.Fatalf("conflict dry run: %v", err)
	}
	if len(rc.Conflicts) == 0 || rc.Conflicts[0].FactCount == 0 {
		t.Errorf("expected conflicts for facts with disagreeing dept/acq codes, got %+v", rc.Conflicts)
	}
	if after := cofogRows(t, ctx, app); after != before {
		t.Errorf("conflict dry run wrote rows: %d -> %d", before, after)
	}

	// --- ACCEPT: the cofog view endpoint returns a real rollup, reconciling exactly. ---
	ts := httptest.NewServer((&api.Server{Pool: app}).Router())
	defer ts.Close()
	cofog := getJSON(t, ts.URL+"/v1/us-ca/2014-15/view?scheme=cofog&flow=spending")
	dept := getJSON(t, ts.URL+"/v1/us-ca/2014-15/view?scheme=us_ca_department&flow=spending")
	if cofog["total"] != dept["total"] {
		t.Errorf("cofog total %v != department total %v", cofog["total"], dept["total"])
	}
	nodes := cofog["nodes"].([]any)
	labels := map[string]string{}
	sum := "0.0000"
	hasUnclassified := false
	for _, n := range nodes {
		nm := n.(map[string]any)
		labels[nm["code"].(string)] = nm["label"].(string)
		sum = store.AddDecimals(sum, nm["amount"].(string))
		if nm["code"] == store.UnclassifiedCode {
			hasUnclassified = true
		}
	}
	if len(nodes) < 3 || !hasUnclassified {
		t.Errorf("cofog view should have several code nodes plus __unclassified__, got %d nodes", len(nodes))
	}
	if sum != cofog["total"] {
		t.Errorf("node sum %s != total %v (facts dropped)", sum, cofog["total"])
	}
	if labels["03"] != "Public order and safety" {
		t.Errorf("cofog 03 label = %q, want seeded name", labels["03"])
	}
	if cofog["unmapped"] == "0.0000" || cofog["unmapped"] == cofog["total"] {
		t.Errorf("unmapped = %v: expected partial coverage (honest unmapped bucket)", cofog["unmapped"])
	}

	t.Logf("classify: %d facts mapped, %d up-to-date after re-run, %d unmapped categories, view total %v (unmapped %v)",
		r1.Inserted, r2.UpToDate, len(r1.UnmappedCategories), cofog["total"], cofog["unmapped"])
}

func cofogRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM classification_assignment WHERE scheme_id='cofog'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// modifiedMapping copies the reviewed mapping to a temp file with one entry edited — the
// research deliverable itself is never touched.
func modifiedMapping(t *testing.T, path, key string, edit func(map[string]string)) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m[key]; !ok {
		t.Fatalf("entry %q not in mapping", key)
	}
	edit(m[key])
	return writeJSON(t, m)
}

func writeJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the original file name: ruleId derives from it, and a changed ruleId would
	// legitimately re-version every assignment.
	p := filepath.Join(t.TempDir(), "us-ca-procurement.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func getJSON(t *testing.T, url string) map[string]any {
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
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return m
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}
