// Command conformance runs an adapter binary against recorded fixtures and verifies
// protocol, schema validity, rawHash correctness, and resultHash determinism. Full behavior
// lands in S2. S0: buildable stub.
package main

import (
	"fmt"
	"os"

	"github.com/djmagro/outlays/core/internal/verify"
)

func main() {
	fmt.Fprintf(os.Stderr, "conformance: not implemented until S2 (verify v%s)\n", verify.Version)
}
