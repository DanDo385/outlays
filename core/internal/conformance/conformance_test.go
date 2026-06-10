package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRoot walks up from this file to the repository root.
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
			t.Fatalf("could not locate repo root")
		}
		dir = parent
	}
}

// TestToyAdapterConformance runs the built TS toy adapter through the harness. It is skipped
// when the adapter has not been built or node is unavailable, so `go test ./...` stays green
// without the JS toolchain; the conformance CI job builds the adapters first.
func TestToyAdapterConformance(t *testing.T) {
	root := repoRoot(t)
	cli := filepath.Join(root, "packages", "adapters", "toy-fixture", "dist", "cli.js")
	if _, err := os.Stat(cli); err != nil {
		t.Skip("toy adapter not built (run: pnpm --filter @outlays/adapter-toy-fixture build)")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH")
	}

	res, err := Run([]string{node, cli}, "2024-25", t.TempDir())
	if err != nil {
		t.Fatalf("conformance run error: %v", err)
	}
	for _, c := range res.Checks {
		if !c.Pass {
			t.Errorf("check failed: %s — %s", c.Name, c.Detail)
		}
	}
	if !res.Passed() {
		t.Fatalf("toy adapter did not pass conformance")
	}
}
