// Command api serves the read-only public HTTP surface (ARCHITECTURE.md Section 5). Full
// behavior lands in S6. S0: buildable stub.
package main

import (
	"fmt"
	"os"

	"github.com/djmagro/outlays/core/internal/api"
)

func main() {
	fmt.Fprintf(os.Stderr, "api: not implemented until S6 (api v%s)\n", api.Version)
}
