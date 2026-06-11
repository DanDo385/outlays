package leads

import (
	"strings"
	"testing"
)

func TestLoadRealRule(t *testing.T) {
	ids, err := Rules()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "ca_vendor_concentration_department_category_v1" {
		t.Fatalf("rules = %v, want exactly the one S11 rule", ids)
	}
	r, err := LoadRule(ids[0])
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	m := r.Meta
	if m.RuleVersion != 1 || m.MethodID != "L001" || m.Severity != "low" {
		t.Errorf("meta = %+v", m)
	}
	// The upgrade requirement: the citation anchors a specific methodology-library entry.
	if m.Citation[0] != "docs/leads-methodology.md#l001--vendor-concentration-inside-buyeryear" {
		t.Errorf("first citation = %q, want the L001 anchor", m.Citation[0])
	}
	if len(m.Citation) < 2 {
		t.Error("expected external red-flag citations alongside the library anchor")
	}
	if !strings.Contains(r.SQL, "$1") || !strings.Contains(r.SQL, "$2") {
		t.Error("rule SQL must be parameterized by jurisdiction and fiscal year")
	}
	if strings.TrimSpace(m.SafePublicWording) == "" || strings.TrimSpace(m.Limitations) == "" {
		t.Error("safety texts missing")
	}
}

func TestLoadRuleRejectsUnknown(t *testing.T) {
	if _, err := LoadRule("no_such_rule_v1"); err == nil {
		t.Error("expected error for unknown rule")
	}
}

func TestBannedWordingGuard(t *testing.T) {
	bad := []string{
		"This vendor committed fraud",
		"evidence of bid rigging",
		"evidence of bid-rigging",
		"the agency intended to evade thresholds",
		"a kickback scheme",
		"vendors colluded on pricing",
	}
	for _, s := range bad {
		if !bannedWording.MatchString(s) {
			t.Errorf("guard missed: %q", s)
		}
	}
	good := []string{
		"One vendor accounts for a high share of this department/category/year in the records loaded by Outlays.",
		"across 8 line-item facts; the group has 34 distinct vendors",
		"irrigation equipment purchases", // substring 'rig' must not trip the word-boundary guard
		"corrugated packaging",
	}
	for _, s := range good {
		if bannedWording.MatchString(s) {
			t.Errorf("guard false positive: %q", s)
		}
	}
}

func TestSharePercent(t *testing.T) {
	cases := map[string]string{
		"0.8558": "85.58%",
		"0.5":    "50.00%",
		"1.0000": "100.00%",
		"0.0001": "0.01%",
		"0":      "0.00%",
	}
	for in, want := range cases {
		if got := sharePercent(in); got != want {
			t.Errorf("sharePercent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLeadIDDeterministic(t *testing.T) {
	m := Meta{RuleID: "r", RuleVersion: 1}
	a := leadID(m, "us-ca", "2014-15", []string{"d", "a", "p"}, []string{"f2", "f1"})
	b := leadID(m, "us-ca", "2014-15", []string{"d", "a", "p"}, []string{"f1", "f2"})
	if a != b {
		t.Error("fact order must not change the lead id")
	}
	c := leadID(m, "us-ca", "2014-15", []string{"d", "a", "p"}, []string{"f1", "f3"})
	if a == c {
		t.Error("different evidence must yield a different lead id")
	}
	m2 := Meta{RuleID: "r", RuleVersion: 2}
	if a == leadID(m2, "us-ca", "2014-15", []string{"d", "a", "p"}, []string{"f1", "f2"}) {
		t.Error("rule version must be part of the lead identity")
	}
}
