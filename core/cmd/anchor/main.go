// Command anchor computes the D31 Merkle root over a run's fact hashes and submits it to
// the on-chain AnchorRegistry, persisting the tx ref as an append-only run_anchor row
// (task S12).
//
// Usage:
//
//	anchor run --run-id <uuid> [--uri <uri>]
//	anchor root --run-id <uuid>            (compute + print only, no chain access)
//
// Env: DATABASE_URL, ANCHOR_RPC_URL, ANCHOR_REGISTRY_ADDRESS, and either ANCHOR_FROM
// (unlocked local anvil) or ANCHOR_PRIVATE_KEY.
// Requires the pinned Foundry toolchain (cast) on PATH for chain access.
//
// Re-running for an already-anchored run is a verified no-op: the on-chain root is read
// back and compared to the recomputation; a mismatch is a loud failure, never a rewrite.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/djmagro/outlays/core/internal/anchor"
	"github.com/djmagro/outlays/core/internal/store"
)

const usage = `usage:
  anchor run --run-id <uuid> [--uri <uri>]
  anchor root --run-id <uuid>`

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if len(os.Args) < 2 || (os.Args[1] != "run" && os.Args[1] != "root") {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	mode := os.Args[1]

	fs := flag.NewFlagSet(mode, flag.ExitOnError)
	runID := fs.String("run-id", "", "ingestion run UUID to anchor")
	uri := fs.String("uri", "", "evidence URI stored on chain (default outlays://run/<run-id>)")
	_ = fs.Parse(os.Args[2:])
	if *runID == "" {
		log.Error("--run-id is required")
		os.Exit(2)
	}
	runIDHex, err := anchor.RunIDBytes32(*runID)
	if err != nil {
		log.Error("invalid run id", "err", err)
		os.Exit(2)
	}
	if *uri == "" {
		*uri = "outlays://run/" + *runID
	}

	ctx := context.Background()
	pool, err := store.Connect(ctx, mustEnv(log, "DATABASE_URL"))
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	hashes, err := anchor.FactHashes(ctx, pool, *runID)
	if err != nil {
		log.Error("read fact hashes", "err", err)
		os.Exit(1)
	}
	rootHex, err := anchor.RootHex(hashes)
	if err != nil {
		log.Error("compute merkle root", "err", err)
		os.Exit(1)
	}

	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")

	if mode == "root" {
		_ = out.Encode(map[string]any{
			"runId": *runID, "runIdBytes32": runIDHex, "factCount": len(hashes), "merkleRoot": rootHex,
		})
		return
	}

	cfg := anchor.ChainConfig{
		RPCURL:   mustEnv(log, "ANCHOR_RPC_URL"),
		Registry: mustEnv(log, "ANCHOR_REGISTRY_ADDRESS"),
	}
	if v := os.Getenv("ANCHOR_FROM"); v != "" {
		cfg.From = v
	} else if v := os.Getenv("ANCHOR_PRIVATE_KEY"); v != "" {
		cfg.PrivateKey = v
	} else {
		log.Error("set ANCHOR_FROM (unlocked local node) or ANCHOR_PRIVATE_KEY")
		os.Exit(1)
	}

	// Duplicate runIds are rejected on chain; pre-check so a re-run is a verified no-op.
	if anchored, err := anchor.IsAnchored(ctx, cfg, runIDHex); err != nil {
		log.Error("isAnchored", "err", err)
		os.Exit(1)
	} else if anchored {
		onChain, err := anchor.OnChainRoot(ctx, cfg, runIDHex)
		if err != nil {
			log.Error("read on-chain root", "err", err)
			os.Exit(1)
		}
		if onChain != rootHex {
			log.Error("run already anchored with a DIFFERENT root — evidence diverged",
				"onChain", onChain, "recomputed", rootHex)
			os.Exit(1)
		}
		_ = out.Encode(map[string]any{
			"runId": *runID, "factCount": len(hashes), "merkleRoot": rootHex,
			"alreadyAnchored": true, "onChainRootMatches": true,
		})
		log.Info("already anchored; on-chain root matches recomputation", "runId", *runID)
		return
	}

	chainID, err := anchor.ChainID(ctx, cfg)
	if err != nil {
		log.Error("chain id", "err", err)
		os.Exit(1)
	}
	receipt, err := anchor.Submit(ctx, cfg, runIDHex, rootHex, *uri)
	if err != nil {
		log.Error("submit anchor", "err", err)
		os.Exit(1)
	}
	if err := anchor.Persist(ctx, pool, *runID, rootHex, len(hashes), chainID, cfg.Registry, receipt); err != nil {
		log.Error("persist tx ref", "err", err)
		os.Exit(1)
	}

	_ = out.Encode(map[string]any{
		"runId": *runID, "runIdBytes32": runIDHex, "factCount": len(hashes),
		"merkleRoot": rootHex, "uri": *uri, "chainId": chainID,
		"contract": cfg.Registry, "txHash": receipt.TxHash, "blockNumber": receipt.Block(),
	})
	log.Info("anchored", "runId", *runID, "facts", len(hashes), "tx", receipt.TxHash)
}

func mustEnv(log *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error("missing required env", "key", key)
		os.Exit(1)
	}
	return v
}
