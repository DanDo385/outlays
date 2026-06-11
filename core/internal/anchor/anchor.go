package anchor

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunIDBytes32 encodes a run UUID for the contract: the UUID's 16 bytes occupy the
// high-order bytes of a bytes32, zero-padded (D31), returned as 0x-prefixed hex.
func RunIDBytes32(runID string) (string, error) {
	u, err := uuid.Parse(runID)
	if err != nil {
		return "", fmt.Errorf("run id: %w", err)
	}
	var b [32]byte
	copy(b[:16], u[:])
	return "0x" + hex.EncodeToString(b[:]), nil
}

// FactHashes returns a run's fact hashes (the Merkle leaves). Order is irrelevant — Root
// sorts — but a stable query keeps logs readable.
func FactHashes(ctx context.Context, pool *pgxpool.Pool, runID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT fact_hash FROM fiscal_fact
		WHERE run_id = $1 AND fact_hash ~ '^[0-9a-f]{64}$'
		ORDER BY fact_hash`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ChainConfig locates the registry and signing credentials (env-sourced by the cmd).
// Set From for unlocked local nodes (anvil); otherwise PrivateKey signs via cast.
type ChainConfig struct {
	RPCURL     string
	PrivateKey string
	From       string
	Registry   string
}

// Receipt is the subset of the cast send receipt we persist.
type Receipt struct {
	TxHash      string `json:"transactionHash"`
	BlockNumber string `json:"blockNumber"`
	Status      string `json:"status"`
}

var hex32Re = regexp.MustCompile(`^0x[0-9a-f]{64}$`)

// cast runs the pinned Foundry cast binary — the same subprocess pattern the orchestrator
// uses for adapters, keeping go.mod free of an Ethereum client dependency (D31).
func cast(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "cast", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("cast %s: %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("cast %s: %w", args[0], err)
	}
	return out, nil
}

// IsAnchored reads the registry's view function.
func IsAnchored(ctx context.Context, cfg ChainConfig, runIDHex string) (bool, error) {
	out, err := cast(ctx, "call", "--rpc-url", cfg.RPCURL, cfg.Registry, "isAnchored(bytes32)(bool)", runIDHex)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// OnChainRoot reads the stored root for a run (0x0 when unset).
func OnChainRoot(ctx context.Context, cfg ChainConfig, runIDHex string) (string, error) {
	out, err := cast(ctx, "call", "--rpc-url", cfg.RPCURL, cfg.Registry,
		"get(bytes32)(bytes32,string,address,uint64)", runIDHex)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || !hex32Re.MatchString(strings.TrimSpace(lines[0])) {
		return "", fmt.Errorf("unexpected get() output: %q", string(out))
	}
	return strings.TrimSpace(lines[0]), nil
}

// Submit sends anchor(runId, merkleRoot, uri) and returns the receipt.
func Submit(ctx context.Context, cfg ChainConfig, runIDHex, rootHex, uri string) (*Receipt, error) {
	args := []string{"send", "--json", "--rpc-url", cfg.RPCURL}
	if cfg.From != "" {
		args = append(args, "--unlocked", "--from", cfg.From)
	} else {
		args = append(args, "--private-key", cfg.PrivateKey)
	}
	args = append(args, cfg.Registry, "anchor(bytes32,bytes32,string)", runIDHex, rootHex, uri)
	out, err := cast(ctx, args...)
	if err != nil {
		return nil, err
	}
	var r Receipt
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("parse receipt: %w", err)
	}
	if r.Status != "0x1" && r.Status != "1" {
		return nil, fmt.Errorf("anchor tx reverted (status %s, tx %s)", r.Status, r.TxHash)
	}
	return &r, nil
}

// ChainID queries the RPC endpoint's chain id.
func ChainID(ctx context.Context, cfg ChainConfig) (int64, error) {
	out, err := cast(ctx, "chain-id", "--rpc-url", cfg.RPCURL)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

// BlockNumber parses the receipt's block number (cast emits hex or decimal).
func (r *Receipt) Block() *int64 {
	s := strings.TrimSpace(r.BlockNumber)
	if s == "" {
		return nil
	}
	var n int64
	var err error
	if strings.HasPrefix(s, "0x") {
		n, err = strconv.ParseInt(s[2:], 16, 64)
	} else {
		n, err = strconv.ParseInt(s, 10, 64)
	}
	if err != nil {
		return nil
	}
	return &n
}

// Persist records the tx ref for the run (append-only row; the run row itself is never
// updated, per D20).
func Persist(ctx context.Context, pool *pgxpool.Pool, runID, rootHex string, factCount int, chainID int64, registry string, r *Receipt) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO run_anchor (run_id, merkle_root, fact_count, chain_id, contract_address, tx_hash, block_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		runID, rootHex, factCount, chainID, strings.ToLower(registry), r.TxHash, r.Block())
	if err != nil {
		return fmt.Errorf("persist anchor: %w", err)
	}
	return nil
}
