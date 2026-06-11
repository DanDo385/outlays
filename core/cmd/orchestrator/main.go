// Command orchestrator runs an adapter per the CLI protocol, validates and verifies its
// output by re-derivation, persists it, and marks run status. Supports concurrent multi-year
// backfill. The classify subcommand applies a reviewed COFOG mapping file as versioned
// classification_assignment rows (S9).
//
// Usage:
//
//	orchestrator run --adapter "node /abs/dist/cli.js" --year 2014-15
//	orchestrator run --adapter "node /abs/dist/cli.js" --years 2012-13,2013-14,2014-15 --concurrency 2
//	orchestrator classify --mapping data/cofog/us-ca-procurement.json --jurisdiction us-ca --year 2014-15
//	orchestrator classify --mapping ... --jurisdiction us-ca --year 2014-15 --list-unmapped
//
// DB/object-store come from env (DATABASE_URL, MIGRATE_DATABASE_URL, S3_*). With
// --replay-dir, the adapter runs offline against recorded fixtures.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/djmagro/outlays/core/internal/classify"
	"github.com/djmagro/outlays/core/internal/engine"
	"github.com/djmagro/outlays/core/internal/ingest"
	"github.com/djmagro/outlays/core/internal/store"
)

const usage = `usage:
  orchestrator run --adapter <cmd> (--year <Y> | --years <Y1,Y2>) [--dataset <d>] [--concurrency N] [--replay-dir <dir>] [--max-pages N]
  orchestrator classify --mapping <path> --jurisdiction <jur> --year <Y> [--flow spending|revenue] [--dry-run] [--list-unmapped]
  orchestrator export-parquet --jurisdiction <jur> --year <Y>`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		runMain(os.Args[2:])
	case "classify":
		classifyMain(os.Args[2:])
	case "export-parquet":
		exportParquetMain(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

// exportParquetMain snapshots one partition as content-addressed Parquet in object storage
// and registers it in parquet_export (S10).
func exportParquetMain(args []string) {
	fs := flag.NewFlagSet("export-parquet", flag.ExitOnError)
	jurisdiction := fs.String("jurisdiction", "", "jurisdiction to export, e.g. us-ca")
	year := fs.String("year", "", "fiscal year to export, e.g. 2014-15")
	_ = fs.Parse(args)

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if *jurisdiction == "" || *year == "" {
		log.Error("--jurisdiction and --year are required")
		os.Exit(2)
	}
	if !fiscalYearRe.MatchString(*year) {
		log.Error("invalid fiscal year", "year", *year)
		os.Exit(2)
	}

	ctx := context.Background()
	pool, err := store.Connect(ctx, mustEnv(log, "DATABASE_URL"))
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfigFromEnv())
	if err != nil {
		log.Error("connect object store", "err", err)
		os.Exit(1)
	}

	dir, err := os.MkdirTemp("", "outlays-parquet-export-")
	if err != nil {
		log.Error("temp dir", "err", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	res, err := engine.Export(ctx, pool, obj, *jurisdiction, *year, dir)
	if err != nil {
		log.Error("export", "err", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res)
	log.Info("export done", "exportId", res.ExportID, "artifacts", len(res.Artifacts))
}

var fiscalYearRe = regexp.MustCompile(`^\d{4}(-\d{2})?$`)

// classifyMain loads a reviewed COFOG mapping and applies it to one jurisdiction-year-flow.
// It exits non-zero if the resulting cofog view does not reconcile exactly.
func classifyMain(args []string) {
	fs := flag.NewFlagSet("classify", flag.ExitOnError)
	mapping := fs.String("mapping", "", "path to reviewed mapping JSON (data/cofog/*.json)")
	jurisdiction := fs.String("jurisdiction", "", "jurisdiction the mapping applies to, e.g. us-ca")
	year := fs.String("year", "", "fiscal year to classify, e.g. 2014-15")
	flow := fs.String("flow", "spending", "flow to classify (spending|revenue)")
	dryRun := fs.Bool("dry-run", false, "plan and report without writing")
	listUnmapped := fs.Bool("list-unmapped", false, "print only the unmapped-categories report (implies --dry-run)")
	_ = fs.Parse(args)

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if *mapping == "" || *jurisdiction == "" || *year == "" {
		log.Error("--mapping, --jurisdiction and --year are required")
		os.Exit(2)
	}
	if !fiscalYearRe.MatchString(*year) {
		log.Error("invalid fiscal year", "year", *year)
		os.Exit(2)
	}
	if *flow != "spending" && *flow != "revenue" {
		log.Error("invalid flow", "flow", *flow)
		os.Exit(2)
	}

	m, err := classify.LoadMapping(*mapping, *jurisdiction)
	if err != nil {
		log.Error("load mapping", "err", err)
		os.Exit(2)
	}

	ctx := context.Background()
	pool, err := store.Connect(ctx, mustEnv(log, "DATABASE_URL"))
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	report, err := classify.Apply(ctx, pool, m, *jurisdiction, *year, *flow, *dryRun || *listUnmapped)
	if err != nil {
		log.Error("classify", "err", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if *listUnmapped {
		_ = enc.Encode(map[string]any{
			"mappingFile":        report.MappingFile,
			"mappingSha256":      report.MappingSha256,
			"jurisdiction":       report.Jurisdiction,
			"fiscalYear":         report.FiscalYear,
			"flow":               report.Flow,
			"unmappedCategories": report.UnmappedCategories,
		})
		return
	}
	_ = enc.Encode(report)
	if !report.Reconciliation.Reconciles {
		log.Error("cofog view does not reconcile (facts were dropped)")
		os.Exit(1)
	}
	log.Info("classify done",
		"inserted", report.Inserted, "upToDate", report.UpToDate,
		"unmappedCategories", len(report.UnmappedCategories), "dryRun", report.DryRun)
}

func runMain(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	adapter := fs.String("adapter", "", `adapter base command, e.g. "node /abs/dist/cli.js"`)
	year := fs.String("year", "", "single fiscal year")
	years := fs.String("years", "", "comma-separated fiscal years (multi-year backfill)")
	dataset := fs.String("dataset", "", "object-storage partition (default: manifest datasets[0])")
	concurrency := fs.Int("concurrency", 2, "max concurrent years")
	replayDir := fs.String("replay-dir", "", "OUTLAYS_REPLAY_DIR for offline replay")
	maxPages := fs.Int("max-pages", 0, "OUTLAYS_MAX_PAGES (0 = unbounded)")
	_ = fs.Parse(args)

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

	obj, err := store.NewObjectStore(ctx, store.ObjectStoreConfigFromEnv())
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
