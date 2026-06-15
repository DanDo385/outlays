# Local dev bootstrap for Outlays. See README.md and .env.example.
SHELL := /bin/bash
ROOT := $(CURDIR)
COMPOSE := docker compose -f deploy/docker-compose.yml
PNPM := pnpm

.PHONY: help up down restart env wait migrate bootstrap-db build go contracts node python \
        seed seed-ca run-api stop-api run-orchestrator test integration

help: ## Show targets
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} \
		/^[a-zA-Z0-9_-]+:.*##/ {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

up: env infra wait bootstrap-db build ## Stack up: .env, Postgres+MinIO, migrate, build
	@echo ""
	@echo "Outlays is ready for local dev."
	@echo "  make seed     — federal FY2024 + FY2025 assistance demo (offline fixtures)"
	@echo "  make run-api  — start read API on http://localhost:8080"
	@echo "  make test     — unit tests (no live upstream APIs)"

env: ## Copy .env.example → .env when missing
	@test -f .env || (cp .env.example .env && echo "created .env from .env.example")

infra: env ## Start Postgres + MinIO (docker compose)
	$(COMPOSE) up -d

down: ## Stop compose stack
	$(COMPOSE) down

restart: down up ## Restart stack and rebuild

wait: ## Block until Postgres and MinIO are healthy
	@bash scripts/wait-for-services.sh

migrate: env ## Run goose migrations (MIGRATE_DATABASE_URL)
	@set -a && source .env && set +a && cd core && go run ./cmd/migrate

bootstrap-db: migrate ## Create app login role from .env and grant app_rw
	@bash scripts/bootstrap-db.sh

build: node go contracts ## Build TS workspace, Go core, and contracts

node: env ## pnpm install + build
	$(PNPM) install --frozen-lockfile
	$(PNPM) -r build

go: ## go build ./... in core/
	cd core && go build ./...

contracts: ## forge build (initializes forge-std submodule)
	git submodule update --init --recursive contracts/lib/forge-std
	cd contracts && forge build

python: ## uv sync for py/adapter_sdk (optional)
	cd py/adapter_sdk && uv sync

seed-ca: build ## Legacy CA 2014-15 replay demo + COFOG classify
	@set -a && source .env && set +a && \
	$(MAKE) run-orchestrator ARGS='run --adapter "node $(ROOT)/packages/adapters/us-ca-procurement/dist/cli.js" --year 2014-15 --replay-dir $(ROOT)/packages/adapters/us-ca-procurement/fixtures/replay --max-pages 1' && \
	$(MAKE) run-orchestrator ARGS='classify --mapping $(ROOT)/data/cofog/us-ca-procurement.json --jurisdiction us-ca --year 2014-15'

seed: build ## Ingest federal FY2025 + FY2024 assistance replay fixtures
	@set -a && source .env && set +a && \
	$(MAKE) run-orchestrator ARGS='run --adapter "node $(ROOT)/packages/adapters/us-fed-usaspending/dist/cli.js" --year 2025 --replay-dir $(ROOT)/packages/adapters/us-fed-usaspending/fixtures/replay --max-pages 1' && \
	$(MAKE) run-orchestrator ARGS='run --adapter "node $(ROOT)/packages/adapters/us-fed-usaspending/dist/cli.js" --year 2024 --replay-dir $(ROOT)/packages/adapters/us-fed-usaspending/fixtures/replay --max-pages 1'

run-orchestrator: env ## Internal: run orchestrator with ARGS (loads .env)
	@set -a && source .env && set +a && cd core && go run ./cmd/orchestrator $(ARGS)

run-api: env go ## Start the read API (PORT from env, default 8080)
	@set -a && source .env && set +a && \
	port="$${PORT:-8080}" && \
	if lsof -nP -iTCP:"$$port" -sTCP:LISTEN >/dev/null 2>&1; then \
	  echo "Port $$port is already in use. The API may already be running."; \
	  echo "  curl http://localhost:$$port/v1/jurisdictions"; \
	  echo "  make stop-api   # stop the listener, then re-run make run-api"; \
	  exit 1; \
	fi && \
	cd core && go run ./cmd/api

stop-api: ## Stop whatever is listening on PORT (default 8080)
	@port="$${PORT:-8080}" && \
	pid=$$(lsof -t -iTCP:"$$port" -sTCP:LISTEN 2>/dev/null || true) && \
	if [ -z "$$pid" ]; then echo "nothing listening on $$port"; exit 0; fi && \
	echo "stopping PID $$pid on :$$port" && kill $$pid

test: build ## Go + pnpm unit tests (skips integration tag)
	cd core && go test ./...
	$(PNPM) -r --if-present test

integration: build wait migrate bootstrap-db ## Integration tests (needs compose stack)
	cd core && go test -tags integration -p 1 -count=1 ./internal/store/ ./internal/ingest/ ./internal/api/ ./internal/classify/ ./internal/engine/ ./internal/leads/
