// Command api serves the read-only public HTTP surface (ARCHITECTURE.md Section 5).
//
// Env: DATABASE_URL (read role), PORT (default 8080).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/djmagro/outlays/core/internal/api"
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &api.Server{Pool: pool}
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
