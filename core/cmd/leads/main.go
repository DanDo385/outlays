// Command leads is the PRIVATE leads workbench (task S11): run a rule to generate draft
// leads, then review them. It is an operator tool — nothing here is publicly reachable;
// the public surface remains /v1/leads, which serves only human-published leads.
//
// Usage:
//
//	leads rules
//	leads run --rule <ruleId> --jurisdiction <jur> --year <Y>
//	leads list [--status draft|reviewed|published|dismissed]
//	leads inspect <leadId>
//	leads set-status <leadId> --status reviewed|published|dismissed --reviewer <handle> [--note <text>]
//
// Env: DATABASE_URL.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/djmagro/outlays/core/internal/leads"
	"github.com/djmagro/outlays/core/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const usage = `usage:
  leads rules
  leads run --rule <ruleId> --jurisdiction <jur> --year <Y>
  leads list [--status <s>]
  leads inspect <leadId>
  leads set-status <leadId> --status <s> --reviewer <handle> [--note <text>]`

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	out := json.NewEncoder(os.Stdout)
	out.SetIndent("", "  ")

	switch os.Args[1] {
	case "rules":
		ids, err := leads.Rules()
		if err != nil {
			fatal(log, "list rules", err)
		}
		_ = out.Encode(map[string]any{"rules": ids})

	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		rule := fs.String("rule", "", "rule id (see: leads rules)")
		jur := fs.String("jurisdiction", "", "jurisdiction, e.g. us-ca")
		year := fs.String("year", "", "fiscal year, e.g. 2014-15")
		_ = fs.Parse(os.Args[2:])
		if *rule == "" || *jur == "" || *year == "" {
			fatalMsg(log, "--rule, --jurisdiction and --year are required")
		}
		r, err := leads.LoadRule(*rule)
		if err != nil {
			fatal(log, "load rule", err)
		}
		res, err := leads.Run(context.Background(), connect(log), r, *jur, *year)
		if err != nil {
			fatal(log, "run rule", err)
		}
		_ = out.Encode(res)
		log.Info("rule run complete", "matches", res.Matches, "inserted", res.Inserted, "alreadyKnown", res.AlreadyKnown)

	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		status := fs.String("status", "", "filter by current status")
		_ = fs.Parse(os.Args[2:])
		rows, err := leads.List(context.Background(), connect(log), *status)
		if err != nil {
			fatal(log, "list leads", err)
		}
		_ = out.Encode(map[string]any{"leads": rows})

	case "inspect":
		if len(os.Args) < 3 {
			fatalMsg(log, "leads inspect <leadId>")
		}
		d, err := leads.Inspect(context.Background(), connect(log), os.Args[2])
		if err != nil {
			fatal(log, "inspect lead", err)
		}
		if d == nil {
			fatalMsg(log, "lead not found")
		}
		_ = out.Encode(d)

	case "set-status":
		if len(os.Args) < 3 {
			fatalMsg(log, "leads set-status <leadId> --status <s> --reviewer <handle>")
		}
		leadID := os.Args[2]
		fs := flag.NewFlagSet("set-status", flag.ExitOnError)
		status := fs.String("status", "", "reviewed | published | dismissed")
		reviewer := fs.String("reviewer", "", "reviewer handle (required)")
		note := fs.String("note", "", "optional review note")
		_ = fs.Parse(os.Args[3:])
		if err := leads.SetStatus(context.Background(), connect(log), leadID, *status, *reviewer, *note); err != nil {
			fatal(log, "set status", err)
		}
		log.Info("status recorded", "leadId", leadID, "status", *status, "reviewer", *reviewer)

	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func connect(log *slog.Logger) *pgxpool.Pool {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fatalMsg(log, "DATABASE_URL is required")
	}
	pool, err := store.Connect(context.Background(), url)
	if err != nil {
		fatal(log, "connect db", err)
	}
	return pool
}

func fatal(log *slog.Logger, msg string, err error) {
	log.Error(msg, "err", err)
	os.Exit(1)
}

func fatalMsg(log *slog.Logger, msg string) {
	log.Error(msg)
	os.Exit(2)
}
