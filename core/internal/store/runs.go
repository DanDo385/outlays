package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FailedRun records an ingestion that did not complete (adapter failure, restricted source,
// invalid output, or re-derivation mismatch). ingestion_run is append-only, so a failed run is
// its own single row (Decision D20).
type FailedRun struct {
	RunID          string
	AdapterID      string
	AdapterVersion string
	Jurisdiction   string
	FiscalYear     string
	Reason         string
	ExitCode       int
}

// RecordFailedRun inserts a status='failed' ingestion_run row with an error envelope.
func RecordFailedRun(ctx context.Context, pool *pgxpool.Pool, fr FailedRun) (string, error) {
	if fr.RunID == "" {
		fr.RunID = uuid.NewString()
	}
	if fr.AdapterID == "" {
		fr.AdapterID = "unknown"
	}
	if fr.AdapterVersion == "" {
		fr.AdapterVersion = "unknown"
	}
	envelope, _ := json.Marshal(map[string]any{
		"status":    "failed",
		"adapterId": fr.AdapterID,
		"error":     fr.Reason,
		"exitCode":  fr.ExitCode,
	})
	_, err := pool.Exec(ctx, `
		INSERT INTO ingestion_run (run_id, adapter_id, adapter_version, jurisdiction, fiscal_year, completed_at, status, envelope)
		VALUES ($1,$2,$3,$4,$5,$6,'failed',$7)
		ON CONFLICT (run_id) DO NOTHING`,
		fr.RunID, fr.AdapterID, fr.AdapterVersion, fr.Jurisdiction, fr.FiscalYear, time.Now(), envelope)
	return fr.RunID, err
}

// RunStatus returns the status of an ingestion_run (for tests/observability).
func RunStatus(ctx context.Context, pool *pgxpool.Pool, runID string) (string, error) {
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM ingestion_run WHERE run_id=$1`, runID).Scan(&status)
	return status, err
}
