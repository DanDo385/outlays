// Package ingest runs adapters per the CLI protocol, validates and verifies their output by
// re-derivation, persists via the store, and marks run status. It supports concurrent
// multi-year backfill (errgroup) and projects the User-Agent to adapter subprocesses.
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/djmagro/outlays/core/internal/contract"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/djmagro/outlays/core/internal/verify"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/sync/errgroup"
)

// Version is projected into the adapter User-Agent.
const Version = "0.1.0"

// Options configures an orchestration run.
type Options struct {
	AdapterCmd []string // base command, e.g. {"node", "/abs/dist/cli.js"}
	Dataset    string   // object-storage partition; defaults to manifest.datasets[0]
	Pool       *pgxpool.Pool
	Obj        *store.ObjectStore
	Logger     *slog.Logger
	ExtraEnv   []string // forwarded to the adapter (e.g. OUTLAYS_REPLAY_DIR)
	UserAgent  string   // overrides the default project UA if set
}

type manifest struct {
	AdapterID      string   `json:"adapterId"`
	AdapterVersion string   `json:"adapterVersion"`
	Jurisdiction   string   `json:"jurisdiction"`
	Datasets       []string `json:"datasets"`
}

// Outcome summarizes one year's run.
type Outcome struct {
	Year   string
	RunID  string
	Status string // "succeeded" | "failed"
	Facts  int
}

func (o *Options) userAgent() string {
	if o.UserAgent != "" {
		return o.UserAgent
	}
	return fmt.Sprintf("outlays/%s (+https://github.com/DanDo385/outlay)", Version)
}

func (o *Options) adapterEnv() []string {
	env := append(os.Environ(), o.ExtraEnv...)
	return append(env, "OUTLAYS_USER_AGENT="+o.userAgent())
}

// run executes the adapter with the given args and returns stdout, the process exit code, and
// an error only when the process could not be started.
func (o *Options) run(ctx context.Context, args ...string) (stdout []byte, exitCode int, err error) {
	full := append(append([]string{}, o.AdapterCmd[1:]...), args...)
	cmd := exec.CommandContext(ctx, o.AdapterCmd[0], full...)
	cmd.Env = o.adapterEnv()
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	if runErr != nil {
		var ee *exec.ExitError
		if ok := asExitError(runErr, &ee); ok {
			return outBuf.Bytes(), ee.ExitCode(), nil
		}
		return outBuf.Bytes(), -1, fmt.Errorf("exec %v: %w (stderr: %s)", o.AdapterCmd, runErr, errBuf.String())
	}
	return outBuf.Bytes(), 0, nil
}

func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*target = ee
		return true
	}
	return false
}

