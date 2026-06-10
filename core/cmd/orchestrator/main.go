// Command orchestrator runs an adapter per the CLI protocol, validates and verifies its
// output by re-derivation, persists it, and marks run status. Supports concurrent multi-year
// backfill.
//
// Usage:
//
//	orchestrator run --adapter "node /abs/dist/cli.js" --year 2014-15
//	orchestrator run --adapter "node /abs/dist/cli.js" --years 2012-13,2013-14,2014-15 --concurrency 2
//
// DB/object-store come from env (DATABASE_URL, MIGRATE_DATABASE_URL, S3_*). With
// --replay-dir, the adapter runs offline against recorded fixtures.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintln(os.Stderr, "usage: orchestrator run --adapter <cmd> (--year <Y> | --years <Y1,Y2>) [--dataset <d>] [--concurrency N] [--replay-dir <dir>] [--max-pages N]")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("run", flag.ExitOnError)
	adapter := fs.String("adapter", "", `adapter base command, e.g. "node /abs/dist/cli.js"`)
	year := fs.String("year", "", "single fiscal year")
	years := fs.String("years", "", "comma-separated fiscal years (multi-year backfill)")
	dataset := fs.String("dataset", "", "object-storage partition (default: manifest datasets[0])")
	concurrency := fs.Int("concurrency", 2, "max concurrent years")
	replayDir := fs.String("replay-dir", "", "OUTLAYS_REPLAY_DIR for offline replay")
	maxPages := fs.Int("max-pages", 0, "OUTLAYS_MAX_PAGES (0 = unbounded)")
	_ = fs.Parse(os.Args[2:])

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	if strings.TrimSpace(*adapter) == "" {
		log.Error("--adapter is required")
		os.Exit(2)
	}
	yearList := splitYears(*year, *years)
	if len(yearList) == 0 {
		log.Error("provide --year or --years")
		os.Exit(2)
	}

	ctx := context.Background()

	if migrateURL := os.Getenv("MIGRATE_DATABASE_URL"); migrateURL != "" {
		if err := store.Migrate(migrateURL); err != nil {
			log.Error("migrate failed", "err", err)
			os.Exit(1)
		}
		log.Info("migrations applied")
	}

	pool, err := store.Connect(ctx, mustEnv(log, "DATABASE_URL"))
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	obj, err := store.NewObjectStore(ctx, objConfigFromEnv())
	if err != nil {
		log.Error("connect object store", "err", err)
		os.Exit(1)
	}

	var extraEnv []string
	if *replayDir != "" {
		extraEnv = append(extraEnv, "OUTLAYS_REPLAY_DIR="+*replayDir)
	}
	if *maxPages > 0 {
		extraEnv = append(extraEnv, fmt.Sprintf("OUTLAYS_MAX_PAGES=%d", *maxPages))
	}

	opts := &ingest.Options{
		AdapterCmd: strings.Fields(*adapter),
		Dataset:    *dataset,
		Pool:       pool,
		Obj:        obj,
		Logger:     log,
		ExtraEnv:   extraEnv,
	}

	outcomes, err := ingest.Backfill(ctx, opts, yearList, *concurrency)
	if err != nil {
		log.Error("backfill error", "err", err)
		os.Exit(1)
	}
	failed := 0
	for _, oc := range outcomes {
		log.Info("run outcome", "year", oc.Year, "status", oc.Status, "runId", oc.RunID, "facts", oc.Facts)
		if oc.Status != "succeeded" {
			failed++
		}
	}
	if failed > 0 {
		log.Warn("some years failed", "failed", failed, "total", len(outcomes))
		os.Exit(3)
	}
}

func splitYears(single, multi string) []string {
	if multi != "" {
		out := []string{}
		for _, y := range strings.Split(multi, ",") {
			if y = strings.TrimSpace(y); y != "" {
				out = append(out, y)
			}
		}
		return out
	}
	if single != "" {
		return []string{single}
	}
	return nil
}

func mustEnv(log *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error("missing required env", "key", key)
		os.Exit(1)
	}
	return v
}

func objConfigFromEnv() store.ObjectStoreConfig {
	endpoint := os.Getenv("S3_ENDPOINT")
	useSSL := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	return store.ObjectStoreConfig{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		Bucket:    envOr("S3_BUCKET", "fiscal-raw"),
		Region:    envOr("S3_REGION", "us-east-1"),
		UseSSL:    useSSL,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
