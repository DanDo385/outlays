// Command orchestrator executes adapters per the CLI protocol, validates their output, and
// persists facts. Full behavior lands in S5. S0: buildable stub.
package main

import (
	"fmt"
	"os"

	"github.com/djmagro/outlays/core/internal/ingest"
)

func main() {
	fmt.Fprintf(os.Stderr, "orchestrator: not implemented until S5 (ingest v%s)\n", ingest.Version)
}