// RunYear runs one fiscal year end to end and records the run (succeeded or failed). It
// returns the Outcome; err is non-nil only for orchestrator/infrastructure failures (a
// recorded adapter failure is a normal Outcome with Status "failed").
func RunYear(ctx context.Context, o *Options, year string) (Outcome, error) {
	log := o.Logger.With("year", year)

	work, err := os.MkdirTemp("", "orchestrator-")
	if err != nil {
		return Outcome{}, err
	}
	defer os.RemoveAll(work)

	// info -> manifest
	infoOut, code, err := o.run(ctx, "info")
	if err != nil {
		return Outcome{}, err
	}
	var man manifest
	if code != 0 || json.Unmarshal(infoOut, &man) != nil {
		runID, rerr := store.RecordFailedRun(ctx, o.Pool, store.FailedRun{
			Jurisdiction: "unknown", FiscalYear: year, Reason: "adapter info failed", ExitCode: code,
		})
		log.Error("adapter info failed", "exitCode", code)
		return Outcome{Year: year, RunID: runID, Status: "failed"}, rerr
	}
	dataset := o.Dataset
	if dataset == "" && len(man.Datasets) > 0 {
		dataset = man.Datasets[0]
	}

	// fetch
	rawDir := filepath.Join(work, "raw")
	outPath := filepath.Join(work, "out.json")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return Outcome{}, err
	}
	_, code, err = o.run(ctx, "fetch", "--year", year, "--raw-dir", rawDir, "--out", outPath)
	if err != nil {
		return Outcome{}, err
	}
	if code != 0 {
		reason := map[int]string{2: "source unavailable or restricted", 3: "output failed contract validation"}[code]
		if reason == "" {
			reason = "adapter fetch failed"
		}
		runID, rerr := store.RecordFailedRun(ctx, o.Pool, store.FailedRun{
			AdapterID: man.AdapterID, AdapterVersion: man.AdapterVersion,
			Jurisdiction: man.Jurisdiction, FiscalYear: year, Reason: reason, ExitCode: code,
		})
		log.Warn("adapter fetch non-zero exit", "exitCode", code, "reason", reason, "runId", runID)
		return Outcome{Year: year, RunID: runID, Status: "failed"}, rerr
	}

	outBytes, err := os.ReadFile(outPath)
	if err != nil {
		return Outcome{}, err
	}

	// validate against the contract + verify resultHash by re-derivation
	if failReason := validateAndVerify(outBytes); failReason != "" {
		runID, rerr := store.RecordFailedRun(ctx, o.Pool, store.FailedRun{
			AdapterID: man.AdapterID, AdapterVersion: man.AdapterVersion,
			Jurisdiction: man.Jurisdiction, FiscalYear: year, Reason: failReason, ExitCode: 0,
		})
		log.Error("output verification failed", "reason", failReason, "runId", runID)
		return Outcome{Year: year, RunID: runID, Status: "failed"}, rerr
	}

	doc, err := store.ParseDocument(outBytes)
	if err != nil {
		return Outcome{}, err
	}
	res, err := store.Ingest(ctx, o.Pool, o.Obj, doc, rawDir, dataset)
	if err != nil {
		return Outcome{}, fmt.Errorf("persist: %w", err)
	}
	log.Info("ingested", "runId", res.RunID, "facts", res.Facts, "entities", res.Entities,
		"aliases", res.Aliases, "assignments", res.Assignments)
	return Outcome{Year: year, RunID: res.RunID, Status: "succeeded", Facts: res.Facts}, nil
}

// validateAndVerify returns "" when the document is contract-valid and its envelope.resultHash
// matches a fresh recomputation; otherwise a human-readable failure reason.
func validateAndVerify(outBytes []byte) string {
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(outBytes))
	if err != nil {
		return "output is not valid JSON"
	}
	if err := contract.Validate("AdapterOutput", instance); err != nil {
		return "contract validation: " + err.Error()
	}
	var doc struct {
		Envelope struct {
			ResultHash string `json:"resultHash"`
		} `json:"envelope"`
		Facts []json.RawMessage `json:"facts"`
	}
	if err := json.Unmarshal(outBytes, &doc); err != nil {
		return "parse: " + err.Error()
	}
	recomputed, err := verify.RecomputeResultHash(doc.Facts)
	if err != nil {
		return "recompute resultHash: " + err.Error()
	}
	if recomputed != doc.Envelope.ResultHash {
		return fmt.Sprintf("resultHash mismatch: declared %s != recomputed %s", doc.Envelope.ResultHash, recomputed)
	}
	return ""
}

// Backfill runs multiple years concurrently (bounded) and returns all outcomes. A per-year
// recorded failure does not abort siblings; err is non-nil only on infrastructure failure.
func Backfill(ctx context.Context, o *Options, years []string, concurrency int) ([]Outcome, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	var mu sync.Mutex
	outcomes := make([]Outcome, 0, len(years))

	for _, year := range years {
		year := year
		g.Go(func() error {
			oc, err := RunYear(gctx, o, year)
			if err != nil {
				return fmt.Errorf("year %s: %w", year, err)
			}
			mu.Lock()
			outcomes = append(outcomes, oc)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return outcomes, err
	}
	return outcomes, nil
}
