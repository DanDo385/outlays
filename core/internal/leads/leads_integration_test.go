//go:build integration

package leads_test

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
	"strings"
	"testing"

	"github.com/djmagro/outlays/core/internal/api"
	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/leads"
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

// TestIntegrationLeadsEndToEnd is the S11 acceptance: the L001 rule generates draft leads
// on real CA data; drafts are invisible to /v1/leads; a human-published lead appears with
// its rule citation and fact links; a later dismissal retracts it append-only.
func TestIntegrationLeadsEndToEnd(t *testing.T) {
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

	ts := httptest.NewServer((&api.Server{Pool: app}).Router())
	defer ts.Close()

	// --- Run the one S11 rule. ---
	rule, err := leads.LoadRule("ca_vendor_concentration_department_category_v1")
	if err != nil {
		t.Fatal(err)
	}
	res, err := leads.Run(ctx, app, rule, jur, year)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// The replay slice yields exactly two matches at the library's v1 thresholds.
	if res.Matches != 2 || res.Inserted != 2 {
		t.Fatalf("run = %+v, want 2 matches / 2 inserted", res)
	}

	// Re-run is idempotent (deterministic lead ids over unchanged evidence).
	res2, err := leads.Run(ctx, app, rule, jur, year)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Inserted != 0 || res2.AlreadyKnown != 2 {
		t.Errorf("re-run = %+v, want 0 inserted / 2 already known", res2)
	}

	// --- Drafts are invisible to the public endpoint. ---
	if got := publicLeads(t, ts.URL); len(got) != 0 {
		t.Fatalf("drafts leaked to /v1/leads: %v", got)
	}

	// --- Review CLI surface: list + inspect. ---
	drafts, err := leads.List(ctx, app, "draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(drafts) != 2 {
		t.Fatalf("draft list = %+v", drafts)
	}
	// Highest share first in generation order; find the McKesson lead by title.
	var target leads.Summary
	for _, d := range drafts {
		if strings.Contains(d.Title, "MCKESSON") {
			target = d
		}
	}
	if target.LeadID == "" {
		t.Fatalf("expected a MCKESSON concentration draft, got %+v", drafts)
	}
	det, err := leads.Inspect(ctx, app, target.LeadID)
	if err != nil || det == nil {
		t.Fatalf("inspect: %v %v", det, err)
	}
	if det.Status != "draft" || len(det.FactIDs) != 8 {
		t.Errorf("inspect = status %s, %d facts; want draft/8", det.Status, len(det.FactIDs))
	}
	var body map[string]any
	if err := json.Unmarshal(det.Body, &body); err != nil {
		t.Fatal(err)
	}
	if cites, _ := body["basisCitation"].([]any); len(cites) == 0 ||
		cites[0] != "docs/leads-methodology.md#l001--vendor-concentration-inside-buyeryear" {
		t.Errorf("lead body citation = %v, want the L001 anchor first", body["basisCitation"])
	}
	if body["severity"] != "low" || body["methodLimitations"] == "" {
		t.Errorf("body missing severity/limitations: %v %v", body["severity"], body["methodLimitations"])
	}
	subj, _ := body["subject"].(map[string]any)
	if subj["share"] != "0.8558" || subj["vendorAmount"] != "808324.8100" || subj["groupAmount"] != "944525.7400" {
		t.Errorf("subject stats = %v", subj)
	}

	// --- Review actions: reviewer handle is mandatory; bogus status rejected. ---
	if err := leads.SetStatus(ctx, app, target.LeadID, "published", "", ""); err == nil {
		t.Error("publish without reviewer handle must fail")
	}
	if err := leads.SetStatus(ctx, app, target.LeadID, "draft", "reviewer:test", ""); err == nil {
		t.Error("human events cannot set status back to machine 'draft'")
	}

	// --- Publish (with handle) and verify the public shape. ---
	if err := leads.SetStatus(ctx, app, target.LeadID, "published", "reviewer:integration-test", "publishing for S11 acceptance"); err != nil {
		t.Fatalf("publish: %v", err)
	}
	pub := publicLeads(t, ts.URL)
	if len(pub) != 1 {
		t.Fatalf("published leads = %d, want exactly 1 (other draft must stay hidden)", len(pub))
	}
	p := pub[0].(map[string]any)
	if p["leadId"] != target.LeadID || p["ruleId"] != rule.Meta.RuleID || p["reviewer"] != "reviewer:integration-test" {
		t.Errorf("published lead = %v", p)
	}
	cites, _ := p["citation"].([]any)
	if len(cites) < 2 || cites[0] != "docs/leads-methodology.md#l001--vendor-concentration-inside-buyeryear" {
		t.Errorf("public citation = %v", p["citation"])
	}
	facts, _ := p["factIds"].([]any)
	if len(facts) != 8 {
		t.Errorf("public fact links = %d, want 8", len(facts))
	}
	// Fact links resolve to real provenance.
	prov := getJSON(t, ts.URL+"/v1/fact/"+facts[0].(string)+"/provenance")
	if prov["rawSha256"] == nil || prov["derivationQuery"] == "" {
		t.Errorf("lead fact provenance incomplete: %v", prov)
	}
	// Published wording stays neutral and carries limitations.
	if p["limitations"] == "" || p["severity"] != "low" {
		t.Errorf("published lead missing context: %v", p)
	}

	// --- Append-only retraction: dismiss wins as the latest event. ---
	if err := leads.SetStatus(ctx, app, target.LeadID, "dismissed", "reviewer:integration-test", "retracting in test"); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	if got := publicLeads(t, ts.URL); len(got) != 0 {
		t.Errorf("dismissed lead still public: %v", got)
	}
	det2, _ := leads.Inspect(ctx, app, target.LeadID)
	if det2.Status != "dismissed" || len(det2.Events) != 2 {
		t.Errorf("event history = status %s, %d events; want dismissed/2", det2.Status, len(det2.Events))
	}

	// The status filter guard on the public endpoint is unchanged.
	resp, _ := http.Get(ts.URL + "/v1/leads?status=draft")
	if resp.StatusCode != 400 {
		t.Errorf("leads?status=draft -> %d, want 400", resp.StatusCode)
	}
	resp.Body.Close()

	t.Logf("leads: 2 drafts generated; published lead carried citation %v with %d fact links; dismissal retracted it", cites[0], len(facts))
}

func publicLeads(t *testing.T, base string) []any {
	t.Helper()
	return getJSON(t, base+"/v1/leads?status=published")["leads"].([]any)
}

func getJSON(t *testing.T, url string) map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s -> %d: %s", url, resp.StatusCode, b)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
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
