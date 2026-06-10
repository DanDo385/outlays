// Command anchor computes the Merkle root over sorted fact_hash values for a run and
// submits it to the on-chain AnchorRegistry. Full behavior lands in S12. S0: buildable stub.
package main

import (
	"fmt"
	"os"

	"github.com/djmagro/outlays/core/internal/store"
)

func main() {
	fmt.Fprintf(os.Stderr, "anchor: not implemented until S12 (store v%s)\n", store.Version)
}
