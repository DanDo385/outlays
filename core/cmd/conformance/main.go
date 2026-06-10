// Command conformance runs an adapter binary against the CLI protocol and verifies protocol,
// schema validity, rawHash correctness, and resultHash determinism (ARCHITECTURE.md Section 4).
//
// Usage:
//
//	conformance --cmd "node /abs/path/dist/cli.js" --year 2024-25 [--work /tmp/dir]
//
// --cmd is the base adapter command (the subcommands info|list-years|fetch are appended).
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/djmagro/outlays/core/internal/conformance"
)

func main() {
	cmdStr := flag.String("cmd", "", "adapter base command, e.g. \"node /abs/dist/cli.js\"")
	year := flag.String("year", "2024-25", "fiscal year to fetch")
	work := flag.String("work", "", "work directory (default: a temp dir)")
	flag.Parse()

	if strings.TrimSpace(*cmdStr) == "" {
		fmt.Fprintln(os.Stderr, "error: --cmd is required")
		os.Exit(2)
	}
	adapterCmd := strings.Fields(*cmdStr)

	workDir := *work
	if workDir == "" {
		var err error
		workDir, err = os.MkdirTemp("", "conformance-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error creating work dir:", err)
			os.Exit(1)
		}
	}

	res, err := conformance.Run(adapterCmd, *year, workDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("conformance: %s (year %s)\n", *cmdStr, *year)
	for _, c := range res.Checks {
		status := "PASS"
		if !c.Pass {
			status = "FAIL"
		}
		line := fmt.Sprintf("  [%s] %s", status, c.Name)
		if c.Detail != "" {
			line += "  — " + c.Detail
		}
		fmt.Println(line)
	}

	if res.Passed() {
		fmt.Printf("RESULT: PASS (resultHash %s)\n", res.ResultHash)
		os.Exit(0)
	}
	fmt.Println("RESULT: FAIL")
	os.Exit(1)
}
