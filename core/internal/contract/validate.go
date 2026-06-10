// Package contract holds the generated Go types (types.go) and schema-based validation for
// the Outlays contract. Validation uses the canonical JSON Schema directly (draft
// 2020-12) so conditional rules — e.g. transaction/award grain requires rawSha256 — are
// enforced identically to the TS and Python validators.
package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaURL = "https://outlays.org/schemas/fiscal.schema.json"

// SchemaPath returns the absolute path to the canonical schema, located by walking up from
// this source file to the repo root.
func SchemaPath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot determine caller path")
	}
	dir := filepath.Dir(thisFile)
	for {
		candidate := filepath.Join(dir, "packages", "contract", "schemas", "fiscal.schema.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate packages/contract/schemas/fiscal.schema.json")
		}
		dir = parent
	}
}

var (
	schemaOnce sync.Once
	rootSchema *jsonschema.Schema
	schemaErr  error
	compiler   *jsonschema.Compiler

	defMu   sync.Mutex
	defCach = map[string]*jsonschema.Schema{}
)

func loadCompiler() (*jsonschema.Compiler, error) {
	schemaOnce.Do(func() {
		path, err := SchemaPath()
		if err != nil {
			schemaErr = err
			return
		}
		f, err := os.Open(path)
		if err != nil {
			schemaErr = err
			return
		}
		defer f.Close()
		doc, err := jsonschema.UnmarshalJSON(f)
		if err != nil {
			schemaErr = err
			return
		}
		c := jsonschema.NewCompiler()
		if err := c.AddResource(schemaURL, doc); err != nil {
			schemaErr = err
			return
		}
		compiler = c
		rootSchema, schemaErr = c.Compile(schemaURL)
	})
	return compiler, schemaErr
}

// validatorFor returns a compiled validator for a top-level $defs type, e.g. "FiscalFact".
func validatorFor(def string) (*jsonschema.Schema, error) {
	if _, err := loadCompiler(); err != nil {
		return nil, err
	}
	defMu.Lock()
	defer defMu.Unlock()
	if s, ok := defCach[def]; ok {
		return s, nil
	}
	s, err := compiler.Compile(schemaURL + "#/$defs/" + def)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", def, err)
	}
	defCach[def] = s
	return s, nil
}

// Validate checks a decoded JSON instance against the named contract type. The instance must
// be decoded with jsonschema.UnmarshalJSON (or be the standard map[string]any / []any /
// json.Number-free shapes the validator expects). Returns nil when valid.
func Validate(def string, instance any) error {
	s, err := validatorFor(def)
	if err != nil {
		return err
	}
	return s.Validate(instance)
}

// IsValid reports whether the instance satisfies the named contract type.
func IsValid(def string, instance any) bool {
	return Validate(def, instance) == nil
}
