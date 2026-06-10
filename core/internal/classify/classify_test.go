package classify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func realMappingPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..",
		"data", "cofog", "us-ca-procurement.json"))
}

func writeMapping(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mapping.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadRealMapping(t *testing.T) {
	m, err := LoadMapping(realMappingPath(t), "us-ca")
	if err != nil {
		t.Fatalf("load real mapping: %v", err)
	}
	if m.RuleID != "cofog-map/us-ca-procurement" {
		t.Errorf("ruleId = %q", m.RuleID)
	}
	if len(m.Sha256) != 64 {
		t.Errorf("file sha256 = %q", m.Sha256)
	}

	mapped, unmapped, acqMapped := 0, 0, 0
	for _, c := range m.Categories {
		if c.Mapped() {
			mapped++
			if c.SchemeID == "us_ca_acquisition_type" {
				acqMapped++
			}
			if c.BasisJSON == "" || c.Confidence == "" {
				t.Errorf("%s: mapped entry missing basis/confidence", c.Key)
			}
		} else {
			unmapped++
		}
	}
	if mapped != 24 || unmapped != 6 {
		t.Errorf("mapped=%d unmapped=%d, want 24/6 (the reviewed conservative split)", mapped, unmapped)
	}
	// Conservatism by design: acquisition types describe inputs, not functions.
	if acqMapped != 0 {
		t.Errorf("%d acquisition types mapped, want 0", acqMapped)
	}

	// Basis carries rule id + citation + source category + verbatim confidence + entry hash.
	var c Category
	for _, cat := range m.Categories {
		if cat.Key == "department: Franchise Tax Board" {
			c = cat
		}
	}
	var basis map[string]string
	if err := json.Unmarshal([]byte(c.BasisJSON), &basis); err != nil {
		t.Fatalf("basis is not JSON: %v", err)
	}
	if basis["ruleId"] != "cofog-map/us-ca-procurement" ||
		basis["sourceCategory"] != "department: Franchise Tax Board" ||
		basis["confidence"] != "medium" ||
		!strings.HasPrefix(basis["citation"], "docs/cofog-references.md#") ||
		len(basis["entrySha256"]) != 64 {
		t.Errorf("basis missing fields: %s", c.BasisJSON)
	}
	if c.Entry.CofogCode != "01" || c.Confidence != "0.5" {
		t.Errorf("FTB: code=%s confidence=%s, want 01 / 0.5", c.Entry.CofogCode, c.Confidence)
	}
}

func TestLoadMappingDeterministic(t *testing.T) {
	a, err := LoadMapping(realMappingPath(t), "us-ca")
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadMapping(realMappingPath(t), "us-ca")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Categories) != len(b.Categories) {
		t.Fatalf("category counts differ")
	}
	for i := range a.Categories {
		if a.Categories[i] != b.Categories[i] {
			t.Errorf("category %d differs across loads:\n%+v\n%+v", i, a.Categories[i], b.Categories[i])
		}
	}
}

func TestLoadMappingRejectsInvalid(t *testing.T) {
	valid := `{"cofogCode":"03","confidence":"medium","basis":"docs/x.md#cite","note":"n"}`
	cases := map[string]string{
		"unknown prefix":      `{"commodity: Paper": ` + valid + `}`,
		"missing separator":   `{"department": ` + valid + `}`,
		"bad cofog code":      `{"department: X": {"cofogCode":"99","confidence":"low","basis":"docs/x.md#cite"}}`,
		"unpadded code":       `{"department: X": {"cofogCode":"3","confidence":"low","basis":"docs/x.md#cite"}}`,
		"bad confidence":      `{"department: X": {"cofogCode":"03","confidence":"certain","basis":"docs/x.md#cite"}}`,
		"missing basis":       `{"department: X": {"cofogCode":"03","confidence":"low","basis":"  "}}`,
		"unknown entry field": `{"department: X": {"cofogCode":"03","confidence":"low","basis":"docs/x.md#cite","extra":1}}`,
	}
	for name, content := range cases {
		if _, err := LoadMapping(writeMapping(t, content), "us-ca"); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
	}

	if _, err := LoadMapping(writeMapping(t, `{"department: X": `+valid+`}`), "us-zz"); err == nil {
		t.Error("unknown jurisdiction: expected error, got none")
	}
	if _, err := LoadMapping(writeMapping(t, `{"department: X": `+valid+`}`), "us-ca"); err != nil {
		t.Errorf("valid mapping rejected: %v", err)
	}
}

func TestCategoryParsing(t *testing.T) {
	// Category labels may themselves contain ", Office of" etc; only the first ": " splits.
	m, err := LoadMapping(writeMapping(t,
		`{"department: Statewide Health Planning & Development, Office of": {"cofogCode":"07","confidence":"medium","basis":"docs/x.md#cite"}}`), "us-ca")
	if err != nil {
		t.Fatal(err)
	}
	c := m.Categories[0]
	if c.SchemeID != "us_ca_department" || c.Code != "Statewide Health Planning & Development, Office of" {
		t.Errorf("parsed scheme=%q code=%q", c.SchemeID, c.Code)
	}
}
