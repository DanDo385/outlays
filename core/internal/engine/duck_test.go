package engine

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/djmagro/outlays/core/internal/store"
)

// buildFixturePartition writes a tiny synthetic partition as Parquet via DuckDB itself:
//   - f1: dept-classified, cofog 03 at v1 then reassigned 07 at v2 (DISTINCT ON must pick v2)
//   - f2: cofog 03, has a payee
//   - f3: no cofog assignment (unclassified bucket), no payee
//   - f4: superseded by f5 (must be excluded); f5 is the current correction
//   - f6: revenue flow (must be excluded from spending views)
func buildFixturePartition(t *testing.T) *partitionFiles {
	t.Helper()
	dir := t.TempDir()
	pf := &partitionFiles{
		facts:       filepath.Join(dir, "facts.parquet"),
		assignments: filepath.Join(dir, "assignments.parquet"),
		codes:       filepath.Join(dir, "codes.parquet"),
		entities:    filepath.Join(dir, "entities.parquet"),
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE facts (fact_id VARCHAR, jurisdiction VARCHAR, fiscal_year VARCHAR, flow VARCHAR,
			grain VARCHAR, payer_entity VARCHAR, payee_entity VARCHAR, amount DECIMAL(24,4),
			currency VARCHAR, occurred_on DATE, description VARCHAR, raw_sha256 VARCHAR,
			derivation_query VARCHAR, fact_hash VARCHAR, supersedes VARCHAR)`,
		`INSERT INTO facts VALUES
			('f1','us-xx','2014-15','spending','award',NULL,NULL,100.5,'USD',NULL,NULL,'r','d','h1',NULL),
			('f2','us-xx','2014-15','spending','award',NULL,'e1',200.25,'USD',NULL,NULL,'r','d','h2',NULL),
			('f3','us-xx','2014-15','spending','award',NULL,NULL,9.99,'USD',NULL,NULL,'r','d','h3',NULL),
			('f4','us-xx','2014-15','spending','award',NULL,'e1',1000,'USD',NULL,NULL,'r','d','h4',NULL),
			('f5','us-xx','2014-15','spending','award',NULL,'e1',999,'USD',NULL,NULL,'r','d','h5','f4'),
			('f6','us-xx','2014-15','revenue','aggregate',NULL,NULL,5555,'USD',NULL,NULL,'r','d','h6',NULL)`,
		`CREATE TABLE assignments (assignment_id VARCHAR, fact_id VARCHAR, scheme_id VARCHAR, code VARCHAR,
			assigned_by VARCHAR, confidence VARCHAR, basis VARCHAR, version INTEGER)`,
		`INSERT INTO assignments VALUES
			('a1','f1','cofog','03','rule','0.5','b',1),
			('a2','f1','cofog','07','rule','0.5','b',2),
			('a3','f2','cofog','03','rule','0.5','b',1),
			('a4','f5','cofog','03','rule','0.5','b',1),
			('a5','f1','us_xx_department','Dept A','source',NULL,'b',1)`,
		`CREATE TABLE codes (scheme_id VARCHAR, code VARCHAR, parent_code VARCHAR, name VARCHAR)`,
		`INSERT INTO codes VALUES ('cofog','03',NULL,'Public order and safety'), ('cofog','07',NULL,'Health')`,
		`CREATE TABLE entities (entity_id VARCHAR, kind VARCHAR, canonical_name VARCHAR)`,
		`INSERT INTO entities VALUES ('e1','vendor','ACME CORP')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture: %v\n%s", err, s)
		}
	}
	for tbl, path := range map[string]string{
		"facts": pf.facts, "assignments": pf.assignments, "codes": pf.codes, "entities": pf.entities,
	} {
		if _, err := db.Exec(fmt.Sprintf(`COPY %s TO '%s' (FORMAT PARQUET)`, tbl, sqlEscape(path))); err != nil {
			t.Fatalf("copy %s: %v", tbl, err)
		}
	}
	return pf
}

func TestDuckSchemeViewSemantics(t *testing.T) {
	pf := buildFixturePartition(t)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	v := &store.View{Jurisdiction: "us-xx", FiscalYear: "2014-15", Flow: "spending", SchemeID: "cofog", Currency: "USD", Nodes: []store.Node{}}
	v, err = scanDuckView(context.Background(), db, v, schemeViewSQL(pf), "us-xx", "2014-15", "spending", "cofog", "cofog")
	if err != nil {
		t.Fatal(err)
	}

	// Current spending facts: f1(07 via v2, 100.5), f2(03, 200.25), f3(unclassified, 9.99),
	// f5(03, 999; correction row included, superseded f4 excluded). f6 is revenue.
	want := map[string]struct {
		label, amount string
		count         int64
	}{
		"03":                   {"Public order and safety", "1199.2500", 2},
		"07":                   {"Health", "100.5000", 1},
		store.UnclassifiedCode: {"Unclassified", "9.9900", 1},
	}
	if len(v.Nodes) != len(want) {
		t.Fatalf("nodes = %+v, want %d", v.Nodes, len(want))
	}
	for _, n := range v.Nodes {
		w, ok := want[n.Code]
		if !ok {
			t.Errorf("unexpected node %+v", n)
			continue
		}
		if n.Label != w.label || n.Amount != w.amount || n.FactCount != w.count {
			t.Errorf("node %s = {%s %s %d}, want {%s %s %d}",
				n.Code, n.Label, n.Amount, n.FactCount, w.label, w.amount, w.count)
		}
	}
	if v.Total != "1309.7400" || v.Unmapped != "9.9900" {
		t.Errorf("total %s unmapped %s, want 1309.7400 / 9.9900", v.Total, v.Unmapped)
	}
	// Deterministic ordering: amount DESC, then code.
	if v.Nodes[0].Code != "03" || v.Nodes[1].Code != "07" || v.Nodes[2].Code != store.UnclassifiedCode {
		t.Errorf("ordering wrong: %+v", v.Nodes)
	}
}

func TestDuckPayeeViewSemantics(t *testing.T) {
	pf := buildFixturePartition(t)
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	v := &store.View{Jurisdiction: "us-xx", FiscalYear: "2014-15", Flow: "spending", SchemeID: "payee", Currency: "USD", Nodes: []store.Node{}}
	v, err = scanDuckView(context.Background(), db, v, payeeViewSQL(pf), "us-xx", "2014-15", "spending")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Nodes) != 2 {
		t.Fatalf("nodes = %+v, want ACME + unclassified", v.Nodes)
	}
	if v.Nodes[0].Code != "e1" || v.Nodes[0].Label != "ACME CORP" || v.Nodes[0].Amount != "1199.2500" || v.Nodes[0].FactCount != 2 {
		t.Errorf("ACME node = %+v", v.Nodes[0])
	}
	if v.Nodes[1].Code != store.UnclassifiedCode || v.Nodes[1].Amount != "110.4900" {
		t.Errorf("unclassified node = %+v", v.Nodes[1])
	}
	if v.Total != "1309.7400" || v.Unmapped != "110.4900" {
		t.Errorf("total %s unmapped %s", v.Total, v.Unmapped)
	}
}
