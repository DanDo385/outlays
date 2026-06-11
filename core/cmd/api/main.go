// Command api serves the read-only public HTTP surface (ARCHITECTURE.md Section 5).
//
// Env: DATABASE_URL (read role), PORT (default 8080). When S3_* object storage is
// reachable, the internal DuckDB-over-Parquet engine (S10) is enabled for the view
// endpoint's engine=duckdb flag; OUTLAYS_PARQUET_CACHE overrides its local cache dir.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/djmagro/outlays/core/internal/api"
	"github.com/djmagro/outlays/core/internal/engine"
	"github.com/djmagro/outlays/core/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	pool, err := store.Connect(context.Background(), dbURL)
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var duck *engine.Duck
	if obj, oerr := store.NewObjectStore(context.Background(), store.ObjectStoreConfigFromEnv()); oerr == nil {
		cache := os.Getenv("OUTLAYS_PARQUET_CACHE")
		if cache == "" {
			cache = filepath.Join(os.TempDir(), "outlays-parquet-cache")
		}
		duck = &engine.Duck{Pool: pool, Obj: obj, CacheDir: cache}
		log.Info("duckdb engine enabled", "cache", cache)
	} else {
		log.Warn("object storage unreachable; duckdb engine disabled", "err", oerr)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &api.Server{Pool: pool, Duck: duck}
	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Info("api listening", "addr", httpSrv.Addr)
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
