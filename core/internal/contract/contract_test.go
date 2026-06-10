package contract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type fixtureCase struct {
	File  string `json:"file"`
	Def   string `json:"def"`
	Valid bool   `json:"valid"`
}

// TestFixtureValidity runs the shared contract fixtures (packages/contract/fixtures) and
// asserts each fixture's validity matches its declared flag. Must agree with the TS and
// Python validators.
func TestFixtureValidity(t *testing.T) {
	schemaPath, err := SchemaPath()
	if err != nil {
		t.Fatalf("locate schema: %v", err)
	}
	fixturesRoot := filepath.Join(filepath.Dir(schemaPath), "..", "fixtures")

	casesDoc := mustUnmarshal(t, filepath.Join(fixturesRoot, "cases.json"))
	casesMap, ok := casesDoc.(map[string]any)
	if !ok {
		t.Fatalf("cases.json: unexpected shape")
	}
	rawCases, ok := casesMap["cases"].([]any)
	if !ok {
		t.Fatalf("cases.json: missing cases array")
	}

	for _, rc := range rawCases {
		c := rc.(map[string]any)
		file := c["file"].(string)
		def := c["def"].(string)
		valid := c["valid"].(bool)

		t.Run(file, func(t *testing.T) {
			instance := mustUnmarshal(t, filepath.Join(fixturesRoot, file))
			got := IsValid(def, instance)
			if got != valid {
				err := Validate(def, instance)
				t.Errorf("%s (%s): expected valid=%v, got %v; err=%v", file, def, valid, got, err)
			}
		})
	}
}

func mustUnmarshal(t *testing.T, path string) any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	doc, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return doc
}
